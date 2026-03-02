import 'package:flutter/material.dart';

/// Screen-width breakpoints for adaptive poker layout.
enum PokerBreakpoint {
  /// Phone portrait, narrow (< 390 logical px)
  compact,

  /// Phone portrait, typical (390–599)
  regular,

  /// Tablet portrait or small landscape (600–899)
  expanded,

  /// Desktop / large tablet landscape (≥ 900)
  wide,
}

extension PokerBreakpointQuery on PokerBreakpoint {
  /// Resolve the breakpoint from the current [BuildContext].
  static PokerBreakpoint of(BuildContext context) {
    final w = MediaQuery.sizeOf(context).width;
    if (w < 390) return PokerBreakpoint.compact;
    if (w < 600) return PokerBreakpoint.regular;
    if (w < 900) return PokerBreakpoint.expanded;
    return PokerBreakpoint.wide;
  }

  /// Resolve from a raw width value (useful inside LayoutBuilder).
  static PokerBreakpoint fromWidth(double width) {
    if (width < 390) return PokerBreakpoint.compact;
    if (width < 600) return PokerBreakpoint.regular;
    if (width < 900) return PokerBreakpoint.expanded;
    return PokerBreakpoint.wide;
  }

  bool get isCompact => this == PokerBreakpoint.compact;
  bool get isRegular => this == PokerBreakpoint.regular;
  bool get isExpanded => this == PokerBreakpoint.expanded;
  bool get isWide => this == PokerBreakpoint.wide;

  bool get isNarrow => isCompact || isRegular;
}

/// Adaptive table aspect ratio per breakpoint.
///
/// Portrait phones use a taller ratio so the table occupies more vertical
/// space; landscape / desktop keeps the standard 16:9.
double tableAspectRatio(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 1.0;
    case PokerBreakpoint.regular:
      return 1.15;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 16 / 9;
  }
}

/// Maximum pixel width for the table canvas on wide screens.
double tableMaxWidth(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
    case PokerBreakpoint.regular:
      return double.infinity;
    case PokerBreakpoint.expanded:
      return 900;
    case PokerBreakpoint.wide:
      return 1200;
  }
}

/// Fixed width of the side-rail panel (only rendered on wide).
const double kSideRailWidth = 280;

/// Minimum height reserved for the bottom action dock.
double actionDockMinHeight(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 52;
    case PokerBreakpoint.regular:
      return 56;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 60;
  }
}

/// Scale factor applied to action-bar buttons.
double buttonScale(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 0.85;
    case PokerBreakpoint.regular:
      return 0.92;
    case PokerBreakpoint.expanded:
      return 1.0;
    case PokerBreakpoint.wide:
      return 1.05;
  }
}

/// Scale factor for general font sizing.
double fontScale(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 0.88;
    case PokerBreakpoint.regular:
      return 0.94;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 1.0;
  }
}

/// Whether a side rail should be rendered at this breakpoint.
bool showSideRail(PokerBreakpoint bp) => bp == PokerBreakpoint.wide;

/// Vertical share used by the table canvas in the phone layout.
double mobileTableHeightFraction(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 0.56;
    case PokerBreakpoint.regular:
      return 0.6;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 0.65;
  }
}

/// Minimum reserved height for the mobile hero/action panel.
double mobileHeroPanelMinHeight(PokerBreakpoint bp) {
  switch (bp) {
    case PokerBreakpoint.compact:
      return 210;
    case PokerBreakpoint.regular:
      return 196;
    case PokerBreakpoint.expanded:
    case PokerBreakpoint.wide:
      return 180;
  }
}

/// Bottom safe-area padding, falling back to a sensible minimum.
double safeBottomPadding(BuildContext context, {double minPadding = 8}) {
  final viewPadding = MediaQuery.viewPaddingOf(context).bottom;
  return viewPadding > 0 ? viewPadding : minPadding;
}
