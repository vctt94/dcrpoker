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

void main() {
  testWidgets('narrow number cards still render pip symbols',
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

    expect(rankRects, hasLength(2));
    expect(suitRects.length, greaterThan(2));
    expect(tester.takeException(), isNull);
  });

  testWidgets('wider number cards render without layout errors',
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
    expect(rankRects, hasLength(2));
    expect(tester.takeException(), isNull);
  });
}
