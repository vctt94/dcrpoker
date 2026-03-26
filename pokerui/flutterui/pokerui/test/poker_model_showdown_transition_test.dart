import 'package:fixnum/fixnum.dart';
import 'dart:async';
import 'dart:typed_data';
import 'package:audioplayers_platform_interface/audioplayers_platform_interface.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/models/poker.dart';

class _FakeAudioplayersPlatform extends AudioplayersPlatformInterface {
  final Map<String, StreamController<AudioEvent>> _eventStreams = {};

  @override
  Future<void> create(String playerId) async {
    _eventStreams[playerId] = StreamController<AudioEvent>.broadcast();
  }

  @override
  Future<void> dispose(String playerId) async {
    await _eventStreams.remove(playerId)?.close();
  }

  @override
  Future<void> emitError(String playerId, String code, String message) async {}

  @override
  Future<void> emitLog(String playerId, String message) async {}

  @override
  Future<int?> getCurrentPosition(String playerId) async => 0;

  @override
  Future<int?> getDuration(String playerId) async => 0;

  @override
  Stream<AudioEvent> getEventStream(String playerId) =>
      (_eventStreams[playerId] ?? StreamController<AudioEvent>.broadcast())
          .stream;

  @override
  Future<void> pause(String playerId) async {}

  @override
  Future<void> release(String playerId) async {}

  @override
  Future<void> resume(String playerId) async {}

  @override
  Future<void> seek(String playerId, Duration position) async {}

  @override
  Future<void> setAudioContext(
    String playerId,
    AudioContext audioContext,
  ) async {}

  @override
  Future<void> setBalance(String playerId, double balance) async {}

  @override
  Future<void> setPlaybackRate(String playerId, double playbackRate) async {}

  @override
  Future<void> setPlayerMode(String playerId, PlayerMode playerMode) async {}

  @override
  Future<void> setReleaseMode(String playerId, ReleaseMode releaseMode) async {}

  @override
  Future<void> setSourceBytes(
    String playerId,
    Uint8List bytes, {
    String? mimeType,
  }) async {}

  @override
  Future<void> setSourceUrl(
    String playerId,
    String url, {
    bool? isLocal,
    String? mimeType,
  }) async {}

  @override
  Future<void> setVolume(String playerId, double volume) async {}

  @override
  Future<void> stop(String playerId) async {}
}

class _FakeGlobalAudioplayersPlatform
    extends GlobalAudioplayersPlatformInterface {
  final StreamController<GlobalAudioEvent> _events =
      StreamController<GlobalAudioEvent>.broadcast();

  @override
  Future<void> emitGlobalError(String code, String message) async {}

  @override
  Future<void> emitGlobalLog(String message) async {}

  @override
  Stream<GlobalAudioEvent> getGlobalEventStream() => _events.stream;

  @override
  Future<void> init() async {}

  @override
  Future<void> setGlobalAudioContext(AudioContext ctx) async {}
}

class _TestPokerModel extends PokerModel {
  _TestPokerModel({required super.playerId}) : super(dataDir: '/tmp/test');

  @override
  Future<void> ensureGameStream() async {}
}

pr.Player _player({
  required String id,
  required String name,
  int balance = 1000,
  bool folded = false,
  bool isAllIn = false,
  bool isDealer = false,
  bool isSmallBlind = false,
  bool isBigBlind = false,
  int tableSeat = 0,
  List<pr.Card> hand = const [],
  bool cardsRevealed = false,
}) {
  return pr.Player(
    id: id,
    name: name,
    balance: Int64(balance),
    hand: hand,
    currentBet: Int64(0),
    folded: folded,
    isTurn: false,
    isAllIn: isAllIn,
    isDealer: isDealer,
    isReady: true,
    handDescription: '',
    playerState: folded
        ? pr.PlayerState.PLAYER_STATE_FOLDED
        : (isAllIn
            ? pr.PlayerState.PLAYER_STATE_ALL_IN
            : pr.PlayerState.PLAYER_STATE_IN_GAME),
    isSmallBlind: isSmallBlind,
    isBigBlind: isBigBlind,
    isDisconnected: false,
    tableSeat: tableSeat,
    presignComplete: false,
    cardsRevealed: cardsRevealed,
  );
}

pr.ShowdownPlayer _showdownPlayer({
  required String id,
  required String name,
  required pr.PlayerState finalState,
  int contribution = 0,
  List<pr.Card> holeCards = const [],
}) {
  return pr.ShowdownPlayer(
    playerId: id,
    finalState: finalState,
    handRank: pr.HandRank.HIGH_CARD,
    contribution: Int64(contribution),
    name: name,
    holeCards: holeCards,
  );
}

