import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'table.dart';

class DisconnectedBadgesOverlay extends StatelessWidget {
  const DisconnectedBadgesOverlay({
    super.key,
    required this.players,
    required this.heroId,
  });

  final List<UiPlayer> players;
  final String heroId;

  @override
  Widget build(BuildContext context) {
    if (players.isEmpty) return const SizedBox.shrink();
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final box = _pokerViewportRect(size);
      final center = Offset(box.left + box.width / 2, box.top + box.height / 2);
      final tableRadius = (box.width * 0.4).clamp(100.0, 200.0);
      final seats = seatPositionsFor(players, heroId, center, tableRadius + 50);

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

  static Rect _pokerViewportRect(Size size) {
    const double aspect = 16 / 9;
    final double containerAspect = size.width / (size.height == 0 ? 1 : size.height);
    double w, h, left, top;
    if (containerAspect > aspect) {
      h = size.height;
      w = h * aspect;
      left = (size.width - w) / 2;
      top = 0;
    } else {
      w = size.width;
      h = w / aspect;
      left = 0;
      top = (size.height - h) / 2;
    }
    return Rect.fromLTWH(left, top, w, h);
  }

  static String _shortName(String name) {
    if (name.length <= 10) return name;
    return '${name.substring(0, 10)}…';
  }
}
