import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/player_seat.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

UiPlayer _player({
  required String id,
  required String name,
  bool isTurn = false,
  bool isAllIn = false,
  int currentBet = 0,
  List<pr.Card> hand = const [],
}) {
  return UiPlayer(
    id: id,
    name: name,
    balance: 1000,
    hand: hand,
    currentBet: currentBet,
    folded: false,
    isTurn: isTurn,
    isAllIn: isAllIn,
    isDealer: false,
    isSmallBlind: false,
    isBigBlind: false,
    isReady: true,
    isDisconnected: false,
    handDesc: '',
  );
}

void main() {
  testWidgets('player seats do not overflow with xl UI and card scales',
      (WidgetTester tester) async {
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero', isTurn: true),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 0,
      currentBet: 20,
      currentPlayerId: 'hero',
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

    const theme = PokerThemeConfig(
      tableTheme: TableThemeConfig.classic,
      cardTheme: CardColorTheme.standard,
      cardSizeMultiplier: 1.4,
      uiSizeMultiplier: 1.3,
      showTableLogo: true,
      logoPosition: 'center',
    );

    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SizedBox(),
        ),
      ),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 390,
            height: 260,
            child: PlayerSeatsOverlay(
              gameState: gameState,
              heroId: 'hero',
              theme: theme,
              aspectRatio: 1.15,
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(PlayerSeatsOverlay), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('all-in opponent cards get larger at xl card size',
      (WidgetTester tester) async {
    UiGameState buildState() {
      return UiGameState(
        tableId: 'table-1',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _player(id: 'hero', name: 'Hero'),
          _player(
            id: 'villain',
            name: 'Villain',
            isAllIn: true,
            hand: [
              pr.Card()
                ..value = 'A'
                ..suit = 'spades',
              pr.Card()
                ..value = 'K'
                ..suit = 'hearts',
            ],
          ),
        ],
        communityCards: const [],
        pot: 0,
        currentBet: 20,
        currentPlayerId: 'hero',
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
    }

    Widget buildOverlay(double cardSizeMultiplier) {
      return MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 390,
            height: 260,
            child: PlayerSeatsOverlay(
              gameState: buildState(),
              heroId: 'hero',
              theme: PokerThemeConfig(
                tableTheme: TableThemeConfig.classic,
                cardTheme: CardColorTheme.standard,
                cardSizeMultiplier: cardSizeMultiplier,
                uiSizeMultiplier: 1.0,
                showTableLogo: true,
                logoPosition: 'center',
              ),
              aspectRatio: 1.15,
            ),
          ),
        ),
      );
    }

    await tester.pumpWidget(buildOverlay(1.0));
    await tester.pump();
    final mediumWidth = tester.getSize(find.byType(CardFace).first).width;

    await tester.pumpWidget(buildOverlay(1.4));
    await tester.pump();
    final xlWidth = tester.getSize(find.byType(CardFace).first).width;

    expect(xlWidth, greaterThan(mediumWidth));
    expect(tester.takeException(), isNull);
  });

  testWidgets('opponent hole cards render in a separate seat rail',
      (WidgetTester tester) async {
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 0,
      currentBet: 20,
      currentPlayerId: 'hero',
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

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 390,
            height: 260,
            child: PlayerSeatsOverlay(
              gameState: gameState,
              heroId: 'hero',
              theme: const PokerThemeConfig(
                tableTheme: TableThemeConfig.classic,
                cardTheme: CardColorTheme.standard,
                cardSizeMultiplier: 1.0,
                uiSizeMultiplier: 1.0,
                showTableLogo: true,
                logoPosition: 'center',
              ),
              aspectRatio: 1.15,
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const ValueKey('seat_cards_villain')), findsOneWidget);
    expect(find.byKey(const ValueKey('seat_cards_hero')), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('committed bets render as chip stacks in front of seats',
      (WidgetTester tester) async {
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'villain', name: 'Villain', currentBet: 20),
      ],
      communityCards: const [],
      pot: 30,
      currentBet: 20,
      currentPlayerId: 'hero',
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

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 800,
            height: 600,
            child: PlayerSeatsOverlay(
              gameState: gameState,
              heroId: 'hero',
              theme: const PokerThemeConfig(
                tableTheme: TableThemeConfig.classic,
                cardTheme: CardColorTheme.standard,
                cardSizeMultiplier: 1.0,
                uiSizeMultiplier: 1.0,
                showTableLogo: true,
                logoPosition: 'center',
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const ValueKey('seat_bet_villain')), findsOneWidget);
    expect(find.byKey(const ValueKey('seat_bet_hero')), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('opponent seat footprint stays stable after folding',
      (WidgetTester tester) async {
    UiGameState buildState({required bool folded}) {
      return UiGameState(
        tableId: 'table-1',
        phase: pr.GamePhase.PRE_FLOP,
        phaseName: 'Pre-Flop',
        players: [
          _player(id: 'hero', name: 'Hero'),
          UiPlayer(
            id: 'villain',
            name: 'Villain',
            balance: 1000,
            hand: const [],
            currentBet: 0,
            folded: folded,
            isTurn: false,
            isAllIn: false,
            isDealer: false,
            isSmallBlind: false,
            isBigBlind: false,
            isReady: true,
            isDisconnected: false,
            handDesc: '',
          ),
        ],
        communityCards: const [],
        pot: 0,
        currentBet: 20,
        currentPlayerId: 'hero',
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
    }

    Widget buildOverlay(UiGameState gameState) {
      return MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 390,
            height: 260,
            child: PlayerSeatsOverlay(
              gameState: gameState,
              heroId: 'hero',
              theme: const PokerThemeConfig(
                tableTheme: TableThemeConfig.classic,
                cardTheme: CardColorTheme.standard,
                cardSizeMultiplier: 1.0,
                uiSizeMultiplier: 1.0,
                showTableLogo: true,
                logoPosition: 'center',
              ),
              aspectRatio: 1.15,
            ),
          ),
        ),
      );
    }

    await tester.pumpWidget(buildOverlay(buildState(folded: false)));
    await tester.pump();
    final beforeSize =
        tester.getSize(find.byKey(const ValueKey('seat_widget_villain')));

    await tester.pumpWidget(buildOverlay(buildState(folded: true)));
    await tester.pump();
    final afterSize =
        tester.getSize(find.byKey(const ValueKey('seat_widget_villain')));

    expect(afterSize, equals(beforeSize));
    expect(tester.takeException(), isNull);
  });
}
