import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:pokerui/config.dart';

class NewConfigModel extends ChangeNotifier {
  // ─── Editable fields ────────────────────────────────────────────────────
  String serverAddr = '';
  String grpcCertPath = '';
  String address = '';
  String debugLevel = 'debug';
  bool soundsEnabled = true;

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
    ..debugLevel         = c.debugLevel
    ..soundsEnabled      = c.soundsEnabled;

  // ─── Helpers ────────────────────────────────────────────────────────────
  Future<void> _initialiseDefaults() async {
    _appDataDir = await defaultAppDataDir();

    grpcCertPath = p.join(_appDataDir, 'server.cert');

    notifyListeners();
  }

  String appDatadir() => _appDataDir;

  Future<String> getConfigFilePath() async =>
      p.join(_appDataDir, '$APPNAME.conf');

  // expose the resolved data directory to the UI for display
  String get dataDir => _appDataDir;
}
