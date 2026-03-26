// lib/models/poker_model.dart
import 'dart:async';
import 'dart:convert';
import 'dart:developer' as developer;
import 'package:flutter/foundation.dart';
import 'package:collection/collection.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart'
    show
        MakeBetArgs,
        EvaluateHandArgs,
        CardArg,
        CreatePokerTableArgs,
        JoinPokerTableArgs,
        InitClient,
        GameUpdateDTO;
import 'package:pokerui/config.dart';
import 'package:pokerui/models/notifications.dart';
import 'package:pokerui/services/sound_service.dart';

/// -------- UI-facing enums --------
enum PokerState {
  idle,
  browsingTables,
  inLobby, // seated, waiting / readying
  handInProgress, // active betting streets
  showdown, // results surfaced
  gameEnded, // game over - winner determined
  tournamentOver, // SNG complete
}

extension PhaseName on pr.GamePhase {
  String get label => switch (this) {
        pr.GamePhase.WAITING => 'Waiting',
        pr.GamePhase.NEW_HAND_DEALING => 'Dealing',
        pr.GamePhase.PRE_FLOP => 'Pre-Flop',
        pr.GamePhase.FLOP => 'Flop',
        pr.GamePhase.TURN => 'Turn',
        pr.GamePhase.RIVER => 'River',
        pr.GamePhase.SHOWDOWN => 'Showdown',
        _ => 'Unknown',
      };
}

/// -------- Immutable view models (derived from proto) --------
@immutable
class UiPlayer {
  final String id;
  final String name;
  final int balance; // chips (in-game)
  final List<pr.Card> hand;
  final int currentBet; // chips
  final bool folded;
  final bool isTurn;
  final bool isAllIn;
  final bool isDealer;
  final bool isSmallBlind;
  final bool isBigBlind;
  final bool isReady;
  final bool isDisconnected;
  final String handDesc; // only meaningful at showdown
  final bool cardsRevealed;
  final String escrowId;
  final bool escrowReady;
  final String escrowState;
  final bool presignComplete;
  final int tableSeat; // 0-based seat index

  const UiPlayer({
    required this.id,
    required this.name,
    required this.balance,
    required this.hand,
    required this.currentBet,
    required this.folded,
    required this.isTurn,
    required this.isAllIn,
    required this.isDealer,
    required this.isSmallBlind,
    required this.isBigBlind,
    required this.isReady,
    required this.isDisconnected,
    required this.handDesc,
    this.cardsRevealed = false,
    this.escrowId = '',
    this.escrowReady = false,
    this.escrowState = '',
    this.presignComplete = false,
    this.tableSeat = -1, // not seated
  });

  factory UiPlayer.fromProto(pr.Player p) {
    return UiPlayer(
      id: p.id,
      name: p.name,
      balance: p.balance.toInt(),
      hand: List.unmodifiable(p.hand),
      currentBet: p.currentBet.toInt(),
      folded: p.folded,
      isTurn: p.isTurn,
      isAllIn: p.isAllIn,
      isDealer: p.isDealer,
      isSmallBlind: p.isSmallBlind,
      isBigBlind: p.isBigBlind,
      isReady: p.isReady,
      isDisconnected: p.isDisconnected,
      handDesc: p.handDescription,
      cardsRevealed: p.cardsRevealed,
      escrowId: p.escrowId,
      escrowReady: p.escrowReady,
      escrowState: '',
      presignComplete: p.presignComplete,
      tableSeat: p.tableSeat,
    );
  }

  UiPlayer copyWith({
    String? name,
    bool? isReady,
    bool? isDisconnected,
    bool? isAllIn,
    String? handDesc,
    bool? cardsRevealed,
    String? escrowId,
    bool? escrowReady,
    String? escrowState,
    bool? presignComplete,
    int? tableSeat,
  }) {
    return UiPlayer(
      id: id,
      name: name ?? this.name,
      balance: balance,
      hand: hand,
      currentBet: currentBet,
      folded: folded,
      isTurn: isTurn,
      isAllIn: isAllIn ?? this.isAllIn,
      isDealer: isDealer,
      isSmallBlind: isSmallBlind,
      isBigBlind: isBigBlind,
      isReady: isReady ?? this.isReady,
      isDisconnected: isDisconnected ?? this.isDisconnected,
      handDesc: handDesc ?? this.handDesc,
      cardsRevealed: cardsRevealed ?? this.cardsRevealed,
      escrowId: escrowId ?? this.escrowId,
      escrowReady: escrowReady ?? this.escrowReady,
      escrowState: escrowState ?? this.escrowState,
      presignComplete: presignComplete ?? this.presignComplete,
      tableSeat: tableSeat ?? this.tableSeat,
    );
  }

  UiPlayer withHand(List<pr.Card> newHand) {
    return UiPlayer(
      id: id,
      name: name,
      balance: balance,
      hand: List<pr.Card>.unmodifiable(newHand),
      currentBet: currentBet,
      folded: folded,
      isTurn: isTurn,
      isAllIn: isAllIn,
      isDealer: isDealer,
      isSmallBlind: isSmallBlind,
      isBigBlind: isBigBlind,
      isReady: isReady,
      isDisconnected: isDisconnected,
      handDesc: handDesc,
      cardsRevealed: cardsRevealed,
      escrowId: escrowId,
      escrowReady: escrowReady,
      escrowState: escrowState,
      presignComplete: presignComplete,
      tableSeat: tableSeat,
    );
  }

  UiPlayer clearHandState({bool clearCards = true}) {
    final next = clearCards ? withHand(const <pr.Card>[]) : this;
    return next.copyWith(handDesc: '', cardsRevealed: false);
  }
}

@immutable
class UiWinner {
  final String playerId;
  final pr.HandRank handRank;
  final List<pr.Card> bestHand;
  final int winnings; // chips distribution from last showdown cache

  const UiWinner({
    required this.playerId,
    required this.handRank,
    required this.bestHand,
    required this.winnings,
  });

  factory UiWinner.fromProto(pr.Winner w) => UiWinner(
        playerId: w.playerId,
        handRank: w.handRank,
        bestHand: List.unmodifiable(w.bestHand),
        winnings: w.winnings.toInt(),
      );
}

@immutable
class UiBetFx {
  final String playerId;
  final int amount; // chips added to currentBet in a single action
  final int createdMs; // wall time when detected

  const UiBetFx({
    required this.playerId,
    required this.amount,
    required this.createdMs,
  });
}

@immutable
class UiTable {
  final String id;
  final List<UiPlayer> players;
  final int smallBlind;
  final int bigBlind;
  final int maxPlayers;
  final int minPlayers;
  final int currentPlayers;
  final int buyInAtoms;
  final pr.GamePhase phase;
  final bool gameStarted;
  final bool allReady;

  const UiTable({
    required this.id,
    required this.players,
    required this.smallBlind,
    required this.bigBlind,
    required this.maxPlayers,
    required this.minPlayers,
    required this.currentPlayers,
    required this.buyInAtoms,
    required this.phase,
    required this.gameStarted,
    required this.allReady,
  });

  factory UiTable.fromProto(pr.Table t) => UiTable(
        id: t.id,
        players: List.unmodifiable(t.players.map(UiPlayer.fromProto)),
        smallBlind: t.smallBlind.toInt(),
        bigBlind: t.bigBlind.toInt(),
        maxPlayers: t.maxPlayers,
        minPlayers: t.minPlayers,
        currentPlayers: t.currentPlayers,
        buyInAtoms: t.buyIn.toInt(),
        phase: t.phase,
        gameStarted: t.gameStarted,
        allReady: t.allPlayersReady,
      );
}

@immutable
class UiGameState {
  final String tableId;
  final pr.GamePhase phase;
  final String phaseName;
  final List<UiPlayer> players;
  final List<pr.Card> communityCards;
  final int pot; // chips
  final int currentBet; // chips
  final String currentPlayerId;
  final int minRaise; // chips
  final int maxRaise; // chips
  final int smallBlind; // chips
  final int bigBlind; // chips
  final bool gameStarted;
  final int playersRequired;
  final int playersJoined;
  final int timeBankSeconds; // configured per-turn timebank (seconds)
  final int
      turnDeadlineUnixMs; // absolute ms deadline for current player (0 if N/A)

  const UiGameState({
    required this.tableId,
    required this.phase,
    required this.phaseName,
    required this.players,
    required this.communityCards,
    required this.pot,
    required this.currentBet,
    required this.currentPlayerId,
    required this.minRaise,
    required this.maxRaise,
    required this.smallBlind,
    required this.bigBlind,
    required this.gameStarted,
    required this.playersRequired,
    required this.playersJoined,
    required this.timeBankSeconds,
    required this.turnDeadlineUnixMs,
  });

