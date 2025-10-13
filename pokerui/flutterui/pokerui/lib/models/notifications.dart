import 'dart:async';

import 'package:flutter/material.dart';

class NotificationModel extends ChangeNotifier {
  String _notification = '';
  Timer? _notificationTimer;

  String get notification => _notification;

  void showNotification(String message, {int durationSeconds = 5}) {
    _notification = message;
    notifyListeners();

    _notificationTimer?.cancel();
    _notificationTimer = Timer(Duration(seconds: durationSeconds), () {
      hideNotification();
    });
  }

  void hideNotification() {
    _notification = '';
    _notificationTimer?.cancel();
    notifyListeners();
  }
}
