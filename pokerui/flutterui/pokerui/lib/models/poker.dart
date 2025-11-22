// lib/models/poker_model.dart
import 'dart:async';
import 'dart:convert';
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

/// -------- UI-facing enums --------
enum PokerState {
  idle,
  browsingTables,
  inLobby, // seated, waiting / readying
  handInProgress, // active betting streets
  showdown, // results surfaced
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
    );
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
  final String hostId;
  final List<UiPlayer> players;
  final int smallBlind;
  final int bigBlind;
  final int maxPlayers;
  final int minPlayers;
  final int currentPlayers;
  final int minBalanceAtoms;
  final int buyInAtoms;
  final pr.GamePhase phase;
  final bool gameStarted;
  final bool allReady;

  const UiTable({
    required this.id,
    required this.hostId,
    required this.players,
    required this.smallBlind,
    required this.bigBlind,
    required this.maxPlayers,
    required this.minPlayers,
    required this.currentPlayers,
    required this.minBalanceAtoms,
    required this.buyInAtoms,
    required this.phase,
    required this.gameStarted,
    required this.allReady,
  });

  factory UiTable.fromProto(pr.Table t) => UiTable(
        id: t.id,
        hostId: t.hostId,
        players: List.unmodifiable(t.players.map(UiPlayer.fromProto)),
        smallBlind: t.smallBlind.toInt(),
        bigBlind: t.bigBlind.toInt(),
        maxPlayers: t.maxPlayers,
        minPlayers: t.minPlayers,
        currentPlayers: t.currentPlayers,
        minBalanceAtoms: t.minBalance.toInt(),
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
}

/// -------- The main ChangeNotifier --------
class PokerModel extends ChangeNotifier {
  // Identity
  final String playerId;

  // UI/state
  PokerState _state = PokerState.idle;
  PokerState get state => _state;

  String? currentTableId;
  UiGameState? game;
  List<UiTable> tables = const [];
  List<UiWinner> lastWinners = const [];
  int lastShowdownFxMs = 0; // monotonic trigger for showdown chip animation
  String errorMessage = '';
  int myAtomsBalance = 0; // DCR atoms (wallet balance for buy-in requirements)

  // Cache hero hole cards for use at showdown when the server may omit them
  List<pr.Card> _myHoleCardsCache = const [];
  List<pr.Card> get myHoleCardsCache => _myHoleCardsCache;

  // Lightweight FX hook: last bet/call animation trigger
  UiBetFx? lastBetFx; // set when a player's currentBet increases

  // Timebank tracking (server-provided deadline)
  int timeBankSeconds = 30; // default when unknown
  DateTime? _turnDeadline;

  // Cached readiness
  bool _iAmReady = false;
  bool _seated = false; // track whether user is seated at any table
  bool _restoring = false; // guard against repeated restore/join loops
  // Track per-player show/hide state from notifications
  final Map<String, bool> playersShowingCards = {};
  bool get myCardsShown => playersShowingCards[playerId] ?? false;

  // Subscription to poker notifications coming from golib.
  StreamSubscription<pr.Notification>? _pokerNtfnSub;
  // Subscription to game updates coming from golib.
  StreamSubscription<pr.GameUpdate>? _gameUpdateSub;

