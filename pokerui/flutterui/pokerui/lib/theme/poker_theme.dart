import 'package:flutter/material.dart';
import 'colors.dart';
import 'typography.dart';

/// Builds the unified poker dark theme.
ThemeData buildPokerTheme() {
  final buttonMouseCursor =
      WidgetStateProperty.resolveWith<MouseCursor>((states) {
    if (states.contains(WidgetState.disabled)) {
      return SystemMouseCursors.basic;
    }
    return SystemMouseCursors.click;
  });
  final buttonOverlay = WidgetStateProperty.resolveWith<Color?>((states) {
    if (states.contains(WidgetState.pressed)) {
      return Colors.white.withValues(alpha: 0.14);
    }
    if (states.contains(WidgetState.hovered)) {
      return Colors.white.withValues(alpha: 0.08);
    }
    if (states.contains(WidgetState.focused)) {
      return PokerColors.primary.withValues(alpha: 0.18);
    }
    return null;
  });

  return ThemeData.dark().copyWith(
    scaffoldBackgroundColor: PokerColors.screenBg,
    primaryColor: PokerColors.primary,
    hoverColor: PokerColors.primary.withValues(alpha: 0.08),
    splashColor: PokerColors.primary.withValues(alpha: 0.14),
    highlightColor: PokerColors.primary.withValues(alpha: 0.1),
    colorScheme: const ColorScheme.dark(
      primary: PokerColors.primary,
      secondary: PokerColors.accent,
      surface: PokerColors.surface,
      error: PokerColors.danger,
      onPrimary: Colors.white,
      onSecondary: Colors.white,
      onSurface: PokerColors.textPrimary,
      onError: Colors.white,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: PokerColors.surfaceDim,
      foregroundColor: PokerColors.textPrimary,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: PokerTypography.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: PokerColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: PokerColors.borderSubtle, width: 1),
      ),
    ),
    drawerTheme: const DrawerThemeData(
      backgroundColor: PokerColors.surfaceDim,
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: PokerColors.surface,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      titleTextStyle: PokerTypography.headlineMedium,
      contentTextStyle: PokerTypography.bodyMedium,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: PokerColors.surfaceDim,
      labelStyle: PokerTypography.bodySmall,
      hintStyle:
          PokerTypography.bodySmall.copyWith(color: PokerColors.textMuted),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: const BorderSide(color: PokerColors.borderSubtle),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: const BorderSide(color: PokerColors.borderSubtle),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: const BorderSide(color: PokerColors.primary, width: 1.5),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ButtonStyle(
        mouseCursor: buttonMouseCursor,
        backgroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return PokerColors.surfaceBright;
          }
          if (states.contains(WidgetState.pressed)) {
            return const Color(0xFF215FD8);
          }
          if (states.contains(WidgetState.hovered)) {
            return const Color(0xFF3A7DFF);
          }
          return PokerColors.primary;
        }),
        foregroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return PokerColors.textMuted;
          }
          return Colors.white;
        }),
        overlayColor: buttonOverlay,
        elevation: WidgetStateProperty.resolveWith<double?>((states) {
          if (states.contains(WidgetState.disabled) ||
              states.contains(WidgetState.pressed)) {
            return 0;
          }
          if (states.contains(WidgetState.hovered)) {
            return 2;
          }
          return 0;
        }),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        ),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
        textStyle: const WidgetStatePropertyAll(PokerTypography.labelLarge),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: ButtonStyle(
        mouseCursor: buttonMouseCursor,
        overlayColor: buttonOverlay,
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        ),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
        textStyle: const WidgetStatePropertyAll(PokerTypography.labelLarge),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        mouseCursor: buttonMouseCursor,
        foregroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return PokerColors.textMuted;
          }
          if (states.contains(WidgetState.hovered)) {
            return Colors.white;
          }
          return PokerColors.primary;
        }),
        backgroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.pressed)) {
            return PokerColors.primary.withValues(alpha: 0.18);
          }
          if (states.contains(WidgetState.hovered)) {
            return PokerColors.primary.withValues(alpha: 0.12);
          }
          return Colors.transparent;
        }),
        side: WidgetStateProperty.resolveWith<BorderSide?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return const BorderSide(color: PokerColors.borderSubtle);
          }
          if (states.contains(WidgetState.hovered)) {
            return const BorderSide(color: PokerColors.borderBright);
          }
          return const BorderSide(color: PokerColors.primary);
        }),
        overlayColor: buttonOverlay,
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        ),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
        textStyle: const WidgetStatePropertyAll(PokerTypography.labelLarge),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: ButtonStyle(
        mouseCursor: buttonMouseCursor,
        foregroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return PokerColors.textMuted;
          }
          if (states.contains(WidgetState.hovered)) {
            return PokerColors.textPrimary;
          }
          return PokerColors.textSecondary;
        }),
        overlayColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.pressed)) {
            return PokerColors.primary.withValues(alpha: 0.14);
          }
          if (states.contains(WidgetState.hovered)) {
            return PokerColors.primary.withValues(alpha: 0.08);
          }
          return null;
        }),
        textStyle: const WidgetStatePropertyAll(PokerTypography.labelLarge),
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: ButtonStyle(
        mouseCursor: buttonMouseCursor,
        foregroundColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.disabled)) {
            return PokerColors.textMuted;
          }
          if (states.contains(WidgetState.hovered)) {
            return PokerColors.textPrimary;
          }
          return PokerColors.textSecondary;
        }),
        overlayColor: WidgetStateProperty.resolveWith<Color?>((states) {
          if (states.contains(WidgetState.pressed)) {
            return PokerColors.primary.withValues(alpha: 0.16);
          }
          if (states.contains(WidgetState.hovered)) {
            return PokerColors.primary.withValues(alpha: 0.1);
          }
          return null;
        }),
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: PokerColors.surfaceBright,
      labelStyle: PokerTypography.labelSmall,
      side: const BorderSide(color: PokerColors.borderSubtle),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
    ),
    dividerTheme: const DividerThemeData(
      color: PokerColors.borderSubtle,
      thickness: 1,
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: PokerColors.surfaceBright,
      contentTextStyle: PokerTypography.bodyMedium,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      behavior: SnackBarBehavior.floating,
    ),
    tooltipTheme: TooltipThemeData(
      decoration: BoxDecoration(
        color: PokerColors.surfaceBright,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      textStyle:
          PokerTypography.bodySmall.copyWith(color: PokerColors.textPrimary),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(
      color: PokerColors.primary,
    ),
    textTheme: const TextTheme(
      displayLarge: PokerTypography.displayLarge,
      headlineLarge: PokerTypography.headlineLarge,
      headlineMedium: PokerTypography.headlineMedium,
      titleLarge: PokerTypography.titleLarge,
      titleMedium: PokerTypography.titleMedium,
      titleSmall: PokerTypography.titleSmall,
      bodyLarge: PokerTypography.bodyLarge,
      bodyMedium: PokerTypography.bodyMedium,
      bodySmall: PokerTypography.bodySmall,
      labelLarge: PokerTypography.labelLarge,
      labelSmall: PokerTypography.labelSmall,
    ),
  );
}
