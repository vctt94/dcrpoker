import 'package:flutter/material.dart';
import 'package:pokerui/components/table.dart';

/// Dedicated screen for the poker table game
class PokerTableScreen extends StatelessWidget {
  const PokerTableScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: Colors.white),
          onPressed: () => Navigator.pop(context),
        ),
        title: const Text(
          'Poker Table',
          style: TextStyle(color: Colors.white),
        ),
        centerTitle: true,
      ),
      body: const SafeArea(
        child: PokerTable(),
      ),
    );
  }
}
