import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/cards.dart';
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
}) {
  final configNotifier = ConfigNotifier()..updateConfig(config);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(size: size),
        child: Scaffold(
          body: SizedBox(
            width: size.width,
            height: size.height,
            child: MobileHeroActionPanel.passive(
              model: model,
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('mobile hero cards respect card_size config',
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

    const phoneSize = Size(390, 844);

    await tester.pumpWidget(_wrap(
      model: model,
      config: _configWithCardSize('xs'),
      size: phoneSize,
    ));
    await tester.pump();

    final xsWidth = tester.getSize(find.byType(CardFace).first).width;

    await tester.pumpWidget(_wrap(
      model: model,
      config: _configWithCardSize('xl'),
      size: phoneSize,
    ));
    await tester.pump();

    final xlWidth = tester.getSize(find.byType(CardFace).first).width;

    expect(xsWidth, closeTo(25.2, 0.1));
    expect(xlWidth, closeTo(58.8, 0.1));
    expect(xlWidth, greaterThan(xsWidth));
  });
}
