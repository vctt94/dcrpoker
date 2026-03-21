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
