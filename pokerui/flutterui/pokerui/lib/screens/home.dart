import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/components/home/main_content.dart' as main_content; // your Poker MainContent
import 'package:pokerui/models/poker.dart'; // your PokerModel

class PokerHomeScreen extends StatefulWidget {
  const PokerHomeScreen({super.key});

  @override
  State<PokerHomeScreen> createState() => _PokerHomeScreenState();
}

class _PokerHomeScreenState extends State<PokerHomeScreen> {
  @override
  Widget build(BuildContext context) {
    // Only rebuild this widget when the game state changes
    final gameStarted =
        context.select<PokerModel, bool>((m) => m.state == PokerState.handInProgress || m.state == PokerState.showdown);

    return SharedLayout(
      title: "Poker - Home",
      child: gameStarted
          ? Padding(
              padding: const EdgeInsets.only(top: 12.0),
              child: Consumer<PokerModel>(
                builder: (_, model, __) => main_content.PokerMainContent(model: model),
              ),
            )
          : Consumer<PokerModel>(builder: (context, pokerModel, _) {
              return RefreshIndicator(
                onRefresh: pokerModel.refreshTables,
                child: SingleChildScrollView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.only(bottom: 24),
                  child: Column(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    // 1) Top area: balance and connection status
                    Center(
                      child: Container(
                        width: MediaQuery.of(context).size.width * 0.85,
                        margin: const EdgeInsets.only(top: 16.0),
                        child: Card(
                          color: const Color(0xFF1B1E2C),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: Padding(
                            padding: const EdgeInsets.all(16.0),
                            child: Row(
                              children: [
                                const Icon(Icons.account_balance_wallet, color: Colors.amber),
                                const SizedBox(width: 8),
                                Text(
                                  "Balance: ${(pokerModel.myAtomsBalance / 1e8).toStringAsFixed(4)} DCR",
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 16,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                                const Spacer(),
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
                    ),

                    // 2) Current table info
                    Center(
                      child: Container(
                        width: MediaQuery.of(context).size.width * 0.85,
                        margin: const EdgeInsets.only(top: 16.0),
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
                                  Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      const Text(
                                        "Not at a table",
                                        style: TextStyle(color: Colors.white),
                                      ),
                                      Row(children: [
                                        ElevatedButton.icon(
                                          onPressed: () {
                                            Navigator.pushNamed(context, '/table');
                                          },
                                          icon: const Icon(Icons.casino),
                                          label: const Text('View Table'),
                                          style: ElevatedButton.styleFrom(
                                              backgroundColor: Colors.green),
                                        ),
                                        const SizedBox(width: 8),
                                        ElevatedButton.icon(
                                          onPressed: () {
                                            // TODO: Implement create table functionality
                                            ScaffoldMessenger.of(context)
                                                .showSnackBar(const SnackBar(
                                                    content: Text(
                                                        'Create table functionality coming soon')));
                                          },
                                          icon: const Icon(Icons.add),
                                          label: const Text('Create Table'),
                                          style: ElevatedButton.styleFrom(
                                              backgroundColor: Colors.blueGrey),
                                        ),
                                      ]),
                                    ],
                                  ),
                                ] else ...[
                                  Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      Text(
                                        "Table ID: ${pokerModel.currentTableId}",
                                        style: const TextStyle(
                                          color: Colors.white,
                                        ),
                                      ),
                                      Text(
                                        "State: ${pokerModel.state.name}",
                                        style: const TextStyle(
                                          color: Colors.white,
                                        ),
                                      ),
                                    ],
                                  ),
                                  const SizedBox(height: 8),
                                  Row(
                                    mainAxisAlignment: MainAxisAlignment.end,
                                    children: [
                                      ElevatedButton(
                                        onPressed: pokerModel.leaveTable,
                                        style: ElevatedButton.styleFrom(
                                          backgroundColor: Colors.redAccent,
                                        ),
                                        child: const Text("Leave Table"),
                                      ),
                                    ],
                                  ),
                                ],
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),

                    // 3) Error message if exists
                    if (pokerModel.errorMessage.isNotEmpty)
                      Center(
                        child: Container(
                          width: MediaQuery.of(context).size.width * 0.85,
                          margin: const EdgeInsets.only(top: 16.0),
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
                                            text:
                                                pokerModel.errorMessage));
                                        if (!context.mounted) return;
                                        ScaffoldMessenger.of(context)
                                            .showSnackBar(const SnackBar(
                                                content: Text(
                                                    'Error copied to clipboard')));
                                      },
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
                                      onTap: () {
                                        pokerModel.clearError();
                                      },
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
                      ),

                    // 4) Main content (tables list / game view etc.)
                    Padding(
                      padding: const EdgeInsets.only(top: 12.0),
                      child: main_content.PokerMainContent(model: pokerModel),
                    ),
                  ],
                  ),
                ),
              );
            }),
    );
  }
}
