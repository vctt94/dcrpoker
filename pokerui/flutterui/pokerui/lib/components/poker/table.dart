import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'cards.dart';
import 'table_theme.dart';

// Canvas-based table and player drawing utilities for CustomPainter usage
const double kPlayerRadius = 30.0;
const double _minPlayerOffset = 32.0;
const double _maxPlayerOffset = 50.0;
const double _edgePadding = 12.0;
// Top overlay sizing so we can reserve space for pot/current bet labels.
const double kPotOverlayHeight = 42.0;
const double kCurrentBetOverlayHeight = 28.0;
const double kTopOverlayGap = 8.0;
const double kTopOverlayMargin = 6.0;
const double kOverlaySeatGap = 6.0;
const double kPotChipYOffsetFactor = -0.34;

class TableLayout {
  const TableLayout({
    required this.viewport,
    required this.center,
    required this.tableRadiusX,
    required this.tableRadiusY,
    required this.playerOffset,
  });

  final Rect viewport;
  final Offset center;
  final double tableRadiusX;
  final double tableRadiusY;
  final double playerOffset;

  double get ringRadiusX => tableRadiusX + playerOffset;
  double get ringRadiusY => tableRadiusY + playerOffset;
}

double _playerOffsetForViewport(Rect viewport) {
  final scaled = viewport.shortestSide * 0.08;
  return scaled.clamp(_minPlayerOffset, _maxPlayerOffset);
}

double _stackSpacing(double radius, double uiSizeMultiplier,
    {double textHeight = 0}) {
  final sizeGap = radius * 0.2;
  final uiGap = 6.0 * uiSizeMultiplier;
  final textGap = textHeight > 0 ? textHeight * 0.45 : 0.0;
  return math.max(sizeGap, math.max(uiGap, textGap));
}

double minSeatTopFor(Rect viewport, bool hasCurrentBet) {
  final basePad = kTopOverlayMargin + kOverlaySeatGap;
  final adaptivePad = (viewport.height * 0.02).clamp(6.0, 14.0);
  final betPad = hasCurrentBet ? 4.0 : 0.0;
  return viewport.top + basePad + adaptivePad + betPad;
}

Offset potChipCenter(TableLayout layout, {double uiSizeMultiplier = 1.0}) {
  final box = layout.viewport;
  final center = layout.center;
  final chipSize = 32.0 * uiSizeMultiplier;
  final labelGap = 6.0 * uiSizeMultiplier;
  final margin = 6.0 * uiSizeMultiplier;

  final labelStyle = TextStyle(
    fontSize: 14 * uiSizeMultiplier,
    fontWeight: FontWeight.bold,
    letterSpacing: 0.4,
    color: Colors.white,
  );
  final padH = 12.0 * uiSizeMultiplier;
  final padV = 6.0 * uiSizeMultiplier;
  final tp = TextPainter(
    text: TextSpan(text: '0', style: labelStyle),
    textDirection: TextDirection.ltr,
  )..layout();
  final labelHeight = tp.height + padV * 2;

  final cw = (box.width * 0.05).clamp(32.0, 56.0);
  final ch = cw * 1.4;
  final cardTop = center.dy - ch / 2 - 20.0;
  final totalHeight = chipSize + labelGap + labelHeight;
  var columnTop = cardTop - margin - totalHeight;
  final overflow = (box.top + 6.0) - columnTop;
  if (overflow > 0) columnTop += overflow;

  return Offset(center.dx, columnTop + chipSize / 2);
}

