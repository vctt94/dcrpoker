import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:pokerui/config.dart';
import 'package:pokerui/client_init.dart';

class LoginScreen extends StatefulWidget {
  final Config config;
  final Function(String nickname) onLoginSuccess;

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
    if (!_formKey.currentState!.validate()) {
      return;
    }

    if (!_clientInitialized) {
      await _initializeClient();
      if (!_clientInitialized) {
        return; // Error already shown
      }
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final nickname = _nicknameController.text.trim();

      // Try to login first
      try {
        final loginResp = await Golib.login(LoginRequest(nickname));
        // Login successful
        widget.onLoginSuccess(loginResp.nickname);
      } catch (loginError) {
        // If login fails with "nickname not found", try to register
        final errorMsg = loginError.toString().toLowerCase();
        if (errorMsg.contains('nickname not found')) {
          // Register the new user
          await Golib.register(RegisterRequest(nickname));
          // After registration, login
          final loginResp = await Golib.login(LoginRequest(nickname));
          widget.onLoginSuccess(loginResp.nickname);
        } else {
          // Other login errors
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
    if (value == null || value.trim().isEmpty) {
      return 'Nickname is required';
    }
    final nickname = value.trim();
    if (nickname.length < 3) {
      return 'Nickname must be at least 3 characters';
    }
    if (nickname.length > 32) {
      return 'Nickname must be at most 32 characters';
    }
    if (!RegExp(r'^[a-zA-Z0-9_-]+$').hasMatch(nickname)) {
      return 'Nickname can only contain letters, numbers, underscore, and hyphen';
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color.fromARGB(255, 25, 23, 44),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24.0),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Poker Game',
                  style: TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 48),
                TextFormField(
                  controller: _nicknameController,
                  decoration: const InputDecoration(
                    labelText: 'Nickname',
                    hintText: 'Enter your nickname',
                    border: OutlineInputBorder(),
                    filled: true,
                    fillColor: Colors.white,
                  ),
                  style: const TextStyle(color: Colors.black),
                  validator: _validateNickname,
                  enabled: !_isLoading,
                  textCapitalization: TextCapitalization.none,
                  autofocus: true,
                  onFieldSubmitted: (_) => _handleLogin(),
                ),
                const SizedBox(height: 12),
                const SizedBox(height: 16),
                if (_errorMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: Builder(builder: (context) {
                      final msg = _errorMessage ?? '';
                      return Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: SelectableText(
                              msg,
                              style: const TextStyle(
                                color: Colors.red,
                                fontSize: 14,
                              ),
                            ),
                          ),
                          IconButton(
                            icon: const Icon(Icons.copy, color: Colors.red),
                            tooltip: 'Copy error',
                            onPressed: () {
                              Clipboard.setData(ClipboardData(text: msg));
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('Error copied')),
                              );
                            },
                          ),
                        ],
                      );
                    }),
                  ),
                ElevatedButton(
                  onPressed: _isLoading ? null : _handleLogin,
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    backgroundColor: Colors.blueAccent,
                  ),
                  child: _isLoading
                      ? const SizedBox(
                          height: 20,
                          width: 20,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            valueColor:
                                AlwaysStoppedAnimation<Color>(Colors.white),
                          ),
                        )
                      : const Text(
                          'Login / Register',
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                            color: Colors.white,
                          ),
                        ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Enter your nickname to login or register a new account.',
                  style: TextStyle(
                    color: Colors.grey,
                    fontSize: 12,
                  ),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
