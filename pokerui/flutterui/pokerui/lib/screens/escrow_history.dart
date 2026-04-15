import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:path/path.dart' as p;
import 'package:provider/provider.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/util.dart';

class EscrowHistoryScreen extends StatefulWidget {
  const EscrowHistoryScreen({super.key});

  @override
  State<EscrowHistoryScreen> createState() => _EscrowHistoryScreenState();
}

class _EscrowHistoryScreenState extends State<EscrowHistoryScreen> {
  bool _loading = false;
  String? _error;
  List<_EscrowEntry> _entries = const [];
  StreamSubscription<pr.Notification>? _ntfnSub;
  final TextEditingController _deleteConfirmCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    _ntfnSub = Golib.pokerNotifications().listen(
      _handleNotification,
      onError: (_) {},
    );
    _refresh();
  }

  @override
  void dispose() {
    _ntfnSub?.cancel();
    _deleteConfirmCtrl.dispose();
    super.dispose();
  }

  Future<String> _historyDir() async {
    String dataDir = '';
    try {
      final model = Provider.of<PokerModel>(context, listen: false);
      dataDir = model.dataDir;
    } catch (_) {}
    if (dataDir.trim().isEmpty) {
      dataDir = await defaultAppDataDir();
    }
    return p.join(dataDir, 'history_session');
  }

  Future<void> _refresh() async {
    setState(() {
      _loading = true;
      _error = null;
      _entries = const [];
    });
    try {
      final entries = await _loadEntries();
      if (!mounted) return;
      setState(() {
        _entries = entries;
      });
      await _refreshStatuses(entries);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  Future<List<_EscrowEntry>> _loadEntries() async {
    final dirPath = await _historyDir();
    final dir = Directory(dirPath);
    if (!await dir.exists()) return [];

    final files = await dir
        .list()
        .where((entity) => entity is File && entity.path.endsWith('.json'))
        .cast<File>()
        .toList();

    final entries = <_EscrowEntry>[];
    for (final f in files) {
      try {
        final contents = await f.readAsString();
        final decoded = jsonDecode(contents);
        if (decoded is! Map) continue;
        final parsed = _parseEntry(f, Map<String, dynamic>.from(decoded));
        if (parsed != null) {
          entries.add(parsed);
        }
      } catch (_) {
        // Ignore malformed files; keep screen resilient.
      }
    }
    entries.sort((a, b) => (b.archivedAt ?? 0).compareTo(a.archivedAt ?? 0));
    return entries;
  }

  _EscrowEntry? _parseEntry(File file, Map<String, dynamic> data) {
    Map<String, dynamic>? escrow;
    String? matchId;
    if (data['escrow_info'] is Map) {
      escrow = Map<String, dynamic>.from(data['escrow_info']);
      matchId = _asString(data['match_id']);
    } else {
      escrow = data;
    }
    if (escrow == null) return null;

    final escrowId = _asString(escrow['escrow_id']);
    if (escrowId.isEmpty) return null;

    int? archived = _asInt(escrow['archived_at']);
    archived ??= file.statSync().modified.millisecondsSinceEpoch ~/ 1000;

    final confirmedConfs =
        _asInt(escrow['confirmed_height']) ?? _asInt(escrow['confs']);

    return _EscrowEntry(
      escrowId: escrowId,
      matchId: matchId,
      archivedAt: archived,
      csvBlocks: _asInt(escrow['csv_blocks']),
      fundedAmount: _asInt(escrow['funded_amount']),
      depositAddress: _asString(escrow['deposit_address']),
      fundingTxid: _asString(escrow['funding_txid']),
      fundingVout: _asInt(escrow['funding_vout']),
      status: _asString(escrow['status']),
      sourceFile: file.path,
      confirmedConfs: confirmedConfs,
    );
  }

  Future<void> _refreshStatuses(List<_EscrowEntry> entries) async {
    for (final entry in entries) {
      try {
        final res = await Golib.getEscrowStatus(entry.escrowId);
        if (!mounted) return;
        setState(() {
          entry.liveStatus = _EscrowLiveStatus.fromJson(res);
          entry.statusError = null;
        });
      } catch (e) {
        if (!mounted) return;
        final msg = e.toString();
        setState(() {
          // Older escrows may no longer be known by the referee after
          // restarts or pruning. Treat "escrow not found" as a benign
          // condition instead of surfacing a noisy RPC error.
          if (msg.contains('escrow not found') ||
              msg.contains('code = NotFound')) {
            entry.statusError = null;
          } else {
            entry.statusError = msg;
          }
        });
      }
    }
  }

  String _stateLabel(_EscrowEntry e) {
    final live = e.liveStatus;

    if (live != null) {
      // Check server-provided state first (most authoritative)
      final fState = (live.fundingState ?? '').toLowerCase();
      if (fState.isNotEmpty) {
        switch (fState) {
          case 'escrow_state_unfunded':
            return 'Not funded';
          case 'escrow_state_mempool':
            return 'Seen in mempool';
          case 'escrow_state_confirming':
            return 'Waiting for confirmations';
          case 'escrow_state_invalid':
            return 'Invalid funding';
          case 'escrow_state_spent':
            return 'Spent';
          case 'escrow_state_csv_matured':
            return 'Expired';
          case 'escrow_state_ready':
            return 'Locked';
        }
      }
      // Check server-provided matureForCsv flag
      if (live.matureForCsv) return 'Expired';

      // Fall back to derived state from live data
      if (live.utxoCount == 0 || (live.fundingTxid ?? '').isEmpty)
        return 'Not funded';
      if (live.utxoCount > 1) return 'Invalid funding';
      if (live.confs == 0) return 'Seen in mempool';
      final required = live.requiredConfs ?? 0;
      if (required > 0 && live.confs < required)
        return 'Waiting for confirmations';
      return 'Locked';
    }

    // No live status - check if CSV expired using cached data
    if (_csvExpired(e)) return 'Expired';

    final cachedStatus = e.status?.trim() ?? '';
    if (cachedStatus.isNotEmpty) {
      return cachedStatus[0].toUpperCase() + cachedStatus.substring(1);
    }
    return 'Not funded';
  }

  Color _stateColor(_EscrowEntry e) {
    final live = e.liveStatus;

    if (live != null) {
      // Check server-provided state first (most authoritative)
      final fState = (live.fundingState ?? '').toLowerCase();
      if (fState.isNotEmpty) {
        switch (fState) {
          case 'escrow_state_invalid':
          case 'escrow_state_spent':
          case 'escrow_state_csv_matured':
            return Colors.redAccent;
          case 'escrow_state_ready':
            return Colors.greenAccent;
          case 'escrow_state_unfunded':
            return Colors.white70;
          case 'escrow_state_mempool':
          case 'escrow_state_confirming':
            return Colors.orangeAccent;
        }
      }

      // Check server-provided matureForCsv flag
      if (live.matureForCsv) return Colors.redAccent;

      // Check if escrow is confirmed/ready based on confirmations
      if (live.utxoCount == 1 && live.confs > 0) {
        final required = live.requiredConfs ?? 0;
        if (required == 0 || live.confs >= required) {
          return Colors.greenAccent;
        }
      }
      // Invalid funding if multiple UTXOs
      if (live.utxoCount > 1) return Colors.redAccent;
      return Colors.orangeAccent;
    }

    // No live status - check if CSV expired using cached data
    if (_csvExpired(e)) return Colors.redAccent;
    return Colors.white70;
  }

  bool _isRefundable(_EscrowEntry e) {
    final live = e.liveStatus;

    // Prefer server-provided matureForCsv flag
    if (live != null && live.matureForCsv) return true;

    // Fall back to client-side calculation if no live status
    if (live == null && _csvExpired(e)) return true;

    // Also check using confirmations
    final csv = e.csvBlocks ?? live?.csvBlocks ?? 0;
    if (csv <= 0) return false;
    final confs = e.confirmedConfs ?? live?.confs ?? 0;
    return confs >= csv;
  }

  bool _csvExpired(_EscrowEntry e) {
    final csv = e.csvBlocks ?? e.liveStatus?.csvBlocks ?? 0;
    if (csv <= 0) return false;
    final fundingHeight = e.confirmedConfs;
    final bestHeight = e.liveStatus?.bestHeight;
    if (fundingHeight != null &&
        fundingHeight > 0 &&
        bestHeight != null &&
        bestHeight > 0 &&
        bestHeight - fundingHeight >= csv) {
      return true;
    }
    final liveConfs = e.liveStatus?.confs ?? 0;
    return liveConfs >= csv;
  }

  Future<void> _openRefundDialog(_EscrowEntry e) async {
    Map<String, dynamic>? escrow;
    try {
      final file = File(e.sourceFile);
      final contents = await file.readAsString();
      final decoded = jsonDecode(contents);
      if (decoded is Map<String, dynamic>) {
        if (decoded['escrow_info'] is Map) {
          escrow = Map<String, dynamic>.from(decoded['escrow_info'] as Map);
        } else {
          escrow = Map<String, dynamic>.from(decoded);
        }
      } else if (decoded is Map) {
        escrow = Map<String, dynamic>.from(decoded.cast<String, dynamic>());
      }
    } catch (_) {
      // Fall back to minimal info from entry if file cannot be read.
    }
    escrow ??= <String, dynamic>{
      'escrow_id': e.escrowId,
      'funding_txid': e.fundingTxid,
      'funding_vout': e.fundingVout,
      'funded_amount': e.fundedAmount,
      'csv_blocks': e.csvBlocks,
      'archived_at': e.archivedAt,
      'status': e.status,
    };
    if (!mounted) return;
    await showDialog(
      context: context,
      barrierDismissible: false,
      builder: (dialogContext) => RefundEscrowDialog(
        escrow: escrow!,
        onDelete: (refundDialogContext) async {
          await _confirmDeleteEscrow(e.escrowId, refundDialogContext);
          if (dialogContext.mounted) {
            Navigator.of(dialogContext).pop();
          }
        },
      ),
    );
  }

  Future<void> _confirmDeleteEscrow(
      String escrowId, BuildContext? dialogContext) async {
    _deleteConfirmCtrl.clear();
    final confirmContext = dialogContext ?? context;
    final confirm = await showDialog<bool>(
      context: confirmContext,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Delete escrow record?'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Are you absolutely sure? Deleting this escrow entry removes the '
              'local record used for refunds. If the refund has not been '
              'recovered yet, the funds may be PERMANENTLY LOST.',
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _deleteConfirmCtrl,
              decoration: const InputDecoration(
                labelText: 'Type OK to confirm',
              ),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) {
                final ok = _deleteConfirmCtrl.text.trim().toLowerCase() == 'ok';
                Navigator.of(dialogContext).pop(ok);
              },
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              final ok = _deleteConfirmCtrl.text.trim().toLowerCase() == 'ok';
              Navigator.of(dialogContext).pop(ok);
            },
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirm != true) {
      return;
    }

    final messenger = ScaffoldMessenger.of(context);
    try {
      final model = Provider.of<PokerModel>(context, listen: false);
      await model.deleteHistoricEscrow(escrowId);
      if (!mounted) return;
      await _refresh();
      messenger.showSnackBar(
        const SnackBar(content: Text('Escrow entry deleted.')),
      );
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('Failed to delete escrow: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Escrow History',
      child: RefreshIndicator(
        onRefresh: _refresh,
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Review locally stored escrows. Live status is fetched when possible to show CSV maturity and spend state.',
                style: TextStyle(color: Colors.white70),
              ),
              const SizedBox(height: 12),
              if (_error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: SelectableText(
                    _error!,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                ),
              if (_loading && _entries.isEmpty)
                const Center(
                  child: Padding(
                    padding: EdgeInsets.symmetric(vertical: 120),
                    child: CircularProgressIndicator(),
                  ),
                )
              else if (_entries.isEmpty)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(vertical: 80),
                  alignment: Alignment.center,
                  child: const Text(
                    'No escrow history found yet.',
                    style: TextStyle(color: Colors.white70),
                  ),
                )
              else
                ..._entries.map(_buildEntryCard),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildEntryCard(_EscrowEntry e) {
    final live = e.liveStatus;
    final amountAtoms = live?.amountAtoms ?? e.fundedAmount ?? 0;
    final amountDcr = amountAtoms / 1e8;
    final updated = (live?.updatedAt ?? e.archivedAt);
    final updatedAt = updated == null
        ? ''
        : DateTime.fromMillisecondsSinceEpoch(updated * 1000)
            .toLocal()
            .toString();
    final openedAt = e.archivedAt == null
        ? ''
        : DateTime.fromMillisecondsSinceEpoch(e.archivedAt! * 1000)
            .toLocal()
            .toString();

    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF1B1E2C),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Colors.blueAccent.withOpacity(.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      e.escrowId,
                      style: const TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    if (e.matchId != null && e.matchId!.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(
                          'Match: ${e.matchId}',
                          style: const TextStyle(color: Colors.white70),
                        ),
                      ),
                  ],
                ),
              ),
              IconButton(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: e.escrowId));
                  ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Escrow ID copied')));
                },
                icon: const Icon(Icons.copy, color: Colors.white70),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                decoration: BoxDecoration(
                  color: _stateColor(e).withOpacity(.15),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _stateColor(e).withOpacity(.6)),
                ),
                child: Text(
                  _stateLabel(e),
                  style: TextStyle(
                      color: _stateColor(e), fontWeight: FontWeight.bold),
                ),
              ),
              const SizedBox(width: 12),
              if (openedAt.isNotEmpty)
                Text(
                  'Opened: $openedAt',
                  style: const TextStyle(color: Colors.white70),
                ),
            ],
          ),
          const SizedBox(height: 8),
          _infoRow('Amount', '${amountDcr.toStringAsFixed(4)} DCR'),
          if (e.csvBlocks != null)
            _infoRow('CSV blocks', e.csvBlocks.toString()),
          if (live?.confs != null) _infoRow('Confs', live!.confs.toString()),
          if (live?.requiredConfs != null)
            _infoRow('Required confs', live!.requiredConfs.toString()),
          if (e.fundingTxid != null && e.fundingTxid!.isNotEmpty)
            _infoRow('Funding outpoint',
                '${e.fundingTxid}${e.fundingVout != null ? ':${e.fundingVout}' : ''}'),
          if (e.depositAddress != null && e.depositAddress!.isNotEmpty)
            _infoRow('Deposit address', e.depositAddress!),
          if (updatedAt.isNotEmpty) _infoRow('Updated', updatedAt),
          if (e.status != null && e.status!.isNotEmpty)
            _infoRow('Archived status', e.status!),
          if (e.statusError != null)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                'Status error: ${e.statusError}',
                style: const TextStyle(color: Colors.redAccent, fontSize: 12),
              ),
            ),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerRight,
            child: Wrap(
              spacing: 8,
              children: [
                TextButton.icon(
                  onPressed: () => _confirmDeleteEscrow(e.escrowId, null),
                  style: TextButton.styleFrom(
                    foregroundColor: Colors.redAccent,
                  ),
                  icon: const Icon(Icons.delete_outline, size: 18),
                  label: const Text('Delete'),
                ),
                TextButton.icon(
                  onPressed: () => _openRefundDialog(e),
                  icon: const Icon(Icons.currency_exchange, size: 18),
                  label: const Text('Review & Refund'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  void _handleNotification(pr.Notification n) {
    final msg = n.message;
    if (msg.isEmpty) return;
    dynamic decoded;
    try {
      decoded = jsonDecode(msg);
    } catch (_) {
      return;
    }
    if (decoded is! Map) return;
    final data = Map<String, dynamic>.from(decoded as Map);
    final kind = _asString(data['type']).toLowerCase();
    if (kind != 'escrow_funding') return;
    final escrowId = _asString(data['escrow_id']);
    if (escrowId.isEmpty) return;
    final idx = _entries.indexWhere((e) => e.escrowId == escrowId);
    if (idx == -1) return;

    setState(() {
      _entries[idx].liveStatus = _EscrowLiveStatus.fromJson(data);
      final err = _asString(data['error']);
      _entries[idx].statusError = err.isEmpty ? null : err;
    });
  }

  Widget _infoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 140,
            child: Text(
              label,
              style: const TextStyle(
                  color: Colors.white70, fontWeight: FontWeight.w600),
            ),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: const TextStyle(color: Colors.white),
            ),
          ),
        ],
      ),
    );
  }
}

