import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:path/path.dart' as p;

import 'package:golib_plugin/definitions.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:pokerui/components/poker/settings_preview.dart';
import 'package:pokerui/components/poker/table_theme.dart';
import 'package:pokerui/components/shared_layout.dart';
import 'package:pokerui/models/newconfig.dart';
import 'package:pokerui/services/sound_service.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/spacing.dart';
import 'package:pokerui/theme/typography.dart';

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

  late final _serverAddr = TextEditingController(text: widget.model.serverAddr);
  late final _grpcCert = TextEditingController(text: widget.model.grpcCertPath);
  late final _nickname = TextEditingController(text: widget.model.nickname);
  late final _address = TextEditingController(text: widget.model.address);
  late final _debugLvl = TextEditingController(text: widget.model.debugLevel);

  bool _soundsEnabled = true;
  late String _tableTheme;
  late String _cardTheme;
  late String _cardSize;
  late String _uiSize;
  late bool _hideTableLogo;
  late String _logoPosition;
  _SettingsSection _selectedSection = _SettingsSection.general;
  String _cfgPath = '';
  String _dataDir = '';

  @override
  void initState() {
    super.initState();
    _soundsEnabled = widget.model.soundsEnabled;
    _tableTheme = widget.model.tableTheme;
    _cardTheme = widget.model.cardTheme;
    _cardSize = widget.model.cardSize;
    _uiSize = widget.model.uiSize;
    _hideTableLogo = widget.model.hideTableLogo;
    _logoPosition = widget.model.logoPosition;
    _initHeaderInfo();
  }

  @override
  void dispose() {
    _serverAddr.dispose();
    _grpcCert.dispose();
    _nickname.dispose();
    _address.dispose();
    _debugLvl.dispose();
    super.dispose();
  }

  Future<void> _initHeaderInfo() async {
    _dataDir = await widget.model.appDatadir();
    _cfgPath = await widget.model.getConfigFilePath();
    if (mounted) setState(() {});
  }

  Future<void> _prepareDataDir() async {
    final grpcCertFile = File(widget.model.grpcCertPath);
    if (!await grpcCertFile.exists()) {
      await _createServerCertViaCommand();
    }
    final logs = Directory(p.join(widget.model.dataDir, 'logs'));
    if (!await logs.exists()) await logs.create(recursive: true);
  }

  Future<void> _createConfigCmd() async {
    try {
      final config = CreateDefaultConfig(
        widget.model.dataDir,
        widget.model.serverAddr,
        widget.model.grpcCertPath,
        widget.model.debugLevel,
      );

      final result = await Golib.createDefaultConfig(config);

      if (result['status'] != 'created') {
        final err = result['error'] ?? 'unknown error';
        throw Exception('Failed to create config: $err');
      }
    } catch (e) {
      debugPrint('Native plugin createDefaultConfig error: $e');
      throw Exception('Create config failed: $e');
    }
  }

  Future<void> _updateConfigCmd() async {
    try {
      final updateArgs = UpdateConfig(
        widget.model.dataDir,
        widget.model.serverAddr,
        widget.model.grpcCertPath,
        widget.model.nickname,
        widget.model.address,
        widget.model.debugLevel,
        _tableTheme,
        _cardTheme,
        _cardSize,
        _uiSize,
        _soundsEnabled,
        _hideTableLogo,
        _logoPosition,
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

  Future<void> _createServerCertViaCommand() async {
    try {
      final result =
          await Golib.createDefaultServerCert(widget.model.grpcCertPath);

      if (result['status'] != 'created') {
        throw Exception(
            'Failed to create server certificate: ${result['error']}');
      }
    } catch (e) {
      debugPrint('Native plugin createDefaultServerCert failed: $e');
      rethrow;
    }
  }

  Future<void> _save() async {
    final serverAddr = _serverAddr.text.trim();
    if (serverAddr.isEmpty) {
      if (_selectedSection != _SettingsSection.general) {
        setState(() => _selectedSection = _SettingsSection.general);
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Server address is required')),
      );
      return;
    }
    if (!_formKey.currentState!.validate()) return;
    try {
      await widget.model.appDatadir();

      widget.model
        ..serverAddr = serverAddr
        ..grpcCertPath = _grpcCert.text
        ..nickname = _nickname.text.trim()
        ..address = _address.text
        ..debugLevel = _debugLvl.text
        ..soundsEnabled = _soundsEnabled
        ..tableTheme = _tableTheme
        ..cardTheme = _cardTheme
        ..cardSize = _cardSize
        ..uiSize = _uiSize
        ..hideTableLogo = _hideTableLogo
        ..logoPosition = _logoPosition;

      await _prepareDataDir();
      await _createConfigCmd();
      await _updateConfigCmd();

      SoundService().setEnabled(_soundsEnabled);
      await widget.onConfigSaved();

      if (!mounted) return;

      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Config saved')),
      );
      await _initHeaderInfo();

      if (!mounted) return;
      final nav = Navigator.of(context);
      if (nav.canPop()) {
        nav.pop(true);
      }
    } catch (e, st) {
      debugPrint('Error saving config: $e\n$st');
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Error: $e')),
      );
    }
  }

  Future<void> _pickGrpcCert() async {
    try {
      const allowedExtensions = {'cert', 'crt', 'pem'};
      final result = await FilePicker.platform.pickFiles(
        allowMultiple: false,
        type: Platform.isAndroid ? FileType.any : FileType.custom,
        allowedExtensions:
            Platform.isAndroid ? null : allowedExtensions.toList(),
      );

      if (result == null) {
        return;
      }

      final selectedFile = result.files.single;
      final selectedExtension =
          p.extension(selectedFile.name).replaceFirst('.', '').toLowerCase();
      if (!allowedExtensions.contains(selectedExtension)) {
        throw Exception(
          'Please select a certificate file (.cert, .crt, or .pem)',
        );
      }

      final selectedPath = selectedFile.path;
      if (selectedPath == null || selectedPath.isEmpty) {
        throw Exception('Selected file path is not accessible on this device');
      }

      await widget.model.appDatadir();
      final importPath = p.join(widget.model.dataDir, selectedFile.name);
      final sourcePath = p.normalize(selectedPath);
      final targetPath = p.normalize(importPath);

      if (sourcePath != targetPath) {
        await Directory(widget.model.dataDir).create(recursive: true);
        await File(sourcePath).copy(targetPath);
      }

      setState(() {
        _grpcCert.text = targetPath;
      });
    } catch (e) {
      debugPrint('Error picking gRPC cert: $e');
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Unable to select certificate: $e')),
      );
    }
  }

  Future<void> _copyValue(String label, String value) async {
    await Clipboard.setData(ClipboardData(text: value));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('$label copied')),
    );
  }

  PokerUiSettings get _draftUiSettings => PokerUiSettings(
        tableThemeKey: _tableTheme,
        cardThemeKey: _cardTheme,
        cardScale: PokerScalePreset.fromKey(_cardSize),
        densityScale: PokerScalePreset.fromKey(_uiSize),
        showTableLogo: !_hideTableLogo,
        logoPosition: _logoPosition,
      );

  @override
  Widget build(BuildContext context) {
    return SharedLayout(
      title: 'Settings',
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final isWide = constraints.maxWidth >= 900;
            final constrainedWidth =
                constraints.maxWidth > 1240 ? 1240.0 : constraints.maxWidth;
            final contentWidth = constrainedWidth - (PokerSpacing.lg * 2);

            return Align(
              alignment: Alignment.topCenter,
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 1240),
                child: Padding(
                  padding: const EdgeInsets.all(PokerSpacing.lg),
                  child: Form(
                    key: _formKey,
                    child: SingleChildScrollView(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Text(
                            'Manage connection details and table preferences.',
                            style: PokerTypography.bodyMedium.copyWith(
                              color: PokerColors.textSecondary,
                            ),
                          ),
                          const SizedBox(height: PokerSpacing.lg),
                          _SettingsMenuRow(
                            selected: _selectedSection,
                            isWide: isWide,
                            onSelected: (section) {
                              setState(() => _selectedSection = section);
                            },
                          ),
                          const SizedBox(height: PokerSpacing.lg),
                          AnimatedSwitcher(
                            duration: const Duration(milliseconds: 180),
                            child: KeyedSubtree(
                              key: ValueKey(_selectedSection),
                              child: _buildSectionContent(contentWidth),
                            ),
                          ),
                          const SizedBox(height: PokerSpacing.lg),
                          Align(
                            alignment: Alignment.centerRight,
                            child: FilledButton.icon(
                              key: const Key('settings-save-button'),
                              onPressed: _save,
                              icon: const Icon(Icons.save_outlined),
                              label: const Text('Save Changes'),
                              style: FilledButton.styleFrom(
                                backgroundColor: PokerColors.primary,
                                foregroundColor: PokerColors.textPrimary,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildSectionContent(double availableWidth) {
    switch (_selectedSection) {
      case _SettingsSection.general:
        return _buildGeneralSection(availableWidth);
      case _SettingsSection.ui:
        return _buildUiSection(availableWidth);
    }
  }

  Widget _buildGeneralSection(double availableWidth) {
    final twoColumn = availableWidth >= 860;
    final halfWidth = twoColumn
        ? ((availableWidth - PokerSpacing.lg - PokerSpacing.md) / 2)
            .clamp(260.0, 420.0)
            .toDouble()
        : availableWidth;

    return _SettingsPanel(
      key: const Key('settings-general-layout'),
      title: 'General',
      subtitle: 'Connection, payout, runtime, and local storage.',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const _SubsectionHeading(
            title: 'Connection',
            subtitle: 'Server endpoint and certificate used for the session.',
          ),
          Wrap(
            spacing: PokerSpacing.md,
            runSpacing: PokerSpacing.md,
            children: [
              _ControlSlot(
                width: halfWidth,
                child: _field(
                  _serverAddr,
                  'Server Address',
                  required: true,
                  helperText:
                      'Host and port of the poker gRPC service, for example 127.0.0.1:12345.',
                ),
              ),
              _ControlSlot(
                width: halfWidth,
                child: _field(
                  _debugLvl,
                  'Debug Level',
                  helperText:
                      'Typical values are info, debug, or trace. Subsystem filters are also allowed.',
                ),
              ),
            ],
          ),
          const SizedBox(height: PokerSpacing.md),
          _field(
            _grpcCert,
            'gRPC Server Cert Path',
            helperText:
                'Imported certificate used to validate the remote server.',
            suffixIcon: IconButton(
              tooltip: 'Import certificate',
              onPressed: _pickGrpcCert,
              icon: const Icon(
                Icons.folder_open_rounded,
                color: PokerColors.textSecondary,
              ),
            ),
          ),
          const Padding(
            padding: EdgeInsets.symmetric(vertical: PokerSpacing.lg),
            child: Divider(height: 1, color: PokerColors.borderSubtle),
          ),
          const _SubsectionHeading(
            title: 'Identity',
            subtitle: 'Where payouts are sent when funds are settled.',
          ),
          Wrap(
            spacing: PokerSpacing.md,
            runSpacing: PokerSpacing.md,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _ControlSlot(
                width: halfWidth,
                child: _field(
                  _nickname,
                  'Nickname',
                  helperText:
                      'Used for login. Leave blank to be prompted on next start.',
                ),
              ),
              _ControlSlot(
                width: halfWidth,
                child: _field(
                  _address,
                  'Payout Address or PubKey (33/65B hex)',
                  helperText:
                      'This value is editable and stored in the config file.',
                ),
              ),
              _ControlSlot(
                width: halfWidth,
                child: _ToggleSettingCard(
                  title: 'Enable Sounds',
                  subtitle:
                      'Play table feedback and turn cues during active sessions.',
                  value: _soundsEnabled,
                  onChanged: (value) {
                    setState(() => _soundsEnabled = value);
                  },
                ),
              ),
            ],
          ),
          const Padding(
            padding: EdgeInsets.symmetric(vertical: PokerSpacing.lg),
            child: Divider(height: 1, color: PokerColors.borderSubtle),
          ),
          const _SubsectionHeading(
            title: 'Storage',
            subtitle: 'Reference paths managed automatically by the app.',
          ),
          Wrap(
            spacing: PokerSpacing.md,
            runSpacing: PokerSpacing.sm,
            children: [
              _ControlSlot(
                width: halfWidth,
                child: _ReadOnlyInfoTile(
                  key: const Key('settings-storage-config-file'),
                  icon: Icons.description_outlined,
                  label: 'Config file',
                  value: _cfgPath,
                  actionLabel: 'Copy',
                  onAction: _cfgPath.isEmpty
                      ? null
                      : () => _copyValue('Config file path', _cfgPath),
                ),
              ),
              _ControlSlot(
                width: halfWidth,
                child: _ReadOnlyInfoTile(
                  key: const Key('settings-storage-data-dir'),
                  icon: Icons.folder_outlined,
                  label: 'Data directory',
                  value: _dataDir,
                  actionLabel: 'Copy',
                  onAction: _dataDir.isEmpty
                      ? null
                      : () => _copyValue('Data directory path', _dataDir),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildUiSection(double availableWidth) {
    const spacing = PokerSpacing.md;
    final columns = availableWidth >= 1080
        ? 3
        : availableWidth >= 720
            ? 2
            : 1;
    final controlWidth = columns == 1
        ? availableWidth
        : ((availableWidth - (spacing * (columns - 1))) / columns)
            .clamp(220.0, 360.0)
            .toDouble();

    return Column(
      key: const Key('settings-ui-layout'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _SettingsPanel(
          title: 'UI',
          subtitle: 'Appearance and sizing.',
          child: Wrap(
            key: const Key('settings-ui-controls-row'),
            spacing: spacing,
            runSpacing: spacing,
            children: [
              _buildUiDropdownControl(
                width: controlWidth,
                label: 'Table Theme',
                value: _tableTheme,
                items: TableThemeConfig.presets
                    .map(
                      (t) => DropdownMenuItem(
                        value: t.key,
                        child: Text(
                          t.displayName,
                          style: PokerTypography.bodyMedium,
                        ),
                      ),
                    )
                    .toList(),
                onChanged: (value) => setState(() => _tableTheme = value),
              ),
              _buildUiDropdownControl(
                width: controlWidth,
                label: 'Card Theme',
                value: _cardTheme,
                items: cardColorThemePresets
                    .map(
                      (t) => DropdownMenuItem(
                        value: cardColorThemeKey(t),
                        child: Text(
                          cardColorThemeDisplayName(t),
                          style: PokerTypography.bodyMedium,
                        ),
                      ),
                    )
                    .toList(),
                onChanged: (value) => setState(() => _cardTheme = value),
              ),
              _buildUiDropdownControl(
                width: controlWidth,
                label: 'Card Size',
                value: _cardSize,
                items: _sizeDropdownItems,
                onChanged: (value) => setState(() => _cardSize = value),
              ),
              _buildUiDropdownControl(
                width: controlWidth,
                label: 'UI Scale',
                value: _uiSize,
                items: _sizeDropdownItems,
                onChanged: (value) => setState(() => _uiSize = value),
              ),
              _ControlSlot(
                width: controlWidth,
                child: _ToggleSettingCard(
                  title: 'Show Table Logo',
                  value: !_hideTableLogo,
                  minHeight: 0,
                  onChanged: (value) {
                    setState(() => _hideTableLogo = !value);
                  },
                ),
              ),
              _buildUiDropdownControl(
                width: controlWidth,
                label: 'Logo Position',
                value: _logoPosition,
                enabled: !_hideTableLogo,
                style: PokerTypography.bodyMedium.copyWith(
                  color: _hideTableLogo
                      ? PokerColors.textMuted
                      : PokerColors.textPrimary,
                ),
                iconEnabledColor: _hideTableLogo
                    ? PokerColors.textMuted
                    : PokerColors.textSecondary,
                items: const [
                  DropdownMenuItem(
                    value: 'center',
                    child: Text('Center'),
                  ),
                  DropdownMenuItem(
                    value: 'top_left',
                    child: Text('Top Left'),
                  ),
                  DropdownMenuItem(
                    value: 'top_right',
                    child: Text('Top Right'),
                  ),
                  DropdownMenuItem(
                    value: 'bottom_left',
                    child: Text('Bottom Left'),
                  ),
                  DropdownMenuItem(
                    value: 'bottom_right',
                    child: Text('Bottom Right'),
                  ),
                ],
                onChanged: (value) => setState(() => _logoPosition = value),
              ),
            ],
          ),
        ),
        const SizedBox(height: PokerSpacing.lg),
        SettingsPokerPreview(settings: _draftUiSettings),
      ],
    );
  }

  Widget _buildUiDropdownControl({
    required double width,
    required String label,
    required String value,
    required List<DropdownMenuItem<String>> items,
    required ValueChanged<String> onChanged,
    bool enabled = true,
    TextStyle? style,
    Color? iconEnabledColor,
  }) {
    return _ControlSlot(
      width: width,
      child: _SimpleSettingControl(
        label: label,
        child: DropdownButtonFormField<String>(
          initialValue: value,
          isExpanded: true,
          dropdownColor: PokerColors.surface,
          style: style ?? PokerTypography.bodyMedium,
          iconEnabledColor: iconEnabledColor ?? PokerColors.textSecondary,
          decoration: _settingDropdownDecoration(enabled: enabled),
          items: items,
          onChanged: enabled
              ? (nextValue) {
                  if (nextValue != null) onChanged(nextValue);
                }
              : null,
        ),
      ),
    );
  }

  InputDecoration _settingDropdownDecoration({bool enabled = true}) =>
      InputDecoration(
        isDense: true,
        filled: true,
        fillColor: enabled ? PokerColors.surfaceBright : PokerColors.surfaceDim,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: PokerSpacing.md,
          vertical: PokerSpacing.md,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: PokerColors.borderSubtle),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: PokerColors.borderSubtle),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: PokerColors.primary, width: 1.4),
        ),
        disabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: PokerColors.borderSubtle),
        ),
      );

  InputDecoration _fieldDecoration(
    String label, {
    String? helperText,
    Widget? suffixIcon,
  }) {
    return InputDecoration(
      labelText: label,
      helperText: helperText,
      helperMaxLines: 2,
      suffixIcon: suffixIcon,
      filled: true,
      fillColor: PokerColors.surfaceDim,
      contentPadding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.md,
        vertical: PokerSpacing.md,
      ),
      labelStyle: PokerTypography.bodySmall.copyWith(
        color: PokerColors.textSecondary,
      ),
      helperStyle: PokerTypography.bodySmall.copyWith(
        color: PokerColors.textMuted,
      ),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: PokerColors.borderSubtle),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: PokerColors.borderSubtle),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: PokerColors.primary, width: 1.4),
      ),
      errorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: PokerColors.danger),
      ),
      focusedErrorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: PokerColors.danger, width: 1.4),
      ),
    );
  }

  List<DropdownMenuItem<String>> get _sizeDropdownItems => const [
        DropdownMenuItem(
          value: 'xs',
          child: Text('Extra Small'),
        ),
        DropdownMenuItem(
          value: 'small',
          child: Text('Small'),
        ),
        DropdownMenuItem(
          value: 'medium',
          child: Text('Medium'),
        ),
        DropdownMenuItem(
          value: 'large',
          child: Text('Large'),
        ),
        DropdownMenuItem(
          value: 'xl',
          child: Text('Extra Large'),
        ),
      ];

  Widget _field(
    TextEditingController controller,
    String label, {
    bool required = false,
    bool obscure = false,
    String? helperText,
    Widget? suffixIcon,
  }) {
    return TextFormField(
      controller: controller,
      obscureText: obscure,
      style: PokerTypography.bodyMedium,
      decoration: _fieldDecoration(
        label,
        helperText: helperText,
        suffixIcon: suffixIcon,
      ),
      validator:
          required ? (v) => v == null || v.isEmpty ? 'Required' : null : null,
    );
  }
}

