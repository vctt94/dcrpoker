import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/ui_config.dart';

double _weightedViewportScale(double baseScale, double weight) {
  final scaled = 1.0 + ((baseScale - 1.0) * weight.clamp(0.0, 1.0));
  return scaled < 1.0 ? 1.0 : scaled;
}

class PokerResolvedScale {
  const PokerResolvedScale({
    required this.tableScale,
    required this.spacingScale,
    required this.communityCardScale,
    required this.heroCardScale,
    required this.seatCardScale,
    required this.cardGlyphScale,
    required this.cornerIndexScale,
    required this.centerPipScale,
    required this.tableChromeScale,
    required this.overlayTextScale,
    required this.actionScale,
    required this.logoScale,
    required this.limits,
  });

  final double tableScale;
  final double spacingScale;
  final double communityCardScale;
  final double heroCardScale;
  final double seatCardScale;
  final double cardGlyphScale;
  final double cornerIndexScale;
  final double centerPipScale;
  final double tableChromeScale;
  final double overlayTextScale;
  final double actionScale;
  final double logoScale;
  final PokerUiLimits limits;

  factory PokerResolvedScale.resolve({
    required PokerUiConfig uiConfig,
    required PokerBreakpoint breakpoint,
    required double cardSizeMultiplier,
    required double uiSizeMultiplier,
  }) {
    final viewportBase = uiConfig.viewportBaseScaleForBreakpoint(breakpoint);
    final tableViewportScale = _weightedViewportScale(
        viewportBase, uiConfig.scales.tableViewportWeight);
    final cardViewportScale = _weightedViewportScale(
        viewportBase, uiConfig.scales.cardsViewportWeight);
    final chromeViewportScale = _weightedViewportScale(
        viewportBase, uiConfig.scales.chromeViewportWeight);
    final textViewportScale = _weightedViewportScale(
        viewportBase, uiConfig.scales.textViewportWeight);
    final spacingViewportScale = _weightedViewportScale(
        viewportBase, uiConfig.scales.spacingViewportWeight);

    final tablePreset = uiConfig.tableSizeMultiplier();

    final cardBoxScale = cardSizeMultiplier * cardViewportScale;
    final chromeScale = uiSizeMultiplier * chromeViewportScale;

    return PokerResolvedScale(
      tableScale: tablePreset * tableViewportScale,
      spacingScale: uiSizeMultiplier * spacingViewportScale,
      communityCardScale: cardBoxScale * uiConfig.scales.communityCardWeight,
      heroCardScale: cardBoxScale * uiConfig.scales.heroCardWeight,
      seatCardScale: cardBoxScale * uiConfig.scales.seatCardWeight,
      cardGlyphScale: cardBoxScale * uiConfig.scales.cardGlyphWeight,
      cornerIndexScale: cardBoxScale *
          uiConfig.scales.cardGlyphWeight *
          uiConfig.scales.cornerIndexWeight,
      centerPipScale: cardBoxScale *
          uiConfig.scales.cardGlyphWeight *
          uiConfig.scales.centerPipWeight,
      tableChromeScale: chromeScale,
      overlayTextScale:
          chromeScale * textViewportScale * uiConfig.scales.overlayTextWeight,
      actionScale:
          chromeScale * textViewportScale * uiConfig.scales.actionTextWeight,
      logoScale: chromeScale * uiConfig.scales.logoWeight,
      limits: uiConfig.limits,
    );
  }

  factory PokerResolvedScale.fromContext(
    BuildContext context, {
    Size? size,
  }) {
    final uiConfig = context.pokerUiConfig;
    final breakpoint = size != null
        ? PokerBreakpointQuery.fromWidth(size.width)
        : PokerBreakpointQuery.of(context);
    return PokerResolvedScale.resolve(
      uiConfig: uiConfig,
      breakpoint: breakpoint,
      cardSizeMultiplier: uiConfig.cardSizeMultiplierForKey(context.cardSize),
      uiSizeMultiplier: uiConfig.uiSizeMultiplierForKey(context.uiSize),
    );
  }
}

PokerResolvedScale resolvePokerScaleForSize(
  BuildContext context,
  Size size,
) {
  return PokerResolvedScale.fromContext(context, size: size);
}

PokerThemeConfig resolvePokerTheme(BuildContext context) {
  final uiConfig = context.pokerUiConfig;
  return PokerThemeConfig(
    tableTheme: TableThemeConfig.fromKey(context.tableTheme),
    cardTheme: cardColorThemeFromKey(context.cardTheme),
    cardSizeMultiplier: uiConfig.cardSizeMultiplierForKey(context.cardSize),
    uiSizeMultiplier: uiConfig.uiSizeMultiplierForKey(context.uiSize),
    showTableLogo: context.showTableLogo,
    logoPosition: context.logoPosition,
  );
}
