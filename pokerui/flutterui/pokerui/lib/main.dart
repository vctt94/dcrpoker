import 'dart:async';
import 'dart:developer' as developer;
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/components/notification_bar.dart';
import 'package:pokerui/models/newconfig.dart';
import 'package:pokerui/models/notifications.dart';
import 'package:provider/provider.dart';
import 'package:window_manager/window_manager.dart';

import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/screens/main.dart';
import 'package:pokerui/screens/newconfig.dart';
import 'package:pokerui/screens/logs.dart';
import 'package:pokerui/screens/startup_error.dart';
import 'package:pokerui/screens/login.dart';
import 'package:pokerui/screens/sign_address.dart';
import 'package:pokerui/screens/open_escrow.dart';
import 'package:pokerui/screens/refund.dart';
import 'package:pokerui/screens/escrow_history.dart';
import 'package:pokerui/client_init.dart';

Future<void> runNewConfigApp(List<String> args) async {
  final newConfig = NewConfigModel(args);

  runApp(
    MaterialApp(
      title: 'New RPC Configuration',
      home: NewConfigScreen(
        model: newConfig,
        onConfigSaved: () async {
          try {
            final cfg = await configFromArgs(args);
            runPokerBootstrap(cfg);
          } catch (e) {
            developer.log('onConfigSaved: Error reloading config', error: e);
            rethrow;
          }
        },
      ),
    ),
  );
}

void main(List<String> args) async {
  try {
    WidgetsFlutterBinding.ensureInitialized();
    if (Platform.isLinux || Platform.isWindows || Platform.isMacOS) {
      await windowManager.ensureInitialized();
    }

    developer.log('Platform: ${Golib.majorPlatform}/${Golib.minorPlatform}');
    Golib.platformVersion
        .then((value) => developer.log('Platform Version: $value'));
    final cfg = await configFromArgs(args);
    runPokerBootstrap(cfg);
  } catch (exception) {
    developer.log('Error during start up', error: exception);
    if (exception == usageException) {
      exit(0);
    } else if (exception == newConfigNeededException) {
      runNewConfigApp(args);
      return;
    }
    runApp(
      MaterialApp(
        title: 'Poker UI - Fatal Error',
        theme: ThemeData.dark(),
        home: Scaffold(
          body: Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: SelectableText(exception.toString()),
            ),
          ),
        ),
      ),
    );
  }
}

void runPokerBootstrap(Config cfg) {
  runApp(PokerBootstrapApp(initialConfig: cfg));
}

class PokerBootstrapApp extends StatefulWidget {
  const PokerBootstrapApp({super.key, required this.initialConfig});

  final Config initialConfig;

  @override
  State<PokerBootstrapApp> createState() => _PokerBootstrapAppState();
}

class _PokerBootstrapAppState extends State<PokerBootstrapApp> {
  Config? _config;
  NotificationModel? _notificationModel;
  PokerModel? _pokerModel;
  bool _loading = true;
  bool _needsLogin = true;
  String? _nickname;
  Object? _lastError;
  List<String> _missingFields = const [];
  StreamSubscription<LocalWaitingRoom>? _wrCreatedSub;
  bool _attemptedSessionRestore = false;

  ThemeData get _theme => ThemeData.dark().copyWith(
        scaffoldBackgroundColor: const Color.fromARGB(255, 25, 23, 44),
        primaryColor: Colors.blueAccent,
      );

  @override
  void initState() {
    super.initState();
    _config = widget.initialConfig;
    _bootstrap();
  }

  @override
  void dispose() {
    _disposeCurrentModel();
    super.dispose();
  }

  void _disposeCurrentModel() {
    _pokerModel?.dispose();
    _notificationModel?.dispose();
    _wrCreatedSub?.cancel();
    _pokerModel = null;
    _notificationModel = null;
  }

  Future<void> _bootstrap({String? nickname}) async {
    final cfg = _config;
    if (cfg == null) {
      return;
    }

    // If nickname is provided, proceed with initialization
    if (nickname != null) {
      _nickname = nickname;
      _needsLogin = false;
    }

    // Attempt to reuse an existing session before showing the login screen.
    if (_needsLogin) {
      final restored = await _tryRestoreSession(cfg);
      if (restored != null) {
        _nickname = restored;
        _needsLogin = false;
      }
    }

    // If we still need login, show login screen
    if (_needsLogin) {
      setState(() {
        _loading = false;
      });
      return;
    }

    _disposeCurrentModel();
    setState(() {
      _loading = true;
      _lastError = null;
      _missingFields = const [];
    });

    final notificationModel = NotificationModel();
    try {
      // Note: Client is already initialized in LoginScreen, so PokerModel.fromConfig
      // will reuse the existing client handle
      final pokerModel = await PokerModel.fromConfig(cfg, notificationModel);
      
      // Authentication already happened in LoginScreen, so we can proceed
      await pokerModel.init();
      if (!mounted) {
        pokerModel.dispose();
        notificationModel.dispose();
        return;
      }
      setState(() {
        _notificationModel = notificationModel;
        _pokerModel = pokerModel;
        _loading = false;
      });

      // Listen for waiting room creation notifications and surface in UI
      _wrCreatedSub?.cancel();
      _wrCreatedSub = Golib.waitingRoomCreated().listen((wr) {
        final bet = (wr.betAmt / 1e8).toStringAsFixed(4);
        _notificationModel?.showNotification(
            'Waiting room created by ${wr.host} • Bet: $bet DCR');
      });
    } catch (error, stackTrace) {
      developer.log(
        'Failed to initialise poker client',
        error: error,
        stackTrace: stackTrace,
      );
      notificationModel.dispose();
      if (!mounted) {
        return;
      }
      setState(() {
        _lastError = error;
        _missingFields = _extractMissingFields(error.toString());
        _loading = false;
        _notificationModel = null;
        _pokerModel = null;
      });
    }
  }

