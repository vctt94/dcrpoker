import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';
import 'scene_layout.dart';
import 'table.dart';
import 'table_theme.dart';
import 'community_placeholders.dart';
import 'disconnected_badges.dart';
import 'table_logo.dart';
import 'pot_display.dart';
import 'player_seat.dart';

class PokerTableBackground extends StatelessWidget {
  const PokerTableBackground({
    super.key,
    required this.layout,
    this.aspectRatio = 16 / 9,
  });
  final TableLayout layout;
  final double aspectRatio;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: CustomPaint(
        painter: _TableBackgroundPainter(
          layout: layout,
        ),
        size: Size.infinite,
      ),
    );
  }
}

class _TableBackgroundPainter extends CustomPainter {
  _TableBackgroundPainter({
    required this.layout,
  });
  final TableLayout layout;

  @override
  void paint(Canvas canvas, Size size) {
    final centerX = layout.center.dx;
    final centerY = layout.center.dy;
    final tableRadiusX = layout.tableRadiusX;
    final tableRadiusY = layout.tableRadiusY;

    final tableRect = Rect.fromCenter(
      center: Offset(centerX, centerY),
      width: tableRadiusX * 2,
      height: tableRadiusY * 2,
    );

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

    final commCenterY = communityCardsCenterY(layout);
    final spotlightRect = Rect.fromCenter(
      center: Offset(centerX, commCenterY + tableRadiusY * 0.08),
      width: tableRadiusX * 1.1,
      height: tableRadiusY * 0.7,
    );
    final spotlightGradient = RadialGradient(
      center: Alignment.center,
      radius: 0.75,
      colors: [
        Colors.white.withOpacity(0.045),
        Colors.white.withOpacity(0.0),
      ],
      stops: const [0.0, 1.0],
    );
    canvas.drawOval(
        spotlightRect,
        Paint()
          ..shader = spotlightGradient.createShader(spotlightRect)
          ..style = PaintingStyle.fill);

    final borderPaint = Paint()
      ..color = const Color(0xFF8B4513)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8;
    canvas.drawOval(tableRect, borderPaint);

    canvas.drawOval(
      Rect.fromCenter(
        center: Offset(centerX, centerY + 5),
        width: tableRadiusX * 2,
        height: tableRadiusY * 2,
      ),
      Paint()
        ..color = Colors.black.withOpacity(0.3)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 15),
    );
  }

  @override
  bool shouldRepaint(covariant _TableBackgroundPainter old) =>
      old.layout != layout;
}

class PokerGame {
  final PokerModel pokerModel;
  final String playerId;
  final PokerThemeConfig theme;

  PokerGame(this.playerId, this.pokerModel, {required this.theme});

  Widget buildWidget(UiGameState gameState, FocusNode focusNode,
      {PokerSceneLayout? scene,
      double aspectRatio = 16 / 9,
      VoidCallback? onReadyHotkey,
      bool? showHeroCardsOverlay,
      bool? showHeroSeatCards}) {
    final renderHeroSeatCards =
        showHeroSeatCards ?? showHeroCardsOverlay ?? true;
    Widget buildTableScene(PokerSceneLayout resolvedScene) {
      final tableLayout = TableLayout.fromScene(resolvedScene);
      final liveStreetBets = gameState.players.fold<int>(
        0,
        (sum, player) => sum + player.currentBet,
      );
      final collectedPot = (gameState.pot - liveStreetBets).clamp(0, 1 << 30);

      return SizedBox.expand(
        child: RepaintBoundary(
          child: Stack(
            fit: StackFit.expand,
            children: [
              PokerTableBackground(
                layout: tableLayout,
                aspectRatio: aspectRatio,
              ),
              CustomPaint(
                painter: _TableThemePainter(
                  theme,
                  layout: tableLayout,
                ),
                isComplex: false,
                willChange: false,
              ),
              if (theme.showTableLogo)
                TableLogoOverlay(
                  layout: tableLayout,
                  logoPosition: theme.logoPosition,
                  uiSizeMultiplier: theme.uiSizeMultiplier,
                ),
              CommunityCardSlots(
                layout: tableLayout,
                cards: gameState.communityCards,
                theme: theme,
              ),
              PlayerSeatsOverlay(
                layout: tableLayout,
                gameState: gameState,
                heroId: playerId,
                theme: theme,
                heroCardsCache: pokerModel.myHoleCardsCache,
                showHeroCardsInSeat: renderHeroSeatCards,
                aspectRatio: aspectRatio,
              ),
              DisconnectedBadgesOverlay(
                layout: tableLayout,
                theme: theme,
                players: gameState.players,
                heroId: playerId,
                hasCurrentBet: gameState.currentBet > 0,
              ),
              if (collectedPot > 0)
                PotDisplay(
                  layout: tableLayout,
                  pot: collectedPot,
                  theme: theme,
                  settleFxMs: pokerModel.lastShowdownFxMs,
                  hideForPayout: gameState.phase == pr.GamePhase.SHOWDOWN &&
                      pokerModel.lastWinners.isNotEmpty,
                ),
            ],
          ),
        ),
      );
    }

    return GestureDetector(
      onTap: () => focusNode.requestFocus(),
      child: Focus(
        child: KeyboardListener(
          focusNode: focusNode,
          autofocus: true,
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
          child: scene != null
              ? buildTableScene(scene)
              : LayoutBuilder(
                  builder: (context, constraints) => buildTableScene(
                    PokerSceneLayout.resolve(
                      constraints.biggest,
                      safePadding: MediaQuery.paddingOf(context),
                    ),
                  ),
                ),
        ),
      ),
    );
  }

