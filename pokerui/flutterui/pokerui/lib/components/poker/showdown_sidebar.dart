import 'package:flutter/material.dart';
import 'package:pokerui/models/poker.dart';
import 'package:pokerui/components/poker/showdown_content.dart';

/// A sidebar that slides in from the left showing showdown information.
class ShowdownSidebar extends StatefulWidget {
  const ShowdownSidebar({
    super.key,
    required this.model,
    required this.isVisible,
    this.onClose,
  });

  final PokerModel model;
  final bool isVisible;
  final VoidCallback? onClose;

  @override
  State<ShowdownSidebar> createState() => _ShowdownSidebarState();
}

class _ShowdownSidebarState extends State<ShowdownSidebar>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<Offset> _slideAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 300),
    );
    _slideAnimation = Tween<Offset>(
      begin: const Offset(-1.0, 0.0), // Start off-screen to the left
      end: Offset.zero, // End at final position
    ).animate(CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOutCubic,
    ));

    if (widget.isVisible) {
      _controller.forward();
    }
  }

  @override
  void didUpdateWidget(ShowdownSidebar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isVisible != oldWidget.isVisible) {
      if (widget.isVisible) {
        _controller.forward();
      } else {
        _controller.reverse();
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.isVisible && _controller.value == 0.0) {
      return const SizedBox.shrink();
    }

    const sidebarWidth = 400.0;
    return Positioned(
      top: 0,
      left: 0,
      bottom: 0,
      child: SlideTransition(
        position: _slideAnimation,
        child: Container(
          width: sidebarWidth,
          margin: EdgeInsets.zero,
          padding: EdgeInsets.zero,
          decoration: BoxDecoration(
            color: const Color(0xFF1A1D2E),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity(0.5),
                blurRadius: 20,
                spreadRadius: 5,
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.max,
            children: [
              // Header with close button
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.3),
                  border: Border(
                    bottom: BorderSide(
                      color: Colors.amber.withOpacity(0.5),
                      width: 2,
                    ),
                  ),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.history, color: Colors.amber, size: 24),
                    const SizedBox(width: 10),
                    const Expanded(
                      child: Text(
                        'Showdown',
                        style: TextStyle(
                          color: Colors.white,
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                    IconButton(
                      onPressed: widget.onClose,
                      icon: const Icon(Icons.close, color: Colors.white70),
                      tooltip: 'Close',
                    ),
                  ],
                ),
              ),
              // Showdown content
              Expanded(
                child: ShowdownContent(
                  model: widget.model,
                  showHeader: false,
                  showCloseButton: false,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
