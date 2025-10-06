// lib/models/poker_model.dart
import 'dart:async';
import 'dart:io';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:collection/collection.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:golib_plugin/grpc/generated/poker.pbgrpc.dart' as prpc;
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:grpc/grpc.dart';
import 'package:fixnum/fixnum.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/notifications.dart';

/// -------- UI-facing enums --------
enum PokerState {
  idle,
  browsingTables,
  inLobby,            // seated, waiting / readying
  handInProgress,     // active betting streets
  showdown,           // results surfaced
  tournamentOver,     // SNG complete
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
  final int balance;       // chips (in-game)
  final List<pr.Card> hand;
  final int currentBet;    // chips
  final bool folded;
  final bool isTurn;
  final bool isAllIn;
  final bool isDealer;
  final bool isReady;
  final String handDesc;   // only meaningful at showdown

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
    required this.isReady,
    required this.handDesc,
  });

  factory UiPlayer.fromProto(pr.Player p) => UiPlayer(
        id: p.id,
        name: p.name,
        balance: p.balance.toInt(),
        hand: List.unmodifiable(p.hand),
        currentBet: p.currentBet.toInt(),
        folded: p.folded,
        isTurn: p.isTurn,
        isAllIn: p.isAllIn,
        isDealer: p.isDealer,
        isReady: p.isReady,
        handDesc: p.handDescription,
      );
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
  final int pot;               // chips
  final int currentBet;        // chips
  final String currentPlayerId;
  final int minRaise;          // chips
  final int maxRaise;          // chips
  final bool gameStarted;
  final int playersRequired;
  final int playersJoined;

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
      );
}

/// -------- The main ChangeNotifier --------
class PokerModel extends ChangeNotifier {
  // Injected RPC clients
  final prpc.LobbyServiceClient lobby;
  final prpc.PokerServiceClient poker;

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

  // Streams
  StreamSubscription<pr.Notification>? _ntfnSub;
  StreamSubscription<pr.GameUpdate>? _gameSub;

  // Backoff
  int _retries = 0;
  Timer? _backoffTimer;

  // Cached readiness
  bool _iAmReady = false;
  bool _seated = false; // track whether user is seated at any table

  PokerModel({
    required this.lobby,
    required this.poker,
    required this.playerId,
  });

  /// Factory method to create PokerModel from Config
  static Future<PokerModel> fromConfig(Config cfg, NotificationModel notificationModel) async {
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
    print('DEBUG: fromConfig - Golib.initClient returned id=${localInfo.id} nick=${localInfo.nick}');
    
    // Create gRPC channel
    final channel = await _createGrpcChannel(cfg);
    
    // Create gRPC clients
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

  /// Create gRPC channel from config
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
        authority: host, // Use the host as the authority for certificate validation
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
      final stream = lobby.startNotificationStream(
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
    print('DEBUG: Notification received type=${n.type} tableId=${n.tableId} playerId=${n.playerId}');
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
      case pr.NotificationType.GAME_STARTED:
      case pr.NotificationType.GAME_ENDED:
      case pr.NotificationType.BET_MADE:
      case pr.NotificationType.CALL_MADE:
      case pr.NotificationType.CHECK_MADE:
      case pr.NotificationType.PLAYER_FOLDED:
      case pr.NotificationType.SMALL_BLIND_POSTED:
      case pr.NotificationType.BIG_BLIND_POSTED:
      case pr.NotificationType.SHOWDOWN_RESULT:
        // Game stream will drive UI; still useful for toasts.
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
      // Ensure we didn't miss the GAME_STARTED snapshot by fetching current state.
      await refreshGameState();
      await _refreshLastWinners(); // useful if a hand just ended
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
      _state = PokerState.browsingTables;
      notifyListeners();
      unawaited(refreshTables());
    }
  }

  Future<void> _restoreCurrentTable() async {
    try {
      // Use golib to discover any existing table and keep Go client in sync
      final tid = await Golib.getPokerCurrentTable();
      print('DEBUG: _restoreCurrentTable - tid=$tid');
      if (tid.isEmpty) return;
      // Re-join via golib to reconcile client-side state and attach streams
      await joinTable(tid);
      // Proactively refresh the state in case we attached mid-hand.
      await refreshGameState();
    } catch (_) {
      // ignore
    }
  }

