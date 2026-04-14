import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

List<pr.Card> _dockCardsForModel(PokerModel model) {
  if (model.isWatching) return const <pr.Card>[];
  final me = model.me;
  return (me?.hand.isNotEmpty ?? false) ? me!.hand : model.heroShowdownHand;
}

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
    final cards = _dockCardsForModel(model);
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
          final sectionTopMargin = showBetInput
              ? 0.0
              : (tightDesktopHeight ? PokerSpacing.sm : PokerSpacing.xl);
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
                physics: showBetInput
                    ? const NeverScrollableScrollPhysics()
                    : const ClampingScrollPhysics(),
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

          return SingleChildScrollView(
            physics: const ClampingScrollPhysics(),
            child: ConstrainedBox(
              constraints: BoxConstraints(minHeight: constraints.maxHeight),
              child: Column(
                mainAxisSize: MainAxisSize.max,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (hasCards)
                    Align(
                      alignment: Alignment.topRight,
                      child: headerPanel,
                    ),
                  if (hasBottomSection)
                    Padding(
                      padding: EdgeInsets.only(
                        top: hasCards ? sectionTopMargin : sectionTopMargin + 2,
                      ),
                      child: bottomSection,
                    ),
                ],
              ),
            ),
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
    this.onShowLastHand,
    this.reserveActionSpace = false,
    this.footer,
  })  : _actionControls = _ActionControls(
          betCtrl: betCtrl,
          onToggleBetInput: onToggleBetInput,
          onCloseBetInput: onCloseBetInput,
        ),
        showActions = true;

  const MobileHeroActionPanel.passive({
    super.key,
    required this.model,
    this.hasLastShowdown = false,
    this.onShowLastHand,
    this.reserveActionSpace = false,
    this.footer,
  })  : showBetInput = false,
        _actionControls = null,
        showActions = false;

  final PokerModel model;
  final bool showBetInput;
  final _ActionControls? _actionControls;
  final bool hasLastShowdown;
  final VoidCallback? onShowLastHand;
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
    final cards = _dockCardsForModel(model);
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
            final hasLastHandButton = hasLastShowdown && onShowLastHand != null;
            final uiSpec = PokerUiSpec.fromContext(context);
            final cardWidth = uiSpec.heroDockCardSize.width;
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
            final cardsRow = hasCards
                ? _CompactHeroCards(cards: cards)
                : const SizedBox.shrink();
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
                          onTap: onShowLastHand!,
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
                  if (hasCards) cardsRow,
                  const Spacer(),
                  if (hasTrailingControls) trailingControls,
                ],
              );
            }

            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (hasCards) cardsRow,
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
              ConstrainedBox(
                constraints: BoxConstraints(minHeight: actionRowHeight),
                child: Visibility(
                  visible: showActions,
                  maintainState: reserveActionSpace,
                  maintainAnimation: reserveActionSpace,
                  maintainSize: reserveActionSpace,
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      physics: showBetInput
                          ? const NeverScrollableScrollPhysics()
                          : const ClampingScrollPhysics(),
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
          ),
          decoration: const BoxDecoration(color: PokerColors.screenBg),
          child: LayoutBuilder(
            builder: (context, innerConstraints) {
              final safeBottom = safeBottomPadding(context, minPadding: 6);
              final maxH = innerConstraints.maxHeight;
              final minMainRegion =
                  math.max(0.0, maxH.isFinite ? maxH - safeBottom : 0.0);

              final mainColumn = Column(
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
              );

              final scrollChild = Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  ConstrainedBox(
                    constraints: BoxConstraints(
                      minWidth: innerConstraints.maxWidth,
                      minHeight: minMainRegion,
                    ),
                    child: mainColumn,
                  ),
                  SizedBox(height: safeBottom),
                ],
              );

              if (!maxH.isFinite) {
                return scrollChild;
              }
              return SingleChildScrollView(
                physics: const ClampingScrollPhysics(),
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    minWidth: innerConstraints.maxWidth,
                    minHeight: maxH,
                  ),
                  child: scrollChild,
                ),
              );
            },
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
    final uiSpec = PokerUiSpec.fromContext(context);
    final theme = PokerThemeConfig.fromSpec(uiSpec);
    final cw = uiSpec.heroDockCardSize.width;
    final ch = uiSpec.heroDockCardSize.height;
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
    final hasCards = _dockCardsForModel(model).isNotEmpty;
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
      required this.onTap,
      this.compact = false,
      this.active = false});
  final VoidCallback onTap;
  final bool compact;
  final bool active;

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
    final isWatchingOnly = model.isWatching;
    final label = isWatchingOnly
        ? 'Watching only'
        : (model.autoAdvanceAllIn ? 'All-in' : 'Waiting...');
    return LayoutBuilder(
      builder: (context, constraints) {
        final compactLayout =
            constraints.maxHeight.isFinite && constraints.maxHeight <= 48;
        final indicator = Container(
          padding: EdgeInsets.symmetric(
            horizontal: compactLayout ? 12 : PokerSpacing.lg,
            vertical: compactLayout ? 6 : PokerSpacing.sm,
          ),
          decoration: BoxDecoration(
            color: PokerColors.overlayMedium,
            borderRadius: BorderRadius.circular(compactLayout ? 10 : 12),
          ),
          child: Text(
            label,
            style: compactLayout
                ? PokerTypography.labelLarge.copyWith(fontSize: 12)
                : PokerTypography.bodyMedium,
          ),
        );

        if (!isWatchingOnly) return indicator;

        final stopWatchingButton = OutlinedButton.icon(
          onPressed: model.leaveTable,
          icon: Icon(Icons.exit_to_app, size: compactLayout ? 14 : 16),
          label: Text(compactLayout ? 'Stop' : 'Stop Watching'),
          style: OutlinedButton.styleFrom(
            foregroundColor: PokerColors.danger,
            side: BorderSide(color: PokerColors.danger.withValues(alpha: 0.55)),
            padding: EdgeInsets.symmetric(horizontal: compactLayout ? 10 : 12),
            minimumSize: Size(0, compactLayout ? 32 : 36),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            visualDensity: compactLayout
                ? const VisualDensity(horizontal: -2, vertical: -2)
                : VisualDensity.standard,
          ),
        );

        if (compactLayout) {
          return Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              indicator,
              const SizedBox(width: PokerSpacing.sm),
              stopWatchingButton,
            ],
          );
        }

        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            indicator,
            const SizedBox(height: PokerSpacing.sm),
            stopWatchingButton,
          ],
        );
      },
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
    final minRaise = g?.minRaise ?? 0;
    final maxRaise = g?.maxRaise ?? 0;
    final myBet = me?.currentBet ?? 0;
    final myBalance = me?.balance ?? 0;
    final canCheck = myBet >= currentBet;
    final toCall = (currentBet - myBet) > 0 ? (currentBet - myBet) : 0;
    final isRaise = currentBet > 0 && myBet < currentBet;
    final wouldBeAllIn = myBalance > 0 && myBalance <= (currentBet - myBet);
    final allInOnly = hasShortAllInOnlyBetOrRaiseOption(
      currentBet: currentBet,
      minRaise: minRaise,
      maxRaise: maxRaise,
      bigBlind: bb,
    );

    if (showBetInput) {
      return _BetInputRow(
        model: model,
        betCtrl: betCtrl,
        currentBet: currentBet,
        minRaise: minRaise,
        maxRaise: maxRaise,
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
          label: (allInOnly || wouldBeAllIn)
              ? 'All-in'
              : (isRaise ? 'Raise' : 'Bet'),
          icon: Icons.arrow_upward,
          onPressed: () {
            _seedDefault(
              betCtrl,
              currentBet: currentBet,
              minRaise: minRaise,
              maxRaise: maxRaise,
              bigBlind: bb,
            );
            onToggleBetInput();
          },
          color: PokerColors.betBtn,
        ),
      ],
    );
  }

  static void _seedDefault(
    TextEditingController ctrl, {
    required int currentBet,
    required int minRaise,
    required int maxRaise,
    required int bigBlind,
  }) {
    final target = initialBetOrRaiseTotal(
      currentBet: currentBet,
      minRaise: minRaise,
      maxRaise: maxRaise,
      bigBlind: bigBlind,
    );
    ctrl.text = target > 0 ? target.toString() : '';
  }
}

