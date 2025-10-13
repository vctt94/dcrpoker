import 'dart:io';
import 'package:flutter/material.dart';
import 'package:path/path.dart' as p;

import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/newconfig.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';

class NewConfigScreen extends StatefulWidget {
  const NewConfigScreen({
    super.key,
    required this.model,
    required this.onConfigSaved,
  });

  final NewConfigModel model;
  final Future<void> Function() onConfigSaved;

  @override
  State<NewConfigScreen> createState() => _NewConfigScreenState();
}

class _NewConfigScreenState extends State<NewConfigScreen> {
  final _formKey = GlobalKey<FormState>();

  // text controllers
  late final _serverAddr    = TextEditingController(text: widget.model.serverAddr);
  late final _grpcCert      = TextEditingController(text: widget.model.grpcCertPath);
  late final _address       = TextEditingController(text: widget.model.address);
  late final _brRpcUrl      = TextEditingController(text: widget.model.brRpcUrl);
  late final _brClientCert  = TextEditingController(text: widget.model.brClientCert);
  late final _brClientRpcCert = TextEditingController(text: widget.model.brClientRpcCert);
  late final _brClientRpcKey = TextEditingController(text: widget.model.brClientRpcKey);
  late final _debugLvl      = TextEditingController(text: widget.model.debugLevel);
  late final _user          = TextEditingController(text: widget.model.rpcUser);
  late final _pass          = TextEditingController(text: widget.model.rpcPass);

  bool _wantsLogNtfns = false;
  String _cfgPath = '', _dataDir = '';

  // Placeholder certificate content
  static const String placeholderCertContent = '''-----BEGIN CERTIFICATE-----
MIIBkzCCATmgAwIBAgIRAOCyLu1U/ZKyD33nXFPgJOQwCgYIKoZIzj0EAwIwFjEU
MBIGA1UEChMLUG9uZyBTZXJ2ZXIwHhcNMjUwMTMxMTg0NzQwWhcNMjYwMTMxMTg0
NzQwWjAWMRQwEgYDVQQKEwtQb25nIFNlcnZlcjBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABLpaje+KDrdAe77RwOaxYAkxRmlDg1cbLspf1riFhskIUyfILM1r8zPd
Ql10MGxeKipbE3LCPOn5BV0KVGxfb2mjaDBmMA4GA1UdDwEB/wQEAwICpDATBgNV
HSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQLw3WW
CxxXpNfuuDgGuZ3c8IX0rDAPBgNVHREECDAGhwRog7QdMAoGCCqGSM49BAMCA0gA
MEUCIEWR7Iw7ua6WAQuIf8Yf0lNzP6s2NczAR0W4uP8zuVK0AiEA6ruxkMcv4CHw
Aq6RDElOTqAlDbNAuV8b/joQjIDLwqA=
-----END CERTIFICATE-----''';
  @override
  void initState() {
    super.initState();
    _wantsLogNtfns = widget.model.wantsLogNtfns;
    _initHeaderInfo();
  }

  Future<void> _initHeaderInfo() async {
    _dataDir = await widget.model.appDatadir();
    _cfgPath = await widget.model.getConfigFilePath();
    if (mounted) setState(() {});
  }

  // ensure server.cert and logs/ exist in the fixed data dir
  Future<void> _prepareDataDir() async {
    final grpcCertFile = File(widget.model.grpcCertPath);
    if (!await grpcCertFile.exists()) {
      // Use the new command to create the certificate instead of direct file writing
      await _createServerCertViaCommand();
    }
    final logs = Directory(p.join(widget.model.dataDir, 'logs'));
    if (!await logs.exists()) await logs.create(recursive: true);
  }

  // Create config using the new command
  Future<void> _createConfigViaCommand() async {
    try {
      // Create the config using the native plugin
      final config = CreateDefaultConfig(
        widget.model.dataDir,
        widget.model.serverAddr,
        widget.model.grpcCertPath,
        widget.model.debugLevel,
        widget.model.brRpcUrl,
        widget.model.brClientCert,
        widget.model.brClientRpcCert,
        widget.model.brClientRpcKey,
        widget.model.rpcUser,
        widget.model.rpcPass,
      );
      
      // Call the native plugin command
      final result = await Golib.createDefaultConfig(config);
      
      if (result['status'] != 'created') {
        final err = result['error'] ?? 'unknown error';
        throw Exception('Failed to create config: $err');
      }
    } catch (e) {
      // Surface native error to UI
      debugPrint('Native plugin createDefaultConfig error: $e');
      throw Exception('Create config failed: $e');
    }
  }

