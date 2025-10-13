import 'dart:io';
import 'package:path/path.dart' as path;
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';

const APPNAME = "pokerui";
const BRUIGNAME = "bruig";
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
      serverAddr: serverAddr.isNotEmpty ? serverAddr : '127.0.0.1:50051',
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

  Future<void> saveNewConfig(String filepath) async {
    final buffer = StringBuffer();
    buffer.writeln('[default]');
    if (serverAddr.isNotEmpty) buffer.writeln('serveraddr=$serverAddr');
    if (grpcCertPath.isNotEmpty) buffer.writeln('grpcservercert=$grpcCertPath');
    if (address.isNotEmpty) buffer.writeln('address=$address');
    if (rpcWebsocketURL.isNotEmpty) buffer.writeln('brrpcurl=$rpcWebsocketURL');
    if (rpcCertPath.isNotEmpty) buffer.writeln('brclientcert=$rpcCertPath');
    if (rpcClientCertPath.isNotEmpty) {
      buffer.writeln('brclientrpccert=$rpcClientCertPath');
    }
    if (rpcClientKeyPath.isNotEmpty) {
      buffer.writeln('brclientrpckey=$rpcClientKeyPath');
    }
    if (rpcUser.isNotEmpty) buffer.writeln('rpcuser=$rpcUser');
    if (rpcPass.isNotEmpty) buffer.writeln('rpcpass=$rpcPass');
    buffer.writeln();
    buffer.writeln('[clientrpc]');
    buffer.writeln('wantsLogNtfns=${wantsLogNtfns ? '1' : '0'}');
    buffer.writeln();
    buffer.writeln('[log]');
    if (debugLevel.isNotEmpty) buffer.writeln('debuglevel=$debugLevel');

    await File(filepath).parent.create(recursive: true);
    await File(filepath).writeAsString(buffer.toString());
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

Future<String> defaultAppDataDir() async {
  final env = Platform.environment;
  if (Platform.isWindows) {
    final base = env['APPDATA'] ?? env['LOCALAPPDATA'] ?? homeDir();
    return path.join(base, APPNAME);
  } else if (Platform.isMacOS) {
    return path.join(homeDir(), 'Library', 'Application Support', APPNAME);
  } else {
    // Linux and others: use hidden dir in HOME
    return path.join(homeDir(), '.${APPNAME}');
  }
}

Future<Config> configFromArgs(List<String> args) async {
  final cfgFilePath = path.join(await defaultAppDataDir(), '$APPNAME.conf');
  if (!File(cfgFilePath).existsSync()) {
    throw newConfigNeededException;
  }
  return Config.loadConfig(cfgFilePath);
}
