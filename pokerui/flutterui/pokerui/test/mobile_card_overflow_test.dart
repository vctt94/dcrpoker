import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
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
}

Config _configWithCardSize(String cardSize) {
  return Config(
    serverAddr: '127.0.0.1:50051',
    grpcCertPath: '',
    payoutAddress: '',
    debugLevel: 'info',
    soundsEnabled: false,
    dataDir: '/tmp/test',
    address: '',
    tableTheme: 'classic',
    cardTheme: 'standard',
    cardSize: cardSize,
    uiSize: 'medium',
    hideTableLogo: false,
    logoPosition: 'center',
  );
}

pr.Card _card(String value, String suit) {
  return pr.Card()
    ..value = value
    ..suit = suit;
}

UiPlayer _player({
  required String id,
  required String name,
  List<pr.Card> hand = const [],
}) {
  return UiPlayer(
    id: id,
    name: name,
    balance: 1000,
    hand: hand,
    currentBet: 0,
    folded: false,
    isTurn: false,
    isAllIn: false,
    isDealer: false,
    isSmallBlind: false,
    isBigBlind: false,
    isReady: true,
    isDisconnected: false,
    handDesc: '',
  );
}

Widget _wrap({
  required PokerModel model,
  required Config config,
  required Size size,
  required double panelWidth,
  double? panelHeight,
  bool showActions = false,
  bool hasLastShowdown = true,
  TextEditingController? betCtrl,
}) {
  final configNotifier = ConfigNotifier()..updateConfig(config);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(size: size),
        child: Scaffold(
          body: SizedBox(
            width: panelWidth,
            height: panelHeight,
            child: showActions
                ? MobileHeroActionPanel(
                    model: model,
                    showBetInput: false,
                    betCtrl: betCtrl!,
                    onToggleBetInput: () {},
                    onCloseBetInput: () {},
                    hasLastShowdown: hasLastShowdown,
                    onShowLastHand: () {},
                  )
                : MobileHeroActionPanel.passive(
                    model: model,
                    hasLastShowdown: hasLastShowdown,
                    onShowLastHand: () {},
                  ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('mobile hero header avoids overflow on narrow widths',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [_card('A', 'spades'), _card('K', 'hearts')],
        ),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 0,
      currentBet: 0,
      currentPlayerId: '',
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
      model: model,
      config: _configWithCardSize('xl'),
      size: const Size(390, 844),
      panelWidth: 202,
    ));
    await tester.pump();

    expect(find.byIcon(Icons.history), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('mobile hero panel avoids overflow in cramped action state',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.PRE_FLOP,
      phaseName: 'Pre-Flop',
      players: [
        UiPlayer(
          id: 'hero',
          name: 'Hero',
          balance: 30,
          hand: [_card('2', 'hearts'), _card('9', 'diamonds')],
          currentBet: 0,
          folded: false,
          isTurn: false,
          isAllIn: false,
          isDealer: true,
          isSmallBlind: false,
          isBigBlind: false,
          isReady: true,
          isDisconnected: false,
          handDesc: '',
        ),
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 30,
      currentBet: 20,
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

    final betCtrl = TextEditingController(text: '20');
    addTearDown(betCtrl.dispose);

    await tester.pumpWidget(_wrap(
      model: model,
      config: _configWithCardSize('medium'),
      size: const Size(426.5, 151.1),
      panelWidth: 426.5,
      panelHeight: 151.1,
      showActions: true,
      hasLastShowdown: false,
      betCtrl: betCtrl,
    ));
    await tester.pump();

    expect(find.byType(MobileHeroActionPanel), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('mobile action buttons stay pinned when bet summary appears',
      (WidgetTester tester) async {
    final model = _MockPokerModel(playerId: 'hero');
    final basePlayers = [
      UiPlayer(
        id: 'hero',
        name: 'Hero',
        balance: 1000,
        hand: [_card('J', 'diamonds'), _card('7', 'hearts')],
        currentBet: 0,
        folded: false,
        isTurn: true,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: true,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
      ),
      _player(id: 'villain', name: 'Villain'),
    ];
    final betCtrl = TextEditingController();
    addTearDown(betCtrl.dispose);

    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.FLOP,
      phaseName: 'Flop',
      players: basePlayers,
      communityCards: const [],
      pot: 40,
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
      model: model,
      config: _configWithCardSize('medium'),
      size: const Size(390, 844),
      panelWidth: 390,
      panelHeight: 176,
      showActions: true,
      hasLastShowdown: false,
      betCtrl: betCtrl,
    ));
    await tester.pump();

    final initialFoldBottom = tester.getRect(find.text('Fold')).bottom;

    model.game = UiGameState(
      tableId: 'table-1',
      phase: pr.GamePhase.FLOP,
      phaseName: 'Flop',
      players: [
        basePlayers.first,
        _player(id: 'villain', name: 'Villain'),
      ],
      communityCards: const [],
      pot: 40,
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
      model: model,
      config: _configWithCardSize('medium'),
      size: const Size(390, 844),
      panelWidth: 390,
      panelHeight: 176,
      showActions: true,
      hasLastShowdown: false,
      betCtrl: betCtrl,
    ));
    await tester.pump();

    expect(find.textContaining('Bet 20'), findsNothing);
    expect(find.textContaining('Call 20'), findsOneWidget);
    expect(tester.getRect(find.text('Fold')).bottom,
        closeTo(initialFoldBottom, 0.01));
    expect(tester.takeException(), isNull);
  });
}
