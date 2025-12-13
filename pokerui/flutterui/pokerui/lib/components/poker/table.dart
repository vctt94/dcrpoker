import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'cards.dart';

// Canvas-based table and player drawing utilities for CustomPainter usage
const double kPlayerRadius = 30.0;
const double _minPlayerOffset = 32.0;
const double _maxPlayerOffset = 50.0;
const double _edgePadding = 12.0;
// Reserve vertical room for stacked pot/current bet labels so the top seat does not overlap them.
const double _topOverlaySafeHeight = 96.0;

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
  final center = Offset(viewport.left + viewport.width / 2, viewport.top + viewport.height / 2);
  final playerOffset = _playerOffsetForViewport(viewport);

  const desiredMinRadiusX = 180.0;
  const desiredMinRadiusY = 130.0;

  final availableX = (viewport.width / 2) - (playerOffset + kPlayerRadius + _edgePadding);
  final availableY = (viewport.height / 2) - (playerOffset + kPlayerRadius + _edgePadding);

  double clampRadius(double target, double available, double minDesired) {
    final maxRadius = available.clamp(0.0, double.infinity);
    if (maxRadius <= 0) return 0;
    final minRadius = math.min(minDesired, maxRadius);
    return target.clamp(minRadius, maxRadius);
  }

  final tableRadiusX = clampRadius(viewport.width * 0.42, availableX, desiredMinRadiusX);
  final tableRadiusY = clampRadius(viewport.height * 0.34, availableY, desiredMinRadiusY);

  return TableLayout(
    viewport: viewport,
    center: center,
    tableRadiusX: tableRadiusX,
    tableRadiusY: tableRadiusY,
    playerOffset: playerOffset,
  );
}