  // -------- Ready / Unready & show/hide cards ----------
  Future<void> setReady() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      final resp = await lobby.setPlayerReady(pr.SetPlayerReadyRequest()
        ..playerId = playerId
        ..tableId = tid);
      _iAmReady = resp.success;
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
      final resp = await lobby.setPlayerUnready(pr.SetPlayerUnreadyRequest()
        ..playerId = playerId
        ..tableId = tid);
      if (resp.success) _iAmReady = false;
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
      await poker.showCards(pr.ShowCardsRequest()
        ..playerId = playerId
        ..tableId = tid);
    } catch (e) {
      errorMessage = 'Show cards failed: $e';
      notifyListeners();
    }
  }

  Future<void> hideCards() async {
    final tid = currentTableId;
    if (tid == null) return;
    try {
      await poker.hideCards(pr.HideCardsRequest()
        ..playerId = playerId
        ..tableId = tid);
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
      final stream = poker.startGameStream(pr.StartGameStreamRequest()
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

    print('DEBUG: Processing game update - phase: ${u.phase}, gameStarted: ${u.gameStarted}, currentPlayer: ${u.currentPlayer}');
    
    game = UiGameState.fromUpdate(u);

    final myP = me;
    final handCnt = myP?.hand.length ?? 0;
    final playersCnt = game?.players.length ?? u.players.length;
    print('DEBUG: GameUpdate snapshot - players=$playersCnt myHandCnt=$handCnt myId=$playerId curr=${u.currentPlayer}');
    if (handCnt > 0) {
      final h = myP!.hand;
      print('DEBUG: My cards: ${h.map((c) => '${c.value} of ${c.suit}').join(', ')}');
    }

    // Drive coarse UI state from server phase:
    // - SHOWDOWN -> showdown view
    // - Any non-WAITING phase -> hand in progress (even NEW_HAND_DEALING)
    // This avoids relying solely on gameStarted, which can lag in some snapshots.
    if (u.phase == pr.GamePhase.SHOWDOWN) {
      _state = PokerState.showdown;
      unawaited(_refreshLastWinners());
    } else if (u.phase != pr.GamePhase.WAITING) {
      _state = PokerState.handInProgress;
    } else {
      _state = PokerState.inLobby;
    }

    print('DEBUG: Updated state to: $_state, isMyTurn: $isMyTurn');
    
    errorMessage = '';
    _resetBackoff();
    notifyListeners();
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
      final r = await poker.makeBet(pr.MakeBetRequest()
        ..playerId = playerId
        ..tableId = tid
        ..amount = Int64(amountChips));
      if (!r.success) {
        errorMessage = r.message;
        notifyListeners();
        return false;
      }
      // r.new_balance is atoms (wallet), not table chips
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
      final r = await poker.callBet(pr.CallBetRequest()
        ..playerId = playerId
        ..tableId = tid);
      if (!r.success) {
        errorMessage = r.message;
        notifyListeners();
      }
      return r.success;
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
      final r = await poker.checkBet(pr.CheckBetRequest()
        ..playerId = playerId
        ..tableId = tid);
      if (!r.success) {
        errorMessage = r.message;
        notifyListeners();
      }
      return r.success;
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
      final r = await poker.foldBet(pr.FoldBetRequest()
        ..playerId = playerId
        ..tableId = tid);
      if (!r.success) {
        errorMessage = r.message;
        notifyListeners();
      }
      return r.success;
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
      final resp = await poker.getGameState(pr.GetGameStateRequest()..tableId = tid);
      print('DEBUG: refreshGameState - phase: ${resp.gameState.phase}, gameStarted: ${resp.gameState.gameStarted}, currentPlayer: ${resp.gameState.currentPlayer}');
      
      game = UiGameState.fromUpdate(resp.gameState);

      final myP = me;
      final handCnt = myP?.hand.length ?? 0;
      final playersCnt = game?.players.length ?? 0;
      print('DEBUG: refreshGameState snapshot - players=$playersCnt myHandCnt=$handCnt myId=$playerId curr=${resp.gameState.currentPlayer}');
      if (handCnt > 0) {
        final h = myP!.hand;
        print('DEBUG: My cards (from GetGameState): ${h.map((c) => '${c.value} of ${c.suit}').join(', ')}');
      }
            
      print('DEBUG: refreshGameState - Updated state to: $_state, isMyTurn: $isMyTurn');
      
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
      final resp = await poker.getLastWinners(pr.GetLastWinnersRequest()..tableId = tid);
      lastWinners = List.unmodifiable(resp.winners.map(UiWinner.fromProto));
      notifyListeners();
    } catch (_) {
      // ignore; cache stays as-is
    }
  }

  Future<pr.EvaluateHandResponse?> evaluateCards(List<pr.Card> cards) async {
    try {
      final resp = await poker.evaluateHand(pr.EvaluateHandRequest()..cards.addAll(cards));
      return resp;
    } catch (e) {
      errorMessage = 'EvaluateHand failed: $e';
      notifyListeners();
      return null;
    }
  }

  // -------- Helpers ----------
  UiPlayer? get me =>
      game?.players.firstWhereOrNull((p) => p.id == playerId);

  bool get iAmReady => _iAmReady;

  bool get isMyTurn =>
      game != null && game!.currentPlayerId == playerId;

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
