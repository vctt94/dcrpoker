import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:path/path.dart' as path;
import 'package:provider/provider.dart';

import 'config.dart';
import 'components/poker/responsive.dart';

double _asDouble(dynamic value, double fallback) {
  if (value is num) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? fallback;
  return fallback;
}

Map<String, dynamic> _asMap(dynamic value) {
  if (value is Map<String, dynamic>) return value;
  if (value is Map) return value.map((k, v) => MapEntry('$k', v));
  return const <String, dynamic>{};
}

Map<String, double> _mergeDoubleMap(
  Map<String, dynamic> json,
  Map<String, double> defaults,
) {
  final merged = <String, double>{...defaults};
  for (final entry in json.entries) {
    merged[entry.key] = _asDouble(entry.value, merged[entry.key] ?? 1.0);
  }
  return merged;
}

bool _mapsMatchDefaults(
  Map<String, double> actual,
  Map<String, double> expected,
) {
  if (actual.length != expected.length) return false;
  for (final entry in expected.entries) {
    final actualValue = actual[entry.key];
    if (actualValue == null || (actualValue - entry.value).abs() > 0.001) {
      return false;
    }
  }
  return true;
}

const Map<String, double> _legacyCardSizePresetsV1 = {
  'xs': 0.6,
  'small': 0.8,
  'medium': 1.0,
  'large': 1.2,
  'xl': 1.4,
};

const Map<String, double> _cardSizePresetsV2 = {
  'xs': 0.8,
  'small': 1.0,
  'medium': 1.2,
  'large': 1.4,
  'xl': 1.6,
};

const Map<String, double> _legacyUiSizePresetsV1 = {
  'xs': 0.7,
  'small': 0.85,
  'medium': 1.0,
  'large': 1.15,
  'xl': 1.3,
};

const Map<String, double> _uiSizePresetsV2 = {
  'xs': 0.85,
  'small': 1.0,
  'medium': 1.15,
  'large': 1.3,
  'xl': 1.45,
};

class PokerUiScaleSection {
  const PokerUiScaleSection({
    required this.tableViewportWeight,
    required this.cardsViewportWeight,
    required this.chromeViewportWeight,
    required this.textViewportWeight,
    required this.spacingViewportWeight,
    required this.communityCardWeight,
    required this.heroCardWeight,
    required this.seatCardWeight,
    required this.cardGlyphWeight,
    required this.cornerIndexWeight,
    required this.centerPipWeight,
    required this.overlayTextWeight,
    required this.actionTextWeight,
    required this.logoWeight,
  });

  final double tableViewportWeight;
  final double cardsViewportWeight;
  final double chromeViewportWeight;
  final double textViewportWeight;
  final double spacingViewportWeight;
  final double communityCardWeight;
  final double heroCardWeight;
  final double seatCardWeight;
  final double cardGlyphWeight;
  final double cornerIndexWeight;
  final double centerPipWeight;
  final double overlayTextWeight;
  final double actionTextWeight;
  final double logoWeight;

  factory PokerUiScaleSection.defaults() => const PokerUiScaleSection(
        tableViewportWeight: 1.0,
        cardsViewportWeight: 0.58,
        chromeViewportWeight: 0.72,
        textViewportWeight: 0.68,
        spacingViewportWeight: 0.5,
        communityCardWeight: 1.08,
        heroCardWeight: 1.0,
        seatCardWeight: 0.96,
        cardGlyphWeight: 1.05,
        cornerIndexWeight: 1.1,
        centerPipWeight: 1.0,
        overlayTextWeight: 1.0,
        actionTextWeight: 1.0,
        logoWeight: 0.95,
      );

  factory PokerUiScaleSection.fromJson(Map<String, dynamic> json) {
    final defaults = PokerUiScaleSection.defaults();
    return PokerUiScaleSection(
      tableViewportWeight: _asDouble(
          json['table_viewport_weight'], defaults.tableViewportWeight),
      cardsViewportWeight: _asDouble(
          json['cards_viewport_weight'], defaults.cardsViewportWeight),
      chromeViewportWeight: _asDouble(
          json['chrome_viewport_weight'], defaults.chromeViewportWeight),
      textViewportWeight:
          _asDouble(json['text_viewport_weight'], defaults.textViewportWeight),
      spacingViewportWeight: _asDouble(
          json['spacing_viewport_weight'], defaults.spacingViewportWeight),
      communityCardWeight: _asDouble(
          json['community_card_weight'], defaults.communityCardWeight),
      heroCardWeight:
          _asDouble(json['hero_card_weight'], defaults.heroCardWeight),
      seatCardWeight:
          _asDouble(json['seat_card_weight'], defaults.seatCardWeight),
      cardGlyphWeight:
          _asDouble(json['card_glyph_weight'], defaults.cardGlyphWeight),
      cornerIndexWeight:
          _asDouble(json['corner_index_weight'], defaults.cornerIndexWeight),
      centerPipWeight:
          _asDouble(json['center_pip_weight'], defaults.centerPipWeight),
      overlayTextWeight:
          _asDouble(json['overlay_text_weight'], defaults.overlayTextWeight),
      actionTextWeight:
          _asDouble(json['action_text_weight'], defaults.actionTextWeight),
      logoWeight: _asDouble(json['logo_weight'], defaults.logoWeight),
    );
  }

