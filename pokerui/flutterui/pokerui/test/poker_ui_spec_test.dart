import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table_theme.dart';

void main() {
  test('resolved ui spec preserves theme and setting intent', () {
    const settings = PokerUiSettings(
      tableThemeKey: 'decred',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.large,
      densityScale: PokerScalePreset.small,
      showTableLogo: true,
      logoPosition: 'left',
    );

    final spec = PokerUiSpec.fromSettings(
      settings,
      viewportSize: const Size(1024, 768),
    );

    expect(spec.tableTheme, equals(TableThemeConfig.decred));
    expect(spec.cardTheme, equals(CardColorTheme.standard));
    expect(spec.breakpoint, equals(PokerBreakpoint.wide));
    expect(spec.layoutMode, equals(PokerLayoutMode.standard));
    expect(spec.cardSizeMultiplier, equals(1.2));
    expect(spec.uiSizeMultiplier, equals(0.85));
    expect(spec.settings.logoPosition, equals('left'));
  });

  test('card-driven tokens scale consistently across surfaces', () {
    const smallSettings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.small,
      densityScale: PokerScalePreset.medium,
      showTableLogo: false,
      logoPosition: 'center',
    );
    const largeSettings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.xl,
      densityScale: PokerScalePreset.medium,
      showTableLogo: false,
      logoPosition: 'center',
    );

    final small = PokerUiSpec.fromSettings(
      smallSettings,
      viewportSize: const Size(390, 844),
    );
    final large = PokerUiSpec.fromSettings(
      largeSettings,
      viewportSize: const Size(390, 844),
    );

    expect(large.heroDockCardSize.width,
        greaterThan(small.heroDockCardSize.width));
    expect(
      large.showdownBoardCardSize().width,
      greaterThan(small.showdownBoardCardSize().width),
    );
    expect(
      large.gameEndedPreviewCardSize().width,
      greaterThan(small.gameEndedPreviewCardSize().width),
    );
  });

  test('density-driven tokens respect compact layouts', () {
    const settings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'decred',
      cardScale: PokerScalePreset.medium,
      densityScale: PokerScalePreset.xl,
      showTableLogo: true,
      logoPosition: 'center',
    );

    final compact = PokerUiSpec.fromSettings(
      settings,
      viewportSize: const Size(390, 844),
    );
    final desktop = PokerUiSpec.fromSettings(
      settings,
      viewportSize: const Size(1440, 900),
    );

    expect(compact.layoutMode, equals(PokerLayoutMode.compactPortrait));
    expect(desktop.layoutMode, equals(PokerLayoutMode.wide));
    expect(compact.spacingScale, lessThan(desktop.spacingScale));
    expect(compact.textScale, lessThanOrEqualTo(desktop.textScale));
  });

  test('seat tokens scale between hero and compact opponents', () {
    const settings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.large,
      densityScale: PokerScalePreset.medium,
      showTableLogo: false,
      logoPosition: 'center',
    );

    final spec = PokerUiSpec.fromSettings(
      settings,
      viewportSize: const Size(390, 844),
    );
    final heroSeat = spec.seatSpec(isHeroSeat: true);
    final opponentSeat = spec.seatSpec(isHeroSeat: false);

    expect(heroSeat.radius, greaterThan(opponentSeat.radius));
    expect(heroSeat.railCardWidth, greaterThan(opponentSeat.railCardWidth));
    expect(heroSeat.plateWidth, greaterThan(opponentSeat.plateWidth));
  });

  test('pot tokens scale with density', () {
    const compactSettings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.medium,
      densityScale: PokerScalePreset.small,
      showTableLogo: false,
      logoPosition: 'center',
    );
    const roomySettings = PokerUiSettings(
      tableThemeKey: 'classic',
      cardThemeKey: 'standard',
      cardScale: PokerScalePreset.medium,
      densityScale: PokerScalePreset.xl,
      showTableLogo: false,
      logoPosition: 'center',
    );

    final compact = PokerUiSpec.fromSettings(
      compactSettings,
      viewportSize: const Size(1024, 768),
    ).potSpec;
    final roomy = PokerUiSpec.fromSettings(
      roomySettings,
      viewportSize: const Size(1024, 768),
    ).potSpec;

    expect(roomy.potLabelFontSize, greaterThan(compact.potLabelFontSize));
    expect(roomy.betStackChipSize, greaterThan(compact.betStackChipSize));
    expect(roomy.potPileChipSize, greaterThan(compact.potPileChipSize));
  });
}
