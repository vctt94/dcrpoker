import 'package:pokerui/models/shutdown.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:window_manager/window_manager.dart';
import 'dart:io';

// Fatal error screen is for errors that are very fatal: no access to anything
// (not even logs). Just option to quit app.
class FatalErrorScreen extends StatelessWidget {
  final Object? exception;
  const FatalErrorScreen({this.exception, super.key});

  @override
  Widget build(BuildContext context) {
    var exc = exception ??
        ModalRoute.of(context)?.settings.arguments ??
        Exception("unknown exception");

    return Scaffold(
        body: Container(
            color: Colors.red,
            width: double.infinity,
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.center,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  SelectionArea(
                      child: Text("Fatal error: $exc",
                          style: TextStyle(color: Colors.grey[300]))),
                  const SizedBox(height: 20),
                  const FilledButton(
                      onPressed: forceQuitApp, child: Text("Force Quit App")),
                ])));
  }
}

void runFatalErrorApp(Object exception) {
  runApp(MaterialApp(
    title: "Fatal Error",
    initialRoute: "/",
    routes: {
      "/": (context) => FatalErrorScreen(exception: exception),
    },
  ));
}

// forceQuitApp bypasses the standard shutdown procedure.
void forceQuitApp() {
  if (Platform.isAndroid || Platform.isIOS) {
    SystemNavigator.pop();
  } else {
    windowManager.destroy();
  }
}