class _EscrowEntry {
  final String escrowId;
  final String? matchId;
  final int? archivedAt;
  final int? csvBlocks;
  final int? fundedAmount;
  final String? depositAddress;
  final String? fundingTxid;
  final int? fundingVout;
  final String? status;
  final String sourceFile;
  final int? confirmedConfs;
  _EscrowLiveStatus? liveStatus;
  String? statusError;

  _EscrowEntry({
    required this.escrowId,
    required this.sourceFile,
    this.matchId,
    this.archivedAt,
    this.csvBlocks,
    this.fundedAmount,
    this.depositAddress,
    this.fundingTxid,
    this.fundingVout,
    this.status,
    this.confirmedConfs,
    this.liveStatus,
    this.statusError,
  });
}

class _EscrowLiveStatus {
  final int confs;
  final int utxoCount;
  final bool matureForCsv;
  final int? csvBlocks;
  final int? requiredConfs;
  final int? amountAtoms;
  final int? updatedAt;
  final String? fundingTxid;
  final int? fundingVout;
  // Optional server-provided hints for status/height.
  final String? fundingState;
  final int? bestHeight;

  _EscrowLiveStatus({
    required this.confs,
    required this.utxoCount,
    required this.matureForCsv,
    this.csvBlocks,
    this.requiredConfs,
    this.amountAtoms,
    this.updatedAt,
    this.fundingTxid,
    this.fundingVout,
    this.fundingState,
    this.bestHeight,
  });

