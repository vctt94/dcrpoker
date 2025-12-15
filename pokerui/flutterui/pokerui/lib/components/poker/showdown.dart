import 'dart:async';
import 'package:flutter/material.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;

class ShowdownView extends StatefulWidget {
  const ShowdownView({super.key, required this.model});
  final PokerModel model;

  @override
  State<ShowdownView> createState() => _ShowdownViewState();
}

class _ShowdownViewState extends State<ShowdownView> {
  Timer? _countdownTimer;
  int _secondsRemaining = 5;

  @override
  void initState() {
    super.initState();
    _startCountdown();
  }

  @override
  void dispose() {
    _countdownTimer?.cancel();
    super.dispose();
  }

  void _startCountdown() {
    _secondsRemaining = 5;
    _countdownTimer?.cancel();
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (mounted) {
        setState(() {
          _secondsRemaining--;
          if (_secondsRemaining <= 0) {
            timer.cancel();
          }
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final model = widget.model;
    final game = model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    final focusNode = FocusNode();
    final pokerGame = PokerGame(
      model.playerId,
      model,
      tableTheme: TableThemeConfig.fromKey(context.tableTheme),
      cardTheme: cardColorThemeFromKey(context.cardTheme),
      showTableLogo: context.showTableLogo,
    );
    final winners = model.lastWinners;
    final players = game.players;

    String _pLabel(String pid) {
      final idx = players.indexWhere((p) => p.id == pid);
      return idx >= 0 ? 'P${idx + 1}' : 'P?';
    }

    String winnerName(String pid) {
      final pl = players.firstWhere((p) => p.id == pid,
          orElse: () => const UiPlayer(
                id: '',
                name: '',
                balance: 0,
                hand: [],
                currentBet: 0,
                folded: false,
                isTurn: false,
                isAllIn: false,
                isDealer: false,
                isSmallBlind: false,
                isBigBlind: false,
                isReady: false,
                handDesc: '',
                isDisconnected: true,
              ));
      return pl.name.isNotEmpty ? pl.name : _pLabel(pid);
    }

    String winnerDesc(pr.HandRank rank) {
      switch (rank) {
        case pr.HandRank.HIGH_CARD:
          return 'High Card';
        case pr.HandRank.PAIR:
          return 'Pair';
        case pr.HandRank.TWO_PAIR:
          return 'Two Pair';
        case pr.HandRank.THREE_OF_A_KIND:
          return 'Three of a Kind';
        case pr.HandRank.STRAIGHT:
          return 'Straight';
        case pr.HandRank.FLUSH:
          return 'Flush';
        case pr.HandRank.FULL_HOUSE:
          return 'Full House';
        case pr.HandRank.FOUR_OF_A_KIND:
          return 'Four of a Kind';
        case pr.HandRank.STRAIGHT_FLUSH:
          return 'Straight Flush';
        case pr.HandRank.ROYAL_FLUSH:
          return 'Royal Flush';
        default:
          return rank.name;
      }
    }

    return Stack(
      children: [
        // Poker game visualization (table + canvas elements)
        pokerGame.buildWidget(game, focusNode),

        // Showdown FX overlay: chip flow to winners
        _ShowdownFxOverlay(model: model),

        // Compact winners banner at the top center
        if (winners.isNotEmpty)
          Positioned(
            top: 16,
            left: 0,
            right: 0,
            child: Center(
              child: Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.78),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.amber, width: 1.5),
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    const Text(
                      'Showdown',
                      style: TextStyle(
                          color: Colors.amber,
                          fontSize: 16,
                          fontWeight: FontWeight.w800),
                    ),
                    const SizedBox(height: 6),
                    for (final w in winners)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 2),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.center,
                          children: [
                            Text(
                              winnerName(w.playerId),
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w700),
                              overflow: TextOverflow.ellipsis,
                            ),
                            Text(
                              winnerDesc(w.handRank),
                              style: const TextStyle(
                                  color: Colors.white70,
                                  fontSize: 12,
                                  fontStyle: FontStyle.italic),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),

        // Countdown and Skip button at bottom center (only if game end is pending)
        if (model.isGameEndPending)
          Positioned(
            bottom: 16,
            left: 0,
            right: 0,
            child: SafeArea(
              child: Center(
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                  decoration: BoxDecoration(
                    color: Colors.black.withOpacity(0.8),
                    borderRadius: BorderRadius.circular(20),
                    border: Border.all(color: Colors.white24),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // Countdown indicator
                      Container(
                        width: 32,
                        height: 32,
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          border: Border.all(color: Colors.amber, width: 2),
                        ),
                        child: Center(
                          child: Text(
                            '$_secondsRemaining',
                            style: const TextStyle(
                              color: Colors.amber,
                              fontSize: 16,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      // Skip button
                      ElevatedButton.icon(
                        onPressed: () {
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
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _ShowdownFxOverlay extends StatefulWidget {
  const _ShowdownFxOverlay({required this.model});
  final PokerModel model;

  @override
  State<_ShowdownFxOverlay> createState() => _ShowdownFxOverlayState();
}

class _ShowdownFxOverlayState extends State<_ShowdownFxOverlay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _chipCtrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _chipCtrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 900));
    _maybeRestartFx();
  }

  @override
  void didUpdateWidget(covariant _ShowdownFxOverlay oldWidget) {
    super.didUpdateWidget(oldWidget);
    _maybeRestartFx();
  }

  @override
  void dispose() {
    _chipCtrl.dispose();
    super.dispose();
  }

  void _maybeRestartFx() {
    final winners = widget.model.lastWinners;
    final fxMs = widget.model.lastShowdownFxMs;
    // ignore: avoid_print
    if (winners.isEmpty || fxMs == 0) return;
    if (fxMs != _lastFxMs) {
      // Debug visibility to trace animation triggers
      // ignore: avoid_print
      _lastFxMs = fxMs;
      _chipCtrl
        ..reset()
        ..forward();
    }
  }

  @override
  Widget build(BuildContext context) {
    final game = widget.model.game;
    if (game == null) return const SizedBox.shrink();
    final winners = widget.model.lastWinners;

    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size);
      final box = layout.viewport;
      final center = layout.center;
      final hasCurrentBet = game.currentBet > 0;
      final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);

      final chipWidgets = <Widget>[];
      if (winners.isNotEmpty && game.players.isNotEmpty) {
        final targets = seatPositionsFor(
          game.players,
          widget.model.playerId,
          center,
          layout.ringRadiusX,
          layout.ringRadiusY,
          clampBounds: layout.viewport,
          minSeatTop: minSeatTop,
        );
        final overlay = computeTopOverlayLayout(layout.viewport, hasCurrentBet);
        final potOrigin = overlay.potCenter(box);

        for (int i = 0; i < winners.length; i++) {
          final w = winners[i];
          final target = targets[w.playerId] ?? center;
          final curved = CurvedAnimation(
              parent: _chipCtrl,
              curve: const Interval(0.0, 1.0, curve: Curves.easeOut));
          final t = Tween<double>(begin: 0, end: 1).animate(curved);
          for (int j = 0; j < 3; j++) {
            final delay = j * 0.06 + i * 0.12;
            chipWidgets.add(_AnimatedChip(
              t: t,
              delay: delay,
              from: potOrigin,
              to: target,
            ));
          }
        }
      }

      return Stack(children: [
        ...chipWidgets,
      ]);
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
        final raw = (t.value - delay).clamp(0.0, 1.0);
        if (raw <= 0.0 || raw >= 1.0) {
          return const SizedBox.shrink();
        }
        final eased = Curves.easeOut.transform(raw);
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
