import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/models/code_instruction_options.dart';
import 'package:gopanel/features/ai_workspace/presentation/widgets/code_instruction_composer.dart';

void main() {
  testWidgets('updates preview and approval instruction options', (
    tester,
  ) async {
    final controller = TextEditingController();
    final focusNode = FocusNode();
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);
    var options = const CodeInstructionOptions();

    Widget subject() => MaterialApp(
      home: Scaffold(
        body: CodeInstructionComposer(
          controller: controller,
          focusNode: focusNode,
          enabled: true,
          closed: false,
          options: options,
          onOptionsChanged: (value) => options = value,
          onSend: () {},
        ),
      ),
    );

    await tester.pumpWidget(subject());
    final chips = find.byType(FilterChip);
    expect(chips, findsNWidgets(2));
    expect(tester.widget<FilterChip>(chips.first).selected, isTrue);
    expect(tester.widget<FilterChip>(chips.last).selected, isFalse);

    await tester.tap(chips.first);
    expect(options.autoPreview, isFalse);
    expect(options.requireApproval, isFalse);

    await tester.pumpWidget(subject());
    await tester.tap(find.byType(FilterChip).last);
    expect(options.autoPreview, isFalse);
    expect(options.requireApproval, isTrue);
  });

  testWidgets('disables options and send action with no active session', (
    tester,
  ) async {
    final controller = TextEditingController();
    final focusNode = FocusNode();
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);
    var optionChanges = 0;
    var sends = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: CodeInstructionComposer(
            controller: controller,
            focusNode: focusNode,
            enabled: false,
            closed: false,
            options: const CodeInstructionOptions(),
            onOptionsChanged: (_) => optionChanges++,
            onSend: () => sends++,
          ),
        ),
      ),
    );

    await tester.tap(find.byType(FilterChip).first);
    await tester.tap(find.byType(IconButton));
    expect(optionChanges, 0);
    expect(sends, 0);
  });
}
