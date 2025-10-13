import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class TopStatusCard extends StatelessWidget {
  final PokerModel pokerModel;
  final VoidCallback? onErrorDismissed;

  const TopStatusCard({
    Key? key,
    required this.pokerModel,
    this.onErrorDismissed,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Compact Status Section
        Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            children: [
              Card(
                elevation: 2,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Padding(
                  padding: const EdgeInsets.all(12.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text(
                            "Balance: ${(pokerModel.myAtomsBalance / 1e8).toStringAsFixed(4)} DCR",
                            style: Theme.of(context).textTheme.bodyMedium,
                          ),
                          Text(
                            pokerModel.iAmReady
                                ? (pokerModel.game?.gameStarted ?? false
                                    ? "In Game"
                                    : "Ready")
                                : "Not Ready",
                            style: Theme.of(context).textTheme.bodyMedium,
                          ),
                        ],
                      ),
                      // If game hasn't started, show table info
                      if (!(pokerModel.game?.gameStarted ?? false)) ...[
                        const SizedBox(height: 12),
                        Divider(color: Colors.grey.shade400),
                        const SizedBox(height: 12),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              "Current Table:",
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            if (pokerModel.currentTableId != null)
                              Row(
                                children: [
                                  FilledButton(
                                    onPressed: pokerModel.iAmReady 
                                        ? pokerModel.setUnready 
                                        : pokerModel.setReady,
                                    child: Text(
                                      pokerModel.iAmReady
                                          ? "Cancel Ready"
                                          : "Ready",
                                    ),
                                  ),
                                  const SizedBox(width: 8),
                                  FilledButton(
                                    style: FilledButton.styleFrom(
                                      backgroundColor: Colors.redAccent,
                                    ),
                                    onPressed: pokerModel.leaveTable,
                                    child: const Text("Leave Table"),
                                  ),
                                ],
                              ),
                          ],
                        ),
                        const SizedBox(height: 8),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              "Table ID: ${pokerModel.currentTableId ?? "None"}",
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          ],
                        ),
                        const SizedBox(height: 8),
                        Text(
                          "State: ${pokerModel.state.name}",
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),

        // Error Message
        if (pokerModel.errorMessage.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Card(
              color: Colors.red.shade100,
              child: Padding(
                padding: const EdgeInsets.all(8.0),
                child: Row(
                  children: [
                    const Icon(Icons.error, color: Colors.red),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        pokerModel.errorMessage,
                        style: Theme.of(context)
                            .textTheme
                            .bodyMedium
                            ?.copyWith(color: Colors.red),
                      ),
                    ),
                    Material(
                      color: Colors.transparent,
                      child: InkWell(
                        onTap: () {
                          pokerModel.clearError();
                          onErrorDismissed?.call();
                        },
                        borderRadius: BorderRadius.circular(20),
                        child: const Padding(
                          padding: EdgeInsets.all(8.0),
                          child: Icon(Icons.close, color: Colors.red, size: 20),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }
}
