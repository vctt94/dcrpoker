import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:golib_plugin/grpc/generated/poker.pb.dart' as pr;
import 'package:pokerui/components/poker/cards.dart';

List<Rect> _textRects(WidgetTester tester, String text) {
  final finder = find.text(text);
  return List<Rect>.generate(
    finder.evaluate().length,
    (index) => tester.getRect(finder.at(index)),
  );
}

Rect _topRect(List<Rect> rects) =>
    rects.reduce((a, b) => a.top <= b.top ? a : b);

Rect _bottomRect(List<Rect> rects) =>
    rects.reduce((a, b) => a.bottom >= b.bottom ? a : b);

void main() {
  testWidgets('narrow number cards keep center content between corners',
      (WidgetTester tester) async {
    final card = pr.Card()
      ..value = '10'
      ..suit = 'diamonds';

    await tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: const MediaQueryData(size: Size(390, 844)),
          child: Scaffold(
            body: Center(
              child: SizedBox(
                width: 40,
                height: 56,
                child: CardFace(card: card),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final rankRects = _textRects(tester, '10');
    final suitRects = _textRects(tester, '♦');
    final topRankRect = _topRect(rankRects);
    final bottomRankRect = _bottomRect(rankRects);
    final centerRect =
        tester.getRect(find.byKey(const ValueKey('card_center_content')));

    expect(rankRects, hasLength(2));
    expect(suitRects.length, greaterThan(2));
    expect(centerRect.top, greaterThan(topRankRect.bottom));
    expect(centerRect.bottom, lessThan(bottomRankRect.top));
    expect(tester.takeException(), isNull);
  });

  testWidgets('wider number cards keep center content between corners',
      (WidgetTester tester) async {
    final card = pr.Card()
      ..value = '10'
      ..suit = 'diamonds';

    await tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: const MediaQueryData(size: Size(390, 844)),
          child: Scaffold(
            body: Center(
              child: SizedBox(
                width: 58.8,
                height: 82.32,
                child: CardFace(card: card),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final rankRects = _textRects(tester, '10');
    final topRankRect = _topRect(rankRects);
    final bottomRankRect = _bottomRect(rankRects);
    final centerRect =
        tester.getRect(find.byKey(const ValueKey('card_center_content')));

    expect(rankRects, hasLength(2));
    expect(centerRect.top, greaterThan(topRankRect.bottom));
    expect(centerRect.bottom, lessThan(bottomRankRect.top));
    expect(tester.takeException(), isNull);
  });

  testWidgets('narrow face cards render without center overflow',
      (WidgetTester tester) async {
    final card = pr.Card()
      ..value = 'Q'
      ..suit = 'spades';

    await tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: const MediaQueryData(size: Size(390, 844)),
          child: Scaffold(
            body: Center(
              child: SizedBox(
                width: 34,
                height: 47.6,
                child: CardFace(card: card),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const ValueKey('card_center_content')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
