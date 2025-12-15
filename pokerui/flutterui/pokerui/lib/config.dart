import 'dart:io';
import 'package:flutter/material.dart';
import 'package:path/path.dart' as path;
import 'package:provider/provider.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:path_provider/path_provider.dart';

// Logical app name (used for .conf and log filenames).
const APPNAME = "dcrpoker";
// macOS Application Support directory name (matches bundle identifier).
const APP_SUPPORT_DIR = "com.dcrpoker";
String mainConfigFilename = "";

class Config {
  final String serverAddr;
  final String grpcCertPath;
  final String payoutAddress;
  final String debugLevel;
  final bool soundsEnabled;
  final String dataDir;
  final String address;
  final String tableTheme;
  final String cardTheme;
  final String cardSize;
  final String uiSize;
  final bool hideTableLogo;
  final String logoPosition;

  Config({
    required this.serverAddr,
    required this.grpcCertPath,
    required this.payoutAddress,
    required this.debugLevel,
    required this.soundsEnabled,
    required this.dataDir,
    required this.address,
    required this.tableTheme,
    required this.cardTheme,
    required this.cardSize,
    required this.uiSize,
    required this.hideTableLogo,
    required this.logoPosition,
  });

  factory Config.empty() => Config(
        serverAddr: '',
        grpcCertPath: '',
        payoutAddress: '',
        debugLevel: 'info',
        soundsEnabled: true,
        dataDir: '',
        address: '',
      tableTheme: 'decred',
      cardTheme: 'standard',
      cardSize: 'medium',
      uiSize: 'medium',
      hideTableLogo: false,
      logoPosition: 'center',
      );

  // Synchronous fallback for UI prefill when async is not possible.
  factory Config.filled() => Config.empty();

  factory Config.fromMap(Map<String, dynamic> m) {
    String pick(String key) => (m[key] ?? '').toString();
    String pickPath(String key) {
      final v = pick(key);
      if (v.isEmpty) return v;
      return cleanAndExpandPath(v);
    }

    final serverAddr = pick('server_addr');
    if (serverAddr.isEmpty) {
      throw Exception('Server address is required');
    }
    return Config(
      serverAddr: serverAddr,
      grpcCertPath: pickPath('grpc_cert_path'),
      payoutAddress: pick('payout_address'),
      debugLevel: pick('debug_level').isNotEmpty ? pick('debug_level') : 'info',

      soundsEnabled: (m['sounds_enabled'] ?? true) == true,
      dataDir: pickPath('datadir'),
      address: pick('address'),
      tableTheme: pick('table_theme').isNotEmpty ? pick('table_theme') : (pick('tabletheme').isNotEmpty ? pick('tabletheme') : 'decred'),
      cardTheme: pick('card_theme').isNotEmpty ? pick('card_theme') : (pick('cardtheme').isNotEmpty ? pick('cardtheme') : 'standard'),
      cardSize: pick('card_size').isNotEmpty ? pick('card_size') : (pick('cardsize').isNotEmpty ? pick('cardsize') : 'medium'),
      uiSize: pick('ui_size').isNotEmpty ? pick('ui_size') : (pick('uisize').isNotEmpty ? pick('uisize') : 'medium'),
      hideTableLogo: (m['hide_table_logo'] ?? false) == true || (m['hidetablelogo'] ?? false) == true,
      logoPosition: pick('logo_position').isNotEmpty ? pick('logo_position') : (pick('logoposition').isNotEmpty ? pick('logoposition') : 'center'),
    );
  }

  Config copyWith({
    String? serverAddr,
    String? grpcCertPath,
    String? payoutAddress,
    String? rpcCertPath,
    String? rpcClientCertPath,
    String? rpcClientKeyPath,
    String? rpcWebsocketURL,
    String? debugLevel,
    String? rpcUser,
    String? rpcPass,
    bool? soundsEnabled,
    String? dataDir,
    String? address,
    String? tableTheme,
    String? cardTheme,
    String? cardSize,
    String? uiSize,
    bool? hideTableLogo,
  }) {
    return Config(
      serverAddr: serverAddr ?? this.serverAddr,
      grpcCertPath: grpcCertPath ?? this.grpcCertPath,
      payoutAddress: payoutAddress ?? this.payoutAddress,
      debugLevel: debugLevel ?? this.debugLevel,
      soundsEnabled: soundsEnabled ?? this.soundsEnabled,
      dataDir: dataDir ?? this.dataDir,
      address: address ?? this.address,
      tableTheme: tableTheme ?? this.tableTheme,
      cardTheme: cardTheme ?? this.cardTheme,
      cardSize: cardSize ?? this.cardSize,
      uiSize: uiSize ?? this.uiSize,
      hideTableLogo: hideTableLogo ?? this.hideTableLogo,
      logoPosition: logoPosition ?? this.logoPosition,
    );
  }

  bool get showTableLogo => !hideTableLogo;

  static Future<Config> loadConfig(String filepath) async {
    final m = await Golib.loadConfig(filepath);
    return Config.fromMap(Map<String, dynamic>.from(m));
  }
}