Offset _positionForSeat(
  int idx,
  int heroIndex,
  int count,
  Offset center,
  double ringRadiusX,
  double ringRadiusY,
  Rect? clampBounds,
  double? minSeatTop, {
  double uiSizeMultiplier = 1.0,
  double playerRadius = kPlayerRadius,
}) {
  final angle = _angleForPlayerIndex(idx, heroIndex, count);
  var x = center.dx + ringRadiusX * math.cos(angle);
  var y = center.dy + ringRadiusY * math.sin(angle);

  if (minSeatTop != null) {
    final seatTop = y - playerRadius;
    if (seatTop < minSeatTop) {
      y += (minSeatTop - seatTop);
    }
  }

  if (clampBounds != null) {
    final hPad = playerRadius + 12.0 * uiSizeMultiplier;
    final verticalBadgeAllowance =
        math.max(playerRadius * 0.75, 18.0 * uiSizeMultiplier);
    final vPad =
        playerRadius + verticalBadgeAllowance + 6.0 * uiSizeMultiplier;
    final left = clampBounds.left + hPad;
    final right = clampBounds.right - hPad;
    final top = clampBounds.top + vPad;
    final bottom = clampBounds.bottom - vPad;
    x = x.clamp(left, right);
    y = y.clamp(top, bottom);
  }

  return Offset(x, y);
}

double _angleForPlayerIndex(int idx, int heroIndex, int count) {
  if (count <= 0) return 0;
  if (heroIndex < 0 || heroIndex >= count) {
    return (idx * 2 * math.pi) / count;
  }
  final step = (2 * math.pi) / count;
  var angle = (math.pi / 2) + (idx - heroIndex) * step;
  angle %= (2 * math.pi);
  if (angle < 0) angle += 2 * math.pi;
  return angle;
}

TableLayout resolveTableLayout(Size size) {
  final viewport = pokerViewportRect(size);
  final center = Offset(
      viewport.left + viewport.width / 2, viewport.top + viewport.height / 2);
  final playerOffset = _playerOffsetForViewport(viewport);

  const desiredMinRadiusX = 180.0;
  const desiredMinRadiusY = 130.0;

  final availableX =
      (viewport.width / 2) - (playerOffset + kPlayerRadius + _edgePadding);
  final availableY =
      (viewport.height / 2) - (playerOffset + kPlayerRadius + _edgePadding);

  double clampRadius(double target, double available, double minDesired) {
    final maxRadius = available.clamp(0.0, double.infinity);
    if (maxRadius <= 0) return 0;
    final minRadius = math.min(minDesired, maxRadius);
    return target.clamp(minRadius, maxRadius);
  }

  final tableRadiusX =
      clampRadius(viewport.width * 0.42, availableX, desiredMinRadiusX);
  final tableRadiusY =
      clampRadius(viewport.height * 0.34, availableY, desiredMinRadiusY);

  return TableLayout(
    viewport: viewport,
    center: center,
    tableRadiusX: tableRadiusX,
    tableRadiusY: tableRadiusY,
    playerOffset: playerOffset,
  );
}

void drawPokerTable(Canvas canvas, double centerX, double centerY,
    double tableRadiusX, double tableRadiusY, TableThemeConfig theme) {
  // Table surface - draw as ellipse using theme colors
  final tablePaint = Paint()
    ..color = theme.feltColor
    ..style = PaintingStyle.fill;

  final tableRect = Rect.fromCenter(
    center: Offset(centerX, centerY),
    width: tableRadiusX * 2,
    height: tableRadiusY * 2,
  );
  canvas.drawOval(tableRect, tablePaint);

  // Table border
  final borderPaint = Paint()
    ..color = theme.borderColor
    ..style = PaintingStyle.stroke
    ..strokeWidth = 8;

  canvas.drawOval(tableRect, borderPaint);
}