  Widget buildReadyToPlayOverlay(
      BuildContext context,
      bool isReadyToPlay,
      bool countdownStarted,
      String countdownMessage,
      Function onReadyPressed,
      UiGameState gameState) {
    if (countdownStarted) {
      return Center(
        child: Container(
          padding: const EdgeInsets.all(PokerSpacing.xl),
          decoration: BoxDecoration(
            color: PokerColors.surface.withAlpha(230),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: PokerColors.primary.withOpacity(0.3)),
            boxShadow: [
              BoxShadow(
                color: PokerColors.primary.withOpacity(0.15),
                blurRadius: 20,
                spreadRadius: 4,
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.casino, size: 48, color: PokerColors.primary),
              const SizedBox(height: PokerSpacing.lg),
              Text(countdownMessage, style: PokerTypography.displayLarge),
            ],
          ),
        ),
      );
    }

    if (!isReadyToPlay) {
      return Container(
        color: PokerColors.overlayHeavy,
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.style,
                  size: 64, color: PokerColors.primary.withOpacity(0.8)),
              const SizedBox(height: PokerSpacing.xxl),
              Text("Ready to play?",
                  style: PokerTypography.headlineLarge.copyWith(
                    color: PokerColors.primary,
                    fontSize: 28,
                  )),
              const SizedBox(height: PokerSpacing.xxl),
              ElevatedButton(
                onPressed: () => onReadyPressed(),
                style: ElevatedButton.styleFrom(
                  backgroundColor: PokerColors.primary,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 48, vertical: 16),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(28),
                  ),
                ),
                child: Text("I'm Ready!",
                    style: PokerTypography.titleMedium
                        .copyWith(color: Colors.white)),
              ),
              const SizedBox(height: PokerSpacing.xxxl),
              Container(
                padding: const EdgeInsets.all(PokerSpacing.lg),
                decoration: BoxDecoration(
                  color: PokerColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: PokerColors.borderSubtle),
                ),
                child: Column(
                  children: [
                    Text("CONTROLS",
                        style: PokerTypography.labelSmall.copyWith(
                          color: PokerColors.primary,
                          letterSpacing: 1.2,
                        )),
                    const SizedBox(height: PokerSpacing.md),
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

    return Center(
      child: Container(
        padding: const EdgeInsets.all(PokerSpacing.xl),
        decoration: BoxDecoration(
          color: PokerColors.surface.withAlpha(230),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: PokerColors.primary.withOpacity(0.3)),
          boxShadow: [
            BoxShadow(
              color: PokerColors.primary.withOpacity(0.15),
              blurRadius: 20,
              spreadRadius: 4,
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.casino, size: 48, color: PokerColors.primary),
            const SizedBox(height: PokerSpacing.lg),
            Text("Waiting for players...", style: PokerTypography.titleLarge),
            const SizedBox(height: PokerSpacing.lg),
            SizedBox(
              width: 36,
              height: 36,
              child: CircularProgressIndicator(
                color: PokerColors.primary,
                backgroundColor: PokerColors.borderSubtle,
                strokeWidth: 3,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _controlKey(String key, String action) {
    return Column(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: PokerColors.surfaceBright,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: PokerColors.borderMedium),
          ),
          child: Center(
            child: Text(key, style: PokerTypography.titleMedium),
          ),
        ),
        const SizedBox(height: 4),
        Text(action, style: PokerTypography.bodySmall),
      ],
    );
  }

  Future<void> handleInput(String playerId, String data) async {
    await _sendKeyInput(data);
  }

  Future<void> _sendKeyInput(String data) async {
    try {
      if (!pokerModel.canAct) return;
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
          final g = pokerModel.game;
          final currentBet = g?.currentBet ?? 0;
          final tid = pokerModel.currentTableId;
          final table = tid == null
              ? null
              : pokerModel.tables
                  .where((t) => t.id == tid)
                  .cast<UiTable?>()
                  .firstWhere((t) => t != null, orElse: () => null);
          final bb = g?.bigBlind ?? table?.bigBlind ?? 0;
          final threeBB = bb * 3;
          final targetTotal = currentBet > threeBB ? (currentBet * 3) : threeBB;
          if (targetTotal > 0) await pokerModel.makeBet(targetTotal);
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

/// Draws the themed table border over the default background.
class _TableThemePainter extends CustomPainter {
  final PokerThemeConfig theme;
  final TableLayout layout;
  _TableThemePainter(
    this.theme, {
    required this.layout,
  });

  @override
  void paint(Canvas canvas, Size size) {
    drawPokerTable(canvas, layout.center.dx, layout.center.dy,
        layout.tableRadiusX, layout.tableRadiusY, theme.tableTheme);
  }

  @override
  bool shouldRepaint(covariant _TableThemePainter old) =>
      old.theme != theme || old.layout != layout;
}
