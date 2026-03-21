import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/bet_amounts.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/bet_sidebar.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/components/poker/table.dart';
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

class _HandInProgressViewState extends State<HandInProgressView>
    with SingleTickerProviderStateMixin {
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
    final bp = PokerBreakpointQuery.of(context);
    final isMobile = bp.isNarrow;
    final pokerGame = PokerGame(model.playerId, model, theme: theme);
    final tableAr = tableAspectRatio(bp);
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

    if (isMobile) {
      return Column(
        children: [
          Expanded(
            child: Stack(
              fit: StackFit.expand,
              children: [
                pokerGame.buildWidget(gameState, _gameFocusNode,
                    aspectRatio: tableAr,
                    showHeroCardsOverlay: false,
                    onReadyHotkey: isWaiting ? () => model.setReady() : null),
                if (isWaiting)
                  pokerGame.buildReadyToPlayOverlay(context, isReady, false, '',
                      () => model.setReady(), gameState),
                _AnimatedChipOverlay(
                    model: model, theme: theme, aspectRatio: tableAr),
                if (!isWaiting && gameState.currentBet > 0)
                  BetSidebar(model: model),
              ],
            ),
          ),
          if (!isWaiting)
            MobileHeroActionPanel(
              model: model,
              showBetInput: _showBetInput,
              betCtrl: _betCtrl,
              onToggleBetInput: () =>
                  setState(() => _showBetInput = !_showBetInput),
              onCloseBetInput: () => setState(() => _showBetInput = false),
              hasLastShowdown: _hasLastShowdown,
              showSidebar: _showSidebar,
              onToggleSidebar: () =>
                  setState(() => _showSidebar = !_showSidebar),
            ),
          if (_showSidebar && _hasLastShowdown)
            SizedBox(
              height: 200,
              child: ShowdownSidebar(
                model: model,
                visible: true,
              ),
            ),
        ],
      );
    }

    // Desktop layout
    return Stack(
      fit: StackFit.expand,
      children: [
        pokerGame.buildWidget(gameState, _gameFocusNode,
            aspectRatio: tableAr,
            onReadyHotkey: isWaiting ? () => model.setReady() : null),
        if (isWaiting)
          pokerGame.buildReadyToPlayOverlay(
              context, isReady, false, '', () => model.setReady(), gameState),
        _AnimatedChipOverlay(model: model, theme: theme, aspectRatio: tableAr),
        if (!isWaiting && gameState.currentBet > 0) BetSidebar(model: model),
        if (!isWaiting)
          Positioned(
            bottom: 0,
            left: 0,
            right: 0,
            child: BottomActionDock(
              model: model,
              showBetInput: _showBetInput,
              betCtrl: _betCtrl,
              onToggleBetInput: () =>
                  setState(() => _showBetInput = !_showBetInput),
              onCloseBetInput: () => setState(() => _showBetInput = false),
              hasLastShowdown: _hasLastShowdown,
              showSidebar: _showSidebar,
              onToggleSidebar: () =>
                  setState(() => _showSidebar = !_showSidebar),
            ),
          ),
        if (_showSidebar && _hasLastShowdown)
          Positioned(
            left: 0,
            top: 0,
            bottom: 80,
            width: 300,
            child: ShowdownSidebar(
              model: model,
              visible: true,
            ),
          ),
      ],
    );
  }
}

/// Animated chip particles from each player's bet to the pot.
class _AnimatedChipOverlay extends StatefulWidget {
  const _AnimatedChipOverlay({
    required this.model,
    required this.theme,
    required this.aspectRatio,
  });
  final PokerModel model;
  final PokerThemeConfig theme;
  final double aspectRatio;

  @override
  State<_AnimatedChipOverlay> createState() => _AnimatedChipOverlayState();
}

class _AnimatedChipOverlayState extends State<_AnimatedChipOverlay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 700),
    );
  }

  @override
  void didUpdateWidget(covariant _AnimatedChipOverlay old) {
    super.didUpdateWidget(old);
    final fx = widget.model.lastBetFx;
    if (fx != null && fx.createdMs != _lastFxMs) {
      _lastFxMs = fx.createdMs;
      _ctrl
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final fx = widget.model.lastBetFx;
    final game = widget.model.game;
    if (fx == null || game == null) return const SizedBox.shrink();

    return AnimatedBuilder(
      animation: _ctrl,
      builder: (context, child) {
        return LayoutBuilder(builder: (context, c) {
          final layout =
              resolveTableLayout(c.biggest, aspectRatio: widget.aspectRatio);
          final hasCurrentBet = game.currentBet > 0;
          final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);
          final seatPositions = seatPositionsFor(
            game.players,
            widget.model.playerId,
            layout.center,
            layout.ringRadiusX,
            layout.ringRadiusY,
            clampBounds: layout.canvasBounds,
            minSeatTop: minSeatTop,
            uiSizeMultiplier: widget.theme.uiSizeMultiplier,
          );
          final from = seatPositions[fx.playerId] ?? layout.center;
          final to = potChipCenter(layout,
              uiSizeMultiplier: widget.theme.uiSizeMultiplier);
          final anim =
              CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic);
          final t = Tween(begin: 0.0, end: 1.0).animate(anim);

          const particles = 4;
          final children = <Widget>[];
          for (int i = 0; i < particles; i++) {
            final delay = i * 0.06;
            children.add(_AnimatedChip(t: t, delay: delay, from: from, to: to));
          }

          return IgnorePointer(child: Stack(children: children));
        });
      },
    );
  }
}

class _AnimatedChip extends StatelessWidget {
  const _AnimatedChip(
      {required this.t,
      required this.delay,
      required this.from,
      required this.to});
  final Animation<double> t;
  final double delay;
  final Offset from;
  final Offset to;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: t,
      builder: (context, child) {
        final span = 1.0 - delay;
        if (span <= 0) return const SizedBox.shrink();
        final raw = (t.value - delay) / span;
        if (raw <= 0.0 || raw >= 1.0) {
          return const SizedBox.shrink();
        }
        final eased = Curves.easeOut.transform(raw.clamp(0.0, 1.0));
        final dx = from.dx + (to.dx - from.dx) * eased;
        final dy = from.dy + (to.dy - from.dy) * eased;
        return Positioned(
          left: dx - 6,
          top: dy - 6,
          child: Container(
            width: 12,
            height: 12,
            decoration: BoxDecoration(
              color: Colors.amber,
              border: Border.all(color: Colors.orange.shade900, width: 1.5),
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                    color: Colors.black.withOpacity(0.3),
                    blurRadius: 4,
                    spreadRadius: 0.5),
              ],
            ),
          ),
        );
      },
    );
  }
}