void drawPlayers(
    Canvas canvas,
    List<UiPlayer> players,
    String currentPlayerId,
    UiGameState gameState,
    double centerX,
    double centerY,
    double tableRadiusX,
    double tableRadiusY,
    int showdownStartMs,
    Size size,
    double cardSizeMultiplier,
    double uiSizeMultiplier,
    {double? playerOffsetOverride,
    Rect? clampBounds,
    double? minSeatTop}) {
  final playerRadius = kPlayerRadius * uiSizeMultiplier;
  final baseOffset = playerOffsetOverride ??
      _playerOffsetForViewport(
          clampBounds ?? Rect.fromLTWH(0, 0, size.width, size.height));
  // Keep player centers further from the table edge as icons scale up.
  final playerOffset =
      baseOffset + math.max(0, (playerRadius - kPlayerRadius) * 0.9);
  final clampRect = clampBounds ?? Rect.fromLTWH(0, 0, size.width, size.height);
  final count = players.length;
  if (count == 0) return;

  // Find hero index for positioning
  final heroIndex = players.indexWhere((p) => p.id == currentPlayerId);

  for (int i = 0; i < count; i++) {
    final player = players[i];
    final pos = _positionForSeat(
      i,
      heroIndex,
      count,
      Offset(centerX, centerY),
      tableRadiusX + playerOffset,
      tableRadiusY + playerOffset,
      clampRect,
      minSeatTop,
      uiSizeMultiplier: uiSizeMultiplier,
      playerRadius: playerRadius,
    );

    drawPlayer(
      canvas,
      pos.dx,
      pos.dy,
      playerRadius,
      player,
      i,
      Offset(centerX, centerY),
      currentPlayerId,
      gameState,
      uiSizeMultiplier,
      clampRect,
    );

    if (player.id != currentPlayerId) {
      final playerX = pos.dx;
      final playerY = pos.dy;
      // Skip rendering hole cards for folded opponents to avoid implying they are still in-hand.
      if (player.folded) {
        continue;
      }
      final hasAnyCards = player.hand.isNotEmpty;
      // Draw cards inside the player circle (positioned at bottom of circle)
      // Calculate card size as proportion of player radius
      // Two cards + gap should fit within ~70% of circle diameter
      final cardWidthRatio = .75;
      final cw = playerRadius * cardWidthRatio * cardSizeMultiplier;
      final ch = cw * 1.4;
      final gap =
          playerRadius * 0.05 * cardSizeMultiplier; // Small gap between cards
      final startX = playerX - cw - gap / 2;
      // Position cards centered in the player circle
      final cardY = playerY + playerRadius * 0.6 - ch; // Centered in circle

      if (gameState.phase == pr.GamePhase.SHOWDOWN) {
        if (hasAnyCards) {
          // Reveal known opponent hands with a subtle slide-in animation
          final now = DateTime.now().millisecondsSinceEpoch;
          final elapsed = (now - showdownStartMs - i * 120);
          final t = (elapsed / 450.0).clamp(0.0, 1.0);
          final y = cardY - (1.0 - t) * 8.0; // Slide up slightly
          drawCardFace(canvas, startX, y, cw, ch, player.hand[0]);
          if (player.hand.length > 1) {
            drawCardFace(canvas, startX + cw + gap, y, cw, ch, player.hand[1]);
          }
        } else {
          // If still hidden at showdown, show subtle backs inside circle
          drawCardBack(canvas, startX, cardY, cw, ch);
          drawCardBack(canvas, startX + cw + gap, cardY, cw, ch);
        }
      } else if (!hasAnyCards &&
          (gameState.phase != pr.GamePhase.WAITING &&
              gameState.phase != pr.GamePhase.NEW_HAND_DEALING)) {
        // Non-showdown phases: use backs to indicate in-hand cards for opponents
        drawCardBack(canvas, startX, cardY, cw, ch);
        drawCardBack(canvas, startX + cw + gap, cardY, cw, ch);
      }
    }
  }
}

