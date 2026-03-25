import 'dart:math' as math;

import 'package:flutter/material.dart';

enum PokerLayoutMode {
  compactPortrait,
  compactLandscape,
  standard,
  wide,
}

extension PokerLayoutModeX on PokerLayoutMode {
  bool get isCompact =>
      this == PokerLayoutMode.compactPortrait ||
      this == PokerLayoutMode.compactLandscape;

  bool get isWideDesktop => this == PokerLayoutMode.wide;
}

class PokerSceneLayout {
  const PokerSceneLayout({
    required this.mode,
    required this.screenRect,
    required this.safeRect,
    required this.contentRect,
    required this.tableRect,
    required this.communityRect,
    required this.potRect,
    required this.topSeatBandRect,
    required this.heroDockRect,
    required this.rightRailRect,
    required this.leftRailRect,
    required this.bodyRect,
    required this.boardRect,
  });

  final PokerLayoutMode mode;
  final Rect screenRect;
  final Rect safeRect;
  final Rect contentRect;
  final Rect tableRect;
  final Rect communityRect;
  final Rect potRect;
  final Rect topSeatBandRect;
  final Rect heroDockRect;
  final Rect rightRailRect;
  final Rect leftRailRect;
  final Rect bodyRect;
  final Rect boardRect;

  bool get isCompact => mode.isCompact;
  bool get isWide => mode.isWideDesktop;
  bool get isPhonePortrait => mode == PokerLayoutMode.compactPortrait;

  Offset get tableCenter => tableRect.center;

  double get tableRadiusX => tableRect.width / 2;
  double get tableRadiusY => tableRect.height / 2;

  double get tableAspectRatio =>
      tableRect.height == 0 ? 1.0 : tableRect.width / tableRect.height;

  static PokerLayoutMode resolveMode(Size size) {
    final portrait = size.height >= size.width;
    if (portrait && size.width < 720) {
      return PokerLayoutMode.compactPortrait;
    }
    if (!portrait && (size.width < 960 || size.height < 720)) {
      return PokerLayoutMode.compactLandscape;
    }
    if (size.width >= 1400 && size.height >= 820) {
      return PokerLayoutMode.wide;
    }
    return PokerLayoutMode.standard;
  }

  static PokerSceneLayout resolve(
    Size size, {
    EdgeInsets safePadding = EdgeInsets.zero,
  }) {
    final mode = resolveMode(size);
    final screenRect = Offset.zero & size;
    final safeRect = Rect.fromLTWH(
      safePadding.left,
      safePadding.top,
      math.max(0, size.width - safePadding.horizontal),
      math.max(0, size.height - safePadding.vertical),
    );
    final outerPad = (safeRect.shortestSide * 0.024).clamp(10.0, 22.0);
    final contentRect = safeRect.deflate(outerPad);
    final gap = (contentRect.shortestSide * 0.02).clamp(8.0, 18.0);

    double topBandHeight;
    double heroDockHeight;
    double railWidth;
    switch (mode) {
      case PokerLayoutMode.compactPortrait:
        topBandHeight = (contentRect.height * 0.12).clamp(72.0, 98.0);
        heroDockHeight = (contentRect.height * 0.2).clamp(156.0, 188.0);
        railWidth = 0;
        break;
      case PokerLayoutMode.compactLandscape:
        topBandHeight = (contentRect.height * 0.18).clamp(90.0, 124.0);
        heroDockHeight = (contentRect.height * 0.21).clamp(136.0, 176.0);
        railWidth = 0;
        break;
      case PokerLayoutMode.standard:
        topBandHeight = (contentRect.height * 0.17).clamp(102.0, 136.0);
        heroDockHeight = (contentRect.height * 0.26).clamp(184.0, 240.0);
        railWidth = 0;
        break;
      case PokerLayoutMode.wide:
        topBandHeight = (contentRect.height * 0.18).clamp(120.0, 164.0);
        heroDockHeight = (contentRect.height * 0.2).clamp(164.0, 220.0);
        railWidth = (contentRect.width * 0.19).clamp(220.0, 300.0);
        break;
    }

    final maxRail = math.max(0.0, contentRect.width - 220.0);
    railWidth = math.min(railWidth, maxRail);

    final reservedHeight = topBandHeight + heroDockHeight + gap * 2;
    final minBodyHeight = mode.isCompact ? 24.0 : 40.0;
    final maxReservedHeight = math.max(0.0, contentRect.height - minBodyHeight);
    if (reservedHeight > maxReservedHeight && reservedHeight > 0) {
      final scale = maxReservedHeight / reservedHeight;
      topBandHeight *= scale;
      heroDockHeight *= scale;
    }

    final heroDockRect = Rect.fromLTWH(
      contentRect.left,
      contentRect.bottom - heroDockHeight,
      contentRect.width,
      heroDockHeight,
    );
    final bodyRect = Rect.fromLTRB(
      contentRect.left,
      contentRect.top + topBandHeight + gap,
      contentRect.right,
      heroDockRect.top - gap,
    );

    final rightRailRect = railWidth <= 0
        ? Rect.fromLTWH(bodyRect.right, bodyRect.top, 0, bodyRect.height)
        : Rect.fromLTWH(
            bodyRect.right - railWidth,
            bodyRect.top,
            railWidth,
            bodyRect.height,
          );
    final leftRailRect = Rect.fromLTWH(bodyRect.left, bodyRect.top, 0, 0);
    final tableWidthRight = math.max(bodyRect.left, rightRailRect.left - gap);
    final tableRect = mode == PokerLayoutMode.compactPortrait
        ? Rect.fromLTRB(
            bodyRect.left + gap * 0.25,
            bodyRect.top + bodyRect.height * 0.12,
            tableWidthRight - gap * 0.25,
            bodyRect.bottom - bodyRect.height * 0.08,
          )
        : Rect.fromLTRB(
            bodyRect.left,
            bodyRect.top,
            tableWidthRight,
            bodyRect.bottom,
          );
    final topSeatBandRect = Rect.fromLTRB(
      contentRect.left,
      contentRect.top,
      tableRect.right,
      contentRect.top + topBandHeight,
    );

    final boardWidthFactor = switch (mode) {
      PokerLayoutMode.compactPortrait => 0.5,
      PokerLayoutMode.compactLandscape => 0.66,
      PokerLayoutMode.standard => 0.58,
      PokerLayoutMode.wide => 0.58,
    };
    final boardWidth =
        (tableRect.width * boardWidthFactor).clamp(148.0, tableRect.width);
    final maxBoardHeight = (tableRect.height *
            (mode == PokerLayoutMode.compactPortrait
                ? 0.145
                : (mode.isCompact ? 0.19 : 0.17)))
        .clamp(34.0, 84.0);
    final cardWidthFromHeight = maxBoardHeight / 1.4;
    final cardWidthFromWidth = (boardWidth / 5.45).clamp(24.0, 84.0);
    final communityCardWidth =
        math.min(cardWidthFromHeight, cardWidthFromWidth);
    final communityCardHeight = communityCardWidth * 1.4;
    final communityWidth = communityCardWidth * 5 + communityCardWidth * 0.4;
    final communityTop = tableRect.top +
        tableRect.height *
            (mode == PokerLayoutMode.compactPortrait ? 0.36 : 0.31);
    final communityRect = Rect.fromLTWH(
      tableRect.center.dx - communityWidth / 2,
      communityTop,
      communityWidth,
      communityCardHeight,
    );

    final potHeight = (communityCardHeight * 0.72).clamp(28.0, 44.0);
    final potWidth =
        (communityWidth * 0.62).clamp(120.0, tableRect.width * 0.64);
    final potTop = math.min(
      communityRect.bottom + gap * 0.8,
      tableRect.bottom - potHeight - gap * 2.2,
    );
    final potRect = Rect.fromLTWH(
      tableRect.center.dx - potWidth / 2,
      potTop,
      potWidth,
      potHeight,
    );

    final boardTop = communityRect.top - gap;
    final boardBottom = potRect.bottom + gap;
    final boardRect = Rect.fromLTRB(
      math.max(tableRect.left + gap, communityRect.left - gap),
      math.max(tableRect.top + gap, boardTop),
      math.min(tableRect.right - gap, communityRect.right + gap),
      math.min(tableRect.bottom - gap, boardBottom),
    );

    return PokerSceneLayout(
      mode: mode,
      screenRect: screenRect,
      safeRect: safeRect,
      contentRect: contentRect,
      tableRect: tableRect,
      communityRect: communityRect,
      potRect: potRect,
      topSeatBandRect: topSeatBandRect,
      heroDockRect: heroDockRect,
      rightRailRect: rightRailRect,
      leftRailRect: leftRailRect,
      bodyRect: bodyRect,
      boardRect: boardRect,
    );
  }

