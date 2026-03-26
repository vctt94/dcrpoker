import 'dart:math' as math;

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

  testWidgets('smaller square layouts keep opponents fully on-screen',
      (WidgetTester tester) async {
    const viewport = Size(754, 767);
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'left', name: 'Left'),
        _player(id: 'top', name: 'Top'),
        _player(id: 'right', name: 'Right'),
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
      playersJoined: 4,
      timeBankSeconds: 30,
      turnDeadlineUnixMs: 0,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: viewport.width,
            height: viewport.height,
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

    final leftRect =
        tester.getRect(find.byKey(const ValueKey('seat_widget_left')));
    final topRect =
        tester.getRect(find.byKey(const ValueKey('seat_widget_top')));
    final rightRect =
        tester.getRect(find.byKey(const ValueKey('seat_widget_right')));
    expect(leftRect.overlaps(topRect), isFalse);
    expect(topRect.overlaps(rightRect), isFalse);
    expect(leftRect.overlaps(rightRect), isFalse);
    for (final rect in [leftRect, topRect, rightRect]) {
      expect(rect.left, greaterThanOrEqualTo(0));
      expect(rect.top, greaterThanOrEqualTo(0));
      expect(rect.right, lessThanOrEqualTo(viewport.width));
      expect(rect.bottom, lessThanOrEqualTo(viewport.height));
    }
    expect(leftRect.center.dx, lessThan(topRect.center.dx));
    expect(rightRect.center.dx, greaterThan(topRect.center.dx));
    expect(tester.takeException(), isNull);
  });

  testWidgets('six-player layouts keep every seat on-screen',
      (WidgetTester tester) async {
    const viewport = Size(900, 700);
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'p2', name: 'P2'),
        _player(id: 'p3', name: 'P3'),
        _player(id: 'p4', name: 'P4'),
        _player(id: 'p5', name: 'P5'),
        _player(id: 'p6', name: 'P6'),
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
      playersJoined: 6,
      timeBankSeconds: 30,
      turnDeadlineUnixMs: 0,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: viewport.width,
            height: viewport.height,
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

    for (final id in ['hero', 'p2', 'p3', 'p4', 'p5', 'p6']) {
      final rect = tester.getRect(find.byKey(ValueKey('seat_widget_$id')));
      expect(rect.left, greaterThanOrEqualTo(0));
      expect(rect.top, greaterThanOrEqualTo(0));
      expect(rect.right, lessThanOrEqualTo(viewport.width));
      expect(rect.bottom, lessThanOrEqualTo(viewport.height));
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('six-player opponent cards stay attached above seat cores',
      (WidgetTester tester) async {
    const viewport = Size(900, 700);
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'p2', name: 'P2'),
        _player(id: 'p3', name: 'P3'),
        _player(id: 'p4', name: 'P4'),
        _player(id: 'p5', name: 'P5'),
        _player(id: 'p6', name: 'P6'),
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
      playersJoined: 6,
      timeBankSeconds: 30,
      turnDeadlineUnixMs: 0,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: viewport.width,
            height: viewport.height,
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

    for (final id in ['p2', 'p3', 'p4', 'p5', 'p6']) {
      final cardsRect = tester.getRect(find.byKey(ValueKey('seat_cards_$id')));
      final coreRect = tester.getRect(find.byKey(ValueKey('seat_core_$id')));
      final horizontalOverlap = math.min(cardsRect.right, coreRect.right) -
          math.max(cardsRect.left, coreRect.left);

      expect(horizontalOverlap, greaterThan(12));
      expect(cardsRect.center.dy, lessThan(coreRect.center.dy));
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('opponent bet stacks do not overlap seat info plates',
      (WidgetTester tester) async {
    const viewport = Size(900, 700);
    final gameState = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'left', name: 'Left', currentBet: 10),
        _player(id: 'top', name: 'Top', currentBet: 20),
        _player(id: 'right', name: 'Right', currentBet: 30),
      ],
      communityCards: const [],
      pot: 60,
      currentBet: 30,
      currentPlayerId: 'hero',
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

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: viewport.width,
            height: viewport.height,
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

    for (final id in ['left', 'top', 'right']) {
      final plateRect = tester.getRect(find.byKey(ValueKey('seat_plate_$id')));
      final betRect = tester.getRect(find.byKey(ValueKey('seat_bet_$id')));
      expect(
        betRect.overlaps(plateRect),
        isFalse,
        reason: 'bet stack for $id overlaps seat info plate',
      );
    }
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
