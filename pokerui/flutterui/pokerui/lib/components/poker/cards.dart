import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

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

class CardFace extends StatelessWidget {
  const CardFace({super.key, required pr.Card? card}) : _card = card;
  final pr.Card? _card;

  @override
  Widget build(BuildContext context) {
    final value = _card?.value ?? '';
    final suit = _card?.suit ?? '';
    final isRed = suit.toLowerCase() == 'hearts' || suit.toLowerCase() == 'diamonds' || suit == '♥' || suit == '♦';
    final suitSymbol = suitSym(suit);
    return RepaintBoundary(
      child: LayoutBuilder(
        builder: (context, c) {
          final w = c.maxWidth.clamp(20.0, double.infinity);
          final h = c.maxHeight.clamp(28.0, double.infinity);
          final rankFs = (w * 0.30).clamp(10.0, 28.0).toDouble();
          final suitFs = (w * 0.26).clamp(8.0, 24.0).toDouble();
          final centerFs = (math.min(w, h) * 0.35).clamp(12.0, 40.0).toDouble();
          final textColor = isRed ? Colors.red : Colors.black;
          return Container(
            constraints: const BoxConstraints(
              minWidth: 20.0,
              minHeight: 28.0,
            ),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.black, width: 2),
              boxShadow: [
                BoxShadow(color: Colors.black.withOpacity(0.30), blurRadius: 6, spreadRadius: 1),
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
                          Text(value, style: TextStyle(color: textColor, fontSize: rankFs, fontWeight: FontWeight.w900)),
                          Text(suitSymbol, style: TextStyle(color: textColor, fontSize: suitFs, fontWeight: FontWeight.w700)),
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
                            Text(value, style: TextStyle(color: textColor, fontSize: rankFs, fontWeight: FontWeight.w900)),
                            Text(suitSymbol, style: TextStyle(color: textColor, fontSize: suitFs, fontWeight: FontWeight.w700)),
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
                        style: TextStyle(color: textColor, fontSize: centerFs, fontWeight: FontWeight.w600),
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
      ),
    );
  }
}

class FlipCard extends StatelessWidget {
  const FlipCard({super.key, required this.faceUp, required this.card});
  final bool faceUp;
  final pr.Card? card;

  @override
  Widget build(BuildContext context) {
    final id = cardId(card);
    final frontKey = ValueKey('face_$id');
    final backKey = ValueKey('back_$id');
    final front = CardFace(card: card, key: frontKey);
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
            final angle = isUnder ? math.min(rotate.value, math.pi / 2) : rotate.value;
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
  const HeroCardFlipOverlay({super.key, required this.cards, required this.showFace, this.onToggle, this.showHint});
  final List<pr.Card> cards;
  final bool showFace;
  final VoidCallback? onToggle;
  // When provided, overrides the default hint behavior. If null, falls back
  // to showing the hint when !showFace.
  final bool? showHint;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final box = _viewport16by9(size);
      final cw = math.max(math.min(box.width * 0.06, 54.0), 40.0);
      final ch = cw * 1.4;
      final gap = cw * 0.12;
      final centerX = box.left + box.width / 2;
      final centerY = box.top + box.height / 2;
      final tableRadiusY = (box.height * 0.35).clamp(80.0, 150.0);
      
      // Position hero cards directly above the hero player (who is at bottom center)
      // Hero is at angle pi/2 (90 degrees = bottom)
      // For ellipse: x = centerX + radiusX * cos(π/2) = centerX, y = centerY + radiusY * sin(π/2) = centerY + radiusY
      const playerRadius = 30.0;
      const playerOffset = 50.0; // Offset from table edge to player center (matches painter)
      final ringRadiusY = tableRadiusY + playerOffset;
      // Clamp hero seat so cards track the rendered player when the viewport is tight.
      const seatPadding = playerRadius + 60.0;
      final heroY = (centerY + ringRadiusY).clamp(box.top + seatPadding, box.bottom - seatPadding);
      
      // Lightly scale spacing so cards stay tethered to the hero as width grows.
      const minSpacingAbovePlayer = 40.0;
      const maxSpacingAbovePlayer = 60.0;
      final spacingAbovePlayer = (ringRadiusY * 0.18).clamp(minSpacingAbovePlayer, maxSpacingAbovePlayer);
      
      // Calculate primary position: above player with damped spacing
      var y = heroY - playerRadius - spacingAbovePlayer - ch;
      
      // Soft constraint: ensure reasonable gap from community cards if they would be too close
      // Scale the minimum gap proportionally with table radius to maintain relative spacing
      final communityCardHeight = (box.width * 0.05 * 1.4).clamp(32.0 * 1.4, 56.0 * 1.4);
      final communityCardsBottom = centerY + communityCardHeight / 2 - 20.0;
      // Minimum gap scales with table radius to maintain relative spacing
      final minGapFromCommunity = math.max(24.0, tableRadiusY * 0.24);
      final minSafeY = communityCardsBottom + minGapFromCommunity;
      final maxAllowedY = heroY - playerRadius - ch - 6.0; // avoid overlapping the seat
      // Keep cards between the community row and the hero seat; if constraints conflict, favor the seat.
      if (minSafeY > maxAllowedY) {
        y = maxAllowedY;
      } else {
        y = y.clamp(minSafeY, maxAllowedY);
      }
      final x1 = centerX - cw - gap / 2;
      final x2 = centerX + gap / 2;

      final children = <Widget>[
        Positioned(left: x1, top: y, width: cw, height: ch, child: FlipCard(faceUp: showFace, card: cards.isNotEmpty ? cards[0] : null)),
        Positioned(left: x2, top: y, width: cw, height: ch, child: FlipCard(faceUp: showFace, card: cards.length > 1 ? cards[1] : null)),
      ];

      final shouldShowHint = (showHint ?? !showFace) && onToggle != null;
      if (shouldShowHint) {
        // Subtle outline and hint badge to indicate hidden state
        final totalW = (cw * 2) + gap;
        final outlineLeft = centerX - totalW / 2 - 6;
        final outlineTop = y - 6;
        final outlineH = ch + 12;
        final outlineW = totalW + 12;
        children.add(Positioned(
          left: outlineLeft,
          top: outlineTop,
          width: outlineW,
          height: outlineH,
          child: Container(
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.white24, width: 2),
              color: Colors.black.withOpacity(0.15),
            ),
          ),
        ));

        children.add(Positioned(
          left: centerX - 54,
          top: outlineTop - 26,
          child: Opacity(
            opacity: 0.9,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: Colors.black.withOpacity(0.8),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white24),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: const [
                  Icon(Icons.visibility_off, size: 14, color: Colors.white70),
                  SizedBox(width: 6),
                  Text('Tap to show cards', style: TextStyle(color: Colors.white70, fontSize: 12, fontWeight: FontWeight.w600)),
                ],
              ),
            ),
          ),
        ));
      }

      return Stack(children: [
        // Tap target covering the two cards area (active only if onToggle is provided)
        if (onToggle != null)
          Positioned(
            left: x1,
            top: y,
            width: (cw * 2) + gap,
            height: ch,
            child: GestureDetector(
              behavior: HitTestBehavior.translucent,
              onTap: onToggle,
              child: const SizedBox.expand(),
            ),
          ),
        ...children,
      ]);
    });
  }
}

