import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/pot_display.dart';
import 'package:pokerui/components/poker/player_seat.dart';
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

    return LayoutBuilder(
      builder: (context, constraints) {
        final scene = PokerSceneLayout.resolve(
          constraints.biggest,
          safePadding: MediaQuery.paddingOf(context),
        );
        final useMobileDock = scene.mode == PokerLayoutMode.compactPortrait;
        final showTableHeroCards = !useMobileDock;
        const toggleInset = 4.0;
        const sidebarGap = 24.0;
        final sidebarWidth = useMobileDock
            ? (constraints.maxWidth * 0.74).clamp(260.0, 320.0)
            : (constraints.maxWidth * 0.32).clamp(280.0, 396.0);
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
              scene: scene,
              showHeroSeatCards: showTableHeroCards,
            ),
            _ShowdownFxOverlay(
              model: model,
              layout: TableLayout.fromScene(scene),
            ),
            if (model.hasLastShowdown)
              AnimatedPositioned(
                duration: const Duration(milliseconds: 260),
                curve: Curves.easeOutCubic,
                left: _showSidebar ? 0 : -(sidebarWidth + sidebarGap),
                top: 0,
                bottom: 0,
                width: sidebarWidth + sidebarGap,
                child: IgnorePointer(
                  ignoring: !_showSidebar,
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: sidebarWidth,
                        child: ShowdownSidebar(
                          model: model,
                          visible: true,
                          onClose: _closeSidebar,
                        ),
                      ),
                      const IgnorePointer(
                        child: SizedBox(width: sidebarGap),
                      ),
                    ],
                  ),
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
  const _ShowdownFxOverlay({
    required this.model,
    required this.layout,
  });
  final PokerModel model;
  final TableLayout layout;

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
      duration: const Duration(milliseconds: 780),
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
    final theme = PokerThemeConfig.fromContext(context);
    final layout = widget.layout;
    final center = layout.center;

    final payoutWidgets = <Widget>[];
    if (winners.isNotEmpty && game.players.isNotEmpty) {
      final targets = seatAvatarCentersFor(
        gameState: game,
        heroId: widget.model.playerId,
        theme: theme,
        layout: layout,
      );
      final potOrigin = potStackAnchor(layout, theme);
      final originSpread =
          20.0 * theme.uiSizeMultiplier * (winners.length > 1 ? 1.0 : 0.0);

      for (int i = 0; i < winners.length; i++) {
        final w = winners[i];
        final target = targets[w.playerId] ?? center;
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
          delay: i * 0.07,
        ));
      }
    }

    return IgnorePointer(
      child: Stack(children: payoutWidgets),
    );
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
        final arcHeight = 20.0 * theme.uiSizeMultiplier;
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
        final anchorY = Tween<double>(
          begin: -0.32,
          end: -0.5,
        ).transform(Curves.easeOut.transform(progress));

        return Positioned(
          left: dx,
          top: dy,
          child: FractionalTranslation(
            translation: Offset(-0.5, anchorY),
            child: Opacity(
              opacity: opacity,
              child: Transform.scale(
                scale: scale,
                child: PotPileVisual(
                  key: ValueKey('showdown-payout-visual-$paletteIndex'),
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
