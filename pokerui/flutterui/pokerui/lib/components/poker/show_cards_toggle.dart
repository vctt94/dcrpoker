import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class ShowCardsToggle extends StatelessWidget {
  const ShowCardsToggle({super.key, required this.model, this.compact = false});

  final PokerModel model;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final hasCards =
        (model.me?.hand.isNotEmpty ?? false) || model.myHoleCardsCache.isNotEmpty;
    if (!hasCards) return const SizedBox.shrink();

    final showing = model.me?.cardsRevealed ?? false;
    final label = showing ? 'Hide cards' : 'Show cards';
    final actionLabel = showing ? 'HIDE' : 'SHOW';
    final icon = showing ? Icons.visibility_off : Icons.visibility;
    final accent = showing ? Colors.amber : Colors.white70;
    final padding = compact
        ? const EdgeInsets.symmetric(horizontal: 10, vertical: 6)
        : const EdgeInsets.symmetric(horizontal: 12, vertical: 8);

    return Tooltip(
      message: actionLabel,
      child: OutlinedButton.icon(
        onPressed: () {
          if (showing) {
            model.hideCards();
          } else {
            model.showCards();
          }
        },
        icon: Icon(icon, size: compact ? 16 : 18),
        label: Text(label, style: TextStyle(fontSize: compact ? 12 : 13)),
        style: OutlinedButton.styleFrom(
          foregroundColor: accent,
          backgroundColor: Colors.black.withOpacity(0.35),
          side: BorderSide(
            color: showing ? Colors.amber.withOpacity(0.8) : Colors.white24,
          ),
          padding: padding,
          shape:
              RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
      ),
    );
  }
}