void drawPokerTable(Canvas canvas, double centerX, double centerY, double tableRadiusX, double tableRadiusY) {
  // Table surface - draw as ellipse
  final tablePaint = Paint()
    ..color = const Color(0xFF0D4F3C)
    ..style = PaintingStyle.fill;
  
  final tableRect = Rect.fromCenter(
    center: Offset(centerX, centerY),
    width: tableRadiusX * 2,
    height: tableRadiusY * 2,
  );
  canvas.drawOval(tableRect, tablePaint);
  
  // Table border
  final borderPaint = Paint()
    ..color = const Color(0xFF8B4513)
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
  {double? playerOffsetOverride, Rect? clampBounds}
) {
  const playerRadius = kPlayerRadius;
  final playerOffset = playerOffsetOverride ??
      _playerOffsetForViewport(clampBounds ?? Rect.fromLTWH(0, 0, size.width, size.height));
  final clampRect = clampBounds ?? Rect.fromLTWH(0, 0, size.width, size.height);
  final count = players.length;
  if (count == 0) return;

  // Find hero index for positioning
  final heroIndex = players.indexWhere((p) => p.id == currentPlayerId);

  for (int i = 0; i < count; i++) {
    final player = players[i];
    final angle = _angleForPlayerIndex(i, heroIndex, count);
    
    // Position players on ellipse perimeter
    // For ellipse: x = centerX + radiusX * cos(angle), y = centerY + radiusY * sin(angle)
    final rawX = centerX + (tableRadiusX + playerOffset) * math.cos(angle);
    final rawY = centerY + (tableRadiusY + playerOffset) * math.sin(angle);
    // Ensure players don't get cut off at edges (with padding for badges/cards)
    final padding = playerRadius + playerOffset + 12.0; // Extra space for badges and cards
    final maxY = clampRect.bottom - padding;
    final minY = math.min(clampRect.top + math.max(padding, _topOverlaySafeHeight + playerRadius), maxY);
    final playerX = rawX.clamp(clampRect.left + padding, clampRect.right - padding);
    final playerY = rawY.clamp(minY, maxY);

    drawPlayer(
      canvas,
      playerX,
      playerY,
      playerRadius,
      player,
      i,
      angle,
      currentPlayerId,
      gameState,
    );

    if (player.id != currentPlayerId) {
      // Skip rendering hole cards for folded opponents to avoid implying they are still in-hand.
      if (player.folded) {
        continue;
      }
      final hasAnyCards = player.hand.isNotEmpty;
      if (gameState.phase == pr.GamePhase.SHOWDOWN) {
        if (hasAnyCards) {
          // Reveal known opponent hands near their seat with a subtle slide-in.
          const cw = 18.0;
          const ch = cw * 1.4;
          const gap = 4.0;
          final startX = playerX - cw - gap / 2;
          final baseY = playerY - playerRadius - ch - 6;
          final now = DateTime.now().millisecondsSinceEpoch;
          final elapsed = (now - showdownStartMs - i * 120);
          final t = (elapsed / 450.0).clamp(0.0, 1.0);
          final y = baseY + (1.0 - t) * 14.0;
          drawCardFace(canvas, startX, y, cw, ch, player.hand[0]);
          if (player.hand.length > 1) {
            drawCardFace(canvas, startX + cw + gap, y, cw, ch, player.hand[1]);
          }
        } else {
          // If still hidden at showdown, show subtle backs.
          const cw = 16.0;
          const ch = cw * 1.4;
          const gap = 4.0;
          final startX = playerX - cw - gap / 2;
          final y = playerY - playerRadius - ch - 6;
          drawCardBack(canvas, startX, y, cw, ch);
          drawCardBack(canvas, startX + cw + gap, y, cw, ch);
        }
      } else if (!hasAnyCards && (gameState.phase != pr.GamePhase.WAITING && gameState.phase != pr.GamePhase.NEW_HAND_DEALING)) {
        // Non-showdown phases: use backs to indicate in-hand cards for opponents.
        const cw = 16.0;
        const ch = cw * 1.4;
        const gap = 4.0;
        final startX = playerX - cw - gap / 2;
        final y = playerY - playerRadius - ch - 6; // place just above the seat circle
        drawCardBack(canvas, startX, y, cw, ch);
        drawCardBack(canvas, startX + cw + gap, y, cw, ch);
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
  double angle,
  String currentPlayerId,
  UiGameState gameState,
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
    canvas.drawCircle(Offset(x, y), radius + 4, haloPaint);
  }
  
  // Player border
  final borderPaint = Paint()
    ..color = player.isDisconnected
        ? Colors.orangeAccent
        : (isFolded ? Colors.white24.withOpacity(0.6) : (isCurrent ? Colors.yellowAccent : Colors.white24))
    ..style = PaintingStyle.stroke
    ..strokeWidth = isCurrent ? 2.5 : 1.5;
  
  canvas.drawCircle(Offset(x, y), radius, borderPaint);
  
  // Dim folded players with an overlay
  if (isFolded) {
    final foldOverlay = Paint()..color = Colors.black.withOpacity(0.45);
    canvas.drawCircle(Offset(x, y), radius, foldOverlay);
    // Keep a red marker around folded players for quick recognition.
    final foldRing = Paint()
      ..color = Colors.redAccent.withOpacity(0.9)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.0;
    canvas.drawCircle(Offset(x, y), radius + 3, foldRing);
    final arrowPaint = Paint()
      ..color = Colors.redAccent.withOpacity(0.85)
      ..style = PaintingStyle.fill;
    final arrow = Path()
      ..moveTo(x, y + radius + 6)
      ..lineTo(x - 7, y + radius + 16)
      ..lineTo(x + 7, y + radius + 16)
      ..close();
    canvas.drawPath(arrow, arrowPaint);
  }
  
  // Player name (show more characters)
  final displayName = player.name.isNotEmpty
      ? player.name
      : 'Player ${index + 1}';
  final nameStyle = TextStyle(
    color: isFolded ? Colors.white70 : Colors.white,
    fontSize: 13,
    fontWeight: FontWeight.w800,
    decoration: isFolded ? TextDecoration.lineThrough : TextDecoration.none,
    decorationColor: isFolded ? Colors.white54 : null,
    decorationThickness: isFolded ? 2 : null,
  );
  final textPainter = TextPainter(
    text: TextSpan(
      text: displayName,
      style: nameStyle,
    ),
    textDirection: TextDirection.ltr,
    maxLines: 1,
    ellipsis: '…',
  );
  textPainter.layout(maxWidth: 98.0);
  textPainter.paint(
    canvas,
    Offset(x - textPainter.width / 2, y - textPainter.height / 2),
  );

  // Use blind positions from server instead of calculating client-side
  final badges = <SeatBadge>[];
  
  if (player.isDealer) {
    badges.add(const SeatBadge('D', Colors.amber));
  }
  if (player.isSmallBlind) {
    badges.add(const SeatBadge('SB', Colors.cyan));
  }
  if (player.isBigBlind) {
    badges.add(const SeatBadge('BB', Colors.pinkAccent));
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
        style: const TextStyle(
          color: Colors.white,
          fontSize: 10,
          fontWeight: FontWeight.w600,
        ),
      ),
      textDirection: TextDirection.ltr,
    );
    chipText.layout();
    
    // Draw chip badge background
    final chipBadgeWidth = chipText.width + 12;
    const chipBadgeHeight = 16.0;
    final chipBadgeY = y + radius + 8;
    final chipBadgeRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(
        x - chipBadgeWidth / 2,
        chipBadgeY,
        chipBadgeWidth,
        chipBadgeHeight,
      ),
      const Radius.circular(8),
    );
    final chipBgPaint = Paint()..color = Colors.black.withOpacity(0.7);
    canvas.drawRRect(chipBadgeRect, chipBgPaint);
    
    chipText.paint(
      canvas,
      Offset(x - chipText.width / 2, chipBadgeY + 2),
    );
  }
  
  // Draw role badges to the left of the player circle
  drawRoleBadges(canvas, x, y, radius, badges, isHero, angle);
}

