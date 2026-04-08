import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

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
  bool _showSessionPrivateKey = false;
  final Set<String> _revealedEscrowFields = <String>{};

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
      _showSessionPrivateKey = false;
      _revealedEscrowFields.clear();
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
        _revealedEscrowFields.clear();
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

  void debugSetSessionKeyForTest({
    required String publicKey,
    required String privateKey,
    required String keyIndex,
  }) {
    setState(() {
      _compPubController.text = publicKey;
      _compPrivController.text = privateKey;
      _keyIndex = keyIndex;
      _showSessionPrivateKey = false;
    });
  }

  void debugSetEscrowResultForTest(Map<String, dynamic> result) {
    setState(() {
      _result = Map<String, dynamic>.from(result);
      _revealedEscrowFields.clear();
    });
  }

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Open Escrow',
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
                    'Fund escrow',
                    style: PokerTypography.headlineMedium,
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  Text(
                    'Create a funded escrow before joining a table. A session key is generated automatically when the escrow is opened.',
                    style: PokerTypography.bodyMedium
                        .copyWith(color: PokerColors.textSecondary),
                  ),
                  if ((_payoutAddress ?? '').trim().isNotEmpty) ...[
                    const SizedBox(height: PokerSpacing.md),
                    Text(
                      'Verified payout address: $_payoutAddress',
                      style: PokerTypography.bodySmall
                          .copyWith(color: PokerColors.success),
                    ),
                  ],
                  const SizedBox(height: PokerSpacing.xl),
                  Text(
                    'Bet Amount (DCR)',
                    style: PokerTypography.titleSmall,
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  TextField(
                    controller: _betDcrController,
                    decoration: const InputDecoration(
                      hintText: '0.10',
                    ),
                    keyboardType:
                        const TextInputType.numberWithOptions(decimal: true),
                    style: PokerTypography.bodyMedium,
                  ),
                  const SizedBox(height: PokerSpacing.lg),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          'CSV Blocks',
                          style: PokerTypography.titleSmall,
                        ),
                      ),
                      const _EscrowInfoTooltip(
                        message:
                            'CSV blocks define the relative lock time used for escrow recovery. The default of 64 is suitable for normal play.',
                      ),
                    ],
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  TextField(
                    controller: _csvBlocksController,
                    decoration: const InputDecoration(
                      hintText: '64',
                    ),
                    keyboardType: TextInputType.number,
                    style: PokerTypography.bodyMedium,
                  ),
                  const SizedBox(height: PokerSpacing.xl),
                  if (_error != null) ...[
                    _EscrowStatusCard(
                      icon: Icons.error_outline,
                      color: PokerColors.danger,
                      message: _error!,
                    ),
                    const SizedBox(height: PokerSpacing.lg),
                  ],
                  if (_needsPayoutAddress) ...[
                    Container(
                      width: double.infinity,
                      padding: const EdgeInsets.all(PokerSpacing.lg),
                      decoration: BoxDecoration(
                        color: PokerColors.danger.withValues(alpha: 0.1),
                        borderRadius: BorderRadius.circular(14),
                        border: Border.all(
                          color: PokerColors.danger.withValues(alpha: 0.25),
                        ),
                      ),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Icon(
                            Icons.warning_amber_rounded,
                            color: PokerColors.danger,
                          ),
                          const SizedBox(width: PokerSpacing.md),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  'Verify a payout address first',
                                  style: PokerTypography.titleSmall
                                      .copyWith(color: PokerColors.danger),
                                ),
                                const SizedBox(height: PokerSpacing.xs),
                                Text(
                                  'Open Sign Address to verify the address that will receive escrow settlements.',
                                  style: PokerTypography.bodySmall.copyWith(
                                    color: PokerColors.textSecondary,
                                  ),
                                ),
                                const SizedBox(height: PokerSpacing.md),
                                Align(
                                  alignment: Alignment.centerLeft,
                                  child: ElevatedButton(
                                    onPressed: () {
                                      Navigator.of(context)
                                          .pushNamed('/sign-address');
                                    },
                                    style: ElevatedButton.styleFrom(
                                      backgroundColor: PokerColors.danger,
                                    ),
                                    child: const Text('Open Sign Address'),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: PokerSpacing.lg),
                  ],
                  Align(
                    alignment: Alignment.centerRight,
                    child: SizedBox(
                      width: 220,
                      child: ElevatedButton(
                        onPressed: _isOpening ? null : _openEscrow,
                        style: ElevatedButton.styleFrom(
                          minimumSize: const Size.fromHeight(48),
                        ),
                        child: _isOpening
                            ? const SizedBox(
                                height: 18,
                                width: 18,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              )
                            : const Text('Open Escrow'),
                      ),
                    ),
                  ),
                  if (_result != null) ...[
                    const SizedBox(height: PokerSpacing.xl),
                    _EscrowSurfaceCard(
                      title: 'Escrow Created',
                      child: Column(
                        children: _result!.entries
                            .map(
                              (e) => _escrowResultRow(e.key, '${e.value}'),
                            )
                            .toList(),
                      ),
                    ),
                  ],
                  if (_compPubController.text.trim().isNotEmpty ||
                      _compPrivController.text.trim().isNotEmpty ||
                      (_keyIndex ?? '').isNotEmpty) ...[
                    const SizedBox(height: PokerSpacing.lg),
                    _EscrowSurfaceCard(
                      title: 'Session Key',
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          _keyValueRow(
                            'Compressed Pubkey',
                            _compPubController.text.trim(),
                          ),
                          _sensitiveKeyValueRow(
                            'Session Private Key',
                            _compPrivController.text.trim(),
                            revealed: _showSessionPrivateKey,
                            onToggle: () {
                              setState(() {
                                _showSessionPrivateKey =
                                    !_showSessionPrivateKey;
                              });
                            },
                          ),
                          if ((_keyIndex ?? '').isNotEmpty)
                            Padding(
                              padding:
                                  const EdgeInsets.only(top: PokerSpacing.sm),
                              child: Text(
                                'Key index: $_keyIndex. Keep this with the escrow details for recovery.',
                                style: PokerTypography.bodySmall.copyWith(
                                  color: PokerColors.textSecondary,
                                ),
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
        ),
      ),
    );
  }

  Widget _keyValueRow(String label, String value) {
    if (value.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: PokerSpacing.xs),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 150,
            child: Text(label, style: PokerTypography.bodySmall),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: PokerTypography.bodyMedium,
            ),
          ),
          IconButton(
            icon: const Icon(Icons.copy),
            color: PokerColors.textSecondary,
            onPressed: () => Clipboard.setData(ClipboardData(text: value)),
          ),
        ],
      ),
    );
  }

  Widget _escrowResultRow(String label, String value) {
    if (_isSensitiveEscrowField(label)) {
      final revealed = _revealedEscrowFields.contains(label);
      return _sensitiveEscrowResultRow(
        label,
        value,
        revealed: revealed,
        onToggle: () {
          setState(() {
            if (revealed) {
              _revealedEscrowFields.remove(label);
            } else {
              _revealedEscrowFields.add(label);
            }
          });
        },
      );
    }

    return Padding(
      padding: const EdgeInsets.symmetric(
        vertical: PokerSpacing.xs,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 140,
            child: Text(
              label,
              style: PokerTypography.bodySmall,
            ),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: PokerTypography.bodyMedium,
            ),
          ),
          IconButton(
            icon: const Icon(Icons.copy),
            color: PokerColors.textSecondary,
            onPressed: () => Clipboard.setData(
              ClipboardData(text: value),
            ),
          ),
        ],
      ),
    );
  }

  Widget _sensitiveEscrowResultRow(
    String label,
    String value, {
    required bool revealed,
    required VoidCallback onToggle,
  }) {
    final displayValue = revealed ? value : _maskedValue(value);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: PokerSpacing.xs),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 140,
            child: Text(label, style: PokerTypography.bodySmall),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SelectableText(
                  displayValue,
                  style: PokerTypography.bodyMedium.copyWith(
                    color: revealed
                        ? PokerColors.textPrimary
                        : PokerColors.textSecondary,
                    fontFamily: revealed ? null : 'monospace',
                  ),
                ),
                const SizedBox(height: PokerSpacing.xs),
                TextButton.icon(
                  onPressed: onToggle,
                  style: TextButton.styleFrom(
                    padding: EdgeInsets.zero,
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  icon: Icon(
                    revealed ? Icons.visibility_off_outlined : Icons.visibility,
                    size: 16,
                  ),
                  label: Text(revealed ? 'Hide' : 'Show'),
                ),
              ],
            ),
          ),
          if (revealed)
            IconButton(
              icon: const Icon(Icons.copy),
              color: PokerColors.textSecondary,
              onPressed: () => Clipboard.setData(
                ClipboardData(text: value),
              ),
            ),
        ],
      ),
    );
  }

  bool _isSensitiveEscrowField(String label) {
    return label.endsWith('_hex');
  }

  Widget _sensitiveKeyValueRow(
    String label,
    String value, {
    required bool revealed,
    required VoidCallback onToggle,
  }) {
    if (value.isEmpty) return const SizedBox.shrink();
    final displayValue = revealed ? value : _maskedValue(value);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: PokerSpacing.xs),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 150,
            child: Text(label, style: PokerTypography.bodySmall),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SelectableText(
                  displayValue,
                  style: PokerTypography.bodyMedium.copyWith(
                    color: revealed
                        ? PokerColors.textPrimary
                        : PokerColors.textSecondary,
                    fontFamily: revealed ? null : 'monospace',
                  ),
                ),
                const SizedBox(height: PokerSpacing.xs),
                TextButton.icon(
                  onPressed: onToggle,
                  style: TextButton.styleFrom(
                    padding: EdgeInsets.zero,
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  icon: Icon(
                    revealed ? Icons.visibility_off_outlined : Icons.visibility,
                    size: 16,
                  ),
                  label: Text(revealed ? 'Hide' : 'Show'),
                ),
              ],
            ),
          ),
          if (revealed)
            IconButton(
              icon: const Icon(Icons.copy),
              color: PokerColors.textSecondary,
              onPressed: () => Clipboard.setData(ClipboardData(text: value)),
            ),
        ],
      ),
    );
  }

  String _maskedValue(String value) {
    if (value.length <= 12) {
      return '••••••••';
    }
    return '${value.substring(0, 6)}••••••${value.substring(value.length - 6)}';
  }
}

class _EscrowSurfaceCard extends StatelessWidget {
  const _EscrowSurfaceCard({
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: PokerTypography.titleSmall),
          const SizedBox(height: PokerSpacing.md),
          child,
        ],
      ),
    );
  }
}

class _EscrowStatusCard extends StatelessWidget {
  const _EscrowStatusCard({
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

class _EscrowInfoTooltip extends StatelessWidget {
  const _EscrowInfoTooltip({required this.message});

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
