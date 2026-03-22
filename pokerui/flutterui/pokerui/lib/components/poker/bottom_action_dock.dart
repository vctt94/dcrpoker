import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class _ActionControls {
  const _ActionControls({
    required this.betCtrl,
    required this.onToggleBetInput,
    required this.onCloseBetInput,
  });

  final TextEditingController betCtrl;
  final VoidCallback onToggleBetInput;
  final VoidCallback onCloseBetInput;
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.label,
    required this.onPressed,
    required this.color,
    this.icon,
  });

  final String label;
  final VoidCallback? onPressed;
  final Color color;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final scale = buttonScale(bp);
    return ElevatedButton(
      onPressed: onPressed,
      style: ElevatedButton.styleFrom(
        backgroundColor: color,
        foregroundColor: Colors.white,
        padding: EdgeInsets.symmetric(
          horizontal: 20 * scale,
          vertical: 12 * scale,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12 * scale),
        ),
        elevation: 2,
        shadowColor: color.withValues(alpha: 0.3),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 16 * scale),
            SizedBox(width: 6 * scale),
          ],
          Text(
            label,
            style: TextStyle(
              fontSize: 14 * scale,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.3,
            ),
          ),
        ],
      ),
    );
  }
}

class BottomActionDock extends StatelessWidget {
  BottomActionDock({
    super.key,
    required this.model,
    required this.showBetInput,
    required TextEditingController betCtrl,
    required VoidCallback onToggleBetInput,
    required VoidCallback onCloseBetInput,
    this.reserveActionSpace = false,
    this.footer,
  })  : _actionControls = _ActionControls(
          betCtrl: betCtrl,
          onToggleBetInput: onToggleBetInput,
          onCloseBetInput: onCloseBetInput,
        ),
        showActions = true;

  const BottomActionDock.passive({
    super.key,
    required this.model,
    this.reserveActionSpace = false,
    this.footer,
  })  : showBetInput = false,
        _actionControls = null,
        showActions = false;

  final PokerModel model;
  final bool showBetInput;
  final _ActionControls? _actionControls;
  final bool showActions;
  final bool reserveActionSpace;
  final Widget? footer;

  int _resolveBigBlind() {
    if ((model.game?.bigBlind ?? 0) > 0) return model.game!.bigBlind;
    final tid = model.currentTableId;
    if (tid == null) return 0;
    final matches = model.tables.where((t) => t.id == tid).toList();
    return matches.isNotEmpty ? matches.first.bigBlind : 0;
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final canAct = model.canAct;
    final me = model.me;
    final cards =
        (me?.hand.isNotEmpty ?? false) ? me!.hand : model.myHoleCardsCache;
    final hasCards = cards.isNotEmpty;

    return Container(
      padding: EdgeInsets.only(
        left: PokerSpacing.md,
        right: PokerSpacing.md,
        top: PokerSpacing.sm,
        bottom: safeBottomPadding(context, minPadding: 8),
      ),
      constraints: BoxConstraints(minHeight: actionDockMinHeight(bp)),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            PokerColors.screenBg.withValues(alpha: 0.0),
            PokerColors.screenBg.withValues(alpha: 0.95),
          ],
        ),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final tightDesktopHeight = constraints.maxHeight <= 124;
          final headerGap =
              tightDesktopHeight ? PokerSpacing.xs : PokerSpacing.sm;
          final actionControls = _actionControls;
          final actions = Visibility(
            visible: showActions,
            maintainState: reserveActionSpace,
            maintainAnimation: reserveActionSpace,
            maintainSize: reserveActionSpace,
            child: Align(
              alignment: Alignment.centerRight,
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: showActions && canAct
                    ? _ActionButtons(
                        model: model,
                        showBetInput: showBetInput,
                        betCtrl: actionControls!.betCtrl,
                        onToggleBetInput: actionControls.onToggleBetInput,
                        onCloseBetInput: actionControls.onCloseBetInput,
                        bb: _resolveBigBlind(),
                      )
                    : showActions
                        ? _WaitingIndicator(model: model)
                        : const SizedBox.shrink(),
              ),
            ),
          );
          final headerPanel = Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              if (hasCards) ...[
                Align(
                  alignment: Alignment.centerRight,
                  child: _ShowCardsDockToggle(
                    model: model,
                    compact: true,
                  ),
                ),
                if (showActions || reserveActionSpace)
                  SizedBox(height: headerGap),
              ],
            ],
          );
          final hasBottomSection =
              showActions || reserveActionSpace || footer != null;
          final bottomSection = Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              if (showActions || reserveActionSpace) actions,
              if (footer != null) ...[
                if (showActions || reserveActionSpace)
                  const SizedBox(height: PokerSpacing.sm),
                footer!,
              ],
            ],
          );

          return Column(
            mainAxisSize: MainAxisSize.max,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (hasCards)
                Align(
                  alignment: Alignment.topRight,
                  child: headerPanel,
                ),
              if (hasBottomSection) const Spacer(),
              if (hasBottomSection) bottomSection,
            ],
          );
        },
      ),
    );
  }
}

