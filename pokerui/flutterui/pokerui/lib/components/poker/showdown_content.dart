import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/shadows.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

/// Reusable widget that displays showdown content (player hands and winners).
class ShowdownContent extends StatelessWidget {
  const ShowdownContent({
    super.key,
    required this.showdown,
    this.heroId = '',
    this.showHeader = true,
    this.showCloseButton = false,
    this.onClose,
    this.cardScale = 1.0,
  });

  final UiShowdownState showdown;
  final String heroId;
  final bool showHeader;
  final bool showCloseButton;
  final VoidCallback? onClose;
  final double cardScale;

  String _handRankName(pr.HandRank rank) {
    switch (rank) {
      case pr.HandRank.HIGH_CARD:
        return 'High Card';
      case pr.HandRank.PAIR:
        return 'Pair';
      case pr.HandRank.TWO_PAIR:
        return 'Two Pair';
      case pr.HandRank.THREE_OF_A_KIND:
        return 'Three of a Kind';
      case pr.HandRank.STRAIGHT:
        return 'Straight';
      case pr.HandRank.FLUSH:
        return 'Flush';
      case pr.HandRank.FULL_HOUSE:
        return 'Full House';
      case pr.HandRank.FOUR_OF_A_KIND:
        return 'Four of a Kind';
      case pr.HandRank.STRAIGHT_FLUSH:
        return 'Straight Flush';
      case pr.HandRank.ROYAL_FLUSH:
        return 'Royal Flush';
      default:
        return rank.name;
    }
  }

  String _playerLabel(String playerId) {
    final players = showdown.players;
    final idx = players.indexWhere((p) => p.id == playerId);
    if (idx >= 0) {
      final player = players[idx];
      if (player.name.isNotEmpty) return player.name;
      return 'Player ${idx + 1}';
    }
    return playerId.length > 8 ? '${playerId.substring(0, 8)}...' : playerId;
  }

  UiWinner? _getWinner(String playerId) {
    try {
      return showdown.winners
          .firstWhere((winner) => winner.playerId == playerId);
    } catch (_) {
      return null;
    }
  }

  String _playerSummary(UiPlayer player, UiWinner? winner) {
    final parts = <String>[];
    if (winner != null) {
      parts.add(_handRankName(winner.handRank));
      if (winner.winnings > 0) {
        parts.add('+${winner.winnings}');
      }
    } else if (player.handDesc.trim().isNotEmpty) {
      parts.add(player.handDesc.trim());
    }
    if (player.folded) {
      parts.add('Folded');
    }
    return parts.join(' • ');
  }

