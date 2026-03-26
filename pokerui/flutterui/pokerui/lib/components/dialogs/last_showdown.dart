import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/showdown_content.dart';
import 'package:pokerui/models/poker.dart';

class LastShowdownDialog extends StatelessWidget {
  const LastShowdownDialog({super.key, required this.model});

  final PokerModel model;

  static Future<void> show(BuildContext context, PokerModel model) {
    return showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.7),
      builder: (ctx) => LastShowdownDialog(model: model),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
      backgroundColor: Colors.transparent,
      insetPadding: const EdgeInsets.all(16),
      child: Container(
        key: const Key('last-showdown-dialog'),
        constraints: const BoxConstraints(maxWidth: 500, maxHeight: 600),
        decoration: BoxDecoration(
          color: const Color(0xFF1A1D2E),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: Colors.amber.withValues(alpha: 0.5),
            width: 2,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.5),
              blurRadius: 20,
              spreadRadius: 5,
            ),
          ],
        ),
        child: ShowdownContent(
          model: model,
          showHeader: true,
          showCloseButton: true,
          onClose: () => Navigator.of(context).pop(),
        ),
      ),
    );
  }
}
