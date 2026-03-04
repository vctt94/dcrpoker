import 'package:flutter/material.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/cards.dart';
import 'package:pokerui/components/poker/responsive.dart';
import 'package:pokerui/models/poker.dart';

/// Styled action button used inside the [BottomActionDock].
class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.label,
    required this.onPressed,
    required this.color,
  });

  final String label;
  final VoidCallback? onPressed;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final scale = buttonScale(bp);
    return ElevatedButton(
      onPressed: onPressed,
      style: ElevatedButton.styleFrom(
        backgroundColor: color,
        foregroundColor: Colors.white,
        padding: EdgeInsets.symmetric(
          horizontal: 18 * scale,
          vertical: 10 * scale,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24 * scale),
        ),
        elevation: 4,
        shadowColor: color.withOpacity(0.4),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 14 * scale,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.3,
        ),
      ),
    );
  }
}

/// Bottom action dock: Fold / Call / Raise buttons, bet input, waiting state.
///
/// This is a normal layout widget (not a Positioned overlay) designed to sit at
/// the bottom of a Column below the table canvas.
class BottomActionDock extends StatelessWidget {
  const BottomActionDock({
    super.key,
    required this.model,
    required this.showBetInput,
    required this.betCtrl,
    required this.onToggleBetInput,
    required this.onCloseBetInput,
    this.hasLastShowdown = false,
    this.showSidebar = false,
    this.onToggleSidebar,
  });

  final PokerModel model;
  final bool showBetInput;
  final TextEditingController betCtrl;
  final VoidCallback onToggleBetInput;
  final VoidCallback onCloseBetInput;
  final bool hasLastShowdown;
  final bool showSidebar;
  final VoidCallback? onToggleSidebar;

  int _resolveBigBlind() {
    if ((model.game?.bigBlind ?? 0) > 0) return model.game!.bigBlind;
    final tid = model.currentTableId;
    if (tid == null) return 0;
    final matches = model.tables.where((t) => t.id == tid).toList();
    return matches.isNotEmpty ? matches.first.bigBlind : 0;
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final canAct = model.canAct;

    return Container(
      padding: EdgeInsets.only(
        left: 12,
        right: 12,
        top: 8,
        bottom: safeBottomPadding(context, minPadding: 8),
      ),
      constraints: BoxConstraints(minHeight: actionDockMinHeight(bp)),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            const Color(0xFF121212).withOpacity(0.0),
            const Color(0xFF121212).withOpacity(0.95),
          ],
        ),
      ),
      child: Row(
        children: [
          if (hasLastShowdown && onToggleSidebar != null)
            _LastHandButton(
              active: showSidebar,
              onTap: onToggleSidebar!,
            ),
          if (hasLastShowdown && onToggleSidebar != null)
            const SizedBox(width: 8),
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: canAct
                  ? _ActionButtons(
                      model: model,
                      showBetInput: showBetInput,
                      betCtrl: betCtrl,
                      onToggleBetInput: onToggleBetInput,
                      onCloseBetInput: onCloseBetInput,
                      bb: _resolveBigBlind(),
                    )
                  : _WaitingIndicator(model: model),
            ),
          ),
        ],
      ),
    );
  }
}

/// Mobile-only bottom panel that separates hero cards from the table canvas.
class MobileHeroActionPanel extends StatelessWidget {
  const MobileHeroActionPanel({
    super.key,
    required this.model,
    this.showBetInput = false,
    this.betCtrl,
    this.onToggleBetInput,
    this.onCloseBetInput,
    this.hasLastShowdown = false,
    this.showSidebar = false,
    this.onToggleSidebar,
    this.showActions = true,
    this.footer,
  }) : assert(
          !showActions ||
              (betCtrl != null &&
                  onToggleBetInput != null &&
                  onCloseBetInput != null),
        );

  final PokerModel model;
  final bool showBetInput;
  final TextEditingController? betCtrl;
  final VoidCallback? onToggleBetInput;
  final VoidCallback? onCloseBetInput;
  final bool hasLastShowdown;
  final bool showSidebar;
  final VoidCallback? onToggleSidebar;
  final bool showActions;
  final Widget? footer;

