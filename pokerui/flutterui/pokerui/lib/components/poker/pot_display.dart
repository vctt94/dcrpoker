import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';

class PokerChipDenomination {
  const PokerChipDenomination({
    required this.value,
    required this.fill,
    required this.ring,
    required this.edge,
  });

  final int value;
  final Color fill;
  final Color ring;
  final Color edge;
}

class PokerChipToken {
  const PokerChipToken({
    required this.denomination,
    required this.count,
  });

  final PokerChipDenomination denomination;
  final int count;
}

const List<PokerChipDenomination> _realisticChipDenominations = [
  PokerChipDenomination(
    value: 1000,
    fill: Color(0xFFD1A53A),
    ring: Color(0xFFF8E8A6),
    edge: Color(0xFF8C6714),
  ),
  PokerChipDenomination(
    value: 500,
    fill: Color(0xFF6D43C6),
    ring: Color(0xFFD9C6FF),
    edge: Color(0xFF37206D),
  ),
  PokerChipDenomination(
    value: 100,
    fill: Color(0xFF1B1F28),
    ring: Color(0xFFD7DCE7),
    edge: Color(0xFF08090D),
  ),
  PokerChipDenomination(
    value: 25,
    fill: Color(0xFF1F8B4C),
    ring: Color(0xFFC6F4D0),
    edge: Color(0xFF0C4A24),
  ),
  PokerChipDenomination(
    value: 5,
    fill: Color(0xFFD24343),
    ring: Color(0xFFFFD1D1),
    edge: Color(0xFF7A1F1F),
  ),
  PokerChipDenomination(
    value: 1,
    fill: Color(0xFFF2F3F6),
    ring: Color(0xFFFFFFFF),
    edge: Color(0xFFB5BDC9),
  ),
];

List<PokerChipToken> chipTokensForAmount(
  int amount, {
  int maxColumns = 4,
  int maxChipsPerColumn = 6,
}) {
  if (amount <= 0) return const [];

  var remaining = amount;
  final tokens = <PokerChipToken>[];

  for (final denomination in _realisticChipDenominations) {
    if (remaining < denomination.value) continue;
    final rawCount = remaining ~/ denomination.value;
    remaining %= denomination.value;
    final cappedCount = rawCount.clamp(1, maxChipsPerColumn);
    tokens.add(PokerChipToken(
      denomination: denomination,
      count: cappedCount,
    ));
    if (tokens.length >= maxColumns) break;
  }

  if (tokens.isEmpty) {
    final fallback = _realisticChipDenominations.last;
    return [PokerChipToken(denomination: fallback, count: 1)];
  }

  if (remaining > 0 && tokens.length < maxColumns) {
    final lowest = _realisticChipDenominations.last;
    tokens.add(PokerChipToken(
      denomination: lowest,
      count: 1,
    ));
  }

  return tokens;
}

Offset potStackAnchor(TableLayout layout, PokerThemeConfig theme) {
  final potSpec = PokerPotSpec.resolve(
    layoutMode: layout.scene.mode,
    uiSizeMultiplier: theme.uiSizeMultiplier,
  );
  final base = potChipCenter(
    layout,
    uiSizeMultiplier: theme.uiSizeMultiplier,
  );
  return base.translate(0, -potSpec.stackLift);
}

Offset potTotalAnchor(TableLayout layout, PokerThemeConfig theme) {
  final potSpec = PokerPotSpec.resolve(
    layoutMode: layout.scene.mode,
    uiSizeMultiplier: theme.uiSizeMultiplier,
  );
  final centerX = layout.scene.communityRect.center.dx;
  return Offset(
    centerX,
    layout.scene.communityRect.top - potSpec.totalGap,
  );
}

class PotDisplay extends StatefulWidget {
  const PotDisplay({
    super.key,
    required this.layout,
    required this.pot,
    required this.theme,
    this.settleFxMs = 0,
    this.hideForPayout = false,
  });

  final TableLayout layout;
  final int pot;
  final PokerThemeConfig theme;
  final int settleFxMs;
  final bool hideForPayout;

  @override
  State<PotDisplay> createState() => _PotDisplayState();
}