  @override
  Widget build(BuildContext context) {
    final uiSpec = PokerUiSpec.fromContext(context);
    final cardTheme = cardColorThemeFromKey(context.cardTheme);
    final communityCards = showdown.communityCards;
    final players = showdown.players;
    final winners = showdown.winners;
    final pot = showdown.pot;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (showHeader) _buildHeader(winners.isNotEmpty, pot),
        Padding(
          padding: const EdgeInsets.all(PokerSpacing.lg),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _buildBoardStrip(communityCards, cardTheme, uiSpec),
              if (players.isNotEmpty) const SizedBox(height: PokerSpacing.lg),
              if (players.isNotEmpty) ...[
                Row(
                  children: [
                    Text(
                      'Hands',
                      style: PokerTypography.labelLarge.copyWith(
                        color: PokerColors.textSecondary,
                      ),
                    ),
                    const Spacer(),
                    Text(
                      '${players.length} players',
                      style: PokerTypography.labelSmall,
                    ),
                  ],
                ),
                const SizedBox(height: PokerSpacing.sm),
                Container(
                  decoration: BoxDecoration(
                    color: PokerColors.surfaceDim,
                    borderRadius: BorderRadius.circular(18),
                    border: Border.all(color: PokerColors.borderSubtle),
                    boxShadow: PokerShadows.subtle,
                  ),
                  child: Column(
                    children: [
                      for (int i = 0; i < players.length; i++) ...[
                        _buildPlayerHandRow(players[i], cardTheme, uiSpec),
                        if (i != players.length - 1)
                          Divider(
                            height: 1,
                            color: PokerColors.borderSubtle.withOpacity(0.8),
                          ),
                      ],
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
        if (showCloseButton && onClose != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(
              PokerSpacing.lg,
              0,
              PokerSpacing.lg,
              PokerSpacing.lg,
            ),
            child: SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: onClose,
                style: ElevatedButton.styleFrom(
                  backgroundColor: PokerColors.primary,
                  foregroundColor: PokerColors.textPrimary,
                  padding:
                      const EdgeInsets.symmetric(vertical: PokerSpacing.md),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(14),
                  ),
                ),
                child: Text(
                  'Close',
                  style: PokerTypography.titleSmall.copyWith(
                    color: PokerColors.textPrimary,
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildBoardStrip(
    List<pr.Card> communityCards,
    CardColorTheme cardTheme,
    PokerUiSpec uiSpec,
  ) {
    const totalBoardSlots = 5;
    final boardCardSize = uiSpec.showdownBoardCardSize(surfaceScale: cardScale);
    return Container(
      key: const Key('showdown-board-strip'),
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Board',
            style: PokerTypography.labelSmall.copyWith(
              color: PokerColors.textSecondary,
            ),
          ),
          const SizedBox(height: PokerSpacing.sm),
          Wrap(
            spacing: PokerSpacing.xs,
            runSpacing: PokerSpacing.xs,
            children: List<Widget>.generate(totalBoardSlots, (index) {
              final hasCard = index < communityCards.length;
              return SizedBox(
                width: boardCardSize.width,
                height: boardCardSize.height,
                child: hasCard
                    ? CardFace(
                        key: Key('showdown-board-card-$index'),
                        card: communityCards[index],
                        cardTheme: cardTheme,
                      )
                    : _BoardPlaceholderSlot(
                        key: Key('showdown-board-slot-$index'),
                      ),
              );
            }),
          ),
        ],
      ),
    );
  }

  Widget _buildHeader(bool hasWinners, int pot) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surfaceBright,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(18)),
        border: Border(
          bottom: BorderSide(
            color: PokerColors.borderSubtle.withOpacity(0.9),
          ),
        ),
      ),
      child: Row(
        children: [
          Text(
            hasWinners ? 'Showdown' : 'Last Showdown',
            style: PokerTypography.titleLarge,
          ),
          const Spacer(),
          if (pot > 0)
            Text(
              'Pot $pot',
              style: PokerTypography.labelSmall.copyWith(
                color: PokerColors.warning,
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildPlayerHandRow(
    UiPlayer player,
    CardColorTheme cardTheme,
    PokerUiSpec uiSpec,
  ) {
    final winner = _getWinner(player.id);
    final isWinner = winner != null;
    final isMe = player.id == heroId;
    final showCards = player.hand.isNotEmpty &&
        (!player.folded || player.cardsRevealed || isMe);
    final summary = _playerSummary(player, winner);

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.md,
        vertical: 10,
      ),
      decoration: BoxDecoration(
        color: isWinner
            ? PokerColors.success.withOpacity(0.04)
            : Colors.transparent,
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        _playerLabel(player.id) + (isMe ? ' (you)' : ''),
                        style: PokerTypography.titleSmall.copyWith(
                          color: isMe
                              ? PokerColors.primary.withOpacity(0.96)
                              : PokerColors.textPrimary,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    if (isWinner)
                      Text(
                        'Winner',
                        style: PokerTypography.labelSmall.copyWith(
                          color: PokerColors.success,
                        ),
                      ),
                  ],
                ),
                if (summary.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    summary,
                    style: PokerTypography.bodySmall.copyWith(
                      color: isWinner
                          ? PokerColors.success.withOpacity(0.92)
                          : PokerColors.textSecondary,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(width: PokerSpacing.md),
          _buildCardsRow(
            player: player,
            showCards: showCards,
            cardTheme: cardTheme,
            uiSpec: uiSpec,
          ),
        ],
      ),
    );
  }

  Widget _buildCardsRow({
    required UiPlayer player,
    required bool showCards,
    required CardColorTheme cardTheme,
    required PokerUiSpec uiSpec,
  }) {
    final playerCardSize =
        uiSpec.showdownPlayerCardSize(surfaceScale: cardScale);

    if (showCards) {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: player.hand.asMap().entries.map((entry) {
          final index = entry.key;
          final card = entry.value;
          return Padding(
            padding: const EdgeInsets.only(left: PokerSpacing.xs),
            child: SizedBox(
              width: playerCardSize.width,
              height: playerCardSize.height,
              child: CardFace(
                key: Key('showdown-player-card-${player.id}-$index'),
                card: card,
                cardTheme: cardTheme,
              ),
            ),
          );
        }).toList(),
      );
    }

    if (player.folded) {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: List.generate(
          2,
          (_) => Padding(
            padding: const EdgeInsets.only(left: PokerSpacing.xs),
            child: Container(
              width: playerCardSize.width,
              height: playerCardSize.height,
              decoration: BoxDecoration(
                color: PokerColors.surfaceBright,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: PokerColors.borderMedium),
              ),
              child: const Icon(
                Icons.block,
                color: PokerColors.textMuted,
                size: 16,
              ),
            ),
          ),
        ),
      );
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: PokerSpacing.xs),
          child: SizedBox(
            width: playerCardSize.width,
            height: playerCardSize.height,
            child: CardBack(),
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(left: PokerSpacing.xs),
          child: SizedBox(
            width: playerCardSize.width,
            height: playerCardSize.height,
            child: CardBack(),
          ),
        ),
      ],
    );
  }
}

class _BoardPlaceholderSlot extends StatelessWidget {
  const _BoardPlaceholderSlot({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.04),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: PokerColors.borderSubtle.withOpacity(0.75),
        ),
      ),
    );
  }
}
