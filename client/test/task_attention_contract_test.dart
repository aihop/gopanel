import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/task_center/data/task_attention_repository.dart';
import 'package:gopanel/features/task_center/models/task_attention.dart';

void main() {
  test('task attention uses the registered aggregate route', () {
    expect(taskAttentionListPath, '/api/code/attention');
  });

  test('task attention parses server actions and ignores invalid sessions', () {
    final items = parseTaskAttentionList({
      'items': [
        {
          'id': 'approval:8',
          'type': 'approval',
          'severity': 'warning',
          'title': '等待你确认',
          'summary': 'git push origin main',
          'sessionId': 12,
          'approvalId': 8,
          'updatedAt': '2026-08-05T10:00:00Z',
          'actions': [
            {
              'type': 'approve',
              'label': '允许执行',
              'method': 'POST',
              'path': '/api/code/approvals/8/approve',
              'requiresConfirmation': true,
            },
          ],
        },
        {'id': 'invalid', 'sessionId': 0},
      ],
    });

    expect(items, hasLength(1));
    expect(items.single.sessionId, 12);
    expect(items.single.actions.single.type, 'approve');
    expect(items.single.actions.single.requiresConfirmation, isTrue);
  });
}