void drawPlayer(
  Canvas canvas,
  double x,
  double y,
  double radius,
  UiPlayer player,
  int index,
  Offset tableCenter,
  String currentPlayerId,
  UiGameState gameState,
  double uiSizeMultiplier,
  Rect viewportBounds,
) {
  final isHero = player.id == currentPlayerId;
  final isFolded = player.folded;
  // Compute turn highlight based on authoritative currentPlayerId from
  // the game state to avoid transient races in per-player isTurn flags.
  final isCurrent = player.id == gameState.currentPlayerId && !isFolded;
  const heroColor = Color(0xFF2E6DD8);
  final otherColor = Colors.grey.shade700;

  // Player circle
  final playerPaint = Paint()
    ..color = player.isDisconnected
        ? Colors.red.shade700
        : (isFolded ? Colors.grey.shade800 : (isHero ? heroColor : otherColor))
    ..style = PaintingStyle.fill;

  canvas.drawCircle(Offset(x, y), radius, playerPaint);

  // Soft halo when it's their turn
  if (isCurrent) {
    final haloPaint = Paint()
      ..color = Colors.yellowAccent.withOpacity(0.3)
      ..style = PaintingStyle.fill
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 12);
    canvas.drawCircle(Offset(x, y), radius + 4 * uiSizeMultiplier, haloPaint);
  }

  // Player border
  final borderPaint = Paint()
    ..color = player.isDisconnected
        ? Colors.orangeAccent
        : (isFolded
            ? Colors.white24.withOpacity(0.6)
            : (isCurrent ? Colors.yellowAccent : Colors.white24))
    ..style = PaintingStyle.stroke
    ..strokeWidth = (isCurrent ? 2.5 : 1.5) * uiSizeMultiplier;

  canvas.drawCircle(Offset(x, y), radius, borderPaint);

  // Dim folded players with an overlay
  if (isFolded) {
    final foldOverlay = Paint()..color = Colors.black.withOpacity(0.45);
    canvas.drawCircle(Offset(x, y), radius, foldOverlay);
    // Keep a red marker around folded players for quick recognition.
    final foldRing = Paint()
      ..color = Colors.redAccent.withOpacity(0.9)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.0 * uiSizeMultiplier;
    canvas.drawCircle(Offset(x, y), radius + 3 * uiSizeMultiplier, foldRing);
    final arrowPaint = Paint()
      ..color = Colors.redAccent.withOpacity(0.85)
      ..style = PaintingStyle.fill;
    final arrow = Path()
      ..moveTo(x, y + radius + 6 * uiSizeMultiplier)
      ..lineTo(x - 7 * uiSizeMultiplier, y + radius + 16 * uiSizeMultiplier)
      ..lineTo(x + 7 * uiSizeMultiplier, y + radius + 16 * uiSizeMultiplier)
      ..close();
    canvas.drawPath(arrow, arrowPaint);
  }

  // Use blind positions from server instead of calculating client-side
  final badges = <SeatBadge>[];

  if (player.isDealer) {
    badges.add(const SeatBadge('D', Colors.amber));
  }
  if (player.isSmallBlind) {
    badges.add(const SeatBadge('SB', decredBlue));
  }
  if (player.isBigBlind) {
    badges.add(const SeatBadge('BB', decredGreen));
  }
  // Add ALL-IN badge when player is all-in
  if (player.isAllIn) {
    badges.add(const SeatBadge('ALL-IN', Colors.redAccent));
  }

  // Player chips (styled like a badge)
  if (player.balance > 0) {
    final chipText = TextPainter(
      text: TextSpan(
        text: '${player.balance}',
        style: TextStyle(
          color: Colors.white,
          fontSize: 10 * uiSizeMultiplier,
          fontWeight: FontWeight.w600,
        ),
      ),
      textDirection: TextDirection.ltr,
    );
    chipText.layout();

    // Draw chip badge background
    final chipBadgeWidth = chipText.width + 12 * uiSizeMultiplier;
    final chipBadgeHeight =
        math.max(16.0 * uiSizeMultiplier, radius * 0.45);
    final chipBadgeX = x - chipBadgeWidth / 2;

    // Layout strategy: All players use the same pattern
    // - Chips above the player circle
    // - Name below the player circle
    final spacingFromCircle = _stackSpacing(radius, uiSizeMultiplier);

    // Position chips above the circle for all players: [chip badge] ... [player circle] ... [name]
    final chipBadgeY = y - radius - spacingFromCircle - chipBadgeHeight;

    // Clamp chips to remain fully inside the viewport (both top and bottom)
    final minChipY = viewportBounds.top + 6.0 * uiSizeMultiplier;
    final maxChipY =
        viewportBounds.bottom - chipBadgeHeight - 6.0 * uiSizeMultiplier;
    final chipBadgeYClamped = chipBadgeY.clamp(minChipY, maxChipY);
    final chipBadgeRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(
        chipBadgeX,
        chipBadgeYClamped,
        chipBadgeWidth,
        chipBadgeHeight,
      ),
      const Radius.circular(8),
    );
    final chipBgPaint = Paint()..color = Colors.black.withOpacity(0.7);
    canvas.drawRRect(chipBadgeRect, chipBgPaint);

    chipText.paint(
      canvas,
      Offset(
        x - chipText.width / 2,
        chipBadgeYClamped + 2,
      ),
    );
  }

  // Player name label (below chips)
  final displayName =
      player.name.isNotEmpty ? player.name : 'Player ${index + 1}';
  final nameStyle = TextStyle(
    color: isFolded ? Colors.white70 : Colors.white,
    fontSize: 11 * uiSizeMultiplier,
    fontWeight: FontWeight.w600,
    decoration: isFolded ? TextDecoration.lineThrough : TextDecoration.none,
    decorationColor: isFolded ? Colors.white54 : null,
    decorationThickness: isFolded ? 2 * uiSizeMultiplier : null,
  );
  final namePainter = TextPainter(
    text: TextSpan(
      text: displayName,
      style: nameStyle,
    ),
    textDirection: TextDirection.ltr,
    maxLines: 1,
    ellipsis: '…',
  );
  namePainter.layout(maxWidth: radius * 2.6);

  // Name label positioning: All players use the same pattern
  // - Name directly below the player circle (chips are above the circle)
  final spacingFromCircle = _stackSpacing(
    radius,
    uiSizeMultiplier,
    textHeight: namePainter.height,
  );
  double nameY = y + radius + spacingFromCircle;

  // Clamp name label so it does not go off the bottom of the viewport
  final minNameY = viewportBounds.top + 2.0 * uiSizeMultiplier;
  final maxNameY =
      viewportBounds.bottom - namePainter.height - 4.0 * uiSizeMultiplier;
  if (nameY < minNameY) nameY = minNameY;
  if (nameY > maxNameY) nameY = maxNameY;
  namePainter.paint(
    canvas,
    Offset(x - namePainter.width / 2, nameY),
  );

  // Draw role badges to the left of the player circle
  drawRoleBadges(canvas, x, y, radius, badges, isHero, uiSizeMultiplier);
}

