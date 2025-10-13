import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:pokerui/config.dart';

class NewConfigModel extends ChangeNotifier {
  // ─── Editable fields ────────────────────────────────────────────────────
  String serverAddr        = '127.0.0.1:50051';
  String grpcCertPath      = '';
  String address           = '';
  String brRpcUrl          = 'wss://127.0.0.1:7777/ws';
  String brClientCert      = '';
  String brClientRpcCert   = '';
  String brClientRpcKey    = '';
  String rpcUser           = 'rpcuser';
  String rpcPass           = 'rpcpass';
  String debugLevel        = 'debug';
  bool   wantsLogNtfns     = false;

  final List<String> appArgs;
  String _appDataDir = '';
  // String _brDataDir = '';

  // ─── Construction ───────────────────────────────────────────────────────
  NewConfigModel(this.appArgs) {
    _initialiseDefaults();
  }

  factory NewConfigModel.fromConfig(Config c) => NewConfigModel([])
    ..serverAddr         = c.serverAddr
    ..grpcCertPath       = c.grpcCertPath
    ..address            = c.address
    ..brRpcUrl           = c.rpcWebsocketURL.isNotEmpty ? c.rpcWebsocketURL : 'wss://127.0.0.1:7777/ws'
    ..brClientCert       = c.rpcCertPath
    ..brClientRpcCert    = c.rpcClientCertPath
    ..brClientRpcKey     = c.rpcClientKeyPath
    ..rpcUser            = c.rpcUser.isNotEmpty ? c.rpcUser : 'rpcuser'
    ..rpcPass            = c.rpcPass.isNotEmpty ? c.rpcPass : 'rpcpass'
    ..debugLevel         = c.debugLevel
    ..wantsLogNtfns      = c.wantsLogNtfns;

  // ─── Helpers ────────────────────────────────────────────────────────────
  Future<void> _initialiseDefaults() async {
    _appDataDir = await _defaultAppDataDir();

    grpcCertPath = p.join(_appDataDir, 'server.cert');
    // Set default paths for BisonRelay certs (these would need to be configured)
    brClientCert = '';
    brClientRpcCert = '';
    brClientRpcKey = '';

    notifyListeners();
  }

  String appDatadir()  => _appDataDir;

  Future<String> getConfigFilePath() async =>
      p.join(_appDataDir, 'pokerui.conf');

  // ─── Save to disk ───────────────────────────────────────────────────────
  // Note: Config saving is now handled by the native plugin via CTCreateDefaultConfig command
  Future<void> saveConfig() async {
    // This method is kept for backward compatibility but now delegates to native plugin
    // The actual implementation would call the native plugin's CTCreateDefaultConfig command
    throw UnimplementedError('Config saving is now handled by native plugin');
  }

  // expose the resolved data directory to the UI for display
  String get dataDir => _appDataDir;

  // Helper method to get default app data directory
  Future<String> _defaultAppDataDir() async {
    if (Platform.isLinux) {
      final home = Platform.environment["HOME"];
      if (home != null && home != "") {
        return p.join(home, ".pokerui");
      }
    } else if (Platform.isWindows &&
        Platform.environment.containsKey("LOCALAPPDATA")) {
      return p.join(Platform.environment["LOCALAPPDATA"]!, "pokerui");
    } else if (Platform.isMacOS) {
      final home = Platform.environment["HOME"];
      if (home != null && home != "") {
        return p.join(home, "Library", "Application Support", "pokerui");
      }
    }
    // Fallback
    return p.join(Platform.environment["HOME"] ?? "/tmp", ".pokerui");
  }
}
