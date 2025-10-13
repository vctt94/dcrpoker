import 'dart:math' as math;
import 'package:flutter/material.dart';

/// ===========================================================================
/// Const-friendly public models
/// ===========================================================================
class PokerCardModel {
  final String rank; // e.g. 'A', 'K', 'T', '9'
  final String suit; // '♠', '♥', '♦', '♣'
  const PokerCardModel(this.rank, this.suit);
  bool get isRed => suit == '♥' || suit == '♦';
}

class SeatData {
  final String name;
  final int chips;
  final bool isHero;
  final bool isTurn;
  final bool dealer;
  final bool smallBlind;
  final bool bigBlind;
  final List<PokerCardModel> hole;

  const SeatData({
    required this.name,
    required this.chips,
    this.isHero = false,
    this.isTurn = false,
    this.dealer = false,
    this.smallBlind = false,
    this.bigBlind = false,
    this.hole = const [],
  });
}

/// ===========================================================================
/// Poker Table (responsive, seat-safe)
/// ===========================================================================
class PokerTable extends StatelessWidget {
  const PokerTable({
    super.key,
    this.title = 'Poker Table',
    this.seats = const [
      SeatData(
        name: 'Player 1',
        chips: 1500,
        isHero: true,
        isTurn: true,
        dealer: true,
        hole: [PokerCardModel('A', '♠'), PokerCardModel('K', '♥')],
      ),
      SeatData(name: 'Player 2', chips: 2300, smallBlind: true),
      SeatData(name: 'Player 3', chips: 800, bigBlind: true),
      SeatData(name: 'Player 4', chips: 3200),
      SeatData(name: 'Player 5', chips: 1800),
      SeatData(name: 'Player 6', chips: 2100),
      SeatData(name: 'Player 7', chips: 950),
      SeatData(name: 'Player 8', chips: 2750),
      SeatData(name: 'Player 9', chips: 1200),
    ],
    this.board = const [
      PokerCardModel('A', '♠'),
      PokerCardModel('K', '♥'),
      PokerCardModel('Q', '♦'),
    ],
    this.pot = 450,
  });

  final String title;
  final List<SeatData> seats;
  final List<PokerCardModel> board; // up to 5
  final int pot;

