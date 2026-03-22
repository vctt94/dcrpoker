import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'pot_display.dart';
import 'scene_layout.dart';
import 'table.dart';
import 'table_theme.dart';
import 'cards.dart';

Color _seatColorFromId(String id) {
  var hash = 0;
  for (int i = 0; i < id.length; i++) {
    hash = id.codeUnitAt(i) + ((hash << 5) - hash);
  }
  final hue = (hash % 360).abs().toDouble();
  return HSLColor.fromAHSL(1.0, hue, 0.5, 0.4).toColor();
}

/// Whether the game phase warrants showing hole-card slots.
bool _showCardsPhase(pr.GamePhase phase) {
  return phase != pr.GamePhase.WAITING &&
      phase != pr.GamePhase.NEW_HAND_DEALING;
}

bool _showSeatCards(UiPlayer player, bool isHero, UiGameState gameState) {
  return !isHero &&
      _showCardsPhase(gameState.phase) &&
      !(player.folded && !player.cardsRevealed);
}

bool _showSeatFaceUpCards(UiPlayer player, UiGameState gameState,
    {required bool showCards}) {
  final isShowdown = gameState.phase == pr.GamePhase.SHOWDOWN;
  return showCards &&
      (!isShowdown || player.cardsRevealed) &&
      player.hand.isNotEmpty;
}

bool _showSeatCardBacks(UiPlayer player, UiGameState gameState,
    {required bool showCards, required bool showFaceUpCards}) {
  return showCards &&
      !showFaceUpCards &&
      (!player.folded || gameState.phase != pr.GamePhase.SHOWDOWN);
}

bool _reserveSeatRail(UiPlayer player, bool isHero, UiGameState gameState,
    {bool showHeroCardsInSeat = false}) {
  if (!_showCardsPhase(gameState.phase)) return false;
  if (isHero) return showHeroCardsInSeat;
  return true;
}

class _SeatCardMetrics {
  const _SeatCardMetrics({
    required this.width,
    required this.height,
    required this.visibleHeight,
    required this.gap,
    required this.railWidth,
  });

  final double width;
  final double height;
  final double visibleHeight;
  final double gap;
  final double railWidth;
}

class _ResolvedSeatLayout {
  const _ResolvedSeatLayout({
    required this.player,
    required this.displayName,
    required this.isHero,
    required this.isCurrent,
    required this.seatColor,
    required this.cards,
    required this.showFaceUpCards,
    required this.showCardBacks,
    required this.showRailCards,
    required this.reserveRail,
    required this.radius,
    required this.uiScale,
    required this.cardScale,
    required this.turnDeadlineMs,
    required this.timeBankSeconds,
    required this.isAutoAdvance,
    required this.left,
    required this.top,
    required this.width,
    required this.height,
    required this.avatarBox,
    required this.coreWidth,
    required this.coreHeight,
    required this.plateLeft,
    required this.plateWidth,
    required this.plateHeight,
    required this.betAnchor,
    this.railMetrics,
    this.cardLeft = 0,
  });

  final UiPlayer player;
  final String displayName;
  final bool isHero;
  final bool isCurrent;
  final Color seatColor;
  final List<pr.Card> cards;
  final bool showFaceUpCards;
  final bool showCardBacks;
  final bool showRailCards;
  final bool reserveRail;
  final double radius;
  final double uiScale;
  final double cardScale;
  final int turnDeadlineMs;
  final int timeBankSeconds;
  final bool isAutoAdvance;
  final double left;
  final double top;
  final double width;
  final double height;
  final double avatarBox;
  final double coreWidth;
  final double coreHeight;
  final double plateLeft;
  final double plateWidth;
  final double plateHeight;
  final double cardLeft;
  final Offset? betAnchor;
  final _SeatCardMetrics? railMetrics;
}

_SeatCardMetrics _seatCardMetrics(
    double radius, double cardScale, double uiScale,
    {required bool isHero}) {
  final baseMultiplier = isHero ? 1.3 : 1.0;
  final minWidth = isHero ? 42.0 : 30.0;
  final maxWidth = isHero ? 70.0 : 58.0;
  final cw = (radius * baseMultiplier * cardScale)
      .clamp(minWidth, maxWidth)
      .toDouble();
  final gap = (cw * 0.12).clamp(3.0, isHero ? 8.0 : 6.0).toDouble();
  final railSideInset =
      (cw * (isHero ? 0.16 : 0.12)).clamp(4.0, 10.0).toDouble();
  final ch = cw * 1.4;
  final visibleHeight = ch * 0.5;
  return _SeatCardMetrics(
    width: cw,
    height: ch,
    visibleHeight: visibleHeight,
    gap: gap,
    railWidth: (cw * 2) + gap + (railSideInset * 2) + 4.0,
  );
}

