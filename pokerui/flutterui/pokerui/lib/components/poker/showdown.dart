import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/pot_display.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';

class ShowdownView extends StatefulWidget {
  final PokerModel model;
  const ShowdownView({super.key, required this.model});

  @override
  State<ShowdownView> createState() => _ShowdownViewState();
}

class _ShowdownViewState extends State<ShowdownView> {
  bool _showSidebar = false;

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

    final theme = PokerThemeConfig.fromContext(context);
    final pokerGame = PokerGame(model.playerId, model, theme: theme);
    final tableAr = 1.3;

    return LayoutBuilder(
      builder: (context, constraints) {
        final scene = PokerSceneLayout.resolve(
          constraints.biggest,
          safePadding: MediaQuery.paddingOf(context),
        );
        final useMobileDock = scene.mode == PokerLayoutMode.compactPortrait;
        final showTableHeroCards = !useMobileDock;
        final sidebarInset = 0.0;
        final toggleInset = 4.0;
        final sidebarWidth =
            (scene.contentRect.width * 0.28).clamp(248.0, 320.0);
        final sidebarLeft = (scene.contentRect.left + sidebarInset)
            .clamp(0.0, scene.contentRect.right);
        final sidebarTop = scene.contentRect.top + sidebarInset;
        final sidebarBottom = scene.heroDockRect.top - sidebarInset;
        final sidebarRect = Rect.fromLTRB(
          sidebarLeft,
          sidebarTop,
          (sidebarLeft + sidebarWidth).clamp(
            sidebarLeft,
            scene.contentRect.right - sidebarInset,
          ),
          sidebarBottom > sidebarTop ? sidebarBottom : sidebarTop + 1,
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
        return Stack(
          fit: StackFit.expand,
          children: [
            pokerGame.buildWidget(
              game,
              FocusNode(),
              aspectRatio: tableAr,
              showHeroSeatCards: showTableHeroCards,
            ),
            _ShowdownFxOverlay(model: model),
            if (_showSidebar)
              Positioned.fromRect(
                rect: sidebarRect,
                child: ShowdownSidebar(
                  model: model,
                  visible: true,
                  onClose: _closeSidebar,
                ),
              ),
            if (model.hasLastShowdown && !_showSidebar)
              Positioned(
                left: scene.contentRect.left + toggleInset,
                top: scene.contentRect.top + toggleInset,
                child: PokerLastHandButton(
                  active: _showSidebar,
                  onTap: () => setState(() => _showSidebar = !_showSidebar),
                ),
              ),
            Positioned.fromRect(
              rect: scene.heroDockRect,
              child: Container(
                key: const Key('poker-hero-dock'),
                child: useMobileDock
                    ? MobileHeroActionPanel.passive(
                        model: model,
                        reserveActionSpace: false,
                        footer: showdownFooter,
                      )
                    : BottomActionDock.passive(
                        model: model,
                        reserveActionSpace: false,
                        footer: showdownFooter,
                      ),
              ),
            ),
          ],
        );
      },
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
  late final AnimationController _payoutCtrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _payoutCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1150),
    );
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
      _payoutCtrl
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _payoutCtrl.dispose();
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
      final scene = layout.scene;
      final center = layout.center;
      final hasCurrentBet = game.currentBet > 0;
      final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);

      final payoutWidgets = <Widget>[];
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
          sceneLayout: scene,
        );
        final potOrigin = potStackAnchor(layout, theme);
        final seatRadius = kPlayerRadius * theme.uiSizeMultiplier;
        final originSpread =
            20.0 * theme.uiSizeMultiplier * (winners.length > 1 ? 1.0 : 0.0);

        for (int i = 0; i < winners.length; i++) {
          final w = winners[i];
          final targetTop = targets[w.playerId] ?? center;
          final target = Offset(targetTop.dx, targetTop.dy + seatRadius * 0.95);
          final startXOffset =
              (i - ((winners.length - 1) / 2)) * originSpread.clamp(0, 28);
          payoutWidgets.add(_AnimatedPotFlight(
            key: ValueKey('showdown-payout-flight-$i'),
            animation: _payoutCtrl,
            amount: w.winnings,
            from: potOrigin.translate(startXOffset, 0),
            to: target,
            theme: theme,
            paletteIndex: i,
            delay: i * 0.11,
          ));
        }
      }

      return IgnorePointer(
        child: Stack(children: payoutWidgets),
      );
    });
  }
}

class _AnimatedPotFlight extends StatelessWidget {
  const _AnimatedPotFlight({
    super.key,
    required this.animation,
    required this.amount,
    required this.from,
    required this.to,
    required this.theme,
    required this.paletteIndex,
    required this.delay,
  });

  final Animation<double> animation;
  final int amount;
  final double delay;
  final Offset from;
  final Offset to;
  final PokerThemeConfig theme;
  final int paletteIndex;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: animation,
      builder: (context, child) {
        final span = 1.0 - delay;
        if (span <= 0) return const SizedBox.shrink();
        final raw = (animation.value - delay) / span;
        if (raw <= 0.0 || raw >= 1.0) {
          return const SizedBox.shrink();
        }
        final progress = raw.clamp(0.0, 1.0);
        final eased = Curves.easeOutCubic.transform(progress);
        final dx = from.dx + (to.dx - from.dx) * eased;
        final arcHeight = 26.0 * theme.uiSizeMultiplier;
        final dy = from.dy +
            (to.dy - from.dy) * eased -
            (1 - ((progress * 2) - 1).abs()) * arcHeight;
        final scale = Tween<double>(
          begin: 1.0,
          end: 0.92,
        ).transform(Curves.easeOut.transform(progress));
        final opacity = progress > 0.84
            ? (1 - ((progress - 0.84) / 0.16)).clamp(0.0, 1.0)
            : 1.0;

        return Positioned(
          left: dx,
          top: dy,
          child: FractionalTranslation(
            translation: const Offset(-0.5, -0.32),
            child: Opacity(
              opacity: opacity,
              child: Transform.scale(
                scale: scale,
                child: PotPileVisual(
                  amount: amount,
                  theme: theme,
                  paletteIndex: paletteIndex,
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}
