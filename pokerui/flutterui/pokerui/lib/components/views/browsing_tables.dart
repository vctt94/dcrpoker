import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/dialogs/create_table.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

enum _SortBy { players, blinds, buyIn }

class BrowsingTablesView extends StatefulWidget {
  const BrowsingTablesView({super.key, required this.model});
  final PokerModel model;

  @override
  State<BrowsingTablesView> createState() => _BrowsingTablesViewState();
}

class _BrowsingTablesViewState extends State<BrowsingTablesView> {
  _SortBy _sort = _SortBy.players;
  bool _hideFull = false;
  bool _showWaitingOnly = false;

  String _shortId(String s, [int n = 8]) =>
      s.isEmpty ? '' : (s.length <= n ? s : s.substring(0, n));
  double _toDcr(int atoms) => atoms / 1e8;

  List<UiTable> get _filteredSortedTables {
    var list = widget.model.tables;
    if (_hideFull) list = list.where((t) => t.currentPlayers < t.maxPlayers).toList();
    if (_showWaitingOnly) list = list.where((t) => !t.gameStarted).toList();
    switch (_sort) {
      case _SortBy.players:
        list = List.of(list)..sort((a, b) => b.currentPlayers.compareTo(a.currentPlayers));
        break;
      case _SortBy.blinds:
        list = List.of(list)..sort((a, b) => b.bigBlind.compareTo(a.bigBlind));
        break;
      case _SortBy.buyIn:
        list = List.of(list)..sort((a, b) => a.buyInAtoms.compareTo(b.buyInAtoms));
        break;
    }
    return list;
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;

    if (model.tables.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.style, size: 64, color: PokerColors.textMuted),
            const SizedBox(height: PokerSpacing.lg),
            Text('No Tables', style: PokerTypography.headlineMedium),
            const SizedBox(height: PokerSpacing.sm),
            Text('Create one to start playing', style: PokerTypography.bodySmall),
            const SizedBox(height: PokerSpacing.xl),
            ElevatedButton.icon(
              onPressed: () => CreateTableDialog.open(context, model),
              icon: const Icon(Icons.add, size: 18),
              label: const Text('Create Table'),
            ),
          ],
        ),
      );
    }

    final tables = _filteredSortedTables;
    final screenWidth = MediaQuery.of(context).size.width;
    final crossAxisCount = screenWidth > 900 ? 3 : (screenWidth > 600 ? 2 : 1);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Toolbar
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: PokerSpacing.lg),
          child: Row(
            children: [
              Expanded(
                child: Wrap(
                  spacing: PokerSpacing.sm,
                  runSpacing: PokerSpacing.sm,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    _FilterChip(
                      label: 'Hide full',
                      selected: _hideFull,
                      onSelected: (v) => setState(() => _hideFull = v),
                    ),
                    _FilterChip(
                      label: 'Waiting',
                      selected: _showWaitingOnly,
                      onSelected: (v) => setState(() => _showWaitingOnly = v),
                    ),
                    IconButton(
                      tooltip: 'Refresh',
                      onPressed: model.browseTables,
                      icon: const Icon(Icons.refresh, color: PokerColors.textSecondary, size: 20),
                    ),
                  ],
                ),
              ),
              ElevatedButton.icon(
                onPressed: () => CreateTableDialog.open(context, model),
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Create'),
              ),
            ],
          ),
        ),
        const SizedBox(height: PokerSpacing.md),

        // Table grid
        if (crossAxisCount > 1)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: PokerSpacing.lg),
            child: GridView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: crossAxisCount,
                crossAxisSpacing: PokerSpacing.md,
                mainAxisSpacing: PokerSpacing.md,
                // Fixed card height is more stable than aspect ratio here:
                // table metadata plus the join action do not compress well.
                mainAxisExtent: 168,
              ),
              itemCount: tables.length,
              itemBuilder: (context, i) => _TableCard(
                table: tables[i],
                model: model,
                shortId: _shortId,
                toDcr: _toDcr,
              ),
            ),
          )
        else
          ListView.builder(
            padding: const EdgeInsets.symmetric(horizontal: PokerSpacing.lg),
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: tables.length,
            itemBuilder: (context, i) => Padding(
              padding: const EdgeInsets.only(bottom: PokerSpacing.md),
              child: _TableCard(
                table: tables[i],
                model: model,
                shortId: _shortId,
                toDcr: _toDcr,
              ),
            ),
          ),
      ],
    );
  }
}

