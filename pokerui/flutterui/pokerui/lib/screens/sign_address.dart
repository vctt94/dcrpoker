import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/poker.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/config.dart';
import 'package:path/path.dart' as p;

class SignAddressScreen extends StatefulWidget {
  const SignAddressScreen({super.key});

  @override
  State<SignAddressScreen> createState() => _SignAddressScreenState();
}

class _SignAddressScreenState extends State<SignAddressScreen> {
  final _addressController = TextEditingController();
  final _signatureController = TextEditingController();
  String? _code;
  String? _addressHint;
  int? _ttlSec;
  String? _error;
  String? _success;
  bool _isRequesting = false;
  bool _isSubmitting = false;
  bool _prefilled = false;

  @override
  void dispose() {
    _addressController.dispose();
    _signatureController.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    _prefillSavedAddress();
  }

  Future<void> _prefillSavedAddress() async {
    if (_prefilled) return;
    try {
      final cfgPath = p.join(await defaultAppDataDir(), '$APPNAME.conf');
      final cfg = await Golib.loadConfig(cfgPath);
      final addr = (cfg['payout_address'] ?? '').toString().trim();
      if (!mounted) return;
      if (addr.isNotEmpty && _addressController.text.isEmpty) {
        setState(() {
          _addressController.text = addr;
          _addressHint = addr;
          _prefilled = true;
        });
      }
    } catch (_) {
      // Best effort; ignore if missing or unreadable.
    }
  }

  Future<void> _requestCode() async {
    setState(() {
      _isRequesting = true;
      _error = null;
      _success = null;
    });
    try {
      final resp = await Golib.requestLoginCode();
      setState(() {
        _code = resp.code;
        _ttlSec = resp.ttlSec;
        _addressHint = resp.addressHint;
      });
    } catch (e) {
      setState(() {
        _error = 'Failed to request code: $e';
      });
    } finally {
      setState(() {
        _isRequesting = false;
      });
    }
  }

  Future<void> _submit() async {
    if ((_code ?? '').isEmpty) {
      setState(() {
        _error = 'Request a code first';
        _success = null;
      });
      return;
    }
    final address = _addressController.text.trim();
    final signature = _signatureController.text.trim();
    if (address.isEmpty || signature.isEmpty) {
      setState(() {
        _error = 'Address and signature are required';
        _success = null;
      });
      return;
    }

    setState(() {
      _isSubmitting = true;
      _error = null;
      _success = null;
    });

    try {
      final resp = await Golib.setPayoutAddress(
        SetPayoutAddressRequest(address, signature, _code!),
      );
      if (!resp.ok) {
        setState(() {
          _error = resp.error.isNotEmpty ? resp.error : 'Failed to set address';
        });
        return;
      }
      // Update authed payout address in the model so escrow checks use server-bound value.
      final model = mounted ? Provider.of<PokerModel?>(context, listen: false) : null;
      model?.updateAuthedPayoutAddress(resp.address);
      setState(() {
        _success = 'Address verified and saved: ${resp.address}';
        _addressHint = resp.address;
        _prefilled = true;
      });
    } catch (e) {
      setState(() {
        _error = 'Failed to save address: $e';
      });
    } finally {
      setState(() {
        _isSubmitting = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Sign Address',
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Prove control of your Decred address for payouts.',
                style: TextStyle(color: Colors.white, fontSize: 16),
              ),
              const SizedBox(height: 12),
              ElevatedButton(
                onPressed: _isRequesting ? null : _requestCode,
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.teal,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
                child: _isRequesting
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : const Text('Request Code'),
              ),
              if (_code != null) ...[
                const SizedBox(height: 12),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: const Color(0xFF1B1E2C),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.blueAccent.withOpacity(.4)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const Text('Code',
                              style: TextStyle(
                                  color: Colors.white, fontWeight: FontWeight.bold)),
                          IconButton(
                            icon: const Icon(Icons.copy, color: Colors.white70),
                            onPressed: () => Clipboard.setData(ClipboardData(text: _code!)),
                          )
                        ],
                      ),
                      const SizedBox(height: 6),
                      SelectableText(
                        _code!,
                        style:
                            const TextStyle(color: Colors.white, fontFamily: 'monospace'),
                      ),
                      if (_ttlSec != null)
                        Padding(
                          padding: const EdgeInsets.only(top: 4),
                          child: Text(
                            'Expires in ~$_ttlSec sec',
                            style: const TextStyle(color: Colors.white70),
                          ),
                        ),
                      if (_addressHint != null && _addressHint!.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(top: 4),
                          child: Text(
                            'Last verified: $_addressHint',
                            style: const TextStyle(color: Colors.white70),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
              const SizedBox(height: 16),
              TextField(
                controller: _addressController,
                decoration: const InputDecoration(
                  labelText: 'Payout Address',
                  hintText: 'Paste the address you will receive payouts to',
                  border: OutlineInputBorder(),
                  filled: true,
                  fillColor: Colors.white,
                ),
                style: const TextStyle(color: Colors.black),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _signatureController,
                decoration: const InputDecoration(
                  labelText: 'Wallet Signature',
                  hintText: 'Sign the code above using your Decred wallet (base64)',
                  border: OutlineInputBorder(),
                  filled: true,
                  fillColor: Colors.white,
                ),
                style: const TextStyle(color: Colors.black),
                maxLines: 2,
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
              if (_success != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: SelectableText(
                    _success!,
                    style: const TextStyle(color: Colors.greenAccent),
                  ),
                ),
              ElevatedButton(
                onPressed: _isSubmitting ? null : _submit,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  backgroundColor: Colors.blueAccent,
                ),
                child: _isSubmitting
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : const Text('Verify & Save'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
