import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class BrowsingTablesView extends StatelessWidget {
  const BrowsingTablesView({super.key, required this.model});
  final PokerModel model;

  // Safely shorten an ID for debug/UI without throwing on short/empty strings.
  String _shortId(String s, [int n = 8]) {
    if (s.isEmpty) return '';
    return s.length <= n ? s : s.substring(0, n);
  }

  @override
  Widget build(BuildContext context) {
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
                        await model.joinTable(table.id);
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
}
