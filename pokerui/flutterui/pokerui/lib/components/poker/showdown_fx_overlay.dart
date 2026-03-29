import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/player_seat.dart';
import 'package:pokerui/components/poker/pot_display.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';

/// Pot collect / payout animations over the table during showdown.
class ShowdownFxOverlay extends StatefulWidget {
  const ShowdownFxOverlay({
    super.key,
    required this.model,
    required this.layout,
  });

  final PokerModel model;
  final TableLayout layout;

  @override
  State<ShowdownFxOverlay> createState() => _ShowdownFxOverlayState();
}

class _ShowdownFxOverlayState extends State<ShowdownFxOverlay>
    with TickerProviderStateMixin {
  late final AnimationController _collectCtrl;
  late final AnimationController _payoutCtrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _collectCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 380),
    );
    _payoutCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 780),
    );
    _collectCtrl.addStatusListener((status) {
      if (status == AnimationStatus.completed) {
        _payoutCtrl
          ..reset()
          ..forward();
      }
    });
    _maybeRestartFx();
  }

  @override
  void didUpdateWidget(covariant ShowdownFxOverlay oldWidget) {
    super.didUpdateWidget(oldWidget);
    _maybeRestartFx();
  }

  void _maybeRestartFx() {
    final winners = widget.model.showdownWinners;
    final fxMs = widget.model.lastShowdownFxMs;
    if (winners.isEmpty || fxMs == 0) return;
    if (fxMs != _lastFxMs) {
      _lastFxMs = fxMs;
      _payoutCtrl.reset();
      _collectCtrl
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _collectCtrl.dispose();
    _payoutCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final game = widget.model.game;
    if (game == null) return const SizedBox.shrink();
    final winners = widget.model.showdownWinners;
    final theme = PokerThemeConfig.fromContext(context);
    final layout = widget.layout;
    final center = layout.center;

    final allWidgets = <Widget>[];
    if (winners.isNotEmpty && game.players.isNotEmpty) {
      final seatCenters = seatAvatarCentersFor(
        gameState: game,
        heroId: widget.model.playerId,
        theme: theme,
        layout: layout,
        showdownWinners: winners,
      );
      final potOrigin = potStackAnchor(layout, theme);
      final totalWinnings = winners.fold<int>(0, (sum, w) => sum + w.winnings);

      allWidgets.add(_FadeInPot(
        key: const ValueKey('showdown-collect-fade'),
        fadeIn: _collectCtrl,
        fadeOut: _payoutCtrl,
        position: potOrigin,
        amount: totalWinnings,
        theme: theme,
      ));

      final originSpread =
          20.0 * theme.uiSizeMultiplier * (winners.length > 1 ? 1.0 : 0.0);
      for (int i = 0; i < winners.length; i++) {
        final w = winners[i];
        final target = seatCenters[w.playerId] ?? center;
        final startXOffset =
            (i - ((winners.length - 1) / 2)) * originSpread.clamp(0, 28);
        allWidgets.add(_AnimatedPotFlight(
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
      child: Stack(children: allWidgets),
    );
  }
}

class _FadeInPot extends StatelessWidget {
  const _FadeInPot({
    super.key,
    required this.fadeIn,
    required this.fadeOut,
    required this.position,
    required this.amount,
    required this.theme,
  });

  final Animation<double> fadeIn;
  final Animation<double> fadeOut;
  final Offset position;
  final int amount;
  final PokerThemeConfig theme;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: Listenable.merge([fadeIn, fadeOut]),
      builder: (context, child) {
        final inT = fadeIn.value;
        if (inT <= 0.0) return const SizedBox.shrink();
        final fadeOutT = fadeOut.value;
        if (fadeOutT > 0.0) return const SizedBox.shrink();

        final opacity = Curves.easeOut.transform(inT.clamp(0.0, 1.0));
        if (opacity <= 0.0) return const SizedBox.shrink();

        final scale =
            0.85 + 0.15 * Curves.easeOut.transform(inT.clamp(0.0, 1.0));

        return Positioned(
          left: position.dx,
          top: position.dy,
          child: FractionalTranslation(
            translation: const Offset(-0.5, -0.1),
            child: Opacity(
              opacity: opacity,
              child: Transform.scale(
                scale: scale,
                child: child,
              ),
            ),
          ),
        );
      },
      child: PotPileVisual(
        amount: amount,
        theme: theme,
      ),
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