class _BetInputRow extends StatelessWidget {
  const _BetInputRow({
    required this.model,
    required this.betCtrl,
    required this.currentBet,
    required this.minRaise,
    required this.maxRaise,
    required this.myBet,
    required this.bb,
    required this.isRaise,
    required this.onClose,
  });

  final PokerModel model;
  final TextEditingController betCtrl;
  final int currentBet, minRaise, maxRaise, myBet, bb;
  final bool isRaise;
  final VoidCallback onClose;

  int _initialTarget() {
    return initialBetOrRaiseTotal(
      currentBet: currentBet,
      minRaise: minRaise,
      maxRaise: maxRaise,
      bigBlind: bb,
    );
  }

  int? _enteredTarget() {
    final raw = betCtrl.text.trim();
    if (raw.isEmpty) return null;
    return int.tryParse(raw);
  }

  int _currentTarget({bool clampToMax = true}) {
    final entered = _enteredTarget();
    if (entered != null && entered > 0) {
      if (clampToMax && maxRaise > 0) {
        return math.min(entered, maxRaise);
      }
      return entered;
    }
    return _initialTarget();
  }

  int _sliderTarget() {
    final entered = int.tryParse(betCtrl.text.trim()) ?? 0;
    if (entered > 0) {
      return clampBetTargetToLegalRange(
        target: entered,
        currentBet: currentBet,
        minRaise: minRaise,
        maxRaise: maxRaise,
        bigBlind: bb,
      );
    }
    return _initialTarget();
  }

