import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/showdown_board_label.dart';
import 'package:pokerui/components/poker/showdown_content.dart';
import 'package:pokerui/components/poker/showdown_fx_overlay.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/spacing.dart';

class TableSessionView extends StatefulWidget {
  const TableSessionView({super.key, required this.model});

  final PokerModel model;

  @override
  State<TableSessionView> createState() => _TableSessionViewState();

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

class _TableSessionViewState extends State<TableSessionView> {
  final FocusNode _gameFocusNode = FocusNode();
  final TextEditingController _betCtrl = TextEditingController();
  bool _showBetInput = false;
  bool _showSidebar = false;
  String? _betInputSeedKey;

  bool get _isShowdown => widget.model.state == PokerState.showdown;
  bool get _hasLastShowdown => widget.model.hasLastShowdown;

  @override
  void dispose() {
    _gameFocusNode.dispose();
    _betCtrl.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(TableSessionView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.model.state != widget.model.state) {
      _showSidebar = false;
    }
  }

  void _closeSidebar() {
    if (!mounted) return;
    setState(() => _showSidebar = false);
  }

  void _syncBetInputSeed(UiGameState gameState) {
    if (!_showBetInput) {
      _betInputSeedKey = null;
      return;
    }

    final me = widget.model.me;
    final seedKey = [
      gameState.phase.value,
      gameState.currentBet,
      gameState.minRaise,
      gameState.maxRaise,
      gameState.bigBlind,
      me?.currentBet ?? 0,
    ].join(':');

    if (_betInputSeedKey == seedKey) return;

    final target = initialBetOrRaiseTotal(
      currentBet: gameState.currentBet,
      minRaise: gameState.minRaise,
      maxRaise: gameState.maxRaise,
      bigBlind: gameState.bigBlind,
    );
    final text = target > 0 ? target.toString() : '';
    if (_betCtrl.text != text) {
      _betCtrl.value = TextEditingValue(
        text: text,
        selection: TextSelection.collapsed(offset: text.length),
      );
    }
    _betInputSeedKey = seedKey;
  }

