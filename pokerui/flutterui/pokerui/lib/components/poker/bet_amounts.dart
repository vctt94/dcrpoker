import 'dart:math' as math;

int normalizeBetInputToTotal({
  required int entered,
  required int myBet,
  required int myBalance,
}) {
  if (entered <= 0) return entered;

  final maxTotal = myBet + myBalance;
  if (maxTotal <= 0) return entered;

  if (entered >= maxTotal) {
    return maxTotal;
  }

  // If the player already has chips committed this street, allow entering the
  // visible remaining stack as an all-in shorthand.
  if (myBet > 0 && entered == myBalance) {
    return maxTotal;
  }

  return entered;
}

bool isAllInTarget({
  required int totalTarget,
  required int myBet,
  required int myBalance,
}) {
  final maxTotal = myBet + myBalance;
  return maxTotal > 0 && totalTarget >= maxTotal;
}

bool isShortAllInTarget({
  required int totalTarget,
  required int myBet,
  required int myBalance,
  required int currentBet,
}) {
  if (currentBet <= 0) return false;
  if (totalTarget >= currentBet) return false;

  return isAllInTarget(
    totalTarget: totalTarget,
    myBet: myBet,
    myBalance: myBalance,
  );
}

int effectiveMinRaiseIncrement({
  required int currentBet,
  required int minRaise,
  required int bigBlind,
}) {
  if (minRaise > 0) return minRaise;
  if (bigBlind > 0) return bigBlind;
  return currentBet > 0 ? currentBet : 0;
}

int legalMinimumBetOrRaiseTotal({
  required int currentBet,
  required int minRaise,
  required int bigBlind,
}) {
  final minIncrement = effectiveMinRaiseIncrement(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  if (currentBet > 0) return currentBet + minIncrement;
  return minIncrement;
}

bool hasShortAllInOnlyBetOrRaiseOption({
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  if (maxRaise <= 0) return false;
  final legalMin = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  return legalMin > 0 && maxRaise < legalMin;
}

int minimumBetOrRaiseTotal({
  required int currentBet,
  required int minRaise,
  required int bigBlind,
}) {
  return legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
}

int initialBetOrRaiseTotal({
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  final minTotal = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  if (minTotal <= 0) return 0;
  if (hasShortAllInOnlyBetOrRaiseOption(
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  )) {
    return maxRaise;
  }
  return minTotal;
}

int clampBetTargetToLegalRange({
  required int target,
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  if (target <= 0) return target;

  final minTotal = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  final capped = maxRaise > 0 ? math.min(target, maxRaise) : target;
  if (hasShortAllInOnlyBetOrRaiseOption(
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  )) {
    return maxRaise;
  }
  return math.max(capped, minTotal);
}

int betSizingStep({
  required int currentBet,
  required int minRaise,
  required int bigBlind,
}) {
  final step = effectiveMinRaiseIncrement(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  return step > 0 ? step : 1;
}

int snapBetTargetToStep({
  required int target,
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  final clamped = clampBetTargetToLegalRange(
    target: target,
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  );
  final legalMin = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  if (hasShortAllInOnlyBetOrRaiseOption(
        currentBet: currentBet,
        minRaise: minRaise,
        maxRaise: maxRaise,
        bigBlind: bigBlind,
      ) ||
      (maxRaise > 0 && maxRaise <= legalMin)) {
    return maxRaise;
  }

  final step = betSizingStep(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  if (maxRaise > legalMin && (maxRaise - legalMin) < step) {
    final midpoint = legalMin + ((maxRaise - legalMin) / 2);
    return clamped >= midpoint ? maxRaise : legalMin;
  }

  final anchor = currentBet > 0 ? currentBet : 0;
  final snappedSteps = ((clamped - anchor) / step).round();
  final snapped = anchor + (snappedSteps * step);
  return clampBetTargetToLegalRange(
    target: snapped,
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  );
}

int suggestedBetOrRaiseTotal({
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  final minTotal = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  final preferred = currentBet > 0
      ? currentBet * 3
      : (bigBlind > 0 ? bigBlind * 3 : minTotal);
  return clampBetTargetToLegalRange(
    target: preferred > 0 ? preferred : minTotal,
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  );
}

int raiseThreeXTotal({
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  if (currentBet <= 0) return 0;
  return clampBetTargetToLegalRange(
    target: currentBet * 3,
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  );
}

int recommendedBetOrRaiseTotal({
  required int currentBet,
  required int minRaise,
  required int maxRaise,
  required int bigBlind,
}) {
  return suggestedBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    maxRaise: maxRaise,
    bigBlind: bigBlind,
  );
}

String? validateBetOrRaiseTarget({
  required int totalTarget,
  required int currentBet,
  required int myBet,
  required int myBalance,
  required int minRaise,
  required int bigBlind,
}) {
  final allIn = isAllInTarget(
    totalTarget: totalTarget,
    myBet: myBet,
    myBalance: myBalance,
  );

  if (currentBet > 0 && totalTarget < currentBet && !allIn) {
    return 'Must call $currentBet total or go all-in for less';
  }

  final minTotal = legalMinimumBetOrRaiseTotal(
    currentBet: currentBet,
    minRaise: minRaise,
    bigBlind: bigBlind,
  );
  if (minTotal <= 0 || allIn) return null;

  if (currentBet > 0 && totalTarget == currentBet) {
    return 'Use Call to match $currentBet. Minimum raise to $minTotal';
  }

  if (totalTarget < minTotal) {
    return currentBet > 0
        ? 'Minimum raise to $minTotal'
        : 'Minimum bet is $minTotal';
  }

  return null;
}
