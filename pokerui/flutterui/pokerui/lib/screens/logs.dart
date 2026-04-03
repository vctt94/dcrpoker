import 'package:flutter/material.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/config.dart';

class LogsScreen extends StatefulWidget {
  const LogsScreen({super.key});

  @override
  State<LogsScreen> createState() => _LogsScreenState();
}

class _LogsScreenState extends State<LogsScreen> {
  static const int _pageSizeLines = 50;
  static const int _maxBytes = 256 * 1024;
  static const double _loadOlderThresholdPx = 24.0;

  final TextEditingController _searchController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  List<String> _lines = [];
  bool _initialLoading = true;
  bool _loadingOlder = false;
  bool _hasMoreBefore = true;
  int _nextBeforeOffset = -1; // -1 means EOF (tail)

  @override
  void initState() {
    super.initState();
    _searchController.addListener(() => setState(() {}));
    _scrollController.addListener(() {
      if (!_scrollController.hasClients) return;
      final pos = _scrollController.position;
      if (pos.pixels <= pos.minScrollExtent + _loadOlderThresholdPx) {
        _loadOlder();
      }
    });
    _loadInitial();
  }

  @override
  void dispose() {
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _loadInitial() async {
    setState(() {
      _initialLoading = true;
    });
    try {
      final dataDir = context.read<ConfigNotifier>().value.dataDir;
      final res = await Golib.readLogPage(
        ReadLogPageArgs(
          dataDir: dataDir,
          beforeOffset: -1,
          maxLines: _pageSizeLines,
          maxBytes: _maxBytes,
        ),
      );
      if (!mounted) return;
      setState(() {
        _lines = res.lines;
        _nextBeforeOffset = res.nextBeforeOffset;
        _hasMoreBefore = res.hasMoreBefore;
        _initialLoading = false;
      });
      // Stick to bottom on first load.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || !_scrollController.hasClients) return;
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _lines = const ['Error reading log file'];
        _hasMoreBefore = false;
        _initialLoading = false;
      });
    }
  }

  Future<void> _loadOlder() async {
    if (_loadingOlder || !_hasMoreBefore) return;
    if (_nextBeforeOffset <= 0) {
      setState(() {
        _hasMoreBefore = false;
      });
      return;
    }

    if (!_scrollController.hasClients) return;
    final prevMax = _scrollController.position.maxScrollExtent;
    final prevPixels = _scrollController.position.pixels;

    setState(() {
      _loadingOlder = true;
    });
    try {
      final dataDir = context.read<ConfigNotifier>().value.dataDir;
      final res = await Golib.readLogPage(
        ReadLogPageArgs(
          dataDir: dataDir,
          beforeOffset: _nextBeforeOffset,
          maxLines: _pageSizeLines,
          maxBytes: _maxBytes,
        ),
      );
      if (!mounted) return;
      setState(() {
        // Prepend older lines.
        _lines = [...res.lines, ..._lines];
        _nextBeforeOffset = res.nextBeforeOffset;
        _hasMoreBefore = res.hasMoreBefore;
        _loadingOlder = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _loadingOlder = false;
        _hasMoreBefore = false;
      });
    }

    // Keep content anchored after prepend.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      final newMax = _scrollController.position.maxScrollExtent;
      final delta = newMax - prevMax;
      if (delta > 0) {
        _scrollController.jumpTo(prevPixels + delta);
      }
    });
  }

  List<String> get _visibleLines {
    final search = _searchController.text.trim().toLowerCase();
    if (search.isEmpty) return _lines;
    return _lines.where((l) => l.toLowerCase().contains(search)).toList();
  }

  Color _getLogLevelColor(String line) {
    final lowerLine = line.toLowerCase();
    if (lowerLine.contains('error') || lowerLine.contains('err')) {
      return Colors.red;
    } else if (lowerLine.contains('warn')) {
      return Colors.orange;
    } else if (lowerLine.contains('info')) {
      return Colors.blue;
    } else if (lowerLine.contains('debug')) {
      return Colors.green;
    } else if (lowerLine.contains('trace')) {
      return Colors.grey;
    }
    return Colors.white;
  }

  List<InlineSpan> _buildLogTextSpans(List<String> lines) {
    return List<InlineSpan>.generate(lines.length, (index) {
      final line = lines[index];
      final suffix = index == lines.length - 1 ? '' : '\n';
      return TextSpan(
        text: '${index + 1}: $line$suffix',
        style: TextStyle(color: _getLogLevelColor(line)),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final lines = _visibleLines;
    return SharedLayout(
      title: "Application Logs",
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                'Showing last $_pageSizeLines lines (scroll up to load more).',
                style: const TextStyle(color: Colors.white54, fontSize: 12),
              ),
            ),
          ),

          // Log Content
          Expanded(
            child: _initialLoading
                ? const Center(
                    child: CircularProgressIndicator(),
                  )
                : Container(
                    margin: const EdgeInsets.all(8.0),
                    decoration: BoxDecoration(
                      color: const Color(0xFF0F0F0F),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: Colors.grey.shade700),
                    ),
                    child: lines.isEmpty
                        ? const Center(
                            child: Text(
                              'No logs found',
                              style: TextStyle(color: Colors.white54),
                            ),
                          )
                        : SingleChildScrollView(
                            controller: _scrollController,
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8.0,
                              vertical: 6.0,
                            ),
                            child: SelectableText.rich(
                              key: const Key('logs-selectable-content'),
                              TextSpan(children: _buildLogTextSpans(lines)),
                              style: const TextStyle(
                                color: Colors.white,
                                fontFamily: 'monospace',
                                fontSize: 12,
                                height: 1.3,
                              ),
                            ),
                          ),
                  ),
          ),
        ],
      ),
    );
  }
}