  factory UiGameState.fromUpdate(pr.GameUpdate u) => UiGameState(
        tableId: u.tableId,
        phase: u.phase,
        phaseName: u.phaseName.isNotEmpty ? u.phaseName : u.phase.label,
        players: List.unmodifiable(u.players.map(UiPlayer.fromProto)),
        communityCards: List.unmodifiable(u.communityCards),
        pot: u.pot.toInt(),
        currentBet: u.currentBet.toInt(),
        currentPlayerId: u.currentPlayer,
        minRaise: u.minRaise.toInt(),
        maxRaise: u.maxRaise.toInt(),
        smallBlind: u.hasSmallBlind() ? u.smallBlind.toInt() : 0,
        bigBlind: u.hasBigBlind() ? u.bigBlind.toInt() : 0,
        gameStarted: u.gameStarted,
        playersRequired: u.playersRequired,
        playersJoined: u.playersJoined,
        timeBankSeconds: u.hasTimeBankSeconds() ? u.timeBankSeconds : 0,
        turnDeadlineUnixMs:
            u.hasTurnDeadlineUnixMs() ? u.turnDeadlineUnixMs.toInt() : 0,
      );

  UiGameState copyWith({
    List<UiPlayer>? players,
  }) {
    return UiGameState(
      tableId: tableId,
      phase: phase,
      phaseName: phaseName,
      players: players ?? this.players,
      communityCards: communityCards,
      pot: pot,
      currentBet: currentBet,
      currentPlayerId: currentPlayerId,
      minRaise: minRaise,
      maxRaise: maxRaise,
      smallBlind: smallBlind,
      bigBlind: bigBlind,
      gameStarted: gameStarted,
      playersRequired: playersRequired,
      playersJoined: playersJoined,
      timeBankSeconds: timeBankSeconds,
      turnDeadlineUnixMs: turnDeadlineUnixMs,
    );
  }
}

@immutable
class UiShowdownState {
  final List<UiPlayer> players;
  final List<pr.Card> communityCards;
  final int pot;

  const UiShowdownState({
    required this.players,
    required this.communityCards,
    required this.pot,
  });

  UiShowdownState copyWith({
    List<UiPlayer>? players,
    List<pr.Card>? communityCards,
    int? pot,
  }) {
    return UiShowdownState(
      players: players ?? this.players,
      communityCards: communityCards ?? this.communityCards,
      pot: pot ?? this.pot,
    );
  }
}

bool isAutoAdvanceAllIn(UiGameState? g) {
  if (g == null) return false;
  final actionablePhase = g.phase == pr.GamePhase.PRE_FLOP ||
      g.phase == pr.GamePhase.FLOP ||
      g.phase == pr.GamePhase.TURN ||
      g.phase == pr.GamePhase.RIVER;
  if (!actionablePhase) return false;

  // When all remaining (non-folded) players are all-in, streets will
  // auto-advance and no manual decision is expected. Also guard against a
  // current player that is already marked all-in.
  final current = g.players.firstWhereOrNull((p) => p.id == g.currentPlayerId);
  final everyoneAllIn = g.players
      .where((p) => !p.folded)
      .every((p) => p.isAllIn); // folded players can't act

  return (current?.isAllIn ?? false) || everyoneAllIn;
}

/// -------- The main ChangeNotifier --------
class PokerModel extends ChangeNotifier {
  // Identity
  final String playerId;
  final String dataDir;

  // UI/state
  PokerState _state = PokerState.idle;
  PokerState get state => _state;

  String? currentTableId;
  UiGameState? game;
  List<UiTable> tables = const [];
  List<UiWinner> lastWinners = const [];
  int lastShowdownFxMs = 0; // monotonic trigger for showdown chip animation
  String errorMessage = '';
  String successMessage = '';
  String gameEndingMessage = ''; // message shown when game ends

  // Showdown preservation: cache data so it survives game clearing
  UiShowdownState? _showdown;
  bool _showdownCaptured = false;
  final Map<String, List<pr.Card>> _showdownHandsCache = {};
  UiShowdownState? get showdown => _showdown;
  List<UiPlayer> get showdownPlayers => _showdown?.players ?? const [];
  List<pr.Card> get showdownCommunityCards =>
      _showdown?.communityCards ?? const [];
  int get showdownPot => _showdown?.pot ?? 0;

  // Pending game end: store info to show after the user leaves showdown.
  String? _pendingGameEndMessage;
  int myAtomsBalance = 0; // DCR atoms (wallet balance for buy-in requirements)
  // Track outpoints that have failed binding so we can hide them from future bind dialogs.
  final Set<String> _invalidEscrowOutpoints = {};
  String? _lastBoundEscrowId;
  bool _lastBoundEscrowReady = false;
  String _lastBoundEscrowState = '';

  // Cache hero hole cards for use at showdown when the server may omit them
  List<pr.Card> _myHoleCardsCache = const [];
  List<pr.Card> get myHoleCardsCache => _myHoleCardsCache;
  bool _lastWinnersLoading = false;

  void _maybeRefreshLastWinners() {
    if (lastWinners.isEmpty && !_lastWinnersLoading) {
      unawaited(_refreshLastWinners());
    }
  }

  // Lightweight FX hook: last bet/call animation trigger
  UiBetFx? lastBetFx; // set when a player's currentBet increases

  // Timebank tracking (server-provided deadline)
  int timeBankSeconds = 30; // default when unknown
  DateTime? _turnDeadline;

  // Cached readiness
  bool _iAmReady = false;
  bool _seated = false; // track whether user is seated at any table
  bool _restoring = false; // guard against repeated restore/join loops
  bool _showTableView =
      true; // controls whether UI should render the active table or lobby

  // Payout address bound to the authenticated session on the server (empty if not signed).
  String _authedPayoutAddress = '';
  String get authedPayoutAddress => _authedPayoutAddress;
  bool get hasAuthedPayoutAddress => _authedPayoutAddress.trim().isNotEmpty;
  void updateAuthedPayoutAddress(String addr) {
    _authedPayoutAddress = addr.trim();
  }

  // Control whether UI renders the active table or the home/browsing view.
  void showHomeView() {
    if (_showTableView) {
      _showTableView = false;
      notifyListeners();
    }
  }

  void openTableView() {
    if (!_showTableView) {
      _showTableView = true;
      notifyListeners();
    }
  }

  // Settlement presigning state
  bool _presignInProgress = false;
  String _presignError = '';

  bool get presignInProgress => _presignInProgress;

  /// Returns true if server reports presigning complete for current player
  bool get presignCompleted => me?.presignComplete ?? false;
  String get presignError => _presignError;
  bool get showTableView => _showTableView;

  // Subscription to poker notifications coming from golib.
  StreamSubscription<pr.Notification>? _pokerNtfnSub;
  // Subscription to game updates coming from golib.
  StreamSubscription<pr.GameUpdate>? _gameUpdateSub;

  // Sound service for playing game sounds
  final SoundService _soundService = SoundService();

  // Track previous current player ID to detect turn changes
  String? _previousCurrentPlayerId;

  PokerModel({
    required this.playerId,
    required this.dataDir,
  });

  /// Factory method to create PokerModel from Config
  static Future<PokerModel> fromConfig(
      Config cfg, NotificationModel notificationModel) async {
    // Initialize the Go library with configuration
    final initClientArgs = InitClient(
      cfg.serverAddr,
      cfg.grpcCertPath,
      cfg.dataDir,
      cfg.payoutAddress,
      '${cfg.dataDir}/logs/pokerui.log',
      cfg.debugLevel,
      cfg.soundsEnabled,
    );

    // Initialize the Go library client
    final localInfo = await Golib.initClient(initClientArgs);

    // Use the player ID from the Go library initialization
    final playerId = localInfo.id;

    final model = PokerModel(
      playerId: playerId,
      dataDir: cfg.dataDir,
    );

    // Initialize sound service from config
    model._soundService.setEnabled(cfg.soundsEnabled);

    return model;
  }

  // -------- Lifecycle ----------
  Future<void> init() async {
    // Subscribe to poker notifications emitted by golib so we can update the
    // UI in response to server events without polling.
    _pokerNtfnSub ??= Golib.pokerNotifications().listen(
      _onNotification,
      onError: (e, st) {
        errorMessage = 'Stream error: $e';
        notifyListeners();
      },
    );
    // Subscribe to game updates from the game stream
    _gameUpdateSub ??= Golib.gameUpdates().listen(
      _onGameUpdate,
      onError: (e, st) {
        errorMessage = 'Game update stream error: $e';
        notifyListeners();
      },
    );
    await _fetchInitialTables();
    // If server remembers seat, restore:
    await _restoreCurrentTable();
  }

