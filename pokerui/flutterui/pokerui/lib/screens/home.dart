import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/views/idle.dart';
import 'package:pokerui/components/views/browsing_tables.dart';
import 'package:pokerui/components/views/in_lobby.dart';
import 'package:pokerui/components/views/table_session_view.dart';
import 'package:pokerui/components/views/game_ended.dart';
import 'package:pokerui/components/views/tournament_over.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class PokerHomeScreen extends StatefulWidget {
  const PokerHomeScreen({super.key});

  @override
  State<PokerHomeScreen> createState() => _PokerHomeScreenState();
}

class _PokerHomeScreenState extends State<PokerHomeScreen> {
  Widget _buildPokerContent(PokerModel model) {
    final effectiveState =
        model.currentTableId == null ? PokerState.browsingTables : model.state;

    switch (effectiveState) {
      case PokerState.idle:
        return IdleView(model: model);
      case PokerState.browsingTables:
        return BrowsingTablesView(model: model);
      case PokerState.inLobby:
        return InLobbyView(model: model);
      case PokerState.handInProgress:
      case PokerState.showdown:
        return TableSessionView(model: model);
      case PokerState.gameEnded:
        return GameEndedView(model: model);
      case PokerState.tournamentOver:
        return TournamentOverView(model: model);
    }
  }

  @override
  Widget build(BuildContext context) {
    final gameStarted = context.select<PokerModel, bool>((m) =>
        m.state == PokerState.handInProgress ||
        m.state == PokerState.showdown ||
        m.state == PokerState.gameEnded);

    // Full-bleed game shell during active gameplay
    if (gameStarted) {
      return GameShell(
        child: Consumer<PokerModel>(
          builder: (_, model, __) => _buildPokerContent(model),
        ),
      );
    }

    // Lobby shell for non-gameplay screens
    return SharedLayout(
      title: "Poker",
      child: Consumer<PokerModel>(builder: (context, pokerModel, _) {
        return RefreshIndicator(
          onRefresh: pokerModel.browseTables,
          child: SingleChildScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.only(bottom: PokerSpacing.xl),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                // Error banner
                if (pokerModel.errorMessage.isNotEmpty)
                  _ErrorBanner(
                    message: pokerModel.errorMessage,
                    onCopy: () async {
                      await Clipboard.setData(
                          ClipboardData(text: pokerModel.errorMessage));
                      if (!context.mounted) return;
                      ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Copied')));
                    },
                    onDismiss: pokerModel.clearError,
                  ),

                // Success banner
                if (pokerModel.successMessage.isNotEmpty)
                  _SuccessBanner(message: pokerModel.successMessage),

                // Main content
                Padding(
                  padding: const EdgeInsets.only(top: PokerSpacing.sm),
                  child: _buildPokerContent(pokerModel),
                ),
              ],
            ),
          ),
        );
      }),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({
    required this.message,
    required this.onCopy,
    required this.onDismiss,
  });
  final String message;
  final VoidCallback onCopy, onDismiss;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.fromLTRB(
        PokerSpacing.lg, PokerSpacing.md, PokerSpacing.lg, 0,
      ),
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.danger.withOpacity(0.12),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: PokerColors.danger.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: PokerColors.danger, size: 20),
          const SizedBox(width: PokerSpacing.sm),
          Expanded(
            child: SelectableText(
              message,
              style: PokerTypography.bodySmall.copyWith(color: PokerColors.danger),
            ),
          ),
          IconButton(
            icon: Icon(Icons.copy, color: PokerColors.danger, size: 16),
            onPressed: onCopy,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
          ),
          const SizedBox(width: PokerSpacing.xs),
          IconButton(
            icon: Icon(Icons.close, color: PokerColors.danger, size: 16),
            onPressed: onDismiss,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
          ),
        ],
      ),
    );
  }
}

class _SuccessBanner extends StatelessWidget {
  const _SuccessBanner({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.fromLTRB(
        PokerSpacing.lg, PokerSpacing.md, PokerSpacing.lg, 0,
      ),
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.success.withOpacity(0.12),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: PokerColors.success.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          Icon(Icons.check_circle_outline, color: PokerColors.success, size: 20),
          const SizedBox(width: PokerSpacing.sm),
          Expanded(
            child: Text(
              message,
              style: PokerTypography.bodySmall.copyWith(color: PokerColors.success),
            ),
          ),
        ],
      ),
    );
  }
}
