import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/views/table_session_view.dart';

void main() {
  test('Entered displayed stack after posting a blind becomes all-in total',
      () {
    // SB has posted 10 and has 990 left behind.
    // If the UI shows 990 as the visible stack and the player enters 990,
    // that should mean "all-in to 1000 total", not "bet to 990 total".
    const myBet = 10; // posted small blind
    const input = 990; // visible remaining stack shown to the player

    final totalBet = TableSessionView.calculateTotalBet(
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
    final totalBet = TableSessionView.calculateTotalBet(
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
    final totalBet = TableSessionView.calculateTotalBet(
      input,
      /*currentBet=*/ 100,
      myBet,
      /*bb=*/ 20,
      myBalance: 960,
    );

    expect(totalBet, equals(60));
  });

  test('Short all-in target below the call amount is valid', () {
    const myBet = 0;
    const myBalance = 1000;
    const currentBet = 2000;

    final totalBet = TableSessionView.calculateTotalBet(
      myBalance,
      currentBet,
      myBet,
      20,
      myBalance: myBalance,
    );

    expect(totalBet, equals(1000));
    expect(
      isShortAllInTarget(
        totalTarget: totalBet,
        myBet: myBet,
        myBalance: myBalance,
        currentBet: currentBet,
      ),
      isTrue,
    );
  });

  test('Minimum raise target follows live blind size when facing the big blind',
      () {
    final minTotal = legalMinimumBetOrRaiseTotal(
      currentBet: 40,
      minRaise: 40,
      bigBlind: 40,
    );

    expect(minTotal, equals(80));
    expect(
      suggestedBetOrRaiseTotal(
        currentBet: 40,
        minRaise: 40,
        maxRaise: 1000,
        bigBlind: 40,
      ),
      equals(120),
    );
    expect(
      initialBetOrRaiseTotal(
        currentBet: 40,
        minRaise: 40,
        maxRaise: 1000,
        bigBlind: 40,
      ),
      equals(80),
    );
    expect(
      raiseThreeXTotal(
        currentBet: 40,
        minRaise: 40,
        maxRaise: 1000,
        bigBlind: 40,
      ),
      equals(120),
    );
  });

  test('Recommended target falls back to big blind when minRaise is missing',
      () {
    expect(
      legalMinimumBetOrRaiseTotal(
        currentBet: 40,
        minRaise: 0,
        bigBlind: 40,
      ),
      equals(80),
    );
  });

  test('Raise validation rejects totals between call and legal minimum raise',
      () {
    expect(
      validateBetOrRaiseTarget(
        totalTarget: 60,
        currentBet: 40,
        myBet: 0,
        myBalance: 1000,
        minRaise: 40,
        bigBlind: 40,
      ),
      equals('Minimum raise to 80'),
    );
  });

  test('Short all-in under the minimum raise stays valid after posting chips',
      () {
    const myBet = 100;
    const myBalance = 105;
    const currentBet = 200;

    final totalBet = TableSessionView.calculateTotalBet(
      myBalance,
      currentBet,
      myBet,
      100,
      myBalance: myBalance,
    );

    expect(totalBet, equals(205));
    expect(
      validateBetOrRaiseTarget(
        totalTarget: totalBet,
        currentBet: currentBet,
        myBet: myBet,
        myBalance: myBalance,
        minRaise: 200,
        bigBlind: 100,
      ),
      isNull,
    );
  });

  test('Exact all-in call target is valid in raise mode', () {
    expect(
      validateBetOrRaiseTarget(
        totalTarget: 200,
        currentBet: 200,
        myBet: 100,
        myBalance: 100,
        minRaise: 200,
        bigBlind: 100,
      ),
      isNull,
    );
  });

  test('Opening all-in below the minimum bet stays valid', () {
    expect(
      validateBetOrRaiseTarget(
        totalTarget: 15,
        currentBet: 0,
        myBet: 0,
        myBalance: 15,
        minRaise: 0,
        bigBlind: 20,
      ),
      isNull,
    );
  });

  test('Suggested target falls back to short all-in max when below legal min',
      () {
    expect(
      hasShortAllInOnlyBetOrRaiseOption(
        currentBet: 100,
        minRaise: 100,
        maxRaise: 150,
        bigBlind: 50,
      ),
      isTrue,
    );
    expect(
      suggestedBetOrRaiseTotal(
        currentBet: 100,
        minRaise: 100,
        maxRaise: 150,
        bigBlind: 50,
      ),
      equals(150),
    );
    expect(
      initialBetOrRaiseTotal(
        currentBet: 100,
        minRaise: 100,
        maxRaise: 150,
        bigBlind: 50,
      ),
      equals(150),
    );
  });

  test('Slider targets snap to betting step from the current bet anchor', () {
    expect(
      snapBetTargetToStep(
        target: 95,
        currentBet: 40,
        minRaise: 40,
        maxRaise: 400,
        bigBlind: 40,
      ),
      equals(80),
    );
    expect(
      snapBetTargetToStep(
        target: 119,
        currentBet: 40,
        minRaise: 40,
        maxRaise: 400,
        bigBlind: 40,
      ),
      equals(120),
    );
  });

  test('Slider can reach short all-in when no full raise step remains', () {
    expect(
      snapBetTargetToStep(
        target: 220,
        currentBet: 100,
        minRaise: 100,
        maxRaise: 230,
        bigBlind: 100,
      ),
      equals(230),
    );
    expect(
      snapBetTargetToStep(
        target: 205,
        currentBet: 100,
        minRaise: 100,
        maxRaise: 230,
        bigBlind: 100,
      ),
      equals(200),
    );
  });
}
