import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/views/table_session_view.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/ui_config.dart';

enum PokerPreviewDevice {
  phone,
  tablet,
  desktop,
}

extension PokerPreviewDeviceX on PokerPreviewDevice {
  String get label => switch (this) {
        PokerPreviewDevice.phone => 'Phone',
        PokerPreviewDevice.tablet => 'Tablet',
        PokerPreviewDevice.desktop => 'Desktop',
      };

  String get uiConfigKey => switch (this) {
        PokerPreviewDevice.phone => 'compact',
        PokerPreviewDevice.tablet => 'expanded',
        PokerPreviewDevice.desktop => 'wide',
      };

  Size get canvasSize => switch (this) {
        PokerPreviewDevice.phone => const Size(390, 844),
        PokerPreviewDevice.tablet => const Size(834, 1112),
        PokerPreviewDevice.desktop => const Size(1280, 760),
      };
}

enum PokerPreviewScene {
  table,
  game,
}

extension PokerPreviewSceneX on PokerPreviewScene {
  String get label => switch (this) {
        PokerPreviewScene.table => 'Table View',
        PokerPreviewScene.game => 'Game View',
      };
}

class PokerConfigPreview extends StatefulWidget {
  const PokerConfigPreview({
    super.key,
    required this.config,
    required this.uiConfig,
    required this.device,
    required this.scene,
  });

  final Config config;
  final PokerUiConfig uiConfig;
  final PokerPreviewDevice device;
  final PokerPreviewScene scene;

  @override
  State<PokerConfigPreview> createState() => _PokerConfigPreviewState();
}

class _PokerConfigPreviewState extends State<PokerConfigPreview> {
  static const _heroId = 'hero';
  late final FocusNode _focusNode;

  @override
  void initState() {
    super.initState();
    _focusNode = FocusNode(
      debugLabel: 'poker-config-preview',
      canRequestFocus: false,
      skipTraversal: true,
    );
  }

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  PokerModel _buildPreviewModel() {
    final players = _samplePlayers();
    final game = UiGameState(
      tableId: 'preview-table',
      phase: pr.GamePhase.FLOP,
      phaseName: 'Flop',
      players: players,
      communityCards: [
        pr.Card(value: 'A', suit: 'Spades'),
        pr.Card(value: '10', suit: 'Hearts'),
        pr.Card(value: '8', suit: 'Clubs'),
      ],
      pot: 360,
      currentBet: 60,
      currentPlayerId: _heroId,
      minRaise: 60,
      maxRaise: 400,
      smallBlind: 10,
      bigBlind: 20,
      gameStarted: true,
      playersRequired: 2,
      playersJoined: players.length,
      timeBankSeconds: 25,
      turnDeadlineUnixMs: 0,
    );

    final model = PokerModel(
      playerId: _heroId,
      dataDir: 'preview',
    )
      ..currentTableId = game.tableId
      ..game = game
      ..tables = [
        UiTable(
          id: game.tableId,
          name: 'Preview Table',
          players: players,
          smallBlind: game.smallBlind,
          bigBlind: game.bigBlind,
          maxPlayers: 6,
          minPlayers: 2,
          currentPlayers: players.length,
          buyInAtoms: 5000,
          phase: game.phase,
          gameStarted: game.gameStarted,
          allReady: true,
        ),
      ];

    return model;
  }

  List<UiPlayer> _samplePlayers() {
    return [
      UiPlayer(
        id: _heroId,
        name: 'You',
        balance: 1280,
        hand: [
          pr.Card(value: 'K', suit: 'Diamonds'),
          pr.Card(value: 'Q', suit: 'Diamonds'),
        ],
        currentBet: 60,
        folded: false,
        isTurn: true,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        cardsRevealed: true,
        tableSeat: 0,
      ),
      const UiPlayer(
        id: 'p1',
        name: 'Ari',
        balance: 940,
        hand: [],
        currentBet: 20,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: true,
        isSmallBlind: true,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        tableSeat: 1,
      ),
      const UiPlayer(
        id: 'p2',
        name: 'Mina',
        balance: 1120,
        hand: [],
        currentBet: 40,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: true,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        tableSeat: 2,
      ),
      const UiPlayer(
        id: 'p3',
        name: 'Luis',
        balance: 760,
        hand: [],
        currentBet: 60,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        tableSeat: 3,
      ),
      const UiPlayer(
        id: 'p4',
        name: 'Jo',
        balance: 540,
        hand: [],
        currentBet: 60,
        folded: true,
        isTurn: false,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        tableSeat: 4,
      ),
      const UiPlayer(
        id: 'p5',
        name: 'Nox',
        balance: 1480,
        hand: [],
        currentBet: 60,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: false,
        isReady: true,
        isDisconnected: true,
        handDesc: '',
        tableSeat: 5,
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    final previewSize = widget.device.canvasSize;
    final previewModel = _buildPreviewModel();
    final previewKey = ValueKey(
      jsonEncode({
        'config': {
          'table_theme': widget.config.tableTheme,
          'card_theme': widget.config.cardTheme,
          'card_size': widget.config.cardSize,
          'ui_size': widget.config.uiSize,
          'hide_table_logo': widget.config.hideTableLogo,
          'logo_position': widget.config.logoPosition,
        },
        'ui': widget.uiConfig.toJson(),
        'device': widget.device.name,
        'scene': widget.scene.name,
      }),
    );

    return Center(
      child: FittedBox(
        fit: BoxFit.contain,
        child: SizedBox(
          width: previewSize.width,
          height: previewSize.height,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(26),
            child: MultiProvider(
              key: previewKey,
              providers: [
                ChangeNotifierProvider(
                  create: (_) => ConfigNotifier()..updateConfig(widget.config),
                ),
                ChangeNotifierProvider(
                  create: (_) =>
                      PokerUiConfigNotifier(initial: widget.uiConfig),
                ),
              ],
              child: MediaQuery(
                data: MediaQueryData(
                  size: previewSize,
                  padding: EdgeInsets.zero,
                  viewPadding: EdgeInsets.zero,
                  viewInsets: EdgeInsets.zero,
                ),
                child: DecoratedBox(
                  decoration: const BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.topCenter,
                      end: Alignment.bottomCenter,
                      colors: [
                        Color(0xFF0B1524),
                        Color(0xFF081019),
                      ],
                    ),
                  ),
                  child: IgnorePointer(
                    ignoring: true,
                    child: widget.scene == PokerPreviewScene.game
                        ? TableSessionView(model: previewModel)
                        : Builder(
                            builder: (context) {
                              final pokerGame = PokerGame(
                                previewModel.playerId,
                                previewModel,
                                theme: PokerThemeConfig.fromContext(context),
                              );
                              return pokerGame.buildWidget(
                                previewModel.game!,
                                _focusNode,
                                showHeroSeatCards: true,
                              );
                            },
                          ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
