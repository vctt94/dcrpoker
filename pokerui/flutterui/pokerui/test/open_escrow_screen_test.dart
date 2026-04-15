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
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller?.text,
      isEmpty,
    );
  });

  testWidgets('table escrow dialog locks amount to table buy-in',
      (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: const Scaffold(
          body: OpenEscrowDialog(
            tableName: 'Micro Stakes',
            buyInAtoms: 1000000,
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Open Escrow'), findsNWidgets(2));
    expect(find.text('Fund for Micro Stakes'), findsOneWidget);
    expect(find.text('Table Buy-in (DCR)'), findsOneWidget);

    final amountField = tester.widget<TextField>(find.byType(TextField).first);
    expect(amountField.readOnly, isTrue);
    expect(amountField.controller?.text, '0.01');
  });

  testWidgets('table escrow dialog keeps funding address visible until closed',
      (tester) async {
    final contentKey = GlobalKey();

    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: Scaffold(
          body: Builder(
            builder: (context) => Center(
              child: ElevatedButton(
                onPressed: () {
                  showDialog<void>(
                    context: context,
                    builder: (_) => OpenEscrowDialog(
                      tableName: 'Micro Stakes',
                      buyInAtoms: 1000000,
                      contentKey: contentKey,
                    ),
                  );
                },
                child: const Text('Launch'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Launch'));
    await tester.pumpAndSettle();

    (contentKey.currentState as dynamic).debugSetEscrowResultForTest({
      'deposit_address': 'DsExampleAddress',
      'pk_script_hex': 'abcdef1234567890',
    });
    await tester.pumpAndSettle();

    expect(find.text('Fund This Escrow'), findsOneWidget);
    expect(find.text('Deposit Address'), findsOneWidget);
    expect(find.text('DsExampleAddress'), findsOneWidget);
    expect(find.text('Close'), findsOneWidget);
    expect(find.text('Table Buy-in (DCR)'), findsNothing);
    expect(find.text('Escrow Created'), findsNothing);
    expect(find.text('Session Key'), findsNothing);
    expect(find.text('Show advanced details'), findsOneWidget);

    await tester.ensureVisible(find.widgetWithText(ElevatedButton, 'Close'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ElevatedButton, 'Close'));
    await tester.pumpAndSettle();

    expect(find.text('Fund This Escrow'), findsNothing);
    expect(find.text('DsExampleAddress'), findsNothing);
  });

  testWidgets('session private key stays hidden until revealed',
      (tester) async {
    final contentKey = GlobalKey();

    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: OpenEscrowScreen(contentKey: contentKey),
      ),
    );
    await tester.pump();

    (contentKey.currentState as dynamic).debugSetSessionKeyForTest(
      publicKey: 'public-key',
      privateKey: 'secret-private-key',
      keyIndex: '5',
    );
    await tester.pump();

    expect(find.text('Session Key'), findsNothing);
    expect(find.text('Show advanced details'), findsOneWidget);
    await tester.ensureVisible(find.text('Show advanced details'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Show advanced details'));
    await tester.pumpAndSettle();

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
    final contentKey = GlobalKey();

    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: OpenEscrowScreen(contentKey: contentKey),
      ),
    );
    await tester.pump();

    (contentKey.currentState as dynamic).debugSetEscrowResultForTest({
      'deposit_address': 'DsExampleAddress',
      'pk_script_hex': 'abcdef1234567890',
      'redeem_script_hex': 'fedcba0987654321',
    });
    await tester.pump();

    expect(find.text('Escrow Created'), findsNothing);
    expect(find.text('Show advanced details'), findsOneWidget);
    await tester.ensureVisible(find.text('Show advanced details'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Show advanced details'));
    await tester.pumpAndSettle();

    expect(find.text('Escrow Created'), findsOneWidget);
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