pr.GameUpdate _gameUpdate({
  required String tableId,
  required pr.GamePhase phase,
  required List<pr.Player> players,
  required String currentPlayer,
  List<pr.Card> communityCards = const [],
  int pot = 0,
}) {
  return pr.GameUpdate(
    tableId: tableId,
    phase: phase,
    players: players,
    communityCards: communityCards,
    pot: Int64(pot),
    currentBet: Int64(20),
    currentPlayer: currentPlayer,
    minRaise: Int64(20),
    maxRaise: Int64(1000),
    gameStarted: true,
    playersRequired: 2,
    playersJoined: players.length,
    phaseName: phase.label,
    timeBankSeconds: 30,
    turnDeadlineUnixMs: Int64(0),
    smallBlind: Int64(10),
    bigBlind: Int64(20),
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  AudioplayersPlatformInterface.instance = _FakeAudioplayersPlatform();
  GlobalAudioplayersPlatformInterface.instance =
      _FakeGlobalAudioplayersPlatform();

  test(
      'next-hand pre-flop update replaces showdown roster after player elimination',
      () {
    const tableId = 'table-1';
    const heroId = 'hero';
    const bustedId = 'busted';
    const winnerId = 'winner';

    final model = _TestPokerModel(playerId: heroId);
    model.currentTableId = tableId;
    model.game = UiGameState.fromUpdate(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.SHOWDOWN,
      players: [
        _player(
          id: heroId,
          name: 'Hero',
          folded: true,
          tableSeat: 0,
        ),
        _player(
          id: bustedId,
          name: 'Busted',
          balance: 0,
          isAllIn: true,
          isSmallBlind: true,
          tableSeat: 1,
        ),
        _player(
          id: winnerId,
          name: 'Winner',
          balance: 2000,
          isAllIn: true,
          isBigBlind: true,
          tableSeat: 2,
        ),
      ],
      currentPlayer: '',
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: [
          pr.Winner(
            playerId: winnerId,
            handRank: pr.HandRank.PAIR,
            bestHand: const [],
            winnings: Int64(2000),
          ),
        ],
        pot: Int64(2000),
        board: const [],
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_FOLDED,
          ),
          _showdownPlayer(
            id: bustedId,
            name: 'Busted',
            finalState: pr.PlayerState.PLAYER_STATE_ALL_IN,
            contribution: 1000,
          ),
          _showdownPlayer(
            id: winnerId,
            name: 'Winner',
            finalState: pr.PlayerState.PLAYER_STATE_ALL_IN,
            contribution: 1000,
          ),
        ],
        handId: 'hand-1',
        round: 1,
      ),
    ));

    expect(model.state, PokerState.showdown);
    expect(model.game!.players, hasLength(3));

    model.applyGameUpdateForTest(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.PRE_FLOP,
      players: [
        _player(
          id: heroId,
          name: 'Hero',
          balance: 1000,
          isBigBlind: true,
          tableSeat: 0,
        ),
        _player(
          id: winnerId,
          name: 'Winner',
          balance: 990,
          isDealer: true,
          isSmallBlind: true,
          tableSeat: 1,
        ),
      ],
      currentPlayer: winnerId,
    ));

    expect(model.state, PokerState.handInProgress);
    expect(model.game!.players, hasLength(2));
    expect(model.game!.players.any((p) => p.id == bustedId), isFalse);
    expect(model.me!.folded, isFalse);
    expect(model.autoAdvanceAllIn, isFalse);
  });

  test('game end remains on showdown until continue is clicked', () {
    const tableId = 'table-1';
    const heroId = 'hero';
    const villainId = 'villain';

    final model = _TestPokerModel(playerId: heroId);
    model.currentTableId = tableId;
    model.game = UiGameState.fromUpdate(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.SHOWDOWN,
      players: [
        _player(id: heroId, name: 'Hero', tableSeat: 0),
        _player(id: villainId, name: 'Villain', tableSeat: 1),
      ],
      currentPlayer: '',
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: [
          pr.Winner(
            playerId: villainId,
            handRank: pr.HandRank.PAIR,
            bestHand: const [],
            winnings: Int64(30),
          ),
        ],
        pot: Int64(30),
        board: const [],
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
          _showdownPlayer(
            id: villainId,
            name: 'Villain',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
        ],
        handId: 'hand-2',
        round: 2,
      ),
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.PLAYER_LOST,
      tableId: tableId,
      playerId: heroId,
      message: 'You lost all your chips and have been removed from the table.',
    ));

    expect(model.state, PokerState.showdown);
    expect(model.isGameEndPending, isTrue);
    expect(model.gameEndingMessage, isEmpty);

    model.skipShowdown();

    expect(model.state, PokerState.gameEnded);
    expect(model.isGameEndPending, isFalse);
    expect(
      model.gameEndingMessage,
      'You lost all your chips and have been removed from the table.',
    );
  });

  test(
      'showdown hands remain visible when later showdown snapshots redact hand',
      () {
    const tableId = 'table-1';
    const heroId = 'hero';
    const villainId = 'villain';
    final villainHand = [
      pr.Card(value: 'A', suit: 'Spades'),
      pr.Card(value: 'K', suit: 'Spades'),
    ];

    final model = _TestPokerModel(playerId: heroId);
    model.currentTableId = tableId;
    model.game = UiGameState.fromUpdate(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.SHOWDOWN,
      players: [
        _player(id: heroId, name: 'Hero', tableSeat: 0),
        _player(id: villainId, name: 'Villain', tableSeat: 1),
      ],
      currentPlayer: '',
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: const [],
        pot: Int64(30),
        board: const [],
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
          _showdownPlayer(
            id: villainId,
            name: 'Villain',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
            holeCards: villainHand,
          ),
        ],
        handId: 'hand-3',
        round: 3,
      ),
    ));

    model.applyGameUpdateForTest(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.SHOWDOWN,
      players: [
        _player(id: heroId, name: 'Hero', tableSeat: 0),
        _player(
          id: villainId,
          name: 'Villain',
          tableSeat: 1,
          cardsRevealed: true,
        ),
      ],
      currentPlayer: '',
    ));

    final villain = model.game!.players.firstWhere((p) => p.id == villainId);
    expect(villain.cardsRevealed, isTrue);
    expect(villain.hand, hasLength(2));
    expect(villain.hand.first.value, equals('A'));
    expect(villain.hand.last.value, equals('K'));
  });

  test(
      'showdown board does not fall back to live game update when payload board is empty',
      () {
    const tableId = 'table-1';
    const heroId = 'hero';
    const villainId = 'villain';
    final board = [
      pr.Card(value: 'A', suit: 'Spades'),
      pr.Card(value: 'K', suit: 'Hearts'),
      pr.Card(value: 'Q', suit: 'Clubs'),
      pr.Card(value: 'J', suit: 'Diamonds'),
      pr.Card(value: '10', suit: 'Spades'),
    ];

    final model = _TestPokerModel(playerId: heroId);
    model.currentTableId = tableId;

    model.applyGameUpdateForTest(_gameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.SHOWDOWN,
      players: [
        _player(id: heroId, name: 'Hero', tableSeat: 0),
        _player(id: villainId, name: 'Villain', tableSeat: 1),
      ],
      currentPlayer: '',
      communityCards: board,
      pot: 120,
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: [
          pr.Winner(
            playerId: villainId,
            handRank: pr.HandRank.STRAIGHT,
            bestHand: const [],
            winnings: Int64(120),
          ),
        ],
        pot: Int64(120),
        board: const [],
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
          _showdownPlayer(
            id: villainId,
            name: 'Villain',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
        ],
        handId: 'hand-4',
        round: 4,
      ),
    ));

    expect(model.showdownCommunityCards, isEmpty);
    expect(model.showdownPot, equals(120));
  });

  test('new pre-flop showdown clears the previous hand board cache', () {
    const tableId = 'table-1';
    const heroId = 'hero';
    const villainId = 'villain';
    final firstBoard = [
      pr.Card(value: 'A', suit: 'Spades'),
      pr.Card(value: 'K', suit: 'Hearts'),
      pr.Card(value: 'Q', suit: 'Clubs'),
      pr.Card(value: 'J', suit: 'Diamonds'),
      pr.Card(value: '10', suit: 'Spades'),
    ];

    final model = _TestPokerModel(playerId: heroId);
    model.currentTableId = tableId;

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: [
          pr.Winner(
            playerId: villainId,
            handRank: pr.HandRank.STRAIGHT,
            bestHand: const [],
            winnings: Int64(120),
          ),
        ],
        pot: Int64(120),
        board: firstBoard,
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
          _showdownPlayer(
            id: villainId,
            name: 'Villain',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
        ],
        handId: 'hand-4',
        round: 4,
      ),
    ));

    expect(model.showdownCommunityCards, hasLength(5));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.NEW_HAND_STARTED,
      tableId: tableId,
    ));

    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.SHOWDOWN_RESULT,
      tableId: tableId,
      showdown: pr.Showdown(
        winners: [
          pr.Winner(
            playerId: heroId,
            handRank: pr.HandRank.HIGH_CARD,
            bestHand: const [],
            winnings: Int64(30),
          ),
        ],
        pot: Int64(30),
        board: const [],
        players: [
          _showdownPlayer(
            id: heroId,
            name: 'Hero',
            finalState: pr.PlayerState.PLAYER_STATE_IN_GAME,
          ),
          _showdownPlayer(
            id: villainId,
            name: 'Villain',
            finalState: pr.PlayerState.PLAYER_STATE_FOLDED,
          ),
        ],
        handId: 'hand-5',
        round: 5,
      ),
    ));

    expect(model.showdownCommunityCards, isEmpty);
  });
}
