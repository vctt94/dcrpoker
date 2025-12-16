import 'package:flutter/material.dart';
import 'table.dart';

class TableLogoOverlay extends StatelessWidget {
  const TableLogoOverlay({
    super.key,
    required this.logoPosition,
    required this.uiSizeMultiplier,
  });
  final String logoPosition;
  final double uiSizeMultiplier;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final size = c.biggest;
      final layout = resolveTableLayout(size);
      final tableCenterX = layout.center.dx;
      final tableCenterY = layout.center.dy;
      final tableRadiusX = layout.tableRadiusX;
      final tableRadiusY = layout.tableRadiusY;
      
      // Logo size based on table size
      final logoSize = (tableRadiusX * 0.25 * uiSizeMultiplier).clamp(40.0, 100.0);
      
      // Calculate position within the table ellipse based on logoPosition config
      // Use a fraction of the table radius to keep logo well within table bounds
      double left, top;
      final pos = logoPosition.toLowerCase();
      switch (pos) {
        case 'top_center':
          left = tableCenterX - logoSize / 2;
          top = tableCenterY - tableRadiusY * 0.6 - logoSize / 2;
          break;
        case 'bottom_center':
          left = tableCenterX - logoSize / 2;
          top = tableCenterY + tableRadiusY * 0.6 - logoSize / 2;
          break;
        case 'left_center':
          left = tableCenterX - tableRadiusX * 0.6 - logoSize / 2;
          top = tableCenterY - logoSize / 2;
          break;
        case 'right_center':
          left = tableCenterX + tableRadiusX * 0.6 - logoSize / 2;
          top = tableCenterY - logoSize / 2;
          break;
        case 'top_left':
          left = tableCenterX - tableRadiusX * 0.5 - logoSize / 2;
          top = tableCenterY - tableRadiusY * 0.5 - logoSize / 2;
          break;
        case 'top_right':
          left = tableCenterX + tableRadiusX * 0.5 - logoSize / 2;
          top = tableCenterY - tableRadiusY * 0.5 - logoSize / 2;
          break;
        case 'bottom_left':
          left = tableCenterX - tableRadiusX * 0.5 - logoSize / 2;
          top = tableCenterY + tableRadiusY * 0.5 - logoSize / 2;
          break;
        case 'bottom_right':
          left = tableCenterX + tableRadiusX * 0.5 - logoSize / 2;
          top = tableCenterY + tableRadiusY * 0.5 - logoSize / 2;
          break;
        case 'center':
        default:
          left = tableCenterX - logoSize / 2;
          top = tableCenterY - logoSize / 2;
          break;
      }
      
      return Stack(
        fit: StackFit.expand,
        children: [
          Positioned(
            left: left,
            top: top,
            width: logoSize,
            height: logoSize,
            child: IgnorePointer(
              child: Opacity(
                opacity: 0.6, // Semi-transparent so it doesn't obstruct gameplay
                child: Image.asset(
                  'assets/images/dcrlogo.png',
                  fit: BoxFit.contain,
                  errorBuilder: (context, error, stackTrace) {
                    // Return empty widget if image fails to load
                    return const SizedBox.shrink();
                  },
                ),
              ),
            ),
          ),
        ],
      );
    });
  }
}

