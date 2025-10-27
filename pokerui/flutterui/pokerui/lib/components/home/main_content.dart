import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker_game.dart';

/// Poker main content widget that displays tables and game state
class PokerMainContent extends StatefulWidget {
  final PokerModel model;
  const PokerMainContent({super.key, required this.model});

  @override
  State<PokerMainContent> createState() => _PokerMainContentState();
}

class _PokerMainContentState extends State<PokerMainContent> {
  final TextEditingController _betCtrl = TextEditingController();
  bool _showBetInput = false;

  // Safely shorten an ID for debug/UI without throwing on short/empty strings.
  String _shortId(String s, [int n = 8]) {
    if (s.isEmpty) return '';
    return s.length <= n ? s : s.substring(0, n);
  }

  @override
  void dispose() {
    _betCtrl.dispose();
    super.dispose();
  }
  @override
  Widget build(BuildContext context) {
    // Guard against stale state: if not seated, always render browsing
    final effectiveState =
        widget.model.currentTableId == null ? PokerState.browsingTables : widget.model.state;
    // Show appropriate content based on effective state
    switch (effectiveState) {
      case PokerState.idle:
        return _buildIdleState(context, widget.model);
      case PokerState.browsingTables:
        return _buildBrowsingTablesState(context, widget.model);
      case PokerState.inLobby:
        return _buildInLobbyState(context, widget.model);
      case PokerState.handInProgress:
        return _buildHandInProgressState(context, widget.model);
      case PokerState.showdown:
        return _buildShowdownState(context, widget.model);
      case PokerState.tournamentOver:
        return _buildTournamentOverState(context, widget.model);
    }
  }