double _seatInfoPlateWidth({required bool isHero, required double uiScale}) {
  final base = isHero ? 122.0 : 108.0;
  return (base * uiScale).clamp(90.0, 156.0).toDouble();
}

double _seatInfoPlateHeight(UiPlayer player,
    {required bool isHero, required double uiScale}) {
  var height = 38.0 * uiScale;
  final statusCount = [
    if (player.isSmallBlind) 'SB',
    if (player.isBigBlind) 'BB',
    if (player.isDisconnected) 'OFF',
  ].length;
  if (statusCount > 0) {
    height += 16.0 * uiScale;
  }
  if (player.isAllIn) {
    height += 18.0 * uiScale;
  }
  if (isHero) {
    height += 2.0 * uiScale;
  }
  return height;
}

double _seatCorePlateLeft(double radius, double uiScale) {
  return (radius + (22.0 * uiScale)).clamp(40.0, 68.0).toDouble();
}

double _heroSeatDockOverlap(PokerLayoutMode mode, double uiScale) {
  final base = switch (mode) {
    PokerLayoutMode.compactPortrait => 18.0,
    PokerLayoutMode.compactLandscape => 22.0,
    PokerLayoutMode.standard => 28.0,
    PokerLayoutMode.wide => 34.0,
  };
  return (base * uiScale).clamp(12.0, 44.0).toDouble();
}

Offset _betAnchorForSeat({
  required Offset seatCenter,
  required Offset potCenter,
  required Rect tableBounds,
  required bool isHero,
  required double seatRadius,
  required double uiScale,
}) {
  if (isHero) {
    final heroAnchor = Offset(
      seatCenter.dx - (seatRadius * 1.55),
      seatCenter.dy - (seatRadius * 1.55),
    );
    final horizontalPad = 42.0 * uiScale;
    final verticalPad = 24.0 * uiScale;
    return Offset(
      heroAnchor.dx.clamp(
        tableBounds.left + horizontalPad,
        tableBounds.right - horizontalPad,
      ),
      heroAnchor.dy.clamp(
        tableBounds.top + verticalPad,
        tableBounds.bottom - verticalPad,
      ),
    );
  }

  final dx = potCenter.dx - seatCenter.dx;
  final dy = potCenter.dy - seatCenter.dy;
  final distance = math.sqrt((dx * dx) + (dy * dy));
  if (distance <= 0.001) return potCenter;

  final dirX = dx / distance;
  final dirY = dy / distance;
  final seatClearance = seatRadius + (24.0 * uiScale);
  final potClearance = (34.0 * uiScale).clamp(28.0, 42.0).toDouble();
  final usableDistance = math.max(0.0, distance - seatClearance - potClearance);
  final progress = 0.18;
  final travel = seatClearance + usableDistance * progress;
  final anchor = Offset(
    seatCenter.dx + (dirX * travel),
    seatCenter.dy + (dirY * travel),
  );
  final horizontalPad = 46.0 * uiScale;
  final verticalPad = 24.0 * uiScale;
  return Offset(
    anchor.dx.clamp(
      tableBounds.left + horizontalPad,
      tableBounds.right - horizontalPad,
    ),
    anchor.dy.clamp(
      tableBounds.top + verticalPad,
      tableBounds.bottom - verticalPad,
    ),
  );
}

