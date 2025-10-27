import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

extension _HandRankLabel on pr.HandRank {
  String get label {
    switch (this) {
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
        return name;
    }
  }
}

/// Showdown UI: shows board, winners (with best 5 cards), and any revealed hands.
class ShowdownWidget extends StatelessWidget {
  final PokerModel model;
  final VoidCallback? onLeave;

  const ShowdownWidget({super.key, required this.model, this.onLeave});

  @override
  Widget build(BuildContext context) {
    final game = model.game;
    final winners = model.lastWinners;
    final players = game?.players ?? const [];

    // Play:Gdiffspliters who chose to show their hole cards (non-empty hand list)
    final revealedPlayers = players.where((p) => p.hand.isNotEmpty).toList();

    // Build a set of card keys used by any winner for lightweight highlighting on the board.
    final Set<String> winningBoardKeys = {};
    if (game != null && game.communityCards.isNotEmpty && winners.isNotEmpty) {
      final ccKeys = game.communityCards.map(_cardKey).toSet();
      for (final w in winners) {
        for (final c in w.bestHand) {
          final k = _cardKey(c);
          if (ccKeys.contains(k)) winningBoardKeys.add(k);
        }
      }
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 24),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 1000),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const Icon(Icons.emoji_events, size: 64, color: Colors.amber),
              const SizedBox(height: 8),
              const Text(
                'Showdown',
                style: TextStyle(fontSize: 26, fontWeight: FontWeight.w800, color: Colors.white),
              ),
              const SizedBox(height: 18),

              // Board / Community cards
              if (game != null && game.communityCards.isNotEmpty)
                _Section(
                  title: 'Community Cards',
                  child: _CardRow(
                    cards: game.communityCards,
                    highlightKeys: winningBoardKeys,
                    dimNonHighlights: winningBoardKeys.isNotEmpty,
                  ),
                ),

              // Winners section
              if (winners.isNotEmpty) ...[
                const SizedBox(height: 12),
                _Section(
                  title: winners.length == 1 ? 'Winner' : 'Winners',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      for (final w in winners)
                        _WinnerRow(
                          winner: w,
                          name: players.firstWhere(
                            (p) => p.id == w.playerId,
                            orElse: () => UiPlayer(
                              id: w.playerId,
                              name: w.playerId,
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
                              handDesc: '',
                            ),
                          ).name,
                          handDesc: players
                                  .firstWhere(
                                      (p) => p.id == w.playerId,
                                      orElse: () => const UiPlayer(
                                            id: '',
                                            name: '',
                                            balance: 0,
                                            hand: [],
                                            currentBet: 0,
                                            folded: false,
                                            isTurn: false,
                                            isAllIn: false,
                                            isDealer: false,
                                            isSmallBlind: false,
                                            isBigBlind: false,
                                            isReady: false,
                                            handDesc: '',
                                          ))
                                  .handDesc,
                        ),
                    ],
                  ),
                ),
              ],

              // Revealed hands (non-winners may still choose to show)
              if (revealedPlayers.isNotEmpty) ...[
                const SizedBox(height: 12),
                _Section(
                  title: 'Revealed Hands',
                  child: Wrap(
                    spacing: 16,
                    runSpacing: 12,
                    children: [
                      for (final p in revealedPlayers)
                        _RevealedPlayerCard(name: p.name, cards: p.hand),
                    ],
                  ),
                ),
              ],

              const SizedBox(height: 16),

              // Controls
              Wrap(
                alignment: WrapAlignment.center,
                spacing: 12,
                runSpacing: 8,
                children: [
                  if (!(model.myCardsShown))
                    ElevatedButton.icon(
                      onPressed: model.showCards,
                      icon: const Icon(Icons.visibility),
                      label: const Text('Show My Cards'),
                    ),
                  if (model.myCardsShown)
                    ElevatedButton.icon(
                      onPressed: model.hideCards,
                      icon: const Icon(Icons.visibility_off),
                      label: const Text('Hide My Cards'),
                    ),
                  ElevatedButton(
                    onPressed: onLeave ?? model.leaveTable,
                    style: ElevatedButton.styleFrom(backgroundColor: Colors.redAccent),
                    child: const Text('Leave Table'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({required this.title, required this.child});
  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Text(
            title,
            style: const TextStyle(color: Colors.white70, fontSize: 16, fontWeight: FontWeight.w700),
          ),
        ),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.black.withOpacity(0.55),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.white24),
          ),
          child: child,
        ),
      ],
    );
  }
}

class _WinnerRow extends StatelessWidget {
  const _WinnerRow({required this.winner, required this.name, this.handDesc = ''});
  final UiWinner winner;
  final String name;
  final String handDesc;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Flexible(
                child: Text(
                  '$name',
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 16),
                ),
              ),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: Colors.amber.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(color: Colors.amber, width: 1),
                ),
                child: Text(
                  winner.handRank.label,
                  style: const TextStyle(color: Colors.amber, fontWeight: FontWeight.w700),
                ),
              ),
              const SizedBox(width: 8),
              if (winner.winnings > 0)
                Text(
                  '+${winner.winnings} chips',
                  style: const TextStyle(color: Colors.greenAccent, fontWeight: FontWeight.w700),
                ),
            ],
          ),
          const SizedBox(height: 8),
          if (handDesc.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 6.0),
              child: Text(
                handDesc,
                style: const TextStyle(color: Colors.white70, fontStyle: FontStyle.italic),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          _CardRow(cards: winner.bestHand),
        ],
      ),
    );
  }
}

