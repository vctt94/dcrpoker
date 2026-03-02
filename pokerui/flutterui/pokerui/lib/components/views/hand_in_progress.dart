import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/poker/showdown_sidebar.dart';
import 'package:pokerui/components/poker/bet_sidebar.dart';
import 'package:pokerui/components/poker/bottom_action_dock.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/models/poker.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class HandInProgressView extends StatefulWidget {
  const HandInProgressView({super.key, required this.model});
  final PokerModel model;

  @override
  State<HandInProgressView> createState() => _HandInProgressViewState();

  static int calculateTotalBet(int amt, int currentBet, int myBet, int bb) {
    return amt;
  }
}

class _HandInProgressViewState extends State<HandInProgressView> {
  final TextEditingController _betCtrl = TextEditingController();
  bool _showBetInput = false;
  bool _wasMyTurn = false;
  bool _showSidebar = false;

  @override
  void dispose() {
    _betCtrl.dispose();
    super.dispose();
  }

  void _toggleSidebar() {
    setState(() {
      _showSidebar = !_showSidebar;
    });
  }

  void _closeSidebar() {
    setState(() {
      _showSidebar = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    final game = widget.model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    final canAct = widget.model.canAct;
    if (canAct && !_wasMyTurn) {
      _betCtrl.clear();
    }
    if (_showBetInput && _wasMyTurn && !canAct) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          setState(() {
            _showBetInput = false;
          });
        }
      });
    }
    _wasMyTurn = canAct;

    final focusNode = FocusNode();
    final theme = PokerThemeConfig.fromContext(context);
    final bp = PokerBreakpointQuery.of(context);
    final isPhone = bp.isNarrow;
    final aspect = tableAspectRatio(bp);
    final pokerGame = PokerGame(
      widget.model.playerId,
      widget.model,
      theme: theme,
    );

    final tableStack = Stack(
      children: [
        pokerGame.buildWidget(
          game,
          focusNode,
          aspectRatio: aspect,
          showHeroCardsOverlay: !isPhone,
        ),
        if (game.currentBet > 0 && game.phase != pr.GamePhase.SHOWDOWN)
          BetSidebar(
            gameState: game,
            playerId: widget.model.playerId,
            theme: theme,
          ),
        _BetFxOverlay(model: widget.model),
        if (_showSidebar && widget.model.hasLastShowdown)
          ShowdownSidebar(
            model: widget.model,
            isVisible: _showSidebar,
            onClose: _closeSidebar,
          ),
      ],
    );

    if (isPhone) {
      return LayoutBuilder(builder: (context, constraints) {
        final maxTableHeight =
            constraints.maxHeight - mobileHeroPanelMinHeight(bp);
        final tableMax = maxTableHeight < 220.0 ? 220.0 : maxTableHeight;
        final tableHeight =
            (constraints.maxHeight * mobileTableHeightFraction(bp))
                .clamp(220.0, tableMax)
                .toDouble();
        return Column(
          children: [
            SizedBox(height: tableHeight, child: tableStack),
            Expanded(
              child: MobileHeroActionPanel(
                model: widget.model,
                showBetInput: _showBetInput,
                betCtrl: _betCtrl,
                onToggleBetInput: () => setState(() => _showBetInput = true),
                onCloseBetInput: () => setState(() => _showBetInput = false),
                hasLastShowdown: widget.model.hasLastShowdown,
                showSidebar: _showSidebar,
                onToggleSidebar: _toggleSidebar,
              ),
            ),
          ],
        );
      });
    }

    return Column(
      children: [
        Expanded(child: tableStack),
        BottomActionDock(
          model: widget.model,
          showBetInput: _showBetInput,
          betCtrl: _betCtrl,
          onToggleBetInput: () => setState(() => _showBetInput = true),
          onCloseBetInput: () => setState(() => _showBetInput = false),
          hasLastShowdown: widget.model.hasLastShowdown,
          showSidebar: _showSidebar,
          onToggleSidebar: _toggleSidebar,
        ),
      ],
    );
  }
}

class _BetFxOverlay extends StatefulWidget {
  const _BetFxOverlay({required this.model});
  final PokerModel model;

  @override
  State<_BetFxOverlay> createState() => _BetFxOverlayState();
}

class _BetFxOverlayState extends State<_BetFxOverlay>
    with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 700));
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant _BetFxOverlay oldWidget) {
    super.didUpdateWidget(oldWidget);
    final fx = widget.model.lastBetFx;
    if (fx != null && fx.createdMs != _lastFxMs) {
      _lastFxMs = fx.createdMs;
      _ctrl
        ..reset()
        ..forward();
    }
  }

  @override
  Widget build(BuildContext context) {
    final fx = widget.model.lastBetFx;
    final game = widget.model.game;
    if (fx == null || game == null) return const SizedBox.shrink();

    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size);
      final theme = PokerThemeConfig.fromContext(context);
      final hasCurrentBet = game.currentBet > 0;
      final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);
      final seatPositions = seatPositionsFor(
        game.players,
        widget.model.playerId,
        layout.center,
        layout.ringRadiusX,
        layout.ringRadiusY,
        clampBounds: layout.viewport,
        minSeatTop: minSeatTop,
        uiSizeMultiplier: theme.uiSizeMultiplier,
      );
      final from = seatPositions[fx.playerId] ?? layout.center;
      final to =
          potChipCenter(layout, uiSizeMultiplier: theme.uiSizeMultiplier);

      final anim = CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic);
      final t = Tween(begin: 0.0, end: 1.0).animate(anim);

      const particles = 4;
      final children = <Widget>[];
      for (int i = 0; i < particles; i++) {
        final delay = i * 0.06;
        children.add(_AnimatedChip(t: t, delay: delay, from: from, to: to));
      }

      return IgnorePointer(child: Stack(children: children));
    });
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