  Widget _buildIdleState(BuildContext context, PokerModel model) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.casino, size: 64, color: Colors.white70),
          const SizedBox(height: 16),
          const Text(
            'Welcome to Poker!',
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 8),
          const Text(
            'Connect to a poker server to start playing',
            style: TextStyle(color: Colors.white70),
          ),
          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton.icon(
                onPressed: () {
                  model.refreshTables();
                },
                icon: const Icon(Icons.refresh),
                label: const Text('Connect & Refresh'),
                style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
              ),
              const SizedBox(width: 16),
              ElevatedButton.icon(
                onPressed: () {
                  // TODO: Implement create table functionality
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Create table functionality coming soon')),
                  );
                },
                icon: const Icon(Icons.add),
                label: const Text('Create Table'),
                style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildBrowsingTablesState(BuildContext context, PokerModel model) {
    if (model.tables.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.table_restaurant, size: 64, color: Colors.white70),
            const SizedBox(height: 16),
            const Text(
              'No Tables Available',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.white),
            ),
            const SizedBox(height: 8),
            const Text(
              'Create a new table to start playing',
              style: TextStyle(color: Colors.white70),
            ),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: () {
                // TODO: Implement create table functionality
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Create table functionality coming soon')),
                );
              },
              icon: const Icon(Icons.add),
              label: const Text('Create Table'),
              style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
            ),
          ],
        ),
      );
    }

    // List is embedded inside a parent scroll view on the Home screen.
    // Make it non-scrollable here to avoid nested scroll conflicts.
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: model.tables.length,
      itemBuilder: (context, index) {
          final table = model.tables[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: const Color(0xFF1B1E2C),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Icon(Icons.table_restaurant, color: Colors.blue, size: 24),
                      const SizedBox(width: 8),
                      Text(
                        'Table ${_shortId(table.id)}...',
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: Colors.white,
                        ),
                      ),
                      const Spacer(),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                        decoration: BoxDecoration(
                          color: table.gameStarted ? Colors.green : Colors.orange,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: Text(
                          table.gameStarted ? 'In Progress' : 'Waiting',
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 12,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      _buildInfoChip(Icons.people, '${table.currentPlayers}/${table.maxPlayers}'),
                      const SizedBox(width: 8),
                      _buildInfoChip(Icons.attach_money, '${table.smallBlind}/${table.bigBlind}'),
                      const SizedBox(width: 8),
                      _buildInfoChip(Icons.account_balance_wallet, '${(table.buyInAtoms / 1e8).toStringAsFixed(2)} DCR'),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          'Phase: ${table.phase.label}',
                          style: const TextStyle(color: Colors.white70),
                        ),
                      ),
                      ElevatedButton(
                        onPressed: () async {
                          final ok = await model.joinTable(table.id);
                          if (ok && context.mounted) {
                            // Navigate to the dedicated table screen
                            Navigator.pushNamed(context, '/table');
                          }
                        },
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.green,
                          foregroundColor: Colors.white,
                        ),
                        child: const Text('Join Table'),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
    );
  }

  Widget _buildInfoChip(IconData icon, String text) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.grey.shade800,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 16, color: Colors.white70),
          const SizedBox(width: 4),
          Text(
            text,
            style: const TextStyle(color: Colors.white70, fontSize: 12),
          ),
        ],
      ),
    );
  }

  Widget _buildInLobbyState(BuildContext context, PokerModel model) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.table_restaurant, size: 64, color: Colors.white70),
          const SizedBox(height: 16),
          Text(
            'Table: ${model.currentTableId}',
            style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 8),
          Text(
            'State: ${model.state.name}',
            style: const TextStyle(color: Colors.white70),
          ),
          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton(
                onPressed: model.iAmReady ? model.setUnready : model.setReady,
                child: Text(model.iAmReady ? 'Unready' : 'Ready'),
              ),
              const SizedBox(width: 16),
              ElevatedButton(
                onPressed: model.leaveTable,
                style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                child: const Text('Leave Table'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildHandInProgressState(BuildContext context, PokerModel model) {
    final game = model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    final focusNode = FocusNode();
    final pokerGame = PokerGame(model.playerId, model);

    return Stack(
      children: [
        // Poker game visualization
        pokerGame.buildWidget(game, focusNode),
        
        // Action buttons overlay
        Positioned(
          bottom: 20,
          left: 0,
          right: 0,
          child: Center(
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                // Always offer a way to leave the table, even when it's not your turn
                ElevatedButton(
                  onPressed: model.leaveTable,
                  style: ElevatedButton.styleFrom(backgroundColor: Colors.redAccent),
                  child: const Text('Leave Table'),
                ),
                const SizedBox(width: 12),
                if (model.isMyTurn) ...[
                  // Fold is always available on your turn
                  ElevatedButton(
                    onPressed: () => model.fold(),
                    style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                    child: const Text('Fold (F)'),
                  ),
                  const SizedBox(width: 8),
                  // Show Check or Call only when appropriate
                  Builder(builder: (context) {
                    final g = model.game;
                    final me = model.me;
                    final currentBet = g?.currentBet ?? 0;
                    final myBet = me?.currentBet ?? 0;
                    final canCheck = myBet >= currentBet;
                    final toCall = (currentBet - myBet) > 0 ? (currentBet - myBet) : 0;
                    return Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (canCheck) ...[
                          ElevatedButton(
                            onPressed: () => model.check(),
                            child: const Text('Check (K)'),
                          ),
                          const SizedBox(width: 8),
                        ] else ...[
                          ElevatedButton(
                            onPressed: () => model.callBet(),
                            child: Text('Call${toCall > 0 ? ' ($toCall)' : ''} (C)'),
                          ),
                          const SizedBox(width: 8),
                        ],
                        // Bet/Raise button toggles bet input visibility
                        Builder(builder: (context) {
                          final tid = model.currentTableId;
                          final table = tid == null
                              ? null
                              : (() {
                                  final matches = model.tables.where((t) => t.id == tid).toList();
                                  return matches.isNotEmpty ? matches.first : null;
                                })();
                          final bb = table?.bigBlind ?? 0;
                          final isRaise = currentBet > 0 && myBet < currentBet;

                          void seedDefault() {
                            // Pre-fill with amount to ADD (not total)
                            // Default: raise to 3x BB or minimum raise if facing a bet
                            final defaultBet = (bb * 3);
                            final targetTotal = (defaultBet > currentBet) ? defaultBet : currentBet;
                            final amountToAdd = targetTotal - myBet;
                            if (amountToAdd > 0) {
                              _betCtrl.text = amountToAdd.toString();
                            }
                          }

                          Future<void> submitBet() async {
                            final raw = _betCtrl.text.trim();
                            final amt = int.tryParse(raw) ?? 0;
                            if (amt <= 0) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('Enter a valid bet amount')),
                              );
                              return;
                            }
                            
                            // Calculate total bet: user enters amount to ADD, server expects TOTAL
                            // If raising, minimum raise is currentBet + (currentBet - myBet)
                            // If opening bet, minimum is typically BB
                            final totalBet = myBet + amt;
                            
                            // Pre-check: when facing a bet, total must be at least currentBet
                            if (currentBet > 0 && totalBet < currentBet) {
                              final minRaise = currentBet - myBet;
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text('Must add at least $minRaise to call ($currentBet total)')),
                              );
                              return;
                            }
                            
                            final ok = await model.makeBet(totalBet);
                            if (!ok && model.errorMessage.isNotEmpty) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text(model.errorMessage)),
                              );
                              return;
                            }
                            setState(() {
                              _showBetInput = false;
                            });
                          }

                          void setTo3xBB() {
                            // Set amount to ADD to reach 3x BB total
                            final defaultBet = (bb * 3);
                            final targetTotal = (defaultBet > currentBet) ? defaultBet : currentBet;
                            final amountToAdd = targetTotal - myBet;
                            _betCtrl.text = amountToAdd.toString();
                          }

                          if (!_showBetInput) {
                            return ElevatedButton(
                              onPressed: () {
                                setState(() {
                                  _showBetInput = true;
                                });
                                if (_betCtrl.text.isEmpty) seedDefault();
                              },
                              style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
                              child: Text(isRaise ? 'Raise' : 'Bet'),
                            );
                          }

                          // Bet input row (visible after pressing Bet/Raise)
                          return Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              SizedBox(
                                width: 90,
                                child: TextField(
                                  controller: _betCtrl,
                                  keyboardType: TextInputType.number,
                                  style: const TextStyle(color: Colors.white),
                                  decoration: InputDecoration(
                                    isDense: true,
                                    contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                                    hintText: isRaise ? 'Raise' : 'Bet',
                                    hintStyle: const TextStyle(color: Colors.white70),
                                    filled: true,
                                    fillColor: Colors.black54,
                                    border: OutlineInputBorder(
                                      borderRadius: BorderRadius.circular(8),
                                      borderSide: const BorderSide(color: Colors.white24),
                                    ),
                                  ),
                                  onSubmitted: (_) => submitBet(),
                                ),
                              ),
                              const SizedBox(width: 6),
                              ElevatedButton(onPressed: bb > 0 ? setTo3xBB : null, child: const Text('3x BB')),
                              const SizedBox(width: 6),
                              ElevatedButton(
                                onPressed: submitBet,
                                style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
                                child: Text(isRaise ? 'Raise' : 'Bet'),
                              ),
                              const SizedBox(width: 6),
                              TextButton(
                                onPressed: () {
                                  setState(() {
                                    _showBetInput = false;
                                  });
                                },
                                child: const Text('Cancel'),
                              )
                            ],
                          );
                        }),
                      ],
                    );
                  }),
                ] else ...[
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.7),
                      borderRadius: BorderRadius.circular(20),
                    ),
                    child: Text(
                      'Waiting for your turn...',
                      style: const TextStyle(color: Colors.white, fontSize: 16),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildShowdownState(BuildContext context, PokerModel model) {
    final game = model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    final focusNode = FocusNode();
    final pokerGame = PokerGame(model.playerId, model);

    // Minimal showdown overlay: keep table visible; show only who won (Pn) without cards.
    final winners = model.lastWinners;
    final players = game.players;

    String _pLabel(String pid) {
      final idx = players.indexWhere((p) => p.id == pid);
      return idx >= 0 ? 'P${idx + 1}' : 'P?';
    }

    return Stack(
      children: [
        // Reuse the poker table canvas (community cards + seats stay visible)
        pokerGame.buildWidget(game, focusNode),

        // Compact winners banner at the top center
        if (winners.isNotEmpty)
          Positioned(
            top: 16,
            left: 0,
            right: 0,
            child: Center(
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.78),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.amber, width: 1.5),
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    const Text(
                      'Showdown',
                      style: TextStyle(color: Colors.amber, fontSize: 16, fontWeight: FontWeight.w800),
                    ),
                    const SizedBox(height: 6),
                    for (final w in winners)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 2),
                        child: Text(
                          _pLabel(w.playerId),
                          style: const TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w700),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),

        // Leave table control stays available
        Positioned(
          bottom: 20,
          left: 0,
          right: 0,
          child: Center(
            child: ElevatedButton(
              onPressed: model.leaveTable,
              style: ElevatedButton.styleFrom(backgroundColor: Colors.redAccent),
              child: const Text('Leave Table'),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildTournamentOverState(BuildContext context, PokerModel model) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.flag, size: 64, color: Colors.green),
          const SizedBox(height: 16),
          const Text(
            'Tournament Over!',
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () {
              model.leaveTable();
            },
            child: const Text('Return to Lobby'),
          ),
        ],
      ),
    );
  }

}
