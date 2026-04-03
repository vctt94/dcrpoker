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
}