class MobileHeroActionPanel extends StatelessWidget {
  MobileHeroActionPanel({
    super.key,
    required this.model,
    required this.showBetInput,
    required TextEditingController betCtrl,
    required VoidCallback onToggleBetInput,
    required VoidCallback onCloseBetInput,
    this.hasLastShowdown = false,
    this.showSidebar = false,
    this.onToggleSidebar,
    this.reserveActionSpace = false,
    this.footer,
  })  : _actionControls = _ActionControls(
          betCtrl: betCtrl,
          onToggleBetInput: onToggleBetInput,
          onCloseBetInput: onCloseBetInput,
        ),
        showActions = true;

  MobileHeroActionPanel.passive({
    super.key,
    required this.model,
    this.hasLastShowdown = false,
    this.showSidebar = false,
    this.onToggleSidebar,
    this.reserveActionSpace = false,
    this.footer,
  })  : showBetInput = false,
        _actionControls = null,
        showActions = false;

  final PokerModel model;
  final bool showBetInput;
  final _ActionControls? _actionControls;
  final bool hasLastShowdown;
  final bool showSidebar;
  final VoidCallback? onToggleSidebar;
  final bool showActions;
  final bool reserveActionSpace;
  final Widget? footer;

  int _resolveBigBlind() {
    if ((model.game?.bigBlind ?? 0) > 0) return model.game!.bigBlind;
    final tid = model.currentTableId;
    if (tid == null) return 0;
    final matches = model.tables.where((t) => t.id == tid).toList();
    return matches.isNotEmpty ? matches.first.bigBlind : 0;
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final canAct = model.canAct;
    final actionControls = _actionControls;
    final actionRowHeight = (48 * buttonScale(bp)).floorToDouble();
    final me = model.me;
    final cards =
        (me?.hand.isNotEmpty ?? false) ? me!.hand : model.myHoleCardsCache;
    final hasCards = cards.isNotEmpty;

    return LayoutBuilder(
      builder: (context, panelConstraints) {
        final availableHeight = panelConstraints.maxHeight;
        final tightVertical =
            availableHeight.isFinite && availableHeight <= 152.0;
        final sectionGap = tightVertical ? 6.0 : PokerSpacing.sm;
        final trailingGap = tightVertical ? 4.0 : 6.0;
        final trailingSectionGap = tightVertical ? 6.0 : PokerSpacing.sm;
        final topPadding = tightVertical ? 6.0 : PokerSpacing.sm;
        final headerSection = LayoutBuilder(
          builder: (context, constraints) {
            final hasLastHandButton =
                hasLastShowdown && onToggleSidebar != null;
            final cardScale = cardSizeMultiplierFromKey(context.cardSize);
            final cardWidth = (42.0 * cardScale).clamp(24.0, 60.0).toDouble();
            final cardGap = (cardWidth * 0.14).clamp(4.0, 8.0).toDouble();
            final cardsWidth = (cardWidth * 2) + cardGap;
            var trailingWidth = 0.0;
            if (hasCards && trailingWidth < 116.0) trailingWidth = 116.0;
            final lastHandWidth = hasLastHandButton ? 92.0 : 0.0;
            if (lastHandWidth > trailingWidth) {
              trailingWidth = lastHandWidth;
            }
            final stackedHeader =
                constraints.maxWidth < cardsWidth + trailingWidth + 36.0;
            final cardsRow = _CompactHeroCards(cards: cards);
            final hasTrailingControls = hasCards || hasLastHandButton;
            final trailingControls = hasTrailingControls
                ? Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      if (hasCards)
                        _ShowCardsDockToggle(
                          model: model,
                          compact: true,
                        ),
                      if (hasCards && hasLastHandButton)
                        SizedBox(height: trailingGap),
                      if (hasLastHandButton) ...[
                        if (hasCards) SizedBox(height: trailingSectionGap),
                        PokerLastHandButton(
                          active: showSidebar,
                          onTap: onToggleSidebar!,
                          compact: true,
                        ),
                      ],
                    ],
                  )
                : const SizedBox.shrink();

            if (!stackedHeader) {
              return Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  cardsRow,
                  const Spacer(),
                  if (hasTrailingControls) trailingControls,
                ],
              );
            }

            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                cardsRow,
                if (hasTrailingControls) ...[
                  SizedBox(height: sectionGap),
                  Align(
                    alignment: Alignment.centerRight,
                    child: trailingControls,
                  ),
                ],
              ],
            );
          },
        );
        final hasBottomSection =
            showActions || reserveActionSpace || footer != null;
        final bottomSection = Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (showActions || reserveActionSpace)
              SizedBox(
                height: actionRowHeight,
                child: Visibility(
                  visible: showActions,
                  maintainState: reserveActionSpace,
                  maintainAnimation: reserveActionSpace,
                  maintainSize: reserveActionSpace,
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      child: showActions && canAct
                          ? _ActionButtons(
                              model: model,
                              showBetInput: showBetInput,
                              betCtrl: actionControls!.betCtrl,
                              onToggleBetInput: actionControls.onToggleBetInput,
                              onCloseBetInput: actionControls.onCloseBetInput,
                              bb: _resolveBigBlind(),
                            )
                          : showActions
                              ? _WaitingIndicator(model: model)
                              : const SizedBox.shrink(),
                    ),
                  ),
                ),
              ),
            if (footer != null) ...[
              if (showActions || reserveActionSpace)
                SizedBox(height: sectionGap),
              footer!,
            ],
          ],
        );
        return Container(
          constraints: BoxConstraints(minHeight: mobileHeroPanelMinHeight(bp)),
          padding: EdgeInsets.only(
            left: PokerSpacing.sm,
            right: PokerSpacing.sm,
            top: topPadding,
            bottom: safeBottomPadding(context, minPadding: 6),
          ),
          decoration: const BoxDecoration(color: PokerColors.screenBg),
          child: Column(
            mainAxisSize: MainAxisSize.max,
            mainAxisAlignment: hasBottomSection
                ? MainAxisAlignment.spaceBetween
                : MainAxisAlignment.start,
            children: [
              headerSection,
              if (hasBottomSection)
                Padding(
                  padding: EdgeInsets.only(top: sectionGap),
                  child: bottomSection,
                ),
            ],
          ),
        );
      },
    );
  }
}

