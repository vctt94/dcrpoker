import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class GameEndedView extends StatelessWidget {
  const GameEndedView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    final message = model.gameEndingMessage;
    final isWin = message.toLowerCase().contains('won') || 
                  message.toLowerCase().contains('congratulations');
    final isDraw = message.toLowerCase().contains('draw');

    return Center(
      child: Container(
        padding: const EdgeInsets.all(32),
        margin: const EdgeInsets.symmetric(horizontal: 24),
        decoration: BoxDecoration(
          color: const Color(0xFF1B1E2C).withAlpha(240),
          borderRadius: BorderRadius.circular(20),
          boxShadow: [
            BoxShadow(
              color: (isWin
                      ? Colors.green
                      : isDraw
                          ? Colors.orange
                          : Colors.red)
                  .withAlpha(76),
              spreadRadius: 4,
              blurRadius: 15,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Game over icon
            Icon(
              isWin
                  ? Icons.emoji_events
                  : isDraw
                      ? Icons.handshake
                      : Icons.sports_tennis,
              size: 80,
              color: isWin
                  ? Colors.green
                  : isDraw
                      ? Colors.orange
                      : Colors.red,
            ),
            const SizedBox(height: 24),
            // Game over title
            Text(
              "Game End!",
              style: TextStyle(
                fontSize: 32,
                fontWeight: FontWeight.bold,
                color: isWin
                    ? Colors.green
                    : isDraw
                        ? Colors.orange
                        : Colors.red,
              ),
            ),
            const SizedBox(height: 16),
            // Result message
            Text(
              message.isNotEmpty ? message : 'Game ended',
              style: const TextStyle(
                fontSize: 20,
                color: Colors.white,
                fontWeight: FontWeight.w500,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 32),
            // Action buttons
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
              children: [
                ElevatedButton.icon(
                  onPressed: () {
                    // Leave table and return to main menu
                    model.leaveTable();
                  },
                  icon: const Icon(Icons.home),
                  label: const Text("Main Menu"),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.blueAccent,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 24, vertical: 12),
                  ),
                ),
                ElevatedButton.icon(
                  onPressed: () {
                    // Leave table (can rejoin for quick rematch)
                    model.leaveTable();
                  },
                  icon: const Icon(Icons.refresh),
                  label: const Text("Play Again"),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.green,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 24, vertical: 12),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