enum _SettingsSection {
  general(
    key: 'general',
    label: 'General',
    icon: Icons.tune_rounded,
  ),
  ui(
    key: 'ui',
    label: 'UI',
    icon: Icons.style_outlined,
  );

  const _SettingsSection({
    required this.key,
    required this.label,
    required this.icon,
  });

  final String key;
  final String label;
  final IconData icon;
}

class _SettingsMenuRow extends StatelessWidget {
  const _SettingsMenuRow({
    required this.selected,
    required this.isWide,
    required this.onSelected,
  });

  final _SettingsSection selected;
  final bool isWide;
  final ValueChanged<_SettingsSection> onSelected;

  @override
  Widget build(BuildContext context) {
    const sections = _SettingsSection.values;
    final children = <Widget>[
      for (var index = 0; index < sections.length; index++) ...[
        _SettingsMenuButton(
          key: Key('settings-section-${sections[index].key}'),
          section: sections[index],
          selected: sections[index] == selected,
          onTap: () => onSelected(sections[index]),
        ),
        if (index < sections.length - 1) const SizedBox(width: PokerSpacing.sm),
      ],
    ];

    return isWide
        ? Row(mainAxisSize: MainAxisSize.min, children: children)
        : SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(children: children),
          );
  }
}

class _SettingsMenuButton extends StatelessWidget {
  const _SettingsMenuButton({
    super.key,
    required this.section,
    required this.selected,
    required this.onTap,
  });

