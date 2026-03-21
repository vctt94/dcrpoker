import 'dart:math' as math;
import 'dart:ui';
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/theme/colors.dart';

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

// ── Pip layout positions for number cards ──
// Normalized (x, y) positions where 0,0 is top-left and 1,1 is bottom-right
// of the pip area. Some bottom-half pips are drawn upside-down.
const Map<int, List<_PipPos>> _pipLayouts = {
  1: [_PipPos(0.5, 0.5, false)],
  2: [_PipPos(0.5, 0.2, false), _PipPos(0.5, 0.8, true)],
  3: [
    _PipPos(0.5, 0.2, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.5, 0.8, true)
  ],
  4: [
    _PipPos(0.3, 0.2, false),
    _PipPos(0.7, 0.2, false),
    _PipPos(0.3, 0.8, true),
    _PipPos(0.7, 0.8, true),
  ],
  5: [
    _PipPos(0.3, 0.2, false),
    _PipPos(0.7, 0.2, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.3, 0.8, true),
    _PipPos(0.7, 0.8, true),
  ],
  6: [
    _PipPos(0.3, 0.2, false),
    _PipPos(0.7, 0.2, false),
    _PipPos(0.3, 0.5, false),
    _PipPos(0.7, 0.5, false),
    _PipPos(0.3, 0.8, true),
    _PipPos(0.7, 0.8, true),
  ],
  7: [
    _PipPos(0.3, 0.2, false),
    _PipPos(0.7, 0.2, false),
    _PipPos(0.5, 0.35, false),
    _PipPos(0.3, 0.5, false),
    _PipPos(0.7, 0.5, false),
    _PipPos(0.3, 0.8, true),
    _PipPos(0.7, 0.8, true),
  ],
  8: [
    _PipPos(0.3, 0.2, false),
    _PipPos(0.7, 0.2, false),
    _PipPos(0.5, 0.35, false),
    _PipPos(0.3, 0.5, false),
    _PipPos(0.7, 0.5, false),
    _PipPos(0.5, 0.65, true),
    _PipPos(0.3, 0.8, true),
    _PipPos(0.7, 0.8, true),
  ],
  9: [
    _PipPos(0.3, 0.18, false),
    _PipPos(0.7, 0.18, false),
    _PipPos(0.3, 0.39, false),
    _PipPos(0.7, 0.39, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.3, 0.61, true),
    _PipPos(0.7, 0.61, true),
    _PipPos(0.3, 0.82, true),
    _PipPos(0.7, 0.82, true),
  ],
  10: [
    _PipPos(0.3, 0.15, false),
    _PipPos(0.7, 0.15, false),
    _PipPos(0.5, 0.28, false),
    _PipPos(0.3, 0.38, false),
    _PipPos(0.7, 0.38, false),
    _PipPos(0.3, 0.62, true),
    _PipPos(0.7, 0.62, true),
    _PipPos(0.5, 0.72, true),
    _PipPos(0.3, 0.85, true),
    _PipPos(0.7, 0.85, true),
  ],
};

class _PipPos {
  final double x, y;
  final bool inverted;
  const _PipPos(this.x, this.y, this.inverted);
}

int? _rankToCount(String value) {
  switch (value.toUpperCase()) {
    case 'A':
      return 1;
    case '2':
      return 2;
    case '3':
      return 3;
    case '4':
      return 4;
    case '5':
      return 5;
    case '6':
      return 6;
    case '7':
      return 7;
    case '8':
      return 8;
    case '9':
      return 9;
    case '10':
      return 10;
    default:
      return null;
  }
}

bool _isFaceCard(String value) {
  final v = value.toUpperCase();
  return v == 'J' || v == 'Q' || v == 'K';
}

