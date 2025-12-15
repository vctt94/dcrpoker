import 'dart:io';
import 'package:flutter/material.dart';
import 'package:path/path.dart' as p;

import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/models/newconfig.dart';
import 'package:pokerui/services/sound_service.dart';
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
  late final _debugLvl      = TextEditingController(text: widget.model.debugLevel);

  bool _soundsEnabled = true;
  late String _tableTheme;
  late String _cardTheme;
  late String _cardSize;
  late String _uiSize;
  late bool _hideTableLogo;
  String _cfgPath = '', _dataDir = '';

  @override
  void initState() {
    super.initState();
    _soundsEnabled = widget.model.soundsEnabled;
    _tableTheme = widget.model.tableTheme;
    _cardTheme = widget.model.cardTheme;
    _cardSize = widget.model.cardSize;
    _uiSize = widget.model.uiSize;
    _hideTableLogo = widget.model.hideTableLogo;
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
  Future<void> _createConfigCmd() async {
    try {
      // Create the config using the native plugin
      final config = CreateDefaultConfig(
        widget.model.dataDir,
        widget.model.serverAddr,
        widget.model.grpcCertPath,
        widget.model.debugLevel,
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

  // Update config with all settings
  Future<void> _updateConfigCmd() async {
    try {
      final updateArgs = UpdateConfig(
        widget.model.dataDir,
        widget.model.serverAddr,
        widget.model.grpcCertPath,
        widget.model.address,
        widget.model.debugLevel,
        _tableTheme,
        _cardTheme,
        _cardSize,
        _uiSize,
        _soundsEnabled,
        _hideTableLogo,
      );
      
      final result = await Golib.updateConfig(updateArgs);
      
      if (result['status'] != 'updated') {
        final err = result['error'] ?? 'unknown error';
        throw Exception('Failed to update config: $err');
      }
    } catch (e) {
      debugPrint('Native plugin updateConfig error: $e');
      throw Exception('Update config failed: $e');
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
      debugPrint('Native plugin createDefaultServerCert failed: $e');
      rethrow;
    }
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    try {
      // Ensure defaults have resolved before using paths.
      await widget.model.appDatadir();

      // update model from fields
      widget.model
        ..serverAddr        = _serverAddr.text
        ..grpcCertPath      = _grpcCert.text
        ..address           = _address.text
        ..debugLevel        = _debugLvl.text
        ..soundsEnabled     = _soundsEnabled
        ..tableTheme        = _tableTheme
        ..cardTheme         = _cardTheme
        ..cardSize          = _cardSize
        ..uiSize            = _uiSize
        ..hideTableLogo     = _hideTableLogo;

      await _prepareDataDir();
      
      // Use the new command to create config instead of direct file writing
      await _createConfigCmd();
      
      // Update theme and sound settings
      await _updateConfigCmd();
      
      // Update sound service immediately after saving config
      SoundService().setEnabled(_soundsEnabled);
      
      await widget.onConfigSaved();

      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('Config saved!')));
        await _initHeaderInfo();           // refresh header box
        // Close settings so callers can immediately reflect new config (theme/logo) without manual refresh.
        if (mounted) {
          final nav = Navigator.of(context);
          if (nav.canPop()) {
            nav.pop(true);
          }
        }
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
                const Text('Logging Configuration', 
                    style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                _field(_debugLvl, 'Debug Level'),
                const SizedBox(height: 12),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text('Enable Sounds', style: TextStyle(color: Colors.white)),
                    Switch(value: _soundsEnabled,
                           onChanged: (v) => setState(() => _soundsEnabled = v)),
                  ],
                ),
                const SizedBox(height: 12),
                const Text('Table Appearance', 
                    style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue: _tableTheme,
                  dropdownColor: const Color(0xFF1B1E2C),
                  iconEnabledColor: Colors.white,
                  decoration: const InputDecoration(
                    labelText: 'Table Theme',
                    labelStyle: TextStyle(color: Colors.white70),
                    enabledBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white54),
                    ),
                    focusedBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.blueAccent),
                    ),
                  ),
                  items: TableThemeConfig.presets
                      .map((t) => DropdownMenuItem(
                            value: t.key,
                            child: Text(t.displayName, style: const TextStyle(color: Colors.white)),
                          ))
                      .toList(),
                  onChanged: (value) {
                    if (value != null) {
                      setState(() => _tableTheme = value);
                    }
                  },
                ),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue: _cardTheme,
                  dropdownColor: const Color(0xFF1B1E2C),
                  iconEnabledColor: Colors.white,
                  decoration: const InputDecoration(
                    labelText: 'Card Theme',
                    labelStyle: TextStyle(color: Colors.white70),
                    enabledBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white54),
                    ),
                    focusedBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.blueAccent),
                    ),
                  ),
                  items: cardColorThemePresets
                      .map((t) => DropdownMenuItem(
                            value: cardColorThemeKey(t),
                            child: Text(cardColorThemeDisplayName(t), style: const TextStyle(color: Colors.white)),
                          ))
                      .toList(),
                  onChanged: (value) {
                    if (value != null) {
                      setState(() => _cardTheme = value);
                    }
                  },
                ),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue: _cardSize,
                  dropdownColor: const Color(0xFF1B1E2C),
                  iconEnabledColor: Colors.white,
                  decoration: const InputDecoration(
                    labelText: 'Card Size',
                    labelStyle: TextStyle(color: Colors.white70),
                    enabledBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white54),
                    ),
                    focusedBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.blueAccent),
                    ),
                  ),
                  items: const [
                    DropdownMenuItem(
                      value: 'xs',
                      child: Text('Extra Small', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'small',
                      child: Text('Small', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'medium',
                      child: Text('Medium', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'large',
                      child: Text('Large', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'xl',
                      child: Text('Extra Large', style: TextStyle(color: Colors.white)),
                    ),
                  ],
                  onChanged: (value) {
                    if (value != null) {
                      setState(() => _cardSize = value);
                    }
                  },
                ),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue: _uiSize,
                  dropdownColor: const Color(0xFF1B1E2C),
                  iconEnabledColor: Colors.white,
                  decoration: const InputDecoration(
                    labelText: 'UI Size (Icons, Fonts, Player Circles)',
                    labelStyle: TextStyle(color: Colors.white70),
                    enabledBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white54),
                    ),
                    focusedBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.blueAccent),
                    ),
                  ),
                  items: const [
                    DropdownMenuItem(
                      value: 'xs',
                      child: Text('Extra Small', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'small',
                      child: Text('Small', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'medium',
                      child: Text('Medium', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'large',
                      child: Text('Large', style: TextStyle(color: Colors.white)),
                    ),
                    DropdownMenuItem(
                      value: 'xl',
                      child: Text('Extra Large', style: TextStyle(color: Colors.white)),
                    ),
                  ],
                  onChanged: (value) {
                    if (value != null) {
                      setState(() => _uiSize = value);
                    }
                  },
                ),
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text('Show Table Logo', style: TextStyle(color: Colors.white)),
                    Switch(
                      value: !_hideTableLogo,
                      onChanged: (v) => setState(() => _hideTableLogo = !v),
                    ),
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
