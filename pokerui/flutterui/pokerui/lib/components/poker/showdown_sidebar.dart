import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/showdown_content.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

class ShowdownSidebar extends StatelessWidget {
  const ShowdownSidebar({
    super.key,
    required this.model,
    this.visible = true,
    this.onClose,
    @Deprecated('use visible') bool? isVisible,
    dynamic result,
  });

  final PokerModel model;
  final bool visible;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    if (!visible) return const SizedBox.shrink();

    final winners = model.lastWinners;
    final pot = model.showdownPot;
    final bottomInset = MediaQuery.paddingOf(context).bottom;

    return SingleChildScrollView(
      key: const Key('showdown-sidebar-scroll'),
      physics: const ClampingScrollPhysics(),
      child: Padding(
        padding: EdgeInsets.only(bottom: bottomInset),
        child: Container(
          key: const Key('showdown-sidebar'),
          clipBehavior: Clip.antiAlias,
          decoration: BoxDecoration(
            color: PokerColors.surfaceDim,
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity(0.18),
                blurRadius: 12,
                offset: const Offset(6, 0),
              ),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Container(
                width: double.infinity,
                padding: const EdgeInsets.fromLTRB(
                  PokerSpacing.lg,
                  PokerSpacing.xxxl,
                  PokerSpacing.lg,
                  PokerSpacing.lg,
                ),
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: [
                      PokerColors.primary.withOpacity(0.3),
                      PokerColors.surfaceDim,
                    ],
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                  ),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              const Icon(
                                Icons.history,
                                color: PokerColors.primary,
                                size: 24,
                              ),
                              const SizedBox(width: PokerSpacing.sm),
                              Text(
                                winners.isNotEmpty
                                    ? 'Showdown'
                                    : 'Last Showdown',
                                style: PokerTypography.headlineMedium,
                              ),
                            ],
                          ),
                          const SizedBox(height: PokerSpacing.sm),
                          Text(
                            'Review the board, winners, and exposed hole cards from the last hand.',
                            style: PokerTypography.bodySmall.copyWith(
                              color: PokerColors.textSecondary,
                            ),
                          ),
                          if (pot > 0) ...[
                            const SizedBox(height: PokerSpacing.md),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 12,
                                vertical: 4,
                              ),
                              decoration: BoxDecoration(
                                color: PokerColors.overlaySubtle,
                                borderRadius: BorderRadius.circular(12),
                                border: Border.all(
                                  color:
                                      PokerColors.borderBright.withOpacity(0.7),
                                ),
                              ),
                              child: Text(
                                'Pot: $pot',
                                style: PokerTypography.chipCount.copyWith(
                                  color: PokerColors.warning,
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                    if (onClose != null) ...[
                      const SizedBox(width: PokerSpacing.sm),
                      IconButton(
                        onPressed: onClose,
                        icon: const Icon(
                          Icons.close,
                          color: PokerColors.textPrimary,
                        ),
                        tooltip: 'Close last hand details',
                        visualDensity: VisualDensity.compact,
                      ),
                    ],
                  ],
                ),
              ),
              const Divider(height: 1, color: PokerColors.borderSubtle),
              ShowdownContent(
                model: model,
                showHeader: false,
                showCloseButton: false,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
