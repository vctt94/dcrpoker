import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/screens/open_escrow.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

String _shortEscrowText(String s, [int n = 8]) =>
    s.isEmpty ? '' : (s.length <= n ? s : s.substring(0, n));

int _escrowAsInt(dynamic v) {
  if (v is int) return v;
  if (v is num) return v.toInt();
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}

bool _escrowHasRequiredConfirmations(Map<String, dynamic> escrow) {
  final confs = _escrowAsInt(escrow['confs']);
  final required = _escrowAsInt(escrow['required_confirmations']);
  return confs >= (required == 0 ? 1 : required);
}

bool _escrowMatchesBuyIn(Map<String, dynamic> escrow, int buyInAtoms) =>
    _escrowAsInt(escrow['funded_amount']) == buyInAtoms;

class InLobbyView extends StatelessWidget {
  const InLobbyView({super.key, required this.model});
  final PokerModel model;

  String _tableTitle(UiTable table) {
    final name = table.name.trim();
    if (name.isNotEmpty) {
      return name;
    }
    return 'Table ${_shortEscrowText(table.id)}';
  }

  Future<void> _showLeaveTableDialog(BuildContext ctx) async {
    final actionLabel = model.isSeated ? 'Leave Table' : 'Stop Watching';
    final actionMessage = model.isSeated
        ? 'You will need to rejoin to play again.'
        : 'You will stop receiving live updates for this table.';
    if (!ctx.mounted) return;
    final confirmed = await showDialog<bool>(
      context: ctx,
      builder: (dctx) => AlertDialog(
        title: Text('$actionLabel?'),
        content: Text(actionMessage),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(dctx, false),
              child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () => Navigator.pop(dctx, true),
            style:
                ElevatedButton.styleFrom(backgroundColor: PokerColors.danger),
            child: Text(model.isSeated ? 'Leave' : 'Stop'),
          ),
        ],
      ),
    );
    if (confirmed == true && ctx.mounted) await model.leaveTable();
  }

  Future<void> _showBindDialog(BuildContext ctx, UiTable t) async {
    if (!model.hasAuthedPayoutAddress) {
      if (!ctx.mounted) return;
      await showDialog(
        context: ctx,
        builder: (dctx) => AlertDialog(
          title: const Text('Sign Address Required'),
          content: const Text('Please sign a payout address first.'),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(dctx),
                child: const Text('Later')),
            ElevatedButton(
              onPressed: () {
                Navigator.pop(dctx);
                Navigator.pushNamed(ctx, '/sign-address');
              },
              child: const Text('Sign Address'),
            ),
          ],
        ),
      );
      return;
    }
    final escrows = await model.refreshBindableEscrows();
    if (!ctx.mounted) return;
    final hasMatchingEscrow =
        escrows.any((escrow) => _escrowMatchesBuyIn(escrow, t.buyInAtoms));
    if (!hasMatchingEscrow) {
      final tableName = t.name.trim().isNotEmpty
          ? t.name.trim()
          : 'Table ${_shortEscrowText(t.id)}';
      await showDialog<bool>(
        context: ctx,
        builder: (dctx) => OpenEscrowDialog(
          tableName: tableName,
          buyInAtoms: t.buyInAtoms,
        ),
      );
      return;
    }
    await showDialog(
      context: ctx,
      builder: (dctx) => _BindEscrowDialog(
        model: model,
        table: t,
        initialEscrows: escrows,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final tableId = model.currentTableId;
    final table = model.tables.firstWhere(
      (t) => t.id == tableId,
      orElse: () => UiTable(
        id: tableId ?? '',
        name: '',
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
    final watchingOnly = !model.isSeated && model.isWatching;
    final mePlayer =
        displayedPlayers.firstWhereOrNull((p) => p.id == model.playerId);

    // Compute progress steps
    final hasEscrow = (mePlayer?.escrowId ?? '').isNotEmpty;
    final escrowReady = mePlayer?.escrowReady ?? false;
    final presignDone = model.presignCompleted;
    final allReady = displayedPlayers.every((p) => p.isReady);
    final allEscrows =
        displayedPlayers.every((p) => p.escrowId.isNotEmpty && p.escrowReady);
    final allPresigned = displayedPlayers.every((p) => p.presignComplete);
    final enoughPlayers = displayedPlayers.length >= 2;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Table header
              Container(
                padding: const EdgeInsets.all(PokerSpacing.lg),
                decoration: BoxDecoration(
                  color: PokerColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: PokerColors.borderSubtle),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            _tableTitle(table),
                            style: PokerTypography.titleLarge,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        if (watchingOnly)
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: PokerSpacing.md,
                              vertical: PokerSpacing.sm,
                            ),
                            decoration: BoxDecoration(
                              color: PokerColors.primary.withOpacity(0.12),
                              borderRadius: BorderRadius.circular(999),
                              border: Border.all(
                                color: PokerColors.primary.withOpacity(0.35),
                              ),
                            ),
                            child: Text(
                              'Watching',
                              style: PokerTypography.labelSmall
                                  .copyWith(color: PokerColors.primary),
                            ),
                          )
                        else
                          ElevatedButton(
                            onPressed: model.iAmReady
                                ? model.setUnready
                                : model.setReady,
                            style: ElevatedButton.styleFrom(
                              backgroundColor: model.iAmReady
                                  ? PokerColors.surfaceBright
                                  : PokerColors.success,
                              foregroundColor: model.iAmReady
                                  ? PokerColors.textPrimary
                                  : Colors.black,
                            ),
                            child: Text(model.iAmReady ? 'Unready' : 'Ready'),
                          ),
                      ],
                    ),
                    const SizedBox(height: PokerSpacing.sm),
                    Text(
                      'Blinds ${table.smallBlind}/${table.bigBlind}  •  Buy-in ${(table.buyInAtoms / 1e8).toStringAsFixed(4)} DCR',
                      style: PokerTypography.bodySmall,
                    ),
                    if (watchingOnly) ...[
                      const SizedBox(height: PokerSpacing.sm),
                      Text(
                        'Watchers receive table and hand updates but cannot bind escrow, ready up, or act in the hand.',
                        style: PokerTypography.bodySmall
                            .copyWith(color: PokerColors.textMuted),
                      ),
                    ],
                  ],
                ),
              ),

              const SizedBox(height: PokerSpacing.lg),

              if (!watchingOnly)
                _ProgressStepper(
                  steps: [
                    _Step(
                      label: 'Fund',
                      detail: hasEscrow
                          ? (escrowReady
                              ? 'Escrow funded'
                              : 'Waiting for confirmations')
                          : 'Escrow required',
                      done: hasEscrow && escrowReady,
                      active: !hasEscrow || !escrowReady,
                      action: (!hasEscrow || !escrowReady)
                          ? () => _showBindDialog(context, table)
                          : null,
                      actionLabel: hasEscrow ? null : 'Bind Escrow',
                    ),
                    _Step(
                      label: 'Ready',
                      detail: allReady
                          ? 'All players ready'
                          : '${displayedPlayers.where((p) => p.isReady).length}/${displayedPlayers.length} ready',
                      done: allReady && allEscrows && enoughPlayers,
                      active: hasEscrow && escrowReady,
                    ),
                    _Step(
                      label: 'Go',
                      detail: allPresigned
                          ? 'Starting!'
                          : (model.presignInProgress
                              ? 'Presigning...'
                              : 'Waiting'),
                      done: allPresigned,
                      active: allReady && allEscrows,
                      showSpinner: model.presignInProgress,
                    ),
                  ],
                ),

              const SizedBox(height: PokerSpacing.lg),

              // Players
              Container(
                padding: const EdgeInsets.all(PokerSpacing.lg),
                decoration: BoxDecoration(
                  color: PokerColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: PokerColors.borderSubtle),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Players',
                        style: PokerTypography.titleSmall.copyWith(
                          color: PokerColors.textSecondary,
                        )),
                    const SizedBox(height: PokerSpacing.md),
                    if (displayedPlayers.isEmpty)
                      Text('Waiting for players...',
                          style: PokerTypography.bodySmall)
                    else
                      Wrap(
                        spacing: PokerSpacing.sm,
                        runSpacing: PokerSpacing.sm,
                        children: displayedPlayers
                            .map(
                              (p) => _PlayerChip(
                                  player: p, isMe: p.id == model.playerId),
                            )
                            .toList(),
                      ),
                  ],
                ),
              ),

              // Error
              if (model.errorMessage.isNotEmpty) ...[
                const SizedBox(height: PokerSpacing.md),
                Container(
                  padding: const EdgeInsets.all(PokerSpacing.md),
                  decoration: BoxDecoration(
                    color: PokerColors.danger.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(10),
                    border:
                        Border.all(color: PokerColors.danger.withOpacity(0.3)),
                  ),
                  child: Row(
                    children: [
                      Icon(Icons.error_outline,
                          color: PokerColors.danger, size: 18),
                      const SizedBox(width: PokerSpacing.sm),
                      Expanded(
                          child: SelectableText(model.errorMessage,
                              style: PokerTypography.bodySmall
                                  .copyWith(color: PokerColors.danger))),
                      IconButton(
                        icon: Icon(Icons.copy,
                            color: PokerColors.danger, size: 14),
                        onPressed: () async {
                          await Clipboard.setData(
                              ClipboardData(text: model.errorMessage));
                          if (!context.mounted) return;
                          ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Copied')));
                        },
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(),
                      ),
                      IconButton(
                        icon: Icon(Icons.close,
                            color: PokerColors.danger, size: 14),
                        onPressed: model.clearError,
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(),
                      ),
                    ],
                  ),
                ),
              ],

              const SizedBox(height: PokerSpacing.xl),
              Align(
                alignment: Alignment.centerRight,
                child: TextButton(
                  onPressed: () => _showLeaveTableDialog(context),
                  style:
                      TextButton.styleFrom(foregroundColor: PokerColors.danger),
                  child: Text(model.isSeated ? 'Leave Table' : 'Stop Watching'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _BindEscrowDialog extends StatefulWidget {
  const _BindEscrowDialog({
    required this.model,
    required this.table,
    required this.initialEscrows,
  });

  final PokerModel model;
  final UiTable table;
  final List<Map<String, dynamic>> initialEscrows;

  @override
  State<_BindEscrowDialog> createState() => _BindEscrowDialogState();
}

class _BindEscrowDialogState extends State<_BindEscrowDialog> {
  final _formKey = GlobalKey<FormState>();
  final _escrowCtrl = TextEditingController();
  bool _showAdvancedOptions = false;
  String? _selectedOutpoint;
  List<_BindableEscrowOption> _escrowOptions = const [];

  @override
  void initState() {
    super.initState();
    _applyEscrows(widget.initialEscrows);
    widget.model.bindableEscrowsListenable.addListener(_handleEscrowsChanged);
  }

  @override
  void dispose() {
    widget.model.bindableEscrowsListenable
        .removeListener(_handleEscrowsChanged);
    _escrowCtrl.dispose();
    super.dispose();
  }

  void _handleEscrowsChanged() {
    if (!mounted) return;
    final hasTransientRouteAboveDialog =
        !(ModalRoute.of(context)?.isCurrent ?? true);
    _applyEscrows(widget.model.bindableEscrows);
    if (hasTransientRouteAboveDialog) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (!(ModalRoute.of(context)?.isCurrent ?? true)) {
          Navigator.of(context).maybePop();
        }
      });
    }
  }

  void _applyEscrows(List<Map<String, dynamic>> escrows) {
    final options = escrows.where((e) {
      final fundingState = (e['funding_state'] ?? '').toString().toUpperCase();
      return fundingState != 'ESCROW_STATE_INVALID' &&
          _escrowMatchesBuyIn(e, widget.table.buyInAtoms);
    }).map((e) {
      final txid = (e['funding_txid'] ?? '').toString();
      final vout = _escrowAsInt(e['funding_vout']);
      final amountRaw = e['funded_amount'];
      final amount = amountRaw is num
          ? amountRaw.toDouble()
          : double.tryParse(amountRaw.toString()) ?? 0;
      return _BindableEscrowOption(
        outpoint: '$txid:$vout',
        label:
            '${_shortEscrowText(txid)}:$vout - ${(amount / 1e8).toStringAsFixed(4)} DCR',
        confirmed: _escrowHasRequiredConfirmations(e),
      );
    }).toList(growable: false);

    var nextSelectedOutpoint = _selectedOutpoint;
    final selectedStillConfirmed = options.any(
      (opt) => opt.outpoint == nextSelectedOutpoint && opt.confirmed,
    );
    if (!selectedStillConfirmed) {
      nextSelectedOutpoint = null;
      for (final option in options) {
        if (option.confirmed) {
          nextSelectedOutpoint = option.outpoint;
          break;
        }
      }
    }

    setState(() {
      _escrowOptions = options;
      _selectedOutpoint = nextSelectedOutpoint;
    });
  }

  @override
  Widget build(BuildContext context) {
    final pendingCount = _escrowOptions.where((opt) => !opt.confirmed).length;
    final hasMatchingEscrows = _escrowOptions.isNotEmpty;

    return AlertDialog(
      title: const Text('Bind Escrow'),
      content: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Buy-in ${(widget.table.buyInAtoms / 1e8).toStringAsFixed(4)} DCR',
              style: PokerTypography.bodySmall,
            ),
            const SizedBox(height: PokerSpacing.md),
            if (!hasMatchingEscrows)
              Text(
                'No escrows match this table buy-in right now. Open and fund an escrow for ${(widget.table.buyInAtoms / 1e8).toStringAsFixed(4)} DCR first.',
                style: PokerTypography.bodySmall,
              )
            else
              DropdownButtonFormField<String>(
                value: _selectedOutpoint,
                isExpanded: true,
                decoration:
                    const InputDecoration(labelText: 'Funding outpoint'),
                validator: (_) {
                  final chosen = _escrowCtrl.text.trim().isNotEmpty
                      ? _escrowCtrl.text.trim()
                      : (_selectedOutpoint ?? '').trim();
                  return chosen.isEmpty ? 'Required' : null;
                },
                selectedItemBuilder: (context) => _escrowOptions
                    .map(
                      (opt) => Align(
                        alignment: Alignment.centerLeft,
                        child: Text(
                          opt.label,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    )
                    .toList(growable: false),
                items: _escrowOptions
                    .map((opt) => DropdownMenuItem<String>(
                          value: opt.outpoint,
                          enabled: opt.confirmed,
                          child: Text(
                            opt.confirmed
                                ? opt.label
                                : '${opt.label} • Waiting for confirmations',
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color:
                                  opt.confirmed ? null : PokerColors.textMuted,
                            ),
                          ),
                        ))
                    .toList(),
                onChanged: (value) {
                  setState(() {
                    _selectedOutpoint = value;
                  });
                },
              ),
            if (pendingCount > 0) ...[
              const SizedBox(height: PokerSpacing.sm),
              Text(
                pendingCount == 1
                    ? 'One escrow is still waiting for confirmations and cannot be selected yet.'
                    : '$pendingCount escrows are still waiting for confirmations and cannot be selected yet.',
                style: PokerTypography.bodySmall
                    .copyWith(color: PokerColors.textMuted),
              ),
            ],
            if (hasMatchingEscrows) ...[
              const SizedBox(height: PokerSpacing.sm),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton.icon(
                  onPressed: () {
                    setState(() {
                      _showAdvancedOptions = !_showAdvancedOptions;
                    });
                  },
                  style: TextButton.styleFrom(
                    padding: EdgeInsets.zero,
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  icon: Icon(
                    _showAdvancedOptions
                        ? Icons.expand_less
                        : Icons.expand_more,
                    size: 18,
                  ),
                  label: const Text('Advanced options'),
                ),
              ),
              if (_showAdvancedOptions) ...[
                const SizedBox(height: PokerSpacing.sm),
                TextFormField(
                  controller: _escrowCtrl,
                  decoration: const InputDecoration(
                    labelText: 'Override outpoint',
                    hintText: 'Paste a manual txid:vout outpoint',
                  ),
                  validator: (_) {
                    final chosen = _escrowCtrl.text.trim().isNotEmpty
                        ? _escrowCtrl.text.trim()
                        : (_selectedOutpoint ?? '').trim();
                    return chosen.isEmpty ? 'Required' : null;
                  },
                ),
              ],
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        ElevatedButton(
          onPressed: () async {
            if (!hasMatchingEscrows) {
              final tableName = widget.table.name.trim().isNotEmpty
                  ? widget.table.name.trim()
                  : 'Table ${_shortEscrowText(widget.table.id)}';
              final messenger = ScaffoldMessenger.maybeOf(context);
              final navigator = Navigator.of(context);
              final opened = await showDialog<bool>(
                context: context,
                builder: (dctx) => OpenEscrowDialog(
                  tableName: tableName,
                  buyInAtoms: widget.table.buyInAtoms,
                ),
              );
              if (!mounted) return;
              if (opened == true) {
                messenger?.showSnackBar(
                  SnackBar(
                    content: Text(
                      'Escrow opened for ${(widget.table.buyInAtoms / 1e8).toStringAsFixed(4)} DCR. Fund it, then bind it once it confirms.',
                    ),
                  ),
                );
                navigator.pop();
              }
              return;
            }
            final chosenOutpoint = _escrowCtrl.text.trim().isNotEmpty
                ? _escrowCtrl.text.trim()
                : (_selectedOutpoint ?? '').trim();
            if (chosenOutpoint.isEmpty) {
              setState(() {
                _showAdvancedOptions = true;
              });
              return;
            }
            if (!(_formKey.currentState?.validate() ?? false)) return;
            Navigator.pop(context);
            await widget.model.bindEscrow(
              tableId: widget.table.id,
              outpoint: chosenOutpoint,
            );
          },
          child: Text(hasMatchingEscrows ? 'Bind' : 'Open Escrow'),
        ),
      ],
    );
  }
}

