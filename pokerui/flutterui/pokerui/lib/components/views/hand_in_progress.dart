import 'package:flutter/material.dart';
import 'package:pokerui/components/dialogs/last_showdown.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/models/poker.dart';

class HandInProgressView extends StatefulWidget {
  const HandInProgressView({super.key, required this.model});
  final PokerModel model;

  @override
  State<HandInProgressView> createState() => _HandInProgressViewState();

  static int calculateTotalBet(int amt, int currentBet, int myBet, int bb) {
    // Treat the entered amount as the target total bet, regardless of prior
    // contribution (blinds or previous bet).
    return amt;
  }
}

class _HandInProgressViewState extends State<HandInProgressView> {
  final TextEditingController _betCtrl = TextEditingController();
  bool _showBetInput = false;
  bool _wasMyTurn = false;


  @override
  void dispose() {
    _betCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final game = widget.model.game;
    if (game == null) {
      return const Center(child: Text('No game data available'));
    }

    // Close the raise input when it's no longer our turn so we come back to the
    // compact button row on the next action.
    final canAct = widget.model.canAct;
    if (canAct && !_wasMyTurn) {
      // New turn: clear any stale raise input so defaults reseed correctly.
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
    final pokerGame = PokerGame(
      widget.model.playerId,
      widget.model,
      tableTheme: TableThemeConfig.fromKey(context.tableTheme),
      cardTheme: cardColorThemeFromKey(context.cardTheme),
      showTableLogo: context.showTableLogo,
    );

    return Stack(
      children: [
        // Poker game visualization
        pokerGame.buildWidget(game, focusNode),

        // Bet/call FX overlay
        _BetFxOverlay(model: widget.model),

        // "Last Hand" button at bottom left
        if (widget.model.hasLastShowdown)
          Positioned(
            bottom: 12,
            left: 12,
            child: SafeArea(
              child: Tooltip(
                message: 'View last showdown',
                child: Material(
                  color: Colors.transparent,
                  child: InkWell(
                    onTap: () => LastShowdownDialog.show(context, widget.model),
                    borderRadius: BorderRadius.circular(8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 8),
                      decoration: BoxDecoration(
                        color: Colors.black.withOpacity(0.6),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                            color: Colors.white.withOpacity(0.3), width: 1),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.history, color: Colors.white70, size: 16),
                          SizedBox(width: 6),
                          Text(
                            'Last Hand',
                            style:
                                TextStyle(color: Colors.white70, fontSize: 12),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),

        // Action buttons overlay - positioned at bottom right
        Positioned(
          bottom: 0,
          right: 0,
          child: SafeArea(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: [
                    Colors.transparent,
                    Colors.black.withOpacity(0.8),
                  ],
                ),
              ),
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                reverse: true,
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                if (widget.model.canAct) ...[
                  // Fold is always available on your turn
                  ElevatedButton(
                    onPressed: () => widget.model.fold(),
                    style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                    child: const Text('Fold (F)'),
                  ),
                  const SizedBox(width: 8),
                  // Show Check or Call only when appropriate
                  Builder(builder: (context) {
                    final g = widget.model.game;
                    final me = widget.model.me;
                    final currentBet = g?.currentBet ?? 0;
                    final myBet = me?.currentBet ?? 0;
                    final canCheck = myBet >= currentBet;
                    final toCall = (currentBet - myBet) > 0 ? (currentBet - myBet) : 0;
                    return Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (canCheck) ...[
                          ElevatedButton(
                            onPressed: () => widget.model.check(),
                            child: const Text('Check (K)'),
                          ),
                          const SizedBox(width: 8),
                        ] else ...[
                          ElevatedButton(
                            onPressed: () => widget.model.callBet(),
                            child: Text('Call${toCall > 0 ? ' ($toCall)' : ''} (C)'),
                          ),
                          const SizedBox(width: 8),
                        ],
                        // Bet/Raise button toggles bet input visibility
                        Builder(builder: (context) {
                          final tid = widget.model.currentTableId;
                          final table = tid == null
                              ? null
                              : (() {
                                  final matches = widget.model.tables.where((t) => t.id == tid).toList();
                                  return matches.isNotEmpty ? matches.first : null;
                                })();
                          // Prefer big blind from the live game snapshot, fall back to lobby table list
                          final bb = (widget.model.game?.bigBlind ?? 0) > 0
                              ? widget.model.game!.bigBlind
                              : (table?.bigBlind ?? 0);
                          final isRaise = currentBet > 0 && myBet < currentBet;

                          void seedDefault() {
                            // Pre-fill with 3x BB or 3x current bet, whichever is higher
                            // Use 3x current bet if currentBet is greater than or equal to 3x BB
                            final threeBB = bb * 3;
                            final targetTotal = (bb > 0 && currentBet >= threeBB) ? (currentBet * 3) : threeBB;
                            _betCtrl.text = targetTotal.toString();
                          }

                          Future<void> submitBet() async {
                            final raw = _betCtrl.text.trim();
                            final amt = int.tryParse(raw) ?? 0;
                            if (amt <= 0) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('Enter a valid bet amount')),
                              );
                              return;
                            }
                            
                            final totalBet = HandInProgressView.calculateTotalBet(amt, currentBet, myBet, bb);
                            
                            // Pre-check: when facing a bet, total must be at least currentBet
                            if (currentBet > 0 && totalBet < currentBet) {
                              final minRaise = currentBet - myBet;
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text('Must add at least $minRaise to call ($currentBet total)')),
                              );
                              return;
                            }
                            
                            final ok = await widget.model.makeBet(totalBet);
                            if (!ok && widget.model.errorMessage.isNotEmpty) {
                              if (mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(content: Text(widget.model.errorMessage)),
                                );
                              }
                              return;
                            }
                            setState(() {
                              _showBetInput = false;
                            });
                          }

                          void setTo3xBB() {
                            // Set to 3x BB or 3x current bet, whichever is higher
                            // Use 3x current bet if currentBet is greater than or equal to 3x BB
                            final threeBB = bb * 3;
                            final targetTotal = (bb > 0 && currentBet >= threeBB) ? (currentBet * 3) : threeBB;
                            _betCtrl.text = targetTotal.toString();
                          }

                          final myBalance = widget.model.me?.balance ?? 0;
                          final wouldBeAllIn = myBalance > 0 && myBalance <= (currentBet - myBet);
                          if (!_showBetInput) {
                            return ElevatedButton(
                              onPressed: () {
                                setState(() {
                                  _showBetInput = true;
                                });
                                if (_betCtrl.text.isEmpty) seedDefault();
                              },
                              style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
                              child: Text(isRaise
                                  ? (wouldBeAllIn ? 'All-in' : 'Raise')
                                  : 'Bet'),
                            );
                          }

                          // Bet input row (visible after pressing Bet/Raise)
                          return Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  SizedBox(
                                    width: 110,
                                    child: TextField(
                                      controller: _betCtrl,
                                      keyboardType: TextInputType.number,
                                      style: const TextStyle(color: Colors.white),
                                      decoration: InputDecoration(
                                        labelText: isRaise ? 'Total raise' : 'Total bet',
                                        labelStyle: const TextStyle(color: Colors.white70, fontSize: 12),
                                        isDense: true,
                                        contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                                        hintText: isRaise ? 'e.g. ${currentBet > 0 ? currentBet : bb}' : 'e.g. ${bb > 0 ? bb * 3 : 50}',
                                        hintStyle: const TextStyle(color: Colors.white54),
                                        filled: true,
                                        fillColor: Colors.black54,
                                        border: OutlineInputBorder(
                                          borderRadius: BorderRadius.circular(8),
                                          borderSide: const BorderSide(color: Colors.white24),
                                        ),
                                      ),
                                      onSubmitted: (_) => submitBet(),
                                    ),
                                  ),
                                  const SizedBox(height: 4),
                                  Builder(builder: (context) {
                                    final entered = int.tryParse(_betCtrl.text.trim()) ?? 0;
                                    final delta = entered > myBet ? (entered - myBet) : 0;
                                    final maxTotal = (widget.model.me?.balance ?? 0) + myBet;
                                    final capped = entered > maxTotal ? maxTotal : entered;
                                    final displayEntered =
                                        capped > 0 ? capped : (isRaise ? currentBet : bb * 3);
                                    final displayDelta = displayEntered > myBet ? (displayEntered - myBet) : 0;
                                    if (displayDelta == displayEntered) {
                                      return const SizedBox.shrink();
                                    }
                                    final isAllIn = displayEntered == maxTotal && maxTotal > 0;
                                    final label = isAllIn
                                        ? 'All-in $displayEntered'
                                        : 'Adds $displayDelta, total $displayEntered';
                                    return Text(
                                      label,
                                      style: const TextStyle(color: Colors.white70, fontSize: 11),
                                    );
                                  }),
                                ],
                              ),
                              const SizedBox(width: 6),
                              Builder(
                                builder: (context) {
                                  final threeBB = bb * 3;
                                  // Show "3x Bet" if currentBet is greater than or equal to 3x BB
                                  final buttonText = (bb > 0 && currentBet >= threeBB) ? '3x Bet' : '3x BB';
                                  return ElevatedButton(
                                    onPressed: bb > 0 ? setTo3xBB : null,
                                    child: Text(buttonText),
                                  );
                                },
                              ),
                              const SizedBox(width: 6),
                              ElevatedButton(
                                onPressed: submitBet,
                                style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
                                child: Text(() {
                                  final meBal = widget.model.me?.balance ?? 0;
                                  final entered = int.tryParse(_betCtrl.text.trim()) ?? 0;
                                  final target = entered > 0 ? entered : currentBet;
                                  final myTotal = meBal + myBet;
                                  final isAllIn = target >= myTotal && myTotal > 0;
                                  if (isAllIn) return 'All-in';
                                  return isRaise ? 'Raise' : 'Bet';
                                }()),
                              ),
                              const SizedBox(width: 6),
                              TextButton(
                                onPressed: () {
                                  setState(() {
                                    _showBetInput = false;
                                  });
                                },
                                child: const Text('Cancel'),
                              )
                            ],
                          );
                        }),
                      ],
                    );
                  }),
                ] else ...[
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.7),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      widget.model.autoAdvanceAllIn
                          ? 'Auto-advancing (all-in)'
                          : 'Waiting...',
                      style: const TextStyle(color: Colors.white, fontSize: 14),
                    ),
                  ),
                ],
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

class _BetFxOverlay extends StatefulWidget {
  const _BetFxOverlay({required this.model});
  final PokerModel model;

  @override
  State<_BetFxOverlay> createState() => _BetFxOverlayState();
}

class _BetFxOverlayState extends State<_BetFxOverlay> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  int _lastFxMs = 0;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: const Duration(milliseconds: 700));
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
      final box = layout.viewport;
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
      );
      final from = seatPositions[fx.playerId] ?? layout.center;
      final overlay = computeTopOverlayLayout(layout.viewport, hasCurrentBet);
      final to = overlay.potCenter(box); // pot label center

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
  const _AnimatedChip({required this.t, required this.delay, required this.from, required this.to});
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
        // Hide when not in flight or after arrival to avoid chips lingering
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
                BoxShadow(color: Colors.black.withOpacity(0.3), blurRadius: 4, spreadRadius: 0.5),
              ],
            ),
          ),
        );
      },
    );
  }
}
