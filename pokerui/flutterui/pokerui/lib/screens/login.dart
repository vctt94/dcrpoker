import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/client_init.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';
import 'package:pokerui/theme/spacing.dart';

class LoginScreen extends StatefulWidget {
  final Config config;
  final Function(LoginResponse resp) onLoginSuccess;

  const LoginScreen(
      {super.key, required this.config, required this.onLoginSuccess});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _nicknameController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isLoading = false;
  bool _clientInitialized = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _initializeClient();
  }

  @override
  void dispose() {
    _nicknameController.dispose();
    super.dispose();
  }

  Future<void> _initializeClient() async {
    if (_clientInitialized) return;
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });
    try {
      await initializePokerClient(widget.config);
      setState(() {
        _clientInitialized = true;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _errorMessage = 'Failed to initialize client: $e';
        _isLoading = false;
      });
    }
  }

  Future<void> _handleLogin() async {
    if (!_formKey.currentState!.validate()) return;

    if (!_clientInitialized) {
      await _initializeClient();
      if (!_clientInitialized) return;
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final nickname = _nicknameController.text.trim();
      try {
        final loginResp = await Golib.login(LoginRequest(nickname));
        widget.onLoginSuccess(loginResp);
      } catch (loginError) {
        final errorMsg = loginError.toString().toLowerCase();
        if (errorMsg.contains('nickname not found')) {
          await Golib.register(RegisterRequest(nickname));
          final loginResp = await Golib.login(LoginRequest(nickname));
          widget.onLoginSuccess(loginResp);
        } else {
          setState(() {
            _errorMessage = loginError.toString();
            _isLoading = false;
          });
        }
      }
    } catch (e) {
      setState(() {
        _errorMessage = e.toString();
        _isLoading = false;
      });
    }
  }

  String? _validateNickname(String? value) {
    if (value == null || value.trim().isEmpty) return 'Nickname is required';
    final nickname = value.trim();
    if (nickname.length < 3) return 'At least 3 characters';
    if (nickname.length > 32) return 'At most 32 characters';
    if (!RegExp(r'^[a-zA-Z0-9_-]+$').hasMatch(nickname)) {
      return 'Letters, numbers, underscore, hyphen only';
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: PokerColors.screenBg,
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              PokerColors.primary.withOpacity(0.08),
              PokerColors.screenBg,
              PokerColors.accent.withOpacity(0.04),
            ],
            stops: const [0.0, 0.5, 1.0],
          ),
        ),
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(PokerSpacing.xl),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: Form(
                key: _formKey,
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // Logo area
                    Icon(
                      Icons.style,
                      size: 64,
                      color: PokerColors.primary.withOpacity(0.8),
                    ),
                    const SizedBox(height: PokerSpacing.lg),
                    Text(
                      'Decred Poker',
                      style: PokerTypography.displayLarge.copyWith(
                        letterSpacing: -1.0,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: PokerSpacing.sm),
                    Text(
                      'Trustless poker on Bison Relay',
                      style: PokerTypography.bodySmall,
                      textAlign: TextAlign.center,
                    ),

                    const SizedBox(height: PokerSpacing.xxxl),

                    // Login card
                    Container(
                      padding: const EdgeInsets.all(PokerSpacing.xl),
                      decoration: BoxDecoration(
                        color: PokerColors.surface,
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: PokerColors.borderSubtle),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Text('Enter your nickname',
                            style: PokerTypography.titleSmall.copyWith(
                              color: PokerColors.textSecondary,
                            ),
                          ),
                          const SizedBox(height: PokerSpacing.md),
                          TextFormField(
                            controller: _nicknameController,
                            style: PokerTypography.bodyLarge,
                            decoration: const InputDecoration(
                              hintText: 'Nickname',
                              prefixIcon: Icon(Icons.person_outline,
                                  color: PokerColors.textMuted, size: 20),
                            ),
                            validator: _validateNickname,
                            enabled: !_isLoading,
                            textCapitalization: TextCapitalization.none,
                            autofocus: true,
                            onFieldSubmitted: (_) => _handleLogin(),
                          ),

                          if (_errorMessage != null) ...[
                            const SizedBox(height: PokerSpacing.md),
                            Container(
                              padding: const EdgeInsets.all(PokerSpacing.md),
                              decoration: BoxDecoration(
                                color: PokerColors.danger.withOpacity(0.1),
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Row(
                                children: [
                                  Expanded(
                                    child: SelectableText(
                                      _errorMessage!,
                                      style: PokerTypography.bodySmall.copyWith(
                                        color: PokerColors.danger,
                                      ),
                                    ),
                                  ),
                                  IconButton(
                                    icon: Icon(Icons.copy,
                                        color: PokerColors.danger, size: 16),
                                    onPressed: () {
                                      Clipboard.setData(
                                          ClipboardData(text: _errorMessage!));
                                      ScaffoldMessenger.of(context).showSnackBar(
                                          const SnackBar(content: Text('Copied')));
                                    },
                                    padding: EdgeInsets.zero,
                                    constraints: const BoxConstraints(),
                                  ),
                                ],
                              ),
                            ),
                          ],

                          const SizedBox(height: PokerSpacing.lg),
                          SizedBox(
                            height: 48,
                            child: ElevatedButton(
                              onPressed: _isLoading ? null : _handleLogin,
                              style: ElevatedButton.styleFrom(
                                backgroundColor: PokerColors.primary,
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(12),
                                ),
                              ),
                              child: _isLoading
                                  ? const SizedBox(
                                      width: 20, height: 20,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                        color: Colors.white,
                                      ),
                                    )
                                  : Text('Continue',
                                      style: PokerTypography.labelLarge
                                          .copyWith(color: Colors.white)),
                            ),
                          ),
                        ],
                      ),
                    ),

                    const SizedBox(height: PokerSpacing.lg),
                    Text(
                      'New nicknames are auto-registered.',
                      style: PokerTypography.bodySmall.copyWith(
                        color: PokerColors.textMuted,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
