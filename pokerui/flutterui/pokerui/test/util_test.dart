import 'package:flutter_test/flutter_test.dart';
import 'package:pokerui/util.dart';

void main() {
  test('formatDcrFromAtoms trims trailing zeros', () {
    expect(formatDcrFromAtoms(100000000), '1');
    expect(formatDcrFromAtoms(1000000), '0.01');
    expect(formatDcrFromAtoms(123456789), '1.23456789');
    expect(formatDcrFromAtoms(120000000), '1.2');
  });
}
