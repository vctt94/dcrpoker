import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class InLobbyView extends StatelessWidget {
  const InLobbyView({super.key, required this.model});
  final PokerModel model;

  String _short(String s, [int n = 8]) =>
      s.isEmpty ? '' : (s.length <= n ? s : s.substring(0, n));

  int _asInt(dynamic v) {
    if (v is int) return v;
    if (v is num) return v.toInt();
    if (v is String) return int.tryParse(v) ?? 0;
    return 0;
  }

  bool _escrowHasRequiredConfirmations(Map<String, dynamic> escrow) {
    final confs = _asInt(escrow['confs']);
    final required = _asInt(escrow['required_confirmations']);
    final requiredOrDefault = required == 0 ? 1 : required;
    return confs >= requiredOrDefault;
  }

  String _playerLabel(UiPlayer p) {
    final name = p.name.trim();
    if (name.isNotEmpty) return name;
    return _short(p.id, 10);
  }

  Future<void> _showLeaveTableDialog(BuildContext ctx) async {
    if (!ctx.mounted) return;
    final confirmed = await showDialog<bool>(
      context: ctx,
      builder: (dctx) => AlertDialog(
        title: const Text('Leave Table?'),
        content: const Text(
            'Are you sure you want to leave this table? You will need to rejoin if you want to play again.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dctx, false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.pop(dctx, true),
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.redAccent,
            ),
            child: const Text('Leave Table'),
          ),
        ],
      ),
    );
    if (confirmed == true && ctx.mounted) {
      await model.leaveTable();
    }
  }

  Future<void> _showBindDialog(BuildContext ctx, UiTable t) async {
    if (!model.hasAuthedPayoutAddress) {
      if (!ctx.mounted) return;
      await showDialog(
        context: ctx,
        builder: (dctx) => AlertDialog(
          title: const Text('Sign Address Required'),
          content: const Text(
              'Bind escrow needs a verified payout address. Please sign an address before binding or opening a new escrow.'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(dctx),
              child: const Text('Not now'),
            ),
            ElevatedButton(
              onPressed: () {
                Navigator.pop(dctx);
                Navigator.pushNamed(ctx, '/sign-address');
              },
              child: const Text('Go to Sign Address'),
            ),
          ],
        ),
      );
      return;
    }
    final escrows = await model.listCachedEscrows();
    final escrowOptions = escrows.where((e) {
      // Filter out invalid escrows
      final fundingState = (e['funding_state'] ?? '').toString().toUpperCase();
      return fundingState != 'ESCROW_STATE_INVALID';
    }).map((e) {
      final txid = (e['funding_txid'] ?? '').toString();
      final vout = _asInt(e['funding_vout']);
      final amountRaw = e['funded_amount'];
      final amount = amountRaw is num
          ? amountRaw.toDouble()
          : double.tryParse(amountRaw.toString()) ?? 0;
      final outpoint = '$txid:$vout';
      final confirmed = _escrowHasRequiredConfirmations(e);
      return {
        'outpoint': outpoint,
        'label':
            '${_short(txid)}:$vout • ${(amount / 1e8).toStringAsFixed(4)} DCR',
        'confirmed': confirmed,
      };
    }).toList();
    if (escrows.isEmpty) {
      if (!ctx.mounted) return;
      await showDialog(
        context: ctx,
        builder: (dctx) => AlertDialog(
          title: const Text('No Escrows Available'),
          content: const Text(
              'You need to open and fund an escrow before you can bind it to this table.'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(dctx),
              child: const Text('Not now'),
            ),
            ElevatedButton(
              onPressed: () {
                Navigator.pop(dctx);
                Navigator.pushNamed(ctx, '/open-escrow');
              },
              child: const Text('Open Escrow'),
            ),
          ],
        ),
      );
      return;
    }
    final escrowCtrl = TextEditingController();
    String? selectedOutpoint;
    for (final opt in escrowOptions) {
      if (opt['confirmed'] == true) {
        selectedOutpoint = opt['outpoint'] as String;
        break;
      }
    }
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
                Text(
                    'Table ${_short(t.id)} • Buy-in ${(t.buyInAtoms / 1e8).toStringAsFixed(4)} DCR'),
                const SizedBox(height: 12),
                if (escrows.isNotEmpty)
                  DropdownButtonFormField<String>(
                    value: selectedOutpoint,
                    decoration: const InputDecoration(
                        labelText: 'Choose funding outpoint'),
                    items: escrowOptions
                        .map(
                          (opt) => DropdownMenuItem<String>(
                            value: opt['outpoint'] as String,
                            enabled: opt['confirmed'] == true,
                            child: opt['confirmed'] == true
                                ? Text(opt['label'] as String)
                                : Tooltip(
                                    message: 'waiting for confirmation',
                                    child: Text(
                                      '${opt['label']} (pending)',
                                      style: const TextStyle(
                                          color: Colors.white54),
                                    ),
                                  ),
                          ),
                        )
                        .toList(),
                    onChanged: (v) => selectedOutpoint = v,
                  ),
                TextFormField(
                  controller: escrowCtrl,
                  decoration: const InputDecoration(
                      labelText: 'Outpoint txid:vout (override)'),
                  validator: (v) {
                    final chosen = (selectedOutpoint ?? '').trim().isNotEmpty
                        ? selectedOutpoint
                        : v?.trim();
                    return (chosen == null || chosen.isEmpty)
                        ? 'Outpoint required'
                        : null;
                  },
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(dctx),
                child: const Text('Cancel')),
            ElevatedButton(
              onPressed: () async {
                if (!(formKey.currentState?.validate() ?? false)) return;
                Navigator.pop(dctx);
                await model.bindEscrow(
                  tableId: t.id,
                  outpoint: (escrowCtrl.text.trim().isNotEmpty
                          ? escrowCtrl.text.trim()
                          : selectedOutpoint) ??
                      '',
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
        buyInAtoms: 0,
        phase: model.game?.phase ?? pr.GamePhase.WAITING,
        gameStarted: model.game?.gameStarted ?? false,
        allReady: false,
      ),
    );
    final gamePlayers = model.game?.players ?? const <UiPlayer>[];
    final lobbyPlayers = table.players;
    final displayedPlayers =
        gamePlayers.isNotEmpty ? gamePlayers : lobbyPlayers;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 720),
          child: Card(
            color: const Color(0xFF1B1E2C),
            shape:
                RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Icon(Icons.table_restaurant, color: Colors.blue),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text('Table ${_short(table.id)}',
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                                fontSize: 20,
                                fontWeight: FontWeight.bold,
                                color: Colors.white)),
                      ),
                      const SizedBox(width: 8),
                      ElevatedButton(
                        onPressed:
                            model.iAmReady ? model.setUnready : model.setReady,
                        child: Text(model.iAmReady ? 'Unready' : 'Ready'),
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
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    alignment: WrapAlignment.spaceBetween,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      const Text('Players',
                          style: TextStyle(
                              color: Colors.white70,
                              fontWeight: FontWeight.bold)),
                      Chip(
                        label: Text(model.iAmReady ? 'Ready' : 'Not Ready'),
                        backgroundColor: model.iAmReady
                            ? Colors.green.shade700
                            : Colors.orange.shade700,
                        labelStyle: const TextStyle(color: Colors.white),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  if (displayedPlayers.isEmpty)
                    const Text('Waiting for players...',
                        style: TextStyle(color: Colors.white54))
                  else
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: displayedPlayers
                          .map((p) => _buildPlayerPill(p, model.playerId))
                          .toList(),
                    ),
                  // Error message if exists
                  if (model.errorMessage.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Card(
                      color: Colors.red.shade800,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Padding(
                        padding: const EdgeInsets.all(12.0),
                        child: Row(
                          children: [
                            const Icon(Icons.error, color: Colors.white),
                            const SizedBox(width: 8),
                            Expanded(
                              child: SelectableText(
                                model.errorMessage,
                                style: const TextStyle(color: Colors.white),
                              ),
                            ),
                            Material(
                              color: Colors.transparent,
                              child: InkWell(
                                onTap: () async {
                                  await Clipboard.setData(
                                      ClipboardData(text: model.errorMessage));
                                  if (!context.mounted) return;
                                  ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(
                                          content: Text(
                                              'Error copied to clipboard')));
                                },
                                borderRadius: BorderRadius.circular(20),
                                child: const Padding(
                                  padding: EdgeInsets.all(8.0),
                                  child: Icon(Icons.copy,
                                      color: Colors.white, size: 20),
                                ),
                              ),
                            ),
                            Material(
                              color: Colors.transparent,
                              child: InkWell(
                                onTap: () {
                                  model.clearError();
                                },
                                borderRadius: BorderRadius.circular(20),
                                child: const Padding(
                                  padding: EdgeInsets.all(8.0),
                                  child: Icon(Icons.close,
                                      color: Colors.white, size: 20),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                  const SizedBox(height: 12),
                  // Escrow state panel
                  _buildEscrowStatePanel(context, table, model),
                  const SizedBox(height: 12),
                  // Game start status
                  if (table.buyInAtoms > 0)
                    _buildGameStartStatus(model, displayedPlayers),
                  const SizedBox(height: 16),
                  // Leave Table button at the bottom
                  Row(
                    children: [
                      const Spacer(),
                      TextButton(
                        onPressed: () => _showLeaveTableDialog(context),
                        style: TextButton.styleFrom(
                            foregroundColor: Colors.redAccent),
                        child: const Text('Leave Table'),
                      ),
                    ],
                  ),
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
            color: p.isReady
                ? Colors.greenAccent
                : Colors.orangeAccent.withOpacity(0.6),
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
              const Text('(you)',
                  style:
                      TextStyle(color: Colors.lightBlueAccent, fontSize: 12)),
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

  Widget _buildMiniIndicator(
      {required IconData icon, required Color color, required bool filled}) {
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
    if (p.escrowState.isNotEmpty) {
      parts.add('State: ${_friendlyEscrowState(p.escrowState)}');
    }
    parts.add(p.presignComplete ? '✓ Presigned' : '○ Not presigned');
    return parts.join('\n');
  }

  Widget _buildEscrowStatePanel(
      BuildContext context, UiTable table, PokerModel model) {
    final myEscrowId = model.cachedEscrowId;
    final myEscrowReady = model.cachedEscrowReady;
    final myEscrowState = model.cachedEscrowState;
    final presignCompleted = model.presignCompleted;
    final canChangeEscrow =
        !table.gameStarted && !presignCompleted && !model.presignInProgress;

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
                Icon(Icons.warning_amber,
                    size: 20, color: Colors.orange.shade400),
                const SizedBox(width: 8),
                Text('Escrow Required',
                    style: TextStyle(
                        color: Colors.orange.shade400,
                        fontWeight: FontWeight.bold)),
              ],
            ),
            const SizedBox(height: 8),
            const Text('Bind an escrow to participate in this table.',
                style: TextStyle(color: Colors.white70, fontSize: 13)),
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
    final escrowShort = myEscrowId.length > 12
        ? '${myEscrowId.substring(0, 8)}...'
        : myEscrowId;

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
          const Text('Settlement Status',
              style: TextStyle(
                  color: Colors.white70,
                  fontWeight: FontWeight.bold,
                  fontSize: 13)),
          const SizedBox(height: 12),
          // Escrow status row
          _buildStatusRow(
            icon: Icons.account_balance,
            label: 'Escrow',
            value: escrowShort,
            status: myEscrowReady ? 'Funded' : 'Pending',
            statusColor:
                myEscrowReady ? Colors.greenAccent : Colors.amberAccent,
          ),
          if (myEscrowState.isNotEmpty) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(Icons.timeline, size: 16, color: Colors.white54),
                const SizedBox(width: 8),
                const Text('State: ',
                    style: TextStyle(color: Colors.white54, fontSize: 13)),
                Text(
                  _friendlyEscrowState(myEscrowState),
                  style: TextStyle(
                    color: _escrowStateColor(myEscrowState),
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
          ],
          const SizedBox(height: 8),
          // Presign status row (handled automatically by golib)
          _buildStatusRow(
            icon: Icons.draw,
            label: 'Presign',
            value: presignCompleted ? 'Complete' : 'Waiting',
            status: presignCompleted ? '✓' : '○',
            statusColor: presignCompleted ? Colors.greenAccent : Colors.white54,
          ),
          if (canChangeEscrow) ...[
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: () => _showBindDialog(context, table),
              icon: const Icon(Icons.swap_horiz, size: 16),
              label: const Text('Change Escrow'),
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.lightBlueAccent,
                side: const BorderSide(color: Colors.lightBlueAccent),
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
        Text('$label: ',
            style: const TextStyle(color: Colors.white54, fontSize: 13)),
        Expanded(
          child: Text(value,
              style: const TextStyle(color: Colors.white, fontSize: 13)),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: statusColor.withOpacity(0.15),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(status,
              style: TextStyle(
                  color: statusColor,
                  fontSize: 12,
                  fontWeight: FontWeight.bold)),
        ),
      ],
    );
  }

  Widget _buildGameStartStatus(PokerModel model, List<UiPlayer> players) {
    // Calculate what's blocking game start
    final minPlayers = 2;
    final hasEnoughPlayers = players.length >= minPlayers;
    final allReady = players.every((p) => p.isReady);
    final allEscrowsFunded =
        players.every((p) => p.escrowId.isNotEmpty && p.escrowReady);
    final allPresigned = players.every((p) => p.presignComplete);

    final readyCount = players.where((p) => p.isReady).length;
    final escrowCount =
        players.where((p) => p.escrowId.isNotEmpty && p.escrowReady).length;
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
            child: Text(statusMessage,
                style: TextStyle(color: statusColor, fontSize: 13)),
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

  String _friendlyEscrowState(String state) {
    switch (state) {
      case 'ESCROW_STATE_READY':
        return 'Ready';
      case 'ESCROW_STATE_MEMPOOL':
        return 'Mempool';
      case 'ESCROW_STATE_CONFIRMING':
        return 'Confirming';
      case 'ESCROW_STATE_CSV_MATURED':
        return 'CSV matured';
      case 'ESCROW_STATE_SPENT':
        return 'Spent';
      case 'ESCROW_STATE_INVALID':
        return 'Invalid';
      case 'ESCROW_STATE_UNFUNDED':
        return 'Unfunded';
      default:
        return state;
    }
  }

  Color _escrowStateColor(String state) {
    switch (state) {
      case 'ESCROW_STATE_READY':
        return Colors.greenAccent;
      case 'ESCROW_STATE_MEMPOOL':
      case 'ESCROW_STATE_CONFIRMING':
        return Colors.amberAccent;
      case 'ESCROW_STATE_CSV_MATURED':
      case 'ESCROW_STATE_SPENT':
      case 'ESCROW_STATE_INVALID':
        return Colors.redAccent;
      default:
        return Colors.white54;
    }
  }
}