Rect _viewport16by9(Size size) {
  const aspect = 16 / 9;
  final containerAspect = size.width / (size.height == 0 ? 1 : size.height);
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

// Canvas-based card drawing utilities for CustomPainter usage
void drawCardFace(Canvas canvas, double x, double y, double width, double height, pr.Card card) {
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
        color: getSuitColor(card.suit),
        fontSize: 10,
        fontWeight: FontWeight.bold,
      ),
    ),
    textDirection: TextDirection.ltr,
  );
  textPainter.layout();
  textPainter.paint(
    canvas,
    Offset(x + (width - textPainter.width) / 2, y + (height - textPainter.height) / 2),
  );
}

void drawCardBack(Canvas canvas, double x, double y, double width, double height) {
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
      style: TextStyle(color: Colors.white70, fontSize: 12, fontWeight: FontWeight.bold),
    ),
    textDirection: TextDirection.ltr,
  );
  pipPainter.layout();
  pipPainter.paint(
    canvas,
    Offset(x + (width - pipPainter.width) / 2, y + (height - pipPainter.height) / 2),
  );
}

String getSuitSymbol(String suit) {
  switch (suit.toLowerCase()) {
    case 'hearts': return '♥';
    case 'diamonds': return '♦';
    case 'clubs': return '♣';
    case 'spades': return '♠';
    default: return suit;
  }
}

Color getSuitColor(String suit) {
  final s = suit.toLowerCase();
  // Check for Unicode symbols first
  if (suit == '♥' || suit == '\u2665' || suit == '♦' || suit == '\u2666') {
    return Colors.red;
  }
  if (suit == '♣' || suit == '\u2663' || suit == '♠' || suit == '\u2660') {
    return Colors.black;
  }
  // Then check lowercase strings
  switch (s) {
    case 'hearts':
    case 'diamonds':
      return Colors.red;
    case 'clubs':
    case 'spades':
      return Colors.black;
    default:
      return Colors.black;
  }
}
