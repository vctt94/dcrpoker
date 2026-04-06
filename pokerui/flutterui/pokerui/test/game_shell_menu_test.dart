import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/poker.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('game shell shows a menu button and opens the poker drawer',
      (tester) async {
    final model = PokerModel(
      playerId: 'player-1234567890abcdef',
      dataDir: '/tmp/pokerui-test',
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<PokerModel>.value(value: model),
          Provider<Future<void> Function()?>.value(value: () async {}),
        ],
        child: const MaterialApp(
          home: GameShell(
            child: SizedBox.expand(),
          ),
        ),
      ),
    );

    expect(find.byIcon(Icons.menu_rounded), findsOneWidget);
    final menuTopLeft = tester.getTopLeft(find.byIcon(Icons.menu_rounded));
    expect(menuTopLeft.dx, lessThan(60));

    await tester.tap(find.byIcon(Icons.menu_rounded));
    await tester.pumpAndSettle();

    expect(find.text('Home'), findsOneWidget);
    expect(find.text('Settings'), findsOneWidget);
    expect(find.text('player-123456789...'), findsOneWidget);
  });
}
