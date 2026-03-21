import 'package:flutter/material.dart';
import 'package:pokerui/components/dialogs/last_showdown.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

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

    final accentColor = isWin
        ? PokerColors.success
        : isDraw
            ? PokerColors.warning
            : PokerColors.danger;

    return Center(
      child: LayoutBuilder(
        builder: (context, constraints) {
          return Container(
            padding: const EdgeInsets.all(PokerSpacing.xxl),
            margin: const EdgeInsets.symmetric(horizontal: PokerSpacing.xl),
            constraints: BoxConstraints(
              maxHeight: constraints.maxHeight - 64,
              maxWidth: (constraints.maxWidth - 48).clamp(0, 520).toDouble(),
            ),
            decoration: BoxDecoration(
              color: PokerColors.surface.withAlpha(240),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: accentColor.withOpacity(0.3)),
              boxShadow: [
                BoxShadow(
                  color: accentColor.withAlpha(50),
                  spreadRadius: 4,
                  blurRadius: 15,
                ),
              ],
            ),
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    isWin
                        ? Icons.emoji_events
                        : isDraw
                            ? Icons.handshake
                            : Icons.sports_tennis,
                    size: constraints.maxWidth < 360 ? 56 : 80,
                    color: accentColor,
                  ),
                  const SizedBox(height: PokerSpacing.xl),
                  Text(
                    "Game End!",
                    style: PokerTypography.displayLarge.copyWith(
                      fontSize: constraints.maxWidth < 360 ? 24 : 32,
                      color: accentColor,
                    ),
                  ),
                  const SizedBox(height: PokerSpacing.lg),
                  Text(
                    message.isNotEmpty ? message : 'Game ended',
                    style: PokerTypography.titleMedium,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: PokerSpacing.xxl),
                  if (hasShowdown) ...[
                    Container(
                      width: double.infinity,
                      padding: const EdgeInsets.all(PokerSpacing.lg),
                      decoration: BoxDecoration(
                        color: PokerColors.surfaceDim,
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: PokerColors.borderSubtle),
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
                            const SizedBox(height: PokerSpacing.md),
                            Text('Community cards',
                                style: PokerTypography.labelSmall),
                            const SizedBox(height: PokerSpacing.sm),
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
                          const SizedBox(height: PokerSpacing.md),
                          Align(
                            alignment: Alignment.centerRight,
                            child: TextButton.icon(
                              onPressed: () =>
                                  LastShowdownDialog.show(context, model),
                              icon: const Icon(Icons.remove_red_eye, size: 16),
                              label: const Text('View showdown'),
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: PokerSpacing.xl),
                  ],
                  Wrap(
                    alignment: WrapAlignment.spaceEvenly,
                    spacing: 12,
                    runSpacing: 12,
                    children: [
                      ElevatedButton.icon(
                        onPressed: model.leaveTable,
                        icon: const Icon(Icons.home, size: 18),
                        label: const Text("Main Menu"),
                      ),
                      ElevatedButton.icon(
                        onPressed: model.leaveTable,
                        icon: const Icon(Icons.refresh, size: 18),
                        label: const Text("Play Again"),
                        style: ElevatedButton.styleFrom(
                          backgroundColor: PokerColors.success,
                          foregroundColor: Colors.black,
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
