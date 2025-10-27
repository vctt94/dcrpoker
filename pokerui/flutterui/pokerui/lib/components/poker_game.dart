import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/helper.dart';

class PokerTableBackground extends StatelessWidget {
  const PokerTableBackground({super.key, this.frac = 0.70});
  final double frac;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: LayoutBuilder(
        builder: (context, constraints) {
          final shortest = constraints.biggest.shortestSide;
          final size = (shortest.isFinite && shortest > 0)
              ? shortest * frac
              : 300.0;

          return Center(
            child: Container(
              width: size,
              height: size,
              decoration: BoxDecoration(
                color: const Color(0xFF0D4F3C), // Poker table green
                borderRadius: BorderRadius.circular(size / 2),
                border: Border.all(
                  color: const Color(0xFF8B4513), // Brown border
                  width: 8,
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withOpacity(0.3),
                    spreadRadius: 5,
                    blurRadius: 15,
                  ),
                ],
              ),
              child: Center(
                child: Icon(
                  Icons.casino,
                  size: size * 0.3,
                  color: Colors.white.withOpacity(0.1),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

class PokerGame {
  final PokerModel pokerModel;
  final String playerId;
  final RenderLoop _loop = RenderLoop();

  PokerGame(this.playerId, this.pokerModel);

  Widget buildWidget(UiGameState gameState, FocusNode focusNode, {VoidCallback? onReadyHotkey}) {
    // Start/stop lightweight repaint loop only while an authoritative deadline is active.
    if (gameState.turnDeadlineUnixMs > 0) {
      _loop.start();
    } else {
      _loop.stop();
    }
    return GestureDetector(
      onTap: () => focusNode.requestFocus(),
      child: Focus(
        child: KeyboardListener(
          focusNode: focusNode..requestFocus(),
          onKeyEvent: (KeyEvent event) {
            if (event is KeyDownEvent || event is KeyRepeatEvent) {
              String keyLabel = event.logicalKey.keyLabel;
              if (onReadyHotkey != null) {
                if (event.logicalKey == LogicalKeyboardKey.space || keyLabel == 'r' || keyLabel == 'R') {
                  onReadyHotkey();
                  return;
                }
              }
              handleInput(playerId, keyLabel);
            }
          },
          child: LayoutBuilder(
            builder: (context, constraints) {
              return Center(
                child: SizedBox(
                  width: constraints.maxWidth,
                  child: AspectRatio(
                    aspectRatio: 16 / 9, // Poker table aspect ratio
                    child: RepaintBoundary(
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          // Poker table background
                          const PokerTableBackground(),

                          // Game canvas (repaints)
                          CustomPaint(
                            painter: PokerPainter(gameState, playerId, repaint: _loop),
                            isComplex: true,
                            willChange: true,
                          ),

                          // Pot and betting info overlay
                          IgnorePointer(
                            child: Stack(
                              fit: StackFit.expand,
                              children: [
                                // Pot display
                                Positioned(
                                  top: 20,
                                  left: 0,
                                  right: 0,
                                  child: Center(
                                    child: Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                                      decoration: BoxDecoration(
                                        color: Colors.black.withOpacity(0.7),
                                        borderRadius: BorderRadius.circular(20),
                                        border: Border.all(color: Colors.amber, width: 2),
                                      ),
                                      child: Text(
                                        'Pot: ${gameState.pot}',
                                        style: const TextStyle(
                                          color: Colors.amber,
                                          fontSize: 20,
                                          fontWeight: FontWeight.bold,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                                // Current bet display
                                if (gameState.currentBet > 0)
                                  Positioned(
                                    top: 60,
                                    left: 0,
                                    right: 0,
                                    child: Center(
                                      child: Container(
                                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                        decoration: BoxDecoration(
                                          color: Colors.red.withOpacity(0.8),
                                          borderRadius: BorderRadius.circular(15),
                                        ),
                                        child: Text(
                                          'Current Bet: ${gameState.currentBet}',
                                          style: const TextStyle(
                                            color: Colors.white,
                                            fontSize: 16,
                                            fontWeight: FontWeight.bold,
                                          ),
                                        ),
                                      ),
                                  ),
                                ),
                              ],
                            ),
                          ),

                          // Clickable hotspots over hero hole cards to toggle show/hide during showdown.
                          if (gameState.phase == pr.GamePhase.SHOWDOWN)
                            LayoutBuilder(builder: (context, c) {
                              final size = c.biggest;
                              // Match painter sizing/placement
                              final cw = math.min(size.width * 0.06, 54.0);
                              final ch = cw * 1.4;
                              final gap = cw * 0.12;
                              final centerX = size.width / 2;
                              const marginBottom = 80.0;
                              final y = size.height - ch - marginBottom;
                              final x1 = centerX - cw - gap / 2;
                              final x2 = centerX + gap / 2;

                              void toggleShowHide() {
                                if (pokerModel.myCardsShown) {
                                  pokerModel.hideCards();
                                } else {
                                  pokerModel.showCards();
                                }
                              }

                              return Stack(children: [
                                Positioned(
                                  left: x1,
                                  top: y,
                                  width: cw,
                                  height: ch,
                                  child: GestureDetector(
                                    behavior: HitTestBehavior.opaque,
                                    onTap: toggleShowHide,
                                    child: const SizedBox.expand(),
                                  ),
                                ),
                                Positioned(
                                  left: x2,
                                  top: y,
                                  width: cw,
                                  height: ch,
                                  child: GestureDetector(
                                    behavior: HitTestBehavior.opaque,
                                    onTap: toggleShowHide,
                                    child: const SizedBox.expand(),
                                  ),
                                ),
                                // Small eye badge to indicate current show/hide state
                                Positioned(
                                  left: centerX - 12,
                                  top: y - 22,
                                  child: Opacity(
                                    opacity: 0.85,
                                    child: Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                      decoration: BoxDecoration(
                                        color: Colors.black.withOpacity(0.75),
                                        borderRadius: BorderRadius.circular(10),
                                        border: Border.all(color: Colors.white24),
                                      ),
                                      child: Row(
                                        mainAxisSize: MainAxisSize.min,
                                        children: [
                                          Icon(
                                            pokerModel.myCardsShown ? Icons.visibility : Icons.visibility_off,
                                            size: 14,
                                            color: Colors.white70,
                                          ),
                                          const SizedBox(width: 4),
                                          Text(
                                            pokerModel.myCardsShown ? 'Shown' : 'Hidden',
                                            style: const TextStyle(color: Colors.white70, fontSize: 10, fontWeight: FontWeight.w600),
                                          ),
                                        ],
                                      ),
                                    ),
                                  ),
                                ),
                              ]);
                            }),
                        ],
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }

  // Build an overlay widget for the ready-to-play UI and countdown
  Widget buildReadyToPlayOverlay(
      BuildContext context,
      bool isReadyToPlay,
      bool countdownStarted,
      String countdownMessage,
      Function onReadyPressed,
      UiGameState gameState) {
    // If countdown has started, show the countdown message in the center
    if (countdownStarted) {
      return Center(
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: const Color(0xFF1B1E2C).withAlpha(230),
            borderRadius: BorderRadius.circular(15),
            boxShadow: [
              BoxShadow(
                color: Colors.blueAccent.withAlpha(76),
                spreadRadius: 3,
                blurRadius: 10,
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(
                Icons.casino,
                size: 50,
                color: Colors.blueAccent,
              ),
              const SizedBox(height: 20),
              Text(
                countdownMessage,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 40,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ],
          ),
        ),
      );
    }

    // If not ready to play, show the ready button with game controls info
    if (!isReadyToPlay) {
      return Container(
        color: Color.fromRGBO(0, 0, 0, 0.65),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Poker table visualization
              SizedBox(
                height: 80,
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Container(
                      width: 40,
                      height: 60,
                      decoration: BoxDecoration(
                        color: const Color(0xFF0D4F3C),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: const Color(0xFF8B4513), width: 2),
                      ),
                      child: const Center(
                        child: Icon(
                          Icons.casino,
                          color: Colors.white,
                          size: 30,
                        ),
                      ),
                    ),
                    const SizedBox(width: 20),
                    Container(
                      width: 20,
                      height: 20,
                      decoration: BoxDecoration(
                        color: Colors.amber,
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                    const SizedBox(width: 20),
                    Container(
                      width: 40,
                      height: 60,
                      decoration: BoxDecoration(
                        color: const Color(0xFF0D4F3C),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: const Color(0xFF8B4513), width: 2),
                      ),
                      child: const Center(
                        child: Icon(
                          Icons.casino,
                          color: Colors.white,
                          size: 30,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 40),
              const Text(
                "Ready to play poker?",
                style: TextStyle(
                  color: Colors.blueAccent,
                  fontSize: 32,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 40),
              ElevatedButton(
                onPressed: () => onReadyPressed(),
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blueAccent,
                  padding: const EdgeInsets.symmetric(horizontal: 50, vertical: 15),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(30),
                  ),
                ),
                child: const Text(
                  "I'm Ready!",
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              ),
              const SizedBox(height: 50),
              Container(
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: const Color(0xFF1B1E2C),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.blueAccent.withAlpha(76)),
                ),
                child: Column(
                  children: [
                    const Text(
                      "POKER CONTROLS",
                      style: TextStyle(
                        color: Colors.blueAccent,
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 15),
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        _controlKey("F", "Fold"),
                        const SizedBox(width: 10),
                        _controlKey("C", "Call"),
                        const SizedBox(width: 10),
                        _controlKey("K", "Check"),
                        const SizedBox(width: 10),
                        _controlKey("B", "Bet"),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    }

    // If ready but waiting for opponent
    return Center(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: const Color(0xFF1B1E2C).withAlpha(230),
          borderRadius: BorderRadius.circular(15),
          boxShadow: [
            BoxShadow(
              color: Colors.blueAccent.withAlpha(76),
              spreadRadius: 3,
              blurRadius: 10,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(
              Icons.casino,
              size: 50,
              color: Colors.blueAccent,
            ),
            const SizedBox(height: 20),
            const Text(
              "Waiting for players to get ready...",
              style: TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: 40,
              height: 40,
              child: CircularProgressIndicator(
                color: Colors.blueAccent,
                backgroundColor: Colors.grey.withAlpha(51),
                strokeWidth: 4,
              ),
            ),
          ],
        ),
      ),
    );
  }

  // Helper widget for control key display
  Widget _controlKey(String key, String action) {
    return Column(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: Colors.grey.shade800,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: Colors.grey.shade600),
          ),
          child: Center(
            child: Text(
              key,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ),
        const SizedBox(height: 5),
        Text(
          action,
          style: const TextStyle(
            color: Colors.white70,
            fontSize: 12,
          ),
        ),
      ],
    );
  }

  Future<void> handleInput(String playerId, String data) async {
    await _sendKeyInput(data);
  }

  Future<void> _sendKeyInput(String data) async {
    try {
      switch (data.toUpperCase()) {
        case 'F':
          await pokerModel.fold();
          break;
        case 'C':
          await pokerModel.callBet();
          break;
        case 'K':
          await pokerModel.check();
          break;
        case 'B':
          // Smart default: bet/raise to 3x current bet when available.
          final g = pokerModel.game;
          int currentBet = g?.currentBet ?? 0;
          // Find current table to get blinds
          final tid = pokerModel.currentTableId;
          final table = tid == null
              ? null
              : pokerModel.tables.where((t) => t.id == tid).cast<UiTable?>().firstWhere(
                    (t) => t != null,
                    orElse: () => null,
                  );
          final bb = table?.bigBlind ?? 0;
          final targetTotal = math.max(currentBet, bb * 3);
          // Send total bet amount to server
          if (targetTotal > 0) {
            await pokerModel.makeBet(targetTotal);
          }
          break;
        default:
          return;
      }
    } catch (e) {
      print('Poker input error: $e');
    }
  }

  String get name => 'Poker';
}

class PokerPainter extends CustomPainter {
  final UiGameState gameState;
  // This is the viewer's player ID (hero), not necessarily the player to act.
  final String currentPlayerId;
  // Used to stagger simple reveal animations at showdown
  final int showdownStartMs;
  
  PokerPainter(this.gameState, this.currentPlayerId, {Listenable? repaint})
      : showdownStartMs = DateTime.now().millisecondsSinceEpoch,
        super(repaint: repaint);

  @override
  void paint(Canvas canvas, Size size) {
    final centerX = size.width / 2;
    final centerY = size.height / 2;
    final tableRadius = (size.width * 0.4).clamp(100.0, 200.0);

    // Draw poker table
    _drawTable(canvas, size, centerX, centerY, tableRadius);
    
    // Draw community cards
    _drawCommunityCards(canvas, centerX, centerY, tableRadius);
    
    // Draw players
    _drawPlayers(canvas, size, centerX, centerY, tableRadius);

    // Draw hero hole cards as an overlay near the bottom center.
    _drawHeroHoleCards(canvas, size);

    // Draw current player's timebank badge last so it sits above cards/badges.
    _drawCurrentTimebank(canvas, size, centerX, centerY, tableRadius);
  }

  void _drawTable(Canvas canvas, Size size, double centerX, double centerY, double tableRadius) {
    // Table surface
    final tablePaint = Paint()
      ..color = const Color(0xFF0D4F3C)
      ..style = PaintingStyle.fill;
    
    canvas.drawCircle(Offset(centerX, centerY), tableRadius, tablePaint);
    
    // Table border
    final borderPaint = Paint()
      ..color = const Color(0xFF8B4513)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8;
    
    canvas.drawCircle(Offset(centerX, centerY), tableRadius, borderPaint);
  }

  void _drawCommunityCards(Canvas canvas, double centerX, double centerY, double tableRadius) {
    if (gameState.communityCards.isEmpty) return;

    final cardWidth = 30.0;
    final cardHeight = 42.0;
    final cardSpacing = 5.0;
    final totalWidth = (gameState.communityCards.length * cardWidth) + 
                      ((gameState.communityCards.length - 1) * cardSpacing);
    final startX = centerX - (totalWidth / 2);
    final cardY = centerY - 20;

    for (int i = 0; i < gameState.communityCards.length; i++) {
      final card = gameState.communityCards[i];
      final cardX = startX + (i * (cardWidth + cardSpacing));
      
      _drawCard(canvas, cardX, cardY, cardWidth, cardHeight, card);
    }
  }

  void _drawCard(Canvas canvas, double x, double y, double width, double height, pr.Card card) {
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
        text: '${card.value}\n${_getSuitSymbol(card.suit)}',
        style: TextStyle(
          color: _getSuitColor(card.suit),
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

  void _drawCardBack(Canvas canvas, double x, double y, double width, double height) {
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

  String _getSuitSymbol(String suit) {
    switch (suit.toLowerCase()) {
      case 'hearts': return '♥';
      case 'diamonds': return '♦';
      case 'clubs': return '♣';
      case 'spades': return '♠';
      default: return suit;
    }
  }

  Color _getSuitColor(String suit) {
    switch (suit.toLowerCase()) {
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

  void _drawPlayers(Canvas canvas, Size size, double centerX, double centerY, double tableRadius) {
    final playerRadius = 30.0;
    final players = gameState.players;
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
      
      final playerX = centerX + (tableRadius + 50) * math.cos(angle);
      final playerY = centerY + (tableRadius + 50) * math.sin(angle);

      _drawPlayer(
        canvas,
        playerX,
        playerY,
        playerRadius,
        player,
        i,
        angle,
      );

      if (player.id != currentPlayerId) {
        final hasAnyCards = player.hand.isNotEmpty;
        if (gameState.phase == pr.GamePhase.SHOWDOWN) {
          if (hasAnyCards) {
            // Reveal known opponent hands near their seat with a subtle slide-in.
            final cw = 18.0;
            final ch = cw * 1.4;
            final gap = 4.0;
            final startX = playerX - cw - gap / 2;
            final baseY = playerY - playerRadius - ch - 6;
            final now = DateTime.now().millisecondsSinceEpoch;
            final elapsed = (now - showdownStartMs - i * 120);
            final t = (elapsed / 450.0).clamp(0.0, 1.0);
            final y = baseY + (1.0 - t) * 14.0;
            _drawCard(canvas, startX, y, cw, ch, player.hand[0]);
            if (player.hand.length > 1) {
              _drawCard(canvas, startX + cw + gap, y, cw, ch, player.hand[1]);
            }
          } else {
            // If still hidden at showdown, show subtle backs.
            final cw = 16.0;
            final ch = cw * 1.4;
            final gap = 4.0;
            final startX = playerX - cw - gap / 2;
            final y = playerY - playerRadius - ch - 6;
            _drawCardBack(canvas, startX, y, cw, ch);
            _drawCardBack(canvas, startX + cw + gap, y, cw, ch);
          }
        } else if (!hasAnyCards && (gameState.phase != pr.GamePhase.WAITING && gameState.phase != pr.GamePhase.NEW_HAND_DEALING)) {
          // Non-showdown phases: use backs to indicate in-hand cards for opponents.
          final cw = 16.0;
          final ch = cw * 1.4;
          final gap = 4.0;
          final startX = playerX - cw - gap / 2;
          final y = playerY - playerRadius - ch - 6; // place just above the seat circle
          _drawCardBack(canvas, startX, y, cw, ch);
          _drawCardBack(canvas, startX + cw + gap, y, cw, ch);
        }
      }
    }
  }

  void _drawPlayer(
    Canvas canvas,
    double x,
    double y,
    double radius,
    UiPlayer player,
    int index,
    double angle,
  ) {
    final isHero = player.id == currentPlayerId;
    // Compute turn highlight based on authoritative currentPlayerId from
    // the game state to avoid transient races in per-player isTurn flags.
    final isCurrent = player.id == gameState.currentPlayerId;
    final heroColor = const Color(0xFF2E6DD8);
    final otherColor = Colors.grey.shade700;
    
    // Player circle
    final playerPaint = Paint()
      ..color = isHero ? heroColor : otherColor
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
      ..color = isCurrent ? Colors.yellowAccent : Colors.white24
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
    final badges = <_SeatBadge>[];
    
    if (player.isDealer) {
      badges.add(const _SeatBadge('D', Colors.amber));
    }
    if (player.isSmallBlind) {
      badges.add(const _SeatBadge('SB', Colors.cyan));
    }
    if (player.isBigBlind) {
      badges.add(const _SeatBadge('BB', Colors.pinkAccent));
    }
    // Add ALL-IN badge when player is all-in
    if (player.isAllIn) {
      badges.add(const _SeatBadge('ALL-IN', Colors.redAccent));
    }
    _drawRoleBadges(canvas, x, y, radius, badges, isHero, angle);

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
      final chipBadgeHeight = 16.0;
      final chipBadgeRect = RRect.fromRectAndRadius(
        Rect.fromLTWH(
          x - chipBadgeWidth / 2,
          y + radius + 8,
          chipBadgeWidth,
          chipBadgeHeight,
        ),
        const Radius.circular(8),
      );
      final chipBgPaint = Paint()..color = Colors.black.withOpacity(0.7);
      canvas.drawRRect(chipBadgeRect, chipBgPaint);
      
      chipText.paint(
        canvas,
        Offset(x - chipText.width / 2, y + radius + 10),
      );
    }
  }

  @override
  bool shouldRepaint(covariant PokerPainter old) =>
      old.gameState != gameState || old.currentPlayerId != currentPlayerId;

  void _drawRoleBadges(Canvas canvas, double centerX, double centerY, double radius, List<_SeatBadge> badges, bool isHero, double angle) {
    if (badges.isEmpty) return;

    const double badgeHeight = 18.0;
    const double horizontalPadding = 8.0;
    const double gap = 5.0;
    const textStyle = TextStyle(
      color: Colors.black,
      fontSize: 11,
      fontWeight: FontWeight.w900,
    );

    final layouts = <_BadgeLayout>[];
    double totalWidth = -gap;
    for (final badge in badges) {
      final painter = TextPainter(
        text: TextSpan(text: badge.label, style: textStyle),
        textDirection: TextDirection.ltr,
      )..layout();
      final width = painter.width + horizontalPadding * 2;
      layouts.add(_BadgeLayout(badge, painter, width));
      totalWidth += width + gap;
    }

    // Use less spacing for hero at bottom to avoid overlap with hole cards
    // Hero is at angle ≈ pi/2 (90 degrees = bottom)
    final isAtBottom = (angle - math.pi / 2).abs() < 0.1;
    final verticalOffset = (isHero && isAtBottom) ? 12.0 : 30.0;
    
    double drawX = centerX - totalWidth / 2;
    final drawY = centerY - radius - badgeHeight - verticalOffset;
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

  void _drawHeroHoleCards(Canvas canvas, Size size) {
    // Find hero in current players
    UiPlayer? hero;
    for (final p in gameState.players) {
      if (p.id == currentPlayerId) {
        hero = p;
        break;
      }
    }
    if (hero == null) return;

    // do not draw during waiting phase
    if (gameState.phase == pr.GamePhase.WAITING) return;

    // Determine sizes relative to viewport
    final cw = math.min(size.width * 0.06, 54.0);
    final ch = cw * 1.4;
    final gap = cw * 0.12;

    // Bottom-center placement with safe margin
    final centerX = size.width / 2;
    // Leave room for action buttons (at bottom:20) + player circle + badges + chips
    // Total: ~20 (buttons) + 60 (player radius*2) + 20 (badges) + 20 (chips) + gaps = 180px
    final marginBottom = 80.0;
    final y = size.height - ch - marginBottom;
    final startX = centerX - cw - gap / 2;

    final cards = hero.hand;
    if (cards.length >= 2) {
      _drawCard(canvas, startX, y, cw, ch, cards[0]);
      _drawCard(canvas, startX + cw + gap, y, cw, ch, cards[1]);
    } else {
      // Draw facedown placeholders when cards are hidden/unavailable
      _drawCardBack(canvas, startX, y, cw, ch);
      _drawCardBack(canvas, startX + cw + gap, y, cw, ch);
    }
  }

  void _drawCurrentTimebank(Canvas canvas, Size size, double centerX, double centerY, double tableRadius) {
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
    final playerX = centerX + (tableRadius + 50) * math.cos(angle);
    final playerY = centerY + (tableRadius + 50) * math.sin(angle);

    final tbText = TextPainter(
      text: TextSpan(
        text: '⏱ ${remSec.toStringAsFixed(1)}s',
        style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w700),
      ),
      textDirection: TextDirection.ltr,
    )..layout();

    final badgeW = tbText.width + 12;
    const badgeH = 18.0;
    // Prefer to the right of the seat; fallback to left if clipping.
    double bx = playerX + playerRadius + 12;
    double by = playerY - badgeH / 2;
    if (bx + badgeW > size.width - 4) {
      bx = playerX - playerRadius - 12 - badgeW;
    }
    if (by < 2) by = 2;
    if (by + badgeH > size.height - 2) by = size.height - 2 - badgeH;

    final badgeRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(bx, by, badgeW, badgeH),
      const Radius.circular(8),
    );
    final tbBg = Paint()..color = Colors.black.withOpacity(0.9);
    canvas.drawRRect(badgeRect, tbBg);
    tbText.paint(canvas, Offset(bx + (badgeW - tbText.width) / 2, by + (badgeH - tbText.height) / 2));
  }
}

class _SeatBadge {
  const _SeatBadge(this.label, this.color);

  final String label;
  final Color color;
}

class _BadgeLayout {
  _BadgeLayout(this.badge, this.textPainter, this.width);

  final _SeatBadge badge;
  final TextPainter textPainter;
  final double width;
}
