import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'table.dart';
import 'table_theme.dart';
import 'cards.dart';

Color _seatColorFromId(String id) {
  var hash = 0;
  for (int i = 0; i < id.length; i++) {
    hash = id.codeUnitAt(i) + ((hash << 5) - hash);
  }
  final hue = (hash % 360).abs().toDouble();
  return HSLColor.fromAHSL(1.0, hue, 0.5, 0.4).toColor();
}

String _playerInitials(String name) {
  if (name.isEmpty) return '?';
  final parts = name.trim().split(RegExp(r'\s+'));
  if (parts.length >= 2) {
    return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
  }
  return name.substring(0, name.length >= 2 ? 2 : 1).toUpperCase();
}

/// Whether the game phase warrants showing hole-card slots.
bool _showCardsPhase(pr.GamePhase phase) {
  return phase != pr.GamePhase.WAITING &&
      phase != pr.GamePhase.NEW_HAND_DEALING;
}

/// Widget overlay that positions all player seats around the table.
class PlayerSeatsOverlay extends StatelessWidget {
  const PlayerSeatsOverlay({
    super.key,
    required this.gameState,
    required this.heroId,
    required this.theme,
    this.aspectRatio = 16 / 9,
  });

  final UiGameState gameState;
  final String heroId;
  final PokerThemeConfig theme;
  final double aspectRatio;

  @override
  Widget build(BuildContext context) {
    if (gameState.players.isEmpty) return const SizedBox.shrink();

    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size, aspectRatio: aspectRatio);
      final hasCurrentBet = gameState.currentBet > 0;
      final minSeat = minSeatTopFor(layout.viewport, hasCurrentBet);
      final playerRadius = kPlayerRadius * theme.uiSizeMultiplier;

      final seats = seatPositionsFor(
        gameState.players,
        heroId,
        layout.center,
        layout.ringRadiusX,
        layout.ringRadiusY,
        clampBounds: layout.canvasBounds,
        minSeatTop: minSeat,
        uiSizeMultiplier: theme.uiSizeMultiplier,
      );

      final children = <Widget>[];
      for (final player in gameState.players) {
        final pos = seats[player.id];
        if (pos == null) continue;
        final seatCenterX = pos.dx;
        final seatCenterY = pos.dy + playerRadius;

        children.add(Positioned(
          left: seatCenterX - playerRadius - 30,
          top: seatCenterY - playerRadius - 8,
          child: _PlayerSeatWidget(
            player: player,
            isHero: player.id == heroId,
            isCurrent: player.id == gameState.currentPlayerId && !player.folded,
            gameState: gameState,
            radius: playerRadius,
            uiScale: theme.uiSizeMultiplier,
            cardScale: theme.cardSizeMultiplier,
          ),
        ));
      }

      return Stack(children: children);
    });
  }
}

class _PlayerSeatWidget extends StatelessWidget {
  const _PlayerSeatWidget({
    required this.player,
    required this.isHero,
    required this.isCurrent,
    required this.gameState,
    required this.radius,
    required this.uiScale,
    required this.cardScale,
  });

  final UiPlayer player;
  final bool isHero, isCurrent;
  final UiGameState gameState;
  final double radius, uiScale, cardScale;

  @override
  Widget build(BuildContext context) {
    final displayName = player.name.isNotEmpty ? player.name : 'Player';
    final initials = _playerInitials(displayName);
    final isShowdown = gameState.phase == pr.GamePhase.SHOWDOWN;
    final seatColor = isHero
        ? PokerColors.heroSeat
        : (player.isDisconnected
            ? Colors.red.shade700
            : (player.folded
                ? const Color(0xFF3A3D4A)
                : _seatColorFromId(player.id)));
    final diameter = radius * 2;

    // Determine if we should show cards inside the circle for this opponent.
    final showCards = !isHero &&
        _showCardsPhase(gameState.phase) &&
        !(player.folded && !player.cardsRevealed);
    final showFaceUpCards =
        showCards && (!isShowdown || player.cardsRevealed) && player.hand.isNotEmpty;
    final showCardBacks =
        showCards && !showFaceUpCards && (!player.folded || !isShowdown);

    return SizedBox(
      width: diameter + 60,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Chip balance above avatar (except hero, who shows it inside)
          if (!isHero && player.balance > 0)
            _ChipBadge(balance: player.balance, uiScale: uiScale),
          if (!isHero && player.balance > 0) SizedBox(height: 3 * uiScale),

          // Avatar + badges row
          Row(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              // Avatar circle (cards rendered inside when applicable)
              _AvatarCircle(
                initials: initials,
                color: seatColor,
                radius: radius,
                isCurrent: isCurrent,
                isFolded: player.folded,
                isDisconnected: player.isDisconnected,
                isHero: isHero,
                balance: player.balance,
                uiScale: uiScale,
                turnDeadlineMs:
                    isCurrent ? gameState.turnDeadlineUnixMs : 0,
                timeBankSeconds: gameState.timeBankSeconds,
                isAutoAdvance: isAutoAdvanceAllIn(gameState),
                holeCards: showFaceUpCards ? player.hand : const [],
                showCardBacks: showCardBacks,
                cardScale: cardScale,
              ),
              SizedBox(width: 4 * uiScale),
              // Role badges column
              Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: _buildBadges(),
              ),
            ],
          ),

