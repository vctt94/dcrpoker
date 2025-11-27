import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:pokerui/components/shared_layout.dart';

class OpenEscrowScreen extends StatefulWidget {
  const OpenEscrowScreen({super.key});

  @override
  State<OpenEscrowScreen> createState() => _OpenEscrowScreenState();
}

class _OpenEscrowScreenState extends State<OpenEscrowScreen> {
  final _betDcrController = TextEditingController(text: '0.10');
  final _csvBlocksController = TextEditingController(text: '64');
  final _compPrivController = TextEditingController();
  final _compPubController = TextEditingController();
  final _statusEscrowIdController = TextEditingController();
  String? _keyIndex;
  bool _isGenerating = false;
  bool _isOpening = false;
  String? _error;
  bool _needsPayoutAddress = false;
  Map<String, dynamic>? _result;
  Map<String, dynamic>? _status;
  String? _statusError;
  bool _statusLoading = false;

  @override
  void dispose() {
    _betDcrController.dispose();
    _csvBlocksController.dispose();
    _compPrivController.dispose();
    _compPubController.dispose();
    _statusEscrowIdController.dispose();
    super.dispose();
  }

  int? _atomsFromDcr(String v) {
    final cleaned = v.trim();
    if (cleaned.isEmpty) return null;
    final parsed = double.tryParse(cleaned);
    if (parsed == null) return null;
    return (parsed * 1e8).round();
  }

  Future<void> _generateKey() async {
    setState(() {
      _isGenerating = true;
      _error = null;
      _result = null;
    });
    try {
      final res = await Golib.generateSettlementSessionKey();
      _compPrivController.text = res['priv'] ?? '';
      _compPubController.text = res['pub'] ?? '';
      _keyIndex = res['index'];
    } catch (e) {
      setState(() {
        _error = 'Failed to generate key: $e';
      });
    } finally {
      setState(() {
        _isGenerating = false;
      });
    }
  }

  Future<void> _openEscrow() async {
    final betAtoms = _atomsFromDcr(_betDcrController.text);
    final csvBlocks = int.tryParse(_csvBlocksController.text.trim());
    final compPub = _compPubController.text.trim();
    final keyIndexStr = _keyIndex;

    if (betAtoms == null || betAtoms <= 0) {
      setState(() => _error = 'Enter a bet amount > 0');
      return;
    }
    if (compPub.isEmpty) {
      setState(() => _error = 'Provide a compressed session pubkey');
      return;
    }
    if (keyIndexStr == null || keyIndexStr.isEmpty) {
      setState(() => _error = 'Generate a session key first');
      return;
    }
    final keyIndex = int.tryParse(keyIndexStr);
    if (keyIndex == null) {
      setState(() => _error = 'Invalid key index');
      return;
    }

    setState(() {
      _isOpening = true;
      _error = null;
      _needsPayoutAddress = false;
      _result = null;
    });

    try {
      final res = await Golib.openEscrow(
        betAtoms: betAtoms,
        compPubkey: compPub,
        keyIndex: keyIndex,
        csvBlocks: csvBlocks ?? 64,
      );
      setState(() {
        _result = res;
      });
    } catch (e) {
      setState(() {
        final msg = e.toString();
        _error = msg;
        _needsPayoutAddress =
            msg.toLowerCase().contains('payout address not set');
      });
    } finally {
      setState(() {
        _isOpening = false;
      });
    }
  }

  Future<void> _checkStatus() async {
    final escrowID = _statusEscrowIdController.text.trim();
    if (escrowID.isEmpty) {
      setState(() {
        _statusError = 'Enter an escrow ID';
        _status = null;
      });
      return;
    }
    setState(() {
      _statusLoading = true;
      _statusError = null;
      _status = null;
    });
    try {
      final res = await Golib.getEscrowStatus(escrowID);
      setState(() {
        _status = res;
      });
    } catch (e) {
      setState(() {
        _statusError = e.toString();
      });
    } finally {
      setState(() {
        _statusLoading = false;
      });
    }
  }