_ResolvedSeatLayout _resolveSeatLayout({
  required UiPlayer player,
  required String heroId,
  required UiGameState gameState,
  required PokerThemeConfig theme,
  required PokerSceneLayout scene,
  required Offset seatPosition,
  required List<pr.Card> heroCardsCache,
  required bool showHeroCardsInSeat,
}) {
  final isHeroSeat = player.id == heroId;
  final isCurrent = player.id == gameState.currentPlayerId && !player.folded;
  final displayName = player.name.isNotEmpty ? player.name : 'Player';
  final seatColor = isHeroSeat
      ? PokerColors.heroSeat
      : (player.isDisconnected
          ? Colors.red.shade700
          : (player.folded
              ? const Color(0xFF3A3D4A)
              : _seatColorFromId(player.id)));
  final cards = isHeroSeat
      ? (player.hand.isNotEmpty ? player.hand : heroCardsCache)
      : player.hand;
  final showCards = _showSeatCards(player, isHeroSeat, gameState);
  final showFaceUpCards =
      _showSeatFaceUpCards(player, gameState, showCards: showCards);
  final showCardBacks = _showSeatCardBacks(player, gameState,
      showCards: showCards, showFaceUpCards: showFaceUpCards);
  final renderHeroCardsInSeat =
      isHeroSeat && showHeroCardsInSeat && _showCardsPhase(gameState.phase);
  final reserveRail = _reserveSeatRail(player, isHeroSeat, gameState,
      showHeroCardsInSeat: renderHeroCardsInSeat);
  final showRailCards =
      isHeroSeat ? renderHeroCardsInSeat : (showFaceUpCards || showCardBacks);
  final radius = kPlayerRadius * theme.uiSizeMultiplier;
  final uiScale = theme.uiSizeMultiplier;
  final cardScale = theme.cardSizeMultiplier;
  final isAutoAdvance = isAutoAdvanceAllIn(gameState);
  final avatarBox = radius * 2 + 8;
  final plateLeft = _seatCorePlateLeft(radius, uiScale);
  final plateWidth = _seatInfoPlateWidth(isHero: isHeroSeat, uiScale: uiScale);
  final plateHeight =
      _seatInfoPlateHeight(player, isHero: isHeroSeat, uiScale: uiScale);
  final coreWidth = plateLeft + plateWidth + 2.0;
  final coreHeight = math.max(avatarBox, plateHeight);
  final railMetrics = reserveRail
      ? _seatCardMetrics(radius, cardScale, uiScale, isHero: isHeroSeat)
      : null;
  final width = railMetrics == null
      ? coreWidth
      : math.max(coreWidth, railMetrics.railWidth);
  final height =
      railMetrics == null ? coreHeight : railMetrics.visibleHeight + coreHeight;
  final cardLeft = railMetrics == null
      ? 0.0
      : (plateLeft + ((plateWidth - railMetrics.railWidth) / 2))
          .clamp(0.0, math.max(0.0, width - railMetrics.railWidth))
          .toDouble();
  final seatCenter = Offset(seatPosition.dx, seatPosition.dy + radius);

  var top = seatCenter.dy - radius - 8;
  if (!isHeroSeat) {
    final minTop = scene.topSeatBandRect.top + 6.0;
    final maxTop = scene.topSeatBandRect.bottom - height - 6.0;
    top = maxTop >= minTop ? top.clamp(minTop, maxTop).toDouble() : minTop;
  } else {
    final minTop = scene.potRect.bottom + 12.0;
    final maxTop = scene.heroDockRect.top -
        height +
        _heroSeatDockOverlap(scene.mode, uiScale);
    top = maxTop >= minTop
        ? math.min(
            math.max(top, minTop),
            maxTop,
          )
        : minTop;
  }

  return _ResolvedSeatLayout(
    player: player,
    displayName: displayName,
    isHero: isHeroSeat,
    isCurrent: isCurrent,
    seatColor: seatColor,
    cards: cards,
    showFaceUpCards: showFaceUpCards,
    showCardBacks: showCardBacks,
    showRailCards: showRailCards,
    reserveRail: reserveRail,
    radius: radius,
    uiScale: uiScale,
    cardScale: cardScale,
    turnDeadlineMs: isCurrent ? gameState.turnDeadlineUnixMs : 0,
    timeBankSeconds: gameState.timeBankSeconds,
    isAutoAdvance: isAutoAdvance,
    left: seatCenter.dx - width / 2,
    top: top,
    width: width,
    height: height,
    avatarBox: avatarBox,
    coreWidth: coreWidth,
    coreHeight: coreHeight,
    plateLeft: plateLeft,
    plateWidth: plateWidth,
    plateHeight: plateHeight,
    cardLeft: cardLeft,
    betAnchor: player.currentBet > 0
        ? _betAnchorForSeat(
            seatCenter: seatCenter,
            potCenter: scene.potRect.center,
            tableBounds: scene.tableRect,
            isHero: isHeroSeat,
            seatRadius: radius,
            uiScale: uiScale,
          )
        : null,
    railMetrics: railMetrics,
  );
}

