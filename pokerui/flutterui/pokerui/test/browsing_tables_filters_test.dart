import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/views/browsing_tables.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/poker_theme.dart';

void main() {
  UiTable table({
    required String id,
    required String name,
    required int buyInAtoms,
    int currentPlayers = 2,
    int maxPlayers = 6,
  }) {
    return UiTable(
      id: id,
      name: name,
      players: const [],
      smallBlind: 10,
      bigBlind: 20,
      maxPlayers: maxPlayers,
      minPlayers: 2,
      currentPlayers: currentPlayers,
      buyInAtoms: buyInAtoms,
      phase: pr.GamePhase.WAITING,
      gameStarted: false,
      allReady: false,
    );
  }

  Widget appFor(PokerModel model) {
    return MaterialApp(
      theme: buildPokerTheme(),
      home: Scaffold(
        body: SingleChildScrollView(
          child: BrowsingTablesView(model: model),
        ),
      ),
    );
  }

  testWidgets('browse filters narrow the visible tables by buy-in', (
    tester,
  ) async {
    final model = PokerModel(playerId: 'player-1', dataDir: '/tmp');
    model.tables = [
      table(id: 'free-table', name: 'Starter Table', buyInAtoms: 0),
      table(id: 'micro-table', name: 'Micro Rush', buyInAtoms: 1000000),
      table(id: 'high-table', name: 'High Rollers', buyInAtoms: 125000000),
    ];

    await tester.pumpWidget(appFor(model));

    expect(find.text('Starter Table'), findsOneWidget);
    expect(find.text('Micro Rush'), findsOneWidget);
    expect(find.text('High Rollers'), findsOneWidget);

    await tester.tap(find.text('Free'));
    await tester.pumpAndSettle();

    expect(find.text('Starter Table'), findsOneWidget);
    expect(find.text('Micro Rush'), findsNothing);
    expect(find.text('High Rollers'), findsNothing);

    await tester.tap(find.text('> 0.10 DCR'));
    await tester.pumpAndSettle();

    expect(find.text('Starter Table'), findsNothing);
    expect(find.text('Micro Rush'), findsNothing);
    expect(find.text('High Rollers'), findsOneWidget);

    await tester.tap(find.text('Reset'));
    await tester.pumpAndSettle();

    expect(find.text('Starter Table'), findsOneWidget);
    expect(find.text('Micro Rush'), findsOneWidget);
    expect(find.text('High Rollers'), findsOneWidget);
  });

  testWidgets('browse filters search by table name', (tester) async {
    final model = PokerModel(playerId: 'player-1', dataDir: '/tmp');
    model.tables = [
      table(id: 'river', name: 'River Club', buyInAtoms: 0),
      table(id: 'summit', name: 'Summit Room', buyInAtoms: 1000000),
    ];

    await tester.pumpWidget(appFor(model));

    await tester.enterText(find.byType(TextField), 'river');
    await tester.pumpAndSettle();

    expect(find.text('River Club'), findsOneWidget);
    expect(find.text('Summit Room'), findsNothing);
  });

  testWidgets('browse tables default to most players, then alphabetical', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(430, 1400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    final model = PokerModel(playerId: 'player-1', dataDir: '/tmp');
    model.tables = [
      table(id: 'charlie', name: 'Charlie', buyInAtoms: 0, currentPlayers: 2),
      table(id: 'bravo', name: 'Bravo', buyInAtoms: 0, currentPlayers: 3),
      table(id: 'alpha', name: 'Alpha', buyInAtoms: 0, currentPlayers: 2),
    ];

    await tester.pumpWidget(appFor(model));
    await tester.pumpAndSettle();

    expect(find.textContaining('sorted by most players'), findsOneWidget);

    final bravoY = tester.getTopLeft(find.text('Bravo')).dy;
    final alphaY = tester.getTopLeft(find.text('Alpha')).dy;
    final charlieY = tester.getTopLeft(find.text('Charlie')).dy;

    expect(bravoY, lessThan(alphaY));
    expect(alphaY, lessThan(charlieY));
  });

  testWidgets('browse tables can be sorted by name', (tester) async {
    tester.view.physicalSize = const Size(430, 1400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    final model = PokerModel(playerId: 'player-1', dataDir: '/tmp');
    model.tables = [
      table(id: 'charlie', name: 'Charlie', buyInAtoms: 0, currentPlayers: 4),
      table(id: 'bravo', name: 'Bravo', buyInAtoms: 0, currentPlayers: 2),
      table(id: 'alpha', name: 'Alpha', buyInAtoms: 0, currentPlayers: 3),
    ];

    await tester.pumpWidget(appFor(model));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('browse-sort-dropdown')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Name').last);
    await tester.pumpAndSettle();

    expect(find.textContaining('sorted by name'), findsOneWidget);

    final alphaY = tester.getTopLeft(find.text('Alpha')).dy;
    final bravoY = tester.getTopLeft(find.text('Bravo')).dy;
    final charlieY = tester.getTopLeft(find.text('Charlie')).dy;

    expect(alphaY, lessThan(bravoY));
    expect(bravoY, lessThan(charlieY));
  });

  testWidgets('browse filters narrow the visible tables by player count', (
    tester,
  ) async {
    final model = PokerModel(playerId: 'player-1', dataDir: '/tmp');
    model.tables = [
      table(id: 'duel', name: 'Heads Up', buyInAtoms: 0, maxPlayers: 2),
      table(id: 'ring-4', name: 'Four Max', buyInAtoms: 1000000, maxPlayers: 4),
      table(id: 'ring-6', name: 'Six Max', buyInAtoms: 1000000, maxPlayers: 6),
    ];

    await tester.pumpWidget(appFor(model));

    expect(find.text('Heads Up'), findsOneWidget);
    expect(find.text('Four Max'), findsOneWidget);
    expect(find.text('Six Max'), findsOneWidget);

    await tester.tap(find.text('4').first);
    await tester.pumpAndSettle();

    expect(find.text('Heads Up'), findsNothing);
    expect(find.text('Four Max'), findsOneWidget);
    expect(find.text('Six Max'), findsNothing);

    await tester.tap(find.text('Reset'));
    await tester.pumpAndSettle();

    expect(find.text('Heads Up'), findsOneWidget);
    expect(find.text('Four Max'), findsOneWidget);
    expect(find.text('Six Max'), findsOneWidget);
  });
}
