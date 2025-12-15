import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/showdown_content.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

/// A sidebar that slides in from the left showing showdown information.
/// Can be displayed in minimal (collapsed) or expanded (full) mode.
class ShowdownSidebar extends StatefulWidget {
  const ShowdownSidebar({
    super.key,
    required this.model,
    required this.isVisible,
    this.onClose,
    this.minimal = false,
    this.onExpand,
    this.initialExpanded = false,
  });

  final PokerModel model;
  final bool isVisible;
  final VoidCallback? onClose;
  final bool minimal;
  final VoidCallback? onExpand;
  final bool initialExpanded;

  @override
  State<ShowdownSidebar> createState() => _ShowdownSidebarState();
}

class _ShowdownSidebarState extends State<ShowdownSidebar>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<Offset> _slideAnimation;
  bool _isExpanded = false;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 300),
    );
    _slideAnimation = Tween<Offset>(
      begin: const Offset(-1.0, 0.0), // Start off-screen to the left
      end: Offset.zero, // End at final position
    ).animate(CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOutCubic,
    ));

    if (widget.isVisible) {
      _controller.forward();
    }
    // Set initial expansion state
    _isExpanded = widget.initialExpanded || !widget.minimal;
  }

  @override
  void didUpdateWidget(ShowdownSidebar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isVisible != oldWidget.isVisible) {
      if (widget.isVisible) {
        _controller.forward();
      } else {
        _controller.reverse();
      }
    }
    // Reset expansion state if minimal mode or initialExpanded changes
    if (widget.minimal != oldWidget.minimal || widget.initialExpanded != oldWidget.initialExpanded) {
      _isExpanded = widget.initialExpanded || !widget.minimal;
    }
  }

  void _toggleExpanded() {
    setState(() {
      _isExpanded = !_isExpanded;
    });
    if (_isExpanded && widget.onExpand != null) {
      widget.onExpand!();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.isVisible && _controller.value == 0.0) {
      return const SizedBox.shrink();
    }

    // Minimal mode: show compact widget at top-left
    if (widget.minimal && !_isExpanded) {
      return Positioned(
        top: 12,
        left: 12,
        child: SlideTransition(
          position: _slideAnimation,
          child: _MinimalShowdownWidget(
            model: widget.model,
            onExpand: _toggleExpanded,
            onClose: widget.onClose,
          ),
        ),
      );
    }

    // Expanded mode: show full sidebar
    return Positioned(
      top: 0,
      left: 0,
      bottom: 0,
      child: SlideTransition(
        position: _slideAnimation,
        child: Container(
        width: 400,
        margin: EdgeInsets.zero,
        padding: EdgeInsets.zero,
        decoration: BoxDecoration(
          color: const Color(0xFF1A1D2E),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.5),
              blurRadius: 20,
              spreadRadius: 5,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.max,
          children: [
            // Header with close button
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.black.withOpacity(0.3),
                border: Border(
                  bottom: BorderSide(
                    color: Colors.amber.withOpacity(0.5),
                    width: 2,
                  ),
                ),
              ),
              child: Row(
                children: [
                  const Icon(Icons.history, color: Colors.amber, size: 24),
                  const SizedBox(width: 10),
                  const Expanded(
                    child: Text(
                      'Showdown',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                  if (widget.minimal)
                    IconButton(
                      onPressed: _toggleExpanded,
                      icon: const Icon(Icons.unfold_less, color: Colors.white70),
                      tooltip: 'Collapse',
                    ),
                  IconButton(
                    onPressed: widget.onClose,
                    icon: const Icon(Icons.close, color: Colors.white70),
                    tooltip: 'Close',
                  ),
                ],
              ),
            ),
            // Showdown content
            Expanded(
              child: ShowdownContent(
                model: widget.model,
                showHeader: false,
                showCloseButton: false,
              ),
            ),
          ],
        ),
      ),
      ),
    );
  }
}

/// Minimal showdown widget showing only winners and pot in a compact format.
class _MinimalShowdownWidget extends StatelessWidget {
  const _MinimalShowdownWidget({
    required this.model,
    required this.onExpand,
    this.onClose,
  });

  final PokerModel model;
  final VoidCallback onExpand;
  final VoidCallback? onClose;

  String _playerLabel(String playerId) {
    final players = model.showdownPlayers;
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
    final winners = model.lastWinners;
    final pot = model.showdownPot;

    return SafeArea(
      child: Material(
        color: Colors.transparent,
        child: Container(
          constraints: const BoxConstraints(maxWidth: 280),
          decoration: BoxDecoration(
            color: const Color(0xFF1A1D2E).withOpacity(0.95),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: Colors.amber.withOpacity(0.5),
              width: 2,
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity(0.5),
                blurRadius: 10,
                spreadRadius: 2,
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.3),
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(10),
                  ),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.emoji_events, color: Colors.amber, size: 18),
                    const SizedBox(width: 6),
                    const Text(
                      'Showdown',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const Spacer(),
                    IconButton(
                      onPressed: onExpand,
                      icon: const Icon(Icons.unfold_more, color: Colors.white70, size: 18),
                      tooltip: 'View details',
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(),
                    ),
                    if (onClose != null) ...[
                      const SizedBox(width: 4),
                      IconButton(
                        onPressed: onClose,
                        icon: const Icon(Icons.close, color: Colors.white70, size: 18),
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
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                  child: Row(
                    children: [
                      const Icon(Icons.casino, color: Colors.amber, size: 16),
                      const SizedBox(width: 6),
                      Text(
                        'Pot: $pot',
                        style: const TextStyle(
                          color: Colors.amber,
                          fontSize: 13,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                ),
              // Winners
              if (winners.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        winners.length > 1 ? 'Winners' : 'Winner',
                        style: const TextStyle(
                          color: Colors.white70,
                          fontSize: 11,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 4),
                      ...winners.take(3).map((winner) {
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(Icons.star, color: Colors.amber, size: 14),
                              const SizedBox(width: 4),
                              Flexible(
                                child: Text(
                                  _playerLabel(winner.playerId),
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 12,
                                    fontWeight: FontWeight.w600,
                                  ),
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                              const SizedBox(width: 6),
                              Flexible(
                                child: Text(
                                  '${_handRankName(winner.handRank)} (+${winner.winnings})',
                                  style: const TextStyle(
                                    color: Colors.greenAccent,
                                    fontSize: 11,
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
                          padding: const EdgeInsets.only(top: 4),
                          child: Text(
                            '+${winners.length - 3} more',
                            style: const TextStyle(
                              color: Colors.white54,
                              fontSize: 11,
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
    );
  }
}

