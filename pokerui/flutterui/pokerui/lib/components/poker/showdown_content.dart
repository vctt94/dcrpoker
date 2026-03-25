import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

/// Reusable widget that displays showdown content (community cards, player hands, winners).
/// Used by last-hand sidebars, dialogs, and related showdown surfaces.
class ShowdownContent extends StatelessWidget {
  const ShowdownContent({
    super.key,
    required this.model,
    this.showHeader = true,
    this.showCloseButton = false,
    this.onClose,
  });

  final PokerModel model;
  final bool showHeader;
  final bool showCloseButton;
  final VoidCallback? onClose;

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
    final players = model.showdownPlayers;
    final idx = players.indexWhere((p) => p.id == playerId);
    if (idx >= 0) {
      final p = players[idx];
      if (p.name.isNotEmpty) return p.name;
      return 'Player ${idx + 1}';
    }
    return playerId.length > 8 ? '${playerId.substring(0, 8)}...' : playerId;
  }

  bool _isWinner(String playerId) {
    return model.lastWinners.any((w) => w.playerId == playerId);
  }

  UiWinner? _getWinner(String playerId) {
    try {
      return model.lastWinners.firstWhere((w) => w.playerId == playerId);
    } catch (_) {
      return null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final cardTheme = cardColorThemeFromKey(context.cardTheme);
    final communityCards = model.showdownCommunityCards;
    final players = model.showdownPlayers;
    final winners = model.lastWinners;
    final pot = model.showdownPot;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (showHeader)
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: Colors.black.withOpacity(0.3),
              borderRadius:
                  const BorderRadius.vertical(top: Radius.circular(14)),
            ),
            child: Row(
              children: [
                const Icon(Icons.history, color: Colors.amber, size: 24),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    winners.isNotEmpty ? 'Showdown' : 'Last Showdown',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 18,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
                if (pot > 0)
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.amber.withOpacity(0.2),
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: Colors.amber.withOpacity(0.5)),
                    ),
                    child: Text(
                      'Pot: $pot',
                      style: const TextStyle(
                        color: Colors.amber,
                        fontSize: 14,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                if (showCloseButton && onClose != null) ...[
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: onClose,
                    icon: const Icon(Icons.close, color: Colors.white70),
                    tooltip: 'Close',
                  ),
                ],
              ],
            ),
          ),
        if (communityCards.isNotEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              children: [
                const Text(
                  'Community Cards',
                  style: TextStyle(
                    color: Colors.white70,
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 8),
                Wrap(
                  alignment: WrapAlignment.center,
                  spacing: 6,
                  runSpacing: 6,
                  children: communityCards.map((card) {
                    return SizedBox(
                      width: 50,
                      height: 70,
                      child: CardFace(card: card, cardTheme: cardTheme),
                    );
                  }).toList(),
                ),
              ],
            ),
          ),
        const Divider(color: Colors.white24, height: 1),
        if (winners.isNotEmpty)
          Container(
            padding: const EdgeInsets.all(12),
            color: Colors.green.withOpacity(0.1),
            child: Column(
              children: [
                const Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(Icons.emoji_events, color: Colors.amber, size: 20),
                    SizedBox(width: 6),
                    Text(
                      'Winner',
                      style: TextStyle(
                        color: Colors.amber,
                        fontSize: 14,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                for (final winner in winners)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Flexible(
                          child: Text(
                            _playerLabel(winner.playerId),
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 14,
                              fontWeight: FontWeight.bold,
                            ),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Flexible(
                          child: Text(
                            '${_handRankName(winner.handRank)} (+${winner.winnings})',
                            style: const TextStyle(
                              color: Colors.greenAccent,
                              fontSize: 13,
                            ),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
        const Divider(color: Colors.white24, height: 1),
        Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            children: [
              const Text(
                'Player Hands',
                style: TextStyle(
                  color: Colors.white70,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(height: 12),
              for (final player in players) _buildPlayerHandRow(player),
            ],
          ),
        ),
        if (showCloseButton && onClose != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: onClose,
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blue.shade700,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
                child: const Text('Close'),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildPlayerHandRow(UiPlayer player) {
    final isWinner = _isWinner(player.id);
    final winner = _getWinner(player.id);
    final isMe = player.id == model.playerId;
    final showCards = player.hand.isNotEmpty &&
        (!player.folded || player.cardsRevealed || isMe);

    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: isWinner
            ? Colors.green.withOpacity(0.15)
            : Colors.white.withOpacity(0.05),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: isWinner
              ? Colors.greenAccent.withOpacity(0.5)
              : (isMe ? Colors.blue.withOpacity(0.5) : Colors.white12),
          width: isWinner ? 2 : 1,
        ),
      ),
      child: Row(
        children: [
          // Player info
          Expanded(
            flex: 2,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    if (isWinner)
                      const Padding(
                        padding: EdgeInsets.only(right: 4),
                        child: Icon(Icons.emoji_events,
                            color: Colors.amber, size: 16),
                      ),
                    Flexible(
                      child: Text(
                        _playerLabel(player.id) + (isMe ? ' (you)' : ''),
                        style: TextStyle(
                          color: isMe ? Colors.lightBlueAccent : Colors.white,
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
                if (player.handDesc.isNotEmpty || winner != null)
                  Text(
                    winner != null
                        ? _handRankName(winner.handRank)
                        : player.handDesc,
                    style: TextStyle(
                      color: isWinner ? Colors.greenAccent : Colors.white54,
                      fontSize: 11,
                      fontStyle: FontStyle.italic,
                    ),
                  ),
                if (player.folded)
                  const Text(
                    'Folded',
                    style: TextStyle(
                      color: Colors.redAccent,
                      fontSize: 11,
                    ),
                  ),
              ],
            ),
          ),

          // Hole cards
          if (showCards)
            Row(
              children: player.hand.map((card) {
                return Padding(
                  padding: const EdgeInsets.only(left: 4),
                  child: SizedBox(
                    width: 36,
                    height: 50,
                    child: CardFace(card: card),
                  ),
                );
              }).toList(),
            )
          else if (player.folded)
            Row(
              children: [
                for (int i = 0; i < 2; i++)
                  Padding(
                    padding: const EdgeInsets.only(left: 4),
                    child: Container(
                      width: 36,
                      height: 50,
                      decoration: BoxDecoration(
                        color: Colors.grey.shade800,
                        borderRadius: BorderRadius.circular(6),
                        border: Border.all(color: Colors.grey.shade600),
                      ),
                      child: const Center(
                        child: Icon(Icons.block, color: Colors.grey, size: 20),
                      ),
                    ),
                  ),
              ],
            )
          else
            Row(
              children: [
                for (int i = 0; i < 2; i++)
                  const Padding(
                    padding: const EdgeInsets.only(left: 4),
                    child: SizedBox(
                      width: 36,
                      height: 50,
                      child: CardBack(),
                    ),
                  ),
              ],
            ),
        ],
      ),
    );
  }
}