          SizedBox(height: 3 * uiScale),

          // Player name
          Text(
            displayName,
            style: PokerTypography.playerName.copyWith(
              fontSize: 11 * uiScale,
              color: player.folded
                  ? PokerColors.textMuted
                  : PokerColors.textPrimary,
              decoration:
                  player.folded ? TextDecoration.lineThrough : null,
              decorationColor: PokerColors.textMuted,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  List<Widget> _buildBadges() {
    final badges = <Widget>[];
    if (player.isDealer)
      badges.add(_RoleBadge(
          label: 'D', color: PokerColors.warning, uiScale: uiScale));
    if (player.isSmallBlind)
      badges.add(_RoleBadge(
          label: 'SB', color: PokerColors.primary, uiScale: uiScale));
    if (player.isBigBlind)
      badges.add(_RoleBadge(
          label: 'BB', color: PokerColors.accent, uiScale: uiScale));
    if (player.isAllIn)
      badges.add(_RoleBadge(
          label: 'ALL-IN', color: PokerColors.danger, uiScale: uiScale));
    return badges;
  }
}

// ─────────────────────────────────────────────
// Avatar circle — now renders opponent cards inside when available.
// ─────────────────────────────────────────────

class _AvatarCircle extends StatefulWidget {
  const _AvatarCircle({
    required this.initials,
    required this.color,
    required this.radius,
    required this.isCurrent,
    required this.isFolded,
    required this.isDisconnected,
    required this.isHero,
    required this.balance,
    required this.uiScale,
    required this.turnDeadlineMs,
    required this.timeBankSeconds,
    required this.isAutoAdvance,
    this.holeCards = const [],
    this.showCardBacks = false,
    this.cardScale = 1.0,
  });

  final String initials;
  final Color color;
  final double radius, uiScale, cardScale;
  final bool isCurrent, isFolded, isDisconnected, isHero, isAutoAdvance;
  final bool showCardBacks;
  final int balance;
  final int turnDeadlineMs;
  final int timeBankSeconds;
  final List<pr.Card> holeCards;

  @override
  State<_AvatarCircle> createState() => _AvatarCircleState();
}

class _AvatarCircleState extends State<_AvatarCircle>
    with SingleTickerProviderStateMixin {
  late final AnimationController _timerCtrl;

  @override
  void initState() {
    super.initState();
    _timerCtrl = AnimationController(
        vsync: this, duration: const Duration(seconds: 1));
    if (widget.isCurrent &&
        widget.turnDeadlineMs > 0 &&
        !widget.isAutoAdvance) {
      _timerCtrl.repeat();
    }
  }

  @override
  void didUpdateWidget(covariant _AvatarCircle old) {
    super.didUpdateWidget(old);
    if (widget.isCurrent &&
        widget.turnDeadlineMs > 0 &&
        !widget.isAutoAdvance) {
      if (!_timerCtrl.isAnimating) _timerCtrl.repeat();
    } else {
      _timerCtrl.stop();
    }
  }

  @override
  void dispose() {
    _timerCtrl.dispose();
    super.dispose();
  }

  /// Build the inner content of the circle. If hole cards should be
  /// displayed (face-up or face-down), render two mini-cards side by side.
  /// Otherwise fall back to initials / chip count.
  Widget _buildCenter(double diameter) {
    final hasRevealedCards = widget.holeCards.isNotEmpty;
    final showCards = hasRevealedCards || widget.showCardBacks;

    if (showCards) {
      // Size the cards to fit comfortably inside the circle.
      final cw = diameter * 0.34 * widget.cardScale;
      final ch = cw * 1.4;
      final gap = diameter * 0.04;

      Widget card1;
      Widget card2;

      if (hasRevealedCards) {
        card1 = SizedBox(
            width: cw, height: ch, child: CardFace(card: widget.holeCards[0]));
        card2 = SizedBox(
            width: cw,
            height: ch,
            child: widget.holeCards.length > 1
                ? CardFace(card: widget.holeCards[1])
                : const CardBack());
      } else {
        card1 = SizedBox(width: cw, height: ch, child: const CardBack());
        card2 = SizedBox(width: cw, height: ch, child: const CardBack());
      }

      return Row(
        mainAxisSize: MainAxisSize.min,
        children: [card1, SizedBox(width: gap), card2],
      );
    }

    // Hero shows chip balance; others show initials.
    if (widget.isHero && widget.balance > 0) {
      return Text(
        '${widget.balance}',
        style: PokerTypography.chipCount.copyWith(
          fontSize: 11 * widget.uiScale,
        ),
        textAlign: TextAlign.center,
      );
    }

    return Text(
      widget.initials,
      style: TextStyle(
        color: Colors.white.withOpacity(0.9),
        fontSize: widget.radius * 0.52,
        fontWeight: FontWeight.w800,
        letterSpacing: 0.5,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final diameter = widget.radius * 2;
    final opacity = widget.isFolded ? 0.5 : 1.0;

    return Opacity(
      opacity: opacity,
      child: SizedBox(
        width: diameter + 8,
        height: diameter + 8,
        child: Stack(
          alignment: Alignment.center,
          children: [
            // Timebank ring (animated)
            if (widget.isCurrent &&
                widget.turnDeadlineMs > 0 &&
                !widget.isAutoAdvance)
              AnimatedBuilder(
                animation: _timerCtrl,
                builder: (context, _) {
                  final nowMs = DateTime.now().millisecondsSinceEpoch;
                  final remainingMs =
                      (widget.turnDeadlineMs - nowMs).clamp(0, 1 << 30);
                  final totalDurationMs =
                      ((widget.timeBankSeconds > 0 ? widget.timeBankSeconds : 30) *
                              1000)
                          .toDouble();
                  final fraction =
                      (remainingMs / totalDurationMs).clamp(0.0, 1.0);
                  final ringColor = fraction > 0.5
                      ? PokerColors.accent
                      : (fraction > 0.2
                          ? PokerColors.warning
                          : PokerColors.danger);
                  return SizedBox(
                    width: diameter + 8,
                    height: diameter + 8,
                    child: CircularProgressIndicator(
                      value: fraction,
                      strokeWidth: 3 * widget.uiScale,
                      color: ringColor,
                      backgroundColor: PokerColors.borderSubtle,
                    ),
                  );
                },
              ),

            // Turn glow
            if (widget.isCurrent)
              Container(
                width: diameter + 6,
                height: diameter + 6,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  boxShadow: [
                    BoxShadow(
                      color: PokerColors.turnHighlight.withOpacity(0.35),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ],
                ),
              ),

            // Main avatar circle
            Container(
              width: diameter,
              height: diameter,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: widget.color,
                border: Border.all(
                  color: widget.isDisconnected
                      ? Colors.orangeAccent
                      : (widget.isCurrent
                          ? PokerColors.turnHighlight
                          : PokerColors.borderSubtle.withOpacity(0.5)),
                  width:
                      (widget.isCurrent ? 2.5 : 1.5) * widget.uiScale,
                ),
              ),
              child: ClipOval(
                child: Center(child: _buildCenter(diameter)),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Small helper widgets
// ─────────────────────────────────────────────

class _ChipBadge extends StatelessWidget {
  const _ChipBadge({required this.balance, required this.uiScale});
  final int balance;
  final double uiScale;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: 8 * uiScale,
        vertical: 2 * uiScale,
      ),
      decoration: BoxDecoration(
        color: PokerColors.overlayHeavy,
        borderRadius: BorderRadius.circular(8 * uiScale),
      ),
      child: Text(
        '$balance',
        style:
            PokerTypography.chipCount.copyWith(fontSize: 11 * uiScale),
      ),
    );
  }
}

class _RoleBadge extends StatelessWidget {
  const _RoleBadge(
      {required this.label,
      required this.color,
      required this.uiScale});
  final String label;
  final Color color;
  final double uiScale;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(bottom: 2 * uiScale),
      child: Container(
        padding: EdgeInsets.symmetric(
          horizontal: 6 * uiScale,
          vertical: 2 * uiScale,
        ),
        decoration: BoxDecoration(
          color: color,
          borderRadius: BorderRadius.circular(4 * uiScale),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.3),
              blurRadius: 2,
              offset: const Offset(0, 1),
            ),
          ],
        ),
        child: Text(
          label,
          style: PokerTypography.badgeLabel
              .copyWith(fontSize: 10 * uiScale),
        ),
      ),
    );
  }
}
