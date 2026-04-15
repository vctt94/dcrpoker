import 'dart:async';

// Convenience function to sleep in async functions.
Future<void> sleep(Duration d) {
  var p = Completer<void>();
  Timer(d, p.complete);
  return p.future;
}

String formatDcrFromAtoms(int atoms) {
  final whole = atoms ~/ 100000000;
  final fractional = atoms % 100000000;
  if (fractional == 0) {
    return '$whole';
  }
  final fractionalText =
      fractional.toString().padLeft(8, '0').replaceFirst(RegExp(r'0+$'), '');
  return '$whole.$fractionalText';
}
