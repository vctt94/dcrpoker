import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
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

  static const List<TableThemeConfig> presets = [
    decred,
    decredInverse,
    classic
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

enum PokerScalePreset {
  xs('xs'),
  small('small'),
  medium('medium'),
  large('large'),
  xl('xl');

  const PokerScalePreset(this.key);

  final String key;

  static PokerScalePreset fromKey(String key) {
    final normalized = key.toLowerCase();
    for (final preset in values) {
      if (preset.key == normalized) return preset;
    }
    return PokerScalePreset.medium;
  }
}

@immutable
class PokerUiSettings {
  const PokerUiSettings({
    required this.tableThemeKey,
    required this.cardThemeKey,
    required this.cardScale,
    required this.densityScale,
    required this.showTableLogo,
    required this.logoPosition,
  });

  final String tableThemeKey;
  final String cardThemeKey;
  final PokerScalePreset cardScale;
  final PokerScalePreset densityScale;
  final bool showTableLogo;
  final String logoPosition;

  factory PokerUiSettings.fromConfig(Config config) {
    return PokerUiSettings(
      tableThemeKey: config.tableTheme,
      cardThemeKey: config.cardTheme,
      cardScale: PokerScalePreset.fromKey(config.cardSize),
      densityScale: PokerScalePreset.fromKey(config.uiSize),
      showTableLogo: config.showTableLogo,
      logoPosition: config.logoPosition,
    );
  }

  factory PokerUiSettings.fromContext(BuildContext context) {
    return PokerUiSettings.fromConfig(context.config);
  }
}

@immutable
class PokerUiSpec {
  const PokerUiSpec({
    required this.settings,
    required this.breakpoint,
    required this.layoutMode,
    required this.tableTheme,
    required this.cardTheme,
    required this.cardSizeMultiplier,
    required this.uiSizeMultiplier,
    required this.spacingScale,
    required this.textScale,
    required this.iconScale,
  });

  final PokerUiSettings settings;
  final PokerBreakpoint breakpoint;
  final PokerLayoutMode layoutMode;
  final TableThemeConfig tableTheme;
  final CardColorTheme cardTheme;
  final double cardSizeMultiplier;
  final double uiSizeMultiplier;
  final double spacingScale;
  final double textScale;
  final double iconScale;

  factory PokerUiSpec.fromSettings(
    PokerUiSettings settings, {
    required Size viewportSize,
  }) {
    final breakpoint = PokerBreakpointQuery.fromWidth(viewportSize.width);
    final layoutMode = PokerSceneLayout.resolveMode(viewportSize);
    return _buildPokerUiSpec(
      settings: settings,
      breakpoint: breakpoint,
      layoutMode: layoutMode,
      tableTheme: TableThemeConfig.fromKey(settings.tableThemeKey),
      cardTheme: cardColorThemeFromKey(settings.cardThemeKey),
      cardSizeMultiplier: _cardScaleMultiplier(settings.cardScale),
      uiSizeMultiplier: _uiScaleMultiplier(settings.densityScale),
    );
  }

  factory PokerUiSpec.fromContext(BuildContext context) {
    return PokerUiSpec.fromSettings(
      PokerUiSettings.fromContext(context),
      viewportSize: MediaQuery.sizeOf(context),
    );
  }

  factory PokerUiSpec.fromTheme(
    PokerThemeConfig theme, {
    required Size viewportSize,
  }) {
    final breakpoint = PokerBreakpointQuery.fromWidth(viewportSize.width);
    final layoutMode = PokerSceneLayout.resolveMode(viewportSize);
    return _buildPokerUiSpec(
      settings: PokerUiSettings(
        tableThemeKey: theme.tableTheme.key,
        cardThemeKey: cardColorThemeKey(theme.cardTheme),
        cardScale: PokerScalePreset.fromKey(_scaleKeyForMultiplier(
          theme.cardSizeMultiplier,
          isCardScale: true,
        )),
        densityScale: PokerScalePreset.fromKey(_scaleKeyForMultiplier(
          theme.uiSizeMultiplier,
          isCardScale: false,
        )),
        showTableLogo: theme.showTableLogo,
        logoPosition: theme.logoPosition,
      ),
      breakpoint: breakpoint,
      layoutMode: layoutMode,
      tableTheme: theme.tableTheme,
      cardTheme: theme.cardTheme,
      cardSizeMultiplier: theme.cardSizeMultiplier,
      uiSizeMultiplier: theme.uiSizeMultiplier,
    );
  }

  Size get heroDockCardSize {
    final width = (42.0 * cardSizeMultiplier).clamp(24.0, 60.0).toDouble();
    return Size(width, width * 1.4);
  }

  PokerSeatSpec seatSpec({required bool isHeroSeat}) {
    return PokerSeatSpec.resolve(
      layoutMode: layoutMode,
      uiSizeMultiplier: uiSizeMultiplier,
      cardSizeMultiplier: cardSizeMultiplier,
      isHeroSeat: isHeroSeat,
    );
  }

  PokerPotSpec get potSpec {
    return PokerPotSpec.resolve(
      layoutMode: layoutMode,
      uiSizeMultiplier: uiSizeMultiplier,
    );
  }

  Size showdownBoardCardSize({double surfaceScale = 1.0}) {
    final width = _scaledCardWidth(
      36.0,
      min: 28.0,
      max: 80.0,
      surfaceScale: surfaceScale,
    );
    return Size(width, width * (52.0 / 36.0));
  }

  Size showdownPlayerCardSize({double surfaceScale = 1.0}) {
    final width = _scaledCardWidth(
      34.0,
      min: 26.0,
      max: 76.0,
      surfaceScale: surfaceScale,
    );
    return Size(width, width * (48.0 / 34.0));
  }

  Size gameEndedPreviewCardSize({double surfaceScale = 1.0}) {
    final width = _scaledCardWidth(
      40.0,
      min: 32.0,
      max: 72.0,
      surfaceScale: surfaceScale,
    );
    return Size(width, width * 1.4);
  }

  double scaleCommunityCardWidth(double baseWidth) {
    final maxWidth = switch (layoutMode) {
      PokerLayoutMode.compactPortrait => 72.0,
      PokerLayoutMode.compactLandscape => 76.0,
      PokerLayoutMode.standard => 80.0,
      PokerLayoutMode.wide => 84.0,
    };
    return (baseWidth * cardSizeMultiplier).clamp(20.0, maxWidth).toDouble();
  }

  double _scaledCardWidth(
    double baseWidth, {
    required double min,
    required double max,
    double surfaceScale = 1.0,
  }) {
    return (baseWidth * cardSizeMultiplier * surfaceScale)
        .clamp(min, max)
        .toDouble();
  }
}

@immutable
class PokerSeatSpec {
  const PokerSeatSpec({
    required this.isHeroSeat,
    required this.compactOpponent,
    required this.radius,
    required this.uiScale,
    required this.cardScale,
    required this.railCardWidth,
    required this.railCardHeight,
    required this.railCardGap,
    required this.railVisibleHeight,
    required this.railWidth,
    required this.plateWidth,
    required this.plateBaseLeft,
    required this.heroDockOverlap,
  });

  final bool isHeroSeat;
  final bool compactOpponent;
  final double radius;
  final double uiScale;
  final double cardScale;
  final double railCardWidth;
  final double railCardHeight;
  final double railCardGap;
  final double railVisibleHeight;
  final double railWidth;
  final double plateWidth;
  final double plateBaseLeft;
  final double heroDockOverlap;

  factory PokerSeatSpec.resolve({
    required PokerLayoutMode layoutMode,
    required double uiSizeMultiplier,
    required double cardSizeMultiplier,
    required bool isHeroSeat,
  }) {
    final compactOpponent =
        !isHeroSeat && layoutMode == PokerLayoutMode.compactPortrait;
    final radius = 28.0 * uiSizeMultiplier * (compactOpponent ? 0.74 : 1.0);
    final baseCardMultiplier =
        isHeroSeat ? 1.3 : (compactOpponent ? 0.76 : 1.0);
    final minCardWidth = isHeroSeat ? 42.0 : (compactOpponent ? 22.0 : 30.0);
    final maxCardWidth = isHeroSeat ? 70.0 : (compactOpponent ? 40.0 : 58.0);
    final cardWidth = (radius * baseCardMultiplier * cardSizeMultiplier)
        .clamp(minCardWidth, maxCardWidth)
        .toDouble();
    final cardHeight = cardWidth * 1.4;
    final cardGap =
        (cardWidth * 0.12).clamp(3.0, isHeroSeat ? 8.0 : 6.0).toDouble();
    final railSideInset =
        (cardWidth * (isHeroSeat ? 0.16 : 0.12)).clamp(4.0, 10.0).toDouble();
    final plateWidthBase =
        isHeroSeat ? 122.0 : (compactOpponent ? 84.0 : 108.0);
    final heroDockOverlapBase = switch (layoutMode) {
      PokerLayoutMode.compactPortrait => 18.0,
      PokerLayoutMode.compactLandscape => 22.0,
      PokerLayoutMode.standard => 28.0,
      PokerLayoutMode.wide => 34.0,
    };

    return PokerSeatSpec(
      isHeroSeat: isHeroSeat,
      compactOpponent: compactOpponent,
      radius: radius,
      uiScale: uiSizeMultiplier,
      cardScale: cardSizeMultiplier,
      railCardWidth: cardWidth,
      railCardHeight: cardHeight,
      railCardGap: cardGap,
      railVisibleHeight: cardHeight * 0.5,
      railWidth: (cardWidth * 2) + cardGap + (railSideInset * 2) + 4.0,
      plateWidth: (plateWidthBase * uiSizeMultiplier)
          .clamp(compactOpponent ? 72.0 : 90.0, 156.0)
          .toDouble(),
      plateBaseLeft:
          (radius + ((compactOpponent ? 14.0 : 22.0) * uiSizeMultiplier))
              .clamp(compactOpponent ? 22.0 : 40.0, 68.0)
              .toDouble(),
      heroDockOverlap:
          (heroDockOverlapBase * uiSizeMultiplier).clamp(12.0, 44.0).toDouble(),
    );
  }
}

@immutable
class PokerPotSpec {
  const PokerPotSpec({
    required this.stackLift,
    required this.totalGap,
    required this.potLabelFontSize,
    required this.potLabelBlur,
    required this.betStackChipSize,
    required this.betStackLabelFontSize,
    required this.betStackLabelGap,
    required this.betStackLabelBlur,
    required this.potPileChipSize,
  });

  final double stackLift;
  final double totalGap;
  final double potLabelFontSize;
  final double potLabelBlur;
  final double betStackChipSize;
  final double betStackLabelFontSize;
  final double betStackLabelGap;
  final double betStackLabelBlur;
  final double potPileChipSize;

  factory PokerPotSpec.resolve({
    required PokerLayoutMode layoutMode,
    required double uiSizeMultiplier,
  }) {
    final stackLift = switch (layoutMode) {
      PokerLayoutMode.compactPortrait => 6.0 * uiSizeMultiplier,
      PokerLayoutMode.compactLandscape => 4.0 * uiSizeMultiplier,
      PokerLayoutMode.standard => 3.0 * uiSizeMultiplier,
      PokerLayoutMode.wide => 3.0 * uiSizeMultiplier,
    };
    final totalGap = switch (layoutMode) {
      PokerLayoutMode.compactPortrait => 18.0 * uiSizeMultiplier,
      PokerLayoutMode.compactLandscape => 16.0 * uiSizeMultiplier,
      PokerLayoutMode.standard => 14.0 * uiSizeMultiplier,
      PokerLayoutMode.wide => 14.0 * uiSizeMultiplier,
    };
    return PokerPotSpec(
      stackLift: stackLift,
      totalGap: totalGap,
      potLabelFontSize: 13.0 * uiSizeMultiplier,
      potLabelBlur: 6.0 * uiSizeMultiplier,
      betStackChipSize: 18.0 * uiSizeMultiplier,
      betStackLabelFontSize: 12.0 * uiSizeMultiplier,
      betStackLabelGap: 4.0 * uiSizeMultiplier,
      betStackLabelBlur: 4.0 * uiSizeMultiplier,
      potPileChipSize: 20.0 * uiSizeMultiplier,
    );
  }
}

PokerUiSpec _buildPokerUiSpec({
  required PokerUiSettings settings,
  required PokerBreakpoint breakpoint,
  required PokerLayoutMode layoutMode,
  required TableThemeConfig tableTheme,
  required CardColorTheme cardTheme,
  required double cardSizeMultiplier,
  required double uiSizeMultiplier,
}) {
  return PokerUiSpec(
    settings: settings,
    breakpoint: breakpoint,
    layoutMode: layoutMode,
    tableTheme: tableTheme,
    cardTheme: cardTheme,
    cardSizeMultiplier: cardSizeMultiplier,
    uiSizeMultiplier: uiSizeMultiplier,
    spacingScale: uiSizeMultiplier * _layoutTightnessFor(layoutMode),
    textScale: uiSizeMultiplier * _textScaleFactorFor(breakpoint),
    iconScale: uiSizeMultiplier,
  );
}

double _layoutTightnessFor(PokerLayoutMode layoutMode) {
  switch (layoutMode) {
    case PokerLayoutMode.compactPortrait:
      return 0.94;
    case PokerLayoutMode.compactLandscape:
      return 0.97;
    case PokerLayoutMode.standard:
      return 1.0;
    case PokerLayoutMode.wide:
      return 1.03;
  }
}

double _textScaleFactorFor(PokerBreakpoint breakpoint) {
  switch (breakpoint) {
    case PokerBreakpoint.compact:
      return 0.96;
    case PokerBreakpoint.regular:
      return 0.98;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 1.0;
  }
}

double _cardScaleMultiplier(PokerScalePreset preset) {
  switch (preset) {
    case PokerScalePreset.xs:
      return 0.6;
    case PokerScalePreset.small:
      return 0.8;
    case PokerScalePreset.large:
      return 1.2;
    case PokerScalePreset.xl:
      return 1.4;
    case PokerScalePreset.medium:
      return 1.0;
  }
}

double _uiScaleMultiplier(PokerScalePreset preset) {
  switch (preset) {
    case PokerScalePreset.xs:
      return 0.7;
    case PokerScalePreset.small:
      return 0.85;
    case PokerScalePreset.large:
      return 1.15;
    case PokerScalePreset.xl:
      return 1.3;
    case PokerScalePreset.medium:
      return 1.0;
  }
}

String _scaleKeyForMultiplier(double multiplier, {required bool isCardScale}) {
  if ((multiplier - (isCardScale ? 0.6 : 0.7)).abs() < 0.001) return 'xs';
  if ((multiplier - (isCardScale ? 0.8 : 0.85)).abs() < 0.001) return 'small';
  if ((multiplier - (isCardScale ? 1.0 : 1.0)).abs() < 0.001) {
    return 'medium';
  }
  if ((multiplier - (isCardScale ? 1.2 : 1.15)).abs() < 0.001) return 'large';
  if ((multiplier - (isCardScale ? 1.4 : 1.3)).abs() < 0.001) return 'xl';
  return 'medium';
}

double cardSizeMultiplierFromKey(String key) {
  return _cardScaleMultiplier(PokerScalePreset.fromKey(key));
}

double uiSizeMultiplierFromKey(String key) {
  return _uiScaleMultiplier(PokerScalePreset.fromKey(key));
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
    final spec = PokerUiSpec.fromContext(context);
    return PokerThemeConfig.fromSpec(spec);
  }

  factory PokerThemeConfig.fromSpec(PokerUiSpec spec) {
    return PokerThemeConfig(
      tableTheme: spec.tableTheme,
      cardTheme: spec.cardTheme,
      cardSizeMultiplier: spec.cardSizeMultiplier,
      uiSizeMultiplier: spec.uiSizeMultiplier,
      showTableLogo: spec.settings.showTableLogo,
      logoPosition: spec.settings.logoPosition,
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
