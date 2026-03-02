import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

/// Renders 5 card slots on the felt. Dealt community cards fill their
/// corresponding slot; undealt positions show a subtle placeholder outline.
class CommunityCardSlots extends StatelessWidget {
  const CommunityCardSlots({
    super.key,
    required this.cards,
    this.aspectRatio = 16 / 9,
  });

  final List<pr.Card> cards;
  final double aspectRatio;

  static const int _totalSlots = 5;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final theme = PokerThemeConfig.fromContext(context);
      final box = pokerViewportRect(size, aspectRatio: aspectRatio);
      final center = Offset(box.left + box.width / 2, box.top + box.height / 2);
      final baseCw = (box.width * 0.05).clamp(32.0, 56.0).toDouble();
      final cw =
          (baseCw * theme.cardSizeMultiplier).clamp(20.0, 80.0).toDouble();
      final ch = cw * 1.4;
      final gap = cw * 0.10;
      final totalW = (_totalSlots * cw) + ((_totalSlots - 1) * gap);
      final startX = center.dx - totalW / 2;
      final y = center.dy - ch / 2 - 20.0;

      final children = <Widget>[];
      for (int i = 0; i < _totalSlots; i++) {
        final x = startX + i * (cw + gap);
        final hasCard = i < cards.length;
        children.add(Positioned(
          left: x,
          top: y,
          width: cw,
          height: ch,
          child: hasCard
              ? CardFace(card: cards[i], cardTheme: theme.cardTheme)
              : const _PlaceholderSlot(borderRadius: 8),
        ));
      }
      return IgnorePointer(child: Stack(children: children));
    });
  }
}

class _PlaceholderSlot extends StatelessWidget {
  const _PlaceholderSlot({required this.borderRadius});
  final double borderRadius;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.04),
        borderRadius: BorderRadius.circular(borderRadius),
        border: Border.all(
          color: Colors.white.withOpacity(0.10),
          width: 1.5,
        ),
      ),
    );
  }
}
