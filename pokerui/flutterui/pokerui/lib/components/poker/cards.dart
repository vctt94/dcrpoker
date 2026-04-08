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

double _centerPipScale(String suit) {
  final sym = suitSym(suit);
  switch (sym) {
    case '♥':
      return 1.0;
    case '♦':
      return 1.35;
    case '♣':
      return 1.20;
    case '♠':
      return 1.25;
    default:
      return 1.0;
  }
}

class _PipPos {
  final double x, y;
  final bool inverted;
  const _PipPos(this.x, this.y, this.inverted);
}

class _CardFaceLayout {
  const _CardFaceLayout({
    required this.topLeftCorner,
    required this.bottomRightCorner,
    required this.pipRect,
  });

  final Rect topLeftCorner;
  final Rect bottomRightCorner;
  final Rect pipRect;
}

const Map<int, List<_PipPos>> _pipLayouts = {
  1: [_PipPos(0.5, 0.5, false)],
  2: [_PipPos(0.5, 0.18, false), _PipPos(0.5, 0.82, true)],
  3: [
    _PipPos(0.5, 0.18, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.5, 0.82, true),
  ],
  4: [
    _PipPos(0.24, 0.18, false),
    _PipPos(0.76, 0.18, false),
    _PipPos(0.24, 0.82, true),
    _PipPos(0.76, 0.82, true),
  ],
  5: [
    _PipPos(0.24, 0.18, false),
    _PipPos(0.76, 0.18, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.24, 0.82, true),
    _PipPos(0.76, 0.82, true),
  ],
  6: [
    _PipPos(0.24, 0.18, false),
    _PipPos(0.76, 0.18, false),
    _PipPos(0.24, 0.5, false),
    _PipPos(0.76, 0.5, false),
    _PipPos(0.24, 0.82, true),
    _PipPos(0.76, 0.82, true),
  ],
  7: [
    _PipPos(0.24, 0.16, false),
    _PipPos(0.76, 0.16, false),
    _PipPos(0.5, 0.30, false),
    _PipPos(0.24, 0.5, false),
    _PipPos(0.76, 0.5, false),
    _PipPos(0.24, 0.84, true),
    _PipPos(0.76, 0.84, true),
  ],
  8: [
    _PipPos(0.24, 0.16, false),
    _PipPos(0.76, 0.16, false),
    _PipPos(0.5, 0.30, false),
    _PipPos(0.24, 0.44, false),
    _PipPos(0.76, 0.44, false),
    _PipPos(0.5, 0.70, true),
    _PipPos(0.24, 0.84, true),
    _PipPos(0.76, 0.84, true),
  ],
  9: [
    _PipPos(0.25, 0.14, false),
    _PipPos(0.75, 0.14, false),
    _PipPos(0.25, 0.34, false),
    _PipPos(0.75, 0.34, false),
    _PipPos(0.5, 0.5, false),
    _PipPos(0.25, 0.66, true),
    _PipPos(0.75, 0.66, true),
    _PipPos(0.25, 0.86, true),
    _PipPos(0.75, 0.86, true),
  ],
  10: [
    _PipPos(0.25, 0.12, false),
    _PipPos(0.75, 0.12, false),
    _PipPos(0.5, 0.26, false),
    _PipPos(0.25, 0.40, false),
    _PipPos(0.75, 0.40, false),
    _PipPos(0.25, 0.60, true),
    _PipPos(0.75, 0.60, true),
    _PipPos(0.5, 0.74, true),
    _PipPos(0.25, 0.88, true),
    _PipPos(0.75, 0.88, true),
  ],
};

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

const double _compactCardWidthThreshold = 46.0;

