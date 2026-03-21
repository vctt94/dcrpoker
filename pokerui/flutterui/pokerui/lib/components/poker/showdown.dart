import 'dart:async';
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/minimal_showdown.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';

class ShowdownView extends StatefulWidget {
  final PokerModel model;
  const ShowdownView({super.key, required this.model});

  @override
  State<ShowdownView> createState() => _ShowdownViewState();
}

class _ShowdownViewState extends State<ShowdownView> {
  Timer? _autoCloseTimer;
  bool _showSidebar = true;
  int _lastShowdownFxMs = 0;

  @override
  void initState() {
    super.initState();
  }

  @override
  void dispose() {
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

  @override
  Widget build(BuildContext context) {
    final model = widget.model;
    final game = model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    // Re-open and re-arm auto-close when a new showdown event arrives.
    final fxMs = model.lastShowdownFxMs;
    if (fxMs != 0 && fxMs != _lastShowdownFxMs) {
      _lastShowdownFxMs = fxMs;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        setState(() {
          _showSidebar = true;
        });
        _autoCloseTimer?.cancel();
        _autoCloseTimer = Timer(const Duration(seconds: 5), _closeSidebar);
      });
    }

    // Auto-close sidebar if game phase is no longer SHOWDOWN
    if (game.phase != pr.GamePhase.SHOWDOWN && _showSidebar) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _closeSidebar();
        }
      });
    }

    final theme = PokerThemeConfig.fromContext(context);
    final bp = PokerBreakpointQuery.of(context);
    final isMobile = bp.isNarrow;
    final pokerGame = PokerGame(model.playerId, model, theme: theme);
    final winners = model.lastWinners;
    final tableAr = tableAspectRatio(bp);

    final tableStack = Stack(
      fit: StackFit.expand,
      children: [
        pokerGame.buildWidget(
          game,
          FocusNode(),
          aspectRatio: tableAr,
          showHeroCardsOverlay: !isMobile,
        ),
        _ShowdownFxOverlay(model: model),
        if (winners.isNotEmpty && _showSidebar)
          MinimalShowdown(
            model: model,
            isVisible: _showSidebar,
            theme: theme,
            onClose: _closeSidebar,
          ),
      ],
    );

    final Widget? showdownFooter = model.isGameEndPending
        ? Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text(
                  'Game ended. Press Continue.',
                  style: TextStyle(
                    color: Colors.white70,
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 10),
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
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                  ),
                ),
              ],
            ),
          )
        : null;

    if (isMobile) {
      return Column(
        children: [
          Expanded(child: tableStack),
          MobileHeroActionPanel(
            model: model,
            showActions: false,
            reserveActionSpace: true,
            footer: showdownFooter,
          ),
        ],
      );
    }

    return Stack(
      fit: StackFit.expand,
      children: [
        tableStack,
        if (showdownFooter != null)
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: Container(
              constraints: BoxConstraints(minHeight: actionDockMinHeight(bp)),
              padding: EdgeInsets.only(
                left: 16,
                right: 16,
                top: 10,
                bottom: safeBottomPadding(context, minPadding: 10),
              ),
              color: const Color(0xFF121212),
              child: showdownFooter,
            ),
          ),
      ],
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

  void _maybeRestartFx() {
    final winners = widget.model.lastWinners;
    final fxMs = widget.model.lastShowdownFxMs;
    if (winners.isEmpty || fxMs == 0) return;
    if (fxMs != _lastFxMs) {
      _lastFxMs = fxMs;
      _chipCtrl
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _chipCtrl.dispose();
    super.dispose();
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
          clampBounds: layout.canvasBounds,
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
