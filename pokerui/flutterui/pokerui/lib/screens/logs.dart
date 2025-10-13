import 'dart:io';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:path/path.dart' as path;
import 'package:pokerui/config.dart';
import 'package:pokerui/components/shared_layout.dart';

class LogsScreen extends StatefulWidget {
  const LogsScreen({super.key});

  @override
  State<LogsScreen> createState() => _LogsScreenState();
}

class _LogsScreenState extends State<LogsScreen> {
  final TextEditingController _searchController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  
  List<String> _logLines = [];
  List<String> _filteredLogLines = [];
  String _logLevel = 'ALL';
  bool _isLoadingFile = false;
  String? _logFilePath;
  Timer? _refreshTimer;

  final List<String> _logLevels = ['ALL', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
  
  @override
  void initState() {
    super.initState();
    _loadLogFile();
    _startAutoRefresh();
    _searchController.addListener(_filterLogs);
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _loadLogFile() async {
    setState(() {
      _isLoadingFile = true;
    });

    try {
      final appDataDir = await _defaultAppDataDir();
      _logFilePath = path.join(appDataDir, "logs", "pongui.log");
      
      final logFile = File(_logFilePath!);
      if (await logFile.exists()) {
        final contents = await logFile.readAsString();
        _logLines = contents.split('\n').where((line) => line.trim().isNotEmpty).toList();
      } else {
        _logLines = ['Log file not found: $_logFilePath'];
      }
    } catch (e) {
      _logLines = ['Error reading log file: $e'];
    } finally {
      setState(() {
        _isLoadingFile = false;
      });
      _filterLogs();
    }
  }

  void _startAutoRefresh() {
    _refreshTimer = Timer.periodic(const Duration(seconds: 2), (timer) {
      if (mounted) {
        _loadLogFile();
      }
    });
  }

  void _filterLogs() {
    final searchTerm = _searchController.text.toLowerCase();
    final logLevel = _logLevel.toLowerCase();
    
    _filteredLogLines = _logLines.where((line) {
      final matchesSearch = searchTerm.isEmpty || line.toLowerCase().contains(searchTerm);
      final matchesLevel = logLevel == 'all' || line.toLowerCase().contains(logLevel);
      return matchesSearch && matchesLevel;
    }).toList();

    setState(() {});
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

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: "Application Logs",
      child: Column(
        children: [
          
          // Log Content
          Expanded(
            child: _isLoadingFile
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
                    child: _filteredLogLines.isEmpty
                        ? const Center(
                            child: Text(
                              'No logs found',
                              style: TextStyle(color: Colors.white54),
                            ),
                          )
                        : ListView.builder(
                            controller: _scrollController,
                            itemCount: _filteredLogLines.length,
                            itemBuilder: (context, index) {
                              final line = _filteredLogLines[index];
                              return Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8.0,
                                  vertical: 2.0,
                                ),
                                decoration: BoxDecoration(
                                  color: index % 2 == 0
                                      ? Colors.transparent
                                      : Colors.white.withOpacity(0.02),
                                ),
                                child: SelectableText(
                                  '${index + 1}: $line',
                                  style: TextStyle(
                                    color: _getLogLevelColor(line),
                                    fontFamily: 'monospace',
                                    fontSize: 12,
                                  ),
                                ),
                              );
                            },
                          ),
                  ),
          ),
        ],
      ),
    );
  }

  // Helper method to get default app data directory
  Future<String> _defaultAppDataDir() async {
    if (Platform.isLinux) {
      final home = Platform.environment["HOME"];
      if (home != null && home != "") {
        return path.join(home, ".pokerui");
      }
    } else if (Platform.isWindows &&
        Platform.environment.containsKey("LOCALAPPDATA")) {
      return path.join(Platform.environment["LOCALAPPDATA"]!, "pokerui");
    } else if (Platform.isMacOS) {
      final home = Platform.environment["HOME"];
      if (home != null && home != "") {
        return path.join(home, "Library", "Application Support", "pokerui");
      }
    }
    // Fallback
    return path.join(Platform.environment["HOME"] ?? "/tmp", ".pokerui");
  }
}