/// Widget overlay that positions all player seats around the table.
class PlayerSeatsOverlay extends StatelessWidget {
  const PlayerSeatsOverlay({
    super.key,
    required this.gameState,
    required this.heroId,
    required this.theme,
    this.heroCardsCache = const [],
    this.showHeroCardsInSeat = false,
    this.aspectRatio = 16 / 9,
  });

  final UiGameState gameState;
  final String heroId;
  final PokerThemeConfig theme;
  final List<pr.Card> heroCardsCache;
  final bool showHeroCardsInSeat;
  final double aspectRatio;

  @override
  Widget build(BuildContext context) {
    if (gameState.players.isEmpty) return const SizedBox.shrink();

    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size, aspectRatio: aspectRatio);
      final scene = layout.scene;
      final hasCurrentBet = gameState.currentBet > 0;
      final minSeat = minSeatTopFor(layout.viewport, hasCurrentBet);

      final seats = seatPositionsFor(
        gameState.players,
        heroId,
        layout.center,
        layout.ringRadiusX,
        layout.ringRadiusY,
        clampBounds: layout.canvasBounds,
        minSeatTop: minSeat,
        uiSizeMultiplier: theme.uiSizeMultiplier,
        sceneLayout: scene,
      );

      final children = <Widget>[];
      for (final player in gameState.players) {
        final pos = seats[player.id];
        if (pos == null) continue;
        final seatLayout = _resolveSeatLayout(
          player: player,
          heroId: heroId,
          gameState: gameState,
          theme: theme,
          scene: scene,
          seatPosition: pos,
          heroCardsCache: heroCardsCache,
          showHeroCardsInSeat: showHeroCardsInSeat,
        );

        children.add(Positioned(
          left: seatLayout.left,
          top: seatLayout.top,
          child: _PlayerSeatWidget(
            layout: seatLayout,
          ),
        ));

        if (seatLayout.betAnchor != null) {
          children.add(Positioned(
            left: seatLayout.betAnchor!.dx,
            top: seatLayout.betAnchor!.dy,
            child: FractionalTranslation(
              translation: const Offset(-0.5, -0.5),
              child: _SeatBetStack(
                key: ValueKey('seat_bet_${player.id}'),
                amount: player.currentBet,
                theme: theme,
              ),
            ),
          ));
        }
      }

      return Stack(children: children);
    });
  }
}

class _SeatBetStack extends StatelessWidget {
  const _SeatBetStack({
    super.key,
    required this.amount,
    required this.theme,
  });

  final int amount;
  final PokerThemeConfig theme;

  @override
  Widget build(BuildContext context) {
    return BetStackVisual(
      amount: amount,
      theme: theme,
    );
  }
}

class _PlayerSeatWidget extends StatelessWidget {
  const _PlayerSeatWidget({
    required this.layout,
  });

  final _ResolvedSeatLayout layout;

  @override
  Widget build(BuildContext context) {
    final avatar = _AvatarCircle(
      color: layout.seatColor,
      radius: layout.radius,
      isCurrent: layout.isCurrent,
      isFolded: layout.player.folded,
      isDisconnected: layout.player.isDisconnected,
      isHero: layout.isHero,
      uiScale: layout.uiScale,
      turnDeadlineMs: layout.turnDeadlineMs,
      timeBankSeconds: layout.timeBankSeconds,
      isAutoAdvance: layout.isAutoAdvance,
      holeCards: const [],
      showCardBacks: false,
      cardScale: layout.cardScale,
    );
    final core = _SeatCore(
      key: ValueKey('seat_core_${layout.player.id}'),
      layout: layout,
      avatar: avatar,
    );

    if (layout.reserveRail && layout.railMetrics != null) {
      return SizedBox(
        key: ValueKey('seat_widget_${layout.player.id}'),
        width: layout.width,
        height: layout.height,
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            if (layout.showRailCards)
              Positioned(
                top: 0,
                left: layout.cardLeft,
                child: _SeatCardsRail(
                  key: ValueKey('seat_cards_${layout.player.id}'),
                  metrics: layout.railMetrics!,
                  cards: layout.showFaceUpCards || layout.isHero
                      ? layout.cards
                      : const [],
                  showCardBacks: !layout.showFaceUpCards &&
                      !layout.isHero &&
                      layout.showCardBacks,
                ),
              ),
            Positioned(
              top: layout.railMetrics!.visibleHeight,
              left: 0,
              right: 0,
              child: Align(
                alignment: Alignment.topCenter,
                child: core,
              ),
            ),
          ],
        ),
      );
    }

    return SizedBox(
      key: ValueKey('seat_widget_${layout.player.id}'),
      width: layout.width,
      height: layout.height,
      child: Align(
        alignment: Alignment.topCenter,
        child: core,
      ),
    );
  }
}