  Map<String, dynamic> toJson() => {
        'table_viewport_weight': tableViewportWeight,
        'cards_viewport_weight': cardsViewportWeight,
        'chrome_viewport_weight': chromeViewportWeight,
        'text_viewport_weight': textViewportWeight,
        'spacing_viewport_weight': spacingViewportWeight,
        'community_card_weight': communityCardWeight,
        'hero_card_weight': heroCardWeight,
        'seat_card_weight': seatCardWeight,
        'card_glyph_weight': cardGlyphWeight,
        'corner_index_weight': cornerIndexWeight,
        'center_pip_weight': centerPipWeight,
        'overlay_text_weight': overlayTextWeight,
        'action_text_weight': actionTextWeight,
        'logo_weight': logoWeight,
      };
}

class PokerUiLimits {
  const PokerUiLimits({
    required this.communityCardMinWidth,
    required this.communityCardMaxWidth,
    required this.heroCardMinWidth,
    required this.heroCardMaxWidth,
    required this.opponentCardMinWidth,
    required this.opponentCardMaxWidth,
  });

  final double communityCardMinWidth;
  final double communityCardMaxWidth;
  final double heroCardMinWidth;
  final double heroCardMaxWidth;
  final double opponentCardMinWidth;
  final double opponentCardMaxWidth;

  factory PokerUiLimits.defaults() => const PokerUiLimits(
        communityCardMinWidth: 24.0,
        communityCardMaxWidth: 96.0,
        heroCardMinWidth: 24.0,
        heroCardMaxWidth: 84.0,
        opponentCardMinWidth: 22.0,
        opponentCardMaxWidth: 72.0,
      );

  factory PokerUiLimits.fromJson(Map<String, dynamic> json) {
    final defaults = PokerUiLimits.defaults();
    final community = _asMap(json['community_card_width']);
    final hero = _asMap(json['hero_card_width']);
    final opponent = _asMap(json['opponent_card_width']);
    return PokerUiLimits(
      communityCardMinWidth:
          _asDouble(community['min'], defaults.communityCardMinWidth),
      communityCardMaxWidth:
          _asDouble(community['max'], defaults.communityCardMaxWidth),
      heroCardMinWidth: _asDouble(hero['min'], defaults.heroCardMinWidth),
      heroCardMaxWidth: _asDouble(hero['max'], defaults.heroCardMaxWidth),
      opponentCardMinWidth:
          _asDouble(opponent['min'], defaults.opponentCardMinWidth),
      opponentCardMaxWidth:
          _asDouble(opponent['max'], defaults.opponentCardMaxWidth),
    );
  }

  Map<String, dynamic> toJson() => {
        'community_card_width': {
          'min': communityCardMinWidth,
          'max': communityCardMaxWidth,
        },
        'hero_card_width': {
          'min': heroCardMinWidth,
          'max': heroCardMaxWidth,
        },
        'opponent_card_width': {
          'min': opponentCardMinWidth,
          'max': opponentCardMaxWidth,
        },
      };
}

class PokerUiConfig {
  const PokerUiConfig({
    required this.version,
    required this.activeTableSize,
    required this.viewportBaseScales,
    required this.tableSizePresets,
    required this.cardSizePresets,
    required this.uiSizePresets,
    required this.scales,
    required this.limits,
  });

  final int version;
  final String activeTableSize;
  final Map<String, double> viewportBaseScales;
  final Map<String, double> tableSizePresets;
  final Map<String, double> cardSizePresets;
  final Map<String, double> uiSizePresets;
  final PokerUiScaleSection scales;
  final PokerUiLimits limits;

  factory PokerUiConfig.defaults() => PokerUiConfig(
        version: 2,
        activeTableSize: 'medium',
        viewportBaseScales: const {
          'compact': 0.92,
          'regular': 1.0,
          'expanded': 1.04,
          'wide': 1.12,
        },
        tableSizePresets: const {
          'small': 0.92,
          'medium': 1.0,
          'large': 1.08,
        },
        cardSizePresets: _cardSizePresetsV2,
        uiSizePresets: _uiSizePresetsV2,
        scales: PokerUiScaleSection.defaults(),
        limits: PokerUiLimits.defaults(),
      );