  int _resolveBigBlind() {
    if ((model.game?.bigBlind ?? 0) > 0) return model.game!.bigBlind;
    final tid = model.currentTableId;
    if (tid == null) return 0;
    final matches = model.tables.where((t) => t.id == tid).toList();
    return matches.isNotEmpty ? matches.first.bigBlind : 0;
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final canAct = model.canAct;
    final me = model.me;
    final isShowing = me?.cardsRevealed ?? false;
    final cards =
        (me?.hand.isNotEmpty ?? false) ? me!.hand : model.myHoleCardsCache;
    final hasCards = cards.isNotEmpty;

    return Container(
      constraints: BoxConstraints(minHeight: mobileHeroPanelMinHeight(bp)),
      padding: EdgeInsets.only(
        left: 8,
        right: 8,
        top: 6,
        bottom: safeBottomPadding(context, minPadding: 6),
      ),
      decoration: const BoxDecoration(
        color: Color(0xFF121212),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              _CompactHeroCards(cards: cards),
              const SizedBox(width: 6),
              GestureDetector(
                onTap: !hasCards
                    ? null
                    : () {
                        if (isShowing) {
                          model.hideCards();
                        } else {
                          model.showCards();
                        }
                      },
                child: Icon(
                  isShowing ? Icons.visibility : Icons.visibility_off,
                  size: 18,
                  color: isShowing ? Colors.amber : Colors.white38,
                ),
              ),
              const Spacer(),
              if (hasLastShowdown && onToggleSidebar != null)
                _LastHandButton(
                  active: showSidebar,
                  onTap: onToggleSidebar!,
                ),
            ],
          ),
          if (showActions) ...[
            const SizedBox(height: 6),
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: canAct
                  ? _ActionButtons(
                      model: model,
                      showBetInput: showBetInput,
                      betCtrl: betCtrl!,
                      onToggleBetInput: onToggleBetInput!,
                      onCloseBetInput: onCloseBetInput!,
                      bb: _resolveBigBlind(),
                    )
                  : _WaitingIndicator(model: model),
            ),
          ],
          if (footer != null) ...[
            const SizedBox(height: 6),
            footer!,
          ],
        ],
      ),
    );
  }
}

class _CompactHeroCards extends StatelessWidget {
  const _CompactHeroCards({required this.cards});
  final List<pr.Card> cards;

  @override
  Widget build(BuildContext context) {
    const cw = 42.0;
    const ch = cw * 1.4;
    const gap = 6.0;

    Widget buildCard(int index) {
      if (cards.length > index) return CardFace(card: cards[index]);
      return const CardBack();
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(width: cw, height: ch, child: buildCard(0)),
        const SizedBox(width: gap),
        SizedBox(width: cw, height: ch, child: buildCard(1)),
      ],
    );
  }
}

class _LastHandButton extends StatelessWidget {
  const _LastHandButton({required this.active, required this.onTap});
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final accent = active ? Colors.amber : Colors.white70;
    return Tooltip(
      message: 'View last showdown',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            decoration: BoxDecoration(
              color: Colors.black.withOpacity(0.5),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.white.withOpacity(0.2)),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.history, color: accent, size: 16),
                const SizedBox(width: 5),
                Text('Last Hand',
                    style: TextStyle(color: accent, fontSize: 12)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _WaitingIndicator extends StatelessWidget {
  const _WaitingIndicator({required this.model});
  final PokerModel model;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.6),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        model.autoAdvanceAllIn ? 'Auto-advancing (all-in)' : 'Waiting...',
        style: const TextStyle(color: Colors.white, fontSize: 14),
      ),
    );
  }
}

class _ActionButtons extends StatelessWidget {
  const _ActionButtons({
    required this.model,
    required this.showBetInput,
    required this.betCtrl,
    required this.onToggleBetInput,
    required this.onCloseBetInput,
    required this.bb,
  });

  final PokerModel model;
  final bool showBetInput;
  final TextEditingController betCtrl;
  final VoidCallback onToggleBetInput;
  final VoidCallback onCloseBetInput;
  final int bb;

  @override
  Widget build(BuildContext context) {
    final g = model.game;
    final me = model.me;
    final currentBet = g?.currentBet ?? 0;
    final myBet = me?.currentBet ?? 0;
    final canCheck = myBet >= currentBet;
    final toCall = (currentBet - myBet) > 0 ? (currentBet - myBet) : 0;
    final isRaise = currentBet > 0 && myBet < currentBet;
    final myBalance = me?.balance ?? 0;
    final wouldBeAllIn = myBalance > 0 && myBalance <= (currentBet - myBet);

    if (showBetInput) {
      return _BetInputRow(
        model: model,
        betCtrl: betCtrl,
        currentBet: currentBet,
        myBet: myBet,
        bb: bb,
        isRaise: isRaise,
        onClose: onCloseBetInput,
      );
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _ActionButton(
          label: 'Fold (F)',
          onPressed: model.fold,
          color: const Color(0xFFD32F2F),
        ),
        const SizedBox(width: 8),
        if (canCheck)
          _ActionButton(
            label: 'Check (K)',
            onPressed: model.check,
            color: const Color(0xFF37474F),
          )
        else
          _ActionButton(
            label: 'Call${toCall > 0 ? ' ($toCall)' : ''} (C)',
            onPressed: model.callBet,
            color: const Color(0xFF37474F),
          ),
        const SizedBox(width: 8),
        _ActionButton(
          label: isRaise ? (wouldBeAllIn ? 'All-in' : 'Raise') : 'Bet',
          onPressed: () {
            if (betCtrl.text.isEmpty) _seedDefault(betCtrl, bb, currentBet);
            onToggleBetInput();
          },
          color: const Color(0xFF2E7D32),
        ),
      ],
    );
  }

  static void _seedDefault(TextEditingController ctrl, int bb, int currentBet) {
    final threeBB = bb * 3;
    final target =
        (bb > 0 && currentBet >= threeBB) ? (currentBet * 3) : threeBB;
    ctrl.text = target.toString();
  }
}