class _FilterChip extends StatelessWidget {
  const _FilterChip({required this.label, required this.selected, required this.onSelected});
  final String label;
  final bool selected;
  final ValueChanged<bool> onSelected;

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      label: Text(label, style: PokerTypography.labelSmall.copyWith(
        color: selected ? PokerColors.primary : PokerColors.textSecondary,
      )),
      selected: selected,
      onSelected: onSelected,
      selectedColor: PokerColors.primary.withOpacity(0.15),
      checkmarkColor: PokerColors.primary,
      backgroundColor: PokerColors.surface,
      side: BorderSide(
        color: selected ? PokerColors.primary.withOpacity(0.5) : PokerColors.borderSubtle,
      ),
    );
  }
}

class _TableCard extends StatelessWidget {
  const _TableCard({
    required this.table,
    required this.model,
    required this.shortId,
    required this.toDcr,
  });
  final UiTable table;
  final PokerModel model;
  final String Function(String, [int]) shortId;
  final double Function(int) toDcr;

  @override
  Widget build(BuildContext context) {
    final t = table;
    final full = t.currentPlayers >= t.maxPlayers;
    final started = t.gameStarted;
    final alreadySeated = model.currentTableId == t.id ||
        t.players.any((p) => p.id == model.playerId);
    final canJoin = alreadySeated || (!started && !full);

    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final fillHeight = constraints.hasBoundedHeight;
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: fillHeight ? MainAxisSize.max : MainAxisSize.min,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      'Table ${shortId(t.id)}',
                      style: PokerTypography.titleSmall,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  _StatusPill(started: started),
                ],
              ),
              const SizedBox(height: PokerSpacing.md),

              // Stats row
              Wrap(
                spacing: PokerSpacing.md,
                runSpacing: PokerSpacing.xs,
                children: [
                  _Stat(icon: Icons.people_outline, text: '${t.currentPlayers}/${t.maxPlayers}'),
                  _Stat(icon: Icons.toll, text: '${t.smallBlind}/${t.bigBlind}'),
                  _Stat(icon: Icons.account_balance_wallet_outlined,
                      text: '${toDcr(t.buyInAtoms).toStringAsFixed(2)} DCR'),
                ],
              ),
              const SizedBox(height: PokerSpacing.md),

              // Player count bar
              ClipRRect(
                borderRadius: BorderRadius.circular(3),
                child: LinearProgressIndicator(
                  minHeight: 4,
                  value: t.maxPlayers == 0
                      ? 0
                      : (t.currentPlayers / t.maxPlayers).clamp(0.0, 1.0),
                  backgroundColor: PokerColors.borderSubtle,
                  valueColor: AlwaysStoppedAnimation(
                      full ? PokerColors.danger : PokerColors.primary),
                ),
              ),
              if (fillHeight)
                const Spacer()
              else
                const SizedBox(height: PokerSpacing.md),

              // Join button
              Align(
                alignment: Alignment.centerRight,
                child: ElevatedButton(
                  onPressed: canJoin
                      ? () {
                          if (alreadySeated) {
                            model.openTableView();
                          } else {
                            model.joinTable(t.id);
                          }
                        }
                      : null,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: canJoin ? PokerColors.success : PokerColors.surfaceBright,
                    foregroundColor: canJoin ? Colors.black : PokerColors.textMuted,
                    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                  ),
                  child: Text(alreadySeated
                      ? 'Open'
                      : started
                          ? 'Started'
                          : 'Join'),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.started});
  final bool started;

  @override
  Widget build(BuildContext context) {
    final color = started ? PokerColors.success : PokerColors.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withOpacity(0.4)),
      ),
      child: Text(
        started ? 'Playing' : 'Waiting',
        style: PokerTypography.labelSmall.copyWith(color: color, fontSize: 10),
      ),
    );
  }
}

class _Stat extends StatelessWidget {
  const _Stat({required this.icon, required this.text});
  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: PokerColors.textMuted),
        const SizedBox(width: 4),
        Text(text, style: PokerTypography.bodySmall),
      ],
    );
  }
}