  void _syncLockedTarget(TextEditingValue value, int target) {
    final text = target > 0 ? target.toString() : '';
    if (value.text == text) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (betCtrl.text == text) return;
      betCtrl.value = TextEditingValue(
        text: text,
        selection: TextSelection.collapsed(offset: text.length),
      );
    });
  }

  Future<void> _submitBet(BuildContext context) async {
    final entered = _currentTarget();
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

    final validationError = validateBetOrRaiseTarget(
      totalTarget: totalAmt,
      currentBet: currentBet,
      myBet: myBet,
      myBalance: myBalance,
      minRaise: minRaise,
      bigBlind: bb,
    );

    if (validationError != null && !shortAllIn) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(validationError)),
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
    final legalMin = legalMinimumBetOrRaiseTotal(
      currentBet: currentBet,
      minRaise: minRaise,
      bigBlind: bb,
    );
    final shortAllInOnly = hasShortAllInOnlyBetOrRaiseOption(
      currentBet: currentBet,
      minRaise: minRaise,
      maxRaise: maxRaise,
      bigBlind: bb,
    );
    final sliderMin = shortAllInOnly ? maxRaise : legalMin;
    final presetTarget = suggestedBetOrRaiseTotal(
      currentBet: currentBet,
      minRaise: minRaise,
      maxRaise: maxRaise,
      bigBlind: bb,
    );
    final showPresetButton = !shortAllInOnly && presetTarget > 0;
    final suggestedActionLabel = currentBet > 0 ? '3x' : '3xBB';

    return LayoutBuilder(
      builder: (context, constraints) {
        final compactHeight =
            constraints.maxHeight.isFinite && constraints.maxHeight <= 56;
        final compactWidth =
            constraints.maxWidth.isFinite && constraints.maxWidth < 320 * scale;
        final compactLayout = compactHeight || compactWidth;
        final desiredWidth = 340 * scale;
        final editorWidth = constraints.maxWidth.isFinite
            ? math.min(desiredWidth, constraints.maxWidth)
            : desiredWidth;

        return SizedBox(
          width: editorWidth,
          child: ValueListenableBuilder<TextEditingValue>(
            valueListenable: betCtrl,
            builder: (context, value, _) {
              if (shortAllInOnly) {
                _syncLockedTarget(value, maxRaise);
              }
              final target = _currentTarget(clampToMax: false);
              final sliderTarget = _sliderTarget();
              final meBal = model.me?.balance ?? 0;
              final myTotal = meBal + myBet;
              final normalizedTarget = normalizeBetInputToTotal(
                entered: target,
                myBet: myBet,
                myBalance: meBal,
              );
              final isAllIn = normalizedTarget >= myTotal && myTotal > 0;
              final label = (shortAllInOnly || isAllIn)
                  ? 'All-in'
                  : (isRaise ? 'Raise' : 'Bet');
              final displayTarget = shortAllInOnly ? maxRaise : target;
              final sliderDisplayMin =
                  shortAllInOnly ? 0.0 : sliderMin.toDouble();
              final sliderDisplayMax = shortAllInOnly
                  ? 1.0
                  : (maxRaise > 0 ? maxRaise : sliderMin).toDouble();
              final sliderDisplayValue =
                  shortAllInOnly ? 1.0 : sliderTarget.toDouble();
              final sliderEnabled =
                  !shortAllInOnly && sliderDisplayMax > sliderDisplayMin;

              return Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      Expanded(
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: PokerSpacing.sm),
                          decoration: BoxDecoration(
                            color: PokerColors.overlayLight,
                            borderRadius: BorderRadius.circular(
                              compactLayout ? 12 : 14,
                            ),
                            border: Border.all(color: PokerColors.borderSubtle),
                          ),
                          child: TextField(
                            controller: betCtrl,
                            readOnly: shortAllInOnly,
                            keyboardType: TextInputType.number,
                            textInputAction: TextInputAction.done,
                            inputFormatters: [
                              FilteringTextInputFormatter.digitsOnly,
                            ],
                            onSubmitted: (_) => _submitBet(context),
                            style: compactLayout
                                ? PokerTypography.labelLarge
                                    .copyWith(fontSize: 12)
                                : PokerTypography.titleMedium,
                            decoration: InputDecoration(
                              isDense: true,
                              border: InputBorder.none,
                              contentPadding: EdgeInsets.symmetric(
                                vertical: compactLayout ? 8 : 10,
                              ),
                              hintText: _initialTarget().toString(),
                              hintStyle: (compactLayout
                                      ? PokerTypography.labelLarge
                                          .copyWith(fontSize: 12)
                                      : PokerTypography.titleMedium)
                                  .copyWith(
                                color: PokerColors.textMuted,
                              ),
                              prefixText: shortAllInOnly
                                  ? 'All-in '
                                  : (isRaise ? 'Raise to ' : 'Bet '),
                              prefixStyle: (compactLayout
                                      ? PokerTypography.labelLarge
                                          .copyWith(fontSize: 12)
                                      : PokerTypography.titleMedium)
                                  .copyWith(
                                color: PokerColors.textSecondary,
                              ),
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: PokerSpacing.sm),
                      _ActionButton(
                        label: label,
                        onPressed: () => _submitBet(context),
                        color: PokerColors.betBtn,
                      ),
                      const SizedBox(width: PokerSpacing.xs),
                      compactLayout
                          ? Material(
                              color: Colors.transparent,
                              child: InkWell(
                                onTap: onClose,
                                borderRadius: BorderRadius.circular(10),
                                child: Container(
                                  padding: const EdgeInsets.all(8),
                                  decoration: BoxDecoration(
                                    color: PokerColors.overlayLight,
                                    borderRadius: BorderRadius.circular(10),
                                    border: Border.all(
                                      color: PokerColors.borderSubtle,
                                    ),
                                  ),
                                  child: const Icon(
                                    Icons.close_rounded,
                                    size: 16,
                                    color: PokerColors.textSecondary,
                                  ),
                                ),
                              ),
                            )
                          : TextButton(
                              onPressed: onClose,
                              child: const Text(
                                'Cancel',
                                style: PokerTypography.labelSmall,
                              ),
                            ),
                    ],
                  ),
                  const SizedBox(height: PokerSpacing.xs),
                  SliderTheme(
                    data: SliderTheme.of(context).copyWith(
                      activeTrackColor: PokerColors.betBtn,
                      inactiveTrackColor:
                          PokerColors.surfaceBright.withValues(alpha: 0.9),
                      thumbColor: PokerColors.betBtn,
                      disabledActiveTrackColor:
                          PokerColors.betBtn.withValues(alpha: 0.5),
                      disabledInactiveTrackColor:
                          PokerColors.surfaceBright.withValues(alpha: 0.65),
                      overlayColor: PokerColors.betBtn.withValues(alpha: 0.16),
                      trackHeight: compactLayout ? 5 : 6,
                    ),
                    child: Slider(
                      key: const Key('bet-amount-slider'),
                      allowedInteraction: SliderInteraction.tapAndSlide,
                      value: sliderDisplayValue.clamp(
                        sliderDisplayMin,
                        sliderDisplayMax,
                      ),
                      min: sliderDisplayMin,
                      max: sliderDisplayMax,
                      label: shortAllInOnly
                          ? 'All-in $maxRaise'
                          : '$displayTarget',
                      onChanged: sliderEnabled
                          ? (raw) {
                              final snapped = snapBetTargetToStep(
                                target: raw.round(),
                                currentBet: currentBet,
                                minRaise: minRaise,
                                maxRaise: maxRaise,
                                bigBlind: bb,
                              );
                              betCtrl.text = snapped.toString();
                            }
                          : null,
                    ),
                  ),
                  _SliderLegend(
                    minLabel:
                        shortAllInOnly ? 'All-in $maxRaise' : 'Min $legalMin',
                    presetLabel: showPresetButton ? suggestedActionLabel : null,
                    maxLabel: shortAllInOnly
                        ? ''
                        : (maxRaise > 0 ? 'Max $maxRaise' : 'Open size'),
                    compact: compactLayout,
                    onPresetPressed: showPresetButton
                        ? () {
                            betCtrl.text = snapBetTargetToStep(
                              target: presetTarget,
                              currentBet: currentBet,
                              minRaise: minRaise,
                              maxRaise: maxRaise,
                              bigBlind: bb,
                            ).toString();
                          }
                        : null,
                  ),
                ],
              );
            },
          ),
        );
      },
    );
  }
}