  factory PokerUiConfig.fromJson(Map<String, dynamic> json) {
    final defaults = PokerUiConfig.defaults();
    final rawVersion = (json['version'] as num?)?.toInt() ?? 1;
    final viewportClasses = _asMap(json['viewport_classes']);
    final viewportBase = <String, double>{...defaults.viewportBaseScales};
    for (final entry in viewportClasses.entries) {
      final entryMap = _asMap(entry.value);
      viewportBase[entry.key] =
          _asDouble(entryMap['base_scale'], viewportBase[entry.key] ?? 1.0);
    }

    final active = _asMap(json['active']);
    final presets = _asMap(json['presets']);
    final tablePresets = _mergeDoubleMap(
        _asMap(presets['table_size']), defaults.tableSizePresets);
    var cardPresets =
        _mergeDoubleMap(_asMap(presets['card_size']), defaults.cardSizePresets);
    var uiPresets =
        _mergeDoubleMap(_asMap(presets['ui_size']), defaults.uiSizePresets);

    if (rawVersion < 2) {
      if (_mapsMatchDefaults(cardPresets, _legacyCardSizePresetsV1)) {
        cardPresets = Map<String, double>.from(_cardSizePresetsV2);
      }
      if (_mapsMatchDefaults(uiPresets, _legacyUiSizePresetsV1)) {
        uiPresets = Map<String, double>.from(_uiSizePresetsV2);
      }
    }

    return PokerUiConfig(
      version: rawVersion < defaults.version ? defaults.version : rawVersion,
      activeTableSize:
          (active['table_size'] ?? defaults.activeTableSize).toString(),
      viewportBaseScales: viewportBase,
      tableSizePresets: tablePresets,
      cardSizePresets: cardPresets,
      uiSizePresets: uiPresets,
      scales: PokerUiScaleSection.fromJson(_asMap(json['scales'])),
      limits: PokerUiLimits.fromJson(_asMap(json['limits'])),
    );
  }

  Map<String, dynamic> toJson() => {
        'version': version,
        'active': {
          'table_size': activeTableSize,
        },
        'viewport_classes': {
          for (final entry in viewportBaseScales.entries)
            entry.key: {'base_scale': entry.value},
        },
        'presets': {
          'table_size': tableSizePresets,
          'card_size': cardSizePresets,
          'ui_size': uiSizePresets,
        },
        'scales': scales.toJson(),
        'limits': limits.toJson(),
      };

  String get tableSizeKey => tableSizePresets.containsKey(activeTableSize)
      ? activeTableSize
      : 'medium';

  double tableSizeMultiplier() => tableSizePresets[tableSizeKey] ?? 1.0;

  double cardSizeMultiplierForKey(String key) =>
      cardSizePresets[key.toLowerCase()] ?? cardSizePresets['medium'] ?? 1.0;

  double uiSizeMultiplierForKey(String key) =>
      uiSizePresets[key.toLowerCase()] ?? uiSizePresets['medium'] ?? 1.0;

  double viewportBaseScaleForBreakpoint(PokerBreakpoint breakpoint) {
    final key = switch (breakpoint) {
      PokerBreakpoint.compact => 'compact',
      PokerBreakpoint.regular => 'regular',
      PokerBreakpoint.expanded => 'expanded',
      PokerBreakpoint.wide => 'wide',
    };
    return viewportBaseScales[key] ?? 1.0;
  }
}

class PokerUiConfigNotifier extends ChangeNotifier {
  PokerUiConfigNotifier({PokerUiConfig? initial})
      : _value = initial ?? PokerUiConfig.defaults();

  PokerUiConfig _value;
  bool _loaded = false;

  PokerUiConfig get value => _value;
  bool get loaded => _loaded;

  Future<void> load() async {
    _value = await loadPokerUiConfigFromDisk();
    _loaded = true;
    notifyListeners();
  }

  Future<void> reload() => load();

  Future<void> save(PokerUiConfig config) async {
    await savePokerUiConfigToDisk(config);
    _value = config;
    _loaded = true;
    notifyListeners();
  }
}

extension PokerUiConfigContext on BuildContext {
  PokerUiConfig get pokerUiConfig =>
      Provider.of<PokerUiConfigNotifier?>(this)?.value ??
      PokerUiConfig.defaults();
}

Future<String> pokerUiConfigFilePath() async {
  final dataDir = await defaultAppDataDir();
  return path.join(dataDir, 'ui-config.json');
}

Future<PokerUiConfig> loadPokerUiConfigFromDisk() async {
  final file = File(await pokerUiConfigFilePath());
  final defaults = PokerUiConfig.defaults();

  if (!await file.exists()) {
    await file.parent.create(recursive: true);
    await file.writeAsString(
      const JsonEncoder.withIndent('  ').convert(defaults.toJson()),
    );
    return defaults;
  }

  try {
    final raw = await file.readAsString();
    final parsed = jsonDecode(raw);
    if (parsed is Map) {
      return PokerUiConfig.fromJson(parsed.cast<String, dynamic>());
    }
  } catch (_) {
    // Keep defaults on invalid JSON.
  }

  return defaults;
}

Future<void> savePokerUiConfigToDisk(PokerUiConfig config) async {
  final file = File(await pokerUiConfigFilePath());
  await file.parent.create(recursive: true);
  await file.writeAsString(
    const JsonEncoder.withIndent('  ').convert(config.toJson()),
  );
}