class _SeatCardsRail extends StatelessWidget {
  const _SeatCardsRail({
    super.key,
    required this.metrics,
    required this.cards,
    required this.showCardBacks,
  });

  final _SeatCardMetrics metrics;
  final List<pr.Card> cards;
  final bool showCardBacks;

  @override
  Widget build(BuildContext context) {
    Widget buildCard(int index) {
      if (cards.length > index) {
        return FlipCard(faceUp: true, card: cards[index]);
      }
      if (showCardBacks) {
        return const CardBack();
      }
      return const SizedBox.shrink();
    }

    return SizedBox(
      width: metrics.railWidth,
      height: metrics.height,
      child: Align(
        alignment: Alignment.topCenter,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            _SeatRailCard(
              width: metrics.width,
              height: metrics.height,
              angle: -0.05,
              child: buildCard(0),
            ),
            SizedBox(width: metrics.gap),
            _SeatRailCard(
              width: metrics.width,
              height: metrics.height,
              angle: 0.05,
              child: buildCard(1),
            ),
          ],
        ),
      ),
    );
  }
}

class _SeatRailCard extends StatelessWidget {
  const _SeatRailCard({
    required this.width,
    required this.height,
    required this.angle,
    required this.child,
  });

  final double width;
  final double height;
  final double angle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Transform.rotate(
      angle: angle,
      child: SizedBox(
        width: width,
        height: height,
        child: child,
      ),
    );
  }
}

class _SeatCore extends StatelessWidget {
  const _SeatCore({
    super.key,
    required this.layout,
    required this.avatar,
  });

  final _ResolvedSeatLayout layout;
  final Widget avatar;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: layout.coreWidth,
      height: layout.coreHeight,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned(
            left: layout.plateLeft,
            top: (layout.coreHeight - layout.plateHeight) / 2,
            child: _SeatInfoPlate(
              layout: layout,
            ),
          ),
          Positioned(
            left: 0,
            top: (layout.coreHeight - layout.avatarBox) / 2,
            child: avatar,
          ),
        ],
      ),
    );
  }
}

class _SeatInfoPlate extends StatelessWidget {
  const _SeatInfoPlate({
    required this.layout,
  });

  final _ResolvedSeatLayout layout;

