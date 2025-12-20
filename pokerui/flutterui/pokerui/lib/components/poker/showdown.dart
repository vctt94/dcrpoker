import 'dart:async';
import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/poker/minimal_showdown.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class ShowdownView extends StatefulWidget {
  const ShowdownView({super.key, required this.model});
  final PokerModel model;

  @override
  State<ShowdownView> createState() => _ShowdownViewState();
}

class _ShowdownViewState extends State<ShowdownView> {
  Timer? _countdownTimer;
  Timer? _autoCloseTimer;
  int _secondsRemaining = 5;
  bool _showSidebar = true;

  @override
  void initState() {
    super.initState();
    _startCountdown();
    // Auto-close sidebar after 5 seconds
    _autoCloseTimer = Timer(const Duration(seconds: 5), () {
      if (mounted) {
        _closeSidebar();
      }
    });
  }

  @override
  void dispose() {
    _countdownTimer?.cancel();
    _autoCloseTimer?.cancel();
    super.dispose();
  }

  void _closeSidebar() {
    if (mounted) {
      setState(() {
        _showSidebar = false;
      });
    }
  }

  void _startCountdown() {
    _secondsRemaining = 5;
    _countdownTimer?.cancel();
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (mounted) {
        setState(() {
          _secondsRemaining--;
          if (_secondsRemaining <= 0) {
            timer.cancel();
          }
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;
    final game = model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    // Auto-close sidebar if game phase is no longer SHOWDOWN
    if (game.phase != pr.GamePhase.SHOWDOWN && _showSidebar) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _closeSidebar();
        }
      });
    }

    final focusNode = FocusNode();
    final pokerGame = PokerGame(
      model.playerId,
      model,
      theme: PokerThemeConfig.fromContext(context),
    );
    final winners = model.lastWinners;

    return SizedBox.expand(
      child: Stack(
        fit: StackFit.expand,
        children: [
          // Poker game visualization (table + canvas elements)
          pokerGame.buildWidget(game, focusNode),

          // Showdown FX overlay: chip flow to winners
          _ShowdownFxOverlay(model: model),

          // Minimal showdown widget - positioned on the right
          if (winners.isNotEmpty && _showSidebar)
            MinimalShowdown(
              model: model,
              isVisible: _showSidebar,
              theme: PokerThemeConfig.fromContext(context),
              onClose: _closeSidebar,
            ),

          // Countdown and Skip button at bottom center (only if game end is pending)
          if (model.isGameEndPending)
            Positioned(
              bottom: 16,
              left: 0,
              right: 0,
              child: SafeArea(
                child: Center(
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16, vertical: 10),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.8),
                      borderRadius: BorderRadius.circular(20),
                      border: Border.all(color: Colors.white24),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        // Countdown indicator
                        Container(
                          width: 32,
                          height: 32,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            border: Border.all(color: Colors.amber, width: 2),
                          ),
                          child: Center(
                            child: Text(
                              '$_secondsRemaining',
                              style: const TextStyle(
                                color: Colors.amber,
                                fontSize: 16,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 12),
                        // Skip button
                        ElevatedButton.icon(
                          onPressed: () {
                            _closeSidebar();
                            model.skipShowdown();
                          },
                          icon: const Icon(Icons.skip_next, size: 18),
                          label: const Text('Continue'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.blue.shade700,
                            foregroundColor: Colors.white,
                            padding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 8,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _ShowdownFxOverlay extends StatefulWidget {
  const _ShowdownFxOverlay({required this.model});
  final PokerModel model;

  @override
  State<_ShowdownFxOverlay> createState() => _ShowdownFxOverlayState();
}

class _ShowdownFxOverlayState extends State<_ShowdownFxOverlay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _chipCtrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _chipCtrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 900));
    _maybeRestartFx();
  }

  @override
  void didUpdateWidget(covariant _ShowdownFxOverlay oldWidget) {
    super.didUpdateWidget(oldWidget);
    _maybeRestartFx();
  }

  @override
  void dispose() {
    _chipCtrl.dispose();
    super.dispose();
  }

  void _maybeRestartFx() {
    final winners = widget.model.lastWinners;
    final fxMs = widget.model.lastShowdownFxMs;
    // ignore: avoid_print
    if (winners.isEmpty || fxMs == 0) return;
    if (fxMs != _lastFxMs) {
      // Debug visibility to trace animation triggers
      // ignore: avoid_print
      _lastFxMs = fxMs;
      _chipCtrl
        ..reset()
        ..forward();
    }
  }

  @override
  Widget build(BuildContext context) {
    final game = widget.model.game;
    if (game == null) return const SizedBox.shrink();
    final winners = widget.model.lastWinners;

    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final theme = PokerThemeConfig.fromContext(context);
      final layout = resolveTableLayout(size);
      final center = layout.center;
      final hasCurrentBet = game.currentBet > 0;
      final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);

      final chipWidgets = <Widget>[];
      if (winners.isNotEmpty && game.players.isNotEmpty) {
        final targets = seatPositionsFor(
          game.players,
          widget.model.playerId,
          center,
          layout.ringRadiusX,
          layout.ringRadiusY,
          clampBounds: layout.viewport,
          minSeatTop: minSeatTop,
          uiSizeMultiplier: theme.uiSizeMultiplier,
        );
        final potOrigin =
            potChipCenter(layout, uiSizeMultiplier: theme.uiSizeMultiplier);

        for (int i = 0; i < winners.length; i++) {
          final w = winners[i];
          final target = targets[w.playerId] ?? center;
          final curved = CurvedAnimation(
              parent: _chipCtrl,
              curve: const Interval(0.0, 1.0, curve: Curves.easeOut));
          final t = Tween<double>(begin: 0, end: 1).animate(curved);
          for (int j = 0; j < 3; j++) {
            final delay = j * 0.06 + i * 0.12;
            chipWidgets.add(_AnimatedChip(
              t: t,
              delay: delay,
              from: potOrigin,
              to: target,
            ));
          }
        }
      }

      return Stack(children: [
        ...chipWidgets,
      ]);
    });
  }
}

class _AnimatedChip extends StatelessWidget {
  const _AnimatedChip(
      {required this.t,
      required this.delay,
      required this.from,
      required this.to});
  final Animation<double> t;
  final double delay;
  final Offset from;
  final Offset to;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: t,
      builder: (context, child) {
        final span = 1.0 - delay;
        if (span <= 0) return const SizedBox.shrink();
        final raw = (t.value - delay) / span;
        if (raw <= 0.0 || raw >= 1.0) {
          return const SizedBox.shrink();
        }
        final eased = Curves.easeOut.transform(raw.clamp(0.0, 1.0));
        final dx = from.dx + (to.dx - from.dx) * eased;
        final dy = from.dy + (to.dy - from.dy) * eased;
        return Positioned(
          left: dx - 6,
          top: dy - 6,
          child: Container(
            width: 12,
            height: 12,
            decoration: BoxDecoration(
              color: Colors.amber,
              border: Border.all(color: Colors.orange.shade900, width: 1.5),
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                    color: Colors.black.withOpacity(0.3),
                    blurRadius: 4,
                    spreadRadius: 0.5),
              ],
            ),
          ),
        );
      },
    );
  }
}
