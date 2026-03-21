import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class IdleView extends StatelessWidget {
  const IdleView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.style, size: 64, color: PokerColors.primary.withOpacity(0.6)),
          const SizedBox(height: PokerSpacing.lg),
          Text('Welcome to Poker!', style: PokerTypography.headlineLarge),
          const SizedBox(height: PokerSpacing.sm),
          Text(
            'Browse tables or create one to start playing',
            style: PokerTypography.bodySmall,
          ),
          const SizedBox(height: PokerSpacing.xxl),
          ElevatedButton.icon(
            onPressed: model.browseTables,
            icon: const Icon(Icons.table_restaurant, size: 18),
            label: const Text('Browse Tables'),
          ),
        ],
      ),
    );
  }
}
