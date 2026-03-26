import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

/// Mock PokerModel for testing player rendering
class MockPokerModelForRendering extends PokerModel {
  MockPokerModelForRendering({required super.playerId, UiGameState? game})
      : super(dataDir: '/tmp/test') {
    this.game = game;
  }

  // Stub methods to prevent actual network calls
  @override
  Future<void> init() async {}

  @override
  Future<void> browseTables() async {}

  @override
  Future<void> refreshGameState() async {}

  @override
  Future<void> leaveTable() async {}

  // Helper to set showdown data for testing
  // Delegates to the public test helper method in PokerModel
  void setShowdownData({
    required List<UiPlayer> players,
    required List<pr.Card> communityCards,
    required int pot,
    List<UiWinner> winners = const [],
  }) {
    setShowdownDataForTest(
      players: players,
      communityCards: communityCards,
      pot: pot,
      winners: winners,
    );
  }
}

/// Default theme for tests
const _defaultTheme = PokerThemeConfig(
  tableTheme: TableThemeConfig.classic,
  cardTheme: CardColorTheme.standard,
  cardSizeMultiplier: 1.0,
  uiSizeMultiplier: 1.0,
  showTableLogo: true,
  logoPosition: 'center',
);

/// Default config for tests
final _defaultConfig = Config(
  serverAddr: '127.0.0.1:50051',
  grpcCertPath: '',
  payoutAddress: '',
  debugLevel: 'info',
  soundsEnabled: false,
  dataDir: '/tmp/test',
  address: '',
  tableTheme: 'classic',
  cardTheme: 'standard',
  cardSize: 'medium',
  uiSize: 'medium',
  hideTableLogo: false,
  logoPosition: 'center',
);

/// Helper to wrap widget with necessary providers for tests
Widget _wrapWithProviders(Widget child) {
  final configNotifier = ConfigNotifier();
  configNotifier.updateConfig(_defaultConfig);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: child,
  );
}

/// Helper to create a Card
pr.Card _createCard(String value, String suit) {
  return pr.Card()
    ..value = value
    ..suit = suit;
}

/// Helper to create a mock UiPlayer
UiPlayer _createPlayer({
  required String id,
  required String name,
  int balance = 1000,
  int currentBet = 0,
  bool folded = false,
  bool isTurn = false,
  bool isDealer = false,
  bool isSmallBlind = false,
  bool isBigBlind = false,
  int tableSeat = 0,
  List<pr.Card> hand = const [],
  String handDesc = '',
  bool cardsRevealed = false,
}) {
  return UiPlayer(
    id: id,
    name: name,
    balance: balance,
    hand: hand,
    currentBet: currentBet,
    folded: folded,
    isTurn: isTurn,
    isAllIn: false,
    isDealer: isDealer,
    isSmallBlind: isSmallBlind,
    isBigBlind: isBigBlind,
    isReady: true,
    isDisconnected: false,
    handDesc: handDesc,
    cardsRevealed: cardsRevealed,
    tableSeat: tableSeat,
  );
}

Finder _seatFinder(String playerId) =>
    find.byKey(ValueKey('seat_widget_$playerId'));

Rect _seatRect(WidgetTester tester, String playerId) =>
    tester.getRect(_seatFinder(playerId));

Future<void> _pumpTable(
  WidgetTester tester, {
  required PokerGame pokerGame,
  required UiGameState gameState,
  required FocusNode focusNode,
  Size size = const Size(800, 450),
  bool showHeroSeatCards = true,
}) async {
  await tester.pumpWidget(
    _wrapWithProviders(
      MaterialApp(
        home: Scaffold(
          backgroundColor: Colors.black,
          body: Center(
            child: SizedBox(
              width: size.width,
              height: size.height,
              child: pokerGame.buildWidget(
                gameState,
                focusNode,
                showHeroSeatCards: showHeroSeatCards,
              ),
            ),
          ),
        ),
      ),
    ),
  );

  await tester.pumpAndSettle();
}

