import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/views/game_ended.dart';
import 'package:pokerui/components/views/table_session_view.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class _MockPokerModel extends PokerModel {
  _MockPokerModel({required super.playerId}) : super(dataDir: '/tmp/test');

  @override
  Future<void> init() async {}

  @override
  Future<void> browseTables() async {}

  @override
  Future<void> refreshGameState() async {}

  @override
  Future<void> leaveTable() async {}

  @override
  Future<bool> fold() async => false;

  @override
  Future<bool> callBet() async => false;

  @override
  Future<bool> check() async => false;

  @override
  Future<bool> makeBet(int amountChips) async => false;
}

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

UiPlayer _player({
  required String id,
  required String name,
  bool isDealer = false,
  bool isSmallBlind = false,
  bool isBigBlind = false,
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
    isTurn: false,
    isAllIn: false,
    isDealer: isDealer,
    isSmallBlind: isSmallBlind,
    isBigBlind: isBigBlind,
    isReady: true,
    isDisconnected: false,
    handDesc: '',
  );
}

UiGameState _gameState(pr.GamePhase phase) {
  return UiGameState(
    tableId: 'table-1',
    phase: phase,
    phaseName: phase == pr.GamePhase.SHOWDOWN ? 'Showdown' : 'Pre-Flop',
    players: [
      _player(
        id: 'hero',
        name: 'Hero',
        currentBet: phase == pr.GamePhase.SHOWDOWN ? 0 : 10,
        hand: [
          pr.Card()
            ..value = 'A'
            ..suit = 'spades',
          pr.Card()
            ..value = 'K'
            ..suit = 'hearts',
        ],
      ),
      _player(
        id: 'villain',
        name: 'Villain',
        currentBet: phase == pr.GamePhase.SHOWDOWN ? 0 : 20,
      ),
    ],
    communityCards: const [],
    pot: phase == pr.GamePhase.SHOWDOWN ? 30 : 60,
    currentBet: phase == pr.GamePhase.SHOWDOWN ? 0 : 20,
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

Widget _wrap({
  required Widget child,
  required Size size,
  Config? config,
}) {
  final configNotifier = ConfigNotifier()
    ..updateConfig(config ?? _defaultConfig);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(size: size),
        child: Scaffold(
          body: SizedBox(
            width: size.width,
            height: size.height,
            child: child,
          ),
        ),
      ),
    ),
  );
}

