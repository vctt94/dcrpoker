import 'package:flutter/material.dart';

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

  /// Standard card colors (red for hearts/diamonds, black for clubs/spades)
  static const CardColorTheme standard = CardColorTheme(
    heartsColor: Color(0xFFD7263D),
    diamondsColor: Color(0xFFE65100),
    clubsColor: Color.fromARGB(255, 12, 86, 235), // Bright blue
    spadesColor: Color.fromARGB(255, 1, 14, 32),
  );

  /// DCR-themed card colors (bright for hearts/diamonds, dark for clubs/spades)
  static const CardColorTheme decred = CardColorTheme(
    heartsColor: Color(0xFF2ED6A1), // Decred green (bright)
    diamondsColor: Color(0xFF2970FF), // Decred blue (bright)
    clubsColor: Color(0xFF0D2B5A), // Darker blue (dark, still readable)
    spadesColor: Color(0xFF0A4A3A), // Darker green (dark, still readable)
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
    feltColor: Color(0xFF091440),
    borderColor: Color(0xFF2ED6A1), // Decred green
  );

  static const TableThemeConfig classic = TableThemeConfig(
    key: 'classic',
    displayName: 'Classic Felt',
    feltColor: Color(0xFF0D4F3C),
    borderColor: Color(0xFF8B4513),
  );

  static const TableThemeConfig decredInverse = TableThemeConfig(
    key: 'decred_inverse',
    displayName: 'Decred Green',
    feltColor: Color(0xFF2ED6A1), // Decred green as felt
    borderColor: Color(0xFF091440), // Dark blue as border
  );

  static const List<TableThemeConfig> presets = [
    decred,
    decredInverse,
    classic,
  ];

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
