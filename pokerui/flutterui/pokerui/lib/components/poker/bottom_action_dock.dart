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
    this.dense = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final Color color;
  final IconData? icon;
  final bool dense;

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final scale = buttonScale(bp);
    final densityScale = dense ? 0.82 : 1.0;
    return ElevatedButton(
      onPressed: onPressed,
      style: ElevatedButton.styleFrom(
        backgroundColor: color,
        foregroundColor: Colors.white,
        padding: EdgeInsets.symmetric(
          horizontal: 20 * scale * densityScale,
          vertical: 12 * scale * densityScale,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12 * scale * densityScale),
        ),
        elevation: 2,
        shadowColor: color.withValues(alpha: 0.3),
        minimumSize: dense ? Size(0, 32 * scale) : null,
        tapTargetSize: dense
            ? MaterialTapTargetSize.shrinkWrap
            : MaterialTapTargetSize.padded,
        visualDensity: dense
            ? const VisualDensity(horizontal: -2, vertical: -2)
            : VisualDensity.standard,
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 16 * scale * densityScale),
            SizedBox(width: 6 * scale * densityScale),
          ],
          Text(
            label,
            style: TextStyle(
              fontSize: 14 * scale * densityScale,
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
          final sectionTopMargin = showBetInput
              ? 0.0
              : (tightDesktopHeight ? PokerSpacing.sm : PokerSpacing.xl);
          final actionControls = _actionControls;
          final actionChild = showActions && canAct
              ? _ActionButtons(
                  model: model,
                  showBetInput: showBetInput,
                  betCtrl: actionControls!.betCtrl,
                  onToggleBetInput: actionControls.onToggleBetInput,
                  onCloseBetInput: actionControls.onCloseBetInput,
                  bb: _resolveBigBlind(),
                  availableWidth: showBetInput ? constraints.maxWidth : null,
                )
              : showActions
                  ? _WaitingIndicator(model: model)
                  : const SizedBox.shrink();
          final actions = Visibility(
            visible: showActions,
            maintainState: reserveActionSpace,
            maintainAnimation: reserveActionSpace,
            maintainSize: reserveActionSpace,
            child: Align(
              alignment: Alignment.centerRight,
              child: showBetInput
                  ? SizedBox(
                      width: constraints.maxWidth,
                      child: Align(
                        alignment: Alignment.centerRight,
                        child: actionChild,
                      ),
                    )
                  : SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      physics: const ClampingScrollPhysics(),
                      child: actionChild,
                    ),
            ),
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
                  if (hasBottomSection)
                    Padding(
                      padding: EdgeInsets.only(top: sectionTopMargin + 2),
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
    final heroCardsVisibleHeightFactor = showBetInput ? 0.5 : 1.0;

    return LayoutBuilder(
      builder: (context, panelConstraints) {
        final availableHeight = panelConstraints.maxHeight;
        final panelW = panelConstraints.maxWidth;
        final betInputAvailableWidth = showBetInput ? panelW : null;
        final tightVertical =
            availableHeight.isFinite && availableHeight <= 152.0;
        final sectionGap = tightVertical ? 6.0 : PokerSpacing.sm;
        final cardsToActionsGap = showBetInput
            ? (tightVertical ? PokerSpacing.md : 14.0)
            : (tightVertical ? 10.0 : PokerSpacing.md);
        final trailingGap = tightVertical ? 4.0 : 6.0;
        final trailingSectionGap = tightVertical ? 6.0 : PokerSpacing.sm;
        final topPadding = tightVertical ? 6.0 : PokerSpacing.sm;
        final actionChild = showActions && canAct
            ? _ActionButtons(
                model: model,
                showBetInput: showBetInput,
                betCtrl: actionControls!.betCtrl,
                onToggleBetInput: actionControls.onToggleBetInput,
                onCloseBetInput: actionControls.onCloseBetInput,
                bb: _resolveBigBlind(),
                availableWidth: betInputAvailableWidth,
                preferFullWidthBetInput: true,
              )
            : showActions
                ? _WaitingIndicator(model: model)
                : const SizedBox.shrink();
        final headerSection = LayoutBuilder(
          builder: (context, constraints) {
            final hasLastHandButton = hasLastShowdown && onShowLastHand != null;
            final cardMetrics = _CompactHeroCardsMetrics.fromContext(
              context,
              visibleHeightFactor: heroCardsVisibleHeightFactor,
            );
            final cardsClusterWidth = hasCards ? cardMetrics.totalWidth : 0.0;
            final trailingWidth = hasLastHandButton ? 92.0 : 0.0;
            final stackedHeader =
                constraints.maxWidth < cardsClusterWidth + trailingWidth + 36.0;
            final cardsRow = hasCards
                ? _CompactHeroCards(
                    cards: cards,
                    model: model,
                    visibleHeightFactor: heroCardsVisibleHeightFactor,
                  )
                : const SizedBox.shrink();
            final hasTrailingControls = hasLastHandButton;
            final trailingControls = hasTrailingControls
                ? Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      PokerLastHandButton(
                        onTap: onShowLastHand!,
                        compact: true,
                      ),
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
                    alignment: Alignment.center,
                    child: showBetInput
                        ? SizedBox(
                            width: panelW,
                            child: Align(
                              alignment: Alignment.center,
                              child: actionChild,
                            ),
                          )
                        : SingleChildScrollView(
                            scrollDirection: Axis.horizontal,
                            physics: const ClampingScrollPhysics(),
                            child: actionChild,
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
          key: const Key('mobile-hero-action-panel'),
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
                      padding: EdgeInsets.only(top: cardsToActionsGap),
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

class _CompactHeroCardsMetrics {
  const _CompactHeroCardsMetrics({
    required this.cardWidth,
    required this.cardHeight,
    required this.visibleHeight,
    required this.visibleHeightFactor,
    required this.gap,
    required this.cardsWidth,
    required this.affordanceSize,
    required this.affordanceGap,
  });

  factory _CompactHeroCardsMetrics.fromContext(
    BuildContext context, {
    double visibleHeightFactor = 1.0,
  }) {
    final uiSpec = PokerUiSpec.fromContext(context);
    final cardWidth = uiSpec.heroDockCardSize.width;
    final cardHeight = uiSpec.heroDockCardSize.height;
    final clampedVisibleHeightFactor = visibleHeightFactor.clamp(0.0, 1.0);
    final visibleHeight = cardHeight * clampedVisibleHeightFactor;
    final gap = (cardWidth * 0.14).clamp(4.0, 8.0).toDouble();
    final cardsWidth = (cardWidth * 2) + gap;
    final baseAffordanceSize = (cardWidth * 0.42).clamp(24.0, 28.0).toDouble();
    final affordanceSize = math.min(
      baseAffordanceSize,
      math.max(18.0, visibleHeight - 2.0),
    );

    return _CompactHeroCardsMetrics(
      cardWidth: cardWidth,
      cardHeight: cardHeight,
      visibleHeight: visibleHeight,
      visibleHeightFactor: clampedVisibleHeightFactor.toDouble(),
      gap: gap,
      cardsWidth: cardsWidth,
      affordanceSize: affordanceSize,
      affordanceGap: 8.0,
    );
  }

  final double cardWidth;
  final double cardHeight;
  final double visibleHeight;
  final double visibleHeightFactor;
  final double gap;
  final double cardsWidth;
  final double affordanceSize;
  final double affordanceGap;

  double get totalWidth => cardsWidth + affordanceGap + affordanceSize;
}

class _CompactHeroCards extends StatefulWidget {
  const _CompactHeroCards({
    required this.cards,
    required this.model,
    this.visibleHeightFactor = 1.0,
  });
  final List<pr.Card> cards;
  final PokerModel model;
  final double visibleHeightFactor;

  @override
  State<_CompactHeroCards> createState() => _CompactHeroCardsState();
}

class _CompactHeroCardsState extends State<_CompactHeroCards> {
  bool _hovering = false;

  void _setHovering(bool value) {
    if (!mounted || _hovering == value) return;
    setState(() => _hovering = value);
  }

  @override
  Widget build(BuildContext context) {
    final uiSpec = PokerUiSpec.fromContext(context);
    final metrics = _CompactHeroCardsMetrics.fromContext(
      context,
      visibleHeightFactor: widget.visibleHeightFactor,
    );
    final theme = PokerThemeConfig.fromSpec(uiSpec);

    Widget buildCard(int index) {
      if (widget.cards.length > index) {
        return CardFace(card: widget.cards[index], cardTheme: theme.cardTheme);
      }
      return const CardBack();
    }

    final actionLabel =
        widget.model.me?.cardsRevealed ?? false ? 'Hide Cards' : 'Show Cards';

    return SizedBox(
      key: const Key('poker-hero-cards-cluster'),
      width: metrics.totalWidth,
      height: metrics.visibleHeight,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned(
            left: 0,
            top: 0,
            child: ClipRect(
              child: Align(
                alignment: Alignment.topLeft,
                heightFactor: metrics.visibleHeightFactor,
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    SizedBox(
                      width: metrics.cardWidth,
                      height: metrics.cardHeight,
                      child: buildCard(0),
                    ),
                    SizedBox(width: metrics.gap),
                    SizedBox(
                      width: metrics.cardWidth,
                      height: metrics.cardHeight,
                      child: buildCard(1),
                    ),
                  ],
                ),
              ),
            ),
          ),
          if (_hovering)
            Positioned(
              right: metrics.affordanceSize + metrics.affordanceGap + 4,
              top: math.max(0.0, (metrics.visibleHeight - 30) / 2),
              child: _ShowCardsInfoPill(label: actionLabel),
            ),
          Positioned(
            left: metrics.cardsWidth,
            top: math.max(
              0.0,
              (metrics.visibleHeight - metrics.affordanceSize) / 2,
            ),
            child: _ShowCardsAffordance(
              showing: widget.model.me?.cardsRevealed ?? false,
              size: metrics.affordanceSize,
              hitWidth: metrics.affordanceSize + metrics.affordanceGap,
              onEnter: () => _setHovering(true),
              onExit: () => _setHovering(false),
              onTap: () {
                if (widget.model.me?.cardsRevealed ?? false) {
                  widget.model.hideCards();
                } else {
                  widget.model.showCards();
                }
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _ShowCardsAffordance extends StatelessWidget {
  const _ShowCardsAffordance({
    required this.showing,
    required this.size,
    required this.hitWidth,
    required this.onEnter,
    required this.onExit,
    required this.onTap,
  });

  final bool showing;
  final double size;
  final double hitWidth;
  final VoidCallback onEnter;
  final VoidCallback onExit;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final accent = showing ? PokerColors.warning : PokerColors.textPrimary;

    return MouseRegion(
      onEnter: (_) => onEnter(),
      onExit: (_) => onExit(),
      child: GestureDetector(
        key: const Key('poker-show-cards-affordance'),
        behavior: HitTestBehavior.translucent,
        onTap: onTap,
        child: SizedBox(
          width: hitWidth,
          height: size,
          child: Align(
            alignment: Alignment.centerRight,
            child: Container(
              width: size,
              height: size,
              decoration: BoxDecoration(
                color: PokerColors.overlayLight,
                shape: BoxShape.circle,
                border: Border.all(
                  color:
                      showing ? PokerColors.warning : PokerColors.borderSubtle,
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withValues(alpha: 0.18),
                    blurRadius: 8,
                    offset: const Offset(0, 2),
                  ),
                ],
              ),
              child: Icon(
                showing ? Icons.visibility_off : Icons.visibility,
                size: size * 0.54,
                color: accent,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _ShowCardsInfoPill extends StatelessWidget {
  const _ShowCardsInfoPill({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: IgnorePointer(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: PokerColors.overlayMedium,
            borderRadius: BorderRadius.circular(999),
            border: Border.all(
              color: PokerColors.borderSubtle.withValues(alpha: 0.75),
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.18),
                blurRadius: 10,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Text(
            label,
            style: PokerTypography.labelSmall.copyWith(
              color: PokerColors.textPrimary,
              fontWeight: FontWeight.w700,
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
    this.availableWidth,
    this.preferFullWidthBetInput = false,
  });

  final PokerModel model;
  final bool showBetInput;
  final TextEditingController betCtrl;
  final VoidCallback onToggleBetInput;
  final VoidCallback onCloseBetInput;
  final int bb;
  final double? availableWidth;
  final bool preferFullWidthBetInput;

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
        availableWidth: availableWidth,
        preferFullWidth: preferFullWidthBetInput,
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
    this.availableWidth,
    this.preferFullWidth = false,
  });

  final PokerModel model;
  final TextEditingController betCtrl;
  final int currentBet, minRaise, maxRaise, myBet, bb;
  final bool isRaise;
  final VoidCallback onClose;
  final double? availableWidth;
  final bool preferFullWidth;

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
        final layoutWidth =
            availableWidth?.isFinite == true && availableWidth! > 0
                ? availableWidth!
                : constraints.maxWidth;
        final compactHeight =
            constraints.maxHeight.isFinite && constraints.maxHeight <= 56;
        final compactWidthThreshold = bp.isNarrow ? 340 * scale : 410 * scale;
        final compactWidth =
            layoutWidth.isFinite && layoutWidth < compactWidthThreshold;
        final compactLayout = compactHeight || compactWidth;
        final desktopBetChrome =
            !preferFullWidth && !compactLayout && (bp.isExpanded || bp.isWide);
        final fullWidthEditor = preferFullWidth || bp.isNarrow;
        final maxEditorWidth = compactLayout
            ? (fullWidthEditor ? layoutWidth : 400 * scale)
            : (fullWidthEditor
                ? layoutWidth
                : (desktopBetChrome ? 320 * scale : 470 * scale));
        final fallbackWidth = compactLayout
            ? 360 * scale
            : (desktopBetChrome ? 300 * scale : 420 * scale);
        final editorWidth = layoutWidth.isFinite
            ? math.min(maxEditorWidth, layoutWidth)
            : fallbackWidth;

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

              final composerLabel =
                  shortAllInOnly ? 'All-in' : (isRaise ? 'Raise to' : 'Bet');
              final panelRadius = compactLayout ? 13.0 : 15.0;
              final amountRadius = compactLayout ? 11.0 : 13.0;
              final amountFieldStyle = (compactLayout
                      ? PokerTypography.titleLarge
                      : PokerTypography.headlineMedium)
                  .copyWith(
                fontSize: compactLayout ? 16 : (desktopBetChrome ? 19 : 20),
                fontWeight: FontWeight.w700,
                letterSpacing: -0.2,
                color: shortAllInOnly
                    ? PokerColors.warning
                    : PokerColors.textPrimary,
              );
              final amountHintStyle = amountFieldStyle.copyWith(
                color: PokerColors.textMuted,
              );
              final composerLabelStyle = PokerTypography.labelSmall.copyWith(
                color: shortAllInOnly
                    ? PokerColors.warning.withValues(alpha: 0.92)
                    : PokerColors.textSecondary,
                fontSize: compactLayout ? 9.5 : 10.5,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.5,
              );
              final panelDecoration = BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [
                    PokerColors.surface.withValues(alpha: 0.96),
                    PokerColors.surfaceDim.withValues(alpha: 0.98),
                  ],
                ),
                borderRadius: BorderRadius.circular(panelRadius),
                border: Border.all(
                  color: shortAllInOnly
                      ? PokerColors.warning.withValues(alpha: 0.38)
                      : PokerColors.borderMedium,
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withValues(alpha: 0.22),
                    blurRadius: compactLayout ? 12 : 16,
                    offset: Offset(0, compactLayout ? 4 : 6),
                  ),
                ],
              );

              return Container(
                key: const Key('bet-composer-panel'),
                decoration: panelDecoration,
                padding: EdgeInsets.fromLTRB(
                  compactLayout ? 9 : 11,
                  compactLayout ? 5 : 7,
                  compactLayout ? 9 : 11,
                  compactLayout ? 6 : 8,
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      composerLabel.toUpperCase(),
                      style: composerLabelStyle,
                    ),
                    SizedBox(height: compactLayout ? 3 : 4),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.center,
                      children: [
                        Expanded(
                          child: Container(
                            key: const Key('bet-amount-input-shell'),
                            padding: EdgeInsets.symmetric(
                              horizontal: compactLayout ? 10 : 10,
                            ),
                            decoration: BoxDecoration(
                              color: Colors.black.withValues(alpha: 0.18),
                              borderRadius: BorderRadius.circular(amountRadius),
                              border: Border.all(
                                color: shortAllInOnly
                                    ? PokerColors.warning.withValues(alpha: 0.4)
                                    : PokerColors.borderSubtle,
                              ),
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
                              style: amountFieldStyle,
                              decoration: InputDecoration(
                                isDense: true,
                                filled: false,
                                fillColor: Colors.transparent,
                                border: InputBorder.none,
                                enabledBorder: InputBorder.none,
                                focusedBorder: InputBorder.none,
                                disabledBorder: InputBorder.none,
                                errorBorder: InputBorder.none,
                                focusedErrorBorder: InputBorder.none,
                                contentPadding: EdgeInsets.symmetric(
                                  vertical: compactLayout ? 5 : 6,
                                ),
                                hintText: _initialTarget().toString(),
                                hintStyle: amountHintStyle,
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: PokerSpacing.sm),
                        _ActionButton(
                          label: label,
                          onPressed: () => _submitBet(context),
                          color: PokerColors.primary,
                          dense: compactLayout || desktopBetChrome,
                        ),
                        const SizedBox(width: PokerSpacing.xs),
                        (compactLayout || desktopBetChrome)
                            ? Material(
                                color: Colors.transparent,
                                child: InkWell(
                                  onTap: onClose,
                                  borderRadius: BorderRadius.circular(10),
                                  child: Container(
                                    padding: const EdgeInsets.all(7),
                                    decoration: BoxDecoration(
                                      color: PokerColors.overlayLight,
                                      borderRadius: BorderRadius.circular(10),
                                      border: Border.all(
                                        color: PokerColors.borderSubtle,
                                      ),
                                    ),
                                    child: const Icon(
                                      Icons.close_rounded,
                                      size: 15,
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
                    SizedBox(height: compactLayout ? 3 : 4),
                    SliderTheme(
                      data: SliderTheme.of(context).copyWith(
                        activeTrackColor:
                            PokerColors.accent.withValues(alpha: 0.95),
                        inactiveTrackColor:
                            PokerColors.borderMedium.withValues(alpha: 0.88),
                        thumbColor: PokerColors.accent,
                        disabledThumbColor:
                            PokerColors.accent.withValues(alpha: 0.5),
                        disabledActiveTrackColor:
                            PokerColors.accent.withValues(alpha: 0.5),
                        disabledInactiveTrackColor:
                            PokerColors.borderMedium.withValues(alpha: 0.5),
                        overlayColor:
                            PokerColors.accent.withValues(alpha: 0.12),
                        valueIndicatorColor: PokerColors.surfaceBright,
                        valueIndicatorTextStyle:
                            PokerTypography.labelSmall.copyWith(
                          color: PokerColors.textPrimary,
                          fontWeight: FontWeight.w700,
                        ),
                        trackHeight: compactLayout ? 5 : 6,
                        padding: EdgeInsets.zero,
                        thumbShape: RoundSliderThumbShape(
                          enabledThumbRadius: compactLayout ? 7 : 8,
                          disabledThumbRadius: compactLayout ? 6 : 7,
                        ),
                        overlayShape: RoundSliderOverlayShape(
                          overlayRadius: compactLayout ? 12 : 14,
                        ),
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
                      presetLabel:
                          showPresetButton ? suggestedActionLabel : null,
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
                ),
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
      fontSize: compact ? 9.5 : 11,
    );
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: compact ? 2 : 12),
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
            horizontal: compact ? 6 : 10,
            vertical: compact ? 4 : 6,
          ),
          decoration: BoxDecoration(
            color: PokerColors.accent.withValues(alpha: 0.14),
            borderRadius: BorderRadius.circular(999),
            border: Border.all(
              color: PokerColors.accent.withValues(alpha: 0.45),
            ),
          ),
          child: Text(
            label,
            style: PokerTypography.labelSmall.copyWith(
              fontSize: compact ? 9.5 : 11,
              color: PokerColors.accent,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
      ),
    );
  }
}
