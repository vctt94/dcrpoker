import 'package:flutter/material.dart';
import 'package:pokerui/components/dialogs/create_table.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

enum _SortBy { players, blinds, buyIn }

enum _BuyInFilter { any, free, micro, medium, high }

enum _PlayerCountFilter { any, two, three, four, five, six }

extension on _SortBy {
  String get label {
    switch (this) {
      case _SortBy.players:
        return 'Most players';
      case _SortBy.blinds:
        return 'Highest blinds';
      case _SortBy.buyIn:
        return 'Lowest buy-in';
    }
  }
}

extension on _BuyInFilter {
  static const int _microMaxAtoms = 1000000; // 0.01 DCR
  static const int _mediumMaxAtoms = 10000000; // 0.10 DCR

  String get label {
    switch (this) {
      case _BuyInFilter.any:
        return 'Any';
      case _BuyInFilter.free:
        return 'Free';
      case _BuyInFilter.micro:
        return '<= 0.01 DCR';
      case _BuyInFilter.medium:
        return '0.01 - 0.10 DCR';
      case _BuyInFilter.high:
        return '> 0.10 DCR';
    }
  }

  bool matches(UiTable table) {
    switch (this) {
      case _BuyInFilter.any:
        return true;
      case _BuyInFilter.free:
        return table.buyInAtoms == 0;
      case _BuyInFilter.micro:
        return table.buyInAtoms > 0 && table.buyInAtoms <= _microMaxAtoms;
      case _BuyInFilter.medium:
        return table.buyInAtoms > _microMaxAtoms &&
            table.buyInAtoms <= _mediumMaxAtoms;
      case _BuyInFilter.high:
        return table.buyInAtoms > _mediumMaxAtoms;
    }
  }
}

extension on _PlayerCountFilter {
  String get label {
    switch (this) {
      case _PlayerCountFilter.any:
        return 'Any';
      case _PlayerCountFilter.two:
        return '2';
      case _PlayerCountFilter.three:
        return '3';
      case _PlayerCountFilter.four:
        return '4';
      case _PlayerCountFilter.five:
        return '5';
      case _PlayerCountFilter.six:
        return '6';
    }
  }

  int? get maxPlayers {
    switch (this) {
      case _PlayerCountFilter.any:
        return null;
      case _PlayerCountFilter.two:
        return 2;
      case _PlayerCountFilter.three:
        return 3;
      case _PlayerCountFilter.four:
        return 4;
      case _PlayerCountFilter.five:
        return 5;
      case _PlayerCountFilter.six:
        return 6;
    }
  }

  bool matches(UiTable table) {
    final expectedPlayers = maxPlayers;
    return expectedPlayers == null || table.maxPlayers == expectedPlayers;
  }
}

class BrowsingTablesView extends StatefulWidget {
  const BrowsingTablesView({super.key, required this.model});

  final PokerModel model;

  @override
  State<BrowsingTablesView> createState() => _BrowsingTablesViewState();
}

class _BrowsingTablesViewState extends State<BrowsingTablesView> {
  final TextEditingController _searchController = TextEditingController();