  factory _EscrowLiveStatus.fromJson(Map<String, dynamic> json) {
    print('json: $json');
    return _EscrowLiveStatus(
      confs: _asInt(json['confs']) ?? 0,
      utxoCount: _asInt(json['utxo_count']) ?? 0,
      matureForCsv: json['mature_for_csv'] == true,
      csvBlocks: _asInt(json['csv_blocks']),
      requiredConfs: _asInt(json['required_confirmations']),
      amountAtoms: _asInt(json['amount_atoms']),
      updatedAt: _asInt(json['updated_at_unix']),
      fundingTxid: _asString(json['funding_txid']),
      fundingVout: _asInt(json['funding_vout']),
      fundingState: _asString(json['funding_state']),
      bestHeight: _asInt(json['best_height']) ?? _asInt(json['current_height']),
    );
  }
}

class RefundEscrowDialog extends StatefulWidget {
  const RefundEscrowDialog({
    super.key,
    required this.escrow,
    this.onDelete,
  });

  final Map<String, dynamic> escrow;
  final Future<void> Function(BuildContext)? onDelete;

  @override
  State<RefundEscrowDialog> createState() => _RefundEscrowDialogState();
}

class _RefundEscrowDialogState extends State<RefundEscrowDialog> {
  late Map<String, dynamic> _escrow;
  late TextEditingController _destAddressCtrl;
  late TextEditingController _csvBlocksCtrl;
  late TextEditingController _utxoValueCtrl;

