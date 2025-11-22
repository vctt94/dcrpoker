import 'dart:developer' as developer;

import 'package:golib_plugin/definitions.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:pokerui/config.dart';

// Initializes the Go poker client with the provided config.
Future<void> initializePokerClient(Config cfg) async {
  final initClientArgs = InitClient(
    cfg.serverAddr,
    cfg.grpcCertPath,
    cfg.dataDir,
    cfg.payoutAddress,
    '${cfg.dataDir}/logs/pokerui.log',
    cfg.debugLevel,
    cfg.wantsLogNtfns,
  );

  developer.log(
    'Initializing poker client for ${cfg.serverAddr}',
    name: 'pokerui.init',
  );
  await Golib.initClient(initClientArgs);
}