  _SortBy _sort = _SortBy.players;
  _BuyInFilter _buyIn = _BuyInFilter.any;
  _PlayerCountFilter _playerCount = _PlayerCountFilter.any;
  bool _hideFull = false;
  bool _showWaitingOnly = false;
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_handleSearchChanged);
  }

  @override
  void dispose() {
    _searchController
      ..removeListener(_handleSearchChanged)
      ..dispose();
    super.dispose();
  }

  void _handleSearchChanged() {
    final nextQuery = _searchController.text.trimLeft();
    if (nextQuery == _searchQuery) {
      return;
    }
    setState(() => _searchQuery = nextQuery);
  }

  double _toDcr(int atoms) => atoms / 1e8;

  String _tableName(UiTable table) {
    final name = table.name.trim();
    if (name.isNotEmpty) {
      return name;
    }

    final seatLabel = table.maxPlayers > 0 ? '${table.maxPlayers}-Seat ' : '';
    if (table.buyInAtoms == 0) {
      return '${seatLabel}Free Table'.trim();
    }

    return '${_toDcr(table.buyInAtoms).toStringAsFixed(2)} DCR Table';
  }

  String _shortId(String id, [int length = 8]) {
    if (id.isEmpty) {
      return '';
    }
    return id.length <= length ? id : id.substring(0, length);
  }

  String _availabilityLabel(UiTable table) {
    if (table.gameStarted) {
      return 'Playing';
    }
    if (table.currentPlayers >= table.maxPlayers && table.maxPlayers > 0) {
      return 'Full';
    }
    return 'Open';
  }

  int get _activeFilterCount =>
      (_searchQuery.trim().isNotEmpty ? 1 : 0) +
      (_hideFull ? 1 : 0) +
      (_showWaitingOnly ? 1 : 0) +
      (_buyIn == _BuyInFilter.any ? 0 : 1) +
      (_playerCount == _PlayerCountFilter.any ? 0 : 1);

  bool _matchesSearch(UiTable table) {
    final query = _searchQuery.trim().toLowerCase();
    if (query.isEmpty) {
      return true;
    }

    final haystacks = <String>[
      _tableName(table),
      table.name,
      table.id,
      _availabilityLabel(table),
    ];

    return haystacks.any((value) => value.toLowerCase().contains(query));
  }

  List<UiTable> get _filteredSortedTables {
    var list = List<UiTable>.of(widget.model.tables);

    list = list.where(_matchesSearch).toList();

    if (_hideFull) {
      list = list.where((t) => t.currentPlayers < t.maxPlayers).toList();
    }
    if (_showWaitingOnly) {
      list = list.where((t) => !t.gameStarted).toList();
    }
    if (_buyIn != _BuyInFilter.any) {
      list = list.where(_buyIn.matches).toList();
    }
    if (_playerCount != _PlayerCountFilter.any) {
      list = list.where(_playerCount.matches).toList();
    }

    switch (_sort) {
      case _SortBy.players:
        list.sort((a, b) => b.currentPlayers.compareTo(a.currentPlayers));
        break;
      case _SortBy.blinds:
        list.sort((a, b) => b.bigBlind.compareTo(a.bigBlind));
        break;
      case _SortBy.buyIn:
        list.sort((a, b) => a.buyInAtoms.compareTo(b.buyInAtoms));
        break;
    }

    return list;
  }

  void _clearFilters() {
    _searchController.clear();
    setState(() {
      _hideFull = false;
      _showWaitingOnly = false;
      _buyIn = _BuyInFilter.any;
      _playerCount = _PlayerCountFilter.any;
      _searchQuery = '';
    });
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;

    if (model.tables.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.style, size: 64, color: PokerColors.textMuted),
            const SizedBox(height: PokerSpacing.lg),
            const Text('No Tables', style: PokerTypography.headlineMedium),
            const SizedBox(height: PokerSpacing.sm),
            const Text(
              'Create one to start playing',
              style: PokerTypography.bodySmall,
            ),
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
    final wideLayout = screenWidth >= 980;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: PokerSpacing.md),
          child: wideLayout
              ? Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      width: 288,
                      child: _BrowseFiltersCard(
                        searchController: _searchController,
                        sort: _sort,
                        buyIn: _buyIn,
                        playerCount: _playerCount,
                        hideFull: _hideFull,
                        waitingOnly: _showWaitingOnly,
                        activeFilterCount: _activeFilterCount,
                        onSortChanged: (value) => setState(() => _sort = value),
                        onBuyInChanged: (value) =>
                            setState(() => _buyIn = value),
                        onPlayerCountChanged: (value) =>
                            setState(() => _playerCount = value),
                        onHideFullChanged: (value) =>
                            setState(() => _hideFull = value),
                        onWaitingOnlyChanged: (value) =>
                            setState(() => _showWaitingOnly = value),
                        onClearFilters: _clearFilters,
                      ),
                    ),
                    const SizedBox(width: PokerSpacing.lg),
                    Expanded(
                      child: _BrowseResultsPane(
                        tables: tables,
                        totalCount: model.tables.length,
                        activeFilterCount: _activeFilterCount,
                        sort: _sort,
                        searchQuery: _searchQuery,
                        onClearFilters: _clearFilters,
                        onRefresh: model.refreshTables,
                        onCreate: () => CreateTableDialog.open(context, model),
                        model: model,
                        tableName: _tableName,
                        shortId: _shortId,
                        toDcr: _toDcr,
                      ),
                    ),
                  ],
                )
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _BrowseFiltersCard(
                      searchController: _searchController,
                      sort: _sort,
                      buyIn: _buyIn,
                      playerCount: _playerCount,
                      hideFull: _hideFull,
                      waitingOnly: _showWaitingOnly,
                      activeFilterCount: _activeFilterCount,
                      onSortChanged: (value) => setState(() => _sort = value),
                      onBuyInChanged: (value) => setState(() => _buyIn = value),
                      onPlayerCountChanged: (value) =>
                          setState(() => _playerCount = value),
                      onHideFullChanged: (value) =>
                          setState(() => _hideFull = value),
                      onWaitingOnlyChanged: (value) =>
                          setState(() => _showWaitingOnly = value),
                      onClearFilters: _clearFilters,
                    ),
                    const SizedBox(height: PokerSpacing.lg),
                    _BrowseResultsPane(
                      tables: tables,
                      totalCount: model.tables.length,
                      activeFilterCount: _activeFilterCount,
                      sort: _sort,
                      searchQuery: _searchQuery,
                      onClearFilters: _clearFilters,
                      onRefresh: model.refreshTables,
                      onCreate: () => CreateTableDialog.open(context, model),
                      model: model,
                      tableName: _tableName,
                      shortId: _shortId,
                      toDcr: _toDcr,
                    ),
                  ],
                ),
        ),
      ],
    );
  }
}