  @override
  void dispose() {
    _pokerNtfnSub?.cancel();
    _gameUpdateSub?.cancel();
    super.dispose();
  }

  // -------- Notifications (from Go; no direct gRPC stream) ----------
  void _onNotification(pr.Notification n) {
    switch (n.type) {
      case pr.NotificationType.NOTIFICATION_STREAM_CONNECTED:
        // Force a full resync in case lobby/table notifications were missed
        // while the device was asleep.
        unawaited(refreshTables());
        // Force a full resync in case the initial resync GameUpdate was missed.
        unawaited(_restoreCurrentTable());
        break;
      case pr.NotificationType.GAME_STREAM_CONNECTED:
        // Ensure latest state and that the game stream is active.
        unawaited(refreshGameState());
        unawaited(ensureGameStream());
        break;
      case pr.NotificationType.NOTIFICATION_STREAM_DISCONNECTED:
      case pr.NotificationType.GAME_STREAM_DISCONNECTED:
        break;
      case pr.NotificationType.PLAYER_JOINED:
      case pr.NotificationType.PLAYER_LEFT:
      case pr.NotificationType.PLAYER_UNREADY:
      case pr.NotificationType.PLAYER_READY:
      case pr.NotificationType.ALL_PLAYERS_READY:
        // Update table state from notification snapshot (includes players list)
        if (n.hasTable() && n.tableId.isNotEmpty) {
          _updateTableFromSnapshot(n.tableId, n.table);
        }
        // Also update game.players if this is our current table (lobby state)
        if (n.tableId == currentTableId) {
          _updateGamePlayersFromTable(n.table);
        }
        break;

      case pr.NotificationType.TABLE_CREATED:
        if (n.hasTable()) {
          _addTableFromNotification(n.table);
        }
        break;
      case pr.NotificationType.TABLE_REMOVED:
        if (n.tableId.isNotEmpty) {
          _removeTableById(n.tableId);
        }
        break;
      case pr.NotificationType.NEW_HAND_STARTED:
        // Clear cached hero hole cards for the new hand to avoid stale display
        _myHoleCardsCache = const [];
        _showdownHandsCache.clear();
        // Clear any stale bet FX at the start of a new hand
        lastBetFx = null;
        _showdownCaptured = false;
        final g = game;
        if (g != null) {
          game = g.copyWith(
            players: List<UiPlayer>.unmodifiable(
              g.players.map((p) => p.clearHandState()),
            ),
          );
        }
        // Ensure the game stream is attached for the new hand. In the normal
        // case the stream is already active and this is a no-op; if the game
        // stream was silently lost while we were idle, this re-attaches it.
        unawaited(ensureGameStream());
        notifyListeners();
        break;
      case pr.NotificationType.GAME_STARTED:
        // Explicitly transition to handInProgress when game starts
        if (n.tableId == currentTableId ||
            n.tableId.isEmpty ||
            currentTableId == null) {
          _state = PokerState.handInProgress;
          notifyListeners();
        }
        // Don't call refreshGameState here - wait for GameUpdate from stream
        // The stream will send the game state with hand cards
        break;
      case pr.NotificationType.GAME_ENDED:
        if (n.tableId == currentTableId) {
          final msg = n.message.isNotEmpty ? n.message : 'Game ended';
          // Always queue game end so we can finish showing the last showdown
          _queueGameEnd(msg);
        }
        break;
      case pr.NotificationType.PLAYER_LOST:
        // Player lost all chips and was removed from table
        if (n.playerId == playerId && n.tableId == currentTableId) {
          final msg = n.message.isNotEmpty
              ? n.message
              : 'You lost all your chips and have been removed from the table.';
          // Always queue game end so we can show the final showdown before leaving
          _queueGameEnd(msg);
        }
        break;
      case pr.NotificationType.BET_MADE:
        if (n.tableId == currentTableId && n.playerId.isNotEmpty) {
          final amt = n.hasAmount() ? n.amount.toInt() : 0;
          lastBetFx = UiBetFx(
              playerId: n.playerId,
              amount: amt,
              createdMs: DateTime.now().millisecondsSinceEpoch);
          // Play sound for other players' bets only (own bets play sound in makeBet)
          if (n.playerId != playerId) {
            _soundService.playBet();
          }
          notifyListeners();
        }
        // Don't call refreshGameState - stream will send GameUpdate
        break;
      case pr.NotificationType.CALL_MADE:
        if (n.tableId == currentTableId && n.playerId.isNotEmpty) {
          final amt = n.hasAmount() ? n.amount.toInt() : 0;
          lastBetFx = UiBetFx(
              playerId: n.playerId,
              amount: amt,
              createdMs: DateTime.now().millisecondsSinceEpoch);
          notifyListeners();
        }
        // Don't call refreshGameState - stream will send GameUpdate
        break;
      case pr.NotificationType.CHECK_MADE:
        break;
      case pr.NotificationType.PLAYER_FOLDED:
      case pr.NotificationType.SMALL_BLIND_POSTED:
      case pr.NotificationType.BIG_BLIND_POSTED:
        // Don't call refreshGameState - stream will send GameUpdate
        break;
      case pr.NotificationType.PLAYER_ALL_IN:
        if (game != null &&
            n.playerId.isNotEmpty &&
            (n.tableId.isEmpty || n.tableId == currentTableId)) {
          // Mark player as all-in to immediately reflect auto-advance state.
          final updated = game!.players.map((p) {
            if (p.id != n.playerId) return p;
            return p.copyWith(isAllIn: true);
          }).toList(growable: false);
          game = game!.copyWith(players: List.unmodifiable(updated));
          notifyListeners();
        }
        break;

      case pr.NotificationType.SHOWDOWN_RESULT:
        // Handle showdown notification with winners data
        if (n.tableId == currentTableId || n.tableId.isEmpty) {
          _cacheShowdownSnapshot(n.showdown);
          _handleShowdownNotification(n);
        }
        break;

      case pr.NotificationType.CARDS_SHOWN:
        if (n.playerId.isNotEmpty) {
          // If cards are included, hydrate the current game state immediately so UI reflects.
          if (game != null &&
              (n.tableId.isEmpty || n.tableId == currentTableId)) {
            _updatePlayerHandInGame(
              n.playerId,
              hand: n.cards.isNotEmpty ? n.cards : null,
              cardsRevealed: true,
            );
          }
          if (n.cards.isNotEmpty) {
            _showdownHandsCache[n.playerId] =
                List<pr.Card>.unmodifiable(n.cards);
            final showdown = _showdown;
            if (showdown != null && showdown.players.isNotEmpty) {
              final updated = showdown.players
                  .map((p) => p.id == n.playerId
                      ? p
                          .withHand(List<pr.Card>.unmodifiable(n.cards))
                          .copyWith(cardsRevealed: true)
                      : p)
                  .toList(growable: false);
              _showdown = showdown.copyWith(
                players: List.unmodifiable(updated),
              );
            }
          }
          notifyListeners();
        }
        break;

      case pr.NotificationType.CARDS_HIDDEN:
        if (n.playerId.isNotEmpty) {
          if (game != null &&
              (n.tableId.isEmpty || n.tableId == currentTableId)) {
            _updatePlayerHandInGame(
              n.playerId,
              hand: n.playerId == playerId ? null : const [],
              handDesc: '',
              cardsRevealed: false,
            );
          }
          final showdown = _showdown;
          if (showdown != null && showdown.players.isNotEmpty) {
            final updated = showdown.players
                .map((p) => p.id == n.playerId
                    ? (n.playerId == playerId
                        ? p.clearHandState(clearCards: false)
                        : p.clearHandState())
                    : p)
                .toList(growable: false);
            _showdown = showdown.copyWith(
              players: List.unmodifiable(updated),
            );
          }
          notifyListeners();
        }
        break;

      case pr.NotificationType.ESCROW_FUNDING:
        // Escrow readiness/funding updates arrive over the notification stream.
        if (n.tableId.isEmpty || n.tableId != currentTableId) {
          break;
        }
        final payload = _decodeJsonMap(n.message);
        final pid = _firstNonEmpty(payload?['player_id'], n.playerId);
        final escrowId = _firstNonEmpty(payload?['escrow_id'], '');
        final readyRaw = payload?['escrow_ready'] ?? payload?['ready'];
        final ready = readyRaw is bool
            ? readyRaw
            : (readyRaw is num ? readyRaw != 0 : false);
        final fundingState = _firstNonEmpty(payload?['funding_state'], '');
        if (pid.isEmpty) {
          unawaited(refreshGameState());
          break;
        }
        if (game == null) {
          unawaited(refreshGameState());
          break;
        }
        final updatedPlayers = game!.players
            .map((p) => p.id == pid
                ? p.copyWith(
                    escrowId: escrowId,
                    escrowReady: ready,
                    escrowState: fundingState,
                  )
                : p)
            .toList();
        game = game!.copyWith(players: List.unmodifiable(updatedPlayers));
        if (pid == playerId && escrowId.isNotEmpty) {
          _lastBoundEscrowId = escrowId;
          _lastBoundEscrowReady = ready;
          _lastBoundEscrowState = fundingState;
        }
        notifyListeners();
        // When escrow becomes ready, attempt to start presigning
        if (ready) {
          unawaited(_maybeStartPresignForCurrentTable());
        }
        break;

      default:
        break;
    }
  }