void drawRoleBadges(
    Canvas canvas,
    double centerX,
    double centerY,
    double radius,
    List<SeatBadge> badges,
    bool isHero,
    double uiSizeMultiplier) {
  if (badges.isEmpty) return;

  final double badgeHeight = 18.0 * uiSizeMultiplier;
  final double horizontalPadding = 8.0 * uiSizeMultiplier;
  final double gap = 5.0 * uiSizeMultiplier;
  final textStyle = TextStyle(
    color: Colors.black,
    fontSize: 11 * uiSizeMultiplier,
    fontWeight: FontWeight.w900,
  );

  final layouts = <BadgeLayout>[];
  for (final badge in badges) {
    final painter = TextPainter(
      text: TextSpan(text: badge.label, style: textStyle),
      textDirection: TextDirection.ltr,
    )..layout();
    final width = painter.width + horizontalPadding * 2;
    layouts.add(BadgeLayout(badge, painter, width));
  }

  // Position badges to the right of the player circle, 30 degrees south (downward)
  final spacingFromCircle =
      8.0 * uiSizeMultiplier; // Gap between player circle and badges
  // 30 degrees south from right = 0° + 30° = 30° = π/6 radians
  const angleRadians = math.pi / 6; // 30 degrees
  final distanceFromCenter = radius + spacingFromCircle;
  // Position the leftmost badge edge at the calculated angle
  final badgeLeftEdgeX = centerX + distanceFromCenter * math.cos(angleRadians);
  final badgeLeftEdgeY = centerY + distanceFromCenter * math.sin(angleRadians);
  double drawX = badgeLeftEdgeX; // Position badges extending rightward
  final drawY =
      badgeLeftEdgeY - badgeHeight / 2; // Vertically centered at the angle

  for (final layout in layouts) {
    final rect = RRect.fromRectAndRadius(
      Rect.fromLTWH(drawX, drawY, layout.width, badgeHeight),
      const Radius.circular(6),
    );

    // Add subtle shadow for depth
    final shadowPaint = Paint()
      ..color = Colors.black.withOpacity(0.3)
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 2);
    canvas.drawRRect(rect, shadowPaint);

    // Draw badge background
    final paint = Paint()..color = layout.badge.color.withOpacity(0.95);
    canvas.drawRRect(rect, paint);

    layout.textPainter.paint(
      canvas,
      Offset(
        drawX + (layout.width - layout.textPainter.width) / 2,
        drawY + (badgeHeight - layout.textPainter.height) / 2,
      ),
    );
    drawX += layout.width + gap;
  }
}

