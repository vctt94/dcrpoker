import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/dialogs/create_table.dart';

enum _SortBy { players, blinds, buyIn }

class BrowsingTablesView extends StatefulWidget {
  const BrowsingTablesView({super.key, required this.model});
  final PokerModel model;

  @override
  State<BrowsingTablesView> createState() => _BrowsingTablesViewState();
}

class _BrowsingTablesViewState extends State<BrowsingTablesView> {
  _SortBy _sort = _SortBy.players;
  bool _hideFull = false;
  bool _showWaitingOnly = false;

  String _shortId(String s, [int n = 8]) => s.isEmpty ? '' : (s.length <= n ? s : s.substring(0, n));
  double _toDcr(int atoms) => atoms / 1e8;

  Widget _playerPill(UiPlayer p) {
    final ready = p.isReady;
    final color = ready ? Colors.green.shade600 : Colors.orange.shade700;
    final icon = ready ? Icons.check_circle : Icons.hourglass_empty;
    final escrowColor = p.escrowId.isEmpty
        ? Colors.white30
        : (p.escrowReady ? Colors.greenAccent : Colors.amberAccent);
    return Container(
      margin: const EdgeInsets.only(right: 8, bottom: 4),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(color: color.withOpacity(0.15), borderRadius: BorderRadius.circular(12)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 14, color: color),
          const SizedBox(width: 6),
          Text(_shortId(p.id, 10), style: TextStyle(color: color, fontWeight: FontWeight.w600)),
          if (p.escrowId.isNotEmpty) ...[
            const SizedBox(width: 8),
            Container(
              width: 10,
              height: 10,
              decoration: BoxDecoration(color: escrowColor, shape: BoxShape.circle),
            ),
          ],
        ],
      ),
    );
  }

  List<UiTable> get _filteredSortedTables {
    final model = widget.model;
    var list = model.tables;
    if (_hideFull) {
      list = list.where((t) => t.currentPlayers < t.maxPlayers).toList();
    }
    if (_showWaitingOnly) {
      list = list.where((t) => !t.gameStarted).toList();
    }
    switch (_sort) {
      case _SortBy.players:
        list = List.of(list)..sort((a, b) => (b.currentPlayers).compareTo(a.currentPlayers));
        break;
      case _SortBy.blinds:
        list = List.of(list)..sort((a, b) => (b.bigBlind).compareTo(a.bigBlind));
        break;
      case _SortBy.buyIn:
        list = List.of(list)..sort((a, b) => (a.buyInAtoms).compareTo(b.buyInAtoms));
        break;
    }
    return list;
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;

    if (model.tables.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.table_restaurant, size: 64, color: Colors.white70),
            const SizedBox(height: 16),
            const Text('No Tables Available', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.white)),
            const SizedBox(height: 8),
            const Text('Create a new table to start playing', style: TextStyle(color: Colors.white70)),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: () => CreateTableDialog.open(context, model),
              icon: const Icon(Icons.add),
              label: const Text('Create Table'),
              style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
            ),
          ],
        ),
      );
    }

    final tables = _filteredSortedTables;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Toolbar
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16.0),
          child: Row(
            children: [
              // Sort selector
              DropdownButton<_SortBy>(
                dropdownColor: const Color(0xFF1B1E2C),
                value: _sort,
                items: const [
                  DropdownMenuItem(value: _SortBy.players, child: Text('Sort: Players')),
                  DropdownMenuItem(value: _SortBy.blinds, child: Text('Sort: Blinds')),
                  DropdownMenuItem(value: _SortBy.buyIn, child: Text('Sort: Buy-in')),
                ],
                onChanged: (v) => setState(() => _sort = v ?? _sort),
              ),
              const SizedBox(width: 12),
              FilterChip(
                label: const Text('Hide full'),
                selected: _hideFull,
                onSelected: (v) => setState(() => _hideFull = v),
              ),
              const SizedBox(width: 8),
              FilterChip(
                label: const Text('Waiting only'),
                selected: _showWaitingOnly,
                onSelected: (v) => setState(() => _showWaitingOnly = v),
              ),
              const Spacer(),
              IconButton(
                tooltip: 'Refresh',
                onPressed: model.refreshTables,
                icon: const Icon(Icons.refresh, color: Colors.white70),
              ),
              const SizedBox(width: 4),
              ElevatedButton.icon(
                onPressed: () => CreateTableDialog.open(context, model),
                icon: const Icon(Icons.add),
                label: const Text('Create'),
                style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),

        // Tables list
        ListView.builder(
          padding: const EdgeInsets.all(16),
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: tables.length,
          itemBuilder: (context, index) {
            final t = tables[index];
            final full = t.currentPlayers >= t.maxPlayers;
            final needAtoms = t.buyInAtoms > 0 ? t.buyInAtoms : t.minBalanceAtoms;
            final canAfford = widget.model.myAtomsBalance >= needAtoms;
            // final canJoin = !full && canAfford;
            final canJoin = true;
            final statusColor = t.gameStarted ? Colors.green : Colors.orange;
            final statusText = t.gameStarted ? 'In Progress' : 'Waiting';

            return Card(
              margin: const EdgeInsets.only(bottom: 12),
              color: const Color(0xFF1B1E2C),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        const Icon(Icons.table_restaurant, color: Colors.blue, size: 24),
                        const SizedBox(width: 8),
                        Text('Table ${_shortId(t.id)}', style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: Colors.white)),
                        const Spacer(),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                          decoration: BoxDecoration(color: statusColor, borderRadius: BorderRadius.circular(12)),
                          child: Text(statusText, style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.bold)),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        _chip(Icons.people, '${t.currentPlayers}/${t.maxPlayers}'),
                        const SizedBox(width: 8),
                        _chip(Icons.attach_money, '${t.smallBlind}/${t.bigBlind}'),
                        const SizedBox(width: 8),
                        _chip(Icons.account_balance_wallet, '${_toDcr(t.buyInAtoms).toStringAsFixed(2)} DCR'),
                      ],
                    ),
                    const SizedBox(height: 8),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: LinearProgressIndicator(
                        minHeight: 6,
                        value: (t.maxPlayers == 0) ? 0 : (t.currentPlayers / t.maxPlayers).clamp(0, 1).toDouble(),
                        backgroundColor: Colors.white10,
                        valueColor: AlwaysStoppedAnimation(full ? Colors.redAccent : Colors.lightBlueAccent),
                      ),
                    ),
                    const SizedBox(height: 12),
                    if (t.players.isNotEmpty) ...[
                      Wrap(
                        children: t.players.map(_playerPill).toList(),
                      ),
                      const SizedBox(height: 12),
                    ],
                    Row(
                      children: [
                        Expanded(
                          child: Text('Buy-in: ${_toDcr(t.buyInAtoms).toStringAsFixed(4)} DCR  •  Blinds: ${t.smallBlind}/${t.bigBlind}',
                              style: const TextStyle(color: Colors.white70)),
                        ),
                        Tooltip(
                          message: canJoin
                              ? 'Join this table'
                              : full
                                  ? 'Table is full'
                                  : 'Insufficient balance',
                          child: ElevatedButton(
                            onPressed: canJoin ? () => widget.model.joinTable(t.id) : null,
                            style: ElevatedButton.styleFrom(
                              backgroundColor: canJoin ? Colors.green : Colors.grey,
                              foregroundColor: Colors.white,
                            ),
                            child: const Text('Join'),
                          ),
                        ),
                        const SizedBox(width: 8),
                        // Bind escrow moved to in-table lobby view.
                      ],
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ],
    );
  }

  Widget _chip(IconData icon, String text) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(color: Colors.grey.shade800, borderRadius: BorderRadius.circular(8)),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 16, color: Colors.white70),
        const SizedBox(width: 4),
        Text(text, style: const TextStyle(color: Colors.white70, fontSize: 12)),
      ]),
    );
  }
}
