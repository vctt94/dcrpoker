import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/screens/open_escrow.dart';
import 'package:pokerui/theme/poker_theme.dart';

void main() {
  testWidgets('open escrow screen shows concise funding form', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: const OpenEscrowScreen(),
      ),
    );
    await tester.pump();

    expect(find.text('Fund escrow'), findsOneWidget);
    expect(find.text('Bet Amount (DCR)'), findsOneWidget);
    expect(find.text('CSV Blocks'), findsOneWidget);
    expect(find.widgetWithText(ElevatedButton, 'Open Escrow'), findsOneWidget);
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
  });

  testWidgets('session private key stays hidden until revealed',
      (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: const OpenEscrowScreen(),
      ),
    );
    await tester.pump();

    final dynamic state = tester.state(find.byType(OpenEscrowScreen));
    state.debugSetSessionKeyForTest(
      publicKey: 'public-key',
      privateKey: 'secret-private-key',
      keyIndex: '5',
    );
    await tester.pump();

    expect(find.text('Session Key'), findsOneWidget);
    expect(find.text('secret-private-key'), findsNothing);
    expect(find.text('Show'), findsOneWidget);

    await tester.ensureVisible(find.text('Show'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Show'));
    await tester.pumpAndSettle();

    expect(find.text('secret-private-key'), findsOneWidget);
    expect(find.text('Hide'), findsOneWidget);
  });

  testWidgets('escrow hex fields stay hidden until revealed', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: const OpenEscrowScreen(),
      ),
    );
    await tester.pump();

    final dynamic state = tester.state(find.byType(OpenEscrowScreen));
    state.debugSetEscrowResultForTest({
      'deposit_address': 'DsExampleAddress',
      'pk_script_hex': 'abcdef1234567890',
      'redeem_script_hex': 'fedcba0987654321',
    });
    await tester.pump();

    expect(find.text('abcdef1234567890'), findsNothing);
    expect(find.text('fedcba0987654321'), findsNothing);
    expect(find.text('Show'), findsNWidgets(2));

    await tester.ensureVisible(find.text('Show').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Show').first);
    await tester.pumpAndSettle();

    expect(find.text('abcdef1234567890'), findsOneWidget);
  });
}
