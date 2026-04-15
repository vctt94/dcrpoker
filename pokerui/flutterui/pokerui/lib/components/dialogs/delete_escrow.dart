import 'package:flutter/material.dart';

class DeleteEscrowDialog extends StatefulWidget {
  const DeleteEscrowDialog({super.key});

  static Future<bool> show(BuildContext context) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => const DeleteEscrowDialog(),
    );
    return confirmed == true;
  }

  @override
  State<DeleteEscrowDialog> createState() => _DeleteEscrowDialogState();
}

class _DeleteEscrowDialogState extends State<DeleteEscrowDialog> {
  final TextEditingController _confirmCtrl = TextEditingController();

  bool get _isConfirmed => _confirmCtrl.text.trim().toLowerCase() == 'ok';

  @override
  void dispose() {
    _confirmCtrl.dispose();
    super.dispose();
  }

  void _submit() {
    Navigator.of(context).pop(_isConfirmed);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Delete escrow record?'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Are you absolutely sure? Deleting this escrow entry removes the '
            'local record used for refunds. If the refund has not been '
            'recovered yet, the funds may be PERMANENTLY LOST.',
          ),
          const SizedBox(height: 16),
          const Text(
            'Type OK to confirm deletion.',
            style: TextStyle(fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _confirmCtrl,
            autofocus: true,
            decoration: const InputDecoration(
              labelText: 'Confirmation',
              hintText: 'OK',
            ),
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _submit(),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: const Text('Cancel'),
        ),
        TextButton(
          onPressed: _submit,
          child: const Text('Delete'),
        ),
      ],
    );
  }
}