class _SummaryPill extends StatelessWidget {
  const _SummaryPill({
    required this.icon,
    required this.label,
    this.accent,
  });

  final IconData icon;
  final String label;
  final Color? accent;

  @override
  Widget build(BuildContext context) {
    final chipColor = accent ?? PokerColors.surfaceDim;
    final foreground = accent ?? PokerColors.textSecondary;

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.md,
        vertical: PokerSpacing.sm,
      ),
      decoration: BoxDecoration(
        color: chipColor.withValues(alpha: accent == null ? 1 : 0.14),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(
          color: accent == null
              ? PokerColors.borderSubtle
              : chipColor.withValues(alpha: 0.4),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 15, color: foreground),
          const SizedBox(width: PokerSpacing.xs),
          Text(
            label,
            style: PokerTypography.labelSmall.copyWith(color: foreground),
          ),
        ],
      ),
    );
  }
}

class _BrowseFiltersCard extends StatelessWidget {
  const _BrowseFiltersCard({
    required this.searchController,
    required this.sort,
    required this.buyIn,
    required this.playerCount,
    required this.hideFull,
    required this.waitingOnly,
    required this.activeFilterCount,
    required this.onSortChanged,
    required this.onBuyInChanged,
    required this.onPlayerCountChanged,
    required this.onHideFullChanged,
    required this.onWaitingOnlyChanged,
    required this.onClearFilters,
  });

