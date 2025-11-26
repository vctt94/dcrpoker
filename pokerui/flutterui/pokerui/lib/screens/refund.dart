import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:pokerui/components/shared_layout.dart';

class RefundScreen extends StatefulWidget {
  const RefundScreen({super.key});

  @override
  State<RefundScreen> createState() => _RefundScreenState();
}

class _RefundScreenState extends State<RefundScreen> {
  final _indexCtrl = TextEditingController(text: '0');
  String? _priv;
  String? _pub;
  String? _error;
  bool _loading = false;

  @override
  void dispose() {
    _indexCtrl.dispose();
    super.dispose();
  }

  Future<void> _derive() async {
    final idxStr = _indexCtrl.text.trim();
    final idx = int.tryParse(idxStr);
    if (idx == null || idx < 0) {
      setState(() {
        _error = 'Enter a valid non-negative index';
        _priv = null;
        _pub = null;
      });
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
      _priv = null;
      _pub = null;
    });
    try {
      final res = await Golib.deriveSettlementSessionKey(idx);
      setState(() {
        _priv = res['priv'];
        _pub = res['pub'];
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      setState(() {
        _loading = false;
      });
    }
  }

  Widget _keyRow(String label, String? value) {
    if (value == null || value.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 100,
            child: Text(label,
                style: const TextStyle(color: Colors.white70, fontWeight: FontWeight.bold)),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: const TextStyle(color: Colors.white, fontFamily: 'monospace'),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.copy, color: Colors.white70),
            tooltip: 'Copy',
            onPressed: () {
              Clipboard.setData(ClipboardData(text: value));
              ScaffoldMessenger.of(context)
                  .showSnackBar(const SnackBar(content: Text('Copied')));
            },
          )
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Refund Tools',
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Recover a session key deterministically using your index. '
                'Use the recovered private key to rebuild refund transactions.',
                style: TextStyle(color: Colors.white, fontSize: 15),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _indexCtrl,
                decoration: const InputDecoration(
                  labelText: 'Session key index',
                  hintText: 'e.g. 0',
                  filled: true,
                  fillColor: Colors.white,
                  border: OutlineInputBorder(),
                ),
                style: const TextStyle(color: Colors.black),
                keyboardType: TextInputType.number,
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
              ElevatedButton(
                onPressed: _loading ? null : _derive,
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blueAccent,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
                child: _loading
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : const Text('Derive key'),
              ),
              _keyRow('Priv', _priv),
              _keyRow('Pub', _pub),
              const SizedBox(height: 24),
              const Text(
                'Tip: store the session key index next to each escrow ID so you can regenerate it later.',
                style: TextStyle(color: Colors.white70),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
