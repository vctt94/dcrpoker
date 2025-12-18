import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'table_theme.dart';

/// Minimal bet sidebar widget showing betting information in a compact format on the right side.
class BetSidebar extends StatefulWidget {
  const BetSidebar({
    super.key,
    required this.gameState,
    required this.playerId,
    required this.theme,
  });

  final UiGameState gameState;
  final String playerId;
  final PokerThemeConfig theme;

  @override
  State<BetSidebar> createState() => _BetSidebarState();
}

class _BetSidebarState extends State<BetSidebar> {
  bool _isMinimized = false;

  int _getPlayerCurrentBet() {
    try {
      final player =
          widget.gameState.players.firstWhere((p) => p.id == widget.playerId);
      return player.currentBet;
    } catch (e) {
      return 0;
    }
  }

  int _getToCall() {
    final playerBet = _getPlayerCurrentBet();
    final toCall = widget.gameState.currentBet - playerBet;
    return toCall > 0 ? toCall : 0;
  }

  @override
  Widget build(BuildContext context) {
    final toCall = _getToCall();
    final playerBet = _getPlayerCurrentBet();
    final currentBet = widget.gameState.currentBet;
    final theme = widget.theme;

    // Pin near the top-right for quick scanning
    return Positioned(
      top: 12,
      right: 12,
      child: SafeArea(
        child: Material(
          color: Colors.transparent,
          child: _isMinimized
              ? // Minimized view - compact badge showing only current bet value
              Container(
                  padding: EdgeInsets.symmetric(
                    horizontal: 6 * theme.uiSizeMultiplier,
                    vertical: 4 * theme.uiSizeMultiplier,
                  ),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1A1D2E).withOpacity(0.95),
                    borderRadius:
                        BorderRadius.circular(8 * theme.uiSizeMultiplier),
                    border: Border.all(
                      color: Colors.orange.withOpacity(0.5),
                      width: 2 * theme.uiSizeMultiplier,
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withOpacity(0.5),
                        blurRadius: 10 * theme.uiSizeMultiplier,
                        spreadRadius: 2 * theme.uiSizeMultiplier,
                      ),
                    ],
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.attach_money,
                        color: Colors.orange,
                        size: 12 * theme.uiSizeMultiplier,
                      ),
                      SizedBox(width: 3 * theme.uiSizeMultiplier),
                      Text(
                        '$currentBet',
                        style: TextStyle(
                          color: Colors.orange,
                          fontSize: 11 * theme.uiSizeMultiplier,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      SizedBox(width: 3 * theme.uiSizeMultiplier),
                      MouseRegion(
                        cursor: SystemMouseCursors.click,
                        child: GestureDetector(
                          onTap: () {
                            setState(() {
                              _isMinimized = false;
                            });
                          },
                          child: Icon(
                            Icons.chevron_right,
                            color: Colors.white70,
                            size: 14 * theme.uiSizeMultiplier,
                          ),
                        ),
                      ),
                    ],
                  ),
                )
              : // Expanded view - full information
              Container(
                  constraints:
                      BoxConstraints(maxWidth: 200 * theme.uiSizeMultiplier),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1A1D2E).withOpacity(0.95),
                    borderRadius:
                        BorderRadius.circular(12 * theme.uiSizeMultiplier),
                    border: Border.all(
                      color: Colors.orange.withOpacity(0.5),
                      width: 2 * theme.uiSizeMultiplier,
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withOpacity(0.5),
                        blurRadius: 10 * theme.uiSizeMultiplier,
                        spreadRadius: 2 * theme.uiSizeMultiplier,
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
                          horizontal: 12 * theme.uiSizeMultiplier,
                          vertical: 8 * theme.uiSizeMultiplier,
                        ),
                        decoration: BoxDecoration(
                          color: Colors.black.withOpacity(0.3),
                          borderRadius: BorderRadius.vertical(
                            top: Radius.circular(10 * theme.uiSizeMultiplier),
                          ),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(
                              Icons.attach_money,
                              color: Colors.orange,
                              size: 18 * theme.uiSizeMultiplier,
                            ),
                            SizedBox(width: 6 * theme.uiSizeMultiplier),
                            Text(
                              'Betting',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: 14 * theme.uiSizeMultiplier,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            const Spacer(),
                            IconButton(
                              onPressed: () {
                                setState(() {
                                  _isMinimized = true;
                                });
                              },
                              icon: Icon(
                                Icons.remove,
                                color: Colors.white70,
                                size: 18 * theme.uiSizeMultiplier,
                              ),
                              tooltip: 'Minimize',
                              padding: EdgeInsets.zero,
                              constraints: BoxConstraints(),
                            ),
                          ],
                        ),
                      ),
                      // Current bet
                      Padding(
                        padding: EdgeInsets.symmetric(
                          horizontal: 12 * theme.uiSizeMultiplier,
                          vertical: 6 * theme.uiSizeMultiplier,
                        ),
                        child: Text(
                          'Current Bet: $currentBet',
                          style: TextStyle(
                            color: Colors.orange,
                            fontSize: 13 * theme.uiSizeMultiplier,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      // To call (if player needs to call)
                      if (toCall > 0)
                        Padding(
                          padding: EdgeInsets.fromLTRB(
                            12 * theme.uiSizeMultiplier,
                            0,
                            12 * theme.uiSizeMultiplier,
                            8 * theme.uiSizeMultiplier,
                          ),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(
                                    Icons.call_made,
                                    color: Colors.orangeAccent,
                                    size: 14 * theme.uiSizeMultiplier,
                                  ),
                                  SizedBox(width: 4 * theme.uiSizeMultiplier),
                                  Text(
                                    'To Call: $toCall',
                                    style: TextStyle(
                                      color: Colors.orangeAccent,
                                      fontSize: 12 * theme.uiSizeMultiplier,
                                      fontWeight: FontWeight.w600,
                                    ),
                                  ),
                                ],
                              ),
                              if (playerBet > 0) ...[
                                SizedBox(height: 4 * theme.uiSizeMultiplier),
                                Text(
                                  'Your bet: $playerBet',
                                  style: TextStyle(
                                    color: Colors.white70,
                                    fontSize: 11 * theme.uiSizeMultiplier,
                                  ),
                                ),
                              ],
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
        ),
      ),
    );
  }
}