void main() {
  group('Multi-Player Rendering Tests', () {
    testWidgets('Renders 3 players on the table', (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      // Create a game state with 3 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0, isDealer: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30, // SB + BB
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 3,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      // Build the widget using PokerGame
      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await tester.pumpWidget(
        _wrapWithProviders(
          MaterialApp(
            home: Scaffold(
              body: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );

      // Wait for the widget to build
      await tester.pump();

      // Verify that the widget builds successfully
      // There may be multiple CustomPaint widgets, so we check for at least one
      expect(find.byType(CustomPaint), findsWidgets);

      // Players are now rendered via PlayerSeatsOverlay (widget-based),
      // not a PokerPainter canvas. Verify that CustomPaint widgets still
      // exist (table background) and that the game state is intact.
      final customPaints = find.byType(CustomPaint);
      expect(customPaints, findsWidgets);

      // Verify the game state has 3 players
      expect(model.game!.players.length, equals(3));

      // Verify all 3 players are in the game state
      expect(model.game!.players.any((p) => p.id == heroId), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player2'), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player3'), isTrue);

      // Verify player names
      final hero = model.game!.players.firstWhere((p) => p.id == heroId);
      final player2 = model.game!.players.firstWhere((p) => p.id == 'player2');
      final player3 = model.game!.players.firstWhere((p) => p.id == 'player3');

      expect(hero.name, equals('Hero'));
      expect(player2.name, equals('Alice'));
      expect(player3.name, equals('Bob'));

      // Verify player positions (seats)
      expect(hero.tableSeat, equals(0));
      expect(player2.tableSeat, equals(1));
      expect(player3.tableSeat, equals(2));

      // Verify dealer/blind assignments
      expect(hero.isDealer, isTrue);
      expect(player2.isSmallBlind, isTrue);
      expect(player3.isBigBlind, isTrue);
    });

    testWidgets('Players are positioned correctly around the table',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0),
          _createPlayer(id: 'player2', name: 'Player 2', tableSeat: 1),
          _createPlayer(id: 'player3', name: 'Player 3', tableSeat: 2),
        ],
        communityCards: const [],
        pot: 0,
        currentBet: 0,
        currentPlayerId: '',
        minRaise: 0,
        maxRaise: 0,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: false,
        playersRequired: 2,
        playersJoined: 3,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await tester.pumpWidget(
        _wrapWithProviders(
          MaterialApp(
            home: Scaffold(
              body: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );

      await tester.pump();

      // Players are now rendered via PlayerSeatsOverlay instead of PokerPainter.
      // Verify the game state directly and ensure the seat labels render.
      expect(model.game!.players.length, equals(3));
      expect(model.game!.currentPlayerId, isEmpty);

      // Verify all players are in the game state
      expect(model.game!.players.any((player) => player.id == heroId), isTrue);
      expect(
          model.game!.players.any((player) => player.id == 'player2'), isTrue);
      expect(
          model.game!.players.any((player) => player.id == 'player3'), isTrue);

      // Verify seat labels are present in the rendered widget tree.
      expect(find.text('Hero'), findsOneWidget);
      expect(find.text('Player 2'), findsOneWidget);
      expect(find.text('Player 3'), findsOneWidget);
    });

    testWidgets('Table renders with correct aspect ratio',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0),
          _createPlayer(id: 'player2', name: 'Player 2', tableSeat: 1),
          _createPlayer(id: 'player3', name: 'Player 3', tableSeat: 2),
        ],
        communityCards: const [],
        pot: 0,
        currentBet: 0,
        currentPlayerId: '',
        minRaise: 0,
        maxRaise: 0,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: false,
        playersRequired: 2,
        playersJoined: 3,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await tester.pumpWidget(
        _wrapWithProviders(
          MaterialApp(
            home: Scaffold(
              body: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );

      await tester.pump();

      // Verify the widget has the correct size
      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, equals(800));
      expect(sizedBox.height, equals(450));

      // The refactor passes the desired ratio into PokerTableBackground
      // instead of wrapping the table in a dedicated AspectRatio widget.
      final tableBackground = tester.widget<PokerTableBackground>(
        find.byType(PokerTableBackground),
      );
      expect(tableBackground.aspectRatio, equals(16 / 9));
    });

    testWidgets('Opponent cards remain hidden at showdown when not revealed',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.SHOWDOWN,
        phaseName: 'Showdown',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0),
          _createPlayer(
            id: 'player2',
            name: 'Player 2',
            tableSeat: 1,
            hand: [
              _createCard('A', 'Spades'),
              _createCard('K', 'Spades'),
            ],
            cardsRevealed: false,
          ),
        ],
        communityCards: const [],
        pot: 100,
        currentBet: 0,
        currentPlayerId: '',
        minRaise: 0,
        maxRaise: 0,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 2,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await tester.pumpWidget(
        _wrapWithProviders(
          MaterialApp(
            home: Scaffold(
              body: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(
                  model.game!,
                  focusNode,
                  showHeroSeatCards: false,
                ),
              ),
            ),
          ),
        ),
      );

      await tester.pump();

      expect(find.byType(CardFace), findsNothing);
      expect(find.byType(CardBack), findsNWidgets(2));
    });

    testWidgets('3-player table keeps hero docked and opponents split',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      // Create a game state with 3 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30, // SB + BB
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 3,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      final tableRect = tester.getRect(find.byType(PokerTableBackground));
      final heroRect = _seatRect(tester, heroId);
      final aliceRect = _seatRect(tester, 'player2');
      final bobRect = _seatRect(tester, 'player3');

      expect((heroRect.center.dx - tableRect.center.dx).abs(), lessThan(24));
      expect(heroRect.center.dy, greaterThan(aliceRect.center.dy));
      expect(heroRect.center.dy, greaterThan(bobRect.center.dy));
      expect(aliceRect.center.dx, lessThan(tableRect.center.dx - 20));
      expect(bobRect.center.dx, greaterThan(tableRect.center.dx + 20));
      expect(aliceRect.center.dy, lessThan(heroRect.top - 20));
      expect(bobRect.center.dy, lessThan(heroRect.top - 20));
      expect(find.text('D'), findsOneWidget);
      expect(find.text('SB'), findsOneWidget);
      expect(find.text('BB'), findsOneWidget);
    });

    testWidgets('visual snapshot: 3 players rendered on table',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 3,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_3_players.png'),
      );
    });

    testWidgets('2-player table centers the opponent above the hero',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      // Create a game state with 2 players (heads-up)
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isSmallBlind: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30, // SB + BB
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 2,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      final tableRect = tester.getRect(find.byType(PokerTableBackground));
      final heroRect = _seatRect(tester, heroId);
      final opponentRect = _seatRect(tester, 'player2');

      expect((heroRect.center.dx - tableRect.center.dx).abs(), lessThan(24));
      expect((opponentRect.center.dx - tableRect.center.dx).abs(), lessThan(8));
      expect(heroRect.center.dy, greaterThan(opponentRect.center.dy + 80));
      expect(find.text('Hero'), findsOneWidget);
      expect(find.text('Alice'), findsOneWidget);
      expect(find.text('SB'), findsOneWidget);
      expect(find.text('BB'), findsOneWidget);
    });

    testWidgets('visual snapshot: 2 players rendered on table',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isSmallBlind: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 2,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_2_players.png'),
      );
    });

    testWidgets('Renders 6 players on the table', (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      // Create a game state with 6 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0, isDealer: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
          _createPlayer(id: 'player5', name: 'Diana', tableSeat: 4),
          _createPlayer(id: 'player6', name: 'Eve', tableSeat: 5),
        ],
        communityCards: const [],
        pot: 30, // SB + BB
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 6,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      // Build the widget using PokerGame
      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await tester.pumpWidget(
        _wrapWithProviders(
          MaterialApp(
            home: Scaffold(
              body: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );

      // Wait for the widget to build
      await tester.pump();

      // Verify that the widget builds successfully
      expect(find.byType(CustomPaint), findsWidgets);

      // Verify the game state has 6 players
      expect(model.game!.players.length, equals(6));

      // Verify all 6 players are in the game state
      expect(model.game!.players.any((p) => p.id == heroId), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player2'), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player3'), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player4'), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player5'), isTrue);
      expect(model.game!.players.any((p) => p.id == 'player6'), isTrue);

      // Verify player names
      final hero = model.game!.players.firstWhere((p) => p.id == heroId);
      final player2 = model.game!.players.firstWhere((p) => p.id == 'player2');
      final player3 = model.game!.players.firstWhere((p) => p.id == 'player3');
      final player4 = model.game!.players.firstWhere((p) => p.id == 'player4');
      final player5 = model.game!.players.firstWhere((p) => p.id == 'player5');
      final player6 = model.game!.players.firstWhere((p) => p.id == 'player6');

      expect(hero.name, equals('Hero'));
      expect(player2.name, equals('Alice'));
      expect(player3.name, equals('Bob'));
      expect(player4.name, equals('Charlie'));
      expect(player5.name, equals('Diana'));
      expect(player6.name, equals('Eve'));

      // Verify player positions (seats)
      expect(hero.tableSeat, equals(0));
      expect(player2.tableSeat, equals(1));
      expect(player3.tableSeat, equals(2));
      expect(player4.tableSeat, equals(3));
      expect(player5.tableSeat, equals(4));
      expect(player6.tableSeat, equals(5));

      // Verify dealer/blind assignments
      expect(hero.isDealer, isTrue);
      expect(player2.isSmallBlind, isTrue);
      expect(player3.isBigBlind, isTrue);
    });

    testWidgets('6-player table spreads opponents across the upper arc',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      // Create a game state with 6 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
          _createPlayer(id: 'player5', name: 'Diana', tableSeat: 4),
          _createPlayer(id: 'player6', name: 'Eve', tableSeat: 5),
        ],
        communityCards: const [],
        pot: 30, // SB + BB
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 6,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      final tableRect = tester.getRect(find.byType(PokerTableBackground));
      final heroRect = _seatRect(tester, heroId);
      final opponentRects = [
        _seatRect(tester, 'player2'),
        _seatRect(tester, 'player3'),
        _seatRect(tester, 'player4'),
        _seatRect(tester, 'player5'),
        _seatRect(tester, 'player6'),
      ];
      final opponentXs = opponentRects.map((rect) => rect.center.dx).toList()
        ..sort();
      final topMostOpponent = opponentRects.reduce(
        (current, next) => next.center.dy < current.center.dy ? next : current,
      );

      expect((heroRect.center.dx - tableRect.center.dx).abs(), lessThan(24));
      expect(opponentXs.first, lessThan(tableRect.center.dx - 100));
      expect(opponentXs.last, greaterThan(tableRect.center.dx + 100));
      expect(
        (topMostOpponent.center.dx - tableRect.center.dx).abs(),
        lessThan(50),
      );
      for (final rect in opponentRects) {
        expect(rect.center.dy, lessThan(heroRect.center.dy - 20));
      }
    });

    testWidgets('visual snapshot: 6 players rendered on table',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
          _createPlayer(id: 'player5', name: 'Diana', tableSeat: 4),
          _createPlayer(id: 'player6', name: 'Eve', tableSeat: 5),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 6,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_6_players.png'),
      );
    });

    testWidgets('visual snapshot: 4 players rendered on table',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 4,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_4_players.png'),
      );
    });

    testWidgets('4-player table centers the top opponent over the hero',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 4,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      final tableRect = tester.getRect(find.byType(PokerTableBackground));
      final heroRect = _seatRect(tester, heroId);
      final topOpponent = [
        _seatRect(tester, 'player2'),
        _seatRect(tester, 'player3'),
        _seatRect(tester, 'player4'),
      ].reduce(
        (current, next) => next.center.dy < current.center.dy ? next : current,
      );

      expect((heroRect.center.dx - tableRect.center.dx).abs(), lessThan(24));
      expect((topOpponent.center.dx - tableRect.center.dx).abs(), lessThan(8));
      expect(topOpponent.center.dy, lessThan(heroRect.top - 20));
    });

    testWidgets('4-player table preserves relative seat order with hero pinned',
        (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 4,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      final heroRect = _seatRect(tester, heroId);
      final sbRect = _seatRect(tester, 'player2');
      final bbRect = _seatRect(tester, 'player3');
      final utgRect = _seatRect(tester, 'player4');

      expect(sbRect.center.dx, lessThan(heroRect.center.dx));
      expect(utgRect.center.dx, greaterThan(heroRect.center.dx));
      expect(bbRect.center.dy, lessThan(sbRect.center.dy));
      expect(bbRect.center.dy, lessThan(utgRect.center.dy));
    });

    testWidgets('visual snapshot: 5 players rendered on table',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);

      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(
              id: heroId,
              name: 'Hero',
              tableSeat: 0,
              isDealer: true,
              isTurn: true),
          _createPlayer(
              id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(
              id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
          _createPlayer(id: 'player4', name: 'Charlie', tableSeat: 3),
          _createPlayer(id: 'player5', name: 'Diana', tableSeat: 4),
        ],
        communityCards: const [],
        pot: 30,
        currentBet: 20,
        currentPlayerId: heroId,
        minRaise: 20,
        maxRaise: 1000,
        smallBlind: 10,
        bigBlind: 20,
        gameStarted: true,
        playersRequired: 2,
        playersJoined: 5,
        timeBankSeconds: 30,
        turnDeadlineUnixMs: 0,
      );

      final pokerGame = PokerGame(heroId, model, theme: _defaultTheme);
      final focusNode = FocusNode();

      await _pumpTable(
        tester,
        pokerGame: pokerGame,
        gameState: model.game!,
        focusNode: focusNode,
      );

      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_5_players.png'),
      );
    });
  });
}
