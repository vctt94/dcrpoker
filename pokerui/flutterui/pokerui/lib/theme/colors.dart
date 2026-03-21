import 'package:flutter/material.dart';

/// Centralized color tokens for the poker UI design system.
/// All colors derive from Decred brand blue (#2970FF) and green (#2ED6A1).
class PokerColors {
  PokerColors._();

  // ── Brand ──
  static const Color primary = Color(0xFF2970FF);
  static const Color accent = Color(0xFF2ED6A1);

  // ── Background hierarchy (darkest → lightest) ──
  static const Color screenBg = Color(0xFF0D1017);
  static const Color surfaceDim = Color(0xFF141822);
  static const Color surface = Color(0xFF1A1F2E);
  static const Color surfaceBright = Color(0xFF242B3D);

  // ── Text ──
  static const Color textPrimary = Color(0xFFF0F2F8);
  static const Color textSecondary = Color(0xFF8B93A8);
  static const Color textMuted = Color(0xFF4A5168);

  // ── Semantic ──
  static const Color danger = Color(0xFFE53935);
  static const Color dangerDark = Color(0xFFD32F2F);
  static const Color warning = Color(0xFFF5A623);
  static const Color success = Color(0xFF2ED6A1);
  static const Color info = Color(0xFF2970FF);

  // ── Game-specific ──
  static const Color turnHighlight = Color(0xFFFFD54F);
  static const Color heroSeat = Color(0xFF2E6DD8);
  static const Color potBorder = Color(0xFF2970FF);
  static const Color feltClassic = Color(0xFF0D4F3C);
  static const Color feltBorderClassic = Color(0xFF8B4513);
  static const Color feltDecred = Color(0xFF091440);

  // ── Action buttons ──
  static const Color foldBtn = Color(0xFFD32F2F);
  static const Color checkBtn = Color(0xFF37474F);
  static const Color betBtn = Color(0xFF2E7D32);

  // ── Overlays ──
  static const Color overlayHeavy = Color(0xCC000000);
  static const Color overlayMedium = Color(0x99000000);
  static const Color overlayLight = Color(0x47000000);
  static const Color overlaySubtle = Color(0x1AFFFFFF);

  // ── Borders ──
  static const Color borderSubtle = Color(0xFF2A3040);
  static const Color borderMedium = Color(0xFF3A4258);
  static const Color borderBright = Color(0xFF4D5878);

  // ── Card surfaces ──
  static const Color cardFace = Color(0xFFF8F8F8);
  static const Color cardBackStart = Color(0xFF1E2235);
  static const Color cardBackEnd = Color(0xFF0E111A);
}