// ─────────────────────────────────────────────
// CardFace Widget
// ─────────────────────────────────────────────

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
    final tint = suitColor(suit, cardTheme: cardTheme);

    return RepaintBoundary(
      child: LayoutBuilder(
        builder: (context, c) {
          final w = c.maxWidth.clamp(20.0, double.infinity);
          final h = c.maxHeight.clamp(28.0, double.infinity);
          final isSmall = w < 36;
          final rankFs = (w * 0.28).clamp(8.0, 22.0).toDouble();
          final suitFs = (w * 0.22).clamp(6.0, 18.0).toDouble();

          return Container(
            decoration: BoxDecoration(
              color: PokerColors.cardFace,
              borderRadius: BorderRadius.circular((w * 0.1).clamp(4.0, 10.0)),
              border: Border.all(color: const Color(0xFFD0D0D0), width: 1),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x40000000),
                  blurRadius: 6,
                  spreadRadius: 0.5,
                  offset: Offset(0, 2),
                ),
              ],
            ),
            child: ClipRRect(
              borderRadius: BorderRadius.circular((w * 0.1).clamp(4.0, 10.0)),
              child: Stack(
                children: [
                  // Corner index: top-left
                  Positioned(
                    left: w * 0.06,
                    top: h * 0.04,
                    child: _CornerIndex(
                      rank: value,
                      suit: suitSymbol,
                      color: tint,
                      rankSize: rankFs,
                      suitSize: suitFs * 0.85,
                    ),
                  ),
                  // Corner index: bottom-right (rotated 180)
                  Positioned(
                    right: w * 0.06,
                    bottom: h * 0.04,
                    child: Transform.rotate(
                      angle: math.pi,
                      child: _CornerIndex(
                        rank: value,
                        suit: suitSymbol,
                        color: tint,
                        rankSize: rankFs,
                        suitSize: suitFs * 0.85,
                      ),
                    ),
                  ),
                  // Center content
                  if (!isSmall)
                    Positioned.fill(
                      child: Padding(
                        padding: EdgeInsets.symmetric(
                          horizontal: w * 0.18,
                          vertical: h * 0.15,
                        ),
                        child: _CardCenter(
                          value: value,
                          suit: suitSymbol,
                          color: tint,
                          width: w,
                          height: h,
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

class _CornerIndex extends StatelessWidget {
  const _CornerIndex({
    required this.rank,
    required this.suit,
    required this.color,
    required this.rankSize,
    required this.suitSize,
  });
  final String rank, suit;
  final Color color;
  final double rankSize, suitSize;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Text(rank,
            style: TextStyle(
              color: color,
              fontSize: rankSize,
              fontWeight: FontWeight.w900,
              height: 1.1,
            )),
        Text(suit,
            style: TextStyle(
              color: color,
              fontSize: suitSize,
              fontWeight: FontWeight.w700,
              height: 1.0,
            )),
      ],
    );
  }
}

class _CardCenter extends StatelessWidget {
  const _CardCenter({
    required this.value,
    required this.suit,
    required this.color,
    required this.width,
    required this.height,
  });
  final String value, suit;
  final Color color;
  final double width, height;

  @override
  Widget build(BuildContext context) {
    final pipCount = _rankToCount(value);
    final pipSize = (math.min(width, height) * 0.18).clamp(8.0, 20.0);

    if (value.toUpperCase() == 'A') {
      return Center(
        child: Text(suit,
            style: TextStyle(
              color: color,
              fontSize: (math.min(width, height) * 0.45).clamp(14.0, 44.0),
              fontWeight: FontWeight.w600,
            )),
      );
    }

    if (_isFaceCard(value)) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(value,
                style: TextStyle(
                  color: color,
                  fontSize: (math.min(width, height) * 0.32).clamp(12.0, 32.0),
                  fontWeight: FontWeight.w800,
                )),
            Text(suit,
                style: TextStyle(
                  color: color,
                  fontSize: (math.min(width, height) * 0.2).clamp(8.0, 22.0),
                  fontWeight: FontWeight.w600,
                )),
          ],
        ),
      );
    }

    if (pipCount != null && _pipLayouts.containsKey(pipCount)) {
      final positions = _pipLayouts[pipCount]!;
      return LayoutBuilder(builder: (context, c) {
        final areaW = c.maxWidth;
        final areaH = c.maxHeight;
        return Stack(
          children: positions.map((p) {
            final x = p.x * areaW - pipSize / 2;
            final y = p.y * areaH - pipSize / 2;
            return Positioned(
              left: x.clamp(0.0, areaW - pipSize),
              top: y.clamp(0.0, areaH - pipSize),
              child: p.inverted
                  ? Transform.rotate(
                      angle: math.pi,
                      child: Text(suit,
                          style: TextStyle(
                            color: color,
                            fontSize: pipSize,
                            height: 1.0,
                          )),
                    )
                  : Text(suit,
                      style: TextStyle(
                        color: color,
                        fontSize: pipSize,
                        height: 1.0,
                      )),
            );
          }).toList(),
        );
      });
    }

    // Fallback: centered suit
    return Center(
      child: Text(suit,
          style: TextStyle(
            color: color,
            fontSize: (math.min(width, height) * 0.35).clamp(12.0, 36.0),
            fontWeight: FontWeight.w600,
          )),
    );
  }
}

