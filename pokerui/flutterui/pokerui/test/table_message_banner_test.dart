import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:provider/provider.dart';

class _TestPokerModel extends PokerModel {
  _TestPokerModel({required super.playerId}) : super(dataDir: '/tmp/test');

  @override
  PokerState get state => PokerState.handInProgress;
}

final _defaultConfig = Config(
  serverAddr: '127.0.0.1:50051',
  grpcCertPath: '',
  nickname: '',
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
    isDealer: false,
    isSmallBlind: false,
    isBigBlind: false,
    isReady: true,
    isDisconnected: false,
    handDesc: '',
  );
}

UiGameState _gameState() {
  return UiGameState(
    tableId: 'table-1',
    phase: pr.GamePhase.PRE_FLOP,
    phaseName: 'Pre-Flop',
    players: [
      _player(
        id: 'hero',
        name: 'Hero',
        currentBet: 10,
        hand: [
          pr.Card()
            ..value = 'A'
            ..suit = 'spades',
          pr.Card()
            ..value = 'K'
            ..suit = 'hearts',
        ],
      ),
      _player(id: 'villain', name: 'Villain', currentBet: 20),
    ],
    communityCards: const [],
    pot: 60,
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

Widget _wrap({
  required Size size,
  required Widget child,
  EdgeInsets padding = EdgeInsets.zero,
}) {
  final configNotifier = ConfigNotifier()..updateConfig(_defaultConfig);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(size: size, padding: padding),
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
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('compact phone table message sits below the top control band',
      (tester) async {
    const size = Size(390, 844);
    const safePadding = EdgeInsets.only(top: 47, bottom: 34);
    tester.view.devicePixelRatio = 1.0;
    tester.view.physicalSize = size;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    final model = _TestPokerModel(playerId: 'hero')
      ..game = _gameState()
      ..tableMessage =
          'Server restart requested. Current hand will finish, and no new hand will start.';
    final focusNode = FocusNode();
    addTearDown(focusNode.dispose);

    await tester.pumpWidget(
      _wrap(
        size: size,
        padding: safePadding,
        child: Builder(
          builder: (context) {
            final theme = PokerThemeConfig.fromContext(context);
            final game = PokerGame(model.playerId, model, theme: theme);
            return game.buildWidget(model.game!, focusNode);
          },
        ),
      ),
    );
    await tester.pump();

    final bannerRect =
        tester.getRect(find.byKey(const Key('table-message-banner')));
    final scene = PokerSceneLayout.resolve(size, safePadding: safePadding);

    expect(
      bannerRect.top,
      closeTo(
        scene.safeRect.top + PokerSpacing.md + 44.0 + PokerSpacing.sm,
        0.01,
      ),
    );
    expect(bannerRect.center.dx, closeTo(size.width / 2, 0.01));
  });
}