bool _useSimplifiedCardFace(double width) {
  return width < _compactCardWidthThreshold;
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

_CardFaceLayout _computeCardFaceLayout(
  double width,
  double height,
  Size cornerSize,
) {
  final cornerInsetX = width * 0.06;
  final cornerInsetY = height * 0.04;
  final availableWidth = math.max(0.0, width - (cornerInsetX * 2));
  final availableHeight = math.max(0.0, height - (cornerInsetY * 2));
  final maxCornerWidth = availableWidth * 0.34;
  final maxCornerHeight = availableHeight * 0.24;
  final cornerWidthScale = cornerSize.width <= 0
      ? 1.0
      : math.min(1.0, maxCornerWidth / cornerSize.width);
  final cornerHeightScale = cornerSize.height <= 0
      ? 1.0
      : math.min(1.0, maxCornerHeight / cornerSize.height);
  final cornerScale = math.min(cornerWidthScale, cornerHeightScale);
  final cornerWidth =
      math.min(cornerSize.width * cornerScale, availableWidth).toDouble();
  final cornerHeight =
      math.min(cornerSize.height * cornerScale, availableHeight).toDouble();

  final topLeftCorner =
      Rect.fromLTWH(cornerInsetX, cornerInsetY, cornerWidth, cornerHeight);
  final bottomRightCorner = Rect.fromLTWH(
    math.max(cornerInsetX, width - cornerInsetX - cornerWidth),
    math.max(cornerInsetY, height - cornerInsetY - cornerHeight),
    cornerWidth,
    cornerHeight,
  );

  final contentBounds = Rect.fromLTRB(
    cornerInsetX,
    topLeftCorner.bottom,
    width - cornerInsetX,
    bottomRightCorner.top,
  );
  final pipRect = contentBounds.width > 0 && contentBounds.height > 0
      ? () {
          final padX = contentBounds.width * 0.04;
          final padY = contentBounds.height * 0.04;
          final rect = Rect.fromLTRB(
            contentBounds.left + padX,
            contentBounds.top + padY,
            contentBounds.right - padX,
            contentBounds.bottom - padY,
          );
          return rect.width > 0 && rect.height > 0 ? rect : contentBounds;
        }()
      : Rect.zero;

  return _CardFaceLayout(
    topLeftCorner: topLeftCorner,
    bottomRightCorner: bottomRightCorner,
    pipRect: pipRect,
  );
}

/// Computes the largest square cell size that prevents any two pip bounding
/// boxes from overlapping (Chebyshev / L∞ distance), with a padding factor
/// for visual breathing room.  The result is the side length of the SizedBox
/// each pip glyph will be rendered inside via FittedBox.
double _maxPipCellSize(
    List<_PipPos> positions, double areaWidth, double areaHeight) {
  var minChebyshev = double.infinity;
  var minEdgeClearance = double.infinity;

  for (var i = 0; i < positions.length; i++) {
    final p = positions[i];
    final edgeClearance = math.min(
      math.min(p.x * areaWidth, (1 - p.x) * areaWidth),
      math.min(p.y * areaHeight, (1 - p.y) * areaHeight),
    );
    if (edgeClearance < minEdgeClearance) {
      minEdgeClearance = edgeClearance;
    }

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
  if (minEdgeClearance == double.infinity) {
    minEdgeClearance = math.min(areaWidth, areaHeight) / 2;
  }

  // 90% of the minimum Chebyshev distance → guaranteed no overlap with a
  // 10% visual gap between adjacent cells.
  final fromSpacing = minChebyshev * 0.90;
  // Keep the outermost pips inside the center rect instead of clamping them
  // into the edges after layout.
  final fromEdges = minEdgeClearance * 2 * 0.92;
  // Cap so pips stay proportional on low-count cards.
  final baseCap = math.min(areaWidth * 0.42, areaHeight * 0.28);

  return math.min(math.min(fromSpacing, fromEdges), baseCap);
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

double _simplifiedRankFontSize(double cardWidth, {required bool compact}) {
  return math.max(12.0, cardWidth * (compact ? 0.34 : 0.40));
}

double _simplifiedSuitFontSize(double cardWidth, {required bool compact}) {
  return math.max(10.0, cardWidth * (compact ? 0.24 : 0.30));
}

double _simplifiedSoloSuitFontSize(double cardWidth, {required bool compact}) {
  return math.max(14.0, cardWidth * (compact ? 0.40 : 0.50));
}

bool _shouldShowCenterContent(
  String value,
  double width, {
  required bool simplifiedMode,
}) {
  if (!simplifiedMode) {
    return width >= 28;
  }
  return _rankToCount(value) == null ? width >= 32 : width >= 30;
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
    return RepaintBoundary(
      child: LayoutBuilder(
        builder: (context, c) {
          final w = c.maxWidth.clamp(20.0, double.infinity);
          final h = c.maxHeight.clamp(28.0, double.infinity);
          return SizedBox(
            width: w,
            height: h,
            child: CustomPaint(
              painter: _CardFacePainter(
                card: _card,
                cardTheme: cardTheme,
              ),
            ),
          );
        },
      ),
    );
  }
}

class _CardFacePainter extends CustomPainter {
  const _CardFacePainter({
    required this.card,
    required this.cardTheme,
  });

  final pr.Card? card;
  final CardColorTheme? cardTheme;

  @override
  void paint(Canvas canvas, Size size) {
    _paintCardFace(
      canvas,
      Rect.fromLTWH(0, 0, size.width, size.height),
      card,
      cardTheme: cardTheme,
    );
  }

  @override
  bool shouldRepaint(covariant _CardFacePainter oldDelegate) {
    return oldDelegate.card != card || oldDelegate.cardTheme != cardTheme;
  }
}

TextPainter _layoutText(String text, TextStyle style) {
  return TextPainter(
    text: TextSpan(text: text, style: style),
    textDirection: TextDirection.ltr,
  )..layout();
}

void _paintCornerIndex(
  Canvas canvas,
  Rect rect, {
  required String rank,
  required String suit,
  required Color color,
  required double rankSize,
  required double suitSize,
  bool rotated = false,
}) {
  final isWideRank = rank.length > 1;
  final rankPainter = _layoutText(
    rank,
    TextStyle(
      color: color,
      fontSize: isWideRank ? rankSize * 0.86 : rankSize,
      fontWeight: FontWeight.w900,
      height: 1.0,
      letterSpacing: isWideRank ? -0.6 : 0.0,
    ),
  );
  final suitPainter = _layoutText(
    suit,
    TextStyle(
      color: color,
      fontSize: suitSize,
      fontWeight: FontWeight.w700,
      height: 1.0,
    ),
  );
  final gap = suitSize * 0.02;
  final contentWidth = math.max(rankPainter.width, suitPainter.width);
  final contentHeight = rankPainter.height + gap + suitPainter.height;
  final scale = math.min(
    1.0,
    math.min(
      rect.width / math.max(contentWidth, 1.0),
      rect.height / math.max(contentHeight, 1.0),
    ),
  );

  void paintColumn(Rect targetRect) {
    canvas.save();
    canvas.translate(
      targetRect.left + targetRect.width / 2,
      targetRect.top + targetRect.height / 2,
    );
    canvas.scale(scale, scale);
    rankPainter.paint(
      canvas,
      Offset(-rankPainter.width / 2, -contentHeight / 2),
    );
    suitPainter.paint(
      canvas,
      Offset(
        -suitPainter.width / 2,
        -contentHeight / 2 + rankPainter.height + gap,
      ),
    );
    canvas.restore();
  }

  if (!rotated) {
    paintColumn(rect);
    return;
  }

  canvas.save();
  canvas.translate(rect.right, rect.bottom);
  canvas.rotate(math.pi);
  paintColumn(Rect.fromLTWH(0, 0, rect.width, rect.height));
  canvas.restore();
}

void _paintCenteredGlyph(
  Canvas canvas,
  Rect rect, {
  required String text,
  required Color color,
  required double fontSize,
  required FontWeight fontWeight,
}) {
  final painter = _layoutText(
    text,
    TextStyle(
      color: color,
      fontSize: fontSize,
      fontWeight: fontWeight,
      height: 1.0,
    ),
  );
  painter.paint(
    canvas,
    Offset(
      rect.left + (rect.width - painter.width) / 2,
      rect.top + (rect.height - painter.height) / 2,
    ),
  );
}

void _paintCenteredRankSuit(
  Canvas canvas,
  Rect rect, {
  required String rank,
  required String suit,
  required Color color,
  required double rankFontSize,
  required double suitFontSize,
  required bool compact,
}) {
  final rankPainter = _layoutText(
    rank,
    TextStyle(
      color: color,
      fontSize: rankFontSize,
      fontWeight: FontWeight.w800,
      height: 1.0,
    ),
  );
  final suitPainter = _layoutText(
    suit,
    TextStyle(
      color: color,
      fontSize: suitFontSize,
      fontWeight: FontWeight.w600,
      height: 1.0,
    ),
  );
  final gap = rect.height * (compact ? 0.01 : 0.015);
  final contentWidth = math.max(rankPainter.width, suitPainter.width);
  final contentHeight = rankPainter.height + gap + suitPainter.height;
  final scale = math.min(
    1.0,
    math.min(
      rect.width / math.max(contentWidth, 1.0),
      rect.height / math.max(contentHeight, 1.0),
    ),
  );

  canvas.save();
  canvas.translate(rect.center.dx, rect.center.dy);
  canvas.scale(scale, scale);
  rankPainter.paint(
    canvas,
    Offset(-rankPainter.width / 2, -contentHeight / 2),
  );
  suitPainter.paint(
    canvas,
    Offset(
      -suitPainter.width / 2,
      -contentHeight / 2 + rankPainter.height + gap,
    ),
  );
  canvas.restore();
}

void _paintPipLayout(
  Canvas canvas,
  Rect rect, {
  required int pipCount,
  required String suit,
  required Color color,
}) {
  final positions = _pipLayouts[pipCount] ?? const <_PipPos>[];
  if (positions.isEmpty || rect.width <= 0 || rect.height <= 0) {
    return;
  }
  final cellSize = _maxPipCellSize(positions, rect.width, rect.height);
  if (cellSize <= 0) {
    return;
  }
  final maxLeft = math.max(0.0, rect.width - cellSize);
  final maxTop = math.max(0.0, rect.height - cellSize);
  final pipScale = _centerPipScale(suit);
  final pipFs = math.min(cellSize * 0.92 * pipScale, cellSize * 1.2);
  final painter = _layoutText(
    suit,
    TextStyle(
      color: color,
      fontSize: pipFs,
      height: 1.0,
    ),
  );

  for (final p in positions) {
    final left =
        rect.left + (p.x * rect.width - cellSize / 2).clamp(0.0, maxLeft);
    final top =
        rect.top + (p.y * rect.height - cellSize / 2).clamp(0.0, maxTop);
    if (p.inverted) {
      canvas.save();
      canvas.translate(left + cellSize / 2, top + cellSize / 2);
      canvas.rotate(math.pi);
      painter.paint(
        canvas,
        Offset(-painter.width / 2, -painter.height / 2),
      );
      canvas.restore();
    } else {
      painter.paint(
        canvas,
        Offset(
          left + (cellSize - painter.width) / 2,
          top + (cellSize - painter.height) / 2,
        ),
      );
    }
  }
}

void _paintCenterContent(
  Canvas canvas,
  Rect rect, {
  required String value,
  required String suit,
  required Color color,
  required double cardWidth,
  required bool simplifiedMode,
}) {
  final pipCount = _rankToCount(value);

  if (value.toUpperCase() == 'A') {
    final compact = simplifiedMode && cardWidth < 52;
    _paintCenteredGlyph(
      canvas,
      rect,
      text: suit,
      color: color,
      fontSize: _simplifiedSoloSuitFontSize(cardWidth, compact: compact),
      fontWeight: FontWeight.w600,
    );
    return;
  }

  if (_isFaceCard(value)) {
    final compact = simplifiedMode && cardWidth < 52;
    _paintCenteredRankSuit(
      canvas,
      rect,
      rank: value,
      suit: suit,
      color: color,
      rankFontSize: _simplifiedRankFontSize(cardWidth, compact: compact),
      suitFontSize: _simplifiedSuitFontSize(cardWidth, compact: compact),
      compact: compact,
    );
    return;
  }

  if (pipCount != null) {
    if (simplifiedMode) {
      final compact = cardWidth < 52;
      _paintCenteredRankSuit(
        canvas,
        rect,
        rank: value,
        suit: suit,
        color: color,
        rankFontSize: _simplifiedRankFontSize(cardWidth, compact: compact),
        suitFontSize: _simplifiedSuitFontSize(cardWidth, compact: compact),
        compact: compact,
      );
      return;
    }

    _paintPipLayout(
      canvas,
      rect,
      pipCount: pipCount,
      suit: suit,
      color: color,
    );
    return;
  }

  _paintCenteredGlyph(
    canvas,
    rect,
    text: suit,
    color: color,
    fontSize: _simplifiedSoloSuitFontSize(cardWidth, compact: false),
    fontWeight: FontWeight.w600,
  );
}

void _paintCardFace(
  Canvas canvas,
  Rect rect,
  pr.Card? card, {
  CardColorTheme? cardTheme,
}) {
  final width = rect.width;
  final height = rect.height;
  final radius = (width * 0.1).clamp(4.0, 10.0);
  final cardRect = RRect.fromRectAndRadius(rect, Radius.circular(radius));

  canvas.drawRRect(
    cardRect.shift(const Offset(0, 2)),
    Paint()
      ..color = const Color(0x40000000)
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 6),
  );
  canvas.drawRRect(cardRect, Paint()..color = PokerColors.cardFace);
  canvas.drawRRect(
    cardRect,
    Paint()
      ..color = const Color(0xFFD0D0D0)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1,
  );

  final value = card?.value ?? '';
  final suit = card?.suit ?? '';
  final suitSymbol = suitSym(suit);
  final tint = suitColor(suit, cardTheme: cardTheme);
  final simplifiedMode = _useSimplifiedCardFace(width);
  final showCenterContent = _shouldShowCenterContent(
    value,
    width,
    simplifiedMode: simplifiedMode,
  );
  final cornerScale =
      _cornerIndexScale(width, value, simplifiedMode: simplifiedMode);
  final rankFs = math.max(10.0, width * cornerScale);
  final suitFs = math.max(10.0, width * cornerScale);
  final isRedSuit = suitSymbol == '♥' || suitSymbol == '♦';
  final cornerSuitSize = suitFs * (isRedSuit ? 1.2 : 1.12);
  final cornerSize = _measureCornerIndex(
    value,
    suitSymbol,
    tint,
    rankFs,
    cornerSuitSize,
  );
  final layout = _computeCardFaceLayout(width, height, cornerSize);

  _paintCornerIndex(
    canvas,
    Rect.fromLTWH(
      rect.left + layout.topLeftCorner.left,
      rect.top + layout.topLeftCorner.top,
      layout.topLeftCorner.width,
      layout.topLeftCorner.height,
    ),
    rank: value,
    suit: suitSymbol,
    color: tint,
    rankSize: rankFs,
    suitSize: cornerSuitSize,
  );
  _paintCornerIndex(
    canvas,
    Rect.fromLTWH(
      rect.left + layout.bottomRightCorner.left,
      rect.top + layout.bottomRightCorner.top,
      layout.bottomRightCorner.width,
      layout.bottomRightCorner.height,
    ),
    rank: value,
    suit: suitSymbol,
    color: tint,
    rankSize: rankFs,
    suitSize: cornerSuitSize,
    rotated: true,
  );

  if (showCenterContent && !layout.pipRect.isEmpty) {
    final pipRect = Rect.fromLTWH(
      rect.left + layout.pipRect.left,
      rect.top + layout.pipRect.top,
      layout.pipRect.width,
      layout.pipRect.height,
    );
    canvas.save();
    canvas.clipRect(pipRect);
    _paintCenterContent(
      canvas,
      pipRect,
      value: value,
      suit: suitSymbol,
      color: tint,
      cardWidth: width,
      simplifiedMode: simplifiedMode,
    );
    canvas.restore();
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
  _paintCardFace(
    canvas,
    Rect.fromLTWH(x, y, width, height),
    card,
    cardTheme: cardTheme,
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
