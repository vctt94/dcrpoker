// lib/models/poker_model.dart
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:collection/collection.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:golib_plugin/grpc/generated/poker.pbgrpc.dart' as prpc;
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart'
    show
        MakeBetArgs,
        EvaluateHandArgs,
        CardArg,
        CreatePokerTableArgs,
        JoinPokerTableArgs,
        InitClient;
import 'package:grpc/grpc.dart';
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
  // Injected RPC clients - kept for streams only (will be removed once streams are handled by golib)
  prpc.LobbyServiceClient? lobby;
  prpc.PokerServiceClient? poker;

  // Identity
  final String playerId;

  // UI/state
  PokerState _state = PokerState.idle;
  PokerState get state => _state;

  String? currentTableId;
  UiGameState? game;
  List<UiTable> tables = const [];
  List<UiWinner> lastWinners = const [];
  String errorMessage = '';
  int myAtomsBalance = 0; // DCR atoms (wallet balance for buy-in requirements)

  // Cache hero hole cards for use at showdown when the server may omit them
  List<pr.Card> _myHoleCardsCache = const [];
  List<pr.Card> get myHoleCardsCache => _myHoleCardsCache;

  // Lightweight FX hook: last bet/call animation trigger
  UiBetFx? lastBetFx; // set when a player's currentBet increases

  // Streams
  StreamSubscription<pr.Notification>? _ntfnSub;
  StreamSubscription<pr.GameUpdate>? _gameSub;

  // Timebank tracking (server-provided deadline)
  int timeBankSeconds = 30; // default when unknown
  DateTime? _turnDeadline;

  // Backoff
  int _retries = 0;
  Timer? _backoffTimer;

  // Cached readiness
  bool _iAmReady = false;
  bool _seated = false; // track whether user is seated at any table
  bool _restoring = false; // guard against repeated restore/join loops
  // Track per-player show/hide state from notifications
  final Map<String, bool> playersShowingCards = {};
  bool get myCardsShown => playersShowingCards[playerId] ?? false;

  PokerModel({
    this.lobby,
    this.poker,
    required this.playerId,
  });

  /// Factory method to create PokerModel from Config
  static Future<PokerModel> fromConfig(
      Config cfg, NotificationModel notificationModel) async {
    print('DEBUG: fromConfig - starting with cfg: $cfg');
    // Initialize the Go library with configuration
    final initClientArgs = InitClient(
      cfg.serverAddr,
      cfg.grpcCertPath,
      cfg.dataDir,
      '${cfg.dataDir}/logs/pokerui.log',
      cfg.payoutAddress,
      cfg.debugLevel,
      cfg.wantsLogNtfns,
      cfg.rpcWebsocketURL,
      cfg.rpcCertPath,
      cfg.rpcClientCertPath,
      cfg.rpcClientKeyPath,
      cfg.rpcUser,
      cfg.rpcPass,
    );

    // Initialize the Go library client
    final localInfo = await Golib.initClient(initClientArgs);
    print("*****************");
    print(localInfo);
    print(
        'DEBUG: fromConfig - Golib.initClient returned id=${localInfo.id} nick=${localInfo.nick}');

    // TODO: Streams (startGameStream, startNotificationStream) still need gRPC clients
    // For now, create gRPC channel only for streams
    // Once streams are handled by golib, remove this
    final channel = await _createGrpcChannel(cfg);
    final lobby = prpc.LobbyServiceClient(channel);
    final poker = prpc.PokerServiceClient(channel);

    // Use the player ID from the Go library initialization
    final playerId = localInfo.id;

    return PokerModel(
      lobby: lobby,
      poker: poker,
      playerId: playerId,
    );
  }

  /// Create gRPC channel from config - only needed for streams until they're handled by golib
  static Future<ClientChannel> _createGrpcChannel(Config cfg) async {
    final serverAddr = cfg.serverAddr;

    // Parse host and port
    final parts = serverAddr.split(':');
    final host = parts[0];
    final port = int.tryParse(parts[1]) ?? 50051;

    // Create channel options with TLS credentials if cert path is provided
    ChannelOptions options;
    if (cfg.grpcCertPath.isNotEmpty) {
      // Load the server certificate
      final certBytes = await File(cfg.grpcCertPath).readAsBytes();
      final credentials = ChannelCredentials.secure(
        certificates: certBytes,
        authority:
            host, // Use the host as the authority for certificate validation
      );
      options = ChannelOptions(credentials: credentials);
    } else {
      // Fallback to insecure if no cert path provided
      options = ChannelOptions(credentials: ChannelCredentials.insecure());
    }

    return ClientChannel(
      host,
      port: port,
      options: options,
    );
  }

  // -------- Lifecycle ----------
  Future<void> init() async {
    print('DEBUG: PokerModel.init - begin (playerId=$playerId)');
    await _startNotificationStream();
    await refreshTables();
    // If server remembers seat, restore:
    await _restoreCurrentTable();
  }

  @override
  void dispose() {
    _ntfnSub?.cancel();
    _gameSub?.cancel();
    _backoffTimer?.cancel();
    super.dispose();
  }

  // -------- Notifications ----------
  Future<void> _startNotificationStream() async {
    await _ntfnSub?.cancel();
    try {
      final stream = lobby!.startNotificationStream(
        pr.StartNotificationStreamRequest()..playerId = playerId,
      );
      _ntfnSub = stream.listen(_onNotification,
          onError: _onStreamError, onDone: _onStreamDone, cancelOnError: false);
      print('DEBUG: Notification stream attached for playerId=$playerId');
    } catch (e) {
      _scheduleBackoff(() => _startNotificationStream());
    }
  }

  void _onNotification(pr.Notification n) {
    print(
        'DEBUG: Notification received type=${n.type} tableId=${n.tableId} playerId=${n.playerId}');
    switch (n.type) {
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
        notifyListeners();
        break;
      case pr.NotificationType.GAME_STARTED:
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
        break;
      case pr.NotificationType.CHECK_MADE:
      case pr.NotificationType.PLAYER_FOLDED:
      case pr.NotificationType.SMALL_BLIND_POSTED:
      case pr.NotificationType.BIG_BLIND_POSTED:
      case pr.NotificationType.SHOWDOWN_RESULT:
        // Game stream will drive UI; still useful for toasts.
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

  void _onStreamError(Object e, StackTrace st) {
    errorMessage = 'Stream error: $e';
    notifyListeners();
    _scheduleBackoff(() => _startNotificationStream());
  }

  void _onStreamDone() {
    _scheduleBackoff(() => _startNotificationStream());
  }

  void _scheduleBackoff(FutureOr<void> Function() retry) {
    _backoffTimer?.cancel();
    final ms = min(15000, 500 * (1 << _retries)); // 0.5s, 1s, 2s, ... cap 15s
    _backoffTimer = Timer(Duration(milliseconds: ms), () {
      _retries = min(_retries + 1, 6);
      retry();
    });
  }

  void _resetBackoff() {
    _retries = 0;
    _backoffTimer?.cancel();
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
    int autoStartMs = 0,
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
        print(
            'DEBUG: joinTable dedup - already seated at $tableId; reattaching stream');
        await _attachGameStream();
        // Stream drives UI; fetch winners in background.
        unawaited(_refreshLastWinners());
        _resetBackoff();
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
      _state = PokerState.inLobby;
      print('DEBUG: joinTable ok - tableId=$tableId playerId=$playerId');
      await refreshTables();
      await _attachGameStream(); // subscribe immediately with this.playerId
      // Let stream drive UI; fetch winners in background.
      unawaited(_refreshLastWinners());
      _resetBackoff();
      notifyListeners();
      return true;
    } catch (e) {
      errorMessage = 'Join failed: $e';
      notifyListeners();
      return false;
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
      await _detachGameStream();
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
      print('DEBUG: _restoreCurrentTable - tid=$tid');
      if (tid.isEmpty) return;

      // If we're already seated on this table, avoid re-joining; ensure stream/state.
      if (_seated && currentTableId == tid) {
        print('DEBUG: _restoreCurrentTable - already at $tid, skipping rejoin');
        await _attachGameStream();
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

  // -------- Game stream & state ----------
  Future<void> _attachGameStream() async {
    await _gameSub?.cancel();
    final tid = currentTableId;
    if (tid == null) return;

    try {
      print('DEBUG: Attaching game stream - tableId=$tid playerId=$playerId');
      final stream = poker!.startGameStream(pr.StartGameStreamRequest()
        ..playerId = playerId
        ..tableId = tid);
      _gameSub = stream.listen(_onGameUpdate,
          onError: _onGameStreamError, onDone: _onGameStreamDone);
    } catch (e) {
      _scheduleBackoff(_attachGameStream);
    }
  }

  Future<void> _detachGameStream() async {
    await _gameSub?.cancel();
    _gameSub = null;
  }

  void _onGameUpdate(pr.GameUpdate u) {
    // Ignore updates if not seated or not for the active table
    if (!_seated) {
      print('DEBUG: Ignoring game update - not seated');
      return;
    }
    final tid = currentTableId;
    if (tid != null && u.tableId != tid) {
      print('DEBUG: Ignoring game update - wrong table: ${u.tableId} vs $tid');
      return;
    }

    print(
        'DEBUG: Processing game update - phase: ${u.phase}, gameStarted: ${u.gameStarted}, currentPlayer: ${u.currentPlayer}');
    final next = UiGameState.fromUpdate(u);
    // Update hero hole cards cache when available
    final heroNow = next.players.firstWhereOrNull((p) => p.id == playerId);
    if (heroNow != null && heroNow.hand.isNotEmpty) {
      _myHoleCardsCache = List<pr.Card>.from(heroNow.hand);
    }
    // Rely on explicit notifications for bet/call FX; avoid inferring from snapshots
    game = next;

    final myP = me;
    final handCnt = myP?.hand.length ?? 0;
    final playersCnt = game?.players.length ?? u.players.length;
    print(
        'DEBUG: GameUpdate snapshot - players=$playersCnt myHandCnt=$handCnt myId=$playerId curr=${u.currentPlayer}');
    if (handCnt > 0) {
      final h = myP!.hand;
      print(
          'DEBUG: My cards: ${h.map((c) => '${c.value} of ${c.suit}').join(', ')}');
    }

    // Drive coarse UI state from server phase:
    // - SHOWDOWN -> showdown view
    // - Any non-WAITING phase -> hand in progress (even NEW_HAND_DEALING)
    // This avoids relying solely on gameStarted, which can lag in some snapshots.
    if (u.phase == pr.GamePhase.SHOWDOWN) {
      _state = PokerState.showdown;
      unawaited(_refreshLastWinners());
      _turnDeadline = null;
      // Clear transient FX; UI follows server phase strictly.
      lastBetFx = null;
    } else if (u.phase != pr.GamePhase.WAITING) {
      _state = PokerState.handInProgress;
    } else {
      _state = PokerState.inLobby;
      _turnDeadline = null;
      lastBetFx = null;
    }

    // Server-authoritative timebank: read from GameUpdate
    if (u.timeBankSeconds > 0) {
      timeBankSeconds = u.timeBankSeconds;
    }
    _turnDeadline = (u.turnDeadlineUnixMs > 0)
        ? DateTime.fromMillisecondsSinceEpoch(u.turnDeadlineUnixMs.toInt())
        : null;

    print('DEBUG: Updated state to: $_state, isMyTurn: $isMyTurn');

    errorMessage = '';
    _resetBackoff();
    notifyListeners();
  }

  // Removed local ticker; painter handles repainting via a RenderLoop using the deadline

  double get timebankRemainingSeconds {
    final dl = _turnDeadline;
    if (dl == null) return 0;
    final rem = dl.difference(DateTime.now());
    if (rem.isNegative) return 0;
    return rem.inMilliseconds / 1000.0;
  }

  void _onGameStreamError(Object e, StackTrace st) {
    errorMessage = 'Game stream error: $e';
    notifyListeners();
    _scheduleBackoff(_attachGameStream);
  }

  void _onGameStreamDone() {
    _scheduleBackoff(_attachGameStream);
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
      // Convert JSON map back to protobuf GameUpdate
      final gameStateJson = respMap['game_state'] as Map<String, dynamic>;
      final gameStateJsonStr = jsonEncode(gameStateJson);
      final gameUpdate = pr.GameUpdate.fromJson(gameStateJsonStr);
      print(
          'DEBUG: refreshGameState - phase: ${gameUpdate.phase}, gameStarted: ${gameUpdate.gameStarted}, currentPlayer: ${gameUpdate.currentPlayer}');
      game = UiGameState.fromUpdate(gameUpdate);

      final myP = me;
      final handCnt = myP?.hand.length ?? 0;
      final playersCnt = game?.players.length ?? 0;
      print(
          'DEBUG: refreshGameState snapshot - players=$playersCnt myHandCnt=$handCnt myId=$playerId curr=${gameUpdate.currentPlayer}');
      if (handCnt > 0) {
        final h = myP!.hand;
        print(
            'DEBUG: My cards (from GetGameState): ${h.map((c) => '${c.value} of ${c.suit}').join(', ')}');
      }
      // Keep coarse UI state in sync even when attaching mid-hand.
      // This mirrors the logic in _onGameUpdate so that the UI shows
      // the table (and hole cards) immediately on reconnect/restore.
      final phase = gameUpdate.phase;
      if (phase == pr.GamePhase.SHOWDOWN) {
        _state = PokerState.showdown;
        unawaited(_refreshLastWinners());
      } else if (phase != pr.GamePhase.WAITING) {
        _state = PokerState.handInProgress;
      } else {
        _state = PokerState.inLobby;
      }

      print(
          'DEBUG: refreshGameState - Updated state to: $_state, isMyTurn: $isMyTurn');

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
      final winners = winnersJson.map((w) {
        final winnerJsonStr = jsonEncode(w);
        return pr.Winner.fromJson(winnerJsonStr);
      }).toList();
      lastWinners = List.unmodifiable(winners.map(UiWinner.fromProto));
      notifyListeners();
    } catch (_) {
      // ignore; cache stays as-is
    }
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