// ─────────────────────────────────────────────
// CardBack Widget
// ─────────────────────────────────────────────

class CardBack extends StatelessWidget {
  const CardBack({super.key});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final w = c.maxWidth;
      final radius = (w * 0.1).clamp(4.0, 10.0);
      return Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(radius),
          border: Border.all(color: const Color(0xFF0A0D18), width: 1.5),
          boxShadow: const [
            BoxShadow(
              color: Color(0x40000000),
              blurRadius: 6,
              spreadRadius: 0.5,
              offset: Offset(0, 2),
            ),
          ],
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(radius - 1),
          child: CustomPaint(
            painter: _CardBackPainter(),
            size: Size.infinite,
          ),
        ),
      );
    });
  }
}

class _CardBackPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final rect = Rect.fromLTWH(0, 0, size.width, size.height);

    // Background gradient
    final bgPaint = Paint()
      ..shader = const LinearGradient(
        colors: [PokerColors.cardBackStart, PokerColors.cardBackEnd],
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
      ).createShader(rect);
    canvas.drawRect(rect, bgPaint);

    // Diamond lattice pattern
    final linePaint = Paint()
      ..color = PokerColors.primary.withOpacity(0.12)
      ..strokeWidth = 0.8
      ..style = PaintingStyle.stroke;

    final step = size.width * 0.3;
    for (double x = -size.height; x < size.width + size.height; x += step) {
      canvas.drawLine(
          Offset(x, 0), Offset(x + size.height, size.height), linePaint);
      canvas.drawLine(
          Offset(x + size.height, 0), Offset(x, size.height), linePaint);
    }

    // Inner border
    final innerBorder = RRect.fromRectAndRadius(
      Rect.fromLTWH(3, 3, size.width - 6, size.height - 6),
      Radius.circular(size.width * 0.06),
    );
    canvas.drawRRect(
        innerBorder,
        Paint()
          ..color = PokerColors.primary.withOpacity(0.18)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.0);

    // Center diamond emblem
    final cx = size.width / 2;
    final cy = size.height / 2;
    final d = size.width * 0.18;
    final path = Path()
      ..moveTo(cx, cy - d)
      ..lineTo(cx + d * 0.7, cy)
      ..lineTo(cx, cy + d)
      ..lineTo(cx - d * 0.7, cy)
      ..close();
    canvas.drawPath(
        path, Paint()..color = PokerColors.primary.withOpacity(0.22));
    canvas.drawPath(
        path,
        Paint()
          ..color = PokerColors.accent.withOpacity(0.3)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 0.8);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

