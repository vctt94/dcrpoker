import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

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

  PokerGame(this.playerId, this.pokerModel);

  Widget buildWidget(UiGameState gameState, FocusNode focusNode, {VoidCallback? onReadyHotkey}) {
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
                            painter: PokerPainter(gameState, playerId),
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
          // For now, bet a fixed amount. In a real implementation, 
          // you'd want a bet input dialog
          await pokerModel.makeBet(100);
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
  
  PokerPainter(this.gameState, this.currentPlayerId);

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
    
    for (int i = 0; i < gameState.players.length; i++) {
      final player = gameState.players[i];
      final angle = (i * 2 * 3.14159) / gameState.players.length;
      final playerX = centerX + (tableRadius + 50) * math.cos(angle);
      final playerY = centerY + (tableRadius + 50) * math.sin(angle);
      
      _drawPlayer(canvas, playerX, playerY, playerRadius, player, i);

      // Draw opponent backs near their seat if their hand is hidden but they are in-hand.
      // If a player's hand is known (e.g., at showdown or hero), it will be drawn elsewhere.
      if (player.id != currentPlayerId) {
        final hasAnyCards = player.hand.isNotEmpty;
        if (!hasAnyCards && (gameState.phase != pr.GamePhase.WAITING && gameState.phase != pr.GamePhase.NEW_HAND_DEALING)) {
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

  void _drawPlayer(Canvas canvas, double x, double y, double radius, UiPlayer player, int index) {
    // Player circle
    final playerPaint = Paint()
      ..color = player.id == currentPlayerId ? Colors.blue : Colors.grey.shade600
      ..style = PaintingStyle.fill;
    
    canvas.drawCircle(Offset(x, y), radius, playerPaint);
    
    // Player border
    final borderPaint = Paint()
      ..color = player.isTurn ? Colors.yellow : Colors.white
      ..style = PaintingStyle.stroke
      ..strokeWidth = player.isTurn ? 3 : 1;
    
    canvas.drawCircle(Offset(x, y), radius, borderPaint);
    
    // Player name
    final textPainter = TextPainter(
      text: TextSpan(
        text: player.name.isNotEmpty ? player.name.substring(0, 1).toUpperCase() : 'P${index + 1}',
        style: const TextStyle(
          color: Colors.white,
          fontSize: 12,
          fontWeight: FontWeight.bold,
        ),
      ),
      textDirection: TextDirection.ltr,
    );
    textPainter.layout();
    textPainter.paint(
      canvas,
      Offset(x - textPainter.width / 2, y - textPainter.height / 2),
    );
    
    // Player chips
    if (player.balance > 0) {
      final chipText = TextPainter(
        text: TextSpan(
          text: '${player.balance}',
          style: const TextStyle(
            color: Colors.white,
            fontSize: 8,
          ),
        ),
        textDirection: TextDirection.ltr,
      );
      chipText.layout();
      chipText.paint(
        canvas,
        Offset(x - chipText.width / 2, y + radius + 5),
      );
    }
  }

  @override
  bool shouldRepaint(covariant PokerPainter old) => 
      old.gameState != gameState || old.currentPlayerId != currentPlayerId;

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
    // Leave room for action buttons overlay positioned at bottom:20 in UI
    final marginBottom = 96.0;
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
}
