import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class TournamentOverView extends StatelessWidget {
  const TournamentOverView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        padding: const EdgeInsets.all(PokerSpacing.xxl),
        margin: const EdgeInsets.symmetric(horizontal: PokerSpacing.xl),
        decoration: BoxDecoration(
          color: PokerColors.surface.withAlpha(240),
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: PokerColors.accent.withOpacity(0.3)),
          boxShadow: [
            BoxShadow(
              color: PokerColors.accent.withAlpha(50),
              spreadRadius: 4,
              blurRadius: 15,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.flag, size: 64, color: PokerColors.accent),
            const SizedBox(height: PokerSpacing.lg),
            Text('Tournament Over!', style: PokerTypography.headlineLarge.copyWith(
              color: PokerColors.accent,
            )),
            const SizedBox(height: PokerSpacing.xl),
            ElevatedButton(
              onPressed: model.leaveTable,
              child: const Text('Return to Lobby'),
            ),
          ],
        ),
      ),
    );
  }
}