  bool _isBuilding = false;
  bool _isUpdatingFunding = false;
  String? _statusMessage;
  bool _statusIsError = false;
  String? _refundTxHex;
  Map<String, dynamic>? _refundResult;

  @override
  void initState() {
    super.initState();
    _escrow = Map<String, dynamic>.from(widget.escrow);
    _destAddressCtrl = TextEditingController(text: '');
    final csvBlocks = _toInt(_escrow['csv_blocks']);
    _csvBlocksCtrl = TextEditingController(
      text: csvBlocks > 0 ? csvBlocks.toString() : '',
    );
    final storedAmount = _toInt(_escrow['funded_amount']);
    _utxoValueCtrl = TextEditingController(
      text: storedAmount > 0 ? formatDcrFromAtoms(storedAmount) : '',
    );

    // Auto-populate destination with configured payout address if available.
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted) return;
      try {
        final payout = await Golib.getPayoutAddress();
        if (!mounted) return;
        if (payout.trim().isNotEmpty && _destAddressCtrl.text.trim().isEmpty) {
          setState(() {
            _destAddressCtrl.text = payout.trim();
          });
        }
      } catch (_) {
        // Ignore errors; user can still fill manually.
      }
    });
  }

  @override
  void dispose() {
    _destAddressCtrl.dispose();
    _csvBlocksCtrl.dispose();
    _utxoValueCtrl.dispose();
    super.dispose();
  }

  String get _escrowId => _escrow['escrow_id']?.toString() ?? '';

  int? _atomsFromDcr(String value) {
    final cleaned = value.trim();
    if (cleaned.isEmpty) return null;
    final parsed = double.tryParse(cleaned);
    if (parsed == null) return null;
    return (parsed * 1e8).round();
  }

  Future<void> _handleFundingUpdate() async {
    final currentTxid = _escrow['funding_txid']?.toString() ?? '';
    final currentVout = _toInt(_escrow['funding_vout']);

    final txidController = TextEditingController(text: currentTxid);
    final voutController = TextEditingController(
      text: currentVout >= 0 ? currentVout.toString() : '',
    );

    if (!mounted) return;
    final result = await showDialog<Map<String, String>>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Funding Transaction'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Enter the transaction that funded this escrow. '
              'This information is required to build the refund transaction.',
              style: TextStyle(fontSize: 13),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: txidController,
              decoration: const InputDecoration(
                labelText: 'Funding transaction ID',
                hintText:
                    '0000000000000000000000000000000000000000000000000000000000000000',
              ),
              maxLines: 1,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: voutController,
              decoration: const InputDecoration(
                labelText: 'Output index (vout)',
                hintText: '0',
              ),
              keyboardType: TextInputType.number,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(dialogContext).pop({
              'txid': txidController.text.trim(),
              'vout': voutController.text.trim(),
            }),
            child: const Text('Save'),
          ),
        ],
      ),
    );

    if (!mounted || result == null) {
      return;
    }
    final txid = result['txid']?.trim() ?? '';
    final vout = int.tryParse(result['vout'] ?? '') ?? 0;
    if (txid.isEmpty) {
      if (!mounted) return;
      setState(() {
        _statusMessage = 'Funding transaction ID is required.';
        _statusIsError = true;
      });
      return;
    }

    if (!mounted) return;
    setState(() {
      _isUpdatingFunding = true;
      _statusMessage = null;
    });

    final model = context.read<PokerModel>();
    try {
      await model.updateEscrowFundingTx(_escrowId, txid, vout);
      if (!mounted) return;
      setState(() {
        _escrow['funding_txid'] = txid;
        _escrow['funding_vout'] = vout;
        _statusMessage = 'Funding transaction saved.';
        _statusIsError = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _statusMessage = e.toString();
        _statusIsError = true;
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isUpdatingFunding = false;
      });
    }
  }

  Future<void> _handleBuildRefund() async {
    final dest = _destAddressCtrl.text.trim();
    if (dest.isEmpty) {
      setState(() {
        _statusMessage = 'Destination address or pubkey is required.';
        _statusIsError = true;
      });
      return;
    }

    final csvInput = _csvBlocksCtrl.text.trim();
    final csvBlocks = csvInput.isNotEmpty
        ? int.tryParse(csvInput) ?? _toInt(_escrow['csv_blocks'])
        : _toInt(_escrow['csv_blocks']);

    final utxoValueInput = _utxoValueCtrl.text.trim();
    final utxoValue =
        utxoValueInput.isNotEmpty ? _atomsFromDcr(utxoValueInput) : null;
    if (utxoValueInput.isNotEmpty && utxoValue == null) {
      setState(() {
        _statusMessage = 'UTXO value must be a valid DCR amount.';
        _statusIsError = true;
      });
      return;
    }

    setState(() {
      _isBuilding = true;
      _statusMessage = 'Building refund transaction...';
      _statusIsError = false;
      _refundTxHex = null;
      _refundResult = null;
    });

    try {
      final model = context.read<PokerModel>();
      final result = await model.buildRefundTransaction(
        _escrowId,
        dest,
        csvBlocks: csvBlocks > 0 ? csvBlocks : null,
        utxoValue: utxoValue,
      );
      if (!mounted) return;
      setState(() {
        _refundResult = result;
        if (result['can_refund'] == true) {
          _refundTxHex = result['refund_tx_hex']?.toString();
          _statusMessage = 'Refund transaction built successfully.';
          _statusIsError = false;
          // Update escrow info with latest utxo hints if provided
          if (result['utxo_txid'] != null) {
            _escrow['funding_txid'] = result['utxo_txid'];
          }
          if (result['utxo_vout'] != null) {
            _escrow['funding_vout'] = result['utxo_vout'];
          }
          if (result['utxo_value'] != null) {
            _escrow['funded_amount'] = result['utxo_value'];
            _utxoValueCtrl.text =
                formatDcrFromAtoms(_toInt(result['utxo_value']));
          }
          // Mark escrow as having a refund tx built (not broadcast).
          _escrow['status'] = 'tx built';
        } else {
          _refundTxHex = null;
          final reason = result['reason']?.toString();
          _statusMessage = reason?.isNotEmpty == true
              ? 'Cannot refund: $reason'
              : 'Cannot refund this escrow.';
          _statusIsError = true;
        }
      });
      // Persist status change when refund tx is built.
      if (_refundTxHex != null && _refundTxHex!.isNotEmpty) {
        try {
          await Golib.updateEscrowHistory({
            'escrow_id': _escrowId,
            'status': 'tx built',
          });
        } catch (_) {
          // Non-fatal; UI already updated.
        }
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _statusMessage = e.toString();
        _statusIsError = true;
        _refundTxHex = null;
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isBuilding = false;
      });
    }
  }

  Future<void> _copyRefundTx() async {
    if (_refundTxHex == null || _refundTxHex!.isEmpty) return;
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    await Clipboard.setData(ClipboardData(text: _refundTxHex!));
    if (!mounted) return;
    messenger.showSnackBar(
      const SnackBar(content: Text('Refund transaction copied to clipboard')),
    );
  }

  Future<void> _copyFundingTx() async {
    final fundingTx = _escrow['funding_txid']?.toString() ?? '';
    if (fundingTx.isEmpty) return;
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    await Clipboard.setData(ClipboardData(text: fundingTx));
    if (!mounted) return;
    messenger.showSnackBar(
      const SnackBar(
          content: Text('Funding transaction ID copied to clipboard')),
    );
  }

  Future<void> _handleDeleteEscrow() async {
    if (widget.onDelete != null) {
      await widget.onDelete!(context);
    }
  }

  @override
  Widget build(BuildContext context) {
    final fundingTx = _escrow['funding_txid']?.toString() ?? '';
    final fundingVout = _toInt(_escrow['funding_vout']);
    final amountAtoms = _toInt(_escrow['funded_amount']);
    final csvBlocks = _toInt(_escrow['csv_blocks']);
    final archivedAt = _toInt(_escrow['archived_at']);
    final archivedText = archivedAt > 0
        ? DateTime.fromMillisecondsSinceEpoch(archivedAt)
            .toLocal()
            .toString()
            .split('.')
            .first
        : 'Unknown';

    return AlertDialog(
      title: const Text('Refund Escrow'),
      content: SingleChildScrollView(
        child: SizedBox(
          width: MediaQuery.of(context).size.width * 0.6,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              SelectableText(
                'Escrow ID: $_escrowId',
                style:
                    const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
              ),
              const SizedBox(height: 12),
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      width: 120,
                      child: Text(
                        'Funding',
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.grey.shade400,
                        ),
                      ),
                    ),
                    Expanded(
                      child: Row(
                        children: [
                          Flexible(
                            child: ConstrainedBox(
                              constraints: const BoxConstraints(maxWidth: 400),
                              child: Text(
                                fundingTx.isNotEmpty
                                    ? '${_shorten(fundingTx, head: 12, tail: 12)}:${fundingVout >= 0 ? fundingVout : 0}'
                                    : 'Not recorded',
                                style: TextStyle(
                                  fontSize: 12,
                                  color: fundingTx.isEmpty
                                      ? Colors.orangeAccent
                                      : null,
                                  fontStyle: fundingTx.isEmpty
                                      ? FontStyle.italic
                                      : FontStyle.normal,
                                ),
                              ),
                            ),
                          ),
                          if (fundingTx.isNotEmpty) ...[
                            const SizedBox(width: 8),
                            IconButton(
                              icon: const Icon(Icons.copy, size: 16),
                              onPressed: _copyFundingTx,
                              tooltip: 'Copy funding transaction ID',
                              padding: EdgeInsets.zero,
                              constraints: const BoxConstraints(
                                minWidth: 24,
                                minHeight: 24,
                              ),
                              color: Colors.grey.shade400,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              _InfoRow(
                label: 'Amount',
                value: amountAtoms > 0
                    ? '${formatDcrFromAtoms(amountAtoms)} DCR'
                    : 'Unknown',
              ),
              _InfoRow(
                label: 'CSV blocks',
                value: csvBlocks > 0 ? csvBlocks.toString() : 'Unknown',
              ),
              _InfoRow(
                label: 'Archived',
                value: archivedText,
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _destAddressCtrl,
                decoration: const InputDecoration(
                  labelText: 'Refund destination (address or pubkey)',
                  hintText: 'Destination to receive the refund',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _csvBlocksCtrl,
                decoration: InputDecoration(
                  labelText: 'CSV blocks override',
                  hintText: csvBlocks > 0 ? csvBlocks.toString() : 'e.g. 2',
                  helperText:
                      'Optional. Leave empty to use the stored CSV timelock.',
                ),
                keyboardType: TextInputType.number,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _utxoValueCtrl,
                decoration: const InputDecoration(
                  labelText: 'UTXO value (DCR)',
                  helperText:
                      'Optional in case of wrong input. Enter a DCR amount.',
                ),
                keyboardType:
                    const TextInputType.numberWithOptions(decimal: true),
              ),
              const SizedBox(height: 20),
              Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  OutlinedButton.icon(
                    onPressed: _isBuilding ? null : _handleFundingUpdate,
                    icon: _isUpdatingFunding
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.edit),
                    label: Text(
                      fundingTx.isEmpty
                          ? 'Record funding transaction'
                          : 'Edit funding transaction',
                    ),
                  ),
                  ElevatedButton.icon(
                    onPressed: (_isBuilding || _isUpdatingFunding)
                        ? null
                        : _handleBuildRefund,
                    icon: _isBuilding
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Icon(Icons.currency_exchange),
                    label: Text(
                      _isBuilding ? 'Building...' : 'Build refund transaction',
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              if (_statusMessage != null)
                _StatusBanner(
                  message: _statusMessage!,
                  isError: _statusIsError,
                ),
              if (_refundResult != null && _refundResult!['utxo_txid'] != null)
                Padding(
                  padding: const EdgeInsets.only(top: 12),
                  child: _InfoRow(
                    label: 'Refund UTXO',
                    value:
                        '${_shorten(_refundResult!['utxo_txid'].toString(), head: 12, tail: 12)}:${_refundResult!['utxo_vout']}',
                  ),
                ),
              if (_refundTxHex != null && _refundTxHex!.isNotEmpty) ...[
                const SizedBox(height: 16),
                const Text(
                  'Refund transaction (hex)',
                  style: TextStyle(fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.black.withOpacity(0.35),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.grey.shade700),
                  ),
                  child: SelectableText(
                    _refundTxHex!,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                    maxLines: 6,
                  ),
                ),
                const SizedBox(height: 8),
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.blue.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.blue.withOpacity(0.3)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'To rebroadcast this transaction, visit dcrdata:',
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.blue.shade200,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 8),
                      InkWell(
                        onTap: () async {
                          const url = 'https://dcrdata.org/decodetx';
                          try {
                            if (Platform.isWindows) {
                              await Process.run('start', [url],
                                  runInShell: true);
                            } else if (Platform.isMacOS) {
                              await Process.run('open', [url]);
                            } else if (Platform.isLinux) {
                              await Process.run('xdg-open', [url]);
                            } else {
                              await Clipboard.setData(
                                  const ClipboardData(text: url));
                              if (!mounted) return;
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(
                                  content: Text('URL copied to clipboard'),
                                ),
                              );
                            }
                          } catch (e) {
                            await Clipboard.setData(
                                const ClipboardData(text: url));
                            if (!mounted) return;
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content: Text('URL copied to clipboard: $url'),
                              ),
                            );
                          }
                        },
                        child: Text(
                          'https://dcrdata.org/decodetx',
                          style: TextStyle(
                            fontSize: 12,
                            color: Colors.blue.shade300,
                            decoration: TextDecoration.underline,
                          ),
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Paste the transaction hex above into the "Broadcast Tx" field on dcrdata to rebroadcast it to the network.',
                        style: TextStyle(
                          fontSize: 11,
                          color: Colors.grey.shade400,
                          fontStyle: FontStyle.italic,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton.icon(
          onPressed: widget.onDelete != null ? _handleDeleteEscrow : null,
          icon: const Icon(Icons.delete_outline, size: 18),
          label: const Text('Delete escrow'),
          style: TextButton.styleFrom(
            foregroundColor: Colors.redAccent,
          ),
        ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Close'),
        ),
        if (_refundTxHex != null && _refundTxHex!.isNotEmpty)
          TextButton.icon(
            onPressed: _copyRefundTx,
            icon: const Icon(Icons.copy, size: 18),
            label: const Text('Copy transaction'),
          ),
      ],
    );
  }

  static int _toInt(dynamic value) {
    if (value is int) return value;
    if (value is num) return value.toInt();
    if (value is String) return int.tryParse(value) ?? 0;
    return 0;
  }

  static String _shorten(String value, {int head = 6, int tail = 4}) {
    if (value.isEmpty || value.length <= head + tail) {
      return value;
    }
    return '${value.substring(0, head)}...${value.substring(value.length - tail)}';
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({
    required this.label,
    required this.value,
    this.valueStyle,
  });

  final String label;
  final String value;
  final TextStyle? valueStyle;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 120,
            child: Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: Colors.grey.shade400,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: valueStyle ??
                  const TextStyle(
                    fontSize: 12,
                  ),
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusBanner extends StatelessWidget {
  const _StatusBanner({required this.message, required this.isError});

  final String message;
  final bool isError;

  @override
  Widget build(BuildContext context) {
    final color = isError ? Colors.redAccent : Colors.greenAccent;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            isError ? Icons.error : Icons.check_circle,
            color: color,
            size: 20,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: TextStyle(color: color, fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }
}

String _asString(dynamic v) {
  if (v == null) return '';
  return v.toString();
}

int? _asInt(dynamic v) {
  if (v == null) return null;
  if (v is int) return v;
  if (v is double) return v.toInt();
  if (v is String && v.trim().isNotEmpty) {
    return int.tryParse(v);
  }
  return null;
}