  void _syncBetInputVisibility() {
    if (_showBetInput && !widget.model.canAct) {
      _showBetInput = false;
      _betInputSeedKey = null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;
    _syncBetInputVisibility();

    if (_isShowdown && model.game == null) {
      return const Center(child: Text('No game data available'));
    }

    final theme = PokerThemeConfig.fromContext(context);
    final pokerGame = PokerGame(model.playerId, model, theme: theme);

    final gameState = model.game ??
        const UiGameState(
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

    _syncBetInputSeed(gameState);

    final isReady = model.iAmReady;
    final isWaiting = gameState.phase == pr.GamePhase.WAITING;
    final heroPlayerIndex =
        gameState.players.indexWhere((player) => player.id == model.playerId);
    final heroCardsRevealed =
        heroPlayerIndex >= 0 && heroPlayerIndex < gameState.players.length
            ? gameState.players[heroPlayerIndex].cardsRevealed
            : false;

    final showdown = model.showdown;
    final lastShowdown = model.lastShowdown;

    final UiShowdownState? sidebarShowdown =
        _isShowdown ? showdown : lastShowdown;
    final bool showShowdownChrome = _isShowdown
        ? (showdown != null)
        : (!isWaiting && _hasLastShowdown && lastShowdown != null);

    final pendingGameEndMessage = model.pendingGameEndMessage;
    final Widget? showdownFooter = _isShowdown && model.isGameEndPending
        ? Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  pendingGameEndMessage.isNotEmpty
                      ? pendingGameEndMessage
                      : 'Game ended. Press Continue.',
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    color: Colors.white70,
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 10),
                ElevatedButton.icon(
                  onPressed: () {
                    _closeSidebar();
                    model.skipShowdown();
                  },
                  icon: const Icon(Icons.skip_next, size: 18),
                  label: const Text('Continue'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.blue.shade700,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 8,
                    ),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                  ),
                ),
              ],
            ),
          )
        : null;

    return LayoutBuilder(
      builder: (context, constraints) {
        final scene = PokerSceneLayout.resolve(
          constraints.biggest,
          safePadding: MediaQuery.paddingOf(context),
        );
        final safePadding = MediaQuery.paddingOf(context);
        final useMobileDock = scene.mode == PokerLayoutMode.compactPortrait;
        // Match hand-in-progress: on phone portrait, hero hole cards live in the dock
        // only — not also at the seat (showdown used to force seat cards and caused
        // duplicates with the dock).
        final showTableHeroCards = !useMobileDock;
        const toggleInset = 4.0;
        const menuButtonSize = 44.0;
        final topChromeInset = safePadding.top + PokerSpacing.md;
        final menuCornerClearance =
            menuButtonSize + (PokerSpacing.md * 2) + PokerSpacing.sm;
        final panelLeftInset = safePadding.left + PokerSpacing.md;
        final panelRightInset = safePadding.right + PokerSpacing.md;
        final sidebarTopInset =
            topChromeInset + menuButtonSize + PokerSpacing.sm;
        final sidebarBottomInset = () {
          final dockClearance =
              constraints.maxHeight - scene.heroDockRect.top + PokerSpacing.sm;
          final minBottomInset = safePadding.bottom + PokerSpacing.md;
          return dockClearance > minBottomInset
              ? dockClearance
              : minBottomInset;
        }();
        final uiSpec = PokerUiSpec.fromContext(context);
        final minBoardRowWidth =
            ShowdownContent.minPanelWidthForBoardRowSingleLine(
          uiSpec,
          cardScale: ShowdownSidebar.sidebarCardScale,
        );
        final availableSidebarWidth =
            constraints.maxWidth - panelLeftInset - panelRightInset;
        final maxDesktopSidebarWidth =
            availableSidebarWidth < 560.0 ? availableSidebarWidth : 560.0;
        final minDesktopSidebarWidth =
            (minBoardRowWidth + PokerSpacing.lg) < maxDesktopSidebarWidth
                ? (minBoardRowWidth + PokerSpacing.lg)
                : maxDesktopSidebarWidth;
        final preferredDesktopSidebarWidth = maxDesktopSidebarWidth <= 0
            ? 0.0
            : (constraints.maxWidth * 0.42)
                .clamp(
                  minDesktopSidebarWidth,
                  maxDesktopSidebarWidth,
                )
                .toDouble();
        final sidebarWidth = useMobileDock
            ? availableSidebarWidth
            : (preferredDesktopSidebarWidth <= availableSidebarWidth
                ? preferredDesktopSidebarWidth
                : availableSidebarWidth);

        return Stack(
          fit: StackFit.expand,
          children: [
            pokerGame.buildWidget(
              gameState,
              _gameFocusNode,
              scene: scene,
              showHeroSeatCards: showTableHeroCards,
              heroCardsRevealed: heroCardsRevealed,
              onToggleHeroCards: () {
                final currentGame = model.game;
                final currentHeroIndex = currentGame?.players
                        .indexWhere((player) => player.id == model.playerId) ??
                    -1;
                final currentlyRevealed = currentGame != null &&
                        currentHeroIndex >= 0 &&
                        currentHeroIndex < currentGame.players.length
                    ? currentGame.players[currentHeroIndex].cardsRevealed
                    : false;
                if (currentlyRevealed) {
                  model.hideCards();
                } else {
                  model.showCards();
                }
              },
              onReadyHotkey:
                  !_isShowdown && isWaiting ? () => model.setReady() : null,
            ),
            if (_isShowdown && (model.showdownResultLabel ?? '').isNotEmpty)
              ShowdownBoardLabel(
                text: model.showdownResultLabel!,
                scene: scene,
                compact: useMobileDock,
              ),
            if (_isShowdown)
              ShowdownFxOverlay(
                model: model,
                layout: TableLayout.fromScene(scene),
              ),
            if (!_isShowdown && isWaiting)
              pokerGame.buildReadyToPlayOverlay(
                context,
                isReady,
                false,
                '',
                () => model.setReady(),
                gameState,
              ),
            if (showShowdownChrome)
              Positioned.fill(
                child: IgnorePointer(
                  ignoring: !_showSidebar,
                  child: GestureDetector(
                    behavior: HitTestBehavior.opaque,
                    onTap: _closeSidebar,
                    child: AnimatedOpacity(
                      duration: const Duration(milliseconds: 220),
                      curve: Curves.easeOutCubic,
                      opacity: _showSidebar ? 1 : 0,
                      child: Container(
                        color: Colors.black.withValues(alpha: 0.26),
                      ),
                    ),
                  ),
                ),
              ),
            if (showShowdownChrome && sidebarShowdown != null)
              Positioned(
                left: panelLeftInset,
                top: sidebarTopInset,
                bottom: sidebarBottomInset,
                width: sidebarWidth,
                child: IgnorePointer(
                  ignoring: !_showSidebar,
                  child: AnimatedSlide(
                    duration: const Duration(milliseconds: 280),
                    curve: Curves.easeOutCubic,
                    offset: _showSidebar ? Offset.zero : const Offset(-1.08, 0),
                    child: AnimatedOpacity(
                      duration: const Duration(milliseconds: 220),
                      curve: Curves.easeOutCubic,
                      opacity: _showSidebar ? 1 : 0,
                      child: ShowdownSidebar(
                        showdown: sidebarShowdown,
                        heroId: model.playerId,
                        visible: true,
                        onClose: _closeSidebar,
                      ),
                    ),
                  ),
                ),
              ),
            if (showShowdownChrome && !_showSidebar)
              Positioned(
                left: safePadding.left +
                    PokerSpacing.md +
                    menuCornerClearance +
                    toggleInset,
                top: topChromeInset,
                child: PokerLastHandButton(
                  active: _showSidebar,
                  onTap: () => setState(() => _showSidebar = !_showSidebar),
                ),
              ),
            Positioned.fromRect(
              rect: scene.heroDockRect,
              child: Container(
                key: const Key('poker-hero-dock'),
                child: useMobileDock
                    ? (_isShowdown
                        ? MobileHeroActionPanel.passive(
                            model: model,
                            reserveActionSpace: false,
                            footer: showdownFooter,
                          )
                        : MobileHeroActionPanel(
                            model: model,
                            showBetInput: _showBetInput,
                            betCtrl: _betCtrl,
                            onToggleBetInput: () =>
                                setState(() => _showBetInput = !_showBetInput),
                            onCloseBetInput: () =>
                                setState(() => _showBetInput = false),
                          ))
                    : (_isShowdown
                        ? BottomActionDock.passive(
                            model: model,
                            reserveActionSpace: false,
                            footer: showdownFooter,
                          )
                        : BottomActionDock(
                            model: model,
                            showBetInput: _showBetInput,
                            betCtrl: _betCtrl,
                            onToggleBetInput: () =>
                                setState(() => _showBetInput = !_showBetInput),
                            onCloseBetInput: () =>
                                setState(() => _showBetInput = false),
                          )),
              ),
            ),
          ],
        );
      },
    );
  }
}