void drawRoleBadges(Canvas canvas, double centerX, double centerY, double radius, List<SeatBadge> badges, bool isHero, double angle) {
  if (badges.isEmpty) return;

  const double badgeHeight = 18.0;
  const double horizontalPadding = 8.0;
  const double gap = 5.0;
  const textStyle = TextStyle(
    color: Colors.black,
    fontSize: 11,
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
  const spacingFromCircle = 8.0; // Gap between player circle and badges
  // 30 degrees south from right = 0° + 30° = 30° = π/6 radians
  const angleRadians = math.pi / 6; // 30 degrees
  final distanceFromCenter = radius + spacingFromCircle;
  // Position the leftmost badge edge at the calculated angle
  final badgeLeftEdgeX = centerX + distanceFromCenter * math.cos(angleRadians);
  final badgeLeftEdgeY = centerY + distanceFromCenter * math.sin(angleRadians);
  double drawX = badgeLeftEdgeX; // Position badges extending rightward
  final drawY = badgeLeftEdgeY - badgeHeight / 2; // Vertically centered at the angle
  
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
  {double playerOffset = _maxPlayerOffset, Rect? clampBounds}
) {
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

  final angle = _angleForPlayerIndex(idx, heroIndex, count);
  final bounds = clampBounds ?? pokerViewportRect(size);
  final padding = kPlayerRadius + playerOffset + 12.0;

  const playerRadius = kPlayerRadius;
  final rawX = centerX + (tableRadiusX + playerOffset) * math.cos(angle);
  final rawY = centerY + (tableRadiusY + playerOffset) * math.sin(angle);
  final maxY = bounds.bottom - padding;
  final minY = math.min(bounds.top + math.max(padding, _topOverlaySafeHeight + playerRadius), maxY);
  final playerX = rawX.clamp(bounds.left + padding, bounds.right - padding);
  final playerY = rawY.clamp(minY, maxY);

  final tbText = TextPainter(
    text: TextSpan(
      text: '⏱ ${remSec.toStringAsFixed(1)}s',
      style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w700),
    ),
    textDirection: TextDirection.ltr,
  )..layout();

  final badgeW = tbText.width + 12;
  const badgeH = 18.0;
  
  // Position timebank above role badges (consistent for all players)
  final isHero = (idx == heroIndex);
  double bx, by;
  
  // Calculate where role badges are positioned (same logic as drawRoleBadges)
  const spacingFromCircle = 8.0;
  const angleRadians = math.pi / 6; // 30 degrees
  const distanceFromCenter = playerRadius + spacingFromCircle;
  final badgeLeftEdgeX = playerX + distanceFromCenter * math.cos(angleRadians);
  final badgeLeftEdgeY = playerY + distanceFromCenter * math.sin(angleRadians);
  
  // Calculate total width of badges for current player
  final currentPlayer = players[idx];
  final badges = <String>[];
  if (currentPlayer.isDealer) badges.add('D');
  if (currentPlayer.isSmallBlind) badges.add('SB');
  if (currentPlayer.isBigBlind) badges.add('BB');
  if (currentPlayer.isAllIn) badges.add('ALL-IN');
  
  double totalBadgeWidth = 0.0;
  const roleBadgeHeight = 18.0;
  const horizontalPadding = 8.0;
  const gap = 5.0;
  const textStyle = TextStyle(
    color: Colors.black,
    fontSize: 11,
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
  
  // Center timebank on the badge area, with left margin (more for hero)
  final leftMargin = isHero ? 15.0 : 0.0;
  bx = badgeLeftEdgeX + (totalBadgeWidth - badgeW) / 2 - leftMargin;
  // Position above badges with a gap - larger gap for hero to avoid buttons
  final gapAboveBadges = isHero ? 30.0 : 4.0;
  final badgeCenterY = badgeLeftEdgeY - roleBadgeHeight / 2;
  by = badgeCenterY - gapAboveBadges - badgeH;
  
  // Ensure timebank doesn't clip at screen edges
  if (bx < bounds.left + 2) bx = bounds.left + 2;
  if (bx + badgeW > bounds.right - 2) bx = bounds.right - badgeW - 2;
  if (by < bounds.top + 2) by = bounds.top + 2;
  if (by + badgeH > bounds.bottom - 2) by = bounds.bottom - badgeH - 2;

  final badgeRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(bx, by, badgeW, badgeH),
    const Radius.circular(8),
  );
  final tbBg = Paint()..color = Colors.black.withOpacity(0.9);
  canvas.drawRRect(badgeRect, tbBg);
  tbText.paint(canvas, Offset(bx + (badgeW - tbText.width) / 2, by + (badgeH - tbText.height) / 2));
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
  final double containerAspect = size.width / (size.height == 0 ? 1 : size.height);
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
  double playerOffset = _minPlayerOffset,
}) {
  final map = <String, Offset>{};
  if (ps.isEmpty) return map;
  final count = ps.length;
  final heroIndex = ps.indexWhere((p) => p.id == heroId);
  const playerRadius = kPlayerRadius;

  for (int i = 0; i < count; i++) {
    final angle = _angleForPlayerIndex(i, heroIndex, count);
    // Position on ellipse perimeter
    var x = center.dx + ringRadiusX * math.cos(angle);
    var y = center.dy + ringRadiusY * math.sin(angle);

    if (clampBounds != null) {
      final padding = playerRadius + playerOffset + 12.0;
      final maxY = clampBounds.bottom - padding;
      final minY = math.min(clampBounds.top + math.max(padding, _topOverlaySafeHeight + playerRadius), maxY);
      x = x.clamp(clampBounds.left + padding, clampBounds.right - padding);
      y = y.clamp(minY, maxY);
    }

    map[ps[i].id] = Offset(x, y - playerRadius);
  }
  return map;
}
