import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class ShowdownSidebar extends StatelessWidget {
  const ShowdownSidebar({
    super.key,
    required this.model,
    this.visible = true,
    @Deprecated('use visible') bool? isVisible,
    // Keep the old `result` param for backward compat; unused internally.
    dynamic result,
  });

  final PokerModel model;
  final bool visible;

  @override
  Widget build(BuildContext context) {
    if (!visible) return const SizedBox.shrink();

    final winners = model.lastWinners;
    final players = model.showdownPlayers;
    final communityCards = model.showdownCommunityCards;
    final pot = model.showdownPot;

    return Container(
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim.withOpacity(0.96),
        border: const Border(right: BorderSide(color: PokerColors.borderSubtle)),
      ),
      child: ListView(
        padding: const EdgeInsets.all(PokerSpacing.lg),
        children: [
          Row(
            children: [
              Text('Showdown', style: PokerTypography.titleMedium.copyWith(
                color: PokerColors.primary,
              )),
              const Spacer(),
              if (pot > 0)
                Text('Pot: $pot', style: PokerTypography.chipCount.copyWith(
                  color: PokerColors.warning, fontSize: 12,
                )),
            ],
          ),
          const SizedBox(height: PokerSpacing.md),

          if (winners.isNotEmpty) ...[
            for (final w in winners)
              _WinnerRow(winner: w, players: players),
            const Divider(color: PokerColors.borderSubtle),
            const SizedBox(height: PokerSpacing.sm),
          ],

          for (final p in players)
            _PlayerRow(
              player: p,
              isWinner: winners.any((w) => w.playerId == p.id),
            ),

          if (communityCards.isNotEmpty) ...[
            const SizedBox(height: PokerSpacing.md),
            Text('Community', style: PokerTypography.labelSmall),
            const SizedBox(height: PokerSpacing.sm),
            Wrap(
              spacing: 4, runSpacing: 4,
              children: communityCards.map((c) => SizedBox(
                width: 28, height: 40,
                child: CardFace(card: c),
              )).toList(),
            ),
          ],
        ],
      ),
    );
  }
}

class _WinnerRow extends StatelessWidget {
  const _WinnerRow({required this.winner, required this.players});
  final UiWinner winner;
  final List<UiPlayer> players;

  @override
  Widget build(BuildContext context) {
    final player = players.firstWhere(
        (p) => p.id == winner.playerId,
        orElse: () => UiPlayer(
              id: winner.playerId, name: '', balance: 0, hand: const [],
              currentBet: 0, folded: false, isTurn: false, isAllIn: false,
              isDealer: false, isSmallBlind: false, isBigBlind: false,
              isReady: false, isDisconnected: false, handDesc: '',
            ));
    final name = player.name.isNotEmpty ? player.name : winner.playerId;

    return Container(
      margin: const EdgeInsets.only(bottom: PokerSpacing.sm),
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.accent.withOpacity(0.08),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: PokerColors.accent.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          Icon(Icons.emoji_events, color: PokerColors.accent, size: 20),
          const SizedBox(width: PokerSpacing.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name, style: PokerTypography.titleSmall.copyWith(
                  color: PokerColors.accent,
                )),
                if (winner.winnings > 0)
                  Text('Won ${winner.winnings}',
                    style: PokerTypography.chipCount.copyWith(fontSize: 12)),
              ],
            ),
          ),
          if (winner.bestHand.isNotEmpty)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: winner.bestHand.take(5).map((c) => Padding(
                padding: const EdgeInsets.only(left: 2),
                child: SizedBox(width: 22, height: 31, child: CardFace(card: c)),
              )).toList(),
            ),
        ],
      ),
    );
  }
}

class _PlayerRow extends StatelessWidget {
  const _PlayerRow({required this.player, required this.isWinner});
  final UiPlayer player;
  final bool isWinner;

  @override
  Widget build(BuildContext context) {
    final name = player.name.isNotEmpty ? player.name : player.id;
    final cards = player.hand;
    final cardW = 24.0;
    final cardH = cardW * 1.4;

    return Padding(
      padding: const EdgeInsets.only(bottom: PokerSpacing.sm),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  style: PokerTypography.playerName.copyWith(
                    color: isWinner ? PokerColors.accent : PokerColors.textPrimary,
                  ),
                ),
                if (player.handDesc.isNotEmpty)
                  Text(player.handDesc,
                    style: PokerTypography.bodySmall.copyWith(fontSize: 10)),
              ],
            ),
          ),
          if (cards.isNotEmpty)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: cards.map((c) => Padding(
                padding: const EdgeInsets.only(left: 3),
                child: SizedBox(width: cardW, height: cardH, child: CardFace(card: c)),
              )).toList(),
            ),
        ],
      ),
    );
  }
}
