import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'table.dart';
import 'table_theme.dart';
import 'cards.dart';
import 'disconnected_badges.dart';
import 'bet_sidebar.dart';
import 'table_logo.dart';
import 'pot_display.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/helper.dart';

class PokerTableBackground extends StatelessWidget {
  const PokerTableBackground({super.key});
  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: CustomPaint(
        painter: _TableBackgroundPainter(),
        size: Size.infinite,
      ),
    );
  }
}

class _TableBackgroundPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final layout = resolveTableLayout(size);
    final centerX = layout.center.dx;
    final centerY = layout.center.dy;
    final tableRadiusX = layout.tableRadiusX;
    final tableRadiusY = layout.tableRadiusY;

    // Draw table surface as ellipse
    final tableRect = Rect.fromCenter(
      center: Offset(centerX, centerY),
      width: tableRadiusX * 2,
      height: tableRadiusY * 2,
    );

    // Table surface
    final tablePaint = Paint()
      ..color = const Color(0xFF0D4F3C) // Poker table green
      ..style = PaintingStyle.fill;
    canvas.drawOval(tableRect, tablePaint);

    // Table border
    final borderPaint = Paint()
      ..color = const Color(0xFF8B4513) // Brown border
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8;
    canvas.drawOval(tableRect, borderPaint);

    // Subtle shadow
    final shadowPaint = Paint()
      ..color = Colors.black.withOpacity(0.3)
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 15);
    canvas.drawOval(
      Rect.fromCenter(
        center: Offset(centerX, centerY + 5),
        width: tableRadiusX * 2,
        height: tableRadiusY * 2,
      ),
      shadowPaint,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

class PokerGame {
  final PokerModel pokerModel;
  final String playerId;
  final PokerThemeConfig theme;
  final RenderLoop _loop = RenderLoop();

  PokerGame(this.playerId, this.pokerModel, {required this.theme});

  Widget buildWidget(UiGameState gameState, FocusNode focusNode,
      {VoidCallback? onReadyHotkey}) {
    // Start/stop lightweight repaint loop only while an authoritative deadline is active.
    if (gameState.turnDeadlineUnixMs > 0) {
      _loop.start();
    } else {
      _loop.stop();
    }
    return GestureDetector(
      onTap: () => focusNode.requestFocus(),
      child: Focus(
        child: KeyboardListener(
          focusNode: focusNode..requestFocus(),
          onKeyEvent: (KeyEvent event) {
            if (event is KeyDownEvent || event is KeyRepeatEvent) {
              String keyLabel = event.logicalKey.keyLabel;
              if (onReadyHotkey != null) {
                if (event.logicalKey == LogicalKeyboardKey.space ||
                    keyLabel == 'r' ||
                    keyLabel == 'R') {
                  onReadyHotkey();
                  return;
                }
              }
              handleInput(playerId, keyLabel);
            }
          },
          child: LayoutBuilder(
            builder: (context, constraints) {
              return Center(
                child: SizedBox(
                  width: constraints.maxWidth,
                  child: AspectRatio(
                    aspectRatio: 16 / 9, // Poker table aspect ratio
                    child: RepaintBoundary(
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          // Poker table background
                          const PokerTableBackground(),

                          // Game canvas (repaints)
                          CustomPaint(
                            painter: PokerPainter(gameState, playerId, theme,
                                repaint: _loop),
                            isComplex: true,
                            willChange: true,
                          ),

                          // DCR logo overlay (if enabled) - after canvas so it's visible
                          if (theme.showTableLogo)
                            TableLogoOverlay(
                              logoPosition: theme.logoPosition,
                              uiSizeMultiplier: theme.uiSizeMultiplier,
                            ),

                          // Widget-based overlays for cards
                          IgnorePointer(
                              child: _CommunityCardsOverlay(
                                  cards: gameState.communityCards)),

                          // Hero hole cards overlay (visible during all active phases)
                          if (gameState.phase != pr.GamePhase.WAITING)
                            (gameState.phase == pr.GamePhase.SHOWDOWN
                                // Allow interaction at showdown so user can tap to show/hide
                                ? _HeroCardsOverlay(
                                    players: gameState.players,
                                    heroId: playerId,
                                    cache: pokerModel.myHoleCardsCache,
                                    model: pokerModel,
                                  )
                                // Otherwise render non-interactive to avoid stealing input
                                : IgnorePointer(
                                    child: _HeroCardsOverlay(
                                      players: gameState.players,
                                      heroId: playerId,
                                      cache: pokerModel.myHoleCardsCache,
                                      model: pokerModel,
                                    ),
                                  )),

                          // Hover hints for disconnected players
                          DisconnectedBadgesOverlay(
                            players: gameState.players,
                            heroId: playerId,
                            hasCurrentBet: gameState.currentBet > 0,
                          ),

                          // Pot display on the felt with a chips anchor for animations
                          if (gameState.pot > 0)
                            PotDisplay(
                              pot: gameState.pot,
                              theme: theme,
                            ),

                          // Betting info sidebar (right side, minimal pattern)
                          // Hide during showdown when minimal showdown is visible
                          if (gameState.currentBet > 0 &&
                              gameState.phase != pr.GamePhase.SHOWDOWN)
                            BetSidebar(
                              gameState: gameState,
                              playerId: playerId,
                              theme: theme,
                            ),
                        ],
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }

  // Build an overlay widget for the ready-to-play UI and countdown
  Widget buildReadyToPlayOverlay(
      BuildContext context,
      bool isReadyToPlay,
      bool countdownStarted,
      String countdownMessage,
      Function onReadyPressed,
      UiGameState gameState) {
    // If countdown has started, show the countdown message in the center
    if (countdownStarted) {
      return Center(
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: const Color(0xFF1B1E2C).withAlpha(230),
            borderRadius: BorderRadius.circular(15),
            boxShadow: [
              BoxShadow(
                color: Colors.blueAccent.withAlpha(76),
                spreadRadius: 3,
                blurRadius: 10,
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(
                Icons.casino,
                size: 50,
                color: Colors.blueAccent,
              ),
              const SizedBox(height: 20),
              Text(
                countdownMessage,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 40,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ],
          ),
        ),
      );
    }

    // If not ready to play, show the ready button with game controls info
    if (!isReadyToPlay) {
      return Container(
        color: Color.fromRGBO(0, 0, 0, 0.65),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Poker table visualization
              SizedBox(
                height: 80,
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Container(
                      width: 40,
                      height: 60,
                      decoration: BoxDecoration(
                        color: const Color(0xFF0D4F3C),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                            color: const Color(0xFF8B4513), width: 2),
                      ),
                      child: const Center(
                        child: Icon(
                          Icons.casino,
                          color: Colors.white,
                          size: 30,
                        ),
                      ),
                    ),
                    const SizedBox(width: 20),
                    Container(
                      width: 20,
                      height: 20,
                      decoration: BoxDecoration(
                        color: Colors.amber,
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                    const SizedBox(width: 20),
                    Container(
                      width: 40,
                      height: 60,
                      decoration: BoxDecoration(
                        color: const Color(0xFF0D4F3C),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                            color: const Color(0xFF8B4513), width: 2),
                      ),
                      child: const Center(
                        child: Icon(
                          Icons.casino,
                          color: Colors.white,
                          size: 30,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 40),
              const Text(
                "Ready to play poker?",
                style: TextStyle(
                  color: Colors.blueAccent,
                  fontSize: 32,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 40),
              ElevatedButton(
                onPressed: () => onReadyPressed(),
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blueAccent,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 50, vertical: 15),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(30),
                  ),
                ),
                child: const Text(
                  "I'm Ready!",
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              ),
              const SizedBox(height: 50),
              Container(
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: const Color(0xFF1B1E2C),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.blueAccent.withAlpha(76)),
                ),
                child: Column(
                  children: [
                    const Text(
                      "POKER CONTROLS",
                      style: TextStyle(
                        color: Colors.blueAccent,
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 15),
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        _controlKey("F", "Fold"),
                        const SizedBox(width: 10),
                        _controlKey("C", "Call"),
                        const SizedBox(width: 10),
                        _controlKey("K", "Check"),
                        const SizedBox(width: 10),
                        _controlKey("B", "Bet"),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    }

    // If ready but waiting for opponent
    return Center(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: const Color(0xFF1B1E2C).withAlpha(230),
          borderRadius: BorderRadius.circular(15),
          boxShadow: [
            BoxShadow(
              color: Colors.blueAccent.withAlpha(76),
              spreadRadius: 3,
              blurRadius: 10,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(
              Icons.casino,
              size: 50,
              color: Colors.blueAccent,
            ),
            const SizedBox(height: 20),
            const Text(
              "Waiting for players to get ready...",
              style: TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: 40,
              height: 40,
              child: CircularProgressIndicator(
                color: Colors.blueAccent,
                backgroundColor: Colors.grey.withAlpha(51),
                strokeWidth: 4,
              ),
            ),
          ],
        ),
      ),
    );
  }

  // Helper widget for control key display
  Widget _controlKey(String key, String action) {
    return Column(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: Colors.grey.shade800,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: Colors.grey.shade600),
          ),
          child: Center(
            child: Text(
              key,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ),
        const SizedBox(height: 5),
        Text(
          action,
          style: const TextStyle(
            color: Colors.white70,
            fontSize: 12,
          ),
        ),
      ],
    );
  }

  Future<void> handleInput(String playerId, String data) async {
    await _sendKeyInput(data);
  }

  Future<void> _sendKeyInput(String data) async {
    try {
      if (!pokerModel.canAct) {
        return;
      }
      switch (data.toUpperCase()) {
        case 'F':
          await pokerModel.fold();
          break;
        case 'C':
          await pokerModel.callBet();
          break;
        case 'K':
          await pokerModel.check();
          break;
        case 'B':
          // Smart default: bet/raise to 3x big blind, or 3x current bet if higher.
          final g = pokerModel.game;
          final currentBet = g?.currentBet ?? 0;
          // Prefer blinds from the authoritative game snapshot; fall back to lobby table if needed
          final tid = pokerModel.currentTableId;
          final table = tid == null
              ? null
              : pokerModel.tables
                  .where((t) => t.id == tid)
                  .cast<UiTable?>()
                  .firstWhere(
                    (t) => t != null,
                    orElse: () => null,
                  );
          final bb = g?.bigBlind ?? table?.bigBlind ?? 0;
          final threeBB = bb * 3;
          final targetTotal = currentBet > threeBB ? (currentBet * 3) : threeBB;
          // Send total bet amount to server
          if (targetTotal > 0) {
            await pokerModel.makeBet(targetTotal);
          }
          break;
        default:
          return;
      }
    } catch (e) {
      print('Poker input error: $e');
    }
  }

  String get name => 'Poker';
}

class PokerPainter extends CustomPainter {
  final UiGameState gameState;
  // This is the viewer's player ID (hero), not necessarily the player to act.
  final String currentPlayerId;
  final PokerThemeConfig theme;
  // Used to stagger simple reveal animations at showdown
  final int showdownStartMs;
  final double minSeatTop;

  PokerPainter(this.gameState, this.currentPlayerId, this.theme,
      {Listenable? repaint})
      : showdownStartMs = DateTime.now().millisecondsSinceEpoch,
        minSeatTop = 0,
        super(repaint: repaint);

  @override
  void paint(Canvas canvas, Size size) {
    final layout = resolveTableLayout(size);
    final centerX = layout.center.dx;
    final centerY = layout.center.dy;
    final tableRadiusX = layout.tableRadiusX;
    final tableRadiusY = layout.tableRadiusY;
    final hasCurrentBet = gameState.currentBet > 0;
    final minSeatTop = minSeatTopFor(layout.viewport, hasCurrentBet);

    // Draw poker table
    drawPokerTable(
        canvas, centerX, centerY, tableRadiusX, tableRadiusY, theme.tableTheme);

    // Draw players
    drawPlayers(
      canvas,
      gameState.players,
      currentPlayerId,
      gameState,
      centerX,
      centerY,
      tableRadiusX,
      tableRadiusY,
      showdownStartMs,
      size,
      theme.cardSizeMultiplier,
      theme.uiSizeMultiplier,
      playerOffsetOverride: layout.playerOffset,
      clampBounds: layout.viewport,
      minSeatTop: minSeatTop,
    );

    _drawHeroHoleCards(canvas, size);

    // Draw current player's timebank badge last so it sits above cards/badges.
    drawCurrentTimebank(
      canvas,
      size,
      gameState,
      currentPlayerId,
      centerX,
      centerY,
      tableRadiusX,
      tableRadiusY,
      theme.uiSizeMultiplier,
      playerOffset: layout.playerOffset,
      clampBounds: layout.viewport,
      minSeatTop: minSeatTop,
    );
  }

  @override
  bool shouldRepaint(covariant PokerPainter old) =>
      old.gameState != gameState || old.currentPlayerId != currentPlayerId;

  void _drawHeroHoleCards(Canvas canvas, Size size) {}
}

class _CommunityCardsOverlay extends StatelessWidget {
  const _CommunityCardsOverlay({required this.cards});
  final List<pr.Card> cards;

  @override
  Widget build(BuildContext context) {
    if (cards.isEmpty) return const SizedBox.shrink();
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final box = pokerViewportRect(size);
      final center = Offset(box.left + box.width / 2, box.top + box.height / 2);
      final cw = (box.width * 0.05).clamp(32.0, 56.0).toDouble();
      final ch = cw * 1.4;
      final gap = cw * 0.10;
      final totalW = (cards.length * cw) + ((cards.length - 1) * gap);
      final startX = center.dx - totalW / 2;
      final y = center.dy - ch / 2 - 20.0;

      final children = <Widget>[];
      for (int i = 0; i < cards.length; i++) {
        final x = startX + i * (cw + gap);
        children.add(Positioned(
          left: x,
          top: y,
          width: cw,
          height: ch,
          child: CardFace(card: cards[i]),
        ));
      }
      return Stack(children: children);
    });
  }
}

class _OpponentsShowdownHandsOverlay extends StatefulWidget {
  const _OpponentsShowdownHandsOverlay(
      {required this.players, required this.heroId});
  final List<UiPlayer> players;
  final String heroId;

  @override
  State<_OpponentsShowdownHandsOverlay> createState() =>
      _OpponentsShowdownHandsOverlayState();
}

class _OpponentsShowdownHandsOverlayState
    extends State<_OpponentsShowdownHandsOverlay> {
  // Snapshot of shown hands during showdown. We only add new reveals and never
  // remove, so they remain visible even if later snapshots clear hands.
  final Map<String, List<pr.Card>> _shownHands = {};

  @override
  void initState() {
    super.initState();
    _ingest(widget.players);
  }

  @override
  void didUpdateWidget(covariant _OpponentsShowdownHandsOverlay oldWidget) {
    super.didUpdateWidget(oldWidget);
    // If new players reveal cards mid-showdown, capture them
    if (!identical(oldWidget.players, widget.players)) {
      _ingest(widget.players);
    }
  }

  void _ingest(List<UiPlayer> players) {
    for (final p in players) {
      if (p.id == widget.heroId) continue;
      if (p.hand.isNotEmpty) {
        _shownHands[p.id] = List<pr.Card>.from(p.hand);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.players.isEmpty) return const SizedBox.shrink();
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final theme = PokerThemeConfig.fromContext(context);
      final layout = resolveTableLayout(size);
      final box = layout.viewport;
      final center = layout.center;
      final playerRadius = kPlayerRadius * theme.uiSizeMultiplier;
      final minSeatTop = minSeatTopFor(layout.viewport, false);
      final seats = seatPositionsFor(
        widget.players,
        widget.heroId,
        center,
        layout.ringRadiusX,
        layout.ringRadiusY,
        clampBounds: layout.viewport,
        minSeatTop: minSeatTop,
        uiSizeMultiplier: theme.uiSizeMultiplier,
      );

      final cw = (box.width * 0.032).clamp(24.0, 36.0).toDouble();
      final ch = cw * 1.4;
      const gap = 4.0;

      final children = <Widget>[];
      for (final p in widget.players) {
        if (p.id == widget.heroId) continue;
        final pos = seats[p.id];
        if (pos == null) continue;
        final isTopHalf = pos.dy < center.dy;
        final pairW = (cw * 2) + gap;
        final minLeft = box.left + 8.0;
        final maxLeft = box.right - pairW - 8.0;
        final baseLeft = pos.dx - pairW / 2;
        final left = baseLeft.clamp(minLeft, maxLeft).toDouble();

        final minTop = box.top + 8.0;
        final maxTop = box.bottom - ch - 8.0;
        final baseTop = isTopHalf
            ? pos.dy + playerRadius + 22.0 // below chips for top-row players
            : pos.dy - ch - 6.0;
        final top = baseTop.clamp(minTop, maxTop).toDouble();

        final snap = _shownHands[p.id];
        if (snap != null && snap.isNotEmpty) {
          children.addAll([
            Positioned(
                left: left,
                top: top,
                width: cw,
                height: ch,
                child: CardFace(card: snap[0])),
            if (snap.length > 1)
              Positioned(
                  left: left + cw + gap,
                  top: top,
                  width: cw,
                  height: ch,
                  child: CardFace(card: snap[1])),
          ]);
        } else {
          children.addAll([
            Positioned(
                left: left,
                top: top,
                width: cw,
                height: ch,
                child: const CardBack()),
            Positioned(
                left: left + cw + gap,
                top: top,
                width: cw,
                height: ch,
                child: const CardBack()),
          ]);
        }
      }
      return Stack(children: children);
    });
  }
}

class _HeroCardsOverlay extends StatelessWidget {
  const _HeroCardsOverlay(
      {required this.players,
      required this.heroId,
      required this.cache,
      required this.model});
  final List<UiPlayer> players;
  final String heroId;
  final List<pr.Card> cache;
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    final hero = players.firstWhere((p) => p.id == heroId,
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
              isDisconnected: false,
              handDesc: '',
            ));
    if (hero.id.isEmpty) return const SizedBox.shrink();
    // Prefer live hero.hand; fall back to cached hole cards when snapshots omit them (e.g., during showdown).
    final List<pr.Card> cards = hero.hand.isNotEmpty ? hero.hand : cache;
    final bool faceUp = cards.isNotEmpty;
    final bool hint =
        (model.game?.phase == pr.GamePhase.SHOWDOWN) && !model.myCardsShown;
    return HeroCardFlipOverlay(
      cards: cards,
      showFace: faceUp,
      showHint: hint,
      onToggle: () {
        if (model.myCardsShown) {
          model.hideCards();
        } else {
          model.showCards();
        }
      },
    );
  }
}
