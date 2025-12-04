import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/views/hand_in_progress.dart';

void main() {
  test('SB betting displayed balance adds on top of blind', () {
    // SB has already posted 10; UI shows 990 remaining.
    // If the player enters 990, totalBet should be 1000 (10 + 990)
    // and the server delta should equal the displayed balance (990).

    const displayedBalance = 990;
    const myBet = 10; // posted small blind
    const input = 990; // what player types

    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      0, // currentBet
      myBet,
      20, // bb
    );

    expect(totalBet, equals(1000));
    expect(totalBet - myBet, equals(displayedBalance));
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

  test('With prior bet: entered amount is added on top', () {
    const myBet = 40;
    const input = 60;
    final totalBet = HandInProgressView.calculateTotalBet(
      input,
      /*currentBet=*/ 100,
      myBet,
      /*bb=*/ 20,
    );

    expect(totalBet, equals(100)); // 40 already in + 60 entered
  });
}
