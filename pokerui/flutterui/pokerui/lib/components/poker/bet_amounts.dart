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

bool isShortAllInTarget({
  required int totalTarget,
  required int myBet,
  required int myBalance,
  required int currentBet,
}) {
  if (currentBet <= 0) return false;
  if (totalTarget >= currentBet) return false;

  final maxTotal = myBet + myBalance;
  return maxTotal > 0 && totalTarget >= maxTotal;
}
