import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:provider/provider.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/screens/home.dart';
import 'package:pokerui/theme/poker_theme.dart';

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
  }) {
    return pr.Player(
      id: id,
      name: name,
      balance: Int64(1000),
      currentBet: Int64(0),
      isReady: true,
      playerState: pr.PlayerState.PLAYER_STATE_IN_GAME,
      tableSeat: tableSeat,
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
}
