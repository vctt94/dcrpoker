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

  final String rpcCertPath;
  final String rpcClientCertPath;
  final String rpcClientKeyPath;
  final String rpcWebsocketURL;
  final String debugLevel;
  final String rpcUser;
  final String rpcPass;
  final bool wantsLogNtfns;
  final String dataDir;
  final String address;

  Config({
    required this.serverAddr,
    required this.grpcCertPath,
    required this.payoutAddress,
    required this.rpcCertPath,
    required this.rpcClientCertPath,
    required this.rpcClientKeyPath,
    required this.rpcWebsocketURL,
    required this.debugLevel,
    required this.rpcUser,
    required this.rpcPass,
    required this.wantsLogNtfns,
    required this.dataDir,
    required this.address,
  });

  factory Config.empty() => Config(
        serverAddr: '',
        grpcCertPath: '',
        payoutAddress: '',
        rpcCertPath: '',
        rpcClientCertPath: '',
        rpcClientKeyPath: '',
        rpcWebsocketURL: '',
        debugLevel: 'info',
        rpcUser: '',
        rpcPass: '',
        wantsLogNtfns: false,
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
    return Config(
      serverAddr: serverAddr.isNotEmpty ? serverAddr : '127.0.0.1:50050',
      grpcCertPath: pickPath('grpc_cert_path'),
      payoutAddress: pick('payout_address'),
      rpcCertPath: pickPath('rpc_cert_path'),
      rpcClientCertPath: pickPath('rpc_client_cert_path'),
      rpcClientKeyPath: pickPath('rpc_client_key_path'),
      rpcWebsocketURL: pick('rpc_websocket_url'),
      debugLevel: pick('debug_level').isNotEmpty ? pick('debug_level') : 'info',
      rpcUser: pick('rpc_user'),
      rpcPass: pick('rpc_pass'),
      wantsLogNtfns: (m['wants_log_ntfns'] ?? false) == true,
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
    bool? wantsLogNtfns,
    String? dataDir,
    String? address,
  }) {
    return Config(
      serverAddr: serverAddr ?? this.serverAddr,
      grpcCertPath: grpcCertPath ?? this.grpcCertPath,
      payoutAddress: payoutAddress ?? this.payoutAddress,
      rpcCertPath: rpcCertPath ?? this.rpcCertPath,
      rpcClientCertPath: rpcClientCertPath ?? this.rpcClientCertPath,
      rpcClientKeyPath: rpcClientKeyPath ?? this.rpcClientKeyPath,
      rpcWebsocketURL: rpcWebsocketURL ?? this.rpcWebsocketURL,
      debugLevel: debugLevel ?? this.debugLevel,
      rpcUser: rpcUser ?? this.rpcUser,
      rpcPass: rpcPass ?? this.rpcPass,
      wantsLogNtfns: wantsLogNtfns ?? this.wantsLogNtfns,
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
