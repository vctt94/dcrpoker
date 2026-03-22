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
        topBandHeight = (contentRect.height * 0.15).clamp(96.0, 120.0);
        heroDockHeight = (contentRect.height * 0.2).clamp(156.0, 196.0);
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
    final tableRect = Rect.fromLTRB(
      bodyRect.left,
      bodyRect.top,
      math.max(bodyRect.left, rightRailRect.left - gap),
      bodyRect.bottom,
    );
    final topSeatBandRect = Rect.fromLTRB(
      contentRect.left,
      contentRect.top,
      tableRect.right,
      contentRect.top + topBandHeight,
    );

    final boardWidthFactor = mode.isCompact ? 0.66 : 0.58;
    final boardWidth =
        (tableRect.width * boardWidthFactor).clamp(148.0, tableRect.width);
    final maxBoardHeight =
        (tableRect.height * (mode.isCompact ? 0.19 : 0.17)).clamp(38.0, 84.0);
    final cardWidthFromHeight = maxBoardHeight / 1.4;
    final cardWidthFromWidth = (boardWidth / 5.45).clamp(24.0, 84.0);
    final communityCardWidth =
        math.min(cardWidthFromHeight, cardWidthFromWidth);
    final communityCardHeight = communityCardWidth * 1.4;
    final communityWidth = communityCardWidth * 5 + communityCardWidth * 0.4;
    final communityTop = tableRect.top +
        tableRect.height *
            (mode == PokerLayoutMode.compactPortrait ? 0.29 : 0.31);
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

  List<Offset> opponentAnchors(int opponentCount) {
    if (opponentCount <= 0) return const [];
    final shouldUseTwoRows = opponentCount > 4 ||
        (mode == PokerLayoutMode.compactPortrait && opponentCount > 3);
    final rows = shouldUseTwoRows ? 2 : 1;
    final firstRowCount =
        rows == 1 ? opponentCount : (opponentCount / 2).ceil();
    final secondRowCount = opponentCount - firstRowCount;
    final topInset = (topSeatBandRect.height * 0.16).clamp(8.0, 20.0);
    final bottomInset = (topSeatBandRect.height * 0.16).clamp(8.0, 20.0);
    final singleRowYFactor = switch (mode) {
      PokerLayoutMode.compactPortrait => 0.70,
      PokerLayoutMode.compactLandscape => 0.66,
      PokerLayoutMode.standard => 0.58,
      PokerLayoutMode.wide => 0.54,
    };
    final upperRowFactor = mode.isCompact ? 0.24 : 0.18;
    final lowerRowFactor = mode.isCompact ? 0.08 : 0.14;
    final yPositions = rows == 1
        ? [topSeatBandRect.top + topSeatBandRect.height * singleRowYFactor]
        : [
            topSeatBandRect.top +
                topInset +
                (topSeatBandRect.height * upperRowFactor),
            topSeatBandRect.bottom -
                bottomInset -
                (topSeatBandRect.height * lowerRowFactor),
          ];

    List<Offset> distribute(int count, double y, double horizontalInset) {
      if (count <= 0) return const [];
      if (count == 1) {
        return [Offset(topSeatBandRect.center.dx, y)];
      }
      if (count == 2) {
        final leftFactor = switch (mode) {
          PokerLayoutMode.compactPortrait => 0.26,
          PokerLayoutMode.compactLandscape => 0.28,
          PokerLayoutMode.standard => 0.32,
          PokerLayoutMode.wide => 0.34,
        };
        return [
          Offset(topSeatBandRect.left + topSeatBandRect.width * leftFactor, y),
          Offset(
            topSeatBandRect.right - topSeatBandRect.width * leftFactor,
            y,
          ),
        ];
      }
      final adjustedInset = count <= 2
          ? horizontalInset * 2.0
          : (count == 3 ? horizontalInset * 1.45 : horizontalInset);
      final usable = Rect.fromLTRB(
        topSeatBandRect.left + adjustedInset,
        topSeatBandRect.top,
        topSeatBandRect.right - adjustedInset,
        topSeatBandRect.bottom,
      );
      final spacing = usable.width / (count + 1);
      return List<Offset>.generate(
        count,
        (index) => Offset(usable.left + spacing * (index + 1), y),
      );
    }

    final inset = (topSeatBandRect.width * (mode.isCompact ? 0.06 : 0.08))
        .clamp(18.0, 60.0);

    final anchors = <Offset>[
      ...distribute(firstRowCount, yPositions.first, inset),
    ];
    if (secondRowCount > 0) {
      anchors.addAll(distribute(secondRowCount, yPositions.last, inset * 1.4));
    }
    return anchors;
  }

  Offset heroSeatAnchor({double uiScale = 1.0}) {
    final bottomInset = (tableRect.height * 0.08).clamp(28.0, 56.0) * uiScale;
    return Offset(tableRect.center.dx, tableRect.bottom - bottomInset);
  }
}
