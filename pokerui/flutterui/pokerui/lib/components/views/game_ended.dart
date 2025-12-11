import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/dialogs/last_showdown.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/models/poker.dart';

class GameEndedView extends StatelessWidget {
  const GameEndedView({super.key, required this.model});
  final PokerModel model;

  String _winnerLabel(UiWinner w) {
    final player =
        model.showdownPlayers.firstWhere((p) => p.id == w.playerId, orElse: () => UiPlayer(
              id: w.playerId,
              name: '',
              balance: 0,
              hand: const [],
              currentBet: 0,
              folded: false,
              isTurn: false,
              isAllIn: false,
              isDealer: false,
              isSmallBlind: false,
              isBigBlind: false,
              isReady: false,
              isDisconnected: false,
              handDesc: '',
            ));
    if (player.name.isNotEmpty) return player.name;
    final pid = w.playerId;
    return pid.length > 8 ? '${pid.substring(0, 8)}...' : pid;
  }

  @override
  Widget build(BuildContext context) {
    final message = model.gameEndingMessage;
    final isWin = message.toLowerCase().contains('won') || 
                  message.toLowerCase().contains('congratulations');
    final isDraw = message.toLowerCase().contains('draw');
    final hasShowdown =
        model.hasLastShowdown || model.lastWinners.isNotEmpty || model.showdownPlayers.isNotEmpty;

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
            if (hasShowdown) ...[
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.04),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.white24),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text(
                          'Last showdown',
                          style: TextStyle(
                            color: Colors.white,
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        Text(
                          'Pot: ${model.showdownPot}',
                          style: const TextStyle(
                            color: Colors.amber,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: (model.lastWinners.isNotEmpty
                              ? model.lastWinners
                              : model.showdownPlayers
                                  .map((p) => UiWinner(
                                      playerId: p.id,
                                      handRank: pr.HandRank.HIGH_CARD,
                                      bestHand: const [],
                                      winnings: 0))
                              .toList())
                          .map((w) => Chip(
                                backgroundColor: Colors.green.withOpacity(0.15),
                                label: Text(
                                  '${_winnerLabel(w)}${w.winnings > 0 ? " +${w.winnings}" : ""}',
                                  style: const TextStyle(color: Colors.white),
                                ),
                              ))
                          .toList(),
                    ),
                    if (model.showdownCommunityCards.isNotEmpty) ...[
                      const SizedBox(height: 12),
                      const Text(
                        'Community cards',
                        style: TextStyle(
                          color: Colors.white70,
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: model.showdownCommunityCards
                            .map((c) => Padding(
                                  padding: const EdgeInsets.symmetric(horizontal: 3),
                                  child: SizedBox(
                                    width: 40,
                                    height: 56,
                                    child: CardFace(card: c),
                                  ),
                                ))
                            .toList(),
                      ),
                    ],
                    const SizedBox(height: 12),
                    Align(
                      alignment: Alignment.centerRight,
                      child: TextButton.icon(
                        onPressed: () => LastShowdownDialog.show(context, model),
                        icon: const Icon(Icons.remove_red_eye, color: Colors.white70),
                        label: const Text(
                          'View showdown',
                          style: TextStyle(color: Colors.white70),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 24),
            ],
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