final usageException = Exception('Usage Displayed');
final newConfigNeededException = Exception('Config needed');

Future<Config> loadConfig(String filepath) async {
  return Config.loadConfig(filepath);
}

String homeDir() {
  final env = Platform.environment;
  if (Platform.isWindows) {
    return env['UserProfile'] ?? '';
  }
  return env['HOME'] ?? '';
}

String cleanAndExpandPath(String p) {
  if (p.isEmpty) return p;
  if (p.startsWith('~')) {
    p = homeDir() + p.substring(1);
  }
  return path.normalize(path.absolute(p));
}

// Function to get the default app data directory based on the platform
Future<String> defaultAppDataDir() async {
  if (Platform.isLinux) {
    final home = Platform.environment["HOME"];
    if (home != null && home != "") {
      return path.join(home, ".$APPNAME");
    }
  } else if (Platform.isWindows &&
      Platform.environment.containsKey("LOCALAPPDATA")) {
    return path.join(Platform.environment["LOCALAPPDATA"]!, APPNAME);
  } else if (Platform.isMacOS) {
    // Use the platform-provided Application Support directory to remain within
    // writable sandboxed locations. Avoid walking to parent to strip bundle id.
    final baseDir = (await getApplicationSupportDirectory());
    return path.join(baseDir.path, APPNAME);
  }

  // For other platforms, get the parent directory to avoid bundle identifier paths
  final dir = await getApplicationSupportDirectory();
  return path.join(dir.path, APPNAME);
}

Future<Config> configFromArgs(List<String> args) async {
  final cfgFilePath = path.join(await defaultAppDataDir(), '$APPNAME.conf');
  // Do not force the user through the interactive "new config" flow on first
  // start. Instead, let the Go backend auto-create a sane default config based
  // on the computed data directory when none exists yet.
  return Config.loadConfig(cfgFilePath);
}

/// ConfigNotifier manages the application configuration and notifies listeners
/// when the config changes. It loads config from golib and provides reactive
/// updates to widgets that depend on config values.
class ConfigNotifier extends ChangeNotifier {
  Config? _config;
  String? _configFilePath;
  bool _isLoading = false;

  Config? get config => _config;
  bool get isLoading => _isLoading;
  bool get hasConfig => _config != null;

  /// Get the current config, throwing if not loaded
  Config get value {
    if (_config == null) {
      throw StateError('Config not loaded. Call loadConfig() first.');
    }
    return _config!;
  }

  /// Load config from the default location (data directory)
  Future<void> loadConfig() async {
    if (_isLoading) return;
    
    _isLoading = true;
    notifyListeners();
    
    try {
      final dataDir = await defaultAppDataDir();
      _configFilePath = path.join(dataDir, '$APPNAME.conf');
      await _loadFromPath(_configFilePath!);
    } catch (error) {
      _isLoading = false;
      notifyListeners();
      rethrow;
    }
  }

  /// Load config from a specific file path
  Future<void> loadConfigFromPath(String filepath) async {
    if (_isLoading) return;
    
    _isLoading = true;
    notifyListeners();
    
    try {
      _configFilePath = filepath;
      await _loadFromPath(filepath);
    } catch (error) {
      _isLoading = false;
      notifyListeners();
      rethrow;
    }
  }

  /// Internal method to load config from golib
  Future<void> _loadFromPath(String filepath) async {
    try {
      // Load config from golib - this reads from the actual config file
      final m = await Golib.loadConfig(filepath);
      final newConfig = Config.fromMap(Map<String, dynamic>.from(m));
      
      _config = newConfig;
      _isLoading = false;
      notifyListeners();
    } catch (error) {
      _isLoading = false;
      notifyListeners();
      rethrow;
    }
  }

  /// Reload config from the last known path (or default if not set)
  Future<void> reload() async {
    if (_configFilePath != null) {
      await loadConfigFromPath(_configFilePath!);
    } else {
      await loadConfig();
    }
  }

  /// Update config with a new value and notify listeners
  void updateConfig(Config newConfig) {
    if (_config != newConfig) {
      _config = newConfig;
      notifyListeners();
    }
  }
}

/// Extension on BuildContext to simplify accessing config values
extension ConfigExtension on BuildContext {
  /// Get the current config from ConfigNotifier
  Config get config => watch<ConfigNotifier>().value;
  
  /// Get the card theme from config
  String get cardTheme => config.cardTheme;
  
  /// Get the card size from config
  String get cardSize => config.cardSize;
  
  /// Get the UI size from config
  String get uiSize => config.uiSize;
  
  /// Get the table theme from config
  String get tableTheme => config.tableTheme;
  
  /// Get whether to show the table logo from config
  bool get showTableLogo => config.showTableLogo;
  
  /// Get the logo position from config
  String get logoPosition => config.logoPosition;
  
  /// Get whether sounds are enabled from config
  bool get soundsEnabled => config.soundsEnabled;
}