  Widget _label(String text) => Padding(
        padding: const EdgeInsets.only(bottom: 6),
        child: Text(
          text,
          style: const TextStyle(color: Colors.white70, fontWeight: FontWeight.bold),
        ),
      );

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Open Escrow',
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Fund an escrow before joining a table. Verify payout address on the Sign Address screen first.',
                style: TextStyle(color: Colors.white, fontSize: 15),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  ElevatedButton(
                    onPressed: _isGenerating ? null : _generateKey,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.teal,
                      padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
                    ),
                    child: _isGenerating
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                            ),
                          )
                        : const Text('Generate Session Key'),
                  ),
                  const SizedBox(width: 12),
                  const Expanded(
                    child: Text(
                      'Keep the private key safe; it is required for presign/finalization.',
                      style: TextStyle(color: Colors.white70),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              _label('Compressed Pubkey (33B hex)'),
              TextField(
                controller: _compPubController,
                decoration: const InputDecoration(
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                  hintText: '02...',
                ),
                style: const TextStyle(color: Colors.black),
              ),
              const SizedBox(height: 8),
              _label('Session Private Key (hex)'),
              TextField(
                controller: _compPrivController,
                decoration: const InputDecoration(
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                  hintText: 'Save securely',
                ),
                style: const TextStyle(color: Colors.black),
                maxLines: 2,
              ),
              if (_keyIndex != null)
                Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: Text(
                    'Key index: $_keyIndex (store with escrow for recovery)',
                    style: const TextStyle(color: Colors.white70),
                  ),
                ),
              const SizedBox(height: 16),
              _label('Bet Amount (DCR)'),
              TextField(
                controller: _betDcrController,
                decoration: const InputDecoration(
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                  hintText: '0.10',
                ),
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                style: const TextStyle(color: Colors.black),
              ),
              const SizedBox(height: 12),
              _label('CSV Blocks (default 64)'),
              TextField(
                controller: _csvBlocksController,
                decoration: const InputDecoration(
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                ),
                keyboardType: TextInputType.number,
                style: const TextStyle(color: Colors.black),
              ),
              const SizedBox(height: 16),
              if (_error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: SelectableText(
                    _error!,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                ),
              if (_needsPayoutAddress)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  margin: const EdgeInsets.only(bottom: 12),
                  decoration: BoxDecoration(
                    color: Colors.red.withOpacity(0.08),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.redAccent.withOpacity(0.4)),
                  ),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Icon(Icons.warning_amber_rounded,
                          color: Colors.redAccent),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'Set a payout address first',
                              style: TextStyle(
                                  color: Colors.redAccent,
                                  fontWeight: FontWeight.bold),
                            ),
                            const SizedBox(height: 4),
                            const Text(
                              'Open Sign Address to verify a payout address before funding an escrow.',
                              style: TextStyle(color: Colors.white70),
                            ),
                            const SizedBox(height: 8),
                            ElevatedButton(
                              style: ElevatedButton.styleFrom(
                                backgroundColor: Colors.redAccent,
                              ),
                              onPressed: () {
                                Navigator.of(context).pushNamed('/sign-address');
                              },
                              child: const Text('Go to Sign Address'),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              if (_result != null)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  margin: const EdgeInsets.only(bottom: 12),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1B1E2C),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.blueAccent.withOpacity(.4)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Escrow Created',
                          style: TextStyle(
                              color: Colors.white, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      ..._result!.entries.map((e) => Padding(
                            padding: const EdgeInsets.symmetric(vertical: 4),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                SizedBox(
                                  width: 140,
                                  child: Text(
                                    e.key,
                                    style: const TextStyle(color: Colors.white70),
                                  ),
                                ),
                                Expanded(
                                  child: SelectableText(
                                    '${e.value}',
                                    style: const TextStyle(color: Colors.white),
                                  ),
                                ),
                                IconButton(
                                  icon: const Icon(Icons.copy, color: Colors.white70),
                                  onPressed: () => Clipboard.setData(
                                      ClipboardData(text: '${e.value}')),
                                ),
                              ],
                            ),
                          )),
                    ],
                  ),
                ),
              ElevatedButton(
                onPressed: _isOpening ? null : _openEscrow,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  backgroundColor: Colors.blueAccent,
                ),
                child: _isOpening
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : const Text('Open Escrow'),
              ),
              const SizedBox(height: 24),
              const Text(
                'Escrow Status',
                style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: _statusEscrowIdController,
                decoration: const InputDecoration(
                  labelText: 'Escrow ID',
                  hintText: 'escrow_...',
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                ),
                style: const TextStyle(color: Colors.black),
              ),
              const SizedBox(height: 8),
              if (_statusError != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: SelectableText(
                    _statusError!,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                ),
              ElevatedButton(
                onPressed: _statusLoading ? null : _checkStatus,
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blueGrey,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
                child: _statusLoading
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : const Text('Check status'),
              ),
              if (_status != null)
                Container(
                  width: double.infinity,
                  margin: const EdgeInsets.only(top: 12),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1B1E2C),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.blueAccent.withOpacity(.4)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Current status',
                          style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      _statusRow('Confs', _status!['confs']),
                      _statusRow('UTXO count', _status!['utxo_count']),
                      _statusRow('Funding tx', _status!['funding_txid']),
                      _statusRow('Vout', _status!['funding_vout']),
                      _statusRow('Amount (atoms)', _status!['amount_atoms']),
                      _statusRow('CSV blocks', _status!['csv_blocks']),
                      _statusRow('Mature for CSV', _status!['mature_for_csv']),
                      _statusRow('Required confs', _status!['required_confirmations']),
                      if (_status!['updated_at_unix'] != null && _status!['updated_at_unix'] != 0)
                        _statusRow('Updated at (unix)', _status!['updated_at_unix']),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _statusRow(String label, dynamic value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 130,
            child: Text(label, style: const TextStyle(color: Colors.white70)),
          ),
          Expanded(
            child: SelectableText(
              value == null ? '' : value.toString(),
              style: const TextStyle(color: Colors.white),
            ),
          ),
        ],
      ),
    );
  }
}
