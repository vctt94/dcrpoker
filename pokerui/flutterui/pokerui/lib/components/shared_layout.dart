import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/models/poker.dart';

class SharedLayout extends StatelessWidget {
  final String title;
  final Widget child;
  final Future<void> Function()? onLogout;
  final bool hideFooter;

  const SharedLayout({
    super.key,
    required this.title,
    required this.child,
    this.onLogout,
    this.hideFooter = false,
  });

  @override
  Widget build(BuildContext context) {
    final navigator = Navigator.of(context);
    final route = ModalRoute.of(context);
    final isHomeRoute = route?.settings.name == Navigator.defaultRouteName;
    final showBackButton = navigator.canPop() && !isHomeRoute;

    // Try to get PokerModel, but don't throw if it's not available
    PokerModel? pokerModel;
    Future<void> Function()? logoutCb = onLogout;
    if (logoutCb == null) {
      try {
        logoutCb =
            Provider.of<Future<void> Function()?>(context, listen: false);
      } catch (_) {
        logoutCb = null;
      }
    }
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
        leading: showBackButton
            ? IconButton(
                icon: const Icon(Icons.arrow_back),
                onPressed: () async {
                  final didPop = await navigator.maybePop();
                  if (!didPop && navigator.canPop()) {
                    navigator.popUntil((route) => route.isFirst);
                  }
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
                        navigator.pop();
                        pokerModel?.showHomeView();
                        navigator.popUntil((route) => route.isFirst);
                      },
                    ),
                    ListTile(
                      leading: const Icon(Icons.verified, color: Colors.white),
                      title: const Text('Sign Address',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/sign-address');
                      },
                    ),
                    ListTile(
                      leading: const Icon(Icons.lock, color: Colors.white),
                      title: const Text('Open Escrow',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/open-escrow');
                      },
                    ),
                    ListTile(
                      leading: const Icon(Icons.history, color: Colors.white),
                      title: const Text('Escrow History',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/escrow-history');
                      },
                    ),
                    ListTile(
                      leading:
                          const Icon(Icons.description, color: Colors.white),
                      title: const Text('Logs',
                          style: TextStyle(color: Colors.white)),
                      onTap: () {
                        Navigator.of(context).pushNamed('/logs');
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
                    if (logoutCb != null)
                      ListTile(
                        leading: const Icon(Icons.logout, color: Colors.white),
                        title: const Text('Logout',
                            style: TextStyle(color: Colors.white)),
                        onTap: () async {
                          Navigator.of(context).pop();
                          await logoutCb?.call();
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
          if (pokerModel != null && !hideFooter)
            Container(
              padding: const EdgeInsets.all(16),
              decoration: const BoxDecoration(
                color: Color(0xFF1B1E2C),
                borderRadius: BorderRadius.vertical(top: Radius.circular(12)),
              ),
              child: LayoutBuilder(
                builder: (context, constraints) {
                  final model = pokerModel!;
                  final isNarrow = constraints.maxWidth < 480;
                  final status = Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        model.state != PokerState.idle
                            ? Icons.check_circle
                            : Icons.cloud_off,
                        color: model.state != PokerState.idle
                            ? Colors.green
                            : Colors.red,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        model.state != PokerState.idle
                            ? "Connected"
                            : "Disconnected",
                        style: TextStyle(
                          color: model.state != PokerState.idle
                              ? Colors.green
                              : Colors.red,
                          fontWeight: FontWeight.bold,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  );

                  final playerId = Text(
                    "Player ID: ${model.playerId}",
                    style: const TextStyle(
                      color: Colors.white70,
                      fontWeight: FontWeight.w500,
                    ),
                    overflow: TextOverflow.ellipsis,
                  );

                  if (isNarrow) {
                    return Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        status,
                        const SizedBox(height: 6),
                        playerId,
                      ],
                    );
                  }

                  return Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Flexible(child: status),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Align(
                          alignment: Alignment.centerRight,
                          child: playerId,
                        ),
                      ),
                    ],
                  );
                },
              ),
            ),
        ],
      ),
    );
  }
}
