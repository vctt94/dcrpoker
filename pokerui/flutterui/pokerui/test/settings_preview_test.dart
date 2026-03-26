import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/components/poker/settings_preview.dart';
import 'package:pokerui/components/poker/table_theme.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    home: Scaffold(
      body: SizedBox(
        width: 420,
        child: child,
      ),
    ),
  );
}

Widget _wrapWithWidth(Widget child, double width) {
  return MaterialApp(
    home: Scaffold(
      body: SizedBox(
        width: width,
        child: child,
      ),
    ),
  );
}

void main() {
  testWidgets('settings preview renders without config provider',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(
        const SettingsPokerPreview(
          settings: PokerUiSettings(
            tableThemeKey: 'classic',
            cardThemeKey: 'standard',
            cardScale: PokerScalePreset.medium,
            densityScale: PokerScalePreset.medium,
            showTableLogo: true,
            logoPosition: 'center',
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const Key('settings-poker-preview')), findsOneWidget);
    expect(find.byKey(const Key('settings-preview-stage')), findsOneWidget);
    expect(
        find.byKey(const Key('settings-preview-device-phone')), findsOneWidget);
    expect(find.byKey(const Key('settings-preview-device-tablet')),
        findsOneWidget);
    expect(find.byKey(const Key('settings-preview-device-desktop')),
        findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('card scale increases preview card sizes',
      (WidgetTester tester) async {
    Future<Size> renderSize(PokerScalePreset cardScale) async {
      await tester.pumpWidget(
        _wrap(
          SettingsPokerPreview(
            settings: PokerUiSettings(
              tableThemeKey: 'classic',
              cardThemeKey: 'standard',
              cardScale: cardScale,
              densityScale: PokerScalePreset.medium,
              showTableLogo: false,
              logoPosition: 'center',
            ),
          ),
        ),
      );
      await tester.pump();
      return tester
          .getSize(find.byKey(const Key('settings-preview-hero-card-0')));
    }

    final small = await renderSize(PokerScalePreset.small);
    final xl = await renderSize(PokerScalePreset.xl);

    expect(xl.width, greaterThan(small.width));
    expect(xl.height, greaterThan(small.height));
  });

  testWidgets('ui density increases action button height',
      (WidgetTester tester) async {
    Future<Size> renderSize(PokerScalePreset densityScale) async {
      await tester.pumpWidget(
        _wrap(
          SettingsPokerPreview(
            settings: PokerUiSettings(
              tableThemeKey: 'classic',
              cardThemeKey: 'standard',
              cardScale: PokerScalePreset.medium,
              densityScale: densityScale,
              showTableLogo: false,
              logoPosition: 'center',
            ),
          ),
        ),
      );
      await tester.pump();
      return tester
          .getSize(find.byKey(const Key('settings-preview-action-call')));
    }

    final small = await renderSize(PokerScalePreset.small);
    final xl = await renderSize(PokerScalePreset.xl);

    expect(xl.height, greaterThan(small.height));
  });

  testWidgets('device selector switches preview viewport',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(
        const SettingsPokerPreview(
          settings: PokerUiSettings(
            tableThemeKey: 'classic',
            cardThemeKey: 'standard',
            cardScale: PokerScalePreset.medium,
            densityScale: PokerScalePreset.medium,
            showTableLogo: true,
            logoPosition: 'center',
          ),
        ),
      ),
    );
    await tester.pump();

    await tester.tap(find.byKey(const Key('settings-preview-device-phone')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('settings-preview-viewport-phone')),
        findsOneWidget);
    expect(find.textContaining('Compact portrait'), findsOneWidget);

    await tester.tap(find.byKey(const Key('settings-preview-device-tablet')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('settings-preview-viewport-tablet')),
        findsOneWidget);
    expect(find.textContaining('Standard table'), findsOneWidget);

    await tester.tap(find.byKey(const Key('settings-preview-device-desktop')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('settings-preview-viewport-desktop')),
        findsOneWidget);
    expect(find.textContaining('Wide desktop'), findsOneWidget);
  });

  testWidgets('desktop preview stage expands on wide settings layouts',
      (WidgetTester tester) async {
    tester.view.devicePixelRatio = 1.0;
    tester.view.physicalSize = const Size(1800, 1200);
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    await tester.pumpWidget(
      _wrapWithWidth(
        const SettingsPokerPreview(
          settings: PokerUiSettings(
            tableThemeKey: 'classic',
            cardThemeKey: 'standard',
            cardScale: PokerScalePreset.medium,
            densityScale: PokerScalePreset.medium,
            showTableLogo: true,
            logoPosition: 'center',
          ),
        ),
        1600,
      ),
    );
    await tester.pump();

    await tester.tap(find.byKey(const Key('settings-preview-device-desktop')));
    await tester.pumpAndSettle();

    final stageSize =
        tester.getSize(find.byKey(const Key('settings-preview-stage')));
    expect(stageSize.height, greaterThan(360));
  });
}