  final TextEditingController searchController;
  final _SortBy sort;
  final _BuyInFilter buyIn;
  final _PlayerCountFilter playerCount;
  final bool hideFull;
  final bool waitingOnly;
  final int activeFilterCount;
  final ValueChanged<_SortBy> onSortChanged;
  final ValueChanged<_BuyInFilter> onBuyInChanged;
  final ValueChanged<_PlayerCountFilter> onPlayerCountChanged;
  final ValueChanged<bool> onHideFullChanged;
  final ValueChanged<bool> onWaitingOnlyChanged;
  final VoidCallback onClearFilters;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surface,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Text('Filters', style: PokerTypography.titleLarge),
              ),
              if (activeFilterCount > 0)
                TextButton(
                  onPressed: onClearFilters,
                  child: const Text('Reset'),
                ),
            ],
          ),
          const SizedBox(height: PokerSpacing.xs),
          const Text(
            'Use the same filter pattern for every browse refinement: search, select a stake band, then narrow by availability.',
            style: PokerTypography.bodySmall,
          ),
          const SizedBox(height: PokerSpacing.lg),
          _FilterSection(
            title: 'Search',
            child: TextField(
              controller: searchController,
              style: PokerTypography.bodyMedium,
              decoration: InputDecoration(
                hintText: 'Search table names',
                prefixIcon: const Icon(
                  Icons.search,
                  size: 18,
                  color: PokerColors.textSecondary,
                ),
                suffixIcon: searchController.text.isEmpty
                    ? null
                    : IconButton(
                        onPressed: searchController.clear,
                        icon: const Icon(Icons.close, size: 16),
                      ),
              ),
            ),
          ),
          const SizedBox(height: PokerSpacing.lg),
          _FilterSection(
            title: 'Buy-in',
            child: Wrap(
              spacing: PokerSpacing.sm,
              runSpacing: PokerSpacing.sm,
              children: _BuyInFilter.values
                  .map(
                    (option) => _ChoiceFilterPill(
                      label: option.label,
                      selected: buyIn == option,
                      onSelected: (_) => onBuyInChanged(option),
                    ),
                  )
                  .toList(),
            ),
          ),
          const SizedBox(height: PokerSpacing.lg),
          _FilterSection(
            title: 'Players',
            child: Wrap(
              spacing: PokerSpacing.sm,
              runSpacing: PokerSpacing.sm,
              children: _PlayerCountFilter.values
                  .map(
                    (option) => _ChoiceFilterPill(
                      label: option.label,
                      selected: playerCount == option,
                      onSelected: (_) => onPlayerCountChanged(option),
                    ),
                  )
                  .toList(),
            ),
          ),
          const SizedBox(height: PokerSpacing.lg),
          _FilterSection(
            title: 'Availability',
            child: Wrap(
              spacing: PokerSpacing.sm,
              runSpacing: PokerSpacing.sm,
              children: [
                _ToggleFilterPill(
                  label: 'Hide full tables',
                  selected: hideFull,
                  onSelected: onHideFullChanged,
                ),
                _ToggleFilterPill(
                  label: 'Waiting only',
                  selected: waitingOnly,
                  onSelected: onWaitingOnlyChanged,
                ),
              ],
            ),
          ),
          const SizedBox(height: PokerSpacing.lg),
          _FilterSection(
            title: 'Sort by',
            child: DropdownButtonFormField<_SortBy>(
              initialValue: sort,
              items: _SortBy.values
                  .map(
                    (option) => DropdownMenuItem<_SortBy>(
                      value: option,
                      child: Text(option.label),
                    ),
                  )
                  .toList(),
              onChanged: (value) {
                if (value != null) {
                  onSortChanged(value);
                }
              },
              decoration: const InputDecoration(
                isDense: true,
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 12,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _FilterSection extends StatelessWidget {
  const _FilterSection({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title.toUpperCase(),
          style: PokerTypography.labelSmall.copyWith(
            color: PokerColors.textMuted,
            letterSpacing: 0.8,
          ),
        ),
        const SizedBox(height: PokerSpacing.sm),
        child,
      ],
    );
  }
}

class _ChoiceFilterPill extends StatelessWidget {
  const _ChoiceFilterPill({
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool> onSelected;

  @override
  Widget build(BuildContext context) {
    return ChoiceChip(
      label: Text(
        label,
        style: PokerTypography.labelSmall.copyWith(
          color: selected ? PokerColors.textPrimary : PokerColors.textSecondary,
        ),
      ),
      selected: selected,
      onSelected: onSelected,
      backgroundColor: PokerColors.surfaceDim,
      selectedColor: PokerColors.primary.withValues(alpha: 0.14),
      side: BorderSide(
        color: selected ? PokerColors.primary : PokerColors.borderSubtle,
      ),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.xs,
        vertical: PokerSpacing.xs,
      ),
    );
  }
}

class _ToggleFilterPill extends StatelessWidget {
  const _ToggleFilterPill({
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool> onSelected;

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      label: Text(
        label,
        style: PokerTypography.labelSmall.copyWith(
          color: selected ? PokerColors.textPrimary : PokerColors.textSecondary,
        ),
      ),
      selected: selected,
      onSelected: onSelected,
      backgroundColor: PokerColors.surfaceDim,
      selectedColor: PokerColors.primary.withValues(alpha: 0.14),
      checkmarkColor: PokerColors.primary,
      side: BorderSide(
        color: selected
            ? PokerColors.primary.withValues(alpha: 0.45)
            : PokerColors.borderSubtle,
      ),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.xs,
        vertical: PokerSpacing.xs,
      ),
    );
  }
}

class _BrowseResultsPane extends StatelessWidget {
  const _BrowseResultsPane({
    required this.tables,
    required this.totalCount,
    required this.activeFilterCount,
    required this.sort,
    required this.searchQuery,
    required this.onClearFilters,
    required this.onRefresh,
    required this.onCreate,
    required this.model,
    required this.tableName,
    required this.shortId,
    required this.toDcr,
  });

  final List<UiTable> tables;
  final int totalCount;
  final int activeFilterCount;
  final _SortBy sort;
  final String searchQuery;
  final VoidCallback onClearFilters;
  final Future<void> Function() onRefresh;
  final VoidCallback onCreate;
  final PokerModel model;
  final String Function(UiTable) tableName;
  final String Function(String, [int]) shortId;
  final double Function(int) toDcr;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surface,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          LayoutBuilder(
            builder: (context, constraints) {
              final compact = constraints.maxWidth < 720;
              final actions = Wrap(
                spacing: PokerSpacing.sm,
                runSpacing: PokerSpacing.sm,
                children: [
                  OutlinedButton.icon(
                    onPressed: () => onRefresh(),
                    icon: const Icon(Icons.refresh, size: 18),
                    label: const Text('Refresh'),
                  ),
                  ElevatedButton.icon(
                    onPressed: onCreate,
                    icon: const Icon(Icons.add, size: 18),
                    label: const Text('Create Table'),
                  ),
                ],
              );

              final heading = Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('Tables', style: PokerTypography.titleLarge),
                  const SizedBox(height: PokerSpacing.xs),
                  Text(
                    'Showing ${tables.length} of $totalCount tables, sorted by ${sort.label.toLowerCase()}.',
                    style: PokerTypography.bodySmall,
                  ),
                  const SizedBox(height: PokerSpacing.sm),
                  Wrap(
                    spacing: PokerSpacing.sm,
                    runSpacing: PokerSpacing.sm,
                    children: [
                      _SummaryPill(
                        icon: Icons.style_outlined,
                        label: '${tables.length} of $totalCount visible',
                      ),
                      if (activeFilterCount > 0)
                        _SummaryPill(
                          icon: Icons.filter_alt_outlined,
                          label: '$activeFilterCount active filters',
                          accent: PokerColors.primary,
                        ),
                    ],
                  ),
                ],
              );

              if (compact) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    heading,
                    const SizedBox(height: PokerSpacing.md),
                    actions,
                  ],
                );
              }

              return Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(child: heading),
                  const SizedBox(width: PokerSpacing.lg),
                  actions,
                ],
              );
            },
          ),
          const SizedBox(height: PokerSpacing.lg),
          if (tables.isEmpty)
            _EmptyFilterResults(
              searchQuery: searchQuery,
              onClearFilters: onClearFilters,
            )
          else
            LayoutBuilder(
              builder: (context, constraints) {
                final gridCount = constraints.maxWidth >= 1220
                    ? 3
                    : (constraints.maxWidth >= 740 ? 2 : 1);

                if (gridCount == 1) {
                  return ListView.separated(
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    itemCount: tables.length,
                    separatorBuilder: (_, __) =>
                        const SizedBox(height: PokerSpacing.md),
                    itemBuilder: (context, index) => _TableCard(
                      table: tables[index],
                      model: model,
                      tableName: tableName,
                      shortId: shortId,
                      toDcr: toDcr,
                    ),
                  );
                }

                return GridView.builder(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: gridCount,
                    crossAxisSpacing: PokerSpacing.md,
                    mainAxisSpacing: PokerSpacing.md,
                    mainAxisExtent: 344,
                  ),
                  itemCount: tables.length,
                  itemBuilder: (context, index) => _TableCard(
                    table: tables[index],
                    model: model,
                    tableName: tableName,
                    shortId: shortId,
                    toDcr: toDcr,
                  ),
                );
              },
            ),
        ],
      ),
    );
  }
}

