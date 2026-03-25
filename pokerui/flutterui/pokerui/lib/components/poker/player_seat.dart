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

enum _SeatRailPlacement {
  top,
  left,
  right,
  bottom,
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
    required this.avatarLeft,
    required this.coreWidth,
    required this.coreHeight,
    required this.coreLeft,
    required this.coreTop,
    required this.plateLeft,
    required this.plateWidth,
    required this.plateHeight,
    required this.betAnchor,
    this.railMetrics,
    this.cardLeft = 0,
    this.cardTop = 0,
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
  final double avatarLeft;
  final double coreWidth;
  final double coreHeight;
  final double coreLeft;
  final double coreTop;
  final double plateLeft;
  final double plateWidth;
  final double plateHeight;
  final double cardLeft;
  final double cardTop;
  final Offset? betAnchor;
  final _SeatCardMetrics? railMetrics;
}

_SeatCardMetrics _seatCardMetrics(
  double radius,
  double cardScale,
  double uiScale, {
  required bool isHero,
  required PokerLayoutMode mode,
}) {
  final compactOpponent = !isHero && mode == PokerLayoutMode.compactPortrait;
  final baseMultiplier = isHero ? 1.3 : (compactOpponent ? 0.76 : 1.0);
  final minWidth = isHero ? 42.0 : (compactOpponent ? 22.0 : 30.0);
  final maxWidth = isHero ? 70.0 : (compactOpponent ? 40.0 : 58.0);
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

double _seatInfoPlateWidth({
  required bool isHero,
  required double uiScale,
  required PokerLayoutMode mode,
}) {
  final compactOpponent = !isHero && mode == PokerLayoutMode.compactPortrait;
  final base = isHero ? 122.0 : (compactOpponent ? 84.0 : 108.0);
  return (base * uiScale)
      .clamp(compactOpponent ? 72.0 : 90.0, 156.0)
      .toDouble();
}

double _seatInfoPlateHeight(UiPlayer player,
    {required bool isHero,
    required double uiScale,
    required PokerLayoutMode mode}) {
  final compactOpponent = !isHero && mode == PokerLayoutMode.compactPortrait;
  var height = (compactOpponent ? 34.0 : 38.0) * uiScale;
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

double _seatCorePlateLeft(
  double radius,
  double uiScale, {
  required bool isHero,
  required PokerLayoutMode mode,
}) {
  final compactOpponent = !isHero && mode == PokerLayoutMode.compactPortrait;
  return (radius + ((compactOpponent ? 14.0 : 22.0) * uiScale))
      .clamp(compactOpponent ? 22.0 : 40.0, 68.0)
      .toDouble();
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

_SeatRailPlacement _railPlacementForSeat({
  required bool isHeroSeat,
}) {
  if (isHeroSeat) return _SeatRailPlacement.top;
  return _SeatRailPlacement.top;
}

double _safeClamp(double value, double lower, double upper) {
  if (upper < lower) {
    return (lower + upper) / 2;
  }
  return value.clamp(lower, upper).toDouble();
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
      _safeClamp(
        heroAnchor.dx,
        tableBounds.left + horizontalPad,
        tableBounds.right - horizontalPad,
      ),
      _safeClamp(
        heroAnchor.dy,
        tableBounds.top + verticalPad,
        tableBounds.bottom - verticalPad,
      ),
    );
  }

  final tableCenter = tableBounds.center;
  final isTopSeat =
      seatCenter.dy <= tableCenter.dy - (tableBounds.height * 0.08);
  final isLeftSeat = seatCenter.dx < tableCenter.dx;
  final topSeatX = seatCenter.dx < tableCenter.dx - (seatRadius * 0.45)
      ? seatCenter.dx - (seatRadius * 0.22)
      : (seatCenter.dx > tableCenter.dx + (seatRadius * 0.45)
          ? seatCenter.dx + (seatRadius * 0.22)
          : seatCenter.dx);
  final inwardX = seatRadius * 0.04;
  final downwardY = (seatRadius * 1.52) + (4.0 * uiScale);
  final topSeatGap = (seatRadius * 1.32) + (10.0 * uiScale);

  final anchor = isTopSeat
      ? Offset(
          topSeatX,
          seatCenter.dy + topSeatGap,
        )
      : Offset(
          seatCenter.dx + (isLeftSeat ? inwardX : -inwardX),
          seatCenter.dy + downwardY,
        );
  final horizontalPad = 22.0 * uiScale;
  final verticalPad = 20.0 * uiScale;
  return Offset(
    _safeClamp(
      anchor.dx,
      tableBounds.left + horizontalPad,
      tableBounds.right - horizontalPad,
    ),
    _safeClamp(
      anchor.dy,
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
  final radius = kPlayerRadius *
      theme.uiSizeMultiplier *
      (!isHeroSeat && scene.isPhonePortrait ? 0.74 : 1.0);
  final uiScale = theme.uiSizeMultiplier;
  final cardScale = theme.cardSizeMultiplier;
  final isAutoAdvance = isAutoAdvanceAllIn(gameState);
  final avatarBox = radius * 2 + 8;
  final seatCenter = Offset(seatPosition.dx, seatPosition.dy + radius);
  final railPlacement = _railPlacementForSeat(
    isHeroSeat: isHeroSeat,
  );
  final basePlateLeft = _seatCorePlateLeft(
    radius,
    uiScale,
    isHero: isHeroSeat,
    mode: scene.mode,
  );
  final plateWidth = _seatInfoPlateWidth(
    isHero: isHeroSeat,
    uiScale: uiScale,
    mode: scene.mode,
  );
  final plateHeight = _seatInfoPlateHeight(
    player,
    isHero: isHeroSeat,
    uiScale: uiScale,
    mode: scene.mode,
  );
  final coreWidth = basePlateLeft + plateWidth + 2.0;
  final shouldMirror = !isHeroSeat &&
      seatCenter.dx > scene.tableCenter.dx + (scene.tableRect.width * 0.09);
  final avatarLeft = shouldMirror ? coreWidth - avatarBox : 0.0;
  final plateLeft = shouldMirror ? 0.0 : basePlateLeft;
  final coreHeight = math.max(avatarBox, plateHeight);
  final railMetrics = reserveRail
      ? _seatCardMetrics(
          radius,
          cardScale,
          uiScale,
          isHero: isHeroSeat,
          mode: scene.mode,
        )
      : null;
  final sideRailGap = 4.0 * uiScale;
  final width = switch (railPlacement) {
    _ when railMetrics == null => coreWidth,
    _SeatRailPlacement.left || _SeatRailPlacement.right => coreWidth,
    _ => math.max(coreWidth, railMetrics!.railWidth),
  };
  final height = switch (railPlacement) {
    _ when railMetrics == null => coreHeight,
    _SeatRailPlacement.left ||
    _SeatRailPlacement.right =>
      math.max(coreHeight, railMetrics!.height),
    _ => coreHeight + railMetrics!.visibleHeight,
  };
  final coreLeft = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.left || _SeatRailPlacement.right => 0.0,
    _ => (width - coreWidth) / 2,
  };
  final coreTop = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.left ||
    _SeatRailPlacement.right =>
      (height - coreHeight) / 2,
    _SeatRailPlacement.top => railMetrics!.visibleHeight,
    _SeatRailPlacement.bottom => 0.0,
  };
  final avatarCenterX = coreLeft + avatarLeft + avatarBox / 2;
  final centeredRailLeft = railMetrics == null
      ? 0.0
      : (coreLeft + plateLeft + ((plateWidth - railMetrics.railWidth) / 2))
          .clamp(0.0, math.max(0.0, width - railMetrics.railWidth))
          .toDouble();
  final cardLeft = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.left => -(railMetrics!.railWidth + sideRailGap),
    _SeatRailPlacement.right => coreWidth + sideRailGap,
    _ => centeredRailLeft,
  };
  final cardTop = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.left ||
    _SeatRailPlacement.right =>
      (height - railMetrics!.height) / 2,
    _SeatRailPlacement.top => 0.0,
    _SeatRailPlacement.bottom =>
      coreHeight - (railMetrics!.height - railMetrics.visibleHeight),
  };
  final railOverflowLeft = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.left => railMetrics!.railWidth + sideRailGap,
    _ => 0.0,
  };
  final railOverflowRight = switch (railPlacement) {
    _ when railMetrics == null => 0.0,
    _SeatRailPlacement.right => railMetrics!.railWidth + sideRailGap,
    _ => 0.0,
  };
  final sideRailClampFactor = scene.isPhonePortrait ? 0.4 : 1.0;

  var left =
      isHeroSeat ? seatCenter.dx - width / 2 : seatCenter.dx - avatarCenterX;
  final minLeft =
      scene.contentRect.left + 6.0 + railOverflowLeft * sideRailClampFactor;
  final maxLeft = scene.contentRect.right -
      width -
      6.0 -
      railOverflowRight * sideRailClampFactor;
  left = maxLeft >= minLeft ? left.clamp(minLeft, maxLeft).toDouble() : minLeft;

  var top = seatCenter.dy - radius - 8;
  if (!isHeroSeat) {
    final minTop = scene.contentRect.top + 6.0;
    final maxTop = scene.heroDockRect.top - height - 12.0;
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

  if (!isHeroSeat) {
    final seatRect = Rect.fromLTWH(left, top, width, height);
    if (seatRect.overlaps(scene.boardRect)) {
      if (seatCenter.dx < scene.tableCenter.dx - 8.0) {
        left = math.min(left, scene.boardRect.left - width - 8.0);
      } else if (seatCenter.dx > scene.tableCenter.dx + 8.0) {
        left = math.max(left, scene.boardRect.right + 8.0);
      } else {
        top = math.min(top, scene.boardRect.top - height - 10.0);
      }
      left = _safeClamp(
        left,
        scene.contentRect.left + 6.0,
        scene.contentRect.right - width - 6.0,
      );
      top = _safeClamp(
        top,
        scene.contentRect.top + 6.0,
        scene.heroDockRect.top - height - 12.0,
      );

      final adjustedRect = Rect.fromLTWH(left, top, width, height);
      if (adjustedRect.overlaps(scene.boardRect)) {
        top = _safeClamp(
          scene.boardRect.top - height - 10.0,
          scene.contentRect.top + 6.0,
          scene.heroDockRect.top - height - 12.0,
        );
      }
    }
  }

  final avatarCenterY = top + coreTop + (coreHeight / 2);
  final actualSeatCenter = Offset(left + avatarCenterX, avatarCenterY);

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
    left: left,
    top: top,
    width: width,
    height: height,
    avatarBox: avatarBox,
    avatarLeft: avatarLeft,
    coreWidth: coreWidth,
    coreHeight: coreHeight,
    coreLeft: coreLeft,
    coreTop: coreTop,
    plateLeft: plateLeft,
    plateWidth: plateWidth,
    plateHeight: plateHeight,
    cardLeft: cardLeft,
    cardTop: cardTop,
    betAnchor: player.currentBet > 0
        ? _betAnchorForSeat(
            seatCenter: actualSeatCenter,
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
                top: layout.cardTop,
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
              top: layout.coreTop,
              left: layout.coreLeft,
              child: core,
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
              key: ValueKey('seat_plate_${layout.player.id}'),
              layout: layout,
            ),
          ),
          Positioned(
            left: layout.avatarLeft,
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
    super.key,
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
                  maxLines: 1,
                  softWrap: false,
                  overflow: TextOverflow.ellipsis,
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