  // Palette
  static const _felt = Color(0xFF0F5A45);
  static const _feltShade = Color(0xFF0D4F3C);
  static const _rail = Color(0xFF7A3E10);
  static const _bgTop = Color(0xFF0D4F3C);
  static const _bgBottom = Color(0xFF1A5F4A);

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient:
            LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter, colors: [_bgTop, _bgBottom]),
      ),
      child: LayoutBuilder(
        builder: (context, c) {
          // ---------- Table & ring geometry ----------
          final shortest = math.min(c.maxWidth, c.maxHeight);
          final seatCount = math.max(seats.length, 2);

          // Felt sizing scales with viewport; rail proportional
          final tableW = shortest * 0.62;
          final tableH = tableW * 0.70;
          final rail = (tableW * 0.02).clamp(8.0, 18.0);

          // Room outside felt so seats never overlap felt
          final outerPadX = rail * 2 + 24.0;
          final outerPadY = rail * 2 + 28.0;

          final stackW = tableW + outerPadX * 2;
          final stackH = tableH + outerPadY * 2;

          final tableOffsetX = (stackW - tableW) / 2;
          final tableOffsetY = (stackH - tableH) / 2;

          // Seat ring radii + dynamic seat size
          final seatRx = (tableW / 2) + rail + 28;
          final seatRy = (tableH / 2) + rail + 22;

          // Seat width scales with ring circumference / count, but clamped
          final ringPerimeter = math.pi * (3 * (seatRx + seatRy) - math.sqrt((3 * seatRx + seatRy) * (seatRx + 3 * seatRy)));
          final targetSlot = ringPerimeter / seatCount;
          final seatW = targetSlot * 0.55; // leave breathing room
          final seatH = seatW * 0.62; // compact height
          final seatSize = Size(seatW.clamp(120.0, 180.0), seatH.clamp(76.0, 110.0));

          // Board/card sizing relative to felt
          final boardW = tableW * 0.70;
          final boardH = tableH * 0.45;

          // Where to start placing seats: center hero at bottom (pi/2)
          // and distribute counterclockwise.
          final step = 2 * math.pi / seatCount;

          // ---------- UI ----------
          return Stack(
            children: [
              // Main content
              Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const SizedBox(height: 12),
                    Text(title,
                        style: const TextStyle(color: Colors.white, fontSize: 26, fontWeight: FontWeight.w800)),
                    const SizedBox(height: 36),

                    // ---- Table stack ----
                    SizedBox(
                      width: stackW,
                      height: stackH,
                      child: Stack(
                        clipBehavior: Clip.none,
                        children: [
                          // Rail + Felt
                          Positioned(
                            left: tableOffsetX - rail,
                            top: tableOffsetY - rail,
                            child: _TableOval(
                              width: tableW,
                              height: tableH,
                              rail: rail,
                              felt: _felt,
                              feltShade: _feltShade,
                              railColor: _rail,
                            ),
                          ),

                          // Community area (center)
                          Positioned(
                            left: tableOffsetX,
                            top: tableOffsetY,
                            child: SizedBox(
                              width: tableW,
                              height: tableH,
                              child: Center(
                                child: _CommunityArea(
                                  width: boardW,
                                  height: boardH,
                                  cards: board,
                                ),
                              ),
                            ),
                          ),

                          // Pot (bottom inside felt)
                          Positioned(
                            left: tableOffsetX,
                            top: tableOffsetY,
                            child: SizedBox(
                              width: tableW,
                              height: tableH,
                              child: Align(alignment: const Alignment(0, 0.72), child: _Pot(pot: pot)),
                            ),
                          ),

                          // Seats around ring (outside rail)
                          ...List.generate(seatCount, (i) {
                            final a = math.pi / 2 + i * step; // 90° = bottom
                            final cx = tableOffsetX + tableW / 2 + seatRx * math.cos(a);
                            final cy = tableOffsetY + tableH / 2 + seatRy * math.sin(a);
                            final data = seats[i];

                            return Positioned(
                              left: cx - seatSize.width / 2,
                              top: cy - seatSize.height / 2,
                              child: _PlayerSeat(
                                data: data,
                                size: seatSize,
                              ),
                            );
                          }),
                        ],
                      ),
                    ),

                    const SizedBox(height: 18),
                  ],
                ),
              ),

              // Action bar positioned near the hero player
              Positioned(
                bottom: 30,
                left: 0,
                right: 0,
                child: Center(
                  child: _ActionBar(
                    onFold: () {},
                    onCall: () {},
                    onRaise: () {},
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

/// ===========================================================================
/// Felt + Rail
/// ===========================================================================
class _TableOval extends StatelessWidget {
  const _TableOval({
    required this.width,
    required this.height,
    required this.rail,
    required this.felt,
    required this.feltShade,
    required this.railColor,
  });

  final double width, height, rail;
  final Color felt, feltShade, railColor;

  @override
  Widget build(BuildContext context) {
    final outerW = width + rail * 2, outerH = height + rail * 2;
    return SizedBox(
      width: outerW,
      height: outerH,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: railColor,
          borderRadius: BorderRadius.circular(height),
          boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.35), blurRadius: 24, spreadRadius: 2)],
        ),
        child: Center(
          child: Container(
            width: width,
            height: height,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(height),
              gradient: LinearGradient(
                colors: [felt, feltShade],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// ===========================================================================
/// Board / Community Cards
/// ===========================================================================
class _CommunityArea extends StatelessWidget {
  const _CommunityArea({
    required this.width,
    required this.height,
    required this.cards,
  });

  final double width, height;
  final List<PokerCardModel> cards;

  @override
  Widget build(BuildContext context) {
    final count = cards.length.clamp(0, 5);
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.10),
        borderRadius: BorderRadius.circular(height),
        border: Border.all(color: Colors.white24, width: 1.2),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Text('Community Cards',
              style: TextStyle(color: Colors.white70, fontSize: 14, fontWeight: FontWeight.w700)),
          const SizedBox(height: 10),
          LayoutBuilder(builder: (_, c) {
            final available = c.maxWidth;
            final gaps = (math.max(count, 5) - 1) * 8.0;
            final cardW = ((available - gaps) / 5).clamp(46.0, 64.0);
            return Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: List.generate(5, (i) {
                final card = i < count ? cards[i] : null;
                return Padding(
                  padding: EdgeInsets.only(right: i == 4 ? 0 : 8),
                  child: _PokerCard(model: card, width: cardW),
                );
              }),
            );
          }),
        ],
      ),
    );
  }
}

/// ===========================================================================
/// Pot
/// ===========================================================================
class _Pot extends StatelessWidget {
  const _Pot({required this.pot});
  final int pot;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.78),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: Colors.amber, width: 2),
        boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.35), blurRadius: 8)],
      ),
      child: Text('Pot: $pot',
          style: const TextStyle(color: Colors.amber, fontWeight: FontWeight.w800, fontSize: 18)),
    );
  }
}

/// ===========================================================================
/// Player Seat (badges + soft halo when turn)
/// ===========================================================================
class _PlayerSeat extends StatelessWidget {
  const _PlayerSeat({required this.data, required this.size});
  final SeatData data;
  final Size size;

