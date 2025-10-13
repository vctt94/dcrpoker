import 'package:flutter/material.dart';
import 'package:pokerui/models/notifications.dart';
import 'package:provider/provider.dart';

class NotificationBar extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Consumer<NotificationModel>(
      builder: (context, notificationModel, child) {
        if (notificationModel.notification.isEmpty) {
          return SizedBox.shrink(); // Hide when no notification
        }
        return Material(
          color: Colors.transparent,
          child: Container(
            width: double.infinity,
            color: Colors.blueAccent,
            padding: EdgeInsets.all(8.0),
            child: Text(
              notificationModel.notification,
              style: TextStyle(color: Colors.white),
              textAlign: TextAlign.center,
            ),
          ),
        );
      },
    );
  }
}
