import 'package:flutter/material.dart';
import 'package:pokerui/components/poker/scene_layout.dart';
import 'package:pokerui/theme/colors.dart';
import 'package:pokerui/theme/typography.dart';

/// Floating label above the community cards during showdown.
class ShowdownBoardLabel extends StatelessWidget {
  const ShowdownBoardLabel({
    super.key,
    required this.text,
    required this.scene,
    required this.compact,
  });

  final String text;
  final PokerSceneLayout scene;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final maxWidth = compact ? scene.contentRect.width * 0.64 : 360.0;
    return Positioned(
      left: scene.communityRect.center.dx,
      top: scene.communityRect.top - (compact ? 6.0 : 10.0),
      child: FractionalTranslation(
        translation: const Offset(-0.5, -1.0),
        child: ConstrainedBox(
          constraints: BoxConstraints(maxWidth: maxWidth),
          child: Container(
            key: const Key('showdown-board-label'),
            padding: EdgeInsets.symmetric(
              horizontal: compact ? 12.0 : 14.0,
              vertical: compact ? 7.0 : 8.0,
            ),
            decoration: BoxDecoration(
              color: Colors.black.withValues(alpha: 0.76),
              borderRadius: BorderRadius.circular(14),
              border: Border.all(
                color: PokerColors.borderSubtle.withValues(alpha: 0.9),
              ),
            ),
            child: Text(
              text,
              textAlign: TextAlign.center,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: PokerTypography.labelLarge.copyWith(
                color: PokerColors.textPrimary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