  PokerModel({
    required this.playerId,
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
      cfg.wantsLogNtfns,
    );

    // Initialize the Go library client
    final localInfo = await Golib.initClient(initClientArgs);

    // Use the player ID from the Go library initialization
    final playerId = localInfo.id;

    return PokerModel(
      playerId: playerId,
    );
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
    await refreshTables();
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
        // No-op: UI remains usable; we’ll act on reconnect events.
        break;
      case pr.NotificationType.TABLE_CREATED:
      case pr.NotificationType.TABLE_REMOVED:
      case pr.NotificationType.PLAYER_JOINED:
      case pr.NotificationType.PLAYER_LEFT:
      case pr.NotificationType.BALANCE_UPDATED:
      case pr.NotificationType.PLAYER_READY:
      case pr.NotificationType.PLAYER_UNREADY:
      case pr.NotificationType.ALL_PLAYERS_READY:
        // Refresh lightweight lists/balances; avoid spamming server.
        unawaited(refreshTables());
        unawaited(_refreshBalance());
        break;

      case pr.NotificationType.NEW_HAND_STARTED:
        playersShowingCards.clear();
        // Clear cached hero hole cards for the new hand to avoid stale display
        _myHoleCardsCache = const [];
        // Clear any stale bet FX at the start of a new hand
        lastBetFx = null;
        // Don't call refreshGameState here - wait for GameUpdate from stream
        // The stream will send the game state with hand cards
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
      case pr.NotificationType.BET_MADE:
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
      case pr.NotificationType.PLAYER_FOLDED:
      case pr.NotificationType.SMALL_BLIND_POSTED:
      case pr.NotificationType.BIG_BLIND_POSTED:
      case pr.NotificationType.SHOWDOWN_RESULT:
        // Don't call refreshGameState - stream will send GameUpdate
        // Only update UI state if needed, but don't poll
        break;

      case pr.NotificationType.CARDS_SHOWN:
        if (n.playerId.isNotEmpty) {
          playersShowingCards[n.playerId] = true;
          notifyListeners();
        }
        break;

      case pr.NotificationType.CARDS_HIDDEN:
        if (n.playerId.isNotEmpty) {
          playersShowingCards[n.playerId] = false;
          notifyListeners();
        }
        break;

      default:
        break;
    }
  }

  // -------- Game Updates (from game stream) ----------
  void _onGameUpdate(pr.GameUpdate gameUpdate) {
   // Update game state from the stream update
    game = UiGameState.fromUpdate(gameUpdate);
    // Update UI state based on game phase
    final phase = gameUpdate.phase;
    if (phase == pr.GamePhase.SHOWDOWN) {
      _state = PokerState.showdown;
      unawaited(_refreshLastWinners());
    } else if (gameUpdate.gameStarted || phase != pr.GamePhase.WAITING) {
      // Game is active if gameStarted flag is true OR phase is not WAITING
      _state = PokerState.handInProgress;
    } else {
      _state = PokerState.inLobby;
    }

    notifyListeners();
  }

  // -------- Lobby / Tables ----------
  Future<void> refreshTables() async {
    try {
      // Prefer Golib for consistent identity and simpler transport
      final list = await Golib.getPokerTables();
      // Map plugin PokerTable -> UiTable (minimal fields used by lobby UI)
      tables = List.unmodifiable(list.map((t) => UiTable(
            id: t.id,
            hostId: t.hostId,
            players: const [],
            smallBlind: t.smallBlind,
            bigBlind: t.bigBlind,
            maxPlayers: t.maxPlayers,
            minPlayers: t.minPlayers,
            currentPlayers: t.currentPlayers,
            minBalanceAtoms: t.minBalance,
            buyInAtoms: t.buyIn,
            // Phase not provided by plugin; lobby UI already shows status via gameStarted
            phase: pr.GamePhase.WAITING,
            gameStarted: t.gameStarted,
            allReady: t.allPlayersReady,
          )));
      // If not seated, keep UI in browsing mode.
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

  Future<void> _refreshBalance() async {
    try {
      final res = await Golib.getPokerBalance();
      final b = res['balance'];
      if (b is int) {
        myAtomsBalance = b;
        notifyListeners();
      }
    } catch (_) {
      // Best-effort; keep old balance.
    }
  }

  Future<String?> createTable({
    required int smallBlindChips,
    required int bigBlindChips,
    required int maxPlayers,
    required int minPlayers,
    required int minBalanceAtoms,
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
        minBalanceAtoms,
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
      await refreshTables();
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
        // Ensure state is up-to-date via golib.
        await refreshGameState();
        await ensureGameStream();
        unawaited(_refreshLastWinners());
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
      await refreshTables();
      // Refresh game state and winners via golib instead of a direct gRPC stream.
      await refreshGameState();
      await ensureGameStream();
      unawaited(_refreshLastWinners());
      notifyListeners();
      return true;
    } catch (e) {
      errorMessage = 'Join failed: $e';
      notifyListeners();
      return false;
    }
  }

  Future<void> ensureGameStream() async {
    final g = game;
    if (g==null) return;
    if(!(g.gameStarted || g.phase != pr.GamePhase.WAITING)) return;
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
      playersShowingCards.clear();
      _state = PokerState.browsingTables;
      notifyListeners();
      unawaited(refreshTables());
    }
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

  Future<void> showCards() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await Golib.showCards();
      playersShowingCards[playerId] =
          true; // optimistic update; server will confirm via notification
      notifyListeners();
    } catch (e) {
      errorMessage = 'Show cards failed: $e';
      notifyListeners();
    }
  }

  Future<void> hideCards() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await Golib.hideCards();
      playersShowingCards[playerId] = false; // optimistic update
      notifyListeners();
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
      // Refresh balance in background
      unawaited(_refreshBalance());
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
      game = UiGameState.fromUpdate(gameUpdate);

      // Keep coarse UI state in sync even when attaching mid-hand.
      // This mirrors the logic in _onGameUpdate so that the UI shows
      // the table (and hole cards) immediately on reconnect/restore.
      final phase = gameUpdate.phase;
      if (phase == pr.GamePhase.SHOWDOWN) {
        _state = PokerState.showdown;
        unawaited(_refreshLastWinners());
      } else if (gameUpdate.gameStarted || phase != pr.GamePhase.WAITING) {
        // Game is active if gameStarted flag is true OR phase is not WAITING
        _state = PokerState.handInProgress;
      } else {
        _state = PokerState.inLobby;
      }

      notifyListeners();
    } catch (e) {
      errorMessage = 'GetGameState failed: $e';
      notifyListeners();
    }
  }

  Future<void> _refreshLastWinners() async {
    final tid = currentTableId;
    if (tid == null) return;
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
    }
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
        pr.HandRank handRank = pr.HandRank.HIGH_CARD;
        if (hrRaw is int) {
          handRank = pr.HandRank.valueOf(hrRaw) ?? pr.HandRank.HIGH_CARD;
        }
        if (hrRaw is String) {
          final parsed = int.tryParse(hrRaw);
          if (parsed != null) {
            handRank = pr.HandRank.valueOf(parsed) ?? pr.HandRank.HIGH_CARD;
          }
        }
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

  bool get iAmReady => _iAmReady;

  bool get isMyTurn => game != null && game!.currentPlayerId == playerId;

  bool get canBet {
    final g = game;
    if (g == null) return false;
    if (!isMyTurn) return false;
    // You can tighten with min/max raise & balance checks in the widget.
    return g.phase == pr.GamePhase.PRE_FLOP ||
        g.phase == pr.GamePhase.FLOP ||
        g.phase == pr.GamePhase.TURN ||
        g.phase == pr.GamePhase.RIVER;
  }

  void clearError() {
    errorMessage = '';
    notifyListeners();
  }
}