class _CompactHeroCards extends StatelessWidget {
  const _CompactHeroCards({required this.cards});
  final List<pr.Card> cards;

  @override
  Widget build(BuildContext context) {
    final theme = PokerThemeConfig.fromContext(context);
    final cw = (42.0 * theme.cardSizeMultiplier).clamp(24.0, 60.0).toDouble();
    final ch = cw * 1.4;
    final gap = (cw * 0.14).clamp(4.0, 8.0).toDouble();

    Widget buildCard(int index) {
      if (cards.length > index) {
        return CardFace(card: cards[index], cardTheme: theme.cardTheme);
      }
      return const CardBack();
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(width: cw, height: ch, child: buildCard(0)),
        SizedBox(width: gap),
        SizedBox(width: cw, height: ch, child: buildCard(1)),
      ],
    );
  }
}

class _ShowCardsDockToggle extends StatelessWidget {
  const _ShowCardsDockToggle({
    required this.model,
    this.compact = false,
  });

  final PokerModel model;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final hasCards = (model.me?.hand.isNotEmpty ?? false) ||
        model.myHoleCardsCache.isNotEmpty;
    if (!hasCards) return const SizedBox.shrink();

    final showing = model.me?.cardsRevealed ?? false;
    final accent = showing ? PokerColors.warning : PokerColors.textPrimary;
    final label = showing ? 'Hide Cards' : 'Show Cards';