  @override
  Widget build(BuildContext context) {
    final statusBadges = <Widget>[
      if (layout.player.isSmallBlind)
        _InlineSeatBadge(
          label: 'SB',
          background: PokerColors.primary,
          foreground: Colors.white,
          uiScale: layout.uiScale,
        ),
      if (layout.player.isBigBlind)
        _InlineSeatBadge(
          label: 'BB',
          background: PokerColors.accent,
          foreground: Colors.black,
          uiScale: layout.uiScale,
        ),
      if (layout.player.isDisconnected)
        _InlineSeatBadge(
          label: 'OFF',
          background: PokerColors.dangerDark,
          foreground: Colors.white,
          uiScale: layout.uiScale,
        ),
    ];
    final statusColumn = statusBadges.isEmpty
        ? null
        : Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              for (int i = 0; i < statusBadges.length; i++) ...[
                if (i > 0) SizedBox(height: 4 * layout.uiScale),
                statusBadges[i],
              ],
            ],
          );

    return Container(
      width: layout.plateWidth,
      padding: EdgeInsets.fromLTRB(
        20 * layout.uiScale,
        7 * layout.uiScale,
        10 * layout.uiScale,
        7 * layout.uiScale,
      ),
      decoration: BoxDecoration(
        color: layout.isCurrent
            ? PokerColors.surfaceBright
            : (layout.isHero ? PokerColors.surface : PokerColors.surfaceDim),
        borderRadius: BorderRadius.circular(12 * layout.uiScale),
        border: Border.all(
          color: layout.isCurrent
              ? PokerColors.turnHighlight.withValues(alpha: 0.5)
              : PokerColors.borderSubtle.withValues(alpha: 0.8),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: layout.isHero ? 0.3 : 0.24),
            blurRadius: 10,
            offset: const Offset(0, 3),
          ),
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  layout.displayName,
                  style: PokerTypography.playerName.copyWith(
                    fontSize: 11 * layout.uiScale,
                    color: layout.player.folded
                        ? PokerColors.textMuted
                        : PokerColors.textPrimary,
                    decoration: layout.player.folded
                        ? TextDecoration.lineThrough
                        : null,
                    decorationColor: PokerColors.textMuted,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (layout.player.isDealer)
                _InlineSeatBadge(
                  label: 'D',
                  background: PokerColors.warning,
                  foreground: Colors.black,
                  uiScale: layout.uiScale,
                ),
            ],
          ),
          SizedBox(height: 4 * layout.uiScale),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Text(
                  '${layout.player.balance}',
                  style: PokerTypography.chipCount.copyWith(
                    fontSize: layout.isHero
                        ? 12 * layout.uiScale
                        : 11 * layout.uiScale,
                    color: layout.player.folded
                        ? PokerColors.textSecondary
                        : PokerColors.textPrimary,
                  ),
                ),
              ),
              if (statusColumn != null) SizedBox(width: 8 * layout.uiScale),
              if (statusColumn != null) statusColumn,
            ],
          ),
          if (layout.player.isAllIn) SizedBox(height: 5 * layout.uiScale),
          if (layout.player.isAllIn)
            Container(
              padding: EdgeInsets.symmetric(
                horizontal: 8 * layout.uiScale,
                vertical: 3 * layout.uiScale,
              ),
              decoration: BoxDecoration(
                color: PokerColors.danger,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                'ALL-IN',
                style: PokerTypography.badgeLabel.copyWith(
                  fontSize: 10 * layout.uiScale,
                  color: Colors.white,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Avatar circle — now renders opponent cards inside when available.
// ─────────────────────────────────────────────

class _AvatarCircle extends StatefulWidget {
  const _AvatarCircle({
    required this.color,
    required this.radius,
    required this.isCurrent,
    required this.isFolded,
    required this.isDisconnected,
    required this.isHero,
    required this.uiScale,
    required this.turnDeadlineMs,
    required this.timeBankSeconds,
    required this.isAutoAdvance,
    this.holeCards = const [],
    this.showCardBacks = false,
    this.cardScale = 1.0,
  });

  final Color color;
  final double radius, uiScale, cardScale;
  final bool isCurrent, isFolded, isDisconnected, isHero, isAutoAdvance;
  final bool showCardBacks;
  final int turnDeadlineMs;
  final int timeBankSeconds;
  final List<pr.Card> holeCards;

  @override
  State<_AvatarCircle> createState() => _AvatarCircleState();
}

class _AvatarCircleState extends State<_AvatarCircle>
    with SingleTickerProviderStateMixin {
  late final AnimationController _timerCtrl;

  @override
  void initState() {
    super.initState();
    _timerCtrl =
        AnimationController(vsync: this, duration: const Duration(seconds: 1));
    if (widget.isCurrent &&
        widget.turnDeadlineMs > 0 &&
        !widget.isAutoAdvance) {
      _timerCtrl.repeat();
    }
  }

  @override
  void didUpdateWidget(covariant _AvatarCircle old) {
    super.didUpdateWidget(old);
    if (widget.isCurrent &&
        widget.turnDeadlineMs > 0 &&
        !widget.isAutoAdvance) {
      if (!_timerCtrl.isAnimating) _timerCtrl.repeat();
    } else {
      _timerCtrl.stop();
    }
  }

  @override
  void dispose() {
    _timerCtrl.dispose();
    super.dispose();
  }

  /// Build the inner content of the circle. If hole cards should be
  /// displayed (face-up or face-down), render two mini-cards side by side.
  /// Otherwise fall back to initials / chip count.
  Widget _buildCenter(double diameter) {
    final hasRevealedCards = widget.holeCards.isNotEmpty;
    final showCards = hasRevealedCards || widget.showCardBacks;

    if (showCards) {
      // Use the inner circle size after border so the mini-card row never
      // overflows at larger ui/card scale combinations.
      final borderWidth =
          ((widget.isCurrent ? 2.5 : 1.5) * widget.uiScale).toDouble();
      final innerDiameter = math.max(0.0, diameter - (borderWidth * 2));
      final gap = (innerDiameter * 0.04).clamp(2.0, 4.0).toDouble();
      final targetCw = diameter * 0.34 * widget.cardScale;
      final maxWidthCw = math.max(0.0, (innerDiameter - gap) / 2);
      final maxHeightCw = innerDiameter / 1.4;
      final cw = math.min(targetCw, math.min(maxWidthCw, maxHeightCw));
      final ch = cw * 1.4;

      Widget card1;
      Widget card2;

      if (hasRevealedCards) {
        card1 = SizedBox(
            width: cw, height: ch, child: CardFace(card: widget.holeCards[0]));
        card2 = SizedBox(
            width: cw,
            height: ch,
            child: widget.holeCards.length > 1
                ? CardFace(card: widget.holeCards[1])
                : const CardBack());
      } else {
        card1 = SizedBox(width: cw, height: ch, child: const CardBack());
        card2 = SizedBox(width: cw, height: ch, child: const CardBack());
      }

      return Row(
        mainAxisSize: MainAxisSize.min,
        children: [card1, SizedBox(width: gap), card2],
      );
    }

    return Icon(
      Icons.person_rounded,
      size: widget.radius * 0.86,
      color: Colors.white.withValues(alpha: 0.88),
    );
  }

  @override
  Widget build(BuildContext context) {
    final diameter = widget.radius * 2;
    final opacity = widget.isFolded ? 0.5 : 1.0;

    return Opacity(
      opacity: opacity,
      child: SizedBox(
        width: diameter + 8,
        height: diameter + 8,
        child: Stack(
          alignment: Alignment.center,
          children: [
            // Timebank ring (animated)
            if (widget.isCurrent &&
                widget.turnDeadlineMs > 0 &&
                !widget.isAutoAdvance)
              AnimatedBuilder(
                animation: _timerCtrl,
                builder: (context, _) {
                  final nowMs = DateTime.now().millisecondsSinceEpoch;
                  final remainingMs =
                      (widget.turnDeadlineMs - nowMs).clamp(0, 1 << 30);
                  final totalDurationMs = ((widget.timeBankSeconds > 0
                              ? widget.timeBankSeconds
                              : 30) *
                          1000)
                      .toDouble();
                  final fraction =
                      (remainingMs / totalDurationMs).clamp(0.0, 1.0);
                  final ringColor = fraction > 0.5
                      ? PokerColors.accent
                      : (fraction > 0.2
                          ? PokerColors.warning
                          : PokerColors.danger);
                  return SizedBox(
                    width: diameter + 8,
                    height: diameter + 8,
                    child: CircularProgressIndicator(
                      value: fraction,
                      strokeWidth: 3 * widget.uiScale,
                      color: ringColor,
                      backgroundColor: PokerColors.borderSubtle,
                    ),
                  );
                },
              ),

            // Turn glow
            if (widget.isCurrent)
              Container(
                width: diameter + 6,
                height: diameter + 6,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  boxShadow: [
                    BoxShadow(
                      color: PokerColors.turnHighlight.withValues(alpha: 0.35),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ],
                ),
              ),

            // Main avatar circle
            Container(
              width: diameter,
              height: diameter,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: widget.color,
                border: Border.all(
                  color: widget.isDisconnected
                      ? Colors.orangeAccent
                      : (widget.isCurrent
                          ? PokerColors.turnHighlight
                          : PokerColors.borderSubtle.withValues(alpha: 0.5)),
                  width: (widget.isCurrent ? 2.5 : 1.5) * widget.uiScale,
                ),
              ),
              child: ClipOval(
                child: Center(child: _buildCenter(diameter)),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Small helper widgets
// ─────────────────────────────────────────────

class _InlineSeatBadge extends StatelessWidget {
  const _InlineSeatBadge({
    required this.label,
    required this.background,
    required this.foreground,
    required this.uiScale,
  });

  final String label;
  final Color background;
  final Color foreground;
  final double uiScale;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: 6 * uiScale,
        vertical: 2 * uiScale,
      ),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(999),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.24),
            blurRadius: 2,
            offset: const Offset(0, 1),
          ),
        ],
      ),
      child: Text(
        label,
        style: PokerTypography.badgeLabel.copyWith(
          fontSize: 9 * uiScale,
          color: foreground,
        ),
      ),
    );
  }
}
