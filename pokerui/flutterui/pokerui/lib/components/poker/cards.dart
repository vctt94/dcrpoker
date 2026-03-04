import 'dart:math' as math;
import 'dart:ui';
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';

// Shared card rendering widgets to ensure a single source of truth
// for card visuals across the app (faces, backs, and flip animation).

String suitSym(String suit) {
  final s = suit.toLowerCase();
  if (s == 'hearts' || suit == '♥' || suit == '\u2665') return '♥';
  if (s == 'diamonds' || suit == '♦' || suit == '\u2666') return '♦';
  if (s == 'clubs' || suit == '♣' || suit == '\u2663') return '♣';
  if (s == 'spades' || suit == '♠' || suit == '\u2660') return '♠';
  return suit;
}

String cardId(pr.Card? c) {
  if (c == null) return 'blank';
  final v = (c.value.isEmpty ? '_' : c.value);
  final s = (c.suit.isEmpty ? '_' : suitSym(c.suit));
  return '$v$s';
}

Color suitColor(String suit, {CardColorTheme? cardTheme}) {
  final theme = cardTheme ?? CardColorTheme.standard;
  final s = suit.toLowerCase();
  if (s == 'hearts' || suit == '♥' || suit == '\u2665')
    return theme.heartsColor;
  if (s == 'diamonds' || suit == '♦' || suit == '\u2666')
    return theme.diamondsColor;
  if (s == 'clubs' || suit == '♣' || suit == '\u2663') return theme.clubsColor;
  if (s == 'spades' || suit == '♠' || suit == '\u2660')
    return theme.spadesColor;
  return Colors.black;
}

class CardFace extends StatelessWidget {
  const CardFace({super.key, required pr.Card? card, this.cardTheme})
      : _card = card;
  final pr.Card? _card;
  final CardColorTheme? cardTheme;

  @override
  Widget build(BuildContext context) {
    final value = _card?.value ?? '';
    final suit = _card?.suit ?? '';
    final suitSymbol = suitSym(suit);
    final suitTint = suitColor(suit, cardTheme: cardTheme);
    return RepaintBoundary(
      child: LayoutBuilder(
        builder: (context, c) {
          final w = c.maxWidth.clamp(20.0, double.infinity);
          final h = c.maxHeight.clamp(28.0, double.infinity);
          final rankFs = (w * 0.30).clamp(10.0, 28.0).toDouble();
          final suitFs = (w * 0.26).clamp(8.0, 24.0).toDouble();
          final centerFs = (math.min(w, h) * 0.35).clamp(12.0, 40.0).toDouble();
          final textColor = suitTint;
          final borderColor = Colors.black87;
          return Container(
            constraints: const BoxConstraints(
              minWidth: 20.0,
              minHeight: 28.0,
            ),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: borderColor, width: 2),
              boxShadow: [
                BoxShadow(
                    color: Colors.black.withOpacity(0.30),
                    blurRadius: 6,
                    spreadRadius: 1),
              ],
            ),
            child: Padding(
              padding: const EdgeInsets.all(4.0),
              child: Stack(
                children: [
                  Align(
                    alignment: Alignment.topLeft,
                    child: FittedBox(
                      alignment: Alignment.topLeft,
                      fit: BoxFit.scaleDown,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(value,
                              style: TextStyle(
                                  color: textColor,
                                  fontSize: rankFs,
                                  fontWeight: FontWeight.w900)),
                          Text(suitSymbol,
                              style: TextStyle(
                                  color: textColor,
                                  fontSize: suitFs,
                                  fontWeight: FontWeight.w700)),
                        ],
                      ),
                    ),
                  ),
                  Align(
                    alignment: Alignment.bottomRight,
                    child: Transform.rotate(
                      angle: math.pi,
                      child: FittedBox(
                        alignment: Alignment.topLeft,
                        fit: BoxFit.scaleDown,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(value,
                                style: TextStyle(
                                    color: textColor,
                                    fontSize: rankFs,
                                    fontWeight: FontWeight.w900)),
                            Text(suitSymbol,
                                style: TextStyle(
                                    color: textColor,
                                    fontSize: suitFs,
                                    fontWeight: FontWeight.w700)),
                          ],
                        ),
                      ),
                    ),
                  ),
                  Center(
                    child: FittedBox(
                      fit: BoxFit.scaleDown,
                      child: Text(
                        suitSymbol,
                        style: TextStyle(
                            color: textColor,
                            fontSize: centerFs,
                            fontWeight: FontWeight.w600),
                      ),
                    ),
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

class CardBack extends StatelessWidget {
  const CardBack({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF1B1E2C), Color(0xFF0E111A)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.black, width: 2),
        boxShadow: [
          BoxShadow(
              color: Colors.black.withOpacity(0.30),
              blurRadius: 6,
              spreadRadius: 1),
        ],
      ),
    );
  }
}

class FlipCard extends StatelessWidget {
  const FlipCard(
      {super.key, required this.faceUp, required this.card, this.cardTheme});
  final bool faceUp;
  final pr.Card? card;
  final CardColorTheme? cardTheme;

