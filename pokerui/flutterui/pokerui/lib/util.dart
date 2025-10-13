import 'dart:async';

// Convenience function to sleep in async functions.
Future<void> sleep(Duration d) {
  var p = Completer<void>();
  Timer(d, p.complete);
  return p.future;
}