class _PotDisplayState extends State<PotDisplay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _pulseCtrl;
  late final Animation<double> _scale;
  int _prevPot = 0;
  int _lastSettleFxMs = 0;
  bool _hiddenForPayout = false;

  @override
  void initState() {
    super.initState();
    _prevPot = widget.pot;
    _pulseCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 350),
    );
    _scale = TweenSequence<double>([
      TweenSequenceItem(tween: Tween(begin: 1.0, end: 1.12), weight: 40),
      TweenSequenceItem(tween: Tween(begin: 1.12, end: 1.0), weight: 60),
    ]).animate(CurvedAnimation(parent: _pulseCtrl, curve: Curves.easeOut));
  }

  @override
  void didUpdateWidget(covariant PotDisplay old) {
    super.didUpdateWidget(old);
    if (widget.pot > _prevPot && widget.pot > 0) {
      _pulseCtrl
        ..reset()
        ..forward();
    }
    if (widget.hideForPayout &&
        widget.settleFxMs != 0 &&
        widget.settleFxMs != _lastSettleFxMs) {
      _lastSettleFxMs = widget.settleFxMs;
      _hiddenForPayout = true;
    } else if (!widget.hideForPayout && _hiddenForPayout) {
      _hiddenForPayout = false;
    }
    _prevPot = widget.pot;
  }

  @override
  void dispose() {
    _pulseCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.pot <= 0) return const SizedBox.shrink();

    final theme = widget.theme;
    final potSpec = PokerPotSpec.resolve(
      layoutMode: widget.layout.scene.mode,
      uiSizeMultiplier: theme.uiSizeMultiplier,
    );
    return Positioned.fill(
      child: IgnorePointer(
        child: Builder(
          builder: (context) {
            final stackAnchor = potStackAnchor(widget.layout, theme);
            final totalAnchor = potTotalAnchor(widget.layout, theme);

            return Stack(
              children: [
                Positioned(
                  left: totalAnchor.dx,
                  top: totalAnchor.dy,
                  child: FractionalTranslation(
                    translation: const Offset(-0.5, -1.0),
                    child: AnimatedOpacity(
                      opacity: _hiddenForPayout ? 0.0 : 1.0,
                      duration: const Duration(milliseconds: 180),
                      curve: Curves.easeOut,
                      child: Text(
                        'Pot: ${widget.pot}',
                        key: const Key('poker-pot-total'),
                        style: PokerTypography.potLabel.copyWith(
                          fontSize: potSpec.potLabelFontSize,
                          color: PokerColors.textPrimary,
                          shadows: [
                            Shadow(
                              color: Colors.black.withValues(alpha: 0.4),
                              blurRadius: potSpec.potLabelBlur,
                              offset: const Offset(0, 1),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
                Positioned(
                  left: stackAnchor.dx,
                  top: stackAnchor.dy,
                  child: AnimatedBuilder(
                    animation: _scale,
                    builder: (context, child) => Transform.scale(
                      scale: _scale.value,
                      child: child,
                    ),
                    child: FractionalTranslation(
                      translation: const Offset(-0.5, -0.1),
                      child: AnimatedOpacity(
                        key: const Key('poker-pot-display'),
                        opacity: _hiddenForPayout ? 0.0 : 1.0,
                        duration: const Duration(milliseconds: 180),
                        curve: Curves.easeOut,
                        child: PotPileVisual(
                          amount: widget.pot,
                          theme: theme,
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class BetStackVisual extends StatelessWidget {
  const BetStackVisual({
    super.key,
    required this.amount,
    required this.theme,
  });

  final int amount;
  final PokerThemeConfig theme;

  @override
  Widget build(BuildContext context) {
    final potSpec = PokerPotSpec.resolve(
      layoutMode: PokerLayoutMode.standard,
      uiSizeMultiplier: theme.uiSizeMultiplier,
    );
    final columns =
        chipTokensForAmount(amount, maxColumns: 3, maxChipsPerColumn: 5);
    final chipSize = potSpec.betStackChipSize;
    final columnGap = chipSize * 0.3;
    final totalWidth =
        (columns.length * chipSize) + ((columns.length - 1) * columnGap);
    final tallest = columns.fold<int>(
      1,
      (maxCount, token) => token.count > maxCount ? token.count : maxCount,
    );
    final stackHeight = chipSize + ((tallest - 1) * chipSize * 0.18);

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: totalWidth,
          height: stackHeight,
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            mainAxisSize: MainAxisSize.min,
            children: [
              for (int i = 0; i < columns.length; i++) ...[
                if (i > 0) SizedBox(width: columnGap),
                _StraightChipColumn(
                  denomination: columns[i].denomination,
                  count: columns[i].count,
                  chipSize: chipSize,
                ),
              ],
            ],
          ),
        ),
        SizedBox(height: potSpec.betStackLabelGap),
        Text(
          '$amount',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: PokerTypography.potLabel.copyWith(
            fontSize: potSpec.betStackLabelFontSize,
            color: PokerColors.textPrimary,
            shadows: [
              Shadow(
                color: Colors.black.withValues(alpha: 0.35),
                blurRadius: potSpec.betStackLabelBlur,
                offset: const Offset(0, 1),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class PotPileVisual extends StatelessWidget {
  const PotPileVisual({
    super.key,
    required this.amount,
    required this.theme,
    this.paletteIndex = 0,
  });

  final int amount;
  final PokerThemeConfig theme;
  final int paletteIndex;

  @override
  Widget build(BuildContext context) {
    final potSpec = PokerPotSpec.resolve(
      layoutMode: PokerLayoutMode.standard,
      uiSizeMultiplier: theme.uiSizeMultiplier,
    );
    final tokens =
        chipTokensForAmount(amount, maxColumns: 5, maxChipsPerColumn: 3);
    final chipSize = potSpec.potPileChipSize;
    final baseOffsets = <Offset>[
      const Offset(0, 0),
      const Offset(-12, -2),
      const Offset(11, -1),
      const Offset(-5, -10),
      const Offset(9, -11),
      const Offset(-15, -8),
      const Offset(15, -7),
      const Offset(0, -13),
    ];
    final width = chipSize * 3.2;
    final height = chipSize * 2.2;

    return SizedBox(
      width: width,
      height: height,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          for (int i = 0; i < tokens.length; i++)
            Positioned(
              left: (width / 2) +
                  (baseOffsets[i % baseOffsets.length].dx *
                      theme.uiSizeMultiplier) -
                  (chipSize / 2),
              top: (height / 2) +
                  (baseOffsets[i % baseOffsets.length].dy *
                      theme.uiSizeMultiplier) -
                  (chipSize / 2),
              child: Transform.rotate(
                angle: ((i + paletteIndex) % 5 - 2) * 0.08,
                child: _PokerChip(
                  size: chipSize,
                  denomination: tokens[i].denomination,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _StraightChipColumn extends StatelessWidget {
  const _StraightChipColumn({
    required this.denomination,
    required this.count,
    required this.chipSize,
  });

  final PokerChipDenomination denomination;
  final int count;
  final double chipSize;

  @override
  Widget build(BuildContext context) {
    final height = chipSize + ((count - 1) * chipSize * 0.18);
    return SizedBox(
      width: chipSize,
      height: height,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          for (int i = 0; i < count; i++)
            Positioned(
              left: 0,
              bottom: i * chipSize * 0.18,
              child: _PokerChip(
                size: chipSize,
                denomination: denomination,
              ),
            ),
        ],
      ),
    );
  }
}

class _PokerChip extends StatelessWidget {
  const _PokerChip({
    required this.size,
    required this.denomination,
  });

  final double size;
  final PokerChipDenomination denomination;

  @override
  Widget build(BuildContext context) {
    final innerSize = size * 0.62;
    final stripeMain = size * 0.12;
    final stripeCross = size * 0.18;

    return SizedBox(
      width: size,
      height: size,
      child: DecoratedBox(
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: RadialGradient(
            center: const Alignment(-0.2, -0.25),
            radius: 0.95,
            colors: [
              Color.lerp(denomination.fill, Colors.white, 0.14) ??
                  denomination.fill,
              denomination.fill,
              denomination.edge,
            ],
            stops: const [0.0, 0.68, 1.0],
          ),
          border: Border.all(
            color: denomination.ring.withValues(alpha: 0.95),
            width: size * 0.075,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.18),
              blurRadius: size * 0.22,
              spreadRadius: size * 0.03,
            ),
          ],
        ),
        child: Stack(
          children: [
            for (final alignment in const [
              Alignment.topCenter,
              Alignment.bottomCenter,
              Alignment.centerLeft,
              Alignment.centerRight,
            ])
              Align(
                alignment: alignment,
                child: Container(
                  width: alignment == Alignment.topCenter ||
                          alignment == Alignment.bottomCenter
                      ? stripeMain
                      : stripeCross,
                  height: alignment == Alignment.topCenter ||
                          alignment == Alignment.bottomCenter
                      ? stripeCross
                      : stripeMain,
                  decoration: BoxDecoration(
                    color: denomination.ring.withValues(alpha: 0.9),
                    borderRadius: BorderRadius.circular(size * 0.04),
                  ),
                ),
              ),
            Center(
              child: Container(
                width: innerSize,
                height: innerSize,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: denomination.edge.withValues(alpha: 0.78),
                  border: Border.all(
                    color: denomination.ring.withValues(alpha: 0.82),
                    width: size * 0.045,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
