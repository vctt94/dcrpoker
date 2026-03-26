import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/showdown_content.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/shadows.dart';
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
    return LayoutBuilder(
      builder: (context, constraints) {
        final availableHeight = constraints.maxHeight.isFinite
            ? (constraints.maxHeight - bottomInset - PokerSpacing.lg)
                .clamp(0.0, constraints.maxHeight)
            : MediaQuery.sizeOf(context).height - bottomInset - PokerSpacing.lg;

        return Padding(
          padding: EdgeInsets.only(bottom: bottomInset + PokerSpacing.lg),
          child: SizedBox(
            height: availableHeight,
            child: Container(
              key: const Key('showdown-sidebar'),
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: [
                    PokerColors.surfaceBright,
                    PokerColors.surfaceDim,
                  ],
                ),
                borderRadius: const BorderRadius.only(
                  topRight: Radius.circular(28),
                  bottomRight: Radius.circular(28),
                ),
                border: Border.all(
                  color: PokerColors.borderSubtle,
                ),
                boxShadow: PokerShadows.overlay,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.fromLTRB(
                      PokerSpacing.lg,
                      PokerSpacing.xl,
                      PokerSpacing.lg,
                      PokerSpacing.lg,
                    ),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [
                          PokerColors.primary.withOpacity(0.26),
                          PokerColors.surfaceBright,
                        ],
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                      ),
                      border: Border(
                        bottom: BorderSide(
                          color: PokerColors.borderSubtle.withOpacity(0.9),
                        ),
                      ),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: Row(
                            children: [
                              const Icon(
                                Icons.history_edu_outlined,
                                color: PokerColors.primary,
                                size: 20,
                              ),
                              const SizedBox(width: PokerSpacing.sm),
                              Expanded(
                                child: Text(
                                  winners.isNotEmpty
                                      ? 'Showdown'
                                      : 'Last Showdown',
                                  style: PokerTypography.headlineMedium,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                              if (pot > 0)
                                const SizedBox(width: PokerSpacing.sm),
                              if (pot > 0)
                                Text(
                                  'Pot $pot',
                                  style: PokerTypography.labelSmall.copyWith(
                                    color: PokerColors.warning,
                                  ),
                                  overflow: TextOverflow.ellipsis,
                                ),
                            ],
                          ),
                        ),
                        if (onClose != null) ...[
                          const SizedBox(width: PokerSpacing.sm),
                          Container(
                            decoration: BoxDecoration(
                              color: PokerColors.overlaySubtle,
                              borderRadius: BorderRadius.circular(12),
                              border: Border.all(
                                color: PokerColors.borderSubtle,
                              ),
                            ),
                            child: IconButton(
                              onPressed: onClose,
                              icon: const Icon(
                                Icons.close,
                                color: PokerColors.textPrimary,
                                size: 18,
                              ),
                              tooltip: 'Close last hand details',
                              visualDensity: VisualDensity.compact,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                  Expanded(
                    child: SingleChildScrollView(
                      key: const Key('showdown-sidebar-scroll'),
                      physics: const ClampingScrollPhysics(),
                      child: ShowdownContent(
                        model: model,
                        showHeader: false,
                        showCloseButton: false,
                        cardScale: 1.2,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}
