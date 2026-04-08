import 'package:flutter/material.dart';
import 'package:pokerui/components/dialogs/last_showdown.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class GameEndedView extends StatelessWidget {
  const GameEndedView({super.key, required this.model});
  final PokerModel model;

  String _winnerLabel(UiWinner w) {
    final showdown = model.showdown;
    final player = (showdown?.players ?? const <UiPlayer>[])
        .firstWhere((p) => p.id == w.playerId,
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

  String _headline(bool isWin, bool isDraw) {
    if (isWin) return 'You Won';
    if (isDraw) return 'Table Finished';
    return 'You Lost';
  }

  bool? _explicitGameResult() => model.didWinGame;
  int? _gameEndAmountAtoms() => model.gameEndAmountAtoms;

  String _formatDcr(int atoms) => '${(atoms / 1e8).toStringAsFixed(4)} DCR';

  String _summary(bool isWin, bool isDraw) {
    final amountAtoms = _gameEndAmountAtoms();
    if (amountAtoms != null && amountAtoms != 0) {
      final displayAtoms = amountAtoms.abs();
      if (isWin) {
        return 'Congratulations! You won ${_formatDcr(displayAtoms)}.';
      }
      if (!isDraw) {
        return 'Sorry, you lost ${_formatDcr(displayAtoms)}.';
      }
    }
    if (isWin) {
      return 'Congratulations! You won the table.';
    }
    if (isDraw) {
      final names =
          model.showdownWinners.map(_winnerLabel).toList(growable: false);
      return names.isEmpty ? 'Table finished.' : 'Winners: ${names.join(', ')}';
    }
    final message = model.gameEndingMessage.trim();
    if (message.isNotEmpty && message != 'Game ended') {
      return message;
    }
    return 'Game finished.';
  }

  @override
  Widget build(BuildContext context) {
    final explicitResult = _explicitGameResult();
    final hasWinners = model.showdownWinners.isNotEmpty;
    final iWon = model.showdownWinners.any((w) => w.playerId == model.playerId);
    final isDraw = explicitResult == null && model.showdownWinners.length > 1;
    final isWin = explicitResult ?? (hasWinners && iWon);
    final title = _headline(isWin, isDraw);
    final message = _summary(isWin, isDraw);
    final hasShowdown = model.hasShowdown;

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
                    title,
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
                    const SizedBox(height: PokerSpacing.sm),
                    TextButton.icon(
                      key: const Key(
                        'game-ended-view-showdown-button',
                      ),
                      onPressed: () => LastShowdownDialog.show(context, model),
                      icon: const Icon(
                        Icons.remove_red_eye,
                        size: 16,
                      ),
                      label: const Text('View last hand'),
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
