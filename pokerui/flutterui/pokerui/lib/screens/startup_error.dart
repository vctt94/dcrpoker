import 'package:flutter/material.dart';

class StartupErrorScreen extends StatelessWidget {
  const StartupErrorScreen({
    super.key,
    required this.message,
    required this.onRetry,
    required this.onOpenConfig,
    required this.dataDir,
    this.missingFields = const [],
  });

  final String message;
  final Future<void> Function() onRetry;
  final Future<void> Function(BuildContext context) onOpenConfig;
  final List<String> missingFields;
  final String dataDir;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final configPath =
        dataDir.isNotEmpty ? '$dataDir/pokerui.conf' : 'pokerui.conf';
    final suggestions = _buildSuggestions();

    return Scaffold(
      backgroundColor: theme.scaffoldBackgroundColor,
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 520),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Icon(Icons.error_outline,
                    color: Colors.redAccent, size: 64),
                const SizedBox(height: 16),
                Text(
                  'Unable to start Poker UI',
                  style: theme.textTheme.headlineSmall,
                ),
                const SizedBox(height: 12),
                SelectableText(
                  'Config file: $configPath',
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(height: 12),
                if (suggestions.isNotEmpty) ...[
                  Text(
                    'What to fix:',
                    style: theme.textTheme.titleMedium,
                  ),
                  const SizedBox(height: 8),
                  ...suggestions.map(
                    (hint) => Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Padding(
                            padding: EdgeInsets.only(top: 4),
                            child: Icon(Icons.arrow_right,
                                size: 16, color: Colors.white70),
                          ),
                          const SizedBox(width: 4),
                          Expanded(
                            child: Text(
                              hint,
                              style: theme.textTheme.bodyMedium,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                ],
                Text(
                  'Error details:',
                  style: theme.textTheme.titleMedium,
                ),
                const SizedBox(height: 8),
                SelectableText(
                  message,
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(height: 24),
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    ElevatedButton(
                      onPressed: () async {
                        await onRetry();
                      },
                      child: const Text('Try Again'),
                    ),
                    OutlinedButton(
                      onPressed: () async {
                        await onOpenConfig(context);
                      },
                      child: const Text('Edit Settings'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  List<String> _buildSuggestions() {
    if (missingFields.isEmpty) {
      return const [];
    }

    const fieldHelp = <String, String>{
      'brrpcurl':
          'Provide the BR RPC WebSocket URL (Settings → BisonRelay → BR RPC WebSocket URL).',
      'brclientcert':
          'Point to the Bison Relay rpc.cert (Settings → BisonRelay → BR Client Cert Path).',
      'brclientrpccert':
          'Point to the rpc-client.cert from your Bison Relay data directory.',
      'brclientrpckey':
          'Point to the rpc-client.key from your Bison Relay data directory.',
      'rpcuser': 'Set the RPC username used to authenticate with Bison Relay.',
      'rpcpass': 'Set the RPC password used to authenticate with Bison Relay.',
    };

    final seen = <String>{};
    final hints = <String>[];
    for (final field in missingFields) {
      if (!seen.add(field)) {
        continue;
      }
      hints.add(fieldHelp[field] ??
          'Add a value for `$field` in Settings → BisonRelay.');
    }
    return hints;
  }
}