  /// Updates game.players from a table snapshot notification.
  /// This keeps the lobby table list fresh (ready/escrow flags) without mutating
  /// in-hand game state or hole cards.
  void _updateGamePlayersFromTable(pr.Table tableSnapshot) {
    final updatedTable = UiTable.fromProto(tableSnapshot);
    final idx = tables.indexWhere((t) => t.id == updatedTable.id);
    if (idx >= 0) {
      final next = List<UiTable>.from(tables);
      next[idx] = updatedTable;
      tables = List<UiTable>.unmodifiable(next);
    } else {
      tables = List<UiTable>.unmodifiable([...tables, updatedTable]);
    }
    // Mirror lobby player readiness/escrow changes into the current game snapshot
    // while we're still waiting to start. Avoid mutating in-hand state.
    final isCurrentTable = updatedTable.id == currentTableId;
    final waitingGame = game != null &&
        game!.phase == pr.GamePhase.WAITING &&
        !game!.gameStarted;
    if (isCurrentTable && waitingGame) {
      final updatedPlayers =
          List<UiPlayer>.unmodifiable(tableSnapshot.players.map(
        UiPlayer.fromProto,
      ));
      game = game!.copyWith(players: updatedPlayers);
    }
    notifyListeners();
  }

  void _updatePlayerHandInGame(
    String playerId, {
    List<pr.Card>? hand,
    String? handDesc,
    bool? cardsRevealed,
  }) {
    final g = game;
    if (g == null) return;
    final updated = g.players.map((p) {
      if (p.id != playerId) return p;
      var next = p;
      if (hand != null) {
        next = next.withHand(List<pr.Card>.unmodifiable(hand));
      }
      if (handDesc != null) {
        next = next.copyWith(handDesc: handDesc);
      }
      if (cardsRevealed != null) {
        next = next.copyWith(cardsRevealed: cardsRevealed);
      }
      return next;
    }).toList(growable: false);
    game = g.copyWith(players: List.unmodifiable(updated));
  }

  /// Queue game end and complete it once the model has actually reached showdown.
  void _queueGameEnd(String message) {
    _pendingGameEndMessage = message;
    notifyListeners();
  }

  void _enterShowdownState() {
    _state = PokerState.showdown;
  }

  /// Complete the transition from showdown to gameEnded state.
  void _completeGameEnd() {
    if (_pendingGameEndMessage == null) {
      // Nothing to do.
      return;
    }
    gameEndingMessage = _pendingGameEndMessage ?? 'Game ended';
    _pendingGameEndMessage = null;
    _state = PokerState.gameEnded;
    // Don't clear game or showdown data - keep it for display
    // The user will navigate away via the GameEndedView
    notifyListeners();
  }

  /// Check if we're showing showdown results (for UI)
  bool get isShowingShowdown => _state == PokerState.showdown;

  /// Check if game end is pending and waiting for explicit user confirmation.
  bool get isGameEndPending => _pendingGameEndMessage != null;

  /// Check if we have last showdown data to display
  bool get hasLastShowdown =>
      lastWinners.isNotEmpty || showdownPlayers.isNotEmpty;

  /// Specific end-state copy to show while the user is still on showdown.
  String get pendingGameEndMessage => _pendingGameEndMessage?.trim() ?? '';

  /// Compact showdown label intended for in-table display near the board.
  String? get showdownResultLabel {
    if (lastWinners.isEmpty) return null;

    if (lastWinners.length > 1) {
      final labels = lastWinners
          .map(_showdownLabelForWinner)
          .where((label) => label.isNotEmpty)
          .toSet()
          .toList(growable: false);
      if (labels.length == 1) {
        return 'Split pot: ${labels.first}';
      }
      return 'Split pot';
    }

    final label = _showdownLabelForWinner(lastWinners.first);
    return label.isNotEmpty ? label : null;
  }

  String _showdownLabelForWinner(UiWinner winner) {
    final livePlayer =
        game?.players.firstWhereOrNull((p) => p.id == winner.playerId);
    final showdownPlayer =
        showdownPlayers.firstWhereOrNull((p) => p.id == winner.playerId);
    final description =
        (livePlayer?.handDesc ?? showdownPlayer?.handDesc ?? '').trim();
    if (description.isNotEmpty) {
      return description;
    }
    return _handRankLabel(winner.handRank);
  }

  String _handRankLabel(pr.HandRank rank) {
    return switch (rank) {
      pr.HandRank.HIGH_CARD => 'High Card',
      pr.HandRank.PAIR => 'Pair',
      pr.HandRank.TWO_PAIR => 'Two Pair',
      pr.HandRank.THREE_OF_A_KIND => 'Three of a Kind',
      pr.HandRank.STRAIGHT => 'Straight',
      pr.HandRank.FLUSH => 'Flush',
      pr.HandRank.FULL_HOUSE => 'Full House',
      pr.HandRank.FOUR_OF_A_KIND => 'Four of a Kind',
      pr.HandRank.STRAIGHT_FLUSH => 'Straight Flush',
      pr.HandRank.ROYAL_FLUSH => 'Royal Flush',
      _ => rank.name,
    };
  }

  /// Allow UI to skip showdown and go directly to game ended.
  void skipShowdown() {
    if (_state == PokerState.showdown && _pendingGameEndMessage != null) {
      _completeGameEnd();
    }
  }

  /// Test helper to set showdown data for testing purposes
  /// This allows tests to set up showdown state without going through notifications
  @visibleForTesting
  void setShowdownDataForTest({
    required List<UiPlayer> players,
    required List<pr.Card> communityCards,
    required int pot,
    List<UiWinner> winners = const [],
  }) {
    lastWinners = winners;
    _showdown = UiShowdownState(
      players: List.unmodifiable(players),
      communityCards: List.unmodifiable(communityCards),
      pot: pot,
    );
    _showdownCaptured = true;
    if (game != null && game!.phase == pr.GamePhase.SHOWDOWN) {
      game = game!.copyWith(
        players: List<UiPlayer>.unmodifiable(players),
      );
    }
    notifyListeners();
  }

  @visibleForTesting
  void applyNotificationForTest(pr.Notification n) {
    _onNotification(n);
  }

  @visibleForTesting
  void applyGameUpdateForTest(pr.GameUpdate gameUpdate) {
    _onGameUpdate(gameUpdate);
  }

  /// Handle SHOWDOWN_RESULT notification - transition to showdown state with winners.
  void _handleShowdownNotification(pr.Notification n) {
    // Extract winners from notification
    if (n.hasShowdown() && n.showdown.winners.isNotEmpty) {
      lastWinners = List.unmodifiable(
        n.showdown.winners.map((w) => UiWinner.fromProto(w)).toList(),
      );
      lastShowdownFxMs = DateTime.now().millisecondsSinceEpoch;
    }

    // Transition to showdown state
    _enterShowdownState();
    notifyListeners();
  }

  UiGameState _keepGameWithShowdownPlayers(UiGameState nextGame) {
    nextGame = _rehydrateGameWithShowdownHands(nextGame);
    if (nextGame.phase != pr.GamePhase.SHOWDOWN || !_showdownCaptured) {
      return nextGame;
    }
    final liveGame = game;
    // Preserve the full showdown roster only while the server is still
    // broadcasting SHOWDOWN snapshots. Once the next hand starts, the live
    // update must replace the stale showdown players/statuses immediately.
    if (liveGame == null) {
      return nextGame;
    }
    if (nextGame.players.length >= liveGame.players.length) {
      return nextGame;
    }
    return nextGame.copyWith(
      players: List<UiPlayer>.unmodifiable(
        _hydrateShowdownHands(liveGame.players),
      ),
    );
  }

