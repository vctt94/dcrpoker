import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class BetSidebar extends StatefulWidget {
  final PokerModel model;
  final VoidCallback? onToggleSidebar;
  final bool isMinimized;

  const BetSidebar({
    super.key,
    required this.model,
    this.onToggleSidebar,
    this.isMinimized = false,
  });

  @override
  State<BetSidebar> createState() => _BetSidebarState();
}

class _BetSidebarState extends State<BetSidebar> {
  bool _minimized = false;

  @override
  void initState() {
    super.initState();
    _minimized = widget.isMinimized;
  }

  @override
  Widget build(BuildContext context) {
    final g = widget.model.game;
    if (g == null) return const SizedBox.shrink();

    final currentBet = g.currentBet;
    final me = widget.model.me;
    final myBet = me?.currentBet ?? 0;
    final toCall = (currentBet - myBet) > 0 ? currentBet - myBet : 0;

    if (_minimized) {
      return Positioned(
        top: PokerSpacing.md,
        right: PokerSpacing.md,
        child: GestureDetector(
          onTap: () => setState(() => _minimized = false),
          child: Container(
            padding: const EdgeInsets.all(PokerSpacing.sm),
            decoration: BoxDecoration(
              color: PokerColors.surfaceDim.withOpacity(0.92),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: PokerColors.borderSubtle),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.attach_money, size: 14, color: PokerColors.warning),
                const SizedBox(width: PokerSpacing.xxs),
                Text(
                  '$currentBet',
                  style: PokerTypography.chipCount.copyWith(
                    fontSize: 12,
                    color: PokerColors.warning,
                  ),
                ),
                const SizedBox(width: PokerSpacing.xxs),
                Icon(Icons.chevron_left,
                    size: 14, color: PokerColors.textMuted),
              ],
            ),
          ),
        ),
      );
    }

    return Positioned(
      top: PokerSpacing.md,
      right: PokerSpacing.md,
      child: Container(
        width: 190,
        padding: const EdgeInsets.all(PokerSpacing.md),
        decoration: BoxDecoration(
          color: PokerColors.surfaceDim.withOpacity(0.92),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: PokerColors.borderSubtle),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('Betting',
                    style: PokerTypography.labelSmall.copyWith(
                      color: PokerColors.primary,
                    )),
                const Spacer(),
                GestureDetector(
                  onTap: () => setState(() => _minimized = true),
                  child: Icon(Icons.minimize,
                      size: 14, color: PokerColors.textMuted),
                ),
              ],
            ),
            const SizedBox(height: PokerSpacing.sm),
            Text(
              'Current Bet: $currentBet',
              style: PokerTypography.chipCount.copyWith(
                color: PokerColors.warning,
                fontSize: 12,
              ),
            ),
            if (toCall > 0)
              Padding(
                padding: const EdgeInsets.only(top: PokerSpacing.xs),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.call_made,
                          color: PokerColors.warning,
                          size: 14,
                        ),
                        const SizedBox(width: PokerSpacing.xxs),
                        Text(
                          'To Call: $toCall',
                          style: PokerTypography.chipCount.copyWith(
                            color: PokerColors.warning,
                            fontSize: 12,
                          ),
                        ),
                      ],
                    ),
                    if (myBet > 0)
                      Padding(
                        padding: const EdgeInsets.only(top: PokerSpacing.xxs),
                        child: Text(
                          'Your bet: $myBet',
                          style: PokerTypography.bodySmall.copyWith(
                            color: PokerColors.textMuted,
                          ),
                        ),
                      ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}