class _BetInputRow extends StatelessWidget {
  const _BetInputRow({
    required this.model,
    required this.betCtrl,
    required this.currentBet,
    required this.myBet,
    required this.bb,
    required this.isRaise,
    required this.onClose,
  });

  final PokerModel model;
  final TextEditingController betCtrl;
  final int currentBet;
  final int myBet;
  final int bb;
  final bool isRaise;
  final VoidCallback onClose;

  Future<void> _submitBet(BuildContext context) async {
    final raw = betCtrl.text.trim();
    final amt = int.tryParse(raw) ?? 0;
    if (amt <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Enter a valid bet amount')),
      );
      return;
    }

    if (currentBet > 0 && amt < currentBet) {
      final minRaise = currentBet - myBet;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
            content: Text(
                'Must add at least $minRaise to call ($currentBet total)')),
      );
      return;
    }

    final ok = await model.makeBet(amt);
    if (!ok && model.errorMessage.isNotEmpty && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(model.errorMessage)),
      );
      return;
    }
    onClose();
  }

  @override
  Widget build(BuildContext context) {
    final bp = PokerBreakpointQuery.of(context);
    final scale = buttonScale(bp);

    void setTo3xBB() {
      final threeBB = bb * 3;
      final target =
          (bb > 0 && currentBet >= threeBB) ? (currentBet * 3) : threeBB;
      betCtrl.text = target.toString();
    }

    final threeBB = bb * 3;
    final presetLabel = (bb > 0 && currentBet >= threeBB) ? '3x Bet' : '3x BB';

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox(
              width: 110 * scale,
              child: TextField(
                controller: betCtrl,
                keyboardType: TextInputType.number,
                style: const TextStyle(color: Colors.white),
                decoration: InputDecoration(
                  labelText: isRaise ? 'Total raise' : 'Total bet',
                  labelStyle:
                      const TextStyle(color: Colors.white70, fontSize: 12),
                  isDense: true,
                  contentPadding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                  hintText: isRaise
                      ? 'e.g. ${currentBet > 0 ? currentBet : bb}'
                      : 'e.g. ${bb > 0 ? bb * 3 : 50}',
                  hintStyle: const TextStyle(color: Colors.white54),
                  filled: true,
                  fillColor: Colors.black54,
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: Colors.white24),
                  ),
                ),
                onSubmitted: (_) => _submitBet(context),
              ),
            ),
            const SizedBox(height: 4),
            _BetDeltaLabel(
              betCtrl: betCtrl,
              myBet: myBet,
              myBalance: model.me?.balance ?? 0,
              currentBet: currentBet,
              bb: bb,
              isRaise: isRaise,
            ),
          ],
        ),
        const SizedBox(width: 6),
        _ActionButton(
          label: presetLabel,
          onPressed: bb > 0 ? setTo3xBB : null,
          color: const Color(0xFF37474F),
        ),
        const SizedBox(width: 6),
        Builder(builder: (context) {
          final meBal = model.me?.balance ?? 0;
          final entered = int.tryParse(betCtrl.text.trim()) ?? 0;
          final target = entered > 0 ? entered : currentBet;
          final myTotal = meBal + myBet;
          final isAllIn = target >= myTotal && myTotal > 0;
          final label = isAllIn ? 'All-in' : (isRaise ? 'Raise' : 'Bet');
          return _ActionButton(
            label: label,
            onPressed: () => _submitBet(context),
            color: const Color(0xFF2E7D32),
          );
        }),
        const SizedBox(width: 6),
        TextButton(
          onPressed: onClose,
          child: const Text('Cancel', style: TextStyle(color: Colors.white70)),
        ),
      ],
    );
  }
}

class _BetDeltaLabel extends StatelessWidget {
  const _BetDeltaLabel({
    required this.betCtrl,
    required this.myBet,
    required this.myBalance,
    required this.currentBet,
    required this.bb,
    required this.isRaise,
  });

  final TextEditingController betCtrl;
  final int myBet;
  final int myBalance;
  final int currentBet;
  final int bb;
  final bool isRaise;

  @override
  Widget build(BuildContext context) {
    final entered = int.tryParse(betCtrl.text.trim()) ?? 0;
    final maxTotal = myBalance + myBet;
    final capped = entered > maxTotal ? maxTotal : entered;
    final displayEntered =
        capped > 0 ? capped : (isRaise ? currentBet : bb * 3);
    final displayDelta = displayEntered > myBet ? (displayEntered - myBet) : 0;
    if (displayDelta == displayEntered) return const SizedBox.shrink();
    final isAllIn = displayEntered == maxTotal && maxTotal > 0;
    final label = isAllIn
        ? 'All-in $displayEntered'
        : 'Adds $displayDelta, total $displayEntered';
    return Text(label,
        style: const TextStyle(color: Colors.white70, fontSize: 11));
  }
}