  @override
  Widget build(BuildContext context) {
    final id = cardId(card);
    final frontKey = ValueKey('face_$id');
    final backKey = ValueKey('back_$id');
    final front = CardFace(card: card, key: frontKey, cardTheme: cardTheme);
    final back = CardBack(key: backKey);
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 280),
      transitionBuilder: (child, anim) {
        // Backface-correct 3D flip so text never mirrors.
        final rotate = Tween(begin: math.pi, end: 0.0).animate(anim);
        return AnimatedBuilder(
          animation: rotate,
          child: child,
          builder: (context, child) {
            final isUnder = (child!.key != (faceUp ? frontKey : backKey));
            var tilt = (anim.value - 0.5).abs() - 0.5;
            tilt *= 0.02;
            final angle =
                isUnder ? math.min(rotate.value, math.pi / 2) : rotate.value;
            return Transform(
              alignment: Alignment.center,
              transform: Matrix4.identity()
                ..setEntry(3, 2, 0.001)
                ..rotateY(angle)
                ..rotateZ(tilt),
              child: child,
            );
          },
        );
      },
      child: faceUp ? front : back,
      layoutBuilder: (currentChild, previousChildren) => Stack(children: [
        ...previousChildren,
        if (currentChild != null) currentChild,
      ]),
    );
  }
}

class HeroCardFlipOverlay extends StatelessWidget {
  const HeroCardFlipOverlay({
    super.key,
    required this.cards,
    required this.showFace,
    this.onToggle,
    this.toggleShown,
    this.cardTheme,
  });
  final List<pr.Card> cards;
  final bool showFace;
  final VoidCallback? onToggle;
  final bool? toggleShown;
  final CardColorTheme? cardTheme;

  @override
  Widget build(BuildContext context) {
    final cardSizeMultiplier = cardSizeMultiplierFromKey(context.cardSize);
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size);
      final box = layout.viewport;
      final baseCw = math.max(math.min(box.width * 0.06, 56.0), 40.0);
      final cw = baseCw * cardSizeMultiplier;
      final ch = cw * 1.4;
      final gap = cw * 0.12;
      final centerX = layout.center.dx;
      final centerY = layout.center.dy;
      final uiSizeMultiplier = uiSizeMultiplierFromKey(context.uiSize);

      // --- Anchor hero cards just above the hero seat ---
      // Compute toggle/header metrics first so we know the full tray height.
      final headerHeight = (cw * 0.45).clamp(16.0, 24.0);
      final headerGap = (4.0 * uiSizeMultiplier).clamp(2.0, 6.0);
      final trayContentH =
          ch + (onToggle != null ? headerGap + headerHeight : 0);

      // Hero seat position (must match _positionForSeat's hero push).
      // Clamp to canvasBounds so the hero seat can extend below the 16:9 zone.
      final canvas = layout.canvasBounds;
      final heroPush = layout.ringRadiusY * kHeroSeatExtraFraction;
      final seatPadding = kPlayerRadius + layout.playerOffset + 10.0;
      final heroSeatCenterY = (centerY + layout.ringRadiusY + heroPush)
          .clamp(canvas.top + seatPadding, canvas.bottom - seatPadding);
      final heroSeatTop = heroSeatCenterY - kPlayerRadius * uiSizeMultiplier;

