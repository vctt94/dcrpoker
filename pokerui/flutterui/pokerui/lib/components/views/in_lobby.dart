import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class InLobbyView extends StatelessWidget {
  const InLobbyView({super.key, required this.model});
  final PokerModel model;

  String _short(String s, [int n = 8]) =>
      s.isEmpty ? '' : (s.length <= n ? s : s.substring(0, n));

  String _playerLabel(UiPlayer p) {
    final name = p.name.trim();
    if (name.isNotEmpty) return name;
    return _short(p.id, 10);
  }

  Future<void> _showBindDialog(BuildContext ctx, UiTable t) async {
    final escrowCtrl = TextEditingController();
    final escrows = await model.listCachedEscrows();
    String? selectedOutpoint = escrows.isNotEmpty
        ? '${escrows.first['funding_txid'] ?? ''}:${(escrows.first['funding_vout'] ?? 0).toString()}'
        : null;
    int seatIndex =
        t.players.length < t.maxPlayers ? t.players.length : t.maxPlayers - 1;
    final formKey = GlobalKey<FormState>();
    await showDialog(
      context: ctx,
      builder: (dctx) {
        return AlertDialog(
          title: const Text('Bind Escrow'),
          content: Form(
            key: formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Table ${_short(t.id)} • Buy-in ${(t.buyInAtoms / 1e8).toStringAsFixed(4)} DCR'),
                const SizedBox(height: 12),
                if (escrows.isNotEmpty)
                  DropdownButtonFormField<String>(
                    value: selectedOutpoint,
                    decoration: const InputDecoration(labelText: 'Choose funding outpoint'),
                    items: escrows
                        .map((e) => DropdownMenuItem(
                              value: '${e['funding_txid'] ?? ''}:${(e['funding_vout'] ?? 0).toString()}',
                              child: Text('${_short(e['funding_txid'] ?? '')}:${e['funding_vout'] ?? 0} • ${(e['funded_amount'] ?? 0) / 1e8} DCR'),
                            ))
                        .toList(),
                    onChanged: (v) => selectedOutpoint = v,
                  ),
                TextFormField(
                  controller: escrowCtrl,
                  decoration: const InputDecoration(labelText: 'Outpoint txid:vout (override)'),
                  validator: (v) {
                    final chosen = (selectedOutpoint ?? '').trim().isNotEmpty ? selectedOutpoint : v?.trim();
                    return (chosen == null || chosen.isEmpty) ? 'Outpoint required' : null;
                  },
                ),
                DropdownButtonFormField<int>(
                  value: seatIndex,
                  decoration: const InputDecoration(labelText: 'Seat'),
                  items: List.generate(t.maxPlayers, (i) => DropdownMenuItem(value: i, child: Text('Seat $i'))),
                  onChanged: (v) => seatIndex = v ?? seatIndex,
                ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(dctx), child: const Text('Cancel')),
            ElevatedButton(
              onPressed: () async {
                if (!(formKey.currentState?.validate() ?? false)) return;
                Navigator.pop(dctx);
                await model.bindEscrow(
                  tableId: t.id,
                  seatIndex: seatIndex,
                  outpoint: (escrowCtrl.text.trim().isNotEmpty ? escrowCtrl.text.trim() : selectedOutpoint) ?? '',
                );
              },
              child: const Text('Bind'),
            ),
          ],
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final tableId = model.currentTableId;
    final table = model.tables.firstWhere(
      (t) => t.id == tableId,
      orElse: () => UiTable(
        id: tableId ?? '',
        hostId: '',
        players: const [],
        smallBlind: 0,
        bigBlind: 0,
        maxPlayers: 0,
        minPlayers: 0,
        currentPlayers: 0,
        minBalanceAtoms: 0,
        buyInAtoms: 0,
        phase: model.game?.phase ?? pr.GamePhase.WAITING,
        gameStarted: model.game?.gameStarted ?? false,
        allReady: false,
      ),
    );
    final gamePlayers = model.game?.players ?? const <UiPlayer>[];

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 720),
          child: Card(
            color: const Color(0xFF1B1E2C),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Icon(Icons.table_restaurant, color: Colors.blue),
                      const SizedBox(width: 8),
                      Text('Table ${_short(table.id)}', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.white)),
                      const Spacer(),
                      Chip(
                        label: Text(model.iAmReady ? 'Ready' : 'Not Ready'),
                        backgroundColor: model.iAmReady ? Colors.green.shade700 : Colors.orange.shade700,
                        labelStyle: const TextStyle(color: Colors.white),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Blinds ${table.smallBlind}/${table.bigBlind} • Buy-in ${(table.buyInAtoms / 1e8).toStringAsFixed(4)} DCR',
                    style: const TextStyle(color: Colors.white70),
                  ),
                  const SizedBox(height: 12),
                  const Divider(color: Colors.white24),
                  const SizedBox(height: 8),
                  const Text('Players', style: TextStyle(color: Colors.white70, fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  if (gamePlayers.isEmpty)
                    const Text('Waiting for players...', style: TextStyle(color: Colors.white54))
                  else
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: gamePlayers.map((p) {
                        final escrowColor = p.escrowId.isEmpty
                            ? Colors.white30
                            : (p.escrowReady ? Colors.greenAccent : Colors.amberAccent);
                        return Container(
                          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                          decoration: BoxDecoration(
                            color: Colors.white10,
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(color: p.isReady ? Colors.greenAccent : Colors.orangeAccent.withOpacity(0.6)),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(p.isReady ? Icons.check_circle : Icons.hourglass_empty,
                                  size: 14, color: p.isReady ? Colors.greenAccent : Colors.orangeAccent),
                              const SizedBox(width: 6),
                              Text(_playerLabel(p), style: const TextStyle(color: Colors.white)),
                              if (p.id == model.playerId) ...[
                                const SizedBox(width: 6),
                                const Text('(you)', style: TextStyle(color: Colors.lightBlueAccent)),
                              ],
                              if (p.escrowId.isNotEmpty) ...[
                                const SizedBox(width: 8),
                                Icon(Icons.account_balance, size: 14, color: escrowColor),
                              ],
                            ],
                          ),
                        );
                      }).toList(),
                    ),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      ElevatedButton(
                        onPressed: model.iAmReady ? model.setUnready : model.setReady,
                        child: Text(model.iAmReady ? 'Unready' : 'Ready'),
                      ),
                      const Spacer(),
                      TextButton(
                        onPressed: model.leaveTable,
                        style: TextButton.styleFrom(foregroundColor: Colors.redAccent),
                        child: const Text('Leave Table'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Builder(builder: (ctx) {
                    final myEscrowId = model.cachedEscrowId;
                    final myEscrowReady = model.cachedEscrowReady;
                    if (myEscrowId.isNotEmpty) {
                      final tx = myEscrowId.length > 12 ? '${myEscrowId.substring(0, 8)}...' : myEscrowId;
                      return Align(
                        alignment: Alignment.centerLeft,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                          decoration: BoxDecoration(
                            color: myEscrowReady ? Colors.green.withOpacity(0.1) : Colors.amber.withOpacity(0.1),
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: myEscrowReady ? Colors.greenAccent : Colors.amberAccent),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(Icons.account_balance, size: 16, color: myEscrowReady ? Colors.greenAccent : Colors.amberAccent),
                              const SizedBox(width: 6),
                              Text('Escrow bound: $tx', style: const TextStyle(color: Colors.white)),
                            ],
                          ),
                        ),
                      );
                    }
                    return Align(
                      alignment: Alignment.centerLeft,
                      child: OutlinedButton(
                        onPressed: () => _showBindDialog(context, table),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: Colors.lightBlueAccent,
                          side: const BorderSide(color: Colors.lightBlueAccent),
                        ),
                        child: const Text('Bind Escrow'),
                      ),
                    );
                  }),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
