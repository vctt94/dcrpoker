import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/models/poker.dart';

class SharedLayout extends StatelessWidget {
  final String title;
  final Widget child;

  const SharedLayout({
    super.key,
    required this.title,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    // Try to get PokerModel, but don't throw if it's not available
    PokerModel? pokerModel;
    try {
      pokerModel = Provider.of<PokerModel>(context);
    } catch (e) {
      // PokerModel not available, we'll use a simplified layout
    }

    return Scaffold(
      backgroundColor: const Color(0xFF121212), // Dark background
      appBar: AppBar(
        backgroundColor: const Color(0xFF1A1A1A), // Dark app bar
        foregroundColor: Colors.white, // White text and icons
        title: Text(title),
        leading: Navigator.of(context).canPop()
            ? IconButton(
                icon: const Icon(Icons.arrow_back),
                onPressed: () {
                  Navigator.of(context).pop();
                },
              )
            : null,
      ),
      drawer: pokerModel != null
          ? Drawer(
              child: Container(
                color: const Color(0xFF1A1A1A), // Dark drawer background
                child: ListView(
                  padding: EdgeInsets.zero,
                  children: [
                    DrawerHeader(
                      decoration: const BoxDecoration(color: Colors.blueAccent),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          const Text(
                            'Poker Menu',
                            style: TextStyle(
                              color: Colors.white,
                              fontSize: 24,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          const SizedBox(height: 8),
                          Text(
                            'Player ID: ${pokerModel.playerId}',
                            style: const TextStyle(
                              color: Colors.white70,
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                    ListTile(
                      leading: const Icon(Icons.home, color: Colors.white),
                      title: const Text('Home',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushReplacementNamed('/');
                      },
                    ),
                    ListTile(
                      leading: const Icon(Icons.settings, color: Colors.white),
                      title: const Text('Settings',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/settings');
                      },
                    ),
                    ListTile(
                      leading: const Icon(Icons.description, color: Colors.white),
                      title: const Text('Logs',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/logs');
                      },
                    ),
                  ],
                ),
              ),
            )
          : null,
      body: Column(
        children: [
          Expanded(child: child),
          // Footer Section - only shown when PokerModel is available
          if (pokerModel != null)
            Container(
              padding: const EdgeInsets.all(16),
              decoration: const BoxDecoration(
                color: Color(0xFF1B1E2C),
                borderRadius: BorderRadius.vertical(top: Radius.circular(12)),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Icon(
                        pokerModel.state != PokerState.idle
                            ? Icons.check_circle
                            : Icons.cloud_off,
                        color:
                            pokerModel.state != PokerState.idle ? Colors.green : Colors.red,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        pokerModel.state != PokerState.idle ? "Connected" : "Disconnected",
                        style: TextStyle(
                          color:
                              pokerModel.state != PokerState.idle ? Colors.green : Colors.red,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                  Text(
                    "Player ID: ${pokerModel.playerId}",
                    style: const TextStyle(
                      color: Colors.white70,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
