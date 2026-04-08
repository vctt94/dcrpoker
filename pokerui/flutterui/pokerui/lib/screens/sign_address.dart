import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';
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
      final model =
          mounted ? Provider.of<PokerModel?>(context, listen: false) : null;
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

  Future<void> _copyCode() async {
    final code = _code;
    if (code == null || code.isEmpty) return;
    await Clipboard.setData(ClipboardData(text: code));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Verification code copied')));
  }

  @override
  Widget build(BuildContext context) {
    final hasCode = (_code ?? '').isNotEmpty;
    final verifiedAddress = (_addressHint ?? '').trim();

    return SharedLayout(
      title: 'Sign Address',
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 860),
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(PokerSpacing.xl),
            child: Container(
              padding: const EdgeInsets.all(PokerSpacing.xl),
              decoration: BoxDecoration(
                color: PokerColors.surface,
                borderRadius: BorderRadius.circular(20),
                border: Border.all(color: PokerColors.borderSubtle),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'Verify payout address',
                    style: PokerTypography.headlineMedium,
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  Text(
                    'This address is used for settlements and payouts. Sign the verification code with the same wallet to confirm the address belongs to you.',
                    style: PokerTypography.bodyMedium
                        .copyWith(color: PokerColors.textSecondary),
                  ),
                  if (verifiedAddress.isNotEmpty) ...[
                    const SizedBox(height: PokerSpacing.md),
                    Text(
                      'Current verified address: $verifiedAddress',
                      style: PokerTypography.bodySmall
                          .copyWith(color: PokerColors.success),
                    ),
                  ],
                  const SizedBox(height: PokerSpacing.xl),
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      Expanded(
                        child: Text(
                          'Settlement Address',
                          style: PokerTypography.titleSmall,
                        ),
                      ),
                      const _InfoTooltip(
                        message:
                            'This is the address the server will use when settling your balance. Signing the verification code proves the address is yours.',
                      ),
                    ],
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  TextField(
                    controller: _addressController,
                    decoration: const InputDecoration(
                      hintText: 'Paste your Decred payout address',
                    ),
                    style: PokerTypography.bodyMedium,
                  ),
                  const SizedBox(height: PokerSpacing.lg),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          'Verification Code',
                          style: PokerTypography.titleSmall,
                        ),
                      ),
                      ElevatedButton.icon(
                        onPressed: _isRequesting ? null : _requestCode,
                        icon: _isRequesting
                            ? const SizedBox(
                                width: 16,
                                height: 16,
                                child:
                                    CircularProgressIndicator(strokeWidth: 2),
                              )
                            : const Icon(Icons.verified_user_outlined,
                                size: 18),
                        label: Text(hasCode ? 'Refresh Code' : 'Request Code'),
                      ),
                    ],
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(PokerSpacing.lg),
                    decoration: BoxDecoration(
                      color: PokerColors.surfaceDim,
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(color: PokerColors.borderSubtle),
                    ),
                    child: hasCode
                        ? Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    SelectableText(
                                      _code!,
                                      style: PokerTypography.bodyLarge.copyWith(
                                        fontFamily: 'monospace',
                                        letterSpacing: 0.2,
                                      ),
                                    ),
                                    if (_ttlSec != null) ...[
                                      const SizedBox(height: PokerSpacing.xs),
                                      Text(
                                        'Expires in ~$_ttlSec sec',
                                        style: PokerTypography.bodySmall,
                                      ),
                                    ],
                                  ],
                                ),
                              ),
                              IconButton(
                                tooltip: 'Copy code',
                                onPressed: _copyCode,
                                icon: const Icon(Icons.copy_rounded),
                                color: PokerColors.textSecondary,
                              ),
                            ],
                          )
                        : Text(
                            'Request a code, sign it in your wallet, then paste the signature below.',
                            style: PokerTypography.bodySmall
                                .copyWith(color: PokerColors.textSecondary),
                          ),
                  ),
                  const SizedBox(height: PokerSpacing.lg),
                  Text(
                    'Wallet Signature',
                    style: PokerTypography.titleSmall,
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  TextField(
                    controller: _signatureController,
                    maxLines: 4,
                    decoration: const InputDecoration(
                      hintText: 'Paste the base64 signature for the code above',
                    ),
                    style: PokerTypography.bodyMedium,
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: PokerSpacing.lg),
                    _StatusCard(
                      icon: Icons.error_outline,
                      color: PokerColors.danger,
                      message: _error!,
                    ),
                  ],
                  if (_success != null) ...[
                    const SizedBox(height: PokerSpacing.lg),
                    _StatusCard(
                      icon: Icons.check_circle_outline,
                      color: PokerColors.success,
                      message: _success!,
                    ),
                  ],
                  const SizedBox(height: PokerSpacing.xl),
                  Align(
                    alignment: Alignment.centerRight,
                    child: SizedBox(
                      width: 220,
                      child: ElevatedButton(
                        onPressed: _isSubmitting ? null : _submit,
                        style: ElevatedButton.styleFrom(
                          minimumSize: const Size.fromHeight(48),
                        ),
                        child: _isSubmitting
                            ? const SizedBox(
                                height: 18,
                                width: 18,
                                child:
                                    CircularProgressIndicator(strokeWidth: 2),
                              )
                            : const Text('Verify & Save'),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _StatusCard extends StatelessWidget {
  const _StatusCard({
    required this.icon,
    required this.color,
    required this.message,
  });

  final IconData icon;
  final Color color;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: color, size: 20),
          const SizedBox(width: PokerSpacing.md),
          Expanded(
            child: SelectableText(
              message,
              style: PokerTypography.bodyMedium.copyWith(color: color),
            ),
          ),
        ],
      ),
    );
  }
}

class _InfoTooltip extends StatelessWidget {
  const _InfoTooltip({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: message,
      preferBelow: false,
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          color: PokerColors.surfaceBright,
          shape: BoxShape.circle,
          border: Border.all(color: PokerColors.borderSubtle),
        ),
        alignment: Alignment.center,
        child: const Icon(
          Icons.info_outline,
          size: 18,
          color: PokerColors.textSecondary,
        ),
      ),
    );
  }
}
