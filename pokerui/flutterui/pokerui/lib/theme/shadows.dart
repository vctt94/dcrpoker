import 'package:flutter/material.dart';

/// Elevation shadow presets for poker UI surfaces.
class PokerShadows {
  PokerShadows._();

  static List<BoxShadow> get subtle => const [
        BoxShadow(
          color: Color(0x33000000),
          blurRadius: 4,
          spreadRadius: 0,
          offset: Offset(0, 2),
        ),
      ];

  static List<BoxShadow> get card => const [
        BoxShadow(
          color: Color(0x4D000000),
          blurRadius: 8,
          spreadRadius: 1,
          offset: Offset(0, 2),
        ),
      ];

  static List<BoxShadow> get elevated => const [
        BoxShadow(
          color: Color(0x59000000),
          blurRadius: 16,
          spreadRadius: 2,
          offset: Offset(0, 4),
        ),
      ];

  static List<BoxShadow> get overlay => const [
        BoxShadow(
          color: Color(0x80000000),
          blurRadius: 24,
          spreadRadius: 4,
          offset: Offset(0, 8),
        ),
      ];

  static List<BoxShadow> get glow => [
        BoxShadow(
          color: const Color(0xFF2970FF).withOpacity(0.25),
          blurRadius: 12,
          spreadRadius: 2,
        ),
      ];
}
