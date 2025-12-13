import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/views/hand_in_progress.dart';

void main() {
  test('Entered amount is treated as total even after posting a blind', () {
    // SB has posted 10; entering 990 means target total is 990.
    const myBet = 10; // posted small blind
    const input = 990; // what player types

    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      0, // currentBet
      myBet,
      20, // bb
    );

    expect(totalBet, equals(990));
  });

  test('No prior bet: entered amount is total', () {
    const myBet = 0;
    const input = 150;
    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      /*currentBet=*/ 0,
      myBet,
      /*bb=*/ 20,
    );

    expect(totalBet, equals(input));
  });

  test('With prior bet: entered amount stays as total', () {
    const myBet = 40;
    const input = 60;
    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      /*currentBet=*/ 100,
      myBet,
      /*bb=*/ 20,
    );

    expect(totalBet, equals(60));
  });
}
