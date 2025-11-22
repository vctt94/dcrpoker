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
    '${cfg.dataDir}/logs/pokerui.log',
    cfg.payoutAddress,
    cfg.debugLevel,
    cfg.wantsLogNtfns,
    cfg.rpcWebsocketURL,
    cfg.rpcCertPath,
    cfg.rpcClientCertPath,
    cfg.rpcClientKeyPath,
    cfg.rpcUser,
    cfg.rpcPass,
  );

  developer.log(
    'Initializing poker client for ${cfg.serverAddr}',
    name: 'pokerui.init',
  );
  await Golib.initClient(initClientArgs);
}

