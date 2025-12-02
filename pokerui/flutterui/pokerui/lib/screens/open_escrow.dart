import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
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
  String? _keyIndex;
  bool _isOpening = false;
  String? _error;
  bool _needsPayoutAddress = false;
  String? _payoutAddress;
  Map<String, dynamic>? _result;

  @override
  void initState() {
    super.initState();
    _ensurePayoutAddress();
  }

  @override
  void dispose() {
    _betDcrController.dispose();
    _csvBlocksController.dispose();
    _compPrivController.dispose();
    _compPubController.dispose();
    super.dispose();
  }

  bool _isPayoutMissingError(String msg) {
    final lower = msg.toLowerCase();
    return lower.contains('payout address not set') ||
        lower.contains('sign address');
  }

  int? _atomsFromDcr(String v) {
    final cleaned = v.trim();
    if (cleaned.isEmpty) return null;
    final parsed = double.tryParse(cleaned);
    if (parsed == null) return null;
    return (parsed * 1e8).round();
  }

  Future<bool> _ensurePayoutAddress() async {
    if ((_payoutAddress ?? '').trim().isNotEmpty) {
      return true;
    }
    try {
      // Prefer the payout address bound to the authenticated session.
      LoginResponse? session;
      try {
        session = await Golib.resumeSession();
      } catch (_) {
        session = null;
      }
      final serverAddr = session?.address.trim() ?? '';
      final addr = serverAddr.isNotEmpty
          ? serverAddr
          : (await Golib.getPayoutAddress()).trim();
      if (!mounted) return addr.isNotEmpty;
      setState(() {
        _payoutAddress = addr;
        _needsPayoutAddress = addr.isEmpty;
      });
      return addr.isNotEmpty;
    } catch (e) {
      if (mounted) {
        setState(() {
          _needsPayoutAddress = true;
          _error = null;
        });
      }
      return false;
    }
  }

  Future<void> _openEscrow() async {
    final betAtoms = _atomsFromDcr(_betDcrController.text);
    final csvBlocks = int.tryParse(_csvBlocksController.text.trim());
    var compPub = _compPubController.text.trim();
    var keyIndexStr = _keyIndex;

    if (betAtoms == null || betAtoms <= 0) {
      setState(() => _error = 'Enter a bet amount > 0');
      return;
    }

    final hasPayoutAddress = await _ensurePayoutAddress();
    if (!hasPayoutAddress) {
      if (mounted) {
        setState(() {
          _error = null;
          _needsPayoutAddress = true;
          _isOpening = false;
        });
      }
      return;
    }

    setState(() {
      _isOpening = true;
      _error = null;
      _needsPayoutAddress = false;
      _result = null;
    });

    try {
      // Auto-generate session key if not already set
      if (compPub.isEmpty || keyIndexStr == null || keyIndexStr.isEmpty) {
        final res = await Golib.generateSettlementSessionKey();
        compPub = res['pub'] ?? '';
        keyIndexStr = res['index'] ?? '';
        final priv = res['priv'] ?? '';
        setState(() {
          _compPrivController.text = priv;
          _compPubController.text = compPub;
          _keyIndex = keyIndexStr;
        });
      }

      if (compPub.isEmpty || keyIndexStr.isEmpty) {
        setState(() {
          _error = 'Failed to generate session key';
          _isOpening = false;
        });
        return;
      }

      final keyIndex = int.tryParse(keyIndexStr);
      if (keyIndex == null) {
        setState(() {
          _error = 'Invalid key index';
          _isOpening = false;
        });
        return;
      }

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
        // If payout address is not set, show the sign address message directly.
        if (_isPayoutMissingError(msg)) {
          _error = null;
          _needsPayoutAddress = true;
        } else {
          _error = msg;
          _needsPayoutAddress = false;
        }
      });
    } finally {
      setState(() {
        _isOpening = false;
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
                'Fund an escrow before joining a table. Verify payout address on the Sign Address screen first. A session key will be automatically generated when opening an escrow.',
                style: TextStyle(color: Colors.white, fontSize: 15),
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
              const SizedBox(height: 16),
              if (_compPubController.text.trim().isNotEmpty ||
                  _compPrivController.text.trim().isNotEmpty ||
                  (_keyIndex ?? '').isNotEmpty)
                Container(
                  width: double.infinity,
                  margin: const EdgeInsets.only(top: 8),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1B1E2C),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.blueAccent.withOpacity(.4)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Session Key',
                          style: TextStyle(
                              color: Colors.white, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      _keyValueRow('Compressed Pubkey', _compPubController.text.trim()),
                      _keyValueRow('Session Private Key', _compPrivController.text.trim()),
                      if ((_keyIndex ?? '').isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(top: 6),
                          child: Text(
                            'Key index: $_keyIndex (store with escrow for recovery)',
                            style: const TextStyle(color: Colors.white70),
                          ),
                        ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _keyValueRow(String label, String value) {
    if (value.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 150,
            child: Text(label, style: const TextStyle(color: Colors.white70)),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: const TextStyle(color: Colors.white),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.copy, color: Colors.white70),
            onPressed: () => Clipboard.setData(ClipboardData(text: value)),
          ),
        ],
      ),
    );
  }
}