void main() {
  test('expanded and short desktop windows use compact table layout', () {
    expect(
      useCompactTableLayoutForSize(
        PokerBreakpoint.expanded,
        const Size(800, 600),
      ),
      isTrue,
    );
    expect(
      useCompactTableLayoutForSize(
        PokerBreakpoint.wide,
        const Size(1280, 800),
      ),
      isFalse,
    );
  });

  test('scene layout modes resolve as expected for target viewports', () {
    expect(
      PokerSceneLayout.resolveMode(const Size(390, 844)),
      PokerLayoutMode.compactPortrait,
    );
    expect(
      PokerSceneLayout.resolveMode(const Size(800, 600)),
      PokerLayoutMode.compactLandscape,
    );
    expect(
      PokerSceneLayout.resolveMode(const Size(1024, 768)),
      PokerLayoutMode.standard,
    );
  });

  test('desktop scene layout keeps opponent anchors above the board', () {
    final standardLayout = PokerSceneLayout.resolve(const Size(1024, 768));
    final wideLayout = PokerSceneLayout.resolve(const Size(1440, 900));

    final standardAnchor = standardLayout.opponentAnchors(2).first;
    final wideAnchor = wideLayout.opponentAnchors(2).first;

    expect(
      standardLayout.communityRect.top - standardAnchor.dy,
      greaterThan(80),
    );
    expect(
      wideLayout.communityRect.top - wideAnchor.dy,
      greaterThan(90),
    );
  });

  test('desktop two-opponent anchors straddle the table center', () {
    final standardLayout = PokerSceneLayout.resolve(const Size(1024, 768));
    final wideLayout = PokerSceneLayout.resolve(const Size(1440, 900));

    final standardAnchors = standardLayout.opponentAnchors(2);
    final wideAnchors = wideLayout.opponentAnchors(2);

    expect(
      standardAnchors.first.dx,
      lessThan(standardLayout.tableCenter.dx - 40),
    );
    expect(
      standardAnchors.last.dx,
      greaterThan(standardLayout.tableCenter.dx + 40),
    );
    expect(
      wideAnchors.first.dx,
      lessThan(wideLayout.tableCenter.dx - 60),
    );
    expect(
      wideAnchors.last.dx,
      greaterThan(wideLayout.tableCenter.dx + 60),
    );
  });

  test('maximized desktop layout caps the table footprint', () {
    final layout = PokerSceneLayout.resolve(const Size(1920, 1080));

    expect(layout.mode, PokerLayoutMode.wide);
    expect(layout.tableAspectRatio, lessThanOrEqualTo(2.08));
    expect(layout.tableRect.width, lessThanOrEqualTo(1380.0));
    expect(layout.tableRect.left, greaterThan(layout.bodyRect.left + 120.0));
    expect(
      layout.bodyRect.right - layout.tableRect.right,
      greaterThanOrEqualTo(120.0),
    );
  });

  test('ultra-wide desktop keeps the same felt cap', () {
    final maximized = PokerSceneLayout.resolve(const Size(1920, 1080));
    final ultraWide = PokerSceneLayout.resolve(const Size(2560, 1440));

    expect(maximized.mode, PokerLayoutMode.wide);
    expect(ultraWide.mode, PokerLayoutMode.wide);
    expect(maximized.tableRect.width, lessThanOrEqualTo(1380.0));
    expect(ultraWide.tableRect.width, lessThanOrEqualTo(1380.0));
    expect(ultraWide.tableRect.width,
        greaterThanOrEqualTo(maximized.tableRect.width));
  });

  test('standard desktop layout leaves consistent play-area gutters', () {
    final layout = PokerSceneLayout.resolve(const Size(1366, 768));

    expect(layout.mode, PokerLayoutMode.standard);
    expect(layout.tableAspectRatio, lessThanOrEqualTo(1.9201));
    expect(layout.tableRect.width, lessThanOrEqualTo(1120.0));
    expect(layout.tableRect.left, greaterThan(layout.bodyRect.left + 60.0));
    expect(
      layout.bodyRect.right - layout.tableRect.right,
      greaterThanOrEqualTo(60.0),
    );
  });

  test('short desktop compact layout still caps the table footprint', () {
    final layout = PokerSceneLayout.resolve(const Size(1366, 695));

    expect(layout.mode, PokerLayoutMode.compactLandscape);
    expect(layout.tableAspectRatio, lessThanOrEqualTo(2.0));
    expect(layout.tableRect.width, lessThan(layout.bodyRect.width * 0.7));
    expect(layout.tableRect.left, greaterThan(layout.bodyRect.left + 200.0));
    expect(
      layout.bodyRect.right - layout.tableRect.right,
      greaterThanOrEqualTo(200.0),
    );
  });

  testWidgets('live street bets do not also render in the center pot',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero', currentBet: 10),
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

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(800, 600),
    ));
    await tester.pump();

    expect(find.byKey(const ValueKey('seat_bet_hero')), findsOneWidget);
    expect(find.byKey(const ValueKey('seat_bet_villain')), findsOneWidget);
    expect(find.byKey(const Key('poker-pot-display')), findsNothing);
    expect(find.byKey(const Key('poker-pot-total')), findsNothing);
  });

  testWidgets(
      'hand view keeps rail and hero dock clear of the board at 800x600',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(800, 600),
    ));
    await tester.pump();

    expect(find.byType(MobileHeroActionPanel), findsNothing);
    expect(find.byType(BottomActionDock), findsOneWidget);
    expect(find.byKey(const Key('poker-right-rail')), findsNothing);
    expect(find.byKey(const Key('poker-hero-dock')), findsOneWidget);
    expect(find.byKey(const Key('desktop-bet-summary')), findsNothing);
    expect(find.byKey(const ValueKey('seat_bet_villain')), findsOneWidget);
    expect(find.byKey(const Key('poker-show-cards-toggle')), findsOneWidget);
    final potRect = tester.getRect(find.text('Pot: 30'));
    final dockRect = tester.getRect(find.byKey(const Key('poker-hero-dock')));
    final betRect =
        tester.getRect(find.byKey(const ValueKey('seat_bet_villain')));
    final toggleRect =
        tester.getRect(find.byKey(const Key('poker-show-cards-toggle')));
    expect(dockRect.top, greaterThan(potRect.bottom));
    expect(toggleRect.bottom, lessThan(dockRect.bottom));
    expect(betRect.bottom, lessThan(dockRect.top));
  });

  testWidgets(
      'showdown view keeps rail and hero dock clear of the board at 800x600',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.SHOWDOWN);
    model.lastWinners = const [
      UiWinner(
        playerId: 'hero',
        handRank: pr.HandRank.PAIR,
        bestHand: [],
        winnings: 30,
      ),
    ];

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(800, 600),
    ));
    await tester.pump();

    expect(find.byType(MobileHeroActionPanel), findsNothing);
    expect(find.byType(BottomActionDock), findsOneWidget);
    expect(find.byKey(const Key('poker-right-rail')), findsNothing);
    expect(find.byKey(const Key('poker-hero-dock')), findsOneWidget);
    expect(find.byKey(const Key('poker-show-cards-toggle')), findsOneWidget);
    final hiddenSidebarRect =
        tester.getRect(find.byKey(const Key('showdown-sidebar')));
    expect(hiddenSidebarRect.right, lessThanOrEqualTo(0));
    final dockRect = tester.getRect(find.byKey(const Key('poker-hero-dock')));
    final boardPotRect =
        tester.getRect(find.byKey(const Key('poker-pot-display')));
    expect(dockRect.top, greaterThan(boardPotRect.bottom));
  });

  testWidgets(
      'active hand opens the last-hand sidebar from a clear top-left button',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);
    model.setShowdownDataForTest(
      players: model.game!.players,
      communityCards: const [],
      pot: 30,
      winners: const [
        UiWinner(
          playerId: 'hero',
          handRank: pr.HandRank.PAIR,
          bestHand: [],
          winnings: 30,
        ),
      ],
    );

    // Wide enough that 48% viewport width still fits five board cards in one row
    // (otherwise the sidebar goes full width and this test expects a partial panel).
    const viewport = Size(1600, 764);
    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: viewport,
    ));
    await tester.pump();

    expect(find.text('Last Hand'), findsOneWidget);
    expect(find.byType(PokerLastHandButton), findsOneWidget);

    final buttonRect = tester.getRect(find.byType(PokerLastHandButton));
    final dockRect = tester.getRect(find.byKey(const Key('poker-hero-dock')));
    final tableRect = tester.getRect(find.byType(PokerTableBackground));

    expect(buttonRect.left, lessThan(tableRect.left + 32));
    expect(buttonRect.top, lessThan(tableRect.top + 32));
    expect(buttonRect.bottom, lessThan(dockRect.top - 24));

    await tester.tap(find.byType(PokerLastHandButton));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('showdown-sidebar')), findsOneWidget);
    expect(find.text('Showdown'), findsOneWidget);
    expect(find.textContaining('Winner', findRichText: true), findsWidgets);
    expect(find.textContaining('Pair', findRichText: true), findsWidgets);
    expect(find.textContaining('+30', findRichText: true), findsWidgets);
    expect(find.byTooltip('Close last hand details'), findsOneWidget);
    final sidebarRect =
        tester.getRect(find.byKey(const Key('showdown-sidebar')));
    expect(sidebarRect.left, 0);
    expect(sidebarRect.top, 0);
    expect(sidebarRect.height, lessThan(viewport.height));
    expect(sidebarRect.width, lessThan(viewport.width));

    await tester.tap(find.byTooltip('Close last hand details'));
    await tester.pumpAndSettle();

    final hiddenSidebarRect =
        tester.getRect(find.byKey(const Key('showdown-sidebar')));
    expect(hiddenSidebarRect.right, lessThanOrEqualTo(0));
    expect(find.byType(PokerLastHandButton), findsOneWidget);
  });

  testWidgets('showdown sidebar content scrolls inside the panel',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.setShowdownDataForTest(
      players: List<UiPlayer>.generate(
        12,
        (index) => _player(
          id: 'p$index',
          name: 'Player ${index + 1}',
          hand: index.isEven
              ? [
                  pr.Card()
                    ..value = 'A'
                    ..suit = 'spades',
                  pr.Card()
                    ..value = 'K'
                    ..suit = 'hearts',
                ]
              : const [],
        ),
      ),
      communityCards: const [],
      pot: 120,
      winners: const [
        UiWinner(
          playerId: 'p0',
          handRank: pr.HandRank.PAIR,
          bestHand: [],
          winnings: 120,
        ),
      ],
    );

    await tester.pumpWidget(_wrap(
      child: ShowdownSidebar(
        showdown: model.showdown!,
        heroId: model.playerId,
        visible: true,
      ),
      size: const Size(360, 420),
    ));
    await tester.pumpAndSettle();

    final playerHands = find.text('Hands');
    final initialTop = tester.getTopLeft(playerHands).dy;

    await tester.drag(
      find.byKey(const Key('showdown-sidebar-scroll')),
      const Offset(0, -180),
    );
    await tester.pumpAndSettle();

    final sidebarCardSize =
        tester.getSize(find.byKey(const Key('showdown-player-card-p0-0')));
    final scrolledTop = tester.getTopLeft(playerHands).dy;
    expect(sidebarCardSize.width, greaterThan(38));
    expect(sidebarCardSize.height, greaterThan(55));
    expect(scrolledTop, lessThan(initialTop));
  });

  testWidgets('game ended preview enlarges last-hand community cards',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.setShowdownDataForTest(
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [
            pr.Card()
              ..value = 'Q'
              ..suit = 'spades',
            pr.Card()
              ..value = 'Q'
              ..suit = 'hearts',
          ],
        ),
        _player(
          id: 'villain',
          name: 'Villain',
          hand: [
            pr.Card()
              ..value = 'J'
              ..suit = 'clubs',
            pr.Card()
              ..value = '10'
              ..suit = 'clubs',
          ],
        ),
      ],
      communityCards: [
        pr.Card()
          ..value = 'A'
          ..suit = 'spades',
        pr.Card()
          ..value = 'K'
          ..suit = 'hearts',
        pr.Card()
          ..value = '10'
          ..suit = 'diamonds',
        pr.Card()
          ..value = '7'
          ..suit = 'clubs',
        pr.Card()
          ..value = '2'
          ..suit = 'spades',
      ],
      pot: 180,
      winners: const [
        UiWinner(
          playerId: 'hero',
          handRank: pr.HandRank.PAIR,
          bestHand: [],
          winnings: 180,
        ),
      ],
    );

    await tester.pumpWidget(
      _wrap(
        child: GameEndedView(model: model),
        size: const Size(430, 900),
      ),
    );
    await tester.pump();

    final boardCardSize =
        tester.getSize(find.byKey(const Key('game-ended-showdown-card-0')));

    expect(boardCardSize.width, greaterThan(45));
    expect(boardCardSize.height, greaterThan(65));
    final viewShowdownButton =
        find.byKey(const Key('game-ended-view-showdown-button'));
    expect(viewShowdownButton, findsOneWidget);

    await tester.ensureVisible(viewShowdownButton);
    await tester.tap(viewShowdownButton);
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('last-showdown-dialog')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('game ended does not render stale last-showdown preview',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.currentTableId = 'table-1';
    model.gameEndingMessage = 'Game ended';
    model.setShowdownDataForTest(
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [
            pr.Card()
              ..value = 'Q'
              ..suit = 'spades',
            pr.Card()
              ..value = 'Q'
              ..suit = 'hearts',
          ],
        ),
        _player(
          id: 'villain',
          name: 'Villain',
          hand: [
            pr.Card()
              ..value = 'J'
              ..suit = 'clubs',
            pr.Card()
              ..value = '10'
              ..suit = 'clubs',
          ],
        ),
      ],
      communityCards: [
        pr.Card()
          ..value = 'A'
          ..suit = 'spades',
        pr.Card()
          ..value = 'K'
          ..suit = 'hearts',
        pr.Card()
          ..value = '10'
          ..suit = 'diamonds',
        pr.Card()
          ..value = '7'
          ..suit = 'clubs',
        pr.Card()
          ..value = '2'
          ..suit = 'spades',
      ],
      pot: 180,
      winners: const [
        UiWinner(
          playerId: 'hero',
          handRank: pr.HandRank.PAIR,
          bestHand: [],
          winnings: 180,
        ),
      ],
    );
    model.applyNotificationForTest(pr.Notification(
      type: pr.NotificationType.NEW_HAND_STARTED,
      tableId: 'table-1',
    ));

    await tester.pumpWidget(
      _wrap(
        child: GameEndedView(model: model),
        size: const Size(430, 900),
      ),
    );
    await tester.pump();

    expect(find.byKey(const Key('game-ended-showdown-card-0')), findsNothing);
    expect(
      find.byKey(const Key('game-ended-view-showdown-button')),
      findsNothing,
    );
    expect(find.text('Game ended'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
      'top seat badges stay above the pot/community area on small screens',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero', isBigBlind: true),
        _player(
          id: 'villain',
          name: 'Villain',
          isDealer: true,
          isSmallBlind: true,
        ),
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

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(800, 600),
    ));
    await tester.pump();

    final dealerRect = tester.getRect(find.text('D'));
    final smallBlindRect = tester.getRect(find.text('SB').first);
    final potRect = tester.getRect(find.text('Pot: 30'));

    expect(dealerRect.bottom, lessThan(potRect.top));
    expect(smallBlindRect.bottom, lessThan(potRect.top));
  });

  testWidgets('phone portrait stays readable with xl card size',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(390, 844),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    expect(find.byType(MobileHeroActionPanel), findsOneWidget);
    expect(find.byKey(const Key('poker-right-rail')), findsNothing);
    final dockRect = tester.getRect(find.byKey(const Key('poker-hero-dock')));
    final potRect = tester.getRect(find.byKey(const Key('poker-pot-display')));
    final heroCardRect = tester.getRect(find.byType(CardFace).first);
    final betRect =
        tester.getRect(find.byKey(const ValueKey('seat_bet_villain')));
    final toggleRect =
        tester.getRect(find.byKey(const Key('poker-show-cards-toggle')));

    expect(find.byKey(const Key('desktop-bet-summary')), findsNothing);
    expect(find.byKey(const ValueKey('seat_bet_villain')), findsOneWidget);
    expect(find.byKey(const Key('poker-show-cards-toggle')), findsOneWidget);
    expect(dockRect.top, greaterThan(potRect.bottom));
    expect(potRect.bottom, lessThan(heroCardRect.top));
    expect(heroCardRect.width, greaterThan(50));
    expect(toggleRect.bottom, lessThan(dockRect.bottom));
    expect(betRect.bottom, lessThan(dockRect.top));
    expect(tester.takeException(), isNull);
  });

  testWidgets('phone portrait opponents stay clear of community slots',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(id: 'hero', name: 'Hero'),
        _player(id: 'left', name: 'Left', isSmallBlind: true),
        _player(id: 'top', name: 'Top', isBigBlind: true),
        _player(id: 'right', name: 'Right', isDealer: true),
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

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(390, 844),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    final slotRects = List<Rect>.generate(
      5,
      (index) => tester.getRect(find.byKey(ValueKey('community_slot_$index'))),
    );
    final seatRects = [
      tester.getRect(find.byKey(const ValueKey('seat_widget_left'))),
      tester.getRect(find.byKey(const ValueKey('seat_widget_top'))),
      tester.getRect(find.byKey(const ValueKey('seat_widget_right'))),
    ];

    for (final seatRect in seatRects) {
      for (final slotRect in slotRects) {
        expect(seatRect.overlaps(slotRect), isFalse);
      }
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('phone portrait three opponents stay in upper seats',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
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

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(390, 844),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    final layout = PokerSceneLayout.resolve(const Size(390, 844));
    final seatRects = [
      tester.getRect(find.byKey(const ValueKey('seat_widget_left'))),
      tester.getRect(find.byKey(const ValueKey('seat_widget_top'))),
      tester.getRect(find.byKey(const ValueKey('seat_widget_right'))),
    ];

    for (final rect in seatRects) {
      expect(rect.center.dy, lessThan(layout.tableCenter.dy));
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('phone portrait upper arc seats keep cards above the seat core',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
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

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(390, 844),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    final leftCardsRect =
        tester.getRect(find.byKey(const ValueKey('seat_cards_left')));
    final leftCoreRect =
        tester.getRect(find.byKey(const ValueKey('seat_core_left')));
    final rightCardsRect =
        tester.getRect(find.byKey(const ValueKey('seat_cards_right')));
    final rightCoreRect =
        tester.getRect(find.byKey(const ValueKey('seat_core_right')));

    expect(leftCardsRect.center.dy, lessThan(leftCoreRect.center.dy));
    expect(rightCardsRect.center.dy, lessThan(rightCoreRect.center.dy));
    expect(leftCardsRect.right, greaterThan(leftCoreRect.left + 12));
    expect(leftCardsRect.left, lessThan(leftCoreRect.right - 12));
    expect(rightCardsRect.right, greaterThan(rightCoreRect.left + 12));
    expect(rightCardsRect.left, lessThan(rightCoreRect.right - 12));
    expect(tester.takeException(), isNull);
  });

  testWidgets('1024x768 standard layout keeps desktop dock and rail separated',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(1024, 768),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    expect(find.byType(BottomActionDock), findsOneWidget);
    expect(find.byType(MobileHeroActionPanel), findsNothing);
    expect(find.byKey(const Key('poker-right-rail')), findsNothing);
    expect(find.byKey(const Key('desktop-bet-summary')), findsNothing);
    expect(find.byKey(const ValueKey('seat_bet_villain')), findsOneWidget);
    final dockRect = tester.getRect(find.byKey(const Key('poker-hero-dock')));
    final potRect = tester.getRect(find.text('Pot: 30'));
    final heroCardRect = tester.getRect(find.byType(CardFace).first);

    expect(dockRect.top, greaterThan(potRect.bottom));
    // Allow a few px slack for font/layout drift across Flutter / theme versions.
    expect(heroCardRect.bottom, lessThan(dockRect.top + 12));
  });

  testWidgets('desktop hero cards stay anchored across action state changes',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);

    await tester.pumpWidget(_wrap(
      child: TableSessionView(model: model),
      size: const Size(1024, 768),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    final activeCardRect = tester.getRect(find.byType(CardFace).first);
    expect(find.textContaining('Call 10'), findsOneWidget);

    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [
            pr.Card()
              ..value = 'A'
              ..suit = 'spades',
            pr.Card()
              ..value = 'K'
              ..suit = 'hearts',
          ],
        ),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 30,
      currentBet: 0,
      currentPlayerId: 'villain',
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
    model.notifyListeners();
    await tester.pump();

    final waitingCardRect = tester.getRect(find.byType(CardFace).first);
    expect(waitingCardRect.left, closeTo(activeCardRect.left, 0.01));
    expect(waitingCardRect.top, closeTo(activeCardRect.top, 0.01));
  });

  testWidgets('desktop action buttons stay pinned across dock state changes',
      (WidgetTester tester) async {
    final betCtrl = TextEditingController();
    addTearDown(betCtrl.dispose);
    final idleModel = _MockPokerModel(playerId: 'hero');
    idleModel.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [
            pr.Card()
              ..value = 'A'
              ..suit = 'spades',
            pr.Card()
              ..value = 'K'
              ..suit = 'hearts',
          ],
        ),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 30,
      currentBet: 0,
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

    await tester.pumpWidget(_wrap(
      child: Align(
        alignment: Alignment.bottomCenter,
        child: SizedBox(
          width: 1024,
          height: 180,
          child: BottomActionDock(
            model: idleModel,
            showBetInput: false,
            betCtrl: betCtrl,
            onToggleBetInput: () {},
            onCloseBetInput: () {},
          ),
        ),
      ),
      size: const Size(1024, 768),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    final activeFoldBottom = tester.getRect(find.text('Fold')).bottom;

    final betModel = _MockPokerModel(playerId: 'hero');
    betModel.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [
            pr.Card()
              ..value = 'A'
              ..suit = 'spades',
            pr.Card()
              ..value = 'K'
              ..suit = 'hearts',
          ],
        ),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 30,
      currentBet: 0,
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
    await tester.pumpWidget(_wrap(
      child: Align(
        alignment: Alignment.bottomCenter,
        child: SizedBox(
          width: 1024,
          height: 180,
          child: BottomActionDock(
            model: betModel,
            showBetInput: false,
            betCtrl: betCtrl,
            onToggleBetInput: () {},
            onCloseBetInput: () {},
          ),
        ),
      ),
      size: const Size(1024, 768),
      config: _defaultConfig.copyWith(cardSize: 'xl'),
    ));
    await tester.pump();

    expect(
      tester.getRect(find.text('Fold')).bottom,
      closeTo(activeFoldBottom, 0.01),
    );
  });

  testWidgets('table rebuild does not steal focus from text inputs',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = _gameState(pr.GamePhase.PRE_FLOP);
    final gameFocusNode = FocusNode(debugLabel: 'game');
    final textFocusNode = FocusNode(debugLabel: 'bet-input');
    addTearDown(gameFocusNode.dispose);
    addTearDown(textFocusNode.dispose);

    await tester.pumpWidget(_wrap(
      size: const Size(390, 844),
      child: StatefulBuilder(
        builder: (context, setState) {
          final theme = PokerThemeConfig.fromContext(context);
          final game = PokerGame(model.playerId, model, theme: theme);
          return Stack(
            fit: StackFit.expand,
            children: [
              game.buildWidget(
                model.game!,
                gameFocusNode,
                aspectRatio: 16 / 9,
              ),
              Positioned(
                left: 16,
                right: 16,
                bottom: 16,
                child: Material(
                  child: TextField(
                    key: const ValueKey('overlay-input'),
                    focusNode: textFocusNode,
                  ),
                ),
              ),
              Positioned(
                top: 16,
                right: 16,
                child: ElevatedButton(
                  onPressed: () => setState(() {}),
                  child: const Text('Rebuild'),
                ),
              ),
            ],
          );
        },
      ),
    ));
    await tester.pump();

    expect(gameFocusNode.hasFocus, isTrue);

    await tester.tap(find.byType(TextField));
    await tester.pump();

    expect(textFocusNode.hasFocus, isTrue);

    await tester.tap(find.text('Rebuild'));
    await tester.pump();

    expect(textFocusNode.hasFocus, isTrue);
    expect(gameFocusNode.hasFocus, isFalse);
  });
}
