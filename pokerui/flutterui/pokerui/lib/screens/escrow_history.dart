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
        setState(() {
          entry.statusError = e.toString();
        });
      }
    }
  }

  String _stateLabel(_EscrowEntry e) {
    final live = e.liveStatus;
    if (live == null) return 'Status unknown';
    if (live.utxoCount == 0) return 'Spent';
    if (live.matureForCsv) return 'Refundable';
    return 'Locked';
  }

  Color _stateColor(_EscrowEntry e) {
    final live = e.liveStatus;
    if (live == null) return Colors.white70;
    if (live.utxoCount == 0) return Colors.redAccent;
    if (live.matureForCsv) return Colors.greenAccent;
    return Colors.orangeAccent;
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
        : DateTime.fromMillisecondsSinceEpoch(updated * 1000).toLocal().toString();
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
                  ScaffoldMessenger.of(context)
                      .showSnackBar(const SnackBar(content: Text('Escrow ID copied')));
                },
                icon: const Icon(Icons.copy, color: Colors.white70),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                decoration: BoxDecoration(
                  color: _stateColor(e).withOpacity(.15),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _stateColor(e).withOpacity(.6)),
                ),
                child: Text(
                  _stateLabel(e),
                  style: TextStyle(color: _stateColor(e), fontWeight: FontWeight.bold),
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
          if (e.csvBlocks != null) _infoRow('CSV blocks', e.csvBlocks.toString()),
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
              style: const TextStyle(color: Colors.white70, fontWeight: FontWeight.w600),
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
  });

  factory _EscrowLiveStatus.fromJson(Map<String, dynamic> json) {
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
