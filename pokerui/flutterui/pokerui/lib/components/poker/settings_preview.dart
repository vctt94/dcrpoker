import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/community_placeholders.dart';
import 'package:pokerui/components/poker/game.dart';
import 'package:pokerui/components/poker/player_seat.dart';
import 'package:pokerui/components/poker/pot_display.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/components/poker/table.dart';
import 'package:pokerui/components/poker/table_logo.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

class SettingsPokerPreview extends StatefulWidget {
  const SettingsPokerPreview({
    super.key,
    required this.settings,
  });

  final PokerUiSettings settings;

  @override
  State<SettingsPokerPreview> createState() => _SettingsPokerPreviewState();
}

class _SettingsPokerPreviewState extends State<SettingsPokerPreview> {
  _PreviewDevice? _selectedDevice;

  double _stageHeightFor(
    _PreviewDevice device,
    double width,
    double maxHeight,
  ) {
    final preferred = switch (device) {
      _PreviewDevice.phone => width * 1.1,
      _PreviewDevice.tablet => width * 0.82,
      _PreviewDevice.desktop => width * 0.72,
    };
    final minHeight = switch (device) {
      _PreviewDevice.phone => 420.0,
      _PreviewDevice.tablet => 300.0,
      _PreviewDevice.desktop => 280.0,
    };
    final maxPreferred = switch (device) {
      _PreviewDevice.phone => 540.0,
      _PreviewDevice.tablet => 380.0,
      _PreviewDevice.desktop => (width * 0.44).clamp(360.0, 520.0).toDouble(),
    };
    final constrainedMax = maxHeight.isFinite
        ? math.max(minHeight, maxHeight - 220.0)
        : maxPreferred;
    final upperBound = math.min(maxPreferred, constrainedMax);
    return preferred.clamp(minHeight, upperBound).toDouble();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final availableWidth = constraints.maxWidth.isFinite
            ? constraints.maxWidth
            : MediaQuery.sizeOf(context).width - 32;
        final previewWidth = availableWidth < 320 ? 320.0 : availableWidth;
        final maxHeight = constraints.maxHeight.isFinite
            ? constraints.maxHeight
            : double.infinity;
        final selectedDevice = _selectedDevice ?? _PreviewDevice.desktop;
        final viewportSize = selectedDevice.viewportSize;
        final stageHeight =
            _stageHeightFor(selectedDevice, previewWidth, maxHeight);
        final uiSpec = PokerUiSpec.fromSettings(
          widget.settings,
          viewportSize: viewportSize,
        );
        final theme = PokerThemeConfig.fromSpec(uiSpec);
        final scene = PokerSceneLayout.resolve(viewportSize);
        final layout = TableLayout.fromScene(scene);
        final gameState = _previewGameState();

        return Container(
          key: const Key('settings-poker-preview'),
          width: previewWidth,
          padding: const EdgeInsets.all(PokerSpacing.lg),
          decoration: BoxDecoration(
            color: PokerColors.surfaceDim,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: PokerColors.borderSubtle.withOpacity(0.9),
            ),
          ),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Live Preview',
                        style: PokerTypography.titleMedium.copyWith(
                          color: PokerColors.textPrimary,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    const SizedBox(width: PokerSpacing.sm),
                    Text(
                      '${widget.settings.cardScale.key.toUpperCase()} cards • ${widget.settings.densityScale.key.toUpperCase()} UI',
                      style: PokerTypography.labelSmall.copyWith(
                        color: PokerColors.textSecondary,
                      ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
                const SizedBox(height: PokerSpacing.xs),
                Text(
                  'Choose a device to preview the real table layout.',
                  style: PokerTypography.bodySmall.copyWith(
                    color: PokerColors.textSecondary,
                  ),
                ),
                const SizedBox(height: PokerSpacing.md),
                Wrap(
                  spacing: PokerSpacing.sm,
                  runSpacing: PokerSpacing.sm,
                  children: [
                    for (final device in _PreviewDevice.values)
                      ChoiceChip(
                        key: Key('settings-preview-device-${device.name}'),
                        selected: device == selectedDevice,
                        label: Text(device.label),
                        selectedColor: PokerColors.primary.withOpacity(0.22),
                        backgroundColor: PokerColors.surface,
                        labelStyle: PokerTypography.labelSmall.copyWith(
                          color: device == selectedDevice
                              ? PokerColors.textPrimary
                              : PokerColors.textSecondary,
                        ),
                        side: BorderSide(
                          color: device == selectedDevice
                              ? PokerColors.primary.withOpacity(0.75)
                              : PokerColors.borderSubtle,
                        ),
                        onSelected: (_) {
                          setState(() => _selectedDevice = device);
                        },
                      ),
                  ],
                ),
                const SizedBox(height: PokerSpacing.sm),
                Text(
                  '${selectedDevice.label} • ${viewportSize.width.toInt()} x ${viewportSize.height.toInt()} • ${selectedDevice.layoutLabel}',
                  key: const Key('settings-preview-device-summary'),
                  style: PokerTypography.labelSmall.copyWith(
                    color: PokerColors.textSecondary,
                  ),
                ),
                const SizedBox(height: PokerSpacing.md),
                Container(
                  key: const Key('settings-preview-stage'),
                  height: stageHeight,
                  padding: const EdgeInsets.all(PokerSpacing.md),
                  decoration: BoxDecoration(
                    color: PokerColors.surface,
                    borderRadius: BorderRadius.circular(18),
                    border: Border.all(color: PokerColors.borderSubtle),
                  ),
                  child: Center(
                    child: FittedBox(
                      fit: BoxFit.contain,
                      child: _PreviewViewportFrame(
                        device: selectedDevice,
                        viewportSize: viewportSize,
                        uiSpec: uiSpec,
                        theme: theme,
                        scene: scene,
                        layout: layout,
                        gameState: gameState,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _PreviewTableThemePainter extends CustomPainter {
  const _PreviewTableThemePainter({
    required this.theme,
    required this.layout,
  });

  final PokerThemeConfig theme;
  final TableLayout layout;

  @override
  void paint(Canvas canvas, Size size) {
    drawPokerTable(
      canvas,
      layout.center.dx,
      layout.center.dy,
      layout.tableRadiusX,
      layout.tableRadiusY,
      theme.tableTheme,
    );
  }

  @override
  bool shouldRepaint(covariant _PreviewTableThemePainter oldDelegate) {
    return oldDelegate.theme != theme || oldDelegate.layout != layout;
  }
}

enum _PreviewDevice {
  phone,
  tablet,
  desktop,
}

extension on _PreviewDevice {
  String get label => switch (this) {
        _PreviewDevice.phone => 'Phone',
        _PreviewDevice.tablet => 'Tablet',
        _PreviewDevice.desktop => 'Desktop',
      };

  String get layoutLabel => switch (this) {
        _PreviewDevice.phone => 'Compact portrait',
        _PreviewDevice.tablet => 'Standard table',
        _PreviewDevice.desktop => 'Wide desktop',
      };

  Size get viewportSize => switch (this) {
        _PreviewDevice.phone => const Size(393, 852),
        _PreviewDevice.tablet => const Size(1024, 768),
        _PreviewDevice.desktop => const Size(1440, 900),
      };
}

class _PreviewViewportFrame extends StatelessWidget {
  const _PreviewViewportFrame({
    required this.device,
    required this.viewportSize,
    required this.uiSpec,
    required this.theme,
    required this.scene,
    required this.layout,
    required this.gameState,
  });

  final _PreviewDevice device;
  final Size viewportSize;
  final PokerUiSpec uiSpec;
  final PokerThemeConfig theme;
  final PokerSceneLayout scene;
  final TableLayout layout;
  final UiGameState gameState;

  @override
  Widget build(BuildContext context) {
    final heroCards = gameState.players.first.hand;

    return Container(
      key: Key('settings-preview-viewport-${device.name}'),
      width: viewportSize.width,
      height: viewportSize.height,
      decoration: BoxDecoration(
        color: PokerColors.screenBg,
        borderRadius: BorderRadius.circular(
          device == _PreviewDevice.phone ? 30 : 22,
        ),
        border: Border.all(
          color: PokerColors.borderSubtle.withOpacity(0.95),
        ),
        boxShadow: const [
          BoxShadow(
            color: Color(0x44000000),
            blurRadius: 24,
            offset: Offset(0, 16),
          ),
        ],
      ),
      clipBehavior: Clip.antiAlias,
      child: Stack(
        fit: StackFit.expand,
        children: [
          PokerTableBackground(layout: layout),
          CustomPaint(
            painter: _PreviewTableThemePainter(
              theme: theme,
              layout: layout,
            ),
          ),
          if (theme.showTableLogo)
            TableLogoOverlay(
              layout: layout,
              logoPosition: theme.logoPosition,
              uiSizeMultiplier: theme.uiSizeMultiplier,
            ),
          CommunityCardSlots(
            layout: layout,
            cards: gameState.communityCards,
            theme: theme,
          ),
          PlayerSeatsOverlay(
            layout: layout,
            gameState: gameState,
            heroId: 'hero',
            theme: theme,
            heroCardsCache: heroCards,
            showHeroCardsInSeat: false,
          ),
          PotDisplay(
            layout: layout,
            pot: gameState.pot,
            theme: theme,
          ),
          Positioned.fromRect(
            rect: scene.heroDockRect,
            child: scene.mode == PokerLayoutMode.compactPortrait
                ? _PreviewMobileHeroDock(
                    uiSpec: uiSpec,
                    heroCards: heroCards,
                  )
                : _PreviewHeroDock(
                    uiSpec: uiSpec,
                    heroCards: heroCards,
                  ),
          ),
        ],
      ),
    );
  }
}

class _PreviewHeroDock extends StatelessWidget {
  const _PreviewHeroDock({
    required this.uiSpec,
    required this.heroCards,
  });

  final PokerUiSpec uiSpec;
  final List<pr.Card> heroCards;

  @override
  Widget build(BuildContext context) {
    final heroCardSize = uiSpec.heroDockCardSize;
    final buttonHeight = (44.0 * uiSpec.uiSizeMultiplier).clamp(34.0, 58.0);
    final buttonFontSize =
        (11.0 * uiSpec.textScale).clamp(9.0, 16.0).toDouble();
    final gap = (heroCardSize.width * 0.14).clamp(4.0, 8.0);

    Widget heroCard(int index) {
      return SizedBox(
        key: Key('settings-preview-hero-card-$index'),
        width: heroCardSize.width,
        height: heroCardSize.height,
        child: CardFace(card: heroCards[index]),
      );
    }

    return Container(
      padding: EdgeInsets.fromLTRB(
        12 * uiSpec.spacingScale,
        10 * uiSpec.spacingScale,
        12 * uiSpec.spacingScale,
        12 * uiSpec.spacingScale,
      ),
      color: PokerColors.screenBg.withOpacity(0.94),
      child: Row(
        children: [
          heroCard(0),
          SizedBox(width: gap),
          heroCard(1),
          SizedBox(width: 14 * uiSpec.spacingScale),
          Expanded(
            child: Row(
              children: [
                _PreviewActionButton(
                  label: 'Fold',
                  background: PokerColors.dangerDark,
                  foreground: Colors.white,
                  height: buttonHeight,
                  fontSize: buttonFontSize,
                ),
                SizedBox(width: 8 * uiSpec.spacingScale),
                _PreviewActionButton(
                  label: 'Call',
                  background: PokerColors.surfaceBright,
                  foreground: Colors.white,
                  height: buttonHeight,
                  fontSize: buttonFontSize,
                ),
                SizedBox(width: 8 * uiSpec.spacingScale),
                _PreviewActionButton(
                  label: 'Raise',
                  background: PokerColors.primary,
                  foreground: Colors.white,
                  height: buttonHeight,
                  fontSize: buttonFontSize,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _PreviewMobileHeroDock extends StatelessWidget {
  const _PreviewMobileHeroDock({
    required this.uiSpec,
    required this.heroCards,
  });

  final PokerUiSpec uiSpec;
  final List<pr.Card> heroCards;

  @override
  Widget build(BuildContext context) {
    final heroCardSize = uiSpec.heroDockCardSize;
    final buttonHeight = (46.0 * uiSpec.uiSizeMultiplier).clamp(36.0, 60.0);
    final buttonFontSize =
        (11.0 * uiSpec.textScale).clamp(9.0, 16.0).toDouble();
    final gap = (heroCardSize.width * 0.14).clamp(4.0, 8.0);

    Widget heroCard(int index) {
      return SizedBox(
        key: Key('settings-preview-hero-card-$index'),
        width: heroCardSize.width,
        height: heroCardSize.height,
        child: CardFace(card: heroCards[index]),
      );
    }

    return Container(
      color: PokerColors.screenBg.withOpacity(0.96),
      padding: EdgeInsets.fromLTRB(
        10 * uiSpec.spacingScale,
        10 * uiSpec.spacingScale,
        10 * uiSpec.spacingScale,
        10 * uiSpec.spacingScale,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              heroCard(0),
              SizedBox(width: gap),
              heroCard(1),
              const Spacer(),
              Container(
                padding: EdgeInsets.symmetric(
                  horizontal: 10 * uiSpec.spacingScale,
                  vertical: 6 * uiSpec.spacingScale,
                ),
                decoration: BoxDecoration(
                  color: PokerColors.overlayLight,
                  borderRadius: BorderRadius.circular(999),
                  border: Border.all(color: PokerColors.borderSubtle),
                ),
                child: Text(
                  'Your turn',
                  style: PokerTypography.labelSmall.copyWith(
                    color: PokerColors.textPrimary,
                    fontSize:
                        (11.0 * uiSpec.textScale).clamp(9.0, 14.0).toDouble(),
                  ),
                ),
              ),
            ],
          ),
          SizedBox(height: 10 * uiSpec.spacingScale),
          Row(
            children: [
              _PreviewActionButton(
                label: 'Fold',
                background: PokerColors.dangerDark,
                foreground: Colors.white,
                height: buttonHeight,
                fontSize: buttonFontSize,
              ),
              SizedBox(width: 8 * uiSpec.spacingScale),
              _PreviewActionButton(
                label: 'Call',
                background: PokerColors.surfaceBright,
                foreground: Colors.white,
                height: buttonHeight,
                fontSize: buttonFontSize,
              ),
              SizedBox(width: 8 * uiSpec.spacingScale),
              _PreviewActionButton(
                label: 'Raise',
                background: PokerColors.primary,
                foreground: Colors.white,
                height: buttonHeight,
                fontSize: buttonFontSize,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _PreviewActionButton extends StatelessWidget {
  const _PreviewActionButton({
    required this.label,
    required this.background,
    required this.foreground,
    required this.height,
    required this.fontSize,
  });

  final String label;
  final Color background;
  final Color foreground;
  final double height;
  final double fontSize;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        key: Key('settings-preview-action-${label.toLowerCase()}'),
        height: height,
        decoration: BoxDecoration(
          color: background,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: PokerColors.borderSubtle),
        ),
        alignment: Alignment.center,
        child: Text(
          label,
          style: PokerTypography.labelSmall.copyWith(
            fontSize: fontSize,
            color: foreground,
          ),
        ),
      ),
    );
  }
}

UiGameState _previewGameState() {
  return UiGameState(
    tableId: 'preview',
    phase: pr.GamePhase.FLOP,
    phaseName: 'Flop',
    players: [
      UiPlayer(
        id: 'hero',
        name: 'Hero',
        balance: 1220,
        hand: [
          _card('A', 'spades'),
          _card('K', 'hearts'),
        ],
        currentBet: 20,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: true,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
      ),
      UiPlayer(
        id: 'left',
        name: 'Mila',
        balance: 940,
        hand: [
          _card('Q', 'clubs'),
          _card('10', 'clubs'),
        ],
        currentBet: 20,
        folded: false,
        isTurn: false,
        isAllIn: false,
        isDealer: true,
        isSmallBlind: true,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        cardsRevealed: true,
      ),
      UiPlayer(
        id: 'right',
        name: 'Rex',
        balance: 1560,
        hand: [
          _card('8', 'diamonds'),
          _card('8', 'spades'),
        ],
        currentBet: 40,
        folded: false,
        isTurn: true,
        isAllIn: false,
        isDealer: false,
        isSmallBlind: false,
        isBigBlind: false,
        isReady: true,
        isDisconnected: false,
        handDesc: '',
        cardsRevealed: true,
      ),
    ],
    communityCards: [
      _card('A', 'clubs'),
      _card('J', 'hearts'),
      _card('5', 'spades'),
    ],
    pot: 180,
    currentBet: 40,
    currentPlayerId: 'right',
    minRaise: 40,
    maxRaise: 400,
    smallBlind: 10,
    bigBlind: 20,
    gameStarted: true,
    playersRequired: 2,
    playersJoined: 3,
    timeBankSeconds: 20,
    turnDeadlineUnixMs: 0,
  );
}

pr.Card _card(String value, String suit) {
  return pr.Card()
    ..value = value
    ..suit = suit;
}
