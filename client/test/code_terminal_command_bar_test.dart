import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/presentation/widgets/code_terminal_command_bar.dart';

void main() {
  test('terminal symbol insertion replaces selection and keeps cursor', () {
    const value = TextEditingValue(
      text: 'git status',
      selection: TextSelection(baseOffset: 4, extentOffset: 10),
    );

    final result = insertCodeTerminalText(value, 'diff');

    expect(result.text, 'git diff');
    expect(result.selection, const TextSelection.collapsed(offset: 8));
  });

  test('terminal symbol insertion appends without a valid selection', () {
    const value = TextEditingValue(text: 'cd ');

    final result = insertCodeTerminalText(value, '/srv/app');

    expect(result.text, 'cd /srv/app');
    expect(result.selection.extentOffset, result.text.length);
  });
}
