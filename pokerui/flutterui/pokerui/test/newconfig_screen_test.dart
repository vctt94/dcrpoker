import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/models/newconfig.dart';
import 'package:pokerui/screens/newconfig.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<void> pumpSettingsScreen(
    WidgetTester tester, {
    required Size size,
  }) async {
    tester.view.devicePixelRatio = 1.0;
    tester.view.physicalSize = size;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    final model = NewConfigModel(
      const [],
      initialDataDir: '/tmp/pokerui-test',
      initialGrpcCertPath: '/tmp/pokerui-test/server.cert',
    )
      ..serverAddr = '127.0.0.1:12345'
      ..address = 'deadbeef'
      ..debugLevel = 'info';

    await tester.pumpWidget(
      MaterialApp(
        home: NewConfigScreen(
          model: model,
          onConfigSaved: () async {},
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('settings screen defaults to ui tab with controls above preview',
      (tester) async {
    await pumpSettingsScreen(tester, size: const Size(1440, 1200));

    expect(find.byKey(const Key('settings-section-ui')), findsOneWidget);
    expect(find.byKey(const Key('settings-save-button')), findsOneWidget);
    expect(find.byKey(const Key('settings-ui-layout')), findsOneWidget);
    expect(find.byKey(const Key('settings-ui-controls-row')), findsOneWidget);
    expect(find.text('Table Theme'), findsOneWidget);
    expect(find.byKey(const Key('settings-poker-preview')), findsOneWidget);

    final controlsTop = tester.getTopLeft(find.text('Table Theme')).dy;
    final previewTop =
        tester.getTopLeft(find.byKey(const Key('settings-poker-preview'))).dy;
    expect(previewTop, greaterThan(controlsTop));
  });

  testWidgets('settings menu switches between ui and general sections',
      (tester) async {
    await pumpSettingsScreen(tester, size: const Size(900, 1200));

    expect(find.byKey(const Key('settings-ui-layout')), findsOneWidget);
    expect(find.text('Table Theme'), findsOneWidget);

    await tester.tap(find.byKey(const Key('settings-section-general')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('settings-general-layout')), findsOneWidget);
    expect(find.text('Connection'), findsOneWidget);
    expect(find.text('Identity'), findsOneWidget);
    expect(find.text('Storage'), findsOneWidget);
    expect(find.text('Server Address'), findsOneWidget);
    expect(find.text('Debug Level'), findsOneWidget);
    expect(find.text('Enable Sounds'), findsOneWidget);
    expect(find.text('Table Theme'), findsNothing);
  });

  testWidgets('save from ui tab blocks when server address is missing',
      (tester) async {
    tester.view.devicePixelRatio = 1.0;
    tester.view.physicalSize = const Size(900, 1200);
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    final model = NewConfigModel(
      const [],
      initialDataDir: '/tmp/pokerui-test',
      initialGrpcCertPath: '/tmp/pokerui-test/server.cert',
    )
      ..serverAddr = ''
      ..address = 'deadbeef'
      ..debugLevel = 'info';

    await tester.pumpWidget(
      MaterialApp(
        home: NewConfigScreen(
          model: model,
          onConfigSaved: () async {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('settings-ui-layout')), findsOneWidget);

    final saveButton = find.byKey(const Key('settings-save-button'));
    await tester.ensureVisible(saveButton);
    await tester.tap(saveButton);
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('settings-general-layout')), findsOneWidget);
    expect(find.text('Server address is required'), findsOneWidget);
    expect(find.text('Server Address'), findsOneWidget);
  });
}