class _BindableEscrowOption {
  const _BindableEscrowOption({
    required this.outpoint,
    required this.label,
    required this.confirmed,
  });

  final String outpoint;
  final String label;
  final bool confirmed;
}

// ── Progress Stepper ──

class _Step {
  final String label, detail;
  final bool done, active;
  final VoidCallback? action;
  final String? actionLabel;
  final bool showSpinner;
  const _Step({
    required this.label,
    required this.detail,
    this.done = false,
    this.active = false,
    this.action,
    this.actionLabel,
    this.showSpinner = false,
  });
}

class _ProgressStepper extends StatelessWidget {
  const _ProgressStepper({required this.steps});
  final List<_Step> steps;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Row(
        children: [
          for (int i = 0; i < steps.length; i++) ...[
            Expanded(child: _StepTile(step: steps[i], index: i + 1)),
            if (i < steps.length - 1)
              Container(
                width: 32,
                height: 2,
                color: steps[i].done
                    ? PokerColors.success
                    : PokerColors.borderSubtle,
              ),
          ],
        ],
      ),
    );
  }
}

class _StepTile extends StatelessWidget {
  const _StepTile({required this.step, required this.index});
  final _Step step;
  final int index;

  @override
  Widget build(BuildContext context) {
    final color = step.done
        ? PokerColors.success
        : (step.active ? PokerColors.primary : PokerColors.textMuted);

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 36,
          height: 36,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: step.done
                ? PokerColors.success.withOpacity(0.15)
                : PokerColors.surfaceBright,
            border: Border.all(color: color, width: 2),
          ),
          child: Center(
            child: step.done
                ? Icon(Icons.check, color: color, size: 18)
                : (step.showSpinner
                    ? SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: color))
                    : Text('$index',
                        style:
                            PokerTypography.labelLarge.copyWith(color: color))),
          ),
        ),
        const SizedBox(height: PokerSpacing.sm),
        Text(step.label,
            style: PokerTypography.labelSmall.copyWith(color: color)),
        const SizedBox(height: PokerSpacing.xxs),
        Text(
          step.detail,
          style: PokerTypography.bodySmall
              .copyWith(fontSize: 10, color: PokerColors.textMuted),
          textAlign: TextAlign.center,
        ),
        if (step.action != null && step.actionLabel != null) ...[
          const SizedBox(height: PokerSpacing.sm),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton(
              onPressed: step.action,
              style: OutlinedButton.styleFrom(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                minimumSize: const Size.fromHeight(36),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                side: BorderSide(color: color.withOpacity(0.7)),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(999),
                ),
              ),
              child: Text(
                step.actionLabel!,
                style: PokerTypography.labelSmall.copyWith(color: color),
              ),
            ),
          ),
        ],
      ],
    );
  }
}

// ── Player Chip ──

class _PlayerChip extends StatelessWidget {
  const _PlayerChip({required this.player, required this.isMe});
  final UiPlayer player;
  final bool isMe;

  @override
  Widget build(BuildContext context) {
    final name = player.name.trim().isNotEmpty
        ? player.name.trim()
        : player.id.substring(0, 8);
    final ready = player.isReady;
    final color = ready ? PokerColors.success : PokerColors.warning;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withOpacity(0.08),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: color.withOpacity(isMe ? 0.6 : 0.3),
          width: isMe ? 1.5 : 1,
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            ready ? Icons.check_circle : Icons.hourglass_empty,
            size: 14,
            color: color,
          ),
          const SizedBox(width: 6),
          Text(
            name.length > 14 ? '${name.substring(0, 14)}...' : name,
            style: PokerTypography.labelSmall
                .copyWith(color: PokerColors.textPrimary),
          ),
          if (isMe) ...[
            const SizedBox(width: 4),
            Text('(you)',
                style: PokerTypography.bodySmall.copyWith(
                  fontSize: 10,
                  color: PokerColors.primary,
                )),
          ],
        ],
      ),
    );
  }
}
