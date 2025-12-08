import 'dart:io';
import 'package:path/path.dart' as path;
import 'package:golib_plugin/golib_plugin.dart';
import 'package:path_provider/path_provider.dart';

// Logical app name (used for .conf and log filenames).
const APPNAME = "bisonpoker";
// macOS Application Support directory name (matches bundle identifier).
const APP_SUPPORT_DIR = "com.bisonpoker";
String mainConfigFilename = "";

class Config {
  final String serverAddr;
  final String grpcCertPath;
  final String payoutAddress;
  final String debugLevel;
  final bool soundsEnabled;
  final String dataDir;
  final String address;

  Config({
    required this.serverAddr,
    required this.grpcCertPath,
    required this.payoutAddress,
    required this.debugLevel,
    required this.soundsEnabled,
    required this.dataDir,
    required this.address,
  });

  factory Config.empty() => Config(
        serverAddr: '',
        grpcCertPath: '',
        payoutAddress: '',
        debugLevel: 'info',
        soundsEnabled: true,
        dataDir: '',
        address: '',
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
  }) {
    return Config(
      serverAddr: serverAddr ?? this.serverAddr,
      grpcCertPath: grpcCertPath ?? this.grpcCertPath,
      payoutAddress: payoutAddress ?? this.payoutAddress,
      debugLevel: debugLevel ?? this.debugLevel,
      soundsEnabled: soundsEnabled ?? this.soundsEnabled,
      dataDir: dataDir ?? this.dataDir,
      address: address ?? this.address,
    );
  }

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
    print('baseDir: ${baseDir.path}');
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
