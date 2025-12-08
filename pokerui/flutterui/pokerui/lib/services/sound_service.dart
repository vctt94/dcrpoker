import 'package:audioplayers/audioplayers.dart';
import 'dart:developer' as developer;

/// Service for playing poker game sound effects.
class SoundService {
  static final SoundService _instance = SoundService._internal();
  factory SoundService() => _instance;
  SoundService._internal();

  final AudioPlayer _player = AudioPlayer();
  bool _enabled = true;
  bool _audioAvailable = true; // Track if audio system is working

  /// Enable or disable sound effects
  void setEnabled(bool enabled) {
    _enabled = enabled;
  }

  bool get enabled => _enabled;
  bool get audioAvailable => _audioAvailable;

  /// Play the turn notification sound when it becomes the player's turn
  Future<void> playTurnNotification() async {
    if (!_enabled || !_audioAvailable) return;
    try {
      // await _player.play(AssetSource('sounds/your-turn-warning-2.wav'));
      await _player.play(AssetSource('sounds/your-turn.mp3'));
    } catch (e) {
      // If audio fails, disable it to prevent repeated errors
      // This is common on Linux if GStreamer plugins are missing
      _audioAvailable = false;
      developer.log(
        'Sound playback failed (audio may be unavailable): $e',
        name: 'SoundService',
        level: 1000, // Use high level to reduce noise in logs
      );
    }
  }

  /// Play a check sound
  Future<void> playCheck() async {
    if (!_enabled || !_audioAvailable) return;
    try {
      await _player.stop();
      await _player.play(AssetSource('sounds/check.mp3'));
    } catch (e) {
      _audioAvailable = false;
      developer.log(
        'Sound playback failed (audio may be unavailable): $e',
        name: 'SoundService',
        level: 1000,
      );
    }
  }

  /// Play a bet sound
  Future<void> playBet() async {
    if (!_enabled || !_audioAvailable) return;
    try {
      // Stop any currently playing sound to ensure the bet sound plays
      await _player.stop();
      await _player.play(AssetSource('sounds/bet.mp3'));
    } catch (e) {
      _audioAvailable = false;
      developer.log(
        'Sound playback failed (audio may be unavailable): $e',
        name: 'SoundService',
        level: 1000,
      );
    }
  }

  /// Play a call sound (placeholder for future enhancement)
  Future<void> playCall() async {
    if (!_enabled || !_audioAvailable) return;
    try {
      // Stop any currently playing sound to ensure the bet sound plays
      await _player.stop();
      await _player.play(AssetSource('sounds/bet.mp3'));
    } catch (e) {
      _audioAvailable = false;
      developer.log(
        'Sound playback failed (audio may be unavailable): $e',
        name: 'SoundService',
        level: 1000,
      );
    }
  }

  void dispose() {
    _player.dispose();
  }
}

