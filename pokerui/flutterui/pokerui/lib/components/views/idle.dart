import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class IdleView extends StatelessWidget {
  const IdleView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.casino, size: 64, color: Colors.white70),
          const SizedBox(height: 16),
          const Text(
            'Welcome to Poker!',
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 8),
          const Text(
            'Connect to a poker server to start playing',
            style: TextStyle(color: Colors.white70),
          ),
          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton.icon(
                onPressed: () {
                  model.refreshTables();
                },
                icon: const Icon(Icons.refresh),
                label: const Text('Connect & Refresh'),
                style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
              ),
              const SizedBox(width: 16),
              ElevatedButton.icon(
                onPressed: () {
                  // TODO: Implement create table functionality
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Create table functionality coming soon')),
                  );
                },
                icon: const Icon(Icons.add),
                label: const Text('Create Table'),
                style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
