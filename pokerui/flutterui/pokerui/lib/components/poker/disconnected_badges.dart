import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'table.dart';
import 'table_theme.dart';

class DisconnectedBadgesOverlay extends StatelessWidget {
  const DisconnectedBadgesOverlay({
    super.key,
    required this.layout,
    required this.theme,
    required this.players,
    required this.heroId,
    required this.hasCurrentBet,
  });

  final TableLayout layout;
  final PokerThemeConfig theme;
  final List<UiPlayer> players;
  final String heroId;
  final bool hasCurrentBet;

  @override
  Widget build(BuildContext context) {
    if (players.isEmpty) return const SizedBox.shrink();
    final scene = layout.scene;
    final seats = seatPositionsFor(
      players,
      heroId,
      layout.center,
      layout.ringRadiusX,
      layout.ringRadiusY,
      clampBounds: layout.canvasBounds,
      minSeatTop: minSeatTopFor(layout.viewport, hasCurrentBet),
      uiSizeMultiplier: theme.uiSizeMultiplier,
      sceneLayout: scene,
    );

    final widgets = <Widget>[];
    for (final p in players) {
      if (!p.isDisconnected) continue;
      final pos = seats[p.id];
      if (pos == null) continue;
      final isHero = p.id == heroId;
      final preferredTop = pos.dy + (isHero ? 20.0 : 46.0);
      final maxTop = scene.heroDockRect.top - 28.0;
      final top = preferredTop.clamp(
        isHero ? scene.tableRect.bottom - 36.0 : scene.contentRect.top + 6.0,
        maxTop,
      );
      widgets.add(Positioned(
        left: pos.dx - 36,
        top: top,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
          constraints: const BoxConstraints(minWidth: 60, maxWidth: 140),
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: PokerColors.danger.withOpacity(0.8),
            borderRadius: BorderRadius.circular(14),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.signal_wifi_off, size: 14, color: Colors.white),
              const SizedBox(width: 4),
              Flexible(
                child: Text(
                  p.name.isNotEmpty ? _shortName(p.name) : 'Disconnected',
                  overflow: TextOverflow.ellipsis,
                  style: PokerTypography.labelSmall.copyWith(
                    color: Colors.white,
                    fontSize: 11,
                  ),
                ),
              ),
            ],
          ),
        ),
      ));
    }

    return IgnorePointer(
      ignoring: false,
      child: Stack(children: widgets),
    );
  }

  static String _shortName(String name) {
    if (name.length <= 10) return name;
    return '${name.substring(0, 10)}…';
  }
}
