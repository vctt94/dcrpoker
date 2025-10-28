import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class InLobbyView extends StatelessWidget {
  const InLobbyView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.table_restaurant, size: 64, color: Colors.white70),
          const SizedBox(height: 16),
          Text(
            'Table: ${model.currentTableId}',
            style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 8),
          Text(
            'State: ${model.state.name}',
            style: const TextStyle(color: Colors.white70),
          ),
          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton(
                onPressed: model.iAmReady ? model.setUnready : model.setReady,
                child: Text(model.iAmReady ? 'Unready' : 'Ready'),
              ),
              const SizedBox(width: 16),
              ElevatedButton(
                onPressed: model.leaveTable,
                style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                child: const Text('Leave Table'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