  List<Offset> opponentAnchors(int opponentCount, {double uiScale = 1.0}) {
    if (opponentCount <= 0) return const [];
    final offsets = switch ((mode, opponentCount)) {
      (PokerLayoutMode.compactPortrait, 1) => const [0.0],
      (PokerLayoutMode.compactPortrait, 2) => const [-52.0, 52.0],
      (PokerLayoutMode.compactPortrait, 3) => const [-74.0, 0.0, 74.0],
      (PokerLayoutMode.compactPortrait, 4) => const [-86.0, -28.0, 28.0, 86.0],
      (PokerLayoutMode.compactPortrait, 5) => const [
          -124.0,
          -66.0,
          0.0,
          66.0,
          124.0
        ],
      (PokerLayoutMode.compactPortrait, _) => List<double>.generate(
          opponentCount,
          (index) => -126.0 + (252.0 * index) / (opponentCount - 1),
        ),
      (_, 1) => const [0.0],
      (_, 2) => const [-40.0, 40.0],
      (_, 3) => const [-70.0, 0.0, 70.0],
      (_, 4) => const [-82.0, -28.0, 28.0, 82.0],
      (_, 5) => const [-120.0, -70.0, 0.0, 70.0, 120.0],
      _ => List<double>.generate(
          opponentCount,
          (index) => -122.0 + (244.0 * index) / (opponentCount - 1),
        ),
    };

    final xOutside = isPhonePortrait
        ? (tableRect.width * 0.042 + 14.0 * uiScale).clamp(20.0, 38.0)
        : (tableRect.width * 0.018 + 10.0 * uiScale).clamp(12.0, 28.0);
    final yOutside = isPhonePortrait
        ? (tableRect.height * 0.06 + 18.0 * uiScale).clamp(28.0, 48.0)
        : (tableRect.height * 0.038 + 14.0 * uiScale).clamp(20.0, 40.0);
    final xRadius = tableRadiusX + xOutside;
    final yRadius = tableRadiusY + yOutside;

    return offsets.map((offset) {
      final angle = (270.0 + offset) * math.pi / 180.0;
      return Offset(
        tableCenter.dx + xRadius * math.cos(angle),
        tableCenter.dy + yRadius * math.sin(angle),
      );
    }).toList(growable: false);
  }

  Offset heroSeatAnchor({double uiScale = 1.0}) {
    final bottomInset = (tableRect.height * 0.08).clamp(28.0, 56.0) * uiScale;
    return Offset(tableRect.center.dx, tableRect.bottom - bottomInset);
  }
}
