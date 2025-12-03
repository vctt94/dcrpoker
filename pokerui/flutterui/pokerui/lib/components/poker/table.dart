import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'cards.dart';

// Canvas-based table and player drawing utilities for CustomPainter usage

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
) {
  const playerRadius = 30.0;
  final count = players.length;
  if (count == 0) return;

  // Find hero index for positioning
  final heroIndex = players.indexWhere((p) => p.id == currentPlayerId);

  for (int i = 0; i < count; i++) {
    final player = players[i];
    
    // Position hero at the bottom (pi/2 radians = 90 degrees = bottom)
    // Other players arranged around the table
    double angle;
    if (i == heroIndex) {
      // Hero always at bottom
      angle = math.pi / 2;
    } else if (heroIndex == -1) {
      // No hero found, distribute evenly
      angle = (i * 2 * math.pi) / count;
    } else {
      // Arrange other players around the table
      // Adjust index to account for hero being at bottom
      final adjustedIndex = i > heroIndex ? i - 1 : i;
      final otherCount = count - 1; // excluding hero
      if (otherCount > 0) {
        // Distribute others around the top half/sides
        // Start from left (pi) and go counterclockwise, skipping bottom (pi/2)
        final step = (2 * math.pi) / (otherCount + 1);
        angle = math.pi + (adjustedIndex + 1) * step;
      } else {
        angle = (i * 2 * math.pi) / count;
      }
    }
    
    // Position players on ellipse perimeter
    // For ellipse: x = centerX + radiusX * cos(angle), y = centerY + radiusY * sin(angle)
    final playerOffset = 50.0; // Distance from table edge
    final rawX = centerX + (tableRadiusX + playerOffset) * math.cos(angle);
    final rawY = centerY + (tableRadiusY + playerOffset) * math.sin(angle);
    // Ensure players don't get cut off at edges (with padding for badges/cards)
    final padding = playerRadius + 60.0; // Extra space for badges and cards
    final playerX = rawX.clamp(padding, size.width - padding);
    final playerY = rawY.clamp(padding, size.height - padding);

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
  // Compute turn highlight based on authoritative currentPlayerId from
  // the game state to avoid transient races in per-player isTurn flags.
  final isCurrent = player.id == gameState.currentPlayerId;
  const heroColor = Color(0xFF2E6DD8);
  final otherColor = Colors.grey.shade700;
  
  // Player circle
  final playerPaint = Paint()
    ..color = player.isDisconnected
        ? Colors.red.shade700
        : (isHero ? heroColor : otherColor)
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
        : (isCurrent ? Colors.yellowAccent : Colors.white24)
    ..style = PaintingStyle.stroke
    ..strokeWidth = isCurrent ? 2.5 : 1.5;
  
  canvas.drawCircle(Offset(x, y), radius, borderPaint);
  
  // Player name (show more characters)
  final displayName = player.name.isNotEmpty 
      ? (player.name.length > 2 ? player.name.substring(0, 2).toUpperCase() : player.name.toUpperCase())
      : 'P${index + 1}';
  final textPainter = TextPainter(
    text: TextSpan(
      text: displayName,
      style: const TextStyle(
        color: Colors.white,
        fontSize: 13,
        fontWeight: FontWeight.w800,
      ),
    ),
    textDirection: TextDirection.ltr,
  );
  textPainter.layout();
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

  double angle;
  if (idx == heroIndex) {
    angle = math.pi / 2;
  } else if (heroIndex == -1) {
    angle = (idx * 2 * math.pi) / count;
  } else {
    final adjustedIndex = idx > heroIndex ? idx - 1 : idx;
    final otherCount = count - 1;
    if (otherCount > 0) {
      final step = (2 * math.pi) / (otherCount + 1);
      angle = math.pi + (adjustedIndex + 1) * step;
    } else {
      angle = (idx * 2 * math.pi) / count;
    }
  }

  const playerRadius = 30.0;
  const playerOffset = 50.0;
  final playerX = centerX + (tableRadiusX + playerOffset) * math.cos(angle);
  final playerY = centerY + (tableRadiusY + playerOffset) * math.sin(angle);

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
  if (bx < 2) bx = 2;
  if (bx + badgeW > size.width - 2) bx = size.width - badgeW - 2;
  if (by < 2) by = 2;
  if (by + badgeH > size.height - 2) by = size.height - badgeH - 2;

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

Map<String, Offset> seatPositionsFor(List<UiPlayer> ps, String heroId, Offset center, double ringRadiusX, double ringRadiusY) {
  final map = <String, Offset>{};
  if (ps.isEmpty) return map;
  final count = ps.length;
  final heroIndex = ps.indexWhere((p) => p.id == heroId);
  const playerRadius = 30.0;

  for (int i = 0; i < count; i++) {
    double angle;
    if (i == heroIndex) {
      angle = math.pi / 2;
    } else if (heroIndex == -1) {
      angle = (i * 2 * math.pi) / count;
    } else {
      final adjustedIndex = i > heroIndex ? i - 1 : i;
      final otherCount = count - 1;
      if (otherCount > 0) {
        final step = (2 * math.pi) / (otherCount + 1);
        angle = math.pi + (adjustedIndex + 1) * step;
      } else {
        angle = (i * 2 * math.pi) / count;
      }
    }
    // Position on ellipse perimeter
    final x = center.dx + ringRadiusX * math.cos(angle);
    final y = center.dy + ringRadiusY * math.sin(angle);
    map[ps[i].id] = Offset(x, y - playerRadius);
  }
  return map;
}
