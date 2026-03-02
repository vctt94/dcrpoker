import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:pokerui/models/poker.dart';
import 'table.dart';
import 'table_theme.dart';
import 'cards.dart';
import 'community_placeholders.dart';
import 'disconnected_badges.dart';
import 'table_logo.dart';
import 'pot_display.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/helper.dart';

class PokerTableBackground extends StatelessWidget {
  const PokerTableBackground({super.key, this.aspectRatio = 16 / 9});
  final double aspectRatio;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: CustomPaint(
        painter: _TableBackgroundPainter(aspectRatio: aspectRatio),
        size: Size.infinite,
      ),
    );
  }
}

class _TableBackgroundPainter extends CustomPainter {
  _TableBackgroundPainter({this.aspectRatio = 16 / 9});
  final double aspectRatio;

  @override
  void paint(Canvas canvas, Size size) {
    final layout = resolveTableLayout(size, aspectRatio: aspectRatio);
    final centerX = layout.center.dx;
    final centerY = layout.center.dy;
    final tableRadiusX = layout.tableRadiusX;
    final tableRadiusY = layout.tableRadiusY;

    final tableRect = Rect.fromCenter(
      center: Offset(centerX, centerY),
      width: tableRadiusX * 2,
      height: tableRadiusY * 2,
    );

    // Radial gradient for felt depth (lighter center, darker edges)
    final feltGradient = RadialGradient(
      center: Alignment.center,
      radius: 0.9,
      colors: [
        const Color(0xFF11654B),
        const Color(0xFF0D4F3C),
        const Color(0xFF093D2E),
      ],
      stops: const [0.0, 0.6, 1.0],
    );
    final tablePaint = Paint()
      ..shader = feltGradient.createShader(tableRect)
      ..style = PaintingStyle.fill;
    canvas.drawOval(tableRect, tablePaint);

    // Table border
    final borderPaint = Paint()
      ..color = const Color(0xFF8B4513)
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
  bool shouldRepaint(covariant _TableBackgroundPainter old) =>
      old.aspectRatio != aspectRatio;
}

class PokerGame {
  final PokerModel pokerModel;
  final String playerId;
  final PokerThemeConfig theme;
  final RenderLoop _loop = RenderLoop();

  PokerGame(this.playerId, this.pokerModel, {required this.theme});

  Widget buildWidget(UiGameState gameState, FocusNode focusNode,
      {VoidCallback? onReadyHotkey,
      double aspectRatio = 16 / 9,
      bool showHeroCardsOverlay = true}) {
    // Start/stop lightweight repaint loop only while an authoritative deadline
    // is active and the hand isn't auto-advancing through all-in streets.
    final hasDeadline = gameState.turnDeadlineUnixMs > 0;
    final autoAdvancing = isAutoAdvanceAllIn(gameState);
    if (hasDeadline && !autoAdvancing) {
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
                    aspectRatio: aspectRatio,
                    child: RepaintBoundary(
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          PokerTableBackground(aspectRatio: aspectRatio),
                          CustomPaint(
                            painter: PokerPainter(gameState, playerId, theme,
                                repaint: _loop, aspectRatio: aspectRatio),
                            isComplex: true,
                            willChange: true,
                          ),
                          if (theme.showTableLogo)
                            TableLogoOverlay(
                              logoPosition: theme.logoPosition,
                              uiSizeMultiplier: theme.uiSizeMultiplier,
                            ),
                          CommunityCardSlots(
                              cards: gameState.communityCards,
                              aspectRatio: aspectRatio),
                          if (showHeroCardsOverlay &&
                              gameState.phase != pr.GamePhase.WAITING)
                            _HeroCardsOverlay(
                              players: gameState.players,
                              heroId: playerId,
                              cache: pokerModel.myHoleCardsCache,
                              gamePhase: gameState.phase,
                              isShowing: pokerModel.me?.cardsRevealed ?? false,
                              onToggle: () {
                                if (pokerModel.me?.cardsRevealed ?? false) {
                                  pokerModel.hideCards();
                                } else {
                                  pokerModel.showCards();
                                }
                              },
                            ),
                          DisconnectedBadgesOverlay(
                            players: gameState.players,
                            heroId: playerId,
                            hasCurrentBet: gameState.currentBet > 0,
                          ),
                          if (gameState.pot > 0)
                            PotDisplay(
                              pot: gameState.pot,
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
  final double aspectRatio;

  PokerPainter(this.gameState, this.currentPlayerId, this.theme,
      {Listenable? repaint, this.aspectRatio = 16 / 9})
      : showdownStartMs = DateTime.now().millisecondsSinceEpoch,
        minSeatTop = 0,
        super(repaint: repaint);

  @override
  void paint(Canvas canvas, Size size) {
    final layout = resolveTableLayout(size, aspectRatio: aspectRatio);
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
      required this.gamePhase,
      required this.isShowing,
      required this.onToggle});
  final List<UiPlayer> players;
  final String heroId;
  final List<pr.Card> cache;
  final pr.GamePhase gamePhase;
  final bool isShowing;
  final VoidCallback onToggle;

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
    return HeroCardFlipOverlay(
      cards: cards,
      showFace: faceUp,
      onToggle: cards.isNotEmpty ? onToggle : null,
      toggleShown: isShowing,
    );
  }
}