  @override
  Widget build(BuildContext context) {
    final isHero = data.isHero;
    final heroBase = const Color(0xFF2E6DD8);
    final otherBase = Colors.grey.shade700;
    final base = isHero ? heroBase : otherBase;

    final haloColor = data.isTurn ? Colors.yellowAccent : Colors.transparent;

    final cardW = (isHero ? size.width * 0.35 : size.width * 0.24).clamp(28.0, 64.0);
    final gap = 6.0;

    return Stack(
      clipBehavior: Clip.none,
      children: [
        // soft halo
        Positioned.fill(
          child: IgnorePointer(
            child: DecoratedBox(
              decoration: BoxDecoration(
                boxShadow: [
                  if (data.isTurn)
                    BoxShadow(color: haloColor.withOpacity(0.35), blurRadius: 22, spreadRadius: 2),
                ],
              ),
            ),
          ),
        ),

        // seat card
        Container(
          width: size.width,
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            color: base,
            borderRadius: BorderRadius.circular(18),
            border: Border.all(color: data.isTurn ? Colors.yellowAccent : Colors.white24, width: 1.5),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // name + stack + badges row
              Row(
                children: [
                  Expanded(
                    child: Text(
                      data.name,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w800),
                    ),
                  ),
                  const SizedBox(width: 6),
                  if (data.dealer) _badge('D', Colors.amber),
                  if (data.smallBlind) ...[const SizedBox(width: 4), _badge('SB', Colors.cyan)],
                  if (data.bigBlind) ...[const SizedBox(width: 4), _badge('BB', Colors.pinkAccent)],
                  const SizedBox(width: 8),
                  Text('${data.chips}',
                      style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600)),
                ],
              ),
              const SizedBox(height: 6),

              // two cards always visible (face-down if unknown)
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  _PokerCard(model: data.hole.isNotEmpty ? data.hole[0] : null, width: cardW),
                  SizedBox(width: gap),
                  _PokerCard(model: data.hole.length > 1 ? data.hole[1] : null, width: cardW),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _badge(String t, Color c) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
        decoration: BoxDecoration(color: c.withOpacity(0.85), borderRadius: BorderRadius.circular(6)),
        child: Text(t, style: const TextStyle(color: Colors.black, fontWeight: FontWeight.w900, fontSize: 10)),
      );
}

/// ===========================================================================
/// Action Bar (positioned in bottom right)
/// ===========================================================================
class _ActionBar extends StatelessWidget {
  const _ActionBar({required this.onFold, required this.onCall, required this.onRaise});
  final VoidCallback onFold, onCall, onRaise;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _pill('Fold', Colors.red, onFold),
        const SizedBox(width: 12),
        _pill('Call', Colors.black87, onCall),
        const SizedBox(width: 12),
        _pill('Raise', Colors.green, onRaise),
      ],
    );
  }

  Widget _pill(String t, Color bg, VoidCallback onTap) => ElevatedButton(
        onPressed: onTap,
        style: ElevatedButton.styleFrom(
          backgroundColor: bg,
          foregroundColor: Colors.white,
          padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 12),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
          elevation: 2,
        ),
        child: Text(t, style: const TextStyle(fontWeight: FontWeight.w700)),
      );
}

/// ===========================================================================
/// Card widget (face-down if model == null)
/// ===========================================================================
class _PokerCard extends StatelessWidget {
  const _PokerCard({required this.model, required this.width});
  final PokerCardModel? model;
  final double width;

  @override
  Widget build(BuildContext context) {
    final w = width;
    final h = w * 1.4;
    final faceDown = model == null;
    final isRed = model?.isRed ?? false;

    return Container(
      width: w,
      height: h,
      decoration: BoxDecoration(
        color: faceDown ? Colors.black54 : Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.black, width: 2),
        boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.35), blurRadius: 6, spreadRadius: 1)],
        gradient: faceDown
            ? const LinearGradient(colors: [Colors.black54, Colors.black87], begin: Alignment.topLeft, end: Alignment.bottomRight)
            : null,
      ),
      child: faceDown
          ? Center(child: Icon(Icons.casino, color: Colors.white70, size: w * 0.34))
          : Padding(
              padding: const EdgeInsets.all(4.0),
              child: Stack(
                children: [
                  // top-left rank/suit
                  Align(
                    alignment: Alignment.topLeft,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(model!.rank,
                            style: TextStyle(
                                color: isRed ? Colors.red : Colors.black,
                                fontWeight: FontWeight.w900,
                                fontSize: w * 0.30)),
                        Text(model!.suit,
                            style: TextStyle(
                                color: isRed ? Colors.red : Colors.black,
                                fontWeight: FontWeight.w700,
                                fontSize: w * 0.26)),
                      ],
                    ),
                  ),
                  // bottom-right rank/suit (mirrored)
                  Align(
                    alignment: Alignment.bottomRight,
                    child: Transform.rotate(
                      angle: math.pi,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(model!.rank,
                              style: TextStyle(
                                  color: isRed ? Colors.red : Colors.black,
                                  fontWeight: FontWeight.w900,
                                  fontSize: w * 0.30)),
                          Text(model!.suit,
                              style: TextStyle(
                                  color: isRed ? Colors.red : Colors.black,
                                  fontWeight: FontWeight.w700,
                                  fontSize: w * 0.26)),
                        ],
                      ),
                    ),
                  ),
                  // big center pip
                  Center(
                    child: Text(model!.suit,
                        style: TextStyle(
                          color: isRed ? Colors.red : Colors.black,
                          fontSize: w * 0.60,
                        )),
                  ),
                ],
              ),
            ),
    );
  }
}