  final _SettingsSection section;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final background = selected
        ? PokerColors.primary.withValues(alpha: 0.18)
        : PokerColors.surfaceDim.withValues(alpha: 0.92);
    final borderColor =
        selected ? PokerColors.primary : PokerColors.borderSubtle;
    final foreground =
        selected ? PokerColors.textPrimary : PokerColors.textSecondary;

    return Material(
      color: background,
      borderRadius: BorderRadius.circular(18),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onTap,
        mouseCursor: SystemMouseCursors.click,
        child: Container(
          padding: const EdgeInsets.symmetric(
            horizontal: PokerSpacing.lg,
            vertical: PokerSpacing.md,
          ),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(18),
            border: Border.all(color: borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(section.icon, size: 18, color: foreground),
              const SizedBox(width: PokerSpacing.sm),
              Text(
                section.label,
                style: PokerTypography.titleSmall.copyWith(
                  color: selected
                      ? PokerColors.textPrimary
                      : PokerColors.textSecondary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SettingsPanel extends StatelessWidget {
  const _SettingsPanel({
    super.key,
    required this.title,
    required this.subtitle,
    required this.child,
  });

  final String title;
  final String subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.lg),
      decoration: BoxDecoration(
        color: PokerColors.surface,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: PokerTypography.headlineMedium),
          const SizedBox(height: PokerSpacing.xs),
          Text(
            subtitle,
            style: PokerTypography.bodySmall.copyWith(
              color: PokerColors.textSecondary,
            ),
          ),
          const SizedBox(height: PokerSpacing.lg),
          child,
        ],
      ),
    );
  }
}

class _ControlSlot extends StatelessWidget {
  const _ControlSlot({
    required this.width,
    required this.child,
  });

