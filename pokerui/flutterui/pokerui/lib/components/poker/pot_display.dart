import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';

/// Compact pot display anchored on the felt. Also serves as the anchor point
/// for chip animations by aligning to `potChipCenter`.
class PotDisplay extends StatelessWidget {
  const PotDisplay({super.key, required this.pot, required this.theme});

  final int pot;
  final PokerThemeConfig theme;

  @override
  Widget build(BuildContext context) {
    if (pot <= 0) return const SizedBox.shrink();

    return Positioned.fill(
      child: IgnorePointer(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final layout = resolveTableLayout(constraints.biggest);
            final anchor = potChipCenter(
              layout,
              uiSizeMultiplier: theme.uiSizeMultiplier,
            );

            final textStyle = TextStyle(
              color: Colors.white,
              fontSize: 14 * theme.uiSizeMultiplier,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.2,
            );
            final label = TextSpan(text: 'Pot: $pot', style: textStyle);
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
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      color: const Color(0xFF0C1222).withOpacity(0.9),
                      borderRadius:
                          BorderRadius.circular(14 * theme.uiSizeMultiplier),
                      border: Border.all(
                        color: decredBlue.withOpacity(0.8),
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
                            'Pot: $pot',
                            style: textStyle,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ],
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
            colors: [decredBlue, decredGreen],
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
              color: const Color(0xFF0D1A2F).withOpacity(0.9),
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
