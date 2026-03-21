import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/views/hand_in_progress.dart';

void main() {
  test('Entered displayed stack after posting a blind becomes all-in total',
      () {
    // SB has posted 10 and has 990 left behind.
    // If the UI shows 990 as the visible stack and the player enters 990,
    // that should mean "all-in to 1000 total", not "bet to 990 total".
    const myBet = 10; // posted small blind
    const input = 990; // visible remaining stack shown to the player

    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      0, // currentBet
      myBet,
      20, // bb
      myBalance: 990,
    );

    expect(totalBet, equals(1000));
  });

  test('No prior bet: entered amount is total', () {
    const input = 150;
    const myBet = 0;
    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      /*currentBet=*/ 0,
      myBet,
      /*bb=*/ 20,
      myBalance: 1000,
    );

    expect(totalBet, equals(input));
  });

  test('With prior bet: entered amount stays as total', () {
    const input = 60;
    const myBet = 40;
    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      /*currentBet=*/ 100,
      myBet,
      /*bb=*/ 20,
      myBalance: 960,
    );

    expect(totalBet, equals(60));
  });
}
