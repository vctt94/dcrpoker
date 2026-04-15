import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/theme/poker_theme.dart';

void main() {
  test('shared button themes use a click cursor when enabled', () {
    final theme = buildPokerTheme();

    expect(
      theme.elevatedButtonTheme.style?.mouseCursor?.resolve(<WidgetState>{}),
      SystemMouseCursors.click,
    );
    expect(
      theme.elevatedButtonTheme.style?.mouseCursor?.resolve(<WidgetState>{
        WidgetState.disabled,
      }),
      SystemMouseCursors.basic,
    );
    expect(
      theme.outlinedButtonTheme.style?.mouseCursor?.resolve(<WidgetState>{}),
      SystemMouseCursors.click,
    );
    expect(
      theme.textButtonTheme.style?.mouseCursor?.resolve(<WidgetState>{}),
      SystemMouseCursors.click,
    );
    expect(
      theme.iconButtonTheme.style?.mouseCursor?.resolve(<WidgetState>{}),
      SystemMouseCursors.click,
    );
  });

  test('shared button themes expose distinct hover feedback', () {
    final theme = buildPokerTheme();
    final elevatedStyle = theme.elevatedButtonTheme.style;
    final outlinedStyle = theme.outlinedButtonTheme.style;

    expect(
      elevatedStyle?.backgroundColor
          ?.resolve(<WidgetState>{WidgetState.hovered}),
      isNot(
        elevatedStyle?.backgroundColor?.resolve(<WidgetState>{}),
      ),
    );
    expect(
      outlinedStyle?.backgroundColor
          ?.resolve(<WidgetState>{WidgetState.hovered}),
      isNot(
        outlinedStyle?.backgroundColor?.resolve(<WidgetState>{}),
      ),
    );
  });
}
