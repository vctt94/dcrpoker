import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';

class CreateTableDialog extends StatefulWidget {
  const CreateTableDialog({super.key, required this.model});
  final PokerModel model;

  static Future<void> open(BuildContext context, PokerModel model) async {
    await showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF121420),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
        child: CreateTableDialog(model: model),
      ),
    );
  }

  @override
  State<CreateTableDialog> createState() => _CreateTableDialogState();
}

class _CreateTableDialogState extends State<CreateTableDialog> {
  final _form = GlobalKey<FormState>();

  int _maxPlayers = 2;
  String _buyInDcr = '0.0';
  int _startingChips = 1000;
  int _timeBankSeconds = 30;

  int _toAtoms(String dcrStr) {
    final v = double.tryParse(dcrStr.trim()) ?? 0.0;
    return (v * 1e8).round();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Form(
        key: _form,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.add, color: Colors.white70),
                const SizedBox(width: 8),
                const Text('Create Table', style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
                const Spacer(),
                IconButton(
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close, color: Colors.white70),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                _numberField(
                  label: 'Number of Players',
                  initial: _maxPlayers.toString(),
                  onSaved: (v) => _maxPlayers = int.parse(v!),
                ),
                _decimalField(
                  label: 'Buy-in (DCR)',
                  initial: _buyInDcr,
                  onSaved: (v) => _buyInDcr = v!,
                ),
                _numberField(
                  label: 'Starting Chips',
                  initial: _startingChips.toString(),
                  onSaved: (v) => _startingChips = int.parse(v!),
                ),
                _numberField(
                  label: 'Timebank (sec)',
                  initial: _timeBankSeconds.toString(),
                  onSaved: (v) => _timeBankSeconds = int.parse(v!),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('Cancel'),
                ),
                const SizedBox(width: 8),
                ElevatedButton.icon(
                  onPressed: _submit,
                  icon: const Icon(Icons.check),
                  label: const Text('Create'),
                  style: ElevatedButton.styleFrom(backgroundColor: Colors.blue),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _numberField({
    required String label,
    required String initial,
    required FormFieldSetter<String> onSaved,
  }) {
    return SizedBox(
      width: 220,
      child: TextFormField(
        initialValue: initial,
        keyboardType: TextInputType.number,
        style: const TextStyle(color: Colors.white),
        decoration: _decoration(label),
        validator: (v) => (v == null || v.isEmpty || int.tryParse(v) == null) ? 'Enter number' : null,
        onSaved: onSaved,
      ),
    );
  }

  Widget _decimalField({
    required String label,
    required String initial,
    required FormFieldSetter<String> onSaved,
  }) {
    return SizedBox(
      width: 220,
      child: TextFormField(
        initialValue: initial,
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
        style: const TextStyle(color: Colors.white),
        decoration: _decoration(label),
        validator: (v) => (v == null || v.isEmpty || double.tryParse(v) == null) ? 'Enter amount' : null,
        onSaved: onSaved,
      ),
    );
  }

  InputDecoration _decoration(String label) => InputDecoration(
        labelText: label,
        labelStyle: const TextStyle(color: Colors.white70),
        enabledBorder: OutlineInputBorder(
          borderSide: BorderSide(color: Colors.white.withOpacity(0.2)),
          borderRadius: BorderRadius.circular(8),
        ),
        focusedBorder: OutlineInputBorder(
          borderSide: const BorderSide(color: Colors.blue),
          borderRadius: BorderRadius.circular(8),
        ),
        filled: true,
        fillColor: const Color(0xFF1B1E2C),
      );

  Future<void> _submit() async {
    if (!_form.currentState!.validate()) return;
    _form.currentState!.save();

    final buyInAtoms = _toAtoms(_buyInDcr);

    final tid = await widget.model.createTable(
      maxPlayers: _maxPlayers,
      minPlayers: 2,
      buyInAtoms: buyInAtoms,
      startingChips: _startingChips,
      timeBankSeconds: _timeBankSeconds,
    );

    if (tid != null) {
      await widget.model.joinTable(tid);
    }

    if (mounted) Navigator.of(context).pop();
  }
}