class _RevealedPlayerCard extends StatelessWidget {
  const _RevealedPlayerCard({required this.name, required this.cards});
  final String name;
  final List<pr.Card> cards;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.45),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white24),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            name,
            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700),
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 6),
          _CardRow(cards: cards),
        ],
      ),
    );
  }
}

class _CardRow extends StatelessWidget {
  const _CardRow({required this.cards, this.highlightKeys = const {}, this.dimNonHighlights = false});
  final List<pr.Card> cards;
  final Set<String> highlightKeys;
  final bool dimNonHighlights;

  @override
  Widget build(BuildContext context) {
    final count = cards.length;
    final width = MediaQuery.of(context).size.width;
    final maxAcross = (width > 600) ? 8 : 6;
    final perCard = (width.clamp(320.0, 1000.0) as double) / maxAcross;
    final cardW = perCard.clamp(48.0, 72.0);

    return Row(
      mainAxisAlignment: MainAxisAlignment.start,
      children: [
        for (var i = 0; i < count; i++) ...[
          _PlayingCard(
            card: cards[i],
            width: cardW,
            highlighted: highlightKeys.contains(_cardKey(cards[i])),
            dimmed: dimNonHighlights && !highlightKeys.contains(_cardKey(cards[i])),
          ),
          if (i != count - 1) const SizedBox(width: 8),
        ]
      ],
    );
  }
}

class _PlayingCard extends StatelessWidget {
  const _PlayingCard({required this.card, required this.width, this.highlighted = false, this.dimmed = false});
  final pr.Card card;
  final double width;
  final bool highlighted;
  final bool dimmed;

  @override
  Widget build(BuildContext context) {
    final w = width;
    final h = w * 1.4;
    final suitSym = _suitSymbol(card.suit);
    final isRed = _isRed(card.suit);

    return AnimatedContainer(
      width: w,
      height: h,
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: highlighted ? Colors.amber : Colors.black, width: highlighted ? 3 : 2),
        boxShadow: [
          if (highlighted)
            BoxShadow(color: Colors.amber.withOpacity(0.6), blurRadius: 16, spreadRadius: 1),
          BoxShadow(color: Colors.black.withOpacity(0.30), blurRadius: 6, spreadRadius: 1),
        ],
      ),
      duration: const Duration(milliseconds: 250),
      curve: Curves.easeOut,
      child: Padding(
        padding: const EdgeInsets.all(4.0),
        child: Opacity(
          opacity: dimmed ? 0.55 : 1.0,
          child: Stack(
          children: [
            Align(
              alignment: Alignment.topLeft,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    card.value,
                    style: TextStyle(
                      color: isRed ? Colors.red : Colors.black,
                      fontWeight: FontWeight.w900,
                      fontSize: w * 0.30,
                    ),
                  ),
                  Text(
                    suitSym,
                    style: TextStyle(
                      color: isRed ? Colors.red : Colors.black,
                      fontWeight: FontWeight.w700,
                      fontSize: w * 0.26,
                    ),
                  ),
                ],
              ),
            ),
            Align(
              alignment: Alignment.bottomRight,
              child: Transform.rotate(
                angle: 3.14159,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      card.value,
                      style: TextStyle(
                        color: isRed ? Colors.red : Colors.black,
                        fontWeight: FontWeight.w900,
                        fontSize: w * 0.30,
                      ),
                    ),
                    Text(
                      suitSym,
                      style: TextStyle(
                        color: isRed ? Colors.red : Colors.black,
                        fontWeight: FontWeight.w700,
                        fontSize: w * 0.26,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            Center(
              child: Text(
                suitSym,
                style: TextStyle(
                  color: isRed ? Colors.red : Colors.black,
                  fontSize: w * 0.60,
                ),
              ),
            ),
          ],
          ),
        ),
      ),
    );
  }

  static String _suitSymbol(String suit) {
    final s = suit.toLowerCase();
    if (s == 'hearts' || suit == '♥' || suit == '\u2665') return '♥';
    if (s == 'diamonds' || suit == '♦' || suit == '\u2666') return '♦';
    if (s == 'clubs' || suit == '♣' || suit == '\u2663') return '♣';
    if (s == 'spades' || suit == '♠' || suit == '\u2660') return '♠';
    // Unknown suit string; show raw
    return suit;
  }

  static bool _isRed(String suit) {
    final s = suit.toLowerCase();
    return s == 'hearts' || s == 'diamonds' || suit == '♥' || suit == '♦' || suit == '\u2665' || suit == '\u2666';
  }
}

String _cardKey(pr.Card c) => '${c.value}|${c.suit}'.toLowerCase();
