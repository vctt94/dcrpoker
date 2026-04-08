import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/screens/sign_address.dart';
import 'package:pokerui/theme/poker_theme.dart';

void main() {
  testWidgets('sign address screen shows concise settlement verification form',
      (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildPokerTheme(),
        home: const SignAddressScreen(),
      ),
    );
    await tester.pump();

    expect(
      find.text('Verify payout address'),
      findsOneWidget,
    );
    expect(find.text('Settlement Address'), findsOneWidget);
    expect(find.text('Verification Code'), findsOneWidget);
    expect(find.text('Wallet Signature'), findsOneWidget);
    expect(find.text('Request Code'), findsOneWidget);
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
  });
}
