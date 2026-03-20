import 'package:flutter/material.dart';
import 'package:pokerui/components/dialogs/last_showdown.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/config.dart';

class GameEndedView extends StatelessWidget {
  const GameEndedView({super.key, required this.model});
  final PokerModel model;

  String _winnerLabel(UiWinner w) {
    final player = model.showdownPlayers.firstWhere((p) => p.id == w.playerId,
        orElse: () => UiPlayer(
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

  String _winnerChipLabel(UiWinner w) {
    if (w.playerId == model.playerId) {
      return 'You';
    }
    return _winnerLabel(w);
  }

  String _winnerSummary() {
    final winners = model.lastWinners;
    if (winners.isEmpty) {
      return model.gameEndingMessage.isNotEmpty
          ? model.gameEndingMessage
          : 'Game ended';
    }

    final iWon = winners.any((w) => w.playerId == model.playerId);
    if (!iWon) {
      return model.gameEndingMessage.isNotEmpty
          ? model.gameEndingMessage
          : 'You lost.';
    }

    final names = winners.map(_winnerLabel).toList(growable: false);
    if (names.length == 1) {
      return 'Congratulations! You are the winner.';
    }
    if (iWon) {
      return 'Congratulations! You are one of the winners.';
    }
    return 'Winners: ${names.join(', ')}';
  }

  @override
  Widget build(BuildContext context) {
    final cardTheme = cardColorThemeFromKey(context.cardTheme);
    final message = _winnerSummary();
    final hasWinners = model.lastWinners.isNotEmpty;
    final iWon = model.lastWinners.any((w) => w.playerId == model.playerId);
    final isDraw = model.lastWinners.length > 1;
    final isWin = hasWinners && iWon;
    final showWinnerSummary = hasWinners && (iWon || isDraw);
    final hasShowdown = model.hasLastShowdown ||
        model.lastWinners.isNotEmpty ||
        model.showdownPlayers.isNotEmpty;

    return Center(
      child: LayoutBuilder(
        builder: (context, constraints) {
          return Container(
            padding: const EdgeInsets.all(32),
            margin: const EdgeInsets.symmetric(horizontal: 24),
            constraints: BoxConstraints(
              maxHeight: constraints.maxHeight - 64,
              maxWidth: (constraints.maxWidth - 48).clamp(0, 520),
            ),
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
            child: SingleChildScrollView(
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
                    size: constraints.maxWidth < 360 ? 56 : 80,
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
                      fontSize: constraints.maxWidth < 360 ? 24 : 32,
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
                    style: TextStyle(
                      fontSize: constraints.maxWidth < 360 ? 16 : 20,
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
                            children: [
                              Text(
                                showWinnerSummary
                                    ? (model.lastWinners.length > 1
                                        ? 'Winners'
                                        : 'Winner')
                                    : 'Last hand',
                                style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 16,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ],
                          ),
                          if (showWinnerSummary) ...[
                            const SizedBox(height: 8),
                            Wrap(
                              spacing: 8,
                              runSpacing: 8,
                              children: model.lastWinners
                                  .map((w) => Chip(
                                        backgroundColor:
                                            Colors.green.withOpacity(0.15),
                                        label: Text(
                                          _winnerChipLabel(w),
                                          style: const TextStyle(
                                              color: Colors.white),
                                        ),
                                      ))
                                  .toList(),
                            ),
                          ],
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
                            Wrap(
                              alignment: WrapAlignment.center,
                              spacing: 6,
                              runSpacing: 6,
                              children: model.showdownCommunityCards
                                  .map((c) => SizedBox(
                                        width: 40,
                                        height: 56,
                                        child: CardFace(
                                            card: c, cardTheme: cardTheme),
                                      ))
                                  .toList(),
                            ),
                          ],
                          const SizedBox(height: 12),
                          Align(
                            alignment: Alignment.centerRight,
                            child: TextButton.icon(
                              onPressed: () =>
                                  LastShowdownDialog.show(context, model),
                              icon: const Icon(Icons.remove_red_eye,
                                  color: Colors.white70),
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
                  Wrap(
                    alignment: WrapAlignment.spaceEvenly,
                    spacing: 12,
                    runSpacing: 12,
                    children: [
                      ElevatedButton.icon(
                        onPressed: () => model.leaveTable(),
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
                        onPressed: () => model.leaveTable(),
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
        },
      ),
    );
  }
}
