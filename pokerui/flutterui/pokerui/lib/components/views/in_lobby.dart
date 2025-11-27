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
                      children: gamePlayers.map((p) => _buildPlayerPill(p, model.playerId)).toList(),
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
                  // Escrow state panel
                  _buildEscrowStatePanel(context, table, model),
                  const SizedBox(height: 12),
                  // Game start status
                  if (table.buyInAtoms > 0) _buildGameStartStatus(model, gamePlayers),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildPlayerPill(UiPlayer p, String myPlayerId) {
    final isMe = p.id == myPlayerId;
    final escrowColor = p.escrowId.isEmpty
        ? Colors.white30
        : (p.escrowReady ? Colors.greenAccent : Colors.amberAccent);
    final presignColor = p.presignComplete ? Colors.cyanAccent : Colors.white30;

    return Tooltip(
      message: _getPlayerTooltip(p),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: Colors.white10,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: p.isReady ? Colors.greenAccent : Colors.orangeAccent.withOpacity(0.6),
            width: isMe ? 2 : 1,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Ready status icon
            Icon(
              p.isReady ? Icons.check_circle : Icons.hourglass_empty,
              size: 14,
              color: p.isReady ? Colors.greenAccent : Colors.orangeAccent,
            ),
            const SizedBox(width: 6),
            Text(_playerLabel(p), style: const TextStyle(color: Colors.white)),
            if (isMe) ...[
              const SizedBox(width: 6),
              const Text('(you)', style: TextStyle(color: Colors.lightBlueAccent, fontSize: 12)),
            ],
            // Status indicators
            const SizedBox(width: 8),
            // Escrow indicator
            _buildMiniIndicator(
              icon: Icons.account_balance,
              color: escrowColor,
              filled: p.escrowId.isNotEmpty && p.escrowReady,
            ),
            const SizedBox(width: 4),
            // Presign indicator  
            _buildMiniIndicator(
              icon: Icons.draw,
              color: presignColor,
              filled: p.presignComplete,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMiniIndicator({required IconData icon, required Color color, required bool filled}) {
    return Container(
      width: 20,
      height: 20,
      decoration: BoxDecoration(
        color: filled ? color.withOpacity(0.2) : Colors.transparent,
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withOpacity(0.5), width: 1),
      ),
      child: Icon(icon, size: 12, color: color),
    );
  }

  String _getPlayerTooltip(UiPlayer p) {
    final parts = <String>[];
    parts.add(p.isReady ? '✓ Ready' : '○ Not ready');
    if (p.escrowId.isEmpty) {
      parts.add('○ No escrow');
    } else if (p.escrowReady) {
      parts.add('✓ Escrow funded');
    } else {
      parts.add('⏳ Escrow pending');
    }
    parts.add(p.presignComplete ? '✓ Presigned' : '○ Not presigned');
    return parts.join('\n');
  }

  Widget _buildEscrowStatePanel(BuildContext context, UiTable table, PokerModel model) {
    final myEscrowId = model.cachedEscrowId;
    final myEscrowReady = model.cachedEscrowReady;
    final presignInProgress = model.presignInProgress;
    final presignCompleted = model.presignCompleted;
    final presignError = model.presignError;

    if (myEscrowId.isEmpty) {
      // No escrow bound
      return Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Colors.grey.shade900,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: Colors.grey.shade700),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.warning_amber, size: 20, color: Colors.orange.shade400),
                const SizedBox(width: 8),
                Text('Escrow Required', style: TextStyle(color: Colors.orange.shade400, fontWeight: FontWeight.bold)),
              ],
            ),
            const SizedBox(height: 8),
            const Text('Bind an escrow to participate in this table.', style: TextStyle(color: Colors.white70, fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: () => _showBindDialog(context, table),
              icon: const Icon(Icons.link, size: 16),
              label: const Text('Bind Escrow'),
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.lightBlueAccent,
                side: const BorderSide(color: Colors.lightBlueAccent),
              ),
            ),
          ],
        ),
      );
    }

    // Escrow is bound - show detailed status
    final escrowShort = myEscrowId.length > 12 ? '${myEscrowId.substring(0, 8)}...' : myEscrowId;
    
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.grey.shade900,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.grey.shade700),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Settlement Status', style: TextStyle(color: Colors.white70, fontWeight: FontWeight.bold, fontSize: 13)),
          const SizedBox(height: 12),
          // Escrow status row
          _buildStatusRow(
            icon: Icons.account_balance,
            label: 'Escrow',
            value: escrowShort,
            status: myEscrowReady ? 'Funded' : 'Pending',
            statusColor: myEscrowReady ? Colors.greenAccent : Colors.amberAccent,
          ),
          const SizedBox(height: 8),
          // Presign status row
          _buildStatusRow(
            icon: Icons.draw,
            label: 'Presign',
            value: presignInProgress ? 'In progress...' : (presignCompleted ? 'Complete' : 'Waiting'),
            status: presignCompleted ? '✓' : (presignInProgress ? '⏳' : '○'),
            statusColor: presignCompleted ? Colors.greenAccent : (presignInProgress ? Colors.lightBlueAccent : Colors.white54),
          ),
          if (presignError.isNotEmpty) ...[
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: Colors.red.shade900.withOpacity(0.3),
                borderRadius: BorderRadius.circular(6),
              ),
              child: Row(
                children: [
                  Icon(Icons.error_outline, size: 16, color: Colors.red.shade300),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      presignError,
                      style: TextStyle(color: Colors.red.shade300, fontSize: 12),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildStatusRow({
    required IconData icon,
    required String label,
    required String value,
    required String status,
    required Color statusColor,
  }) {
    return Row(
      children: [
        Icon(icon, size: 16, color: Colors.white54),
        const SizedBox(width: 8),
        Text('$label: ', style: const TextStyle(color: Colors.white54, fontSize: 13)),
        Expanded(
          child: Text(value, style: const TextStyle(color: Colors.white, fontSize: 13)),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: statusColor.withOpacity(0.15),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(status, style: TextStyle(color: statusColor, fontSize: 12, fontWeight: FontWeight.bold)),
        ),
      ],
    );
  }

  Widget _buildGameStartStatus(PokerModel model, List<UiPlayer> players) {
    // Calculate what's blocking game start
    final minPlayers = 2;
    final hasEnoughPlayers = players.length >= minPlayers;
    final allReady = players.every((p) => p.isReady);
    final allEscrowsFunded = players.every((p) => p.escrowId.isNotEmpty && p.escrowReady);
    final allPresigned = players.every((p) => p.presignComplete);

    final readyCount = players.where((p) => p.isReady).length;
    final escrowCount = players.where((p) => p.escrowId.isNotEmpty && p.escrowReady).length;
    final presignCount = players.where((p) => p.presignComplete).length;
    final totalPlayers = players.length;

    // Determine overall status
    String statusMessage;
    Color statusColor;
    IconData statusIcon;

    if (!hasEnoughPlayers) {
      statusMessage = 'Waiting for players ($totalPlayers/$minPlayers)';
      statusColor = Colors.grey;
      statusIcon = Icons.people_outline;
    } else if (!allEscrowsFunded) {
      statusMessage = 'Waiting for escrows ($escrowCount/$totalPlayers funded)';
      statusColor = Colors.amber;
      statusIcon = Icons.account_balance_outlined;
    } else if (!allReady) {
      statusMessage = 'Waiting for ready ($readyCount/$totalPlayers ready)';
      statusColor = Colors.orange;
      statusIcon = Icons.hourglass_empty;
    } else if (!allPresigned) {
      statusMessage = 'Presigning in progress ($presignCount/$totalPlayers)';
      statusColor = Colors.lightBlue;
      statusIcon = Icons.sync;
    } else {
      statusMessage = 'Ready to start!';
      statusColor = Colors.greenAccent;
      statusIcon = Icons.check_circle;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: statusColor.withOpacity(0.1),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: statusColor.withOpacity(0.5)),
      ),
      child: Row(
        children: [
          Icon(statusIcon, size: 18, color: statusColor),
          const SizedBox(width: 10),
          Expanded(
            child: Text(statusMessage, style: TextStyle(color: statusColor, fontSize: 13)),
          ),
          // Progress indicator
          if (!allPresigned && allReady && allEscrowsFunded)
            SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                valueColor: AlwaysStoppedAnimation<Color>(statusColor),
              ),
            ),
        ],
      ),
    );
  }
}
