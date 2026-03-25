import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class CommunityCardSlots extends StatelessWidget {
  const CommunityCardSlots({
    super.key,
    required this.layout,
    required this.cards,
    required this.theme,
  });

  final TableLayout layout;
  final List<pr.Card> cards;
  final PokerThemeConfig theme;

  static const int _totalSlots = 5;

  @override
  Widget build(BuildContext context) {
    final communityRect = layout.scene.communityRect;
    final baseCw = (communityRect.width / 5.4).clamp(28.0, 56.0).toDouble();
    final cw = (baseCw * theme.cardSizeMultiplier).clamp(20.0, 80.0).toDouble();
    final ch = cw * 1.4;
    final gap = cw * 0.10;
    final totalW = (_totalSlots * cw) + ((_totalSlots - 1) * gap);
    final startX = communityRect.center.dx - totalW / 2;
    final y = communityRect.center.dy - ch / 2;

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
            : _PlaceholderSlot(
                key: ValueKey('community_slot_$i'),
                borderRadius: (cw * 0.1).clamp(4.0, 10.0),
              ),
      ));
    }
    return IgnorePointer(child: Stack(children: children));
  }
}

class _PlaceholderSlot extends StatelessWidget {
  const _PlaceholderSlot({super.key, required this.borderRadius});
  final double borderRadius;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.03),
        borderRadius: BorderRadius.circular(borderRadius),
        border: Border.all(
          color: Colors.white.withOpacity(0.08),
          width: 1.5,
        ),
      ),
    );
  }
}