  Future<bool> _reloadConfig() async {
    try {
      final updated = await configFromArgs([]);
      if (!mounted) return false;
      setState(() {
        _config = updated;
        _attemptedSessionRestore = false;
      });
      await _bootstrap();
      return _pokerModel != null && !_loading;
    } catch (error, stackTrace) {
      developer.log(
        'Failed to reload config after edit',
        error: error,
        stackTrace: stackTrace,
      );
      if (!mounted) return false;
      setState(() {
        _lastError = error;
        _missingFields = _extractMissingFields(error.toString());
        _loading = false;
      });
      return false;
    }
  }

  Future<void> _openConfig(BuildContext context) async {
    final cfg = _config ?? Config.filled();
    final navigator = Navigator.of(context);
    await navigator.push(
      MaterialPageRoute(
        builder: (_) => NewConfigScreen(
          model: NewConfigModel.fromConfig(cfg),
          onConfigSaved: () async {
            final success = await _reloadConfig();
            if (!success) {
              throw Exception('Configuration still missing required values.');
            }
            if (navigator.canPop()) {
              navigator.pop();
            }
          },
        ),
      ),
    );
  }

  Future<String?> _tryRestoreSession(Config cfg) async {
    if (_attemptedSessionRestore) {
      return null;
    }
    _attemptedSessionRestore = true;

    try {
      await initializePokerClient(cfg);
      final resp = await Golib.resumeSession();
      if (resp == null) {
        return null;
      }
      if (resp.nickname.isEmpty) {
        return null;
      }
      return resp.nickname;
    } catch (error, stackTrace) {
      developer.log(
        'Auto session restore failed',
        error: error,
        stackTrace: stackTrace,
      );
      return null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final cfg = _config;

    // Show login screen if needed
    if (_needsLogin && !_loading && _lastError == null) {
      return MaterialApp(
        debugShowCheckedModeBanner: false,
        theme: _theme,
        home: LoginScreen(
          config: cfg!,
          onLoginSuccess: (nickname) {
            _bootstrap(nickname: nickname);
          },
        ),
      );
    }

    if (_pokerModel != null &&
        _notificationModel != null &&
        cfg != null &&
        !_loading &&
        _lastError == null) {
      return MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: _notificationModel!),
          ChangeNotifierProvider.value(value: _pokerModel!),
        ],
        child: MyApp(cfg),
      );
    }

    if (_lastError != null) {
      return MaterialApp(
        debugShowCheckedModeBanner: false,
        theme: _theme,
        home: StartupErrorScreen(
          message: _lastError.toString(),
          missingFields: _missingFields,
          dataDir: cfg?.dataDir ?? '',
          onRetry: () => _bootstrap(nickname: _nickname),
          onOpenConfig: _openConfig,
        ),
      );
    }

    if (_loading) {
      return MaterialApp(
        debugShowCheckedModeBanner: false,
        theme: _theme,
        home: const Scaffold(
          body: Center(child: CircularProgressIndicator.adaptive()),
        ),
      );
    }

    return MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: _theme,
      home: StartupErrorScreen(
        message: 'Poker UI failed to start',
        missingFields: _missingFields,
        dataDir: cfg?.dataDir ?? '',
        onRetry: () => _bootstrap(nickname: _nickname),
        onOpenConfig: _openConfig,
      ),
    );
  }

  List<String> _extractMissingFields(String message) {
    final match = RegExp(
      r'missing required fields? in client config: (.+)',
    ).firstMatch(message);
    if (match == null) {
      return const [];
    }
    return match
        .group(1)!
        .split(',')
        .map((value) => value.trim())
        .where((value) => value.isNotEmpty)
        .toList();
  }
}

class MyApp extends StatelessWidget {
  final Config cfg;
  const MyApp(this.cfg, {super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'Pong Game App',
      theme: ThemeData.dark().copyWith(
        scaffoldBackgroundColor: const Color.fromARGB(255, 25, 23, 44),
        primaryColor: Colors.blueAccent,
      ),
      builder: (context, child) {
        return Stack(
          children: [
            child!, // The main content of the app
            Align(
              alignment: Alignment.topCenter,
              child: NotificationBar(),
            ),
          ],
        );
      },
      routes: {
        '/': (context) => const PokerHomeScreen(),
        '/settings': (context) => NewConfigScreen(
              model: NewConfigModel.fromConfig(cfg),
              onConfigSaved: () async {
                try {
                  final updatedCfg = await configFromArgs([]);
                  runPokerBootstrap(updatedCfg);
                } catch (e) {
                  rethrow;
                }
              },
            ),
        '/logs': (context) => const LogsScreen(),
        '/sign-address': (context) => const SignAddressScreen(),
        '/open-escrow': (context) => const OpenEscrowScreen(),
        '/refund': (context) => const RefundScreen(),
        '/escrow-history': (context) => const EscrowHistoryScreen(),
      },
      initialRoute: '/',
    );
  }
}