  UiGameState _rehydrateGameWithShowdownHands(UiGameState nextGame) {
    if (nextGame.phase != pr.GamePhase.SHOWDOWN) {
      return nextGame;
    }
    return nextGame.copyWith(
      players: List<UiPlayer>.unmodifiable(
        _hydrateShowdownHands(nextGame.players),
      ),
    );
  }

  // -------- Game Updates (from game stream) ----------
  void _onGameUpdate(pr.GameUpdate gameUpdate) {
    // Detect turn change before updating game state
    final newCurrentPlayer = gameUpdate.currentPlayer;
    final isNowMyTurn = newCurrentPlayer == playerId;
    final wasMyTurn = _previousCurrentPlayerId == playerId;

    // Play sound when it becomes the player's turn
    if (isNowMyTurn && !wasMyTurn) {
      // Just became my turn - play notification sound
      _soundService.playTurnNotification();
    }
    // Preserve showdown hole cards before any redactions in later snapshots.
    _stashShowdownHands(gameUpdate);

    // Update game state from the stream update
    game = _keepGameWithShowdownPlayers(UiGameState.fromUpdate(gameUpdate));
    final mePlayer = me;
    if (mePlayer != null && mePlayer.escrowId.isNotEmpty) {
      _lastBoundEscrowId = mePlayer.escrowId;
      _lastBoundEscrowReady = mePlayer.escrowReady;
    }
    // Update UI state based on game phase
    final phase = gameUpdate.phase;
    if (phase == pr.GamePhase.SHOWDOWN) {
      _enterShowdownState();
    } else if (gameUpdate.gameStarted || phase != pr.GamePhase.WAITING) {
      // Game is active if gameStarted flag is true OR phase is not WAITING
      _state = PokerState.handInProgress;
    } else {
      _state = PokerState.inLobby;
    }

    // Update previous current player ID for next comparison
    _previousCurrentPlayerId = newCurrentPlayer;

    notifyListeners();
  }

  // Keep last known showdown hole cards so they remain visible even if later snapshots clear them.
  void _stashShowdownHands(pr.GameUpdate gameUpdate) {
    final isShowdown = gameUpdate.phase == pr.GamePhase.SHOWDOWN;
    for (final p in gameUpdate.players) {
      if (p.hand.isNotEmpty) {
        _showdownHandsCache[p.id] = List<pr.Card>.unmodifiable(p.hand);
        if (p.id == playerId) {
          _myHoleCardsCache = List<pr.Card>.unmodifiable(p.hand);
        }
      } else if (isShowdown &&
          p.id == playerId &&
          _showdownHandsCache.containsKey(p.id)) {
        // Restore our own cards from cache if later snapshots redact them.
        _myHoleCardsCache =
            List<pr.Card>.unmodifiable(_showdownHandsCache[p.id]!);
      }
    }
  }

  List<UiPlayer> _hydrateShowdownHands(List<UiPlayer> players) {
    return players.map((p) {
      if (p.hand.isNotEmpty) return p;
      if (p.id == playerId) {
        final cached = _myHoleCardsCache.isNotEmpty
            ? _myHoleCardsCache
            : _showdownHandsCache[p.id];
      if (cached != null && cached.isNotEmpty) {
          return p.withHand(cached);
        }
        return p;
      }
      final forcedReveal = showdownPlayers.any(
        (sp) => sp.id == p.id && sp.hand.isNotEmpty,
      );
      if (p.folded && !p.cardsRevealed && !forcedReveal) return p;
      final cached = _showdownHandsCache[p.id];
      if (cached != null && cached.isNotEmpty) {
        if (p.cardsRevealed || forcedReveal) {
          return p.withHand(cached).copyWith(cardsRevealed: true);
        }
      }
      if (!p.cardsRevealed) return p;
      return p;
    }).toList();
  }

  // -------- Lobby / Tables ----------
  /// Updates a specific table in the tables list from a notification snapshot.
  /// This preserves the players list from the server snapshot.
  void _updateTableFromSnapshot(String tableId, pr.Table tableSnapshot) {
    final updatedTable = UiTable.fromProto(tableSnapshot);
    final tableIndex = tables.indexWhere((t) => t.id == tableId);
    if (tableIndex >= 0) {
      // Update existing table
      final updatedTables = List<UiTable>.from(tables);
      updatedTables[tableIndex] = updatedTable;
      tables = List.unmodifiable(updatedTables);
    } else {
      // Table not found in list, add it (shouldn't happen but handle gracefully)
      tables = List.unmodifiable([...tables, updatedTable]);
    }
    notifyListeners();
  }

  /// Fetches initial table list on startup. Called once during init().
  Future<void> _fetchInitialTables() async {
    try {
      final list = await Golib.getPokerTables();
      tables = List.unmodifiable(
        list.map((t) => UiTable.fromProto(t.toProtobuf())),
      );
      if (currentTableId == null) {
        _state = PokerState.browsingTables;
        game = null;
        lastWinners = const [];
      }
      errorMessage = '';
      notifyListeners();
    } catch (e) {
      errorMessage = 'Failed to load tables: $e';
      notifyListeners();
    }
  }

  /// Adds a table from a TABLE_CREATED notification.
  void _addTableFromNotification(pr.Table tableProto) {
    final newTable = UiTable.fromProto(tableProto);
    // Don't add duplicates
    if (tables.any((t) => t.id == newTable.id)) {
      // Update existing table instead
      _updateTableFromSnapshot(newTable.id, tableProto);
      return;
    }
    tables = List.unmodifiable([...tables, newTable]);
    notifyListeners();
  }

  /// Removes a table by ID (TABLE_REMOVED notification).
  void _removeTableById(String tableId) {
    final updated = tables.where((t) => t.id != tableId).toList();
    if (updated.length != tables.length) {
      tables = List.unmodifiable(updated);
      // If we were at this table, handle based on current state
      if (currentTableId == tableId) {
        // If the table is removed after the hand, keep the results visible but
        // stop presenting the removed table as interactive.
        if (_state == PokerState.showdown || _state == PokerState.gameEnded) {
          _seated = false;
          if (_state == PokerState.showdown) {
            _pendingGameEndMessage ??= 'Game ended';
            notifyListeners();
          } else {
            notifyListeners();
          }
          return;
        }
        // Otherwise reset to browsing
        currentTableId = null;
        game = null;
        _iAmReady = false;
        _seated = false;
        _state = PokerState.browsingTables;
      }
      notifyListeners();
    }
  }

  /// Public method for UI to transition to browsing tables.
  /// Fetches tables if list is empty (first browse or after disconnect).
  Future<void> browseTables() async {
    if (tables.isEmpty) {
      await _fetchInitialTables();
    }
    if (currentTableId == null) {
      _state = PokerState.browsingTables;
      notifyListeners();
    }
  }

  Future<void> refreshTables() async {
    await _fetchInitialTables();
  }

  Future<String?> createTable({
    required int smallBlindChips,
    required int bigBlindChips,
    required int maxPlayers,
    required int minPlayers,
    required int buyInAtoms,
    required int startingChips,
    int timeBankSeconds = 30,
    int autoStartMs = 0, // Default set on server (3 seconds)
    int autoAdvanceMs = 0, // Default set on server (1 second)
  }) async {
    try {
      final res = await Golib.createPokerTable(CreatePokerTableArgs(
        smallBlindChips,
        bigBlindChips,
        maxPlayers,
        minPlayers,
        buyInAtoms,
        startingChips,
        timeBankSeconds,
        autoStartMs,
        autoAdvanceMs,
      ));
      // Cache timebank seconds locally for countdowns in this session
      this.timeBankSeconds = timeBankSeconds;
      final tid = res['table_id'] as String?;
      if (tid == null || (res['status'] as String?) != 'created') {
        final msg = res['message'] ?? 'unknown error';
        errorMessage = 'Create table failed: $msg';
        notifyListeners();
        return null;
      }
      // TABLE_CREATED notification will add the table to the list
      return tid;
    } catch (e) {
      errorMessage = 'Create table failed: $e';
      notifyListeners();
      return null;
    }
  }

