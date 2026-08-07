import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/models/ai_dev_session.dart';
import 'package:gopanel/features/ai_workspace/models/code_delivery_job.dart';
import 'package:gopanel/features/ai_workspace/presentation/widgets/code_delivery_card.dart';

void main() {
  testWidgets('shows pending commits as a continued delivery', (tester) async {
    tester.binding.platformDispatcher.localeTestValue = const Locale('zh');
    addTearDown(tester.binding.platformDispatcher.clearLocaleTestValue);
    var starts = 0;
    final session = AiDevSession.fromJson({
      'id': 12,
      'sourceWorkDir': '/srv/app',
      'workDir': '/srv/worktree',
      'worktreeBranch': 'code/session-12',
      'status': 'active',
    });
    final delivery = CodeDeliveryJob.fromJson({
      'id': 21,
      'sessionId': 12,
      'status': 'completed',
      'stage': 'completed',
      'progress': 100,
      'resultCommit': '1234567890abcdef',
      'hasPendingChanges': true,
      'hasPendingCommits': true,
      'hasUncommittedChanges': false,
    });

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: CodeDeliveryCard(
              session: session,
              delivery: delivery,
              loading: false,
              errorMessage: null,
              onStart: () => starts++,
            ),
          ),
        ),
      ),
    );

    expect(find.text('待继续交付'), findsOneWidget);
    expect(find.text('交付后续提交'), findsOneWidget);
    expect(find.textContaining('本批已交付，仍有后续提交待交付'), findsOneWidget);
    expect(find.text('已完成合并与推送'), findsNothing);
    final badge = tester.widget<Text>(find.text('待继续交付'));
    expect(badge.style?.color, Colors.amberAccent);

    await tester.tap(find.text('交付后续提交'));
    expect(starts, 1);
  });
}
