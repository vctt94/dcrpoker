import 'dart:async';
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart';
import 'package:pokerui/grpc/grpc_client.dart';

// A compact snapshot with timestamp for interpolation.
class _Snapshot {
  final GameUpdate s;
  final int tMicros; // monotonic-ish timestamp (DateTime.now().microsecondsSinceEpoch)
  _Snapshot(this.s, this.tMicros);
}

// Interpolates positions between server snapshots, rendering slightly in the past.
class SnapshotInterpolator {
  // ~80-120ms feels good; adjust to your network jitter.
  Duration interpolationDelay = const Duration(milliseconds: 100);

  // Keep only the two most recent snapshots.
  _Snapshot? _prev;
  _Snapshot? _curr;

  // Push the newest server state.
  void push(GameUpdate s) {
    final now = DateTime.now().microsecondsSinceEpoch;
    // Clone minimal fields used for render to avoid mutation surprises.
    final snap = GameUpdate()
      ..gameWidth = s.gameWidth
      ..gameHeight = s.gameHeight
      ..p1X = s.p1X ..p1Y = s.p1Y ..p1Width = s.p1Width ..p1Height = s.p1Height
      ..p2X = s.p2X ..p2Y = s.p2Y ..p2Width = s.p2Width ..p2Height = s.p2Height
      ..ballX = s.ballX ..ballY = s.ballY ..ballWidth = s.ballWidth ..ballHeight = s.ballHeight
      ..p1Score = s.p1Score ..p2Score = s.p2Score;

    _prev = _curr;
    _curr = _Snapshot(snap, now);
  }

  // Sample an interpolated state "renderTime" in the past.
  GameUpdate sample() {
    final current = _curr;
    if (current == null) {
      return GameUpdate()
        ..gameWidth = 800
        ..gameHeight = 600;
    }

    final previous = _prev;
    if (previous == null) return current.s;

    final now = DateTime.now().microsecondsSinceEpoch;
    final target = now - interpolationDelay.inMicroseconds;

    // Clamp to available range (no extrapolation beyond prev/curr).
    if (target <= previous.tMicros) return previous.s;
    if (target >= current.tMicros) return current.s;

    final dt = (current.tMicros - previous.tMicros).toDouble();
    final u = dt <= 0 ? 0.0 : ((target - previous.tMicros) / dt).clamp(0.0, 1.0);

    // Linear interpolate fields that matter visually.
    double lerp(double x, double y) => x + (y - x) * u;

    final out = GameUpdate()
      ..gameWidth = current.s.gameWidth
      ..gameHeight = current.s.gameHeight
      ..p1X = lerp(previous.s.p1X, current.s.p1X)
      ..p1Y = lerp(previous.s.p1Y, current.s.p1Y)
      ..p1Width = lerp(previous.s.p1Width, current.s.p1Width)
      ..p1Height = lerp(previous.s.p1Height, current.s.p1Height)
      ..p2X = lerp(previous.s.p2X, current.s.p2X)
      ..p2Y = lerp(previous.s.p2Y, current.s.p2Y)
      ..p2Width = lerp(previous.s.p2Width, current.s.p2Width)
      ..p2Height = lerp(previous.s.p2Height, current.s.p2Height)
      ..ballX = lerp(previous.s.ballX, current.s.ballX)
      ..ballY = lerp(previous.s.ballY, current.s.ballY)
      ..ballWidth = lerp(previous.s.ballWidth, current.s.ballWidth)
      ..ballHeight = lerp(previous.s.ballHeight, current.s.ballHeight)
      ..p1Score = current.s.p1Score // scores can snap; no need to lerp ints
      ..p2Score = current.s.p2Score;

    return out;
  }
}

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

class InputThrottler {
  String? _active; // 'ArrowUp' | 'ArrowDown' | null

  Future<void> update(GrpcPongClient c, String clientId, double dy) async {
    final want = dy < 0 ? 'ArrowUp' : 'ArrowDown';
    if (want != _active) {
      // stop previous if any
      if (_active != null) {
        await c.sendInput(clientId, _active == 'ArrowUp' ? 'ArrowUpStop' : 'ArrowDownStop');
      }
      _active = want;
      await c.sendInput(clientId, want);
    }
  }

  Future<void> stop(GrpcPongClient c, String clientId) async {
    if (_active != null) {
      await c.sendInput(clientId, _active == 'ArrowUp' ? 'ArrowUpStop' : 'ArrowDownStop');
      _active = null;
    }
  }
}