  Future<bool> joinTable(String tableId) async {
    try {
      // Dedup: if we're already seated at this table, just (re)attach streams
      // and refresh state instead of calling server Join again.
      if (_seated && currentTableId == tableId) {
        // Clear any stale showdown/game-end state from previous session.
        _clearShowdownState();
        // Ensure state is up-to-date via golib.
        await refreshGameState();
        await ensureGameStream();
        notifyListeners();
        return true;
      }

      // Delegate join to embedded Go client to avoid Flutter-side identity mismatches
      final res = await Golib.joinPokerTable(JoinPokerTableArgs(tableId));
      if ((res['status'] as String?) != 'joined') {
        final msg = res['message'] ?? 'unknown error';
        errorMessage = 'Join failed: $msg';
        notifyListeners();
        return false;
      }

      currentTableId = tableId;
      _iAmReady = false;
      _seated = true;
      _showTableView = true;
      _clearShowdownState(); // New table: drop any pending game-end or showdown cache
      // PLAYER_JOINED notification will update table state
      // Refresh game state and winners via golib instead of a direct gRPC stream.
      await refreshGameState();
      await ensureGameStream();
      notifyListeners();
      return true;
    } catch (e) {
      errorMessage = 'Join failed: $e';
      notifyListeners();
      return false;
    }
  }

  Future<void> bindEscrow({
    required String tableId,
    required String outpoint,
  }) async {
    String boundEscrowId = '';
    bool boundEscrowReady = false;
    try {
      if (!hasAuthedPayoutAddress) {
        successMessage = '';
        errorMessage =
            'Bind escrow failed: set and verify a payout address on the Sign Address screen before binding.';
        notifyListeners();
        return;
      }
      // Server determines the seat automatically from the caller's position
      final Map<String, dynamic> resp = await Golib.bindEscrow(
        tableId: tableId,
        seatIndex: -1, // Server will determine seat
        outpoint: outpoint,
      );
      boundEscrowId = (resp['escrow_id'] ?? '').toString();
      final readyRaw = resp['escrow_ready'];
      boundEscrowReady = readyRaw is bool
          ? readyRaw
          : (readyRaw is num ? readyRaw != 0 : false);
      // Escrow binding doesn't change table list; ESCROW_FUNDING notification updates player state
      if (game != null && boundEscrowId.isNotEmpty) {
        final updatedPlayers = game!.players
            .map((p) => p.id == playerId
                ? p.copyWith(
                    escrowId: boundEscrowId, escrowReady: boundEscrowReady)
                : p)
            .toList();
        game = game!.copyWith(players: List.unmodifiable(updatedPlayers));
      }
      if (boundEscrowId.isNotEmpty) {
        _lastBoundEscrowId = boundEscrowId;
        _lastBoundEscrowReady = boundEscrowReady;
      }
      await refreshGameState();
      final shortId = tableId.length <= 8 ? tableId : tableId.substring(0, 8);
      successMessage = 'Escrow bound to table $shortId';
      errorMessage = '';
    } catch (e) {
      final raw = e.toString();
      // If the server reports that this escrow/UTXO is no longer usable,
      // remember this outpoint so we don't offer it again.
      if (raw.contains('escrow not found') ||
          raw.contains('funding output has already been spent') ||
          raw.contains('funding outpoint') ||
          raw.contains('txout not found')) {
        _invalidEscrowOutpoints.add(outpoint.trim());
      }
      successMessage = '';
      if (raw.contains('funding output has already been spent') ||
          raw.contains('txout not found')) {
        errorMessage =
            'Bind escrow failed: the funding output for this escrow has already been spent or is no longer available. Please choose a different escrow UTXO.';
      } else if (raw.contains('escrow not found')) {
        errorMessage =
            'Bind escrow failed: this escrow session is no longer known by the referee. Please open or select a different escrow.';
      } else if (raw.toLowerCase().contains('payout address not set') ||
          raw.toLowerCase().contains('sign address')) {
        errorMessage =
            'Bind escrow failed: set and verify a payout address on the Sign Address screen before binding.';
      } else {
        errorMessage = 'Bind escrow failed: $raw';
      }
    }
    notifyListeners();
  }

  Future<List<Map<String, dynamic>>> listCachedEscrows() async {
    try {
      final res =
          await Golib.getBindableEscrows(); // returns list of escrow maps
      final escrows = <Map<String, dynamic>>[];
      for (final any in res) {
        if (any is! Map<String, dynamic>) {
          continue;
        }
        final m = Map<String, dynamic>.from(any);
        final txid = (m['funding_txid'] ?? '').toString().trim();
        final vout = m['funding_vout'];
        if (txid.isEmpty) {
          continue;
        }
        final voutStr = vout is num ? vout.toInt().toString() : vout.toString();
        final outpoint = '$txid:$voutStr';
        if (_invalidEscrowOutpoints.contains(outpoint)) {
          // Skip outpoints we already know will fail binding.
          continue;
        }
        escrows.add(m);
      }
      return escrows;
    } catch (e) {
      // If history directory doesn't exist yet, return empty list (no escrows)
      // This will trigger the "No Escrows Available" dialog
      final msg = e.toString();
      if (msg.contains('history_session') || msg.contains('no such file')) {
        return <Map<String, dynamic>>[];
      }
      // Re-throw other errors
      rethrow;
    }
  }

  // -------- Local refund helpers (historic escrows) ----------

  Future<Map<String, dynamic>> buildRefundTransaction(
    String escrowId,
    String destAddr, {
    int feeAtoms = 20000,
    int? csvBlocks,
    int? utxoValue,
  }) async {
    try {
      final result = await Golib.refundEscrow(
        escrowId: escrowId,
        destAddr: destAddr,
        feeAtoms: feeAtoms,
        csvBlocks: csvBlocks ?? 64,
        utxoValue: utxoValue,
      );
      developer.log(
        'buildRefundTransaction: escrow=$escrowId can_refund=${result['can_refund']}',
        name: 'refunds',
      );
      return Map<String, dynamic>.from(result);
    } catch (e) {
      throw Exception('Failed to build refund transaction: $e');
    }
  }

  Future<void> updateEscrowFundingTx(
    String escrowId,
    String txid,
    int vout,
  ) async {
    try {
      developer.log(
        'updateEscrowFundingTx: escrow=$escrowId txid=$txid vout=$vout',
        name: 'refunds',
      );
      await Golib.updateEscrowHistory({
        'escrow_id': escrowId,
        'funding_txid': txid,
        'funding_vout': vout,
      });
      developer.log(
        'updateEscrowFundingTx: successfully updated escrow $escrowId',
        name: 'refunds',
      );
    } catch (e) {
      developer.log(
        'updateEscrowFundingTx error: $e',
        name: 'refunds',
      );
      throw Exception('Failed to update escrow funding transaction: $e');
    }
  }

  Future<void> deleteHistoricEscrow(String escrowId) async {
    try {
      developer.log('deleteHistoricEscrow: escrow=$escrowId', name: 'refunds');
      await Golib.deleteEscrowHistory(escrowId);
      developer.log('deleteHistoricEscrow: deleted $escrowId', name: 'refunds');
    } catch (e) {
      developer.log('deleteHistoricEscrow error: $e', name: 'refunds');
      throw Exception('Failed to delete escrow: $e');
    }
  }

  Future<void> ensureGameStream() async {
    final g = game;
    if (g == null) return;
    if (!(g.gameStarted || g.phase != pr.GamePhase.WAITING)) return;
    try {
      await Golib.startGameStream();
    } catch (e) {
      rethrow;
    }
  }

