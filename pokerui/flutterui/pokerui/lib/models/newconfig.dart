import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:pokerui/config.dart';

class NewConfigModel extends ChangeNotifier {
  // ─── Editable fields ────────────────────────────────────────────────────
  String serverAddr = '';
  String grpcCertPath = '';
  String nickname = '';
  String address = '';
  String debugLevel = 'info';
  bool soundsEnabled = true;
  String tableTheme = 'classic';
  String cardTheme = 'standard';
  String cardSize = 'medium';
  String uiSize = 'medium';
  bool hideTableLogo = false;
  String logoPosition = 'center';

  final List<String> appArgs;
  String _appDataDir = '';
  late final Future<void> _initFuture;
  // String _brDataDir = '';

  // ─── Construction ───────────────────────────────────────────────────────
  NewConfigModel(
    this.appArgs, {
    String? initialDataDir,
    String? initialGrpcCertPath,
  }) {
    _initFuture = _initialiseDefaults(
      initialDataDir: initialDataDir,
      initialGrpcCertPath: initialGrpcCertPath,
    );
  }

  factory NewConfigModel.fromConfig(Config c) => NewConfigModel(
        [],
        initialDataDir: c.dataDir,
        initialGrpcCertPath: c.grpcCertPath,
      )
        ..serverAddr = c.serverAddr
        ..nickname = c.nickname
        ..address = c.address
        ..debugLevel = c.debugLevel
        ..soundsEnabled = c.soundsEnabled
        ..tableTheme = c.tableTheme
        ..cardTheme = c.cardTheme
        ..cardSize = c.cardSize
        ..uiSize = c.uiSize
        ..hideTableLogo = c.hideTableLogo
        ..logoPosition = c.logoPosition;

  // ─── Helpers ────────────────────────────────────────────────────────────
  Future<void> _initialiseDefaults({
    String? initialDataDir,
    String? initialGrpcCertPath,
  }) async {
    final resolvedDataDir = initialDataDir?.trim().isNotEmpty == true
        ? cleanAndExpandPath(initialDataDir!)
        : await defaultAppDataDir();

    _appDataDir = resolvedDataDir;
    grpcCertPath = (initialGrpcCertPath?.trim().isNotEmpty ?? false)
        ? initialGrpcCertPath!
        : p.join(resolvedDataDir, 'server.cert');

    notifyListeners();
  }

  Future<String> appDatadir() async {
    await _initFuture;
    return _appDataDir;
  }

  Future<String> getConfigFilePath() async {
    await _initFuture;
    return p.join(_appDataDir, '$APPNAME.conf');
  }

  // expose the resolved data directory to the UI for display
  String get dataDir => _appDataDir;
}
