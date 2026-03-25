import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';

class HandInProgressView extends StatefulWidget {
  final PokerModel model;
  const HandInProgressView({super.key, required this.model});

  @override
  State<HandInProgressView> createState() => _HandInProgressViewState();

  static int calculateTotalBet(
    int amt,
    int currentBet,
    int myBet,
    int bb, {
    int myBalance = 0,
  }) {
    return normalizeBetInputToTotal(
      entered: amt,
      myBet: myBet,
      myBalance: myBalance,
    );
  }
}

class _HandInProgressViewState extends State<HandInProgressView> {
  final FocusNode _gameFocusNode = FocusNode();
  final TextEditingController _betCtrl = TextEditingController();
  bool _showBetInput = false;
  bool _showSidebar = false;

  @override
  void dispose() {
    _gameFocusNode.dispose();
    _betCtrl.dispose();
    super.dispose();
  }

  bool get _hasLastShowdown => widget.model.hasLastShowdown;

  @override
  Widget build(BuildContext context) {
    final model = widget.model;
    final theme = PokerThemeConfig.fromContext(context);
    final pokerGame = PokerGame(model.playerId, model, theme: theme);
    final gameState = model.game ??
        UiGameState(
          tableId: '',
          phase: pr.GamePhase.PRE_FLOP,
          phaseName: 'hand',
          players: [],
          communityCards: [],
          pot: 0,
          currentBet: 0,
          currentPlayerId: '',
          minRaise: 0,
          maxRaise: 0,
          bigBlind: 0,
          smallBlind: 0,
          gameStarted: true,
          playersRequired: 0,
          playersJoined: 0,
          timeBankSeconds: 0,
          turnDeadlineUnixMs: 0,
        );

    final isReady = model.iAmReady;
    final isWaiting = gameState.phase == pr.GamePhase.WAITING;
    return LayoutBuilder(
      builder: (context, constraints) {
        final scene = PokerSceneLayout.resolve(
          constraints.biggest,
          safePadding: MediaQuery.paddingOf(context),
        );
        final useMobileDock = scene.mode == PokerLayoutMode.compactPortrait;
        final showTableHeroCards = !useMobileDock;
        const toggleInset = 4.0;
        const sidebarGap = 24.0;
        final sidebarWidth = useMobileDock
            ? (constraints.maxWidth * 0.74).clamp(260.0, 320.0)
            : (constraints.maxWidth * 0.32).clamp(280.0, 396.0);
        return Stack(
          fit: StackFit.expand,
          children: [
            pokerGame.buildWidget(
              gameState,
              _gameFocusNode,
              scene: scene,
              showHeroSeatCards: showTableHeroCards,
              onReadyHotkey: isWaiting ? () => model.setReady() : null,
            ),
            if (isWaiting)
              pokerGame.buildReadyToPlayOverlay(
                context,
                isReady,
                false,
                '',
                () => model.setReady(),
                gameState,
              ),
            if (!isWaiting && _hasLastShowdown)
              AnimatedPositioned(
                duration: const Duration(milliseconds: 260),
                curve: Curves.easeOutCubic,
                left: _showSidebar ? 0 : -(sidebarWidth + sidebarGap),
                top: 0,
                bottom: 0,
                width: sidebarWidth + sidebarGap,
                child: IgnorePointer(
                  ignoring: !_showSidebar,
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: sidebarWidth,
                        child: ShowdownSidebar(
                          model: model,
                          visible: true,
                          onClose: () => setState(() => _showSidebar = false),
                        ),
                      ),
                      const IgnorePointer(
                        child: SizedBox(width: sidebarGap),
                      ),
                    ],
                  ),
                ),
              ),
            if (!isWaiting && _hasLastShowdown && !_showSidebar)
              Positioned(
                left: scene.contentRect.left + toggleInset,
                top: scene.contentRect.top + toggleInset,
                child: PokerLastHandButton(
                  active: _showSidebar,
                  onTap: () => setState(() => _showSidebar = !_showSidebar),
                ),
              ),
            if (!isWaiting)
              Positioned.fromRect(
                rect: scene.heroDockRect,
                child: Container(
                  key: const Key('poker-hero-dock'),
                  child: useMobileDock
                      ? MobileHeroActionPanel(
                          model: model,
                          showBetInput: _showBetInput,
                          betCtrl: _betCtrl,
                          onToggleBetInput: () =>
                              setState(() => _showBetInput = !_showBetInput),
                          onCloseBetInput: () =>
                              setState(() => _showBetInput = false),
                        )
                      : BottomActionDock(
                          model: model,
                          showBetInput: _showBetInput,
                          betCtrl: _betCtrl,
                          onToggleBetInput: () =>
                              setState(() => _showBetInput = !_showBetInput),
                          onCloseBetInput: () =>
                              setState(() => _showBetInput = false),
                        ),
                ),
              ),
          ],
        );
      },
    );
  }
}