void drawCurrentTimebank(
    Canvas canvas,
    Size size,
    UiGameState gameState,
    String currentPlayerId,
    double centerX,
    double centerY,
    double tableRadiusX,
    double tableRadiusY,
    double uiSizeMultiplier,
    {double playerOffset = _maxPlayerOffset,
    Rect? clampBounds,
    double? minSeatTop}) {
  if (gameState.turnDeadlineUnixMs <= 0) return;
  final nowMs = DateTime.now().millisecondsSinceEpoch;
  final remMs = (gameState.turnDeadlineUnixMs - nowMs).clamp(0, 1 << 30);
  final remSec = remMs / 1000.0;
  if (remSec <= 0) return;

  final players = gameState.players;
  if (players.isEmpty) return;
  final count = players.length;
  final heroIndex = players.indexWhere((p) => p.id == currentPlayerId);
  final idx = players.indexWhere((p) => p.id == gameState.currentPlayerId);
  if (idx < 0) return;

  final bounds = clampBounds ?? pokerViewportRect(size);
  final playerRadius = kPlayerRadius * uiSizeMultiplier;
  final effectivePlayerOffset =
      playerOffset + math.max(0, (playerRadius - kPlayerRadius) * 0.9);
  final pos = _positionForSeat(
    idx,
    heroIndex,
    count,
    Offset(centerX, centerY),
    tableRadiusX + effectivePlayerOffset,
    tableRadiusY + effectivePlayerOffset,
    bounds,
    minSeatTop,
    uiSizeMultiplier: uiSizeMultiplier,
    playerRadius: playerRadius,
  );

  final tbText = TextPainter(
    text: TextSpan(
      text: '⏱ ${remSec.toStringAsFixed(1)}s',
      style: TextStyle(
          color: Colors.white,
          fontSize: 11 * uiSizeMultiplier,
          fontWeight: FontWeight.w700),
    ),
    textDirection: TextDirection.ltr,
  )..layout();

  final badgeW = tbText.width + 12 * uiSizeMultiplier;
  final badgeH = 18.0 * uiSizeMultiplier;

  // Position timebank above role badges (consistent for all players)
  final isHero = (idx == heroIndex);
  double bx, by;

  // Calculate where role badges are positioned (same logic as drawRoleBadges)
  final spacingFromCircle = 8.0 * uiSizeMultiplier;
  const angleRadians = math.pi / 6; // 30 degrees
  final distanceFromCenter = playerRadius + spacingFromCircle;
  final badgeLeftEdgeX = pos.dx + distanceFromCenter * math.cos(angleRadians);
  final badgeLeftEdgeY = pos.dy + distanceFromCenter * math.sin(angleRadians);

  // Calculate total width of badges for current player
  final currentPlayer = players[idx];
  final badges = <String>[];
  if (currentPlayer.isDealer) badges.add('D');
  if (currentPlayer.isSmallBlind) badges.add('SB');
  if (currentPlayer.isBigBlind) badges.add('BB');
  if (currentPlayer.isAllIn) badges.add('ALL-IN');

  double totalBadgeWidth = 0.0;
  final roleBadgeHeight = 18.0 * uiSizeMultiplier;
  final horizontalPadding = 8.0 * uiSizeMultiplier;
  final gap = 5.0 * uiSizeMultiplier;
  final textStyle = TextStyle(
    color: Colors.black,
    fontSize: 11 * uiSizeMultiplier,
    fontWeight: FontWeight.w900,
  );

  for (final badgeLabel in badges) {
    final painter = TextPainter(
      text: TextSpan(text: badgeLabel, style: textStyle),
      textDirection: TextDirection.ltr,
    )..layout();
    totalBadgeWidth += painter.width + horizontalPadding * 2;
    if (badgeLabel != badges.last) {
      totalBadgeWidth += gap;
    }
  }

  // Position timebank above the badges, centered on the badge area
  if (totalBadgeWidth == 0) return; // No badges, don't show timebank

  bx = badgeLeftEdgeX;
  // Position above badges with a gap - larger gap for hero to avoid buttons
  final gapAboveBadges = (isHero ? 30.0 : 4.0) * uiSizeMultiplier;
  final badgeCenterY = badgeLeftEdgeY - roleBadgeHeight / 2;
  by = badgeCenterY - gapAboveBadges - badgeH;

  // Ensure timebank doesn't clip at screen edges
  final edgePadding = 2.0 * uiSizeMultiplier;
  if (bx < bounds.left + edgePadding) bx = bounds.left + edgePadding;
  if (bx + badgeW > bounds.right - edgePadding)
    bx = bounds.right - badgeW - edgePadding;
  if (by < bounds.top + edgePadding) by = bounds.top + edgePadding;
  if (by + badgeH > bounds.bottom - edgePadding)
    by = bounds.bottom - badgeH - edgePadding;

  final badgeRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(bx, by, badgeW, badgeH),
    Radius.circular(8 * uiSizeMultiplier),
  );
  final tbBg = Paint()..color = Colors.black.withOpacity(0.9);
  canvas.drawRRect(badgeRect, tbBg);
  tbText.paint(
      canvas,
      Offset(
          bx + (badgeW - tbText.width) / 2, by + (badgeH - tbText.height) / 2));
}