  final double width;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return SizedBox(width: width, child: child);
  }
}

class _SimpleSettingControl extends StatelessWidget {
  const _SimpleSettingControl({
    required this.label,
    required this.child,
  });

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            label,
            style: PokerTypography.labelSmall.copyWith(
              color: PokerColors.textSecondary,
            ),
          ),
          const SizedBox(height: PokerSpacing.sm),
          child,
        ],
      ),
    );
  }
}

class _ToggleSettingCard extends StatelessWidget {
  const _ToggleSettingCard({
    required this.title,
    required this.value,
    required this.onChanged,
    this.subtitle,
    this.minHeight = 144,
  });

  final String title;
  final String? subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;
  final double minHeight;

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: BoxConstraints(minHeight: minHeight),
      padding: const EdgeInsets.all(PokerSpacing.md),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(title, style: PokerTypography.titleSmall),
              ),
              Switch(value: value, onChanged: onChanged),
            ],
          ),
          if (subtitle != null) ...[
            const SizedBox(height: PokerSpacing.xs),
            Text(
              subtitle!,
              style: PokerTypography.bodySmall.copyWith(
                color: PokerColors.textSecondary,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ReadOnlyInfoTile extends StatelessWidget {
  const _ReadOnlyInfoTile({
    super.key,
    required this.icon,
    required this.label,
    required this.value,
    required this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String label;
  final String value;
  final String actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: PokerSpacing.md,
        vertical: PokerSpacing.sm,
      ),
      decoration: BoxDecoration(
        color: PokerColors.surfaceDim,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: PokerColors.borderSubtle),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: PokerColors.textSecondary),
          const SizedBox(width: PokerSpacing.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: PokerTypography.bodySmall.copyWith(
                    color: PokerColors.textSecondary,
                  ),
                ),
                const SizedBox(height: PokerSpacing.xxs),
                Text(
                  value.isEmpty ? 'Loading…' : value,
                  style: PokerTypography.bodySmall.copyWith(
                    color: value.isEmpty
                        ? PokerColors.textMuted
                        : PokerColors.textPrimary,
                    fontFamily: 'monospace',
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: onAction,
            tooltip: actionLabel,
            icon: const Icon(
              Icons.copy_rounded,
              size: 18,
              color: PokerColors.textSecondary,
            ),
          ),
        ],
      ),
    );
  }
}

class _SubsectionHeading extends StatelessWidget {
  const _SubsectionHeading({
    required this.title,
    required this.subtitle,
  });

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: PokerSpacing.md),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: PokerTypography.titleMedium),
          const SizedBox(height: PokerSpacing.xs),
          Text(
            subtitle,
            style: PokerTypography.bodySmall.copyWith(
              color: PokerColors.textSecondary,
            ),
          ),
        ],
      ),
    );
  }
}
