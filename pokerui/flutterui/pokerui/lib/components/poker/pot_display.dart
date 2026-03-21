import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';

class PotDisplay extends StatefulWidget {
  const PotDisplay({super.key, required this.pot, required this.theme});

  final int pot;
  final PokerThemeConfig theme;

  @override
  State<PotDisplay> createState() => _PotDisplayState();
}

class _PotDisplayState extends State<PotDisplay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _pulseCtrl;
  late final Animation<double> _scale;
  int _prevPot = 0;

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
    return Positioned.fill(
      child: IgnorePointer(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final layout = resolveTableLayout(constraints.biggest);
            final anchor = potChipCenter(
              layout,
              uiSizeMultiplier: theme.uiSizeMultiplier,
            );

            final textStyle = PokerTypography.potLabel.copyWith(
              fontSize: 14 * theme.uiSizeMultiplier,
            );
            final label = TextSpan(text: 'Pot: ${widget.pot}', style: textStyle);
            final painter = TextPainter(
              text: label,
              textDirection: TextDirection.ltr,
              maxLines: 1,
            )..layout();

            final iconSize = 16.0 * theme.uiSizeMultiplier;
            final hPad = 10.0 * theme.uiSizeMultiplier;
            final vPad = 6.0 * theme.uiSizeMultiplier;
            final gap = 6.0 * theme.uiSizeMultiplier;
            final width = painter.width + hPad * 2 + iconSize + gap;
            final height = painter.height + vPad * 2;

            final left = anchor.dx - width / 2;
            final top = anchor.dy - height / 2;

            return Stack(
              children: [
                Positioned(
                  left: left,
                  top: top,
                  width: width,
                  height: height,
                  child: AnimatedBuilder(
                    animation: _scale,
                    builder: (context, child) => Transform.scale(
                      scale: _scale.value,
                      child: child,
                    ),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: PokerColors.surfaceDim.withOpacity(0.92),
                        borderRadius:
                            BorderRadius.circular(14 * theme.uiSizeMultiplier),
                        border: Border.all(
                          color: PokerColors.potBorder.withOpacity(0.7),
                          width: 1.5 * theme.uiSizeMultiplier,
                        ),
                        boxShadow: [
                          BoxShadow(
                            color: Colors.black.withOpacity(0.35),
                            blurRadius: 8 * theme.uiSizeMultiplier,
                            spreadRadius: 1 * theme.uiSizeMultiplier,
                          ),
                        ],
                      ),
                      child: Padding(
                        padding: EdgeInsets.symmetric(
                            horizontal: hPad, vertical: vPad),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            _ChipIcon(size: iconSize),
                            SizedBox(width: gap),
                            Text(
                              'Pot: ${widget.pot}',
                              style: textStyle,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
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

class _ChipIcon extends StatelessWidget {
  const _ChipIcon({required this.size});
  final double size;

  @override
  Widget build(BuildContext context) {
    final innerSize = size * 0.68;
    return SizedBox(
      width: size,
      height: size,
      child: DecoratedBox(
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: const LinearGradient(
            colors: [PokerColors.primary, PokerColors.accent],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
          border: Border.all(
            color: Colors.white.withOpacity(0.85),
            width: size * 0.08,
          ),
        ),
        child: Center(
          child: Container(
            width: innerSize,
            height: innerSize,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: PokerColors.surfaceDim.withOpacity(0.9),
              border: Border.all(
                color: Colors.white.withOpacity(0.7),
                width: size * 0.06,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
