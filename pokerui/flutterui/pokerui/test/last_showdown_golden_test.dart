import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
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

Widget _wrapWithProviders(Widget child, {Size size = const Size(800, 800)}) {
  final configNotifier = ConfigNotifier()..updateConfig(_defaultConfig);
  return ChangeNotifierProvider<ConfigNotifier>.value(
    value: configNotifier,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(size: size),
        child: Scaffold(
          backgroundColor: Colors.black,
          body: child,
        ),
      ),
    ),
  );
}

UiPlayer _player({
  required String id,
  required String name,
  List<pr.Card> hand = const [],
  String handDesc = '',
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
    handDesc: handDesc,
  );
}

pr.Card _card(String value, String suit) {
  return pr.Card()
    ..value = value
    ..suit = suit;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const globalAudioChannel = MethodChannel('xyz.luan/audioplayers.global');
  const globalAudioEventsChannel =
      MethodChannel('xyz.luan/audioplayers.global/events');
  const audioChannel = MethodChannel('xyz.luan/audioplayers');

  setUpAll(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(globalAudioChannel, (call) async {
      if (call.method == 'init') return null;
      return null;
    });
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(globalAudioEventsChannel, (call) async {
      if (call.method == 'listen' || call.method == 'cancel') return null;
      return null;
    });
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(audioChannel, (call) async {
      final args = call.arguments;
      if (call.method == 'create' &&
          args is Map &&
          args['playerId'] is String) {
        final playerId = args['playerId'] as String;
        final playerEventsChannel =
            MethodChannel('xyz.luan/audioplayers/events/$playerId');
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(playerEventsChannel, (eventCall) async {
          if (eventCall.method == 'listen' || eventCall.method == 'cancel') {
            return null;
          }
          return null;
        });
      }
      return null;
    });
  });

  tearDownAll(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(globalAudioChannel, null);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(globalAudioEventsChannel, null);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(audioChannel, null);
  });

  group('Last Showdown Goldens', () {
    testWidgets('last hand button matches golden', (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(240, 120);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      await tester.pumpWidget(
        _wrapWithProviders(
          const Center(
            child: RepaintBoundary(
              child: SizedBox(
                key: Key('last-hand-button-surface'),
                width: 160,
                height: 56,
                child: Center(
                  child: PokerLastHandButton(
                    onTap: _noop,
                  ),
                ),
              ),
            ),
          ),
          size: const Size(240, 120),
        ),
      );
      await tester.pumpAndSettle();

      await expectLater(
        find.byKey(const Key('last-hand-button-surface')),
        matchesGoldenFile('goldens/last_hand_button.png'),
      );
    });

    testWidgets('last showdown sidebar matches golden',
        (WidgetTester tester) async {
      tester.binding.window.physicalSizeTestValue = const Size(800, 800);
      tester.binding.window.devicePixelRatioTestValue = 1.0;
      addTearDown(tester.binding.window.clearPhysicalSizeTestValue);
      addTearDown(tester.binding.window.clearDevicePixelRatioTestValue);

      final model = _MockPokerModel(playerId: 'hero');
      final communityCards = [
        _card('A', 'Hearts'),
        _card('K', 'Hearts'),
        _card('Q', 'Hearts'),
        _card('J', 'Hearts'),
        _card('10', 'Hearts'),
      ];
      final players = [
        _player(
          id: 'hero',
          name: 'Hero',
          hand: [_card('A', 'Spades'), _card('K', 'Spades')],
          handDesc: 'Royal Flush',
        ),
        _player(
          id: 'villain',
          name: 'Alice',
          hand: [_card('Q', 'Clubs'), _card('Q', 'Diamonds')],
          handDesc: 'Three of a Kind',
        ),
      ];
      model.setShowdownDataForTest(
        players: players,
        communityCards: communityCards,
        pot: 1500,
        winners: const [
          UiWinner(
            playerId: 'hero',
            handRank: pr.HandRank.ROYAL_FLUSH,
            bestHand: [],
            winnings: 1500,
          ),
        ],
      );

      await tester.pumpWidget(
        _wrapWithProviders(
          Center(
            child: RepaintBoundary(
              child: SizedBox(
                key: const Key('last-showdown-sidebar-surface'),
                width: 360,
                height: 620,
                child: ShowdownSidebar(
                  model: model,
                  visible: true,
                  onClose: _noop,
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await expectLater(
        find.byKey(const Key('last-showdown-sidebar-surface')),
        matchesGoldenFile('goldens/last_showdown_sidebar.png'),
      );
    });
  });
}

void _noop() {}
