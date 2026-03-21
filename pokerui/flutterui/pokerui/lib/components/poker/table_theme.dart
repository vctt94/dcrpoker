import 'package:flutter/material.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/theme/colors.dart';

const Color decredBlue = PokerColors.primary;
const Color decredGreen = PokerColors.accent;

class CardColorTheme {
  final Color heartsColor;
  final Color diamondsColor;
  final Color clubsColor;
  final Color spadesColor;

  const CardColorTheme({
    required this.heartsColor,
    required this.diamondsColor,
    required this.clubsColor,
    required this.spadesColor,
  });

  static const CardColorTheme standard = CardColorTheme(
    heartsColor: Color(0xFFD7263D),
    diamondsColor: Color(0xFFE65100),
    clubsColor: Color.fromARGB(255, 12, 86, 235),
    spadesColor: Color.fromARGB(255, 1, 14, 32),
  );

  static const CardColorTheme decred = CardColorTheme(
    heartsColor: PokerColors.accent,
    diamondsColor: PokerColors.primary,
    clubsColor: Color(0xFF0D2B5A),
    spadesColor: Color(0xFF0A4A3A),
  );
}

class TableThemeConfig {
  final String key;
  final String displayName;
  final Color feltColor;
  final Color borderColor;

  const TableThemeConfig({
    required this.key,
    required this.displayName,
    required this.feltColor,
    required this.borderColor,
  });

  static const TableThemeConfig decred = TableThemeConfig(
    key: 'decred',
    displayName: 'Decred Blue',
    feltColor: PokerColors.feltDecred,
    borderColor: PokerColors.accent,
  );

  static const TableThemeConfig classic = TableThemeConfig(
    key: 'classic',
    displayName: 'Classic Felt',
    feltColor: PokerColors.feltClassic,
    borderColor: PokerColors.feltBorderClassic,
  );

  static const TableThemeConfig decredInverse = TableThemeConfig(
    key: 'decred_inverse',
    displayName: 'Decred Green',
    feltColor: PokerColors.accent,
    borderColor: PokerColors.feltDecred,
  );

  static const List<TableThemeConfig> presets = [decred, decredInverse, classic];

  static TableThemeConfig fromKey(String key) {
    final normalized = key.toLowerCase();
    for (final theme in presets) {
      if (theme.key == normalized) return theme;
    }
    return classic;
  }
}

CardColorTheme cardColorThemeFromKey(String key) {
  final normalized = key.toLowerCase();
  switch (normalized) {
    case 'decred':
      return CardColorTheme.decred;
    case 'standard':
    default:
      return CardColorTheme.standard;
  }
}

String cardColorThemeKey(CardColorTheme theme) {
  if (theme == CardColorTheme.decred) return 'decred';
  return 'standard';
}

String cardColorThemeDisplayName(CardColorTheme theme) {
  if (theme == CardColorTheme.decred) return 'Decred';
  return 'Standard';
}

const List<CardColorTheme> cardColorThemePresets = [
  CardColorTheme.standard,
  CardColorTheme.decred,
];

double cardSizeMultiplierFromKey(String key) {
  final normalized = key.toLowerCase();
  switch (normalized) {
    case 'xs':
      return 0.6;
    case 'small':
      return 0.8;
    case 'large':
      return 1.2;
    case 'xl':
      return 1.4;
    case 'medium':
    default:
      return 1.0;
  }
}

double uiSizeMultiplierFromKey(String key) {
  final normalized = key.toLowerCase();
  switch (normalized) {
    case 'xs':
      return 0.7;
    case 'small':
      return 0.85;
    case 'large':
      return 1.15;
    case 'xl':
      return 1.3;
    case 'medium':
    default:
      return 1.0;
  }
}

class PokerThemeConfig {
  final TableThemeConfig tableTheme;
  final CardColorTheme cardTheme;
  final double cardSizeMultiplier;
  final double uiSizeMultiplier;
  final bool showTableLogo;
  final String logoPosition;

  const PokerThemeConfig({
    required this.tableTheme,
    required this.cardTheme,
    required this.cardSizeMultiplier,
    required this.uiSizeMultiplier,
    required this.showTableLogo,
    required this.logoPosition,
  });

  factory PokerThemeConfig.fromContext(BuildContext context) {
    return PokerThemeConfig(
      tableTheme: TableThemeConfig.fromKey(context.tableTheme),
      cardTheme: cardColorThemeFromKey(context.cardTheme),
      cardSizeMultiplier: cardSizeMultiplierFromKey(context.cardSize),
      uiSizeMultiplier: uiSizeMultiplierFromKey(context.uiSize),
      showTableLogo: context.showTableLogo,
      logoPosition: context.logoPosition,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PokerThemeConfig &&
          runtimeType == other.runtimeType &&
          tableTheme == other.tableTheme &&
          cardTheme == other.cardTheme &&
          cardSizeMultiplier == other.cardSizeMultiplier &&
          uiSizeMultiplier == other.uiSizeMultiplier &&
          showTableLogo == other.showTableLogo &&
          logoPosition == other.logoPosition;

  @override
  int get hashCode =>
      tableTheme.hashCode ^
      cardTheme.hashCode ^
      cardSizeMultiplier.hashCode ^
      uiSizeMultiplier.hashCode ^
      showTableLogo.hashCode ^
      logoPosition.hashCode;
}
