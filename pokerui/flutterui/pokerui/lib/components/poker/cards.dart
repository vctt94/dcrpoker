import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/table_theme.dart';
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
    _PipPos(0.24, 0.2, false),
    _PipPos(0.76, 0.2, false),
    _PipPos(0.24, 0.8, true),
    _PipPos(0.76, 0.8, true),
  ],
  5: [
    _PipPos(0.24, 0.2, false),
    _PipPos(0.76, 0.2, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.24, 0.8, true),
    _PipPos(0.76, 0.8, true),
  ],
  6: [
    _PipPos(0.24, 0.2, false),
    _PipPos(0.76, 0.2, false),
    _PipPos(0.24, 0.5, false),
    _PipPos(0.76, 0.5, false),
    _PipPos(0.24, 0.8, true),
    _PipPos(0.76, 0.8, true),
  ],
  7: [
    _PipPos(0.24, 0.18, false),
    _PipPos(0.76, 0.18, false),
    _PipPos(0.5, 0.33, false),
    _PipPos(0.24, 0.5, false),
    _PipPos(0.76, 0.5, false),
    _PipPos(0.24, 0.82, true),
    _PipPos(0.76, 0.82, true),
  ],
  8: [
    _PipPos(0.24, 0.18, false),
    _PipPos(0.76, 0.18, false),
    _PipPos(0.5, 0.33, false),
    _PipPos(0.24, 0.47, false),
    _PipPos(0.76, 0.47, false),
    _PipPos(0.5, 0.67, true),
    _PipPos(0.24, 0.82, true),
    _PipPos(0.76, 0.82, true),
  ],
  9: [
    _PipPos(0.25, 0.16, false),
    _PipPos(0.75, 0.16, false),
    _PipPos(0.25, 0.35, false),
    _PipPos(0.75, 0.35, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.25, 0.65, true),
    _PipPos(0.75, 0.65, true),
    _PipPos(0.25, 0.84, true),
    _PipPos(0.75, 0.84, true),
  ],
  10: [
    _PipPos(0.25, 0.14, false),
    _PipPos(0.75, 0.14, false),
    _PipPos(0.5, 0.29, false),
    _PipPos(0.25, 0.40, false),
    _PipPos(0.75, 0.40, false),
    _PipPos(0.25, 0.60, true),
    _PipPos(0.75, 0.60, true),
    _PipPos(0.5, 0.71, true),
    _PipPos(0.25, 0.86, true),
    _PipPos(0.75, 0.86, true),
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

bool _isPhoneViewport(BuildContext context) {
  return MediaQuery.sizeOf(context).shortestSide < 600;
}

EdgeInsets _centerContentPadding(
  String value,
  double width,
  double height, {
  required bool simplifiedMode,
}) {
  final pipCount = _rankToCount(value);
  if (pipCount == null) {
    final compactFaceCard = simplifiedMode && width < 52;
    return EdgeInsets.symmetric(
      horizontal: width * (compactFaceCard ? 0.24 : 0.2),
      vertical: height * (compactFaceCard ? 0.19 : 0.16),
    );
  }

  if (!simplifiedMode) {
    final dense = pipCount >= 8;
    final medium = pipCount >= 5;
    return EdgeInsets.symmetric(
      horizontal: width * (dense ? 0.12 : (medium ? 0.14 : 0.16)),
      vertical: height * (dense ? 0.1 : (medium ? 0.12 : 0.14)),
    );
  }

  final horizontalFactor = switch (pipCount) {
    <= 3 => 0.21,
    4 => 0.2,
    5 || 6 => 0.19,
    7 || 8 => 0.18,
    _ => 0.18,
  };
  final verticalFactor = switch (pipCount) {
    <= 3 => 0.2,
    4 => 0.19,
    5 || 6 => 0.18,
    7 || 8 => 0.18,
    _ => 0.19,
  };

  return EdgeInsets.symmetric(
    horizontal: width * horizontalFactor,
    vertical: height * verticalFactor,
  );
}

Size _measureCornerIndex(
  String rank,
  String suit,
  Color color,
  double rankSize,
  double suitSize,
) {
  final isWideRank = rank.length > 1;
  final rankPainter = TextPainter(
    text: TextSpan(
      text: rank,
      style: TextStyle(
        color: color,
        fontSize: isWideRank ? rankSize * 0.86 : rankSize,
        fontWeight: FontWeight.w900,
        height: 1.0,
        letterSpacing: isWideRank ? -0.6 : 0.0,
      ),
    ),
    textDirection: TextDirection.ltr,
  )..layout();

  final suitPainter = TextPainter(
    text: TextSpan(
      text: suit,
      style: TextStyle(
        color: color,
        fontSize: suitSize,
        fontWeight: FontWeight.w700,
        height: 1.0,
      ),
    ),
    textDirection: TextDirection.ltr,
  )..layout();

  final gap = suitSize * 0.02;
  return Size(
    math.max(rankPainter.width, suitPainter.width),
    rankPainter.height + gap + suitPainter.height,
  );
}

EdgeInsets _protectedCenterPadding(
  String value,
  double width,
  double height,
  Size cornerSize, {
  required bool simplifiedMode,
}) {
  final base = _centerContentPadding(
    value,
    width,
    height,
    simplifiedMode: simplifiedMode,
  );
  if (!simplifiedMode) {
    return base;
  }
  final cornerInsetX = width * 0.06;
  final cornerInsetY = height * 0.04;
  final bufferX = width * 0.05;
  final bufferY = height * 0.05;

  return EdgeInsets.fromLTRB(
    math.max(base.left, cornerInsetX + cornerSize.width + bufferX),
    math.max(base.top, cornerInsetY + cornerSize.height + bufferY),
    math.max(base.right, cornerInsetX + cornerSize.width + bufferX),
    math.max(base.bottom, cornerInsetY + cornerSize.height + bufferY),
  );
}

/// Computes the largest square cell size that prevents any two pip bounding
/// boxes from overlapping (Chebyshev / L∞ distance), with a padding factor
/// for visual breathing room.  The result is the side length of the SizedBox
/// each pip glyph will be rendered inside via FittedBox.
double _maxPipCellSize(
    List<_PipPos> positions, double areaWidth, double areaHeight) {
  var minChebyshev = double.infinity;

  for (var i = 0; i < positions.length; i++) {
    for (var j = i + 1; j < positions.length; j++) {
      final dx = (positions[i].x - positions[j].x).abs() * areaWidth;
      final dy = (positions[i].y - positions[j].y).abs() * areaHeight;
      final dist = math.max(dx, dy);
      if (dist > 0 && dist < minChebyshev) {
        minChebyshev = dist;
      }
    }
  }

  if (minChebyshev == double.infinity) {
    minChebyshev = math.min(areaWidth, areaHeight);
  }

  // 90% of the minimum Chebyshev distance → guaranteed no overlap with a
  // 10% visual gap between adjacent cells.
  final fromSpacing = minChebyshev * 0.90;
  // Cap so pips stay proportional on low-count cards.
  final baseCap = math.min(areaWidth * 0.32, areaHeight * 0.22);

  return math.min(fromSpacing, baseCap).clamp(4.0, 24.0);
}

double _cornerIndexScale(double width, String value,
    {required bool simplifiedMode}) {
  if (!simplifiedMode) return 1.0;
  final isLetterRank = _rankToCount(value) == null;
  if (width < 34) return isLetterRank ? 0.78 : 0.84;
  if (width < 42) return isLetterRank ? 0.84 : 0.9;
  if (width < 52) return isLetterRank ? 0.9 : 0.95;
  return 1.0;
}

bool _shouldShowCenterContent(
  String value,
  double width, {
  required bool simplifiedMode,
}) {
  if (!simplifiedMode) {
    return width >= 28;
  }
  final pipCount = _rankToCount(value);
  if (pipCount == null) {
    return width >= 44;
  }
  return width >= 36;
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
          final simplifiedMode = _isPhoneViewport(context);
          final showCenterContent = _shouldShowCenterContent(
            value,
            w,
            simplifiedMode: simplifiedMode,
          );
          final cornerScale =
              _cornerIndexScale(w, value, simplifiedMode: simplifiedMode);
          final rankFs = (w * 0.32 * cornerScale).clamp(7.0, 24.0).toDouble();
          final suitFs = (w * 0.27 * cornerScale).clamp(6.0, 20.0).toDouble();
          final isRedSuit = suitSymbol == '♥' || suitSymbol == '♦';
          final cornerSuitSize = suitFs * (isRedSuit ? 1.2 : 1.12);
          final cornerSize = _measureCornerIndex(
            value,
            suitSymbol,
            tint,
            rankFs,
            cornerSuitSize,
          );
          final contentPadding = _protectedCenterPadding(
            value,
            w,
            h,
            cornerSize,
            simplifiedMode: simplifiedMode,
          );

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
                      suitSize: cornerSuitSize,
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
                        suitSize: cornerSuitSize,
                      ),
                    ),
                  ),
                  // Center content
                  if (showCenterContent)
                    Positioned.fill(
                      child: Padding(
                        padding: contentPadding,
                        child: _CardCenter(
                          value: value,
                          suit: suitSymbol,
                          color: tint,
                          width: w,
                          height: h,
                          simplifiedMode: simplifiedMode,
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
    final isWideRank = rank.length > 1;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Text(rank,
            style: TextStyle(
              color: color,
              fontSize: isWideRank ? rankSize * 0.86 : rankSize,
              fontWeight: FontWeight.w900,
              height: 1.0,
              letterSpacing: isWideRank ? -0.6 : 0.0,
            )),
        SizedBox(height: suitSize * 0.02),
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
    required this.simplifiedMode,
  });
  final String value, suit;
  final Color color;
  final double width, height;
  final bool simplifiedMode;

  @override
  Widget build(BuildContext context) {
    final pipCount = _rankToCount(value);

    if (value.toUpperCase() == 'A') {
      final compactAce = simplifiedMode && width < 52;
      return Center(
        child: Text(suit,
            style: TextStyle(
              color: color,
              fontSize: (math.min(width, height) * (compactAce ? 0.32 : 0.4))
                  .clamp(10.0, 44.0),
              fontWeight: FontWeight.w600,
            )),
      );
    }

    if (_isFaceCard(value)) {
      final compactFace = simplifiedMode && width < 52;
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(value,
                style: TextStyle(
                  color: color,
                  fontSize:
                      (math.min(width, height) * (compactFace ? 0.24 : 0.3))
                          .clamp(10.0, 32.0),
                  fontWeight: FontWeight.w800,
                )),
            SizedBox(height: height * (compactFace ? 0.01 : 0.015)),
            Text(suit,
                style: TextStyle(
                  color: color,
                  fontSize:
                      (math.min(width, height) * (compactFace ? 0.14 : 0.18))
                          .clamp(7.0, 22.0),
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
        if (areaW <= 0 || areaH <= 0) {
          return const SizedBox.shrink();
        }
        final cellSize = _maxPipCellSize(positions, areaW, areaH);
        if (cellSize <= 0) {
          return const SizedBox.shrink();
        }
        final maxLeft = math.max(0.0, areaW - cellSize);
        final maxTop = math.max(0.0, areaH - cellSize);
        // Font size at 80% of cell leaves margin for glyph overpaint.
        final pipFs = cellSize * 0.80;

        return Stack(
          clipBehavior: Clip.hardEdge,
          children: positions.map((p) {
            final x = p.x * areaW - cellSize / 2;
            final y = p.y * areaH - cellSize / 2;
            Widget pip = ClipRect(
              child: SizedBox(
                width: cellSize,
                height: cellSize,
                child: Center(
                  child: Text(suit,
                      style: TextStyle(
                        color: color,
                        fontSize: pipFs,
                        height: 1.0,
                      )),
                ),
              ),
            );
            if (p.inverted) {
              pip = Transform.rotate(angle: math.pi, child: pip);
            }
            return Positioned(
              left: x.clamp(0.0, maxLeft),
              top: y.clamp(0.0, maxTop),
              child: pip,
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
  final isWideRank = card.value.length > 1;
  final rankSize =
      (width * (isWideRank ? 0.2 : 0.24)).clamp(7.0, 14.0).toDouble();
  final suitSize = (rankSize * 0.84).clamp(6.0, 12.0).toDouble();
  final left = x + width * 0.08;
  final top = y + height * 0.06;

  final rankPainter = TextPainter(
    text: TextSpan(
      text: card.value,
      style: TextStyle(
        color: tint,
        fontSize: rankSize,
        fontWeight: FontWeight.w900,
        height: 1.0,
        letterSpacing: isWideRank ? -0.5 : 0.0,
      ),
    ),
    textDirection: TextDirection.ltr,
  )..layout();
  rankPainter.paint(canvas, Offset(left, top));

  final suitPainter = TextPainter(
    text: TextSpan(
      text: suitSym,
      style: TextStyle(
        color: tint,
        fontSize: suitSize,
        fontWeight: FontWeight.w700,
        height: 1.0,
      ),
    ),
    textDirection: TextDirection.ltr,
  )..layout();
  suitPainter.paint(
    canvas,
    Offset(left + (rankPainter.width - suitPainter.width) / 2,
        top + rankPainter.height - height * 0.01),
  );
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
