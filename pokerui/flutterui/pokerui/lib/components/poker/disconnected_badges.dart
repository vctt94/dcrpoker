import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'table.dart';
import 'table_theme.dart';

class DisconnectedBadgesOverlay extends StatelessWidget {
  const DisconnectedBadgesOverlay({
    super.key,
    required this.players,
    required this.heroId,
    required this.hasCurrentBet,
  });

  final List<UiPlayer> players;
  final String heroId;
  final bool hasCurrentBet;

  @override
  Widget build(BuildContext context) {
    if (players.isEmpty) return const SizedBox.shrink();
    return LayoutBuilder(builder: (context, c) {
      final theme = PokerThemeConfig.fromContext(context);
      final layout = resolveTableLayout(c.biggest);
      final seats = seatPositionsFor(
        players,
        heroId,
        layout.center,
        layout.ringRadiusX,
        layout.ringRadiusY,
        clampBounds: layout.viewport,
        minSeatTop: minSeatTopFor(layout.viewport, hasCurrentBet),
        uiSizeMultiplier: theme.uiSizeMultiplier,
      );

      final widgets = <Widget>[];
      for (final p in players) {
        if (!p.isDisconnected) continue;
        final pos = seats[p.id];
        if (pos == null) continue;
        widgets.add(Positioned(
          left: pos.dx - 36,
          top: pos.dy + 24,
          child: Tooltip(
            message: 'Player disconnected',
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              constraints: const BoxConstraints(minWidth: 60, maxWidth: 140),
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: Colors.red.withOpacity(0.8),
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
                      style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w600),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ));
      }

      return IgnorePointer(
        ignoring: false,
        child: Stack(children: widgets),
      );
    });
  }

  static String _shortName(String name) {
    if (name.length <= 10) return name;
    return '${name.substring(0, 10)}…';
  }
}