// Helper classes for badge management
class SeatBadge {
  const SeatBadge(this.label, this.color);

  final String label;
  final Color color;
}

class BadgeLayout {
  BadgeLayout(this.badge, this.textPainter, this.width);

  final SeatBadge badge;
  final TextPainter textPainter;
  final double width;
}

// Helpers used by overlays to compute positions within the 16:9 viewport
Rect pokerViewportRect(Size size) {
  const double aspect = 16 / 9;
  final double containerAspect =
      size.width / (size.height == 0 ? 1 : size.height);
  double w, h, left, top;
  if (containerAspect > aspect) {
    h = size.height;
    w = h * aspect;
    left = (size.width - w) / 2;
    top = 0;
  } else {
    w = size.width;
    h = w / aspect;
    left = 0;
    top = (size.height - h) / 2;
  }
  return Rect.fromLTWH(left, top, w, h);
}

Map<String, Offset> seatPositionsFor(
  List<UiPlayer> ps,
  String heroId,
  Offset center,
  double ringRadiusX,
  double ringRadiusY, {
  Rect? clampBounds,
  double? minSeatTop,
  double uiSizeMultiplier = 1.0,
}) {
  final map = <String, Offset>{};
  if (ps.isEmpty) return map;
  final count = ps.length;
  final heroIndex = ps.indexWhere((p) => p.id == heroId);
  final playerRadius = kPlayerRadius * uiSizeMultiplier;
  final sizeAwareOffset =
      math.max(0, (playerRadius - kPlayerRadius) * 0.9);
  final ringX = ringRadiusX + sizeAwareOffset;
  final ringY = ringRadiusY + sizeAwareOffset;

  for (int i = 0; i < count; i++) {
    final pos = _positionForSeat(
      i,
      heroIndex,
      count,
      center,
      ringX,
      ringY,
      clampBounds,
      minSeatTop,
      uiSizeMultiplier: uiSizeMultiplier,
      playerRadius: playerRadius,
    );
    map[ps[i].id] = Offset(pos.dx, pos.dy - playerRadius);
  }
  return map;
}