class _EmptyFilterResults extends StatelessWidget {
  const _EmptyFilterResults({
    required this.searchQuery,
    required this.onClearFilters,
  });

  final String searchQuery;
  final VoidCallback onClearFilters;

  @override
  Widget build(BuildContext context) {
    final hasSearch = searchQuery.trim().isNotEmpty;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(PokerSpacing.xl),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        children: [
          const Icon(
            Icons.filter_alt_off,
            size: 36,
            color: PokerColors.textSecondary,
          ),
          const SizedBox(height: PokerSpacing.md),
          const Text(
            'No tables match these filters',
            style: PokerTypography.titleMedium,
          ),
          const SizedBox(height: PokerSpacing.xs),
          Text(
            hasSearch
                ? 'Nothing matched "$searchQuery". Try a broader search or reset the active filters.'
                : 'Try widening the buy-in range or resetting the active filters.',
            textAlign: TextAlign.center,
            style: PokerTypography.bodySmall,
          ),
          const SizedBox(height: PokerSpacing.lg),
          TextButton(
            onPressed: onClearFilters,
            child: const Text('Reset filters'),
          ),
        ],
      ),
    );
  }
}

class _TableCard extends StatelessWidget {
  const _TableCard({
    required this.table,
    required this.model,
    required this.tableName,
    required this.shortId,
    required this.toDcr,
  });

