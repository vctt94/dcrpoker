import 'dart:convert';

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:provider/provider.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/screens/home.dart';
import 'package:pokerui/theme/poker_theme.dart';

class _EscrowRefreshTestModel extends PokerModel {
  _EscrowRefreshTestModel({required super.playerId})
      : super(dataDir: '/tmp/pokerui-test');

  List<Map<String, dynamic>> cachedEscrows = const [];

  @override
  Future<List<Map<String, dynamic>>> listCachedEscrows() async => cachedEscrows
      .map((escrow) => Map<String, dynamic>.from(escrow))
      .toList(growable: false);
}

void main() {
  UiTable table({
    required String id,
    required String name,
    int buyInAtoms = 0,
    int currentPlayers = 2,
  }) {
    return UiTable(
      id: id,
      name: name,
      players: const [],
      smallBlind: 10,
      bigBlind: 20,
      maxPlayers: 6,
      minPlayers: 2,
      currentPlayers: currentPlayers,
      buyInAtoms: buyInAtoms,
      phase: pr.GamePhase.WAITING,
      gameStarted: false,
      allReady: false,
    );
  }

  pr.Player player({
    required String id,
    required String name,
    required int tableSeat,
    String escrowId = '',
    bool escrowReady = false,
  }) {
    return pr.Player(
      id: id,
      name: name,
      balance: Int64(1000),
      currentBet: Int64(0),
      isReady: true,
      playerState: pr.PlayerState.PLAYER_STATE_IN_GAME,
      tableSeat: tableSeat,
      escrowId: escrowId,
      escrowReady: escrowReady,
    );
  }

  pr.GameUpdate activeHand({
    required String tableId,
    required List<pr.Player> players,
    required String currentPlayer,
  }) {
    return pr.GameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.PRE_FLOP,
      players: players,
      communityCards: const [],
      pot: Int64(0),
      currentBet: Int64(20),
      currentPlayer: currentPlayer,
      minRaise: Int64(20),
      maxRaise: Int64(1000),
      gameStarted: true,
      playersRequired: 2,
      playersJoined: players.length,
      phaseName: 'Pre-Flop',
      timeBankSeconds: 30,
      turnDeadlineUnixMs: Int64(0),
      smallBlind: Int64(10),
      bigBlind: Int64(20),
    );
  }

  pr.GameUpdate lobbyState({
    required String tableId,
    required List<pr.Player> players,
  }) {
    return pr.GameUpdate(
      tableId: tableId,
      phase: pr.GamePhase.WAITING,
      players: players,
      communityCards: const [],
      pot: Int64(0),
      currentBet: Int64(0),
      currentPlayer: '',
      minRaise: Int64(20),
      maxRaise: Int64(1000),
      gameStarted: false,
      playersRequired: 2,
      playersJoined: players.length,
      phaseName: 'Waiting',
      timeBankSeconds: 30,
      turnDeadlineUnixMs: Int64(0),
      smallBlind: Int64(10),
      bigBlind: Int64(20),
    );
  }

  testWidgets('showHomeView switches an active table back to browsing tables',
      (tester) async {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.tables = [
      table(id: 'table-live', name: 'Live Table'),
      table(id: 'table-browse', name: 'Browse Target'),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      activeHand(
        tableId: 'table-live',
        currentPlayer: 'hero',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
          player(id: 'villain', name: 'Villain', tableSeat: 1),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.menu_rounded), findsOneWidget);
    expect(find.text('Create Table'), findsNothing);

    model.showHomeView();
    await tester.pumpAndSettle();

    expect(find.text('Create Table'), findsOneWidget);
    expect(find.text('Browse Target'), findsOneWidget);
  });

  testWidgets(
      'eliminated player sees watch instead of open for stale table roster entry',
      (tester) async {
    await tester.binding.setSurfaceSize(const Size(1400, 1200));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.tables = [
      UiTable(
        id: 'table-live',
        name: 'Final Table',
        players: const [
          UiPlayer(
            id: 'hero',
            name: 'Hero',
            balance: 0,
            hand: [],
            currentBet: 0,
            folded: false,
            isTurn: false,
            isAllIn: false,
            isDealer: false,
            isSmallBlind: false,
            isBigBlind: false,
            isReady: false,
            isDisconnected: false,
            handDesc: '',
            tableSeat: 0,
          ),
        ],
        smallBlind: 10,
        bigBlind: 20,
        maxPlayers: 6,
        minPlayers: 2,
        currentPlayers: 2,
        buyInAtoms: 0,
        phase: pr.GamePhase.SHOWDOWN,
        gameStarted: true,
        allReady: true,
      ),
    ];

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Watch'), findsOneWidget);
    expect(find.widgetWithText(ElevatedButton, 'Open'), findsNothing);
  });

  testWidgets(
      'table view keeps only the inline error banner and shows table name',
      (tester) async {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 100000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.errorMessage = 'Set ready failed: escrow required for this table';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('River Room'), findsOneWidget);
    expect(find.text('Table table-li'), findsNothing);
    expect(find.text('Set ready failed: escrow required for this table'),
        findsOneWidget);
  });

  testWidgets('lobby fund step shows one bind escrow action with clearer copy',
      (tester) async {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 100000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Escrow required'), findsOneWidget);
    expect(find.text('Bind Escrow'), findsOneWidget);
    expect(find.text('Bind escrow'), findsNothing);
    expect(find.text('Waiting for confirmations'), findsNothing);
  });

  test('cached escrow state does not leak across table switches', () {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    model.tables = [
      table(id: 'table-old', name: 'Old Table', buyInAtoms: 100000000),
      table(id: 'table-new', name: 'New Table', buyInAtoms: 100000000),
    ];

    model.currentTableId = 'table-old';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-old',
        players: [
          player(
            id: 'hero',
            name: 'Hero',
            tableSeat: 0,
            escrowId: 'escrow-old',
            escrowReady: true,
          ),
        ],
      ),
    );
    expect(model.me?.escrowId, 'escrow-old');
    expect(model.me?.escrowReady, isTrue);

    model.currentTableId = 'table-new';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-new',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    expect(model.me?.escrowId ?? '', isEmpty);
    expect(model.me?.escrowReady ?? false, isFalse);
  });

  testWidgets('success banner can be dismissed manually', (tester) async {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.successMessage = 'Escrow bound to River Room';

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Escrow bound to River Room'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.close));
    await tester.pumpAndSettle();

    expect(find.text('Escrow bound to River Room'), findsNothing);
  });

  testWidgets('game end clears stale success banner', (tester) async {
    final model = PokerModel(playerId: 'hero', dataDir: '/tmp/pokerui-test');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 100000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.successMessage = 'Escrow bound to River Room';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Escrow bound to River Room'), findsOneWidget);

    model.applyNotificationForTest(
      pr.Notification(
        type: pr.NotificationType.GAME_ENDED,
        tableId: 'table-live',
        message: 'Game ended',
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Escrow bound to River Room'), findsNothing);
  });

  testWidgets(
      'bind escrow dialog refreshes when owner escrow confirmations arrive',
      (tester) async {
    final model = _EscrowRefreshTestModel(playerId: 'hero');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.updateAuthedPayoutAddress('DsTestPayoutAddr');
    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 0,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_CONFIRMING',
      },
    ];
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 10000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Bind Escrow'));
    await tester.pumpAndSettle();

    expect(
      find.text(
        'One escrow is still waiting for confirmations and cannot be selected yet.',
      ),
      findsOneWidget,
    );

    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.applyNotificationForTest(
      pr.Notification(
        type: pr.NotificationType.ESCROW_FUNDING,
        playerId: 'hero',
        message: jsonEncode({
          'type': 'escrow_funding',
          'player_id': 'hero',
          'escrow_id': 'escrow-1',
          'funding_state': 'ESCROW_STATE_READY',
        }),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text(
        'One escrow is still waiting for confirmations and cannot be selected yet.',
      ),
      findsNothing,
    );
    expect(find.text('d32bbacf:0 - 0.1000 DCR'), findsOneWidget);
  });

  testWidgets(
      'bind escrow dialog tolerates escrow refresh while dropdown is open',
      (tester) async {
    final model = _EscrowRefreshTestModel(playerId: 'hero');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.updateAuthedPayoutAddress('DsTestPayoutAddr');
    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 0,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_CONFIRMING',
      },
      {
        'escrow_id': 'escrow-2',
        'funding_txid': 'aabbccdd12345678',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 10000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Bind Escrow'));
    await tester.pumpAndSettle();

    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();

    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
      {
        'escrow_id': 'escrow-2',
        'funding_txid': 'aabbccdd12345678',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.applyNotificationForTest(
      pr.Notification(
        type: pr.NotificationType.ESCROW_FUNDING,
        playerId: 'hero',
        message: jsonEncode({
          'type': 'escrow_funding',
          'player_id': 'hero',
          'escrow_id': 'escrow-1',
          'funding_state': 'ESCROW_STATE_READY',
        }),
      ),
    );
    await tester.pump();

    expect(tester.takeException(), isNull);
  });

  testWidgets(
      'bind escrow dialog closes the dropdown when escrow status changes',
      (tester) async {
    final model = _EscrowRefreshTestModel(playerId: 'hero');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.updateAuthedPayoutAddress('DsTestPayoutAddr');
    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 0,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_CONFIRMING',
      },
      {
        'escrow_id': 'escrow-2',
        'funding_txid': 'aabbccdd12345678',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 10000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Bind Escrow'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();

    expect(find.textContaining('Waiting for confirmations'), findsOneWidget);

    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-1',
        'funding_txid': 'd32bbacf12345678',
        'funding_vout': 0,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
      {
        'escrow_id': 'escrow-2',
        'funding_txid': 'aabbccdd12345678',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.applyNotificationForTest(
      pr.Notification(
        type: pr.NotificationType.ESCROW_FUNDING,
        playerId: 'hero',
        message: jsonEncode({
          'type': 'escrow_funding',
          'player_id': 'hero',
          'escrow_id': 'escrow-1',
          'funding_state': 'ESCROW_STATE_READY',
        }),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Waiting for confirmations'), findsNothing);
    expect(find.text('Advanced options'), findsOneWidget);
  });

  testWidgets('bind escrow dialog only shows escrows matching table buy-in',
      (tester) async {
    final model = _EscrowRefreshTestModel(playerId: 'hero');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.updateAuthedPayoutAddress('DsTestPayoutAddr');
    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-small',
        'funding_txid': '1111bbacf1234567',
        'funding_vout': 0,
        'funded_amount': 1000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
      {
        'escrow_id': 'escrow-large',
        'funding_txid': '2222bbacf1234567',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 1000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Bind Escrow'));
    await tester.pumpAndSettle();

    expect(find.text('1111bbac:0 - 0.0100 DCR'), findsOneWidget);
    expect(find.text('2222bbac:1 - 0.1000 DCR'), findsNothing);
  });

  testWidgets(
      'bind escrow action without matching escrows opens the open escrow dialog directly',
      (tester) async {
    final model = _EscrowRefreshTestModel(playerId: 'hero');
    final configNotifier = ConfigNotifier()..updateConfig(Config.empty());
    model.updateAuthedPayoutAddress('DsTestPayoutAddr');
    model.cachedEscrows = [
      {
        'escrow_id': 'escrow-large',
        'funding_txid': '2222bbacf1234567',
        'funding_vout': 1,
        'funded_amount': 10000000,
        'confs': 1,
        'required_confirmations': 1,
        'funding_state': 'ESCROW_STATE_READY',
      },
    ];
    model.tables = [
      table(
        id: 'table-live',
        name: 'River Room',
        buyInAtoms: 1000000,
      ),
    ];
    model.currentTableId = 'table-live';
    model.applyGameUpdateForTest(
      lobbyState(
        tableId: 'table-live',
        players: [
          player(id: 'hero', name: 'Hero', tableSeat: 0),
        ],
      ),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          ChangeNotifierProvider<ConfigNotifier>.value(value: configNotifier),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: MaterialApp(
          theme: buildPokerTheme(),
          home: const PokerHomeScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Bind Escrow'));
    await tester.pumpAndSettle();

    expect(find.text('Open Escrow'), findsNWidgets(2));
    expect(find.text('Table Buy-in (DCR)'), findsOneWidget);
    expect(find.text('Funding outpoint'), findsNothing);
    expect(find.text('Advanced options'), findsNothing);
  });
}
