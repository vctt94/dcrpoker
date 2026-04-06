import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

/// Shell for lobby/non-gameplay screens with minimal app bar and drawer.
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

    final pokerModel = _maybePokerModel(context);
    final logoutCb = _resolveLogoutCallback(context, onLogout);

    return Scaffold(
      backgroundColor: PokerColors.screenBg,
      appBar: AppBar(
        backgroundColor: PokerColors.surfaceDim,
        foregroundColor: PokerColors.textPrimary,
        elevation: 0,
        title: Row(
          children: [
            Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: pokerModel != null && pokerModel.state != PokerState.idle
                    ? PokerColors.success
                    : PokerColors.danger,
              ),
            ),
            const SizedBox(width: PokerSpacing.sm),
            Text(title, style: PokerTypography.titleLarge),
          ],
        ),
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
      drawer: _buildPokerDrawer(context, pokerModel, logoutCb),
      body: child,
    );
  }
}

/// Full-bleed game shell with zero chrome for gameplay.
class GameShell extends StatelessWidget {
  const GameShell({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final pokerModel = _maybePokerModel(context);
    final logoutCb = _resolveLogoutCallback(context, null);
    final drawer = _buildPokerDrawer(context, pokerModel, logoutCb);

    return Scaffold(
      backgroundColor: PokerColors.screenBg,
      drawer: drawer,
      body: Stack(
        fit: StackFit.expand,
        children: [
          child,
          if (drawer != null)
            const Positioned(
              top: 0,
              left: 0,
              child: SafeArea(
                child: Padding(
                  padding: EdgeInsets.all(PokerSpacing.md),
                  child: _GameMenuButton(),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

PokerModel? _maybePokerModel(BuildContext context) {
  try {
    return Provider.of<PokerModel>(context);
  } catch (_) {
    return null;
  }
}

Future<void> Function()? _resolveLogoutCallback(
  BuildContext context,
  Future<void> Function()? fallback,
) {
  if (fallback != null) {
    return fallback;
  }
  try {
    return Provider.of<Future<void> Function()?>(context, listen: false);
  } catch (_) {
    return null;
  }
}

Widget? _buildPokerDrawer(
  BuildContext context,
  PokerModel? pokerModel,
  Future<void> Function()? logoutCb,
) {
  if (pokerModel == null) {
    return null;
  }

  final navigator = Navigator.of(context);

  return Drawer(
    child: Container(
      color: PokerColors.surfaceDim,
      child: ListView(
        padding: EdgeInsets.zero,
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(
              PokerSpacing.lg,
              PokerSpacing.xxxl,
              PokerSpacing.lg,
              PokerSpacing.lg,
            ),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  PokerColors.primary.withValues(alpha: 0.3),
                  PokerColors.surfaceDim,
                ],
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(
                      Icons.style,
                      color: PokerColors.primary,
                      size: 28,
                    ),
                    const SizedBox(width: PokerSpacing.sm),
                    Text('Poker', style: PokerTypography.headlineLarge),
                  ],
                ),
                const SizedBox(height: PokerSpacing.sm),
                Text(
                  pokerModel.playerId.length > 16
                      ? '${pokerModel.playerId.substring(0, 16)}...'
                      : pokerModel.playerId,
                  style: PokerTypography.bodySmall,
                ),
              ],
            ),
          ),
          const Divider(height: 1, color: PokerColors.borderSubtle),
          _DrawerItem(
            icon: Icons.home_outlined,
            label: 'Home',
            onTap: () {
              navigator.pop();
              pokerModel.showHomeView();
              navigator.popUntil((route) => route.isFirst);
            },
          ),
          _DrawerItem(
            icon: Icons.verified_outlined,
            label: 'Sign Address',
            onTap: () => Navigator.of(context).pushNamed('/sign-address'),
          ),
          _DrawerItem(
            icon: Icons.lock_outline,
            label: 'Open Escrow',
            onTap: () => Navigator.of(context).pushNamed('/open-escrow'),
          ),
          _DrawerItem(
            icon: Icons.history_outlined,
            label: 'Escrow History',
            onTap: () => Navigator.of(context).pushNamed('/escrow-history'),
          ),
          const Divider(height: 1, color: PokerColors.borderSubtle),
          _DrawerItem(
            icon: Icons.description_outlined,
            label: 'Logs',
            onTap: () => Navigator.of(context).pushNamed('/logs'),
          ),
          _DrawerItem(
            icon: Icons.settings_outlined,
            label: 'Settings',
            onTap: () => Navigator.of(context).pushNamed('/settings'),
          ),
          if (logoutCb != null)
            _DrawerItem(
              icon: Icons.logout,
              label: 'Logout',
              color: PokerColors.danger,
              onTap: () async {
                Navigator.of(context).pop();
                await logoutCb();
              },
            ),
        ],
      ),
    ),
  );
}

class _GameMenuButton extends StatelessWidget {
  const _GameMenuButton();

  @override
  Widget build(BuildContext context) {
    return Builder(
      builder: (context) => Material(
        color: PokerColors.surfaceDim.withValues(alpha: 0.9),
        elevation: 6,
        shadowColor: Colors.black45,
        shape: const CircleBorder(),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: () => Scaffold.of(context).openDrawer(),
          child: const SizedBox(
            width: 44,
            height: 44,
            child: Icon(
              Icons.menu_rounded,
              color: PokerColors.textPrimary,
              size: 22,
            ),
          ),
        ),
      ),
    );
  }
}

class _DrawerItem extends StatelessWidget {
  const _DrawerItem({
    required this.icon,
    required this.label,
    required this.onTap,
    this.color,
  });
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final c = color ?? PokerColors.textPrimary;
    return ListTile(
      leading: Icon(icon, color: c, size: 22),
      title: Text(label, style: PokerTypography.bodyMedium.copyWith(color: c)),
      onTap: onTap,
      dense: true,
      contentPadding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.lg,
        vertical: PokerSpacing.xxs,
      ),
    );
  }
}