  Future<void> leaveTable() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await Golib.leavePokerTable();
    } catch (_) {
      // ignore; try to clean local state anyway
    } finally {
      currentTableId = null;
      game = null;
      _iAmReady = false;
      _seated = false;
      _showTableView = false;
      _lastBoundEscrowId = null;
      _lastBoundEscrowReady = false;
      _resetPresignState();
      _clearShowdownState();
      _state = PokerState.browsingTables;
      notifyListeners();
      // PLAYER_LEFT notification updates the table state
    }
  }

  /// Clear showdown-related state when leaving table or resetting.
  void _clearShowdownState() {
    _pendingGameEndMessage = null;
    gameEndingMessage = '';
    _showdown = null;
    _showdownCaptured = false;
    lastWinners = const [];
    _showdownHandsCache.clear();
  }

  // Capture the latest known hand state to back the last-showdown dialog
  // when the table ends without an explicit SHOWDOWN_RESULT.
  void _cacheShowdownSnapshot(pr.Showdown? showdown) {
    // Only persist explicit showdown payloads from the server.
    if (_showdownCaptured || showdown == null) return;

    debugPrint(
        '[SHOWDOWN_CACHE_IN] players=${showdown.players.length} board=${showdown.board.length} pot=${showdown.pot}');

    final merged = showdown.players.map(_uiPlayerFromShowdown).toList();
    _showdown = UiShowdownState(
      players: List.unmodifiable(_hydrateShowdownHands(merged)),
      communityCards: List.unmodifiable(showdown.board),
      pot: showdown.pot.toInt(),
    );
    // Seed cache and live game view with revealed showdown hands so the table
    // shows contest winners immediately.
    for (final sp in showdown.players) {
      if (sp.holeCards.isNotEmpty) {
        final hand = List<pr.Card>.unmodifiable(sp.holeCards);
        _showdownHandsCache[sp.playerId] = hand;
        if (sp.playerId == playerId) {
          _myHoleCardsCache = hand;
        }
      }
    }
    if (game != null) {
      final byID = {for (final sp in showdown.players) sp.playerId: sp};
      final updated = game!.players.map((p) {
        final sp = byID[p.id];
        if (sp == null || sp.holeCards.isEmpty) return p;
        return p.withHand(sp.holeCards).copyWith(cardsRevealed: true);
      }).toList();
      game = game!.copyWith(players: List.unmodifiable(updated));
    }

    debugPrint(
        '[SHOWDOWN_CACHE] players=${showdownPlayers.length} board=${showdownCommunityCards.length} pot=$showdownPot');

    _showdownCaptured = true;
  }

  UiPlayer _uiPlayerFromShowdown(pr.ShowdownPlayer sp) {
    final folded = sp.finalState == pr.PlayerState.PLAYER_STATE_FOLDED;
    final allIn = sp.finalState == pr.PlayerState.PLAYER_STATE_ALL_IN;
    var name = sp.name;
    if (name.isEmpty) {
      // Fallback to the latest game snapshot so showdown UI still shows labels
      final fromGame =
          game?.players.firstWhereOrNull((p) => p.id == sp.playerId);
      name = fromGame?.name ?? '';
    }
    return UiPlayer(
      id: sp.playerId,
      name: name,
      balance: 0,
      hand: List<pr.Card>.unmodifiable(sp.holeCards),
      currentBet: 0,
      folded: folded,
      isTurn: false,
      isAllIn: allIn,
      isDealer: false,
      isSmallBlind: false,
      isBigBlind: false,
      isReady: false,
      isDisconnected: false,
      handDesc: '',
      cardsRevealed: sp.holeCards.isNotEmpty,
    );
  }

  Future<void> _restoreCurrentTable() async {
    try {
      // Use golib to discover any existing table and keep Go client in sync
      if (_restoring) return;
      _restoring = true;
      final tid = await Golib.getPokerCurrentTable();
      if (tid.isEmpty) return;

      // If we're already seated on this table, avoid re-joining; ensure stream/state.
      if (_seated && currentTableId == tid) {
        await refreshGameState();
        // If game is already started, ensure game stream is active
        if (game != null &&
            (game!.gameStarted || game!.phase != pr.GamePhase.WAITING)) {
          try {
            await Golib.startGameStream();
          } catch (e) {
            rethrow;
          }
        }
        return;
      }

      // Re-join via golib to reconcile client-side state and attach streams
      await joinTable(tid);
    } catch (_) {
      // ignore
    } finally {
      _restoring = false;
    }
  }

  // -------- Ready / Unready & show/hide cards ----------
  Future<void> setReady() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await Golib.setPlayerReady();
      _iAmReady = true;
      notifyListeners();
      // Attempt to start presigning if conditions are met
      unawaited(_maybeStartPresignForCurrentTable());
    } catch (e) {
      errorMessage = 'Set ready failed: $e';
      notifyListeners();
    }
  }

  Future<void> setUnready() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await Golib.setPlayerUnready();
      _iAmReady = false;
      notifyListeners();
    } catch (e) {
      errorMessage = 'Set unready failed: $e';
      notifyListeners();
    }
  }

  /// Attempts to start presigning when conditions are met for escrow-backed tables.
  /// Called automatically when:
  /// - Table has a buy-in (escrow-backed)
  /// - Player has a funded escrow
  /// - All players are ready
  /// - Presigning is not already in progress or completed
  Future<void> _maybeStartPresignForCurrentTable() async {
    final tid = currentTableId;
    if (tid == null) return;

    // Get current table info
    final table = tables.firstWhereOrNull((t) => t.id == tid);
    if (table == null) return;

    // Only for escrow-backed tables
    if (table.buyInAtoms <= 0) return;

    // Don't start if game already started
    if (table.gameStarted) return;

    // Check escrow status
    final escrowId = cachedEscrowId;
    final escrowReady = cachedEscrowReady;
    if (escrowId.isEmpty || !escrowReady) return;

    // Don't re-presign if already in progress
    if (_presignInProgress) return;

    // Get players from game state (server returns players even before game starts)
    final g = game;
    if (g == null || g.players.isEmpty) return;
    final players = g.players;

    // Find my player
    final mePlayer = players.firstWhereOrNull((p) => p.id == playerId);
    if (mePlayer == null || mePlayer.tableSeat < 0) return;

    // Don't re-presign if already complete (server state)
    if (mePlayer.presignComplete) return;

    // Check if we have enough players
    if (players.length < table.minPlayers) return;

    // Check if all players are ready and have funded escrows
    final allReadyWithEscrow = players
        .every((p) => p.isReady && p.escrowId.isNotEmpty && p.escrowReady);
    if (!allReadyWithEscrow) return;

    // For poker, matchID is just tableID
    final matchId = tid;

    _presignInProgress = true;
    _presignError = '';
    notifyListeners();

    developer.log(
      'Starting settlement presign for match $matchId',
      name: 'settlement',
    );

    try {
      // Get the session private key from the escrow
      final escrowInfo = await Golib.getEscrowById(escrowId);
      final compPriv =
          escrowInfo['comp_priv'] ?? escrowInfo['session_priv'] ?? '';
      if (compPriv is! String || compPriv.isEmpty) {
        throw Exception('Session private key not found for escrow $escrowId');
      }

      await Golib.startPreSign(
        matchId: matchId,
        tableId: tid,
        escrowId: escrowId,
        compPriv: compPriv,
      );

      _presignInProgress = false;
      successMessage = 'Settlement prepared for this match.';
      developer.log(
        'Presign completed for match $matchId',
        name: 'settlement',
      );
    } catch (e, st) {
      _presignInProgress = false;
      _presignError = '$e';
      developer.log(
        'startPreSign error: $e',
        name: 'settlement',
        stackTrace: st,
      );
      errorMessage = 'Settlement presign failed: $e';
    }
    notifyListeners();
  }

  /// Resets local presign state when leaving a table.
  void _resetPresignState() {
    _presignInProgress = false;
    _presignError = '';
  }

  Future<void> showCards() async {
    if (currentTableId == null || !_seated) return;
    try {
      await Golib.showCards();
    } catch (e) {
      errorMessage = 'Show cards failed: $e';
      notifyListeners();
    }
  }

  Future<void> hideCards() async {
    if (currentTableId == null || !_seated) return;
    try {
      await Golib.hideCards();
    } catch (e) {
      errorMessage = 'Hide cards failed: $e';
      notifyListeners();
    }
  }

  // -------- Game state helpers ----------
  double get timebankRemainingSeconds {
    final dl = _turnDeadline;
    if (dl == null) return 0;
    final rem = dl.difference(DateTime.now());
    if (rem.isNegative) return 0;
    return rem.inMilliseconds / 1000.0;
  }

  // -------- Actions (bet/call/check/fold) ----------
  Future<bool> makeBet(int amountChips) async {
    final tid = currentTableId;
    if (tid == null) return false;
    try {
      await Golib.makeBet(MakeBetArgs(amountChips));
      _soundService.playBet();
      return true;
    } catch (e) {
      errorMessage = 'Bet failed: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> callBet() async {
    final tid = currentTableId;
    if (tid == null) return false;
    try {
      await Golib.callBet();
      _soundService.playCall();
      return true;
    } catch (e) {
      errorMessage = 'Call failed: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> check() async {
    final tid = currentTableId;
    if (tid == null) return false;
    try {
      await Golib.checkBet();
      _soundService.playCheck();
      return true;
    } catch (e) {
      errorMessage = 'Check failed: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> fold() async {
    final tid = currentTableId;
    if (tid == null) return false;
    try {
      await Golib.foldBet();
      return true;
    } catch (e) {
      errorMessage = 'Fold failed: $e';
      notifyListeners();
      return false;
    }
  }

  // -------- Queries ----------
  Future<void> refreshGameState() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      final respMap = await Golib.getGameState();
      // Parse simple JSON DTO and convert to protobuf GameUpdate
      final gameStateJson = respMap['game_state'] as Map<String, dynamic>;
      final dto = GameUpdateDTO.fromJson(gameStateJson);
      final gameUpdate = dto.toProtobuf();
      _stashShowdownHands(gameUpdate);
      game = _keepGameWithShowdownPlayers(UiGameState.fromUpdate(gameUpdate));
      final mePlayer = me;
      if (mePlayer != null && mePlayer.escrowId.isNotEmpty) {
        _lastBoundEscrowId = mePlayer.escrowId;
        _lastBoundEscrowReady = mePlayer.escrowReady;
      }
      // Keep coarse UI state in sync even when attaching mid-hand.
      // This mirrors the logic in _onGameUpdate so that the UI shows
      // the table (and hole cards) immediately on reconnect/restore.
      final phase = gameUpdate.phase;
      if (phase == pr.GamePhase.SHOWDOWN) {
        _enterShowdownState();
        _maybeRefreshLastWinners();
      } else if (gameUpdate.gameStarted || phase != pr.GamePhase.WAITING) {
        // Game is active if gameStarted flag is true OR phase is not WAITING
        _state = PokerState.handInProgress;
      } else {
        _state = PokerState.inLobby;
      }

      notifyListeners();
      // After getting fresh game state, check if presigning should start
      unawaited(_maybeStartPresignForCurrentTable());
    } catch (e) {
      errorMessage = 'GetGameState failed: $e';
      notifyListeners();
    }
  }

  Future<void> _refreshLastWinners() async {
    final tid = currentTableId;
    if (tid == null || _lastWinnersLoading) return;
    _lastWinnersLoading = true;
    try {
      final respMap = await Golib.getLastWinners();
      // Convert JSON map back to protobuf GetLastWinnersResponse
      final winnersJson = respMap['winners'] as List<dynamic>;
      final winners =
          winnersJson.map(_winnerFromDynamic).whereType<UiWinner>().toList();
      lastWinners = List.unmodifiable(winners);
      lastShowdownFxMs = DateTime.now().millisecondsSinceEpoch;
      notifyListeners();
    } catch (_) {
      // empty cache
      lastWinners = const [];
    } finally {
      _lastWinnersLoading = false;
    }
  }

  pr.HandRank parseHandRank(dynamic hrRaw) {
    if (hrRaw is pr.HandRank) {
      return hrRaw;
    }
    if (hrRaw is num) {
      return pr.HandRank.valueOf(hrRaw.toInt()) ?? pr.HandRank.HIGH_CARD;
    }
    if (hrRaw is String) {
      final parsed = int.tryParse(hrRaw);
      if (parsed != null) {
        return pr.HandRank.valueOf(parsed) ?? pr.HandRank.HIGH_CARD;
      }
      final normalized = hrRaw.toUpperCase();
      final stripped = normalized.startsWith('HAND_RANK_')
          ? normalized.substring('HAND_RANK_'.length)
          : normalized;
      final match = pr.HandRank.values
          .firstWhereOrNull((h) => h.name.toUpperCase() == stripped);
      return match ?? pr.HandRank.HIGH_CARD;
    }
    return pr.HandRank.HIGH_CARD;
  }

  UiWinner? _winnerFromDynamic(dynamic w) {
    try {
      final winnerJsonStr = jsonEncode(w);
      final parsed = pr.Winner.fromJson(winnerJsonStr);
      return UiWinner.fromProto(parsed);
    } catch (e) {
      if (w is Map<String, dynamic>) {
        final pid = (w['playerId'] ?? w['player_id'] ?? '').toString();
        final winningsRaw = w['winnings'];
        final winnings = winningsRaw is num
            ? winningsRaw.toInt()
            : int.tryParse('$winningsRaw') ?? 0;
        final hrRaw = w['handRank'] ?? w['hand_rank'] ?? w['rank'];
        final handRank = parseHandRank(hrRaw);
        final bestHandRaw = w['bestHand'] ?? w['best_hand'] ?? [];
        final List<pr.Card> bestHand = [];
        if (bestHandRaw is List) {
          for (final c in bestHandRaw) {
            if (c is Map<String, dynamic>) {
              final suit = c['suit']?.toString() ?? '';
              final value = c['value']?.toString() ?? '';
              final card = pr.Card()
                ..suit = suit
                ..value = value;
              bestHand.add(card);
            }
          }
        }
        return UiWinner(
          playerId: pid,
          handRank: handRank,
          bestHand: List.unmodifiable(bestHand),
          winnings: winnings,
        );
      }
    }
    return null;
  }

  Future<pr.EvaluateHandResponse?> evaluateCards(List<pr.Card> cards) async {
    try {
      // Convert protobuf cards to CardArg format
      final cardArgs = cards.map((c) {
        // Convert suit string to int (Spades=0, Hearts=1, Diamonds=2, Clubs=3)
        int suitInt = 0;
        switch (c.suit) {
          case 'Spades':
            suitInt = 0;
            break;
          case 'Hearts':
            suitInt = 1;
            break;
          case 'Diamonds':
            suitInt = 2;
            break;
          case 'Clubs':
            suitInt = 3;
            break;
        }
        // Convert value string to int (2-10, J=11, Q=12, K=13, A=14)
        int valueInt = int.tryParse(c.value) ?? 0;
        if (c.value == 'J') valueInt = 11;
        if (c.value == 'Q') valueInt = 12;
        if (c.value == 'K') valueInt = 13;
        if (c.value == 'A') valueInt = 14;
        return CardArg(suitInt, valueInt);
      }).toList();

      final respMap = await Golib.evaluateHand(EvaluateHandArgs(cardArgs));
      // Convert JSON map back to protobuf EvaluateHandResponse
      final respJsonStr = jsonEncode(respMap);
      return pr.EvaluateHandResponse.fromJson(respJsonStr);
    } catch (e) {
      errorMessage = 'EvaluateHand failed: $e';
      notifyListeners();
      return null;
    }
  }

  // -------- Helpers ----------
  UiPlayer? get me => game?.players.firstWhereOrNull((p) => p.id == playerId);
  String get cachedEscrowId {
    final mePlayer = me;
    if (mePlayer != null && mePlayer.escrowId.isNotEmpty) {
      return mePlayer.escrowId;
    }
    return _lastBoundEscrowId ?? '';
  }

  bool get cachedEscrowReady {
    final mePlayer = me;
    if (mePlayer != null && mePlayer.escrowId.isNotEmpty) {
      return mePlayer.escrowReady;
    }
    return _lastBoundEscrowReady;
  }

  String get cachedEscrowState {
    final mePlayer = me;
    if (mePlayer != null && mePlayer.escrowState.isNotEmpty) {
      return mePlayer.escrowState;
    }
    return _lastBoundEscrowState;
  }

  bool get iAmReady => _iAmReady;

  /// Returns true when the current player is already all-in and the hand is
  /// auto-advancing, meaning no manual decisions remain.
  bool get autoAdvanceAllIn => isAutoAdvanceAllIn(game);

  bool get isMyTurn => game != null && game!.currentPlayerId == playerId;

  bool get canAct {
    final g = game;
    if (g == null) return false;
    final actionablePhase = g.phase == pr.GamePhase.PRE_FLOP ||
        g.phase == pr.GamePhase.FLOP ||
        g.phase == pr.GamePhase.TURN ||
        g.phase == pr.GamePhase.RIVER;
    if (!actionablePhase) return false;
    return isMyTurn && !autoAdvanceAllIn;
  }

  bool get canBet {
    final g = game;
    if (g == null) return false;
    if (!canAct) return false;
    // You can tighten with min/max raise & balance checks in the widget.
    return g.phase == pr.GamePhase.PRE_FLOP ||
        g.phase == pr.GamePhase.FLOP ||
        g.phase == pr.GamePhase.TURN ||
        g.phase == pr.GamePhase.RIVER;
  }

  /// Play turn notification sound (called when timebank countdown starts)
  void playTurnNotificationSound() {
    _soundService.playTurnNotification();
  }

  Map<String, dynamic>? _decodeJsonMap(String raw) {
    try {
      final parsed = jsonDecode(raw);
      if (parsed is Map<String, dynamic>) return parsed;
      if (parsed is Map) {
        return parsed.map((key, value) => MapEntry('$key', value));
      }
    } catch (_) {
      // ignore parse errors
    }
    return null;
  }

  String _firstNonEmpty(dynamic a, String fallback) {
    final aa = (a ?? '').toString().trim();
    if (aa.isNotEmpty) {
      return aa;
    }
    return fallback;
  }

  void clearError() {
    errorMessage = '';
    notifyListeners();
  }
}