  final UiTable table;
  final PokerModel model;
  final String Function(UiTable) tableName;
  final String Function(String, [int]) shortId;
  final double Function(int) toDcr;

  String _statusText(UiTable table) {
    if (table.gameStarted) {
      return 'Hand in progress';
    }
    if (table.currentPlayers >= table.maxPlayers && table.maxPlayers > 0) {
      return 'All seats filled';
    }
    return 'Accepting players';
  }

  List<String> _playerPreview(UiTable table) {
    final preview = <String>[];
    for (final player in table.players.take(3)) {
      final name = player.name.trim().isNotEmpty
          ? player.name.trim()
          : shortId(player.id, 6);
      preview.add(name);
    }
    return preview;
  }

  @override
  Widget build(BuildContext context) {
    final full =
        table.currentPlayers >= table.maxPlayers && table.maxPlayers > 0;
    final started = table.gameStarted;
    final alreadyAtTable = model.currentTableId == table.id ||
        table.players.any((player) => player.id == model.playerId);
    final hasOtherActiveTable =
        model.currentTableId != null && model.currentTableId != table.id;
    final canWatch = !hasOtherActiveTable && !alreadyAtTable;
    final canJoin =
        !hasOtherActiveTable && !alreadyAtTable && !started && !full;
    final showPrimary = alreadyAtTable || canJoin;
    final primaryLabel = alreadyAtTable ? 'Open' : 'Join';
    final previewPlayers = _playerPreview(table);

    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            PokerColors.surfaceBright.withValues(alpha: 0.95),
            PokerColors.surface,
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color:
              alreadyAtTable ? PokerColors.primary : PokerColors.borderSubtle,
          width: alreadyAtTable ? 1.2 : 1,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.12),
            blurRadius: 16,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final fillHeight = constraints.hasBoundedHeight;

          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: fillHeight ? MainAxisSize.max : MainAxisSize.min,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          tableName(table),
                          style: PokerTypography.titleLarge,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: PokerSpacing.xs),
                        Text(
                          'Table ${shortId(table.id)}',
                          style: PokerTypography.bodySmall.copyWith(
                            color: PokerColors.textMuted,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: PokerSpacing.md),
                  _StatusBadge(started: started, full: full),
                ],
              ),
              const SizedBox(height: PokerSpacing.sm),
              Text(
                _statusText(table),
                style: PokerTypography.bodySmall,
              ),
              const SizedBox(height: PokerSpacing.md),
              Wrap(
                spacing: PokerSpacing.sm,
                runSpacing: PokerSpacing.sm,
                children: [
                  _InfoPill(
                    icon: Icons.account_balance_wallet_outlined,
                    label: '${toDcr(table.buyInAtoms).toStringAsFixed(2)} DCR',
                    highlight: true,
                  ),
                  _InfoPill(
                    icon: Icons.toll,
                    label: 'Blinds ${table.smallBlind}/${table.bigBlind}',
                  ),
                  _InfoPill(
                    icon: Icons.people_outline,
                    label: '${table.currentPlayers}/${table.maxPlayers} seats',
                  ),
                ],
              ),
              const SizedBox(height: PokerSpacing.md),
              ClipRRect(
                borderRadius: BorderRadius.circular(999),
                child: LinearProgressIndicator(
                  minHeight: 6,
                  value: table.maxPlayers == 0
                      ? 0
                      : (table.currentPlayers / table.maxPlayers)
                          .clamp(0.0, 1.0),
                  backgroundColor: PokerColors.borderSubtle,
                  valueColor: AlwaysStoppedAnimation<Color>(
                    full ? PokerColors.warning : PokerColors.primary,
                  ),
                ),
              ),
              const SizedBox(height: PokerSpacing.md),
              if (previewPlayers.isEmpty)
                Text(
                  'No players seated yet',
                  style: PokerTypography.bodySmall.copyWith(
                    color: PokerColors.textMuted,
                  ),
                )
              else
                Wrap(
                  spacing: PokerSpacing.sm,
                  runSpacing: PokerSpacing.sm,
                  children: previewPlayers
                      .map(
                        (playerName) => _PlayerBadge(label: playerName),
                      )
                      .toList(),
                ),
              if (fillHeight)
                const Spacer()
              else
                const SizedBox(height: PokerSpacing.md),
              Align(
                alignment: Alignment.centerRight,
                child: Wrap(
                  spacing: PokerSpacing.sm,
                  runSpacing: PokerSpacing.sm,
                  alignment: WrapAlignment.end,
                  children: [
                    if (canWatch)
                      OutlinedButton(
                        onPressed: () => model.watchTable(table.id),
                        child: const Text('Watch'),
                      ),
                    if (showPrimary)
                      ElevatedButton(
                        onPressed: () {
                          if (alreadyAtTable) {
                            model.openTableView();
                          } else {
                            model.joinTable(table.id);
                          }
                        },
                        style: ElevatedButton.styleFrom(
                          backgroundColor: PokerColors.success,
                          foregroundColor: Colors.black,
                        ),
                        child: Text(primaryLabel),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.started, required this.full});

  final bool started;
  final bool full;

  @override
  Widget build(BuildContext context) {
    final Color color;
    final String label;

    if (started) {
      color = PokerColors.success;
      label = 'Playing';
    } else if (full) {
      color = PokerColors.warning;
      label = 'Full';
    } else {
      color = PokerColors.primary;
      label = 'Open';
    }

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.sm,
        vertical: PokerSpacing.xs,
      ),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.4)),
      ),
      child: Text(
        label,
        style: PokerTypography.labelSmall.copyWith(color: color),
      ),
    );
  }
}