      // Card tray sits directly above the hero seat with a small gap.
      final cardGapAboveHero = 10.0 * uiSizeMultiplier;
      var y = heroSeatTop - trayContentH - cardGapAboveHero;

      // Dealer zone bottom — cards must never overlap the pot badge.
      final potCenter =
          potChipCenter(layout, uiSizeMultiplier: uiSizeMultiplier);
      final potBadgeHalfH = 18.0 * uiSizeMultiplier;
      final dealerZoneBottom = potCenter.dy + potBadgeHalfH;

      final minPad = 16.0 * uiSizeMultiplier;
      final minY = dealerZoneBottom + minPad;
      final maxY = heroSeatTop - ch - minPad;
      if (minY <= maxY) {
        y = y.clamp(minY, maxY);
      } else {
        final available = heroSeatTop - dealerZoneBottom;
        y = dealerZoneBottom + (available - ch) / 2;
      }

      final x1 = centerX - cw - gap / 2;
      final x2 = centerX + gap / 2;
      final showing = toggleShown ?? showFace;
      final actionLabel = showing ? 'HIDE' : 'SHOW';
      final headerWidth = (cw * 2) + gap;
      final rawHeaderTop = y + ch + headerGap;
      final headerTop = rawHeaderTop > box.bottom - headerHeight - 2.0
          ? box.bottom - headerHeight - 2.0
          : rawHeaderTop;
      final iconSize = (headerHeight * 0.6).clamp(10.0, 18.0);
      final accent = showing ? Colors.amber : Colors.white70;
      final borderColor =
          showing ? Colors.amber.withOpacity(0.6) : Colors.white30;

      // Tray background wraps cards + toggle.
      final trayPadH = cw * 0.18;
      final trayPadTop = cw * 0.14;
      final trayPadBottom = cw * 0.10;
      final trayLeft = x1 - trayPadH;
      final trayTop = y - trayPadTop;
      final trayWidth = headerWidth + trayPadH * 2;
      final trayBottom =
          (onToggle != null ? headerTop + headerHeight : y + ch) +
              trayPadBottom;
      final trayHeight = trayBottom - trayTop;

      final children = <Widget>[
        Positioned(
          left: trayLeft,
          top: trayTop,
          width: trayWidth,
          height: trayHeight,
          child: IgnorePointer(
            child: Container(
              decoration: BoxDecoration(
                color: Colors.black.withOpacity(0.28),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(
                  color: Colors.white.withOpacity(0.06),
                  width: 1,
                ),
              ),
            ),
          ),
        ),
        Positioned(
            left: x1,
            top: y,
            width: cw,
            height: ch,
            child: FlipCard(
                faceUp: showFace,
                card: cards.isNotEmpty ? cards[0] : null,
                cardTheme: cardTheme)),
        Positioned(
            left: x2,
            top: y,
            width: cw,
            height: ch,
            child: FlipCard(
                faceUp: showFace,
                card: cards.length > 1 ? cards[1] : null,
                cardTheme: cardTheme)),
      ];

