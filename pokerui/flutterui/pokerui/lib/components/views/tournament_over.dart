import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class TournamentOverView extends StatelessWidget {
  const TournamentOverView({super.key, required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.flag, size: 64, color: Colors.green),
          const SizedBox(height: 16),
          const Text(
            'Tournament Over!',
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.white),
          ),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () {
              model.leaveTable();
            },
            child: const Text('Return to Lobby'),
          ),
        ],
      ),
    );
  }
}
