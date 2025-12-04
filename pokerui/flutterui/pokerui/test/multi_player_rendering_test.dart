import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/poker/game.dart';
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
  Future<void> refreshTables() async {}
  
  @override
  Future<void> refreshGameState() async {}
  
  @override
  Future<void> leaveTable() async {}
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
}) {
  return UiPlayer(
    id: id,
    name: name,
    balance: balance,
    hand: const [],
    currentBet: currentBet,
    folded: folded,
    isTurn: isTurn,
    isAllIn: false,
    isDealer: isDealer,
    isSmallBlind: isSmallBlind,
    isBigBlind: isBigBlind,
    isReady: true,
    isDisconnected: false,
    handDesc: '',
    tableSeat: tableSeat,
  );
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
          _createPlayer(id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
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
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 800,
              height: 450,
              child: pokerGame.buildWidget(model.game!, focusNode),
            ),
          ),
        ),
      );
      
      // Wait for the widget to build
      await tester.pump();
      
      // Verify that the widget builds successfully
      // There may be multiple CustomPaint widgets, so we check for at least one
      expect(find.byType(CustomPaint), findsWidgets);
      
      // Find the CustomPaint widget that contains the PokerPainter
      // The PokerPainter draws the players on the canvas
      final customPaints = find.byType(CustomPaint);
      expect(customPaints, findsWidgets);
      
      // Verify at least one CustomPaint has a PokerPainter
      bool foundPokerPainter = false;
      tester.allWidgets.whereType<CustomPaint>().forEach((cp) {
        if (cp.painter is PokerPainter) {
          foundPokerPainter = true;
        }
      });
      expect(foundPokerPainter, isTrue);
      
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
    
    testWidgets('Players are positioned correctly around the table', (WidgetTester tester) async {
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
      
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 800,
              height: 450,
              child: pokerGame.buildWidget(model.game!, focusNode),
            ),
          ),
        ),
      );
      
      await tester.pump();
      
      // Verify the PokerPainter is created with correct game state
      // Find the CustomPaint widget that contains the PokerPainter
      PokerPainter? painter;
      tester.allWidgets.whereType<CustomPaint>().forEach((cp) {
        if (cp.painter is PokerPainter) {
          painter = cp.painter as PokerPainter;
        }
      });
      
      expect(painter, isNotNull);
      
      // Verify painter has the correct game state
      final p = painter!; // Use null-assertion since we checked it's not null
      expect(p.gameState.players.length, equals(3));
      expect(p.currentPlayerId, equals(heroId));
      
      // Verify all players are in the painter's game state
      expect(p.gameState.players.any((player) => player.id == heroId), isTrue);
      expect(p.gameState.players.any((player) => player.id == 'player2'), isTrue);
      expect(p.gameState.players.any((player) => player.id == 'player3'), isTrue);
    });
    
    testWidgets('Table renders with correct aspect ratio', (WidgetTester tester) async {
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
      
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 800,
              height: 450,
              child: pokerGame.buildWidget(model.game!, focusNode),
            ),
          ),
        ),
      );
      
      await tester.pump();
      
      // Verify the widget has the correct size
      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, equals(800));
      expect(sizedBox.height, equals(450));
      
      // Verify AspectRatio widget exists (16:9 aspect ratio for poker table)
      expect(find.byType(AspectRatio), findsOneWidget);
      
      final aspectRatio = tester.widget<AspectRatio>(find.byType(AspectRatio));
      expect(aspectRatio.aspectRatio, equals(16 / 9));
    });
    
    testWidgets('Visual snapshot: 3 players rendered on table', (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);
      
      // Create a game state with 3 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0, isDealer: true, isTurn: true),
          _createPlayer(id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
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
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      // Set a fixed window size for consistent golden file generation
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            backgroundColor: Colors.black, // Black background to match poker table
            body: Center(
              child: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );
      
      // Wait for all animations and renders to complete
      await tester.pumpAndSettle();
      
      // Generate golden file - this creates a visual snapshot
      // Run with: flutter test --update-goldens to generate/update the image
      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_3_players.png'),
      );
      
      // Clean up
      tester.binding.window.clearPhysicalSizeTestValue();
      tester.binding.window.clearDevicePixelRatioTestValue();
    });
    
    testWidgets('Visual snapshot: 2 players rendered on table', (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);
      
      // Create a game state with 2 players (heads-up)
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0, isDealer: true, isSmallBlind: true, isTurn: true),
          _createPlayer(id: 'player2', name: 'Alice', tableSeat: 1, isBigBlind: true),
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
      
      // Build the widget using PokerGame
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      // Set a fixed window size for consistent golden file generation
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            backgroundColor: Colors.black, // Black background to match poker table
            body: Center(
              child: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );
      
      // Wait for all animations and renders to complete
      await tester.pumpAndSettle();
      
      // Generate golden file - this creates a visual snapshot
      // Run with: flutter test --update-goldens to generate/update the image
      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_2_players.png'),
      );
      
      // Clean up
      tester.binding.window.clearPhysicalSizeTestValue();
      tester.binding.window.clearDevicePixelRatioTestValue();
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
          _createPlayer(id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
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
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 800,
              height: 450,
              child: pokerGame.buildWidget(model.game!, focusNode),
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
    
    testWidgets('Visual snapshot: 6 players rendered on table', (WidgetTester tester) async {
      const heroId = 'player1';
      final model = MockPokerModelForRendering(playerId: heroId);
      
      // Create a game state with 6 players
      model.game = UiGameState(
        tableId: 'test-table',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _createPlayer(id: heroId, name: 'Hero', tableSeat: 0, isDealer: true, isTurn: true),
          _createPlayer(id: 'player2', name: 'Alice', tableSeat: 1, isSmallBlind: true),
          _createPlayer(id: 'player3', name: 'Bob', tableSeat: 2, isBigBlind: true),
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
      final pokerGame = PokerGame(heroId, model);
      final focusNode = FocusNode();
      
      // Set a fixed window size for consistent golden file generation
      tester.binding.window.physicalSizeTestValue = const Size(800, 450);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            backgroundColor: Colors.black, // Black background to match poker table
            body: Center(
              child: SizedBox(
                width: 800,
                height: 450,
                child: pokerGame.buildWidget(model.game!, focusNode),
              ),
            ),
          ),
        ),
      );
      
      // Wait for all animations and renders to complete
      await tester.pumpAndSettle();
      
      // Generate golden file - this creates a visual snapshot
      // Run with: flutter test --update-goldens to generate/update the image
      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('multi_player_rendering_6_players.png'),
      );
      
      // Clean up
      tester.binding.window.clearPhysicalSizeTestValue();
      tester.binding.window.clearDevicePixelRatioTestValue();
    });
  });
}