      return Stack(children: [
        ...children,
        if (onToggle != null)
          Positioned(
            left: x1,
            top: headerTop,
            width: headerWidth,
            height: headerHeight,
            child: Tooltip(
              message: actionLabel,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(headerHeight / 2),
                child: BackdropFilter(
                  filter: ImageFilter.blur(sigmaX: 6, sigmaY: 6),
                  child: Material(
                    color: Colors.white.withOpacity(0.12),
                    child: InkWell(
                      onTap: onToggle,
                      borderRadius: BorderRadius.circular(headerHeight / 2),
                      child: Container(
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          borderRadius: BorderRadius.circular(headerHeight / 2),
                          border: Border.all(color: borderColor),
                        ),
                        padding: EdgeInsets.symmetric(
                          horizontal: headerHeight * 0.55,
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(
                              showing ? Icons.visibility : Icons.visibility_off,
                              size: iconSize,
                              color: accent,
                            ),
                            SizedBox(width: headerHeight * 0.25),
                            Text(
                              actionLabel,
                              style: TextStyle(
                                color: accent,
                                fontSize:
                                    (headerHeight * 0.45).clamp(10.0, 14.0),
                                fontWeight: FontWeight.w700,
                                letterSpacing: 0.6,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ]);
    });
  }
}

// Canvas-based card drawing utilities for CustomPainter usage
void drawCardFace(Canvas canvas, double x, double y, double width,
    double height, pr.Card card,
    {CardColorTheme? cardTheme}) {
  // Card background
  final cardPaint = Paint()
    ..color = Colors.white
    ..style = PaintingStyle.fill;

  final cardRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(x, y, width, height),
    const Radius.circular(4),
  );
  canvas.drawRRect(cardRect, cardPaint);

  // Card border
  final borderPaint = Paint()
    ..color = Colors.black
    ..style = PaintingStyle.stroke
    ..strokeWidth = 1;

  canvas.drawRRect(cardRect, borderPaint);

  // Card content
  final textPainter = TextPainter(
    text: TextSpan(
      text: '${card.value}\n${getSuitSymbol(card.suit)}',
      style: TextStyle(
        color: getSuitColor(card.suit, cardTheme: cardTheme),
        fontSize: 10,
        fontWeight: FontWeight.bold,
      ),
    ),
    textDirection: TextDirection.ltr,
  );
  textPainter.layout();
  textPainter.paint(
    canvas,
    Offset(x + (width - textPainter.width) / 2,
        y + (height - textPainter.height) / 2),
  );
}

void drawCardBack(
    Canvas canvas, double x, double y, double width, double height) {
  // Card back background
  final backPaint = Paint()
    ..shader = const LinearGradient(
      colors: [Color(0xFF1B1E2C), Color(0xFF0E111A)],
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
    ).createShader(Rect.fromLTWH(x, y, width, height));

  final cardRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(x, y, width, height),
    const Radius.circular(4),
  );
  canvas.drawRRect(cardRect, backPaint);

  // Border
  final borderPaint = Paint()
    ..color = Colors.black
    ..style = PaintingStyle.stroke
    ..strokeWidth = 1;
  canvas.drawRRect(cardRect, borderPaint);

  // Minimal back pattern
  final pipPainter = TextPainter(
    text: const TextSpan(
      text: '♠',
      style: TextStyle(
          color: Colors.white70, fontSize: 12, fontWeight: FontWeight.bold),
    ),
    textDirection: TextDirection.ltr,
  );
  pipPainter.layout();
  pipPainter.paint(
    canvas,
    Offset(x + (width - pipPainter.width) / 2,
        y + (height - pipPainter.height) / 2),
  );
}

String getSuitSymbol(String suit) {
  switch (suit.toLowerCase()) {
    case 'hearts':
      return '♥';
    case 'diamonds':
      return '♦';
    case 'clubs':
      return '♣';
    case 'spades':
      return '♠';
    default:
      return suit;
  }
}

Color getSuitColor(String suit, {CardColorTheme? cardTheme}) {
  final theme = cardTheme ?? CardColorTheme.standard;
  final s = suit.toLowerCase();
  // Check for Unicode symbols first
  if (suit == '♥' || suit == '\u2665') {
    return theme.heartsColor;
  }
  if (suit == '♦' || suit == '\u2666') {
    return theme.diamondsColor;
  }
  if (suit == '♣' || suit == '\u2663') {
    return theme.clubsColor;
  }
  if (suit == '♠' || suit == '\u2660') {
    return theme.spadesColor;
  }
  // Then check lowercase strings
  switch (s) {
    case 'hearts':
      return theme.heartsColor;
    case 'diamonds':
      return theme.diamondsColor;
    case 'clubs':
      return theme.clubsColor;
    case 'spades':
      return theme.spadesColor;
    default:
      return Colors.black;
  }
}
