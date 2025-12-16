import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'table_theme.dart';

/// Minimal showdown widget showing only winners and pot in a compact format.
/// Positioned on the right side of the screen.
class MinimalShowdown extends StatefulWidget {
  const MinimalShowdown({
    super.key,
    required this.model,
    required this.isVisible,
    required this.theme,
    this.onClose,
    this.onExpand,
  });

  final PokerModel model;
  final bool isVisible;
  final PokerThemeConfig theme;
  final VoidCallback? onClose;
  final VoidCallback? onExpand;

  @override
  State<MinimalShowdown> createState() => _MinimalShowdownState();
}

class _MinimalShowdownState extends State<MinimalShowdown>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<Offset> _slideAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 300),
    );
    _slideAnimation = Tween<Offset>(
      begin: const Offset(1.0, 0.0), // Start off-screen to the right
      end: Offset.zero, // End at final position
    ).animate(CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOutCubic,
    ));

    if (widget.isVisible) {
      _controller.forward();
    }
  }

  @override
  void didUpdateWidget(MinimalShowdown oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isVisible != oldWidget.isVisible) {
      if (widget.isVisible) {
        _controller.forward();
      } else {
        _controller.reverse();
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  String _playerLabel(String playerId) {
    final players = widget.model.showdownPlayers;
    final idx = players.indexWhere((p) => p.id == playerId);
    if (idx >= 0) {
      final p = players[idx];
      if (p.name.isNotEmpty) return p.name;
      return 'Player ${idx + 1}';
    }
    return playerId.length > 8 ? '${playerId.substring(0, 8)}...' : playerId;
  }

  String _handRankName(pr.HandRank rank) {
    switch (rank) {
      case pr.HandRank.HIGH_CARD:
        return 'High Card';
      case pr.HandRank.PAIR:
        return 'Pair';
      case pr.HandRank.TWO_PAIR:
        return 'Two Pair';
      case pr.HandRank.THREE_OF_A_KIND:
        return 'Three of a Kind';
      case pr.HandRank.STRAIGHT:
        return 'Straight';
      case pr.HandRank.FLUSH:
        return 'Flush';
      case pr.HandRank.FULL_HOUSE:
        return 'Full House';
      case pr.HandRank.FOUR_OF_A_KIND:
        return 'Four of a Kind';
      case pr.HandRank.STRAIGHT_FLUSH:
        return 'Straight Flush';
      case pr.HandRank.ROYAL_FLUSH:
        return 'Royal Flush';
      default:
        return rank.name;
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.isVisible && _controller.value == 0.0) {
      return const SizedBox.shrink();
    }

    final winners = widget.model.lastWinners;
    final pot = widget.model.showdownPot;

    // Position below Pot label (Pot is at top: 12, with ~50px height, plus 8px gap)
    return Positioned(
      top: 50, // Positioned below Pot label, same height as bet sidebar
      right: 12,
      child: SlideTransition(
        position: _slideAnimation,
        child: SafeArea(
          child: Material(
            color: Colors.transparent,
            child: Container(
              constraints: BoxConstraints(maxWidth: 200 * widget.theme.uiSizeMultiplier),
              decoration: BoxDecoration(
                color: const Color(0xFF1A1D2E).withOpacity(0.95),
                borderRadius: BorderRadius.circular(12 * widget.theme.uiSizeMultiplier),
                border: Border.all(
                  color: Colors.amber.withOpacity(0.5),
                  width: 2 * widget.theme.uiSizeMultiplier,
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withOpacity(0.5),
                    blurRadius: 10 * widget.theme.uiSizeMultiplier,
                    spreadRadius: 2 * widget.theme.uiSizeMultiplier,
                  ),
                ],
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Header
                  Container(
                    padding: EdgeInsets.symmetric(
                      horizontal: 12 * widget.theme.uiSizeMultiplier,
                      vertical: 8 * widget.theme.uiSizeMultiplier,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.3),
                      borderRadius: BorderRadius.vertical(
                        top: Radius.circular(10 * widget.theme.uiSizeMultiplier),
                      ),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.max,
                      children: [
                        Icon(Icons.emoji_events, color: Colors.amber, size: 18 * widget.theme.uiSizeMultiplier),
                        SizedBox(width: 6 * widget.theme.uiSizeMultiplier),
                        Flexible(
                          child: Text(
                            'Showdown',
                            style: TextStyle(
                              color: Colors.white,
                              fontSize: 14 * widget.theme.uiSizeMultiplier,
                              fontWeight: FontWeight.bold,
                            ),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const Spacer(),
                        if (widget.onExpand != null)
                          IconButton(
                            onPressed: widget.onExpand,
                            icon: Icon(Icons.unfold_more, color: Colors.white70, size: 18 * widget.theme.uiSizeMultiplier),
                            tooltip: 'View details',
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                          ),
                        if (widget.onClose != null) ...[
                          if (widget.onExpand != null) SizedBox(width: 4 * widget.theme.uiSizeMultiplier),
                          IconButton(
                            onPressed: widget.onClose,
                            icon: Icon(Icons.close, color: Colors.white70, size: 18 * widget.theme.uiSizeMultiplier),
                            tooltip: 'Close',
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                          ),
                        ],
                      ],
                    ),
                  ),
                  // Pot
                  if (pot > 0)
                    Padding(
                      padding: EdgeInsets.symmetric(
                        horizontal: 12 * widget.theme.uiSizeMultiplier,
                        vertical: 6 * widget.theme.uiSizeMultiplier,
                      ),
                      child: Row(
                        children: [
                          Icon(Icons.casino, color: Colors.amber, size: 16 * widget.theme.uiSizeMultiplier),
                          SizedBox(width: 6 * widget.theme.uiSizeMultiplier),
                          Text(
                            'Pot: $pot',
                            style: TextStyle(
                              color: Colors.amber,
                              fontSize: 13 * widget.theme.uiSizeMultiplier,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    ),
                  // Winners
                  if (winners.isNotEmpty)
                    Padding(
                      padding: EdgeInsets.fromLTRB(
                        12 * widget.theme.uiSizeMultiplier,
                        0,
                        12 * widget.theme.uiSizeMultiplier,
                        8 * widget.theme.uiSizeMultiplier,
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            winners.length > 1 ? 'Winners' : 'Winner',
                            style: TextStyle(
                              color: Colors.white70,
                              fontSize: 11 * widget.theme.uiSizeMultiplier,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          SizedBox(height: 4 * widget.theme.uiSizeMultiplier),
                          ...winners.take(3).map((winner) {
                            return Padding(
                              padding: EdgeInsets.only(bottom: 4 * widget.theme.uiSizeMultiplier),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(Icons.star, color: Colors.amber, size: 14 * widget.theme.uiSizeMultiplier),
                                  SizedBox(width: 4 * widget.theme.uiSizeMultiplier),
                                  Flexible(
                                    child: Text(
                                      _playerLabel(winner.playerId),
                                      style: TextStyle(
                                        color: Colors.white,
                                        fontSize: 12 * widget.theme.uiSizeMultiplier,
                                        fontWeight: FontWeight.w600,
                                      ),
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                  SizedBox(width: 6 * widget.theme.uiSizeMultiplier),
                                  Flexible(
                                    child: Text(
                                      '${_handRankName(winner.handRank)} (+${winner.winnings})',
                                      style: TextStyle(
                                        color: Colors.greenAccent,
                                        fontSize: 11 * widget.theme.uiSizeMultiplier,
                                      ),
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                ],
                              ),
                            );
                          }),
                          if (winners.length > 3)
                            Padding(
                              padding: EdgeInsets.only(top: 4 * widget.theme.uiSizeMultiplier),
                              child: Text(
                                '+${winners.length - 3} more',
                                style: TextStyle(
                                  color: Colors.white54,
                                  fontSize: 11 * widget.theme.uiSizeMultiplier,
                                  fontStyle: FontStyle.italic,
                                ),
                              ),
                            ),
                        ],
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

