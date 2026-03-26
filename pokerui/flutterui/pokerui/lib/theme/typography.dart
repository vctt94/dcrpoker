import 'package:flutter/material.dart';
import 'colors.dart';

/// Text style presets for the poker UI.
///
/// Uses the platform default font for now. Swap [_fontFamily] to a bundled
/// font (Inter, Outfit, etc.) when assets are available.
class PokerTypography {
  PokerTypography._();

  static const String? _fontFamily = null; // use platform default
  static const String? _monoFamily = null; // use platform default

  // ── Display ──
  static const TextStyle displayLarge = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 32,
    fontWeight: FontWeight.w800,
    color: PokerColors.textPrimary,
    letterSpacing: -0.5,
  );

  // ── Headlines ──
  static const TextStyle headlineLarge = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 24,
    fontWeight: FontWeight.w700,
    color: PokerColors.textPrimary,
    letterSpacing: -0.3,
  );

  static const TextStyle headlineMedium = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 20,
    fontWeight: FontWeight.w700,
    color: PokerColors.textPrimary,
  );

  // ── Titles ──
  static const TextStyle titleLarge = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 18,
    fontWeight: FontWeight.w600,
    color: PokerColors.textPrimary,
  );

  static const TextStyle titleMedium = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 16,
    fontWeight: FontWeight.w600,
    color: PokerColors.textPrimary,
  );

  static const TextStyle titleSmall = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 14,
    fontWeight: FontWeight.w600,
    color: PokerColors.textPrimary,
  );

  // ── Body ──
  static const TextStyle bodyLarge = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 16,
    fontWeight: FontWeight.w400,
    color: PokerColors.textPrimary,
  );

  static const TextStyle bodyMedium = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 14,
    fontWeight: FontWeight.w400,
    color: PokerColors.textPrimary,
  );

  static const TextStyle bodySmall = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 12,
    fontWeight: FontWeight.w400,
    color: PokerColors.textSecondary,
  );

  // ── Labels ──
  static const TextStyle labelLarge = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 14,
    fontWeight: FontWeight.w600,
    color: PokerColors.textPrimary,
    letterSpacing: 0.3,
  );

  static const TextStyle labelSmall = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 11,
    fontWeight: FontWeight.w600,
    color: PokerColors.textSecondary,
    letterSpacing: 0.3,
  );

  // ── Game-specific ──
  static const TextStyle chipCount = TextStyle(
    fontFamily: _monoFamily,
    fontSize: 13,
    fontWeight: FontWeight.w700,
    color: PokerColors.textPrimary,
    letterSpacing: 0.5,
  );

  static const TextStyle potLabel = TextStyle(
    fontFamily: _monoFamily,
    fontSize: 14,
    fontWeight: FontWeight.w700,
    color: PokerColors.textPrimary,
    letterSpacing: 0.3,
  );

  static const TextStyle badgeLabel = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 11,
    fontWeight: FontWeight.w900,
    color: Colors.black,
    letterSpacing: 0.2,
  );

  static const TextStyle playerName = TextStyle(
    fontFamily: _fontFamily,
    fontSize: 12,
    fontWeight: FontWeight.w600,
    color: PokerColors.textPrimary,
  );

  static const TextStyle timerText = TextStyle(
    fontFamily: _monoFamily,
    fontSize: 12,
    fontWeight: FontWeight.w700,
    color: PokerColors.textPrimary,
    letterSpacing: 0.3,
  );

  static const TextStyle cardRank = TextStyle(
    fontSize: 16,
    fontWeight: FontWeight.w900,
    height: 1.0,
  );

  static const TextStyle cardSuit = TextStyle(
    fontSize: 12,
    fontWeight: FontWeight.w700,
    height: 1.0,
  );
}