class _InfoPill extends StatelessWidget {
  const _InfoPill({
    required this.icon,
    required this.label,
    this.highlight = false,
  });

  final IconData icon;
  final String label;
  final bool highlight;

  @override
  Widget build(BuildContext context) {
    const backgroundColor = PokerColors.surfaceDim;
    const borderColor = PokerColors.borderSubtle;
    final iconColor = highlight ? PokerColors.primary : PokerColors.textMuted;
    final textStyle = highlight
        ? PokerTypography.labelLarge.copyWith(
            color: PokerColors.primary,
            fontWeight: FontWeight.w700,
          )
        : PokerTypography.bodySmall;

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.sm,
        vertical: PokerSpacing.sm,
      ),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: borderColor),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 14, color: iconColor),
          const SizedBox(width: PokerSpacing.xs),
          Text(label, style: textStyle),
        ],
      ),
    );
  }
}

class _PlayerBadge extends StatelessWidget {
  const _PlayerBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final initial = label.isEmpty ? '?' : label.substring(0, 1).toUpperCase();

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.sm,
        vertical: PokerSpacing.xs,
      ),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 20,
            height: 20,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: PokerColors.primary.withValues(alpha: 0.18),
              shape: BoxShape.circle,
            ),
            child: Text(
              initial,
              style: PokerTypography.labelSmall.copyWith(
                color: PokerColors.primary,
              ),
            ),
          ),
          const SizedBox(width: PokerSpacing.xs),
          Text(label, style: PokerTypography.bodySmall),
        ],
      ),
    );
  }
}