  // Create server certificate using the new command
  Future<void> _createServerCertViaCommand() async {
    try {
      // Call the native plugin command to create the server certificate
      final result = await Golib.createDefaultServerCert(widget.model.grpcCertPath);
      
      if (result['status'] != 'created') {
        throw Exception('Failed to create server certificate: ${result['error']}');
      }
    } catch (e) {
      // Fall back to the old method if native plugin fails
      debugPrint('Native plugin failed, falling back to old method: $e');
      final grpcCertFile = File(widget.model.grpcCertPath);
      await grpcCertFile.parent.create(recursive: true);
      await grpcCertFile.writeAsString(placeholderCertContent);
    }
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    try {
      // update model from fields
      widget.model
        ..serverAddr        = _serverAddr.text
        ..grpcCertPath      = _grpcCert.text
        ..address           = _address.text
        ..brRpcUrl          = _brRpcUrl.text
        ..brClientCert      = _brClientCert.text
        ..brClientRpcCert   = _brClientRpcCert.text
        ..brClientRpcKey    = _brClientRpcKey.text
        ..debugLevel        = _debugLvl.text
        ..rpcUser           = _user.text
        ..rpcPass           = _pass.text
        ..wantsLogNtfns     = _wantsLogNtfns;

      await _prepareDataDir();
      
      // Use the new command to create config instead of direct file writing
      await _createConfigViaCommand();
      await widget.onConfigSaved();

      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('Config saved!')));
        await _initHeaderInfo();           // refresh header box
      }
    } catch (e, st) {
      debugPrint('Error saving config: $e\n$st');
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Settings',
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: SingleChildScrollView(
            child: Column(
              children: [
                _HeaderBox(cfgPath: _cfgPath, dataDir: _dataDir),
                const SizedBox(height: 20),
                _field(_serverAddr, 'Server Address', required: true),
                _field(_grpcCert, 'gRPC Server Cert Path'),
                _field(_address, 'Payout Address or PubKey (33/65B hex)'),
                const SizedBox(height: 12),
                const Text('BisonRelay Configuration', 
                    style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                _field(_brRpcUrl, 'BR RPC WebSocket URL', required: true),
                _field(_brClientCert, 'BR Client Cert Path', required: true),
                _field(_brClientRpcCert, 'BR Client RPC Cert Path', required: true),
                _field(_brClientRpcKey, 'BR Client RPC Key Path', required: true),
                _field(_user, 'RPC User', required: true),
                _field(_pass, 'RPC Password', required: true, obscure: true),
                const SizedBox(height: 12),
                const Text('Logging Configuration', 
                    style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                _field(_debugLvl, 'Debug Level'),
                const SizedBox(height: 12),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text('Log Notifications', style: TextStyle(color: Colors.white)),
                    Switch(value: _wantsLogNtfns,
                           onChanged: (v) => setState(() => _wantsLogNtfns = v)),
                  ],
                ),
                const SizedBox(height: 20),
                ElevatedButton(onPressed: _save, child: const Text('Save Config')),
              ],
            ),
          ),
        ),
      ),
    );
  }

  // simple builder for text fields
  Widget _field(TextEditingController c, String label,
      {bool required = false, bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: TextFormField(
        controller: c,
        obscureText: obscure,
        style: const TextStyle(color: Colors.white),
        decoration: InputDecoration(
          labelText: label,
          labelStyle: const TextStyle(color: Colors.white70),
          enabledBorder: const UnderlineInputBorder(
            borderSide: BorderSide(color: Colors.white54),
          ),
          focusedBorder: const UnderlineInputBorder(
            borderSide: BorderSide(color: Colors.blueAccent),
          ),
        ),
        validator: required
            ? (v) => v == null || v.isEmpty ? 'Required' : null
            : null,
      ),
    );
  }
}

// ─── Small header widget just for display ──────────────────────────────────
class _HeaderBox extends StatelessWidget {
  const _HeaderBox({required this.cfgPath, required this.dataDir});
  final String cfgPath, dataDir;

  @override
  Widget build(BuildContext context) {
    if (cfgPath.isEmpty) {
      return const Text('Loading...', style: TextStyle(color: Colors.white70));
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFF1B1E2C),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.blueAccent.withOpacity(.3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Row(
            children: [
              Icon(Icons.settings_applications, color: Colors.blueAccent),
              SizedBox(width: 8),
              Text('Config & Data Directory',
                  style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
            ],
          ),
          const SizedBox(height: 12),
          const Text('Config file:', style: TextStyle(color: Colors.white70)),
          _Code(cfgPath),
          const SizedBox(height: 8),
          const Text('Data directory:', style: TextStyle(color: Colors.white70)),
          _Code(dataDir),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}

class _Code extends StatelessWidget {
  const _Code(this.text);
  final String text;
  @override
  Widget build(BuildContext context) => Container(
        width: double.infinity,
        padding: const EdgeInsets.all(8),
        margin: const EdgeInsets.only(top: 4),
        decoration: BoxDecoration(
          color: const Color(0xFF0F0F0F),
          borderRadius: BorderRadius.circular(4),
          border: Border.all(color: Colors.grey.shade700),
        ),
        child: SelectableText(text,
            style: const TextStyle(color: Colors.white, fontFamily: 'monospace', fontSize: 12)),
      );
}