// ─────────────────────────────────────────────
// FlipCard Animation
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// HeroCardFlipOverlay
// ─────────────────────────────────────────────

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

      final headerHeight = (cw * 0.45).clamp(16.0, 24.0);
      final headerGap = (4.0 * uiSizeMultiplier).clamp(2.0, 6.0);
      final trayContentH =
          ch + (onToggle != null ? headerGap + headerHeight : 0);

      final canvas = layout.canvasBounds;
      final heroPush = layout.ringRadiusY * kHeroSeatExtraFraction;
      final seatPadding = kPlayerRadius + layout.playerOffset + 10.0;
      final heroSeatCenterY = (centerY + layout.ringRadiusY + heroPush)
          .clamp(canvas.top + seatPadding, canvas.bottom - seatPadding);
      final heroSeatTop = heroSeatCenterY - kPlayerRadius * uiSizeMultiplier;

      final cardGapAboveHero = 10.0 * uiSizeMultiplier;
      var y = heroSeatTop - trayContentH - cardGapAboveHero;

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
      final labelAccent =
          showing ? PokerColors.warning : PokerColors.textSecondary;
      final labelBorder = showing
          ? PokerColors.warning.withOpacity(0.5)
          : PokerColors.borderSubtle;

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

      return Stack(children: [
        Positioned(
          left: trayLeft,
          top: trayTop,
          width: trayWidth,
          height: trayHeight,
          child: IgnorePointer(
            child: Container(
              decoration: BoxDecoration(
                color: PokerColors.overlayLight,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: PokerColors.overlaySubtle, width: 1),
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
            cardTheme: cardTheme,
          ),
        ),
        Positioned(
          left: x2,
          top: y,
          width: cw,
          height: ch,
          child: FlipCard(
            faceUp: showFace,
            card: cards.length > 1 ? cards[1] : null,
            cardTheme: cardTheme,
          ),
        ),
        if (onToggle != null)
          Positioned(
            left: x1,
            top: headerTop,
            width: headerWidth,
            height: headerHeight,
            child: ClipRRect(
              borderRadius: BorderRadius.circular(headerHeight / 2),
              child: BackdropFilter(
                filter: ImageFilter.blur(sigmaX: 6, sigmaY: 6),
                child: Material(
                  color: PokerColors.overlaySubtle,
                  child: InkWell(
                    onTap: onToggle,
                    borderRadius: BorderRadius.circular(headerHeight / 2),
                    child: Container(
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(headerHeight / 2),
                        border: Border.all(color: labelBorder),
                      ),
                      padding:
                          EdgeInsets.symmetric(horizontal: headerHeight * 0.55),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            showing ? Icons.visibility : Icons.visibility_off,
                            size: iconSize,
                            color: labelAccent,
                          ),
                          SizedBox(width: headerHeight * 0.25),
                          Text(actionLabel,
                              style: TextStyle(
                                color: labelAccent,
                                fontSize:
                                    (headerHeight * 0.45).clamp(10.0, 14.0),
                                fontWeight: FontWeight.w700,
                                letterSpacing: 0.6,
                              )),
                        ],
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

// ─────────────────────────────────────────────
// Canvas-based card utilities (for opponent cards drawn on painter)
// ─────────────────────────────────────────────

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

Color getSuitColor(String suit, {CardColorTheme? cardTheme}) =>
    suitColor(suit, cardTheme: cardTheme);

void drawCardFace(Canvas canvas, double x, double y, double width,
    double height, pr.Card card,
    {CardColorTheme? cardTheme}) {
  final cardRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(x, y, width, height),
    Radius.circular(width * 0.08),
  );
  // White card surface
  canvas.drawRRect(cardRect, Paint()..color = PokerColors.cardFace);
  // Subtle border
  canvas.drawRRect(
      cardRect,
      Paint()
        ..color = const Color(0xFFCCCCCC)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 0.8);
  // Shadow
  canvas.drawRRect(
    cardRect.shift(const Offset(0, 1)),
    Paint()
      ..color = const Color(0x30000000)
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 3),
  );

  final tint = getSuitColor(card.suit, cardTheme: cardTheme);
  final suitSym = getSuitSymbol(card.suit);

  // Rank + suit text
  final tp = TextPainter(
    text: TextSpan(
      text: '${card.value}\n$suitSym',
      style: TextStyle(
          color: tint,
          fontSize: (width * 0.24).clamp(7.0, 14.0),
          fontWeight: FontWeight.w900,
          height: 1.2),
    ),
    textDirection: TextDirection.ltr,
  )..layout();
  tp.paint(canvas, Offset(x + width * 0.08, y + height * 0.06));
}

void drawCardBack(
    Canvas canvas, double x, double y, double width, double height) {
  final cardRect = RRect.fromRectAndRadius(
    Rect.fromLTWH(x, y, width, height),
    Radius.circular(width * 0.08),
  );

  final bgPaint = Paint()
    ..shader = const LinearGradient(
      colors: [PokerColors.cardBackStart, PokerColors.cardBackEnd],
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
    ).createShader(Rect.fromLTWH(x, y, width, height));
  canvas.drawRRect(cardRect, bgPaint);

  canvas.drawRRect(
      cardRect,
      Paint()
        ..color = const Color(0xFF0A0D18)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1);

  // Small diamond emblem
  final cx = x + width / 2;
  final cy = y + height / 2;
  final d = width * 0.15;
  final path = Path()
    ..moveTo(cx, cy - d)
    ..lineTo(cx + d * 0.7, cy)
    ..lineTo(cx, cy + d)
    ..lineTo(cx - d * 0.7, cy)
    ..close();
  canvas.drawPath(path, Paint()..color = PokerColors.primary.withOpacity(0.25));
}
