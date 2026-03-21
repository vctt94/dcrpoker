import 'package:flutter/material.dart';
import 'colors.dart';
import 'typography.dart';

/// Builds the unified poker dark theme.
ThemeData buildPokerTheme() {
  return ThemeData.dark().copyWith(
    scaffoldBackgroundColor: PokerColors.screenBg,
    primaryColor: PokerColors.primary,
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
      hintStyle: PokerTypography.bodySmall.copyWith(color: PokerColors.textMuted),
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
      style: ElevatedButton.styleFrom(
        backgroundColor: PokerColors.primary,
        foregroundColor: Colors.white,
        elevation: 0,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        textStyle: PokerTypography.labelLarge,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: PokerColors.primary,
        side: const BorderSide(color: PokerColors.primary),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        textStyle: PokerTypography.labelLarge,
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: PokerColors.textSecondary,
        textStyle: PokerTypography.labelLarge,
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
      textStyle: PokerTypography.bodySmall.copyWith(color: PokerColors.textPrimary),
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
