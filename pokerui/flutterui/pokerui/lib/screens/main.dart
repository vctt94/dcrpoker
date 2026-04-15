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
import 'package:pokerui/components/dialogs/create_table.dart';

class PokerHomeScreen extends StatefulWidget {
  const PokerHomeScreen({super.key});

  @override
  State<PokerHomeScreen> createState() => _PokerHomeScreenState();
}

class _PokerHomeScreenState extends State<PokerHomeScreen> {
  /// Builds the appropriate poker view based on the current state
  Widget _buildPokerContent(PokerModel model) {
    // Guard against stale state: if not seated, always render browsing
    final effectiveState =
        model.currentTableId == null ? PokerState.browsingTables : model.state;

    // Show appropriate content based on effective state
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
    final hideFooter = context.select<PokerModel, bool>((m) =>
        m.showTableView &&
        m.currentTableId != null &&
        (m.state == PokerState.handInProgress ||
            m.state == PokerState.showdown ||
            m.state == PokerState.gameEnded));

    return SharedLayout(
      title: "Poker - Home",
      hideFooter: hideFooter,
      child: Consumer<PokerModel>(builder: (context, pokerModel, _) {
        final showTableView =
            pokerModel.showTableView && pokerModel.currentTableId != null;
        final inActiveHand = pokerModel.state == PokerState.handInProgress ||
            pokerModel.state == PokerState.showdown;
        final mainContent = showTableView
            ? _buildPokerContent(pokerModel)
            : BrowsingTablesView(model: pokerModel);

        if (showTableView) {
          return mainContent;
        }

        return RefreshIndicator(
          onRefresh: pokerModel.browseTables,
          child: SingleChildScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.only(bottom: 24),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 720),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      // 1) Connection status
                      Padding(
                        padding: const EdgeInsets.only(top: 16.0),
                        child: Card(
                          color: const Color(0xFF1B1E2C),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: Padding(
                            padding: const EdgeInsets.all(16.0),
                            child: Row(
                              children: [
                                Icon(
                                  pokerModel.state != PokerState.idle
                                      ? Icons.check_circle
                                      : Icons.cloud_off,
                                  color: pokerModel.state != PokerState.idle
                                      ? Colors.green
                                      : Colors.red,
                                ),
                                const SizedBox(width: 8),
                                Text(
                                  pokerModel.state != PokerState.idle
                                      ? "Connected"
                                      : "Disconnected",
                                  style: TextStyle(
                                    color: pokerModel.state != PokerState.idle
                                        ? Colors.green
                                        : Colors.red,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),

                      // 2) Current table info
                      Padding(
                        padding: const EdgeInsets.only(top: 16.0),
                        child: Card(
                          color: const Color(0xFF1B1E2C),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: Padding(
                            padding: const EdgeInsets.all(16.0),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  "Current Table",
                                  style: TextStyle(
                                    color: Colors.white,
                                    fontSize: 18,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                                const SizedBox(height: 8),
                                if (pokerModel.currentTableId == null) ...[
                                  Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    crossAxisAlignment:
                                        WrapCrossAlignment.center,
                                    children: [
                                      const Text(
                                        "Not at a table",
                                        style: TextStyle(color: Colors.white),
                                      ),
                                      ElevatedButton.icon(
                                        onPressed: () async {
                                          await CreateTableDialog.open(
                                              context, pokerModel);
                                        },
                                        icon: const Icon(Icons.add),
                                        label: const Text('Create Table'),
                                        style: ElevatedButton.styleFrom(
                                            backgroundColor: Colors.blueGrey),
                                      ),
                                    ],
                                  ),
                                ] else ...[
                                  Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    children: [
                                      Text(
                                        "Table ID: ${pokerModel.currentTableId}",
                                        style: const TextStyle(
                                            color: Colors.white),
                                      ),
                                      Text(
                                        "State: ${pokerModel.state.name} • ${pokerModel.tableRoleLabel}",
                                        style: const TextStyle(
                                            color: Colors.white),
                                      ),
                                    ],
                                  ),
                                  const SizedBox(height: 8),
                                  Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    alignment: WrapAlignment.end,
                                    children: [
                                      ElevatedButton.icon(
                                        onPressed: pokerModel.openTableView,
                                        icon: const Icon(Icons.play_circle),
                                        label: const Text("Open Table"),
                                      ),
                                      if (!inActiveHand)
                                        ElevatedButton(
                                          onPressed: pokerModel.leaveTable,
                                          style: ElevatedButton.styleFrom(
                                            backgroundColor: Colors.redAccent,
                                          ),
                                          child: Text(
                                            pokerModel.isSeated
                                                ? "Leave Table"
                                                : "Stop Watching",
                                          ),
                                        )
                                      else
                                        const Padding(
                                          padding: EdgeInsets.symmetric(
                                              horizontal: 4.0),
                                          child: Text(
                                            "Finish the hand to leave",
                                            style: TextStyle(
                                                color: Colors.white70),
                                          ),
                                        ),
                                    ],
                                  ),
                                ],
                              ],
                            ),
                          ),
                        ),
                      ),

                      // 3) Error message
                      if (pokerModel.errorMessage.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(top: 16.0),
                          child: Card(
                            color: Colors.red.shade800,
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Padding(
                              padding: const EdgeInsets.all(12.0),
                              child: Row(
                                children: [
                                  const Icon(Icons.error, color: Colors.white),
                                  const SizedBox(width: 8),
                                  Expanded(
                                    child: SelectableText(
                                      pokerModel.errorMessage,
                                      style:
                                          const TextStyle(color: Colors.white),
                                    ),
                                  ),
                                  Material(
                                    color: Colors.transparent,
                                    child: InkWell(
                                      onTap: () async {
                                        await Clipboard.setData(ClipboardData(
                                            text: pokerModel.errorMessage));
                                        if (!context.mounted) return;
                                        ScaffoldMessenger.of(context)
                                            .showSnackBar(const SnackBar(
                                                content: Text(
                                                    'Error copied to clipboard')));
                                      },
                                      mouseCursor: SystemMouseCursors.click,
                                      borderRadius: BorderRadius.circular(20),
                                      child: const Padding(
                                        padding: EdgeInsets.all(8.0),
                                        child: Icon(Icons.copy,
                                            color: Colors.white, size: 20),
                                      ),
                                    ),
                                  ),
                                  Material(
                                    color: Colors.transparent,
                                    child: InkWell(
                                      onTap: () => pokerModel.clearError(),
                                      mouseCursor: SystemMouseCursors.click,
                                      borderRadius: BorderRadius.circular(20),
                                      child: const Padding(
                                        padding: EdgeInsets.all(8.0),
                                        child: Icon(Icons.close,
                                            color: Colors.white, size: 20),
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ),

                      // 4) Main content (tables list / game view etc.)
                      Padding(
                        padding: const EdgeInsets.only(top: 12.0),
                        child: mainContent,
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        );
      }),
    );
  }
}