    return Tooltip(
      message: label,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const Key('poker-show-cards-toggle'),
          onTap: () {
            if (showing) {
              model.hideCards();
            } else {
              model.showCards();
            }
          },
          borderRadius: BorderRadius.circular(10),
          child: Container(
            padding: EdgeInsets.symmetric(
              horizontal: compact ? 10 : 12,
              vertical: compact ? 7 : 8,
            ),
            decoration: BoxDecoration(
              color: PokerColors.overlayLight,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(
                color: showing ? PokerColors.warning : PokerColors.borderSubtle,
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  showing ? Icons.visibility_off : Icons.visibility,
                  size: compact ? 14 : 16,
                  color: accent,
                ),
                SizedBox(width: compact ? 5 : 6),
                Text(
                  label,
                  style: PokerTypography.labelSmall.copyWith(
                    color: accent,
                    fontSize: compact ? 10.5 : 11,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class PokerLastHandButton extends StatelessWidget {
  const PokerLastHandButton(
      {super.key,
      required this.active,
      required this.onTap,
      this.compact = false});
  final bool active;
  final VoidCallback onTap;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final accent = active ? PokerColors.warning : PokerColors.textSecondary;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: EdgeInsets.symmetric(
            horizontal: compact ? 8 : 10,
            vertical: 8,
          ),
          decoration: BoxDecoration(
            color: PokerColors.overlayLight,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: PokerColors.borderSubtle),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.history, color: accent, size: 16),
              if (!compact) ...[
                const SizedBox(width: 5),
                Text('Last Hand',
                    style: PokerTypography.labelSmall.copyWith(color: accent)),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _WaitingIndicator extends StatelessWidget {
  const _WaitingIndicator({required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(
          horizontal: PokerSpacing.lg, vertical: PokerSpacing.sm),
      decoration: BoxDecoration(
        color: PokerColors.overlayMedium,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        model.autoAdvanceAllIn ? 'All-in' : 'Waiting...',
        style: PokerTypography.bodyMedium,
      ),
    );
  }
}

class _ActionButtons extends StatelessWidget {
  const _ActionButtons({
    required this.model,
    required this.showBetInput,
    required this.betCtrl,
    required this.onToggleBetInput,
    required this.onCloseBetInput,
    required this.bb,
  });

  final PokerModel model;
  final bool showBetInput;
  final TextEditingController betCtrl;
  final VoidCallback onToggleBetInput;
  final VoidCallback onCloseBetInput;
  final int bb;

  @override
  Widget build(BuildContext context) {
    final g = model.game;
    final me = model.me;
    final currentBet = g?.currentBet ?? 0;
    final myBet = me?.currentBet ?? 0;
    final canCheck = myBet >= currentBet;
    final toCall = (currentBet - myBet) > 0 ? (currentBet - myBet) : 0;
    final isRaise = currentBet > 0 && myBet < currentBet;
    final myBalance = me?.balance ?? 0;
    final wouldBeAllIn = myBalance > 0 && myBalance <= (currentBet - myBet);

    if (showBetInput) {
      return _BetInputRow(
        model: model,
        betCtrl: betCtrl,
        currentBet: currentBet,
        myBet: myBet,
        bb: bb,
        isRaise: isRaise,
        onClose: onCloseBetInput,
      );
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _ActionButton(
          label: 'Fold',
          icon: Icons.cancel_outlined,
          onPressed: model.fold,
          color: PokerColors.foldBtn,
        ),
        const SizedBox(width: PokerSpacing.sm),
        if (canCheck)
          _ActionButton(
            label: 'Check',
            icon: Icons.check,
            onPressed: model.check,
            color: PokerColors.checkBtn,
          )
        else
          _ActionButton(
            label: 'Call${toCall > 0 ? ' $toCall' : ''}',
            icon: Icons.call_made,
            onPressed: model.callBet,
            color: PokerColors.checkBtn,
          ),
        const SizedBox(width: PokerSpacing.sm),
        _ActionButton(
          label: isRaise ? (wouldBeAllIn ? 'All-in' : 'Raise') : 'Bet',
          icon: Icons.arrow_upward,
          onPressed: () {
            if (betCtrl.text.isEmpty) _seedDefault(betCtrl, bb, currentBet);
            onToggleBetInput();
          },
          color: PokerColors.betBtn,
        ),
      ],
    );
  }

  static void _seedDefault(TextEditingController ctrl, int bb, int currentBet) {
    final threeBB = bb * 3;
    final target =
        (bb > 0 && currentBet >= threeBB) ? (currentBet * 3) : threeBB;
    ctrl.text = target.toString();
  }
}

class _BetInputRow extends StatelessWidget {
  const _BetInputRow({
    required this.model,
    required this.betCtrl,
    required this.currentBet,
    required this.myBet,
    required this.bb,
    required this.isRaise,
    required this.onClose,
  });

  final PokerModel model;
  final TextEditingController betCtrl;
  final int currentBet, myBet, bb;
  final bool isRaise;
  final VoidCallback onClose;

  Future<void> _submitBet(BuildContext context) async {
    final raw = betCtrl.text.trim();
    final entered = int.tryParse(raw) ?? 0;
    if (entered <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Enter a valid bet amount')),
      );
      return;
    }

    final myBalance = model.me?.balance ?? 0;
    final totalAmt = normalizeBetInputToTotal(
      entered: entered,
      myBet: myBet,
      myBalance: myBalance,
    );
    final shortAllIn = isShortAllInTarget(
      totalTarget: totalAmt,
      myBet: myBet,
      myBalance: myBalance,
      currentBet: currentBet,
    );

    if (currentBet > 0 && totalAmt < currentBet && !shortAllIn) {
      final minRaise = currentBet - myBet;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
            content: Text(
                'Must add at least $minRaise to call ($currentBet total)')),
      );
      return;
    }

    final ok = await model.makeBet(totalAmt);
    if (!ok && model.errorMessage.isNotEmpty && context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(model.errorMessage)));
      return;
    }
    onClose();
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final scale = buttonScale(bp);
    final threeBB = bb * 3;
    final presetLabel = (bb > 0 && currentBet >= threeBB) ? '3x Bet' : '3x BB';

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: 110 * scale,
          child: TextField(
            controller: betCtrl,
            keyboardType: TextInputType.number,
            style: PokerTypography.bodyMedium,
            decoration: InputDecoration(
              labelText: isRaise ? 'Raise to' : 'Bet',
              isDense: true,
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            ),
            onSubmitted: (_) => _submitBet(context),
          ),
        ),
        const SizedBox(width: PokerSpacing.sm),
        // Quick-bet presets
        _QuickBetChip(
          label: presetLabel,
          onTap: bb > 0
              ? () {
                  final target = (bb > 0 && currentBet >= threeBB)
                      ? (currentBet * 3)
                      : threeBB;
                  betCtrl.text = target.toString();
                }
              : null,
        ),
        const SizedBox(width: PokerSpacing.xs),
        _QuickBetChip(
          label: 'Pot',
          onTap: () {
            final pot = model.game?.pot ?? 0;
            if (pot > 0) betCtrl.text = pot.toString();
          },
        ),
        const SizedBox(width: PokerSpacing.sm),
        Builder(builder: (context) {
          final meBal = model.me?.balance ?? 0;
          final entered = int.tryParse(betCtrl.text.trim()) ?? 0;
          final target = entered > 0
              ? normalizeBetInputToTotal(
                  entered: entered,
                  myBet: myBet,
                  myBalance: meBal,
                )
              : currentBet;
          final myTotal = meBal + myBet;
          final isAllIn = target >= myTotal && myTotal > 0;
          final label = isAllIn ? 'All-in' : (isRaise ? 'Raise' : 'Bet');
          return _ActionButton(
            label: label,
            onPressed: () => _submitBet(context),
            color: PokerColors.betBtn,
          );
        }),
        const SizedBox(width: PokerSpacing.xs),
        TextButton(
          onPressed: onClose,
          child: const Text('Cancel', style: PokerTypography.labelSmall),
        ),
      ],
    );
  }
}

class _QuickBetChip extends StatelessWidget {
  const _QuickBetChip({required this.label, this.onTap});
  final String label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(16),
        child: Container(
          padding: const EdgeInsets.symmetric(
            horizontal: PokerSpacing.sm,
            vertical: PokerSpacing.xs,
          ),
          decoration: BoxDecoration(
            color: PokerColors.overlayLight,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: PokerColors.borderSubtle),
          ),
          child: Text(
            label,
            style: PokerTypography.labelSmall,
          ),
        ),
      ),
    );
  }
}