class _SliderLegend extends StatelessWidget {
  const _SliderLegend({
    required this.minLabel,
    required this.presetLabel,
    required this.maxLabel,
    required this.compact,
    required this.onPresetPressed,
  });

  final String minLabel;
  final String? presetLabel;
  final String maxLabel;
  final bool compact;
  final VoidCallback? onPresetPressed;

  @override
  Widget build(BuildContext context) {
    final style = PokerTypography.labelSmall.copyWith(
      color: PokerColors.textSecondary,
      fontSize: compact ? 10.5 : 11,
    );
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: compact ? 4 : 12),
      child: Row(
        children: [
          Expanded(
            child: Text(
              minLabel,
              overflow: TextOverflow.ellipsis,
              style: style,
            ),
          ),
          if (presetLabel != null) ...[
            SizedBox(width: compact ? 6 : 10),
            _SliderPresetChip(
              label: presetLabel!,
              compact: compact,
              onTap: onPresetPressed,
            ),
            SizedBox(width: compact ? 6 : 10),
          ],
          Expanded(
            child: Text(
              maxLabel,
              textAlign: TextAlign.right,
              overflow: TextOverflow.ellipsis,
              style: style,
            ),
          ),
        ],
      ),
    );
  }
}

class _SliderPresetChip extends StatelessWidget {
  const _SliderPresetChip({
    required this.label,
    required this.compact,
    required this.onTap,
  });

  final String label;
  final bool compact;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        key: const Key('raise-3x-button'),
        onTap: onTap,
        borderRadius: BorderRadius.circular(999),
        child: Ink(
          padding: EdgeInsets.symmetric(
            horizontal: compact ? 8 : 10,
            vertical: compact ? 5 : 6,
          ),
          decoration: BoxDecoration(
            color: PokerColors.betBtn.withValues(alpha: 0.14),
            borderRadius: BorderRadius.circular(999),
            border: Border.all(
              color: PokerColors.betBtn.withValues(alpha: 0.45),
            ),
          ),
          child: Text(
            label,
            style: PokerTypography.labelSmall.copyWith(
              color: PokerColors.betBtn,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
      ),
    );
  }
}
