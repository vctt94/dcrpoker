import 'dart:async';
import 'package:flutter/material.dart';

// Lightweight render ticker to repaint at ~60fps without a TickerProvider.
class RenderLoop extends ChangeNotifier {
  Timer? _timer;

  void start() {
    if (_timer != null) return;
    _timer = Timer.periodic(const Duration(milliseconds: 16), (_) {
      notifyListeners();
    });
  }

  void stop() {
    _timer?.cancel();
    _timer = null;
  }
}

