import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/models/ai_dev_session.dart';
import 'package:gopanel/features/ai_workspace/models/ai_session_state_info.dart';
import 'package:gopanel/features/ai_workspace/models/code_workspace_file.dart';
import 'package:gopanel/features/ai_workspace/models/code_delivery_job.dart';

void main() {
  test('parses Code executor capabilities and policies', () {
    final executor = CodeExecutor.fromJson({
      'id': 'codex',
      'name': 'Codex',
      'available': true,
      'capabilities': ['code', 'automation', 'resume'],
      'approvalPolicies': ['manual', 'safe_auto', 'full_auto'],
    });

    expect(executor.id, 'codex');
    expect(executor.supportsAutomation, isTrue);
    expect(executor.approvalPolicies, contains('safe_auto'));
  });

  test('parses structured Code session state', () {
    final state = AiSessionStateInfo.fromJson({
      'session': {
        'id': 12,
        'projectId': 3,
        'title': 'mobile client',
        'agentName': 'codex',
        'workDir': '/srv/mobile',
        'status': 'active',
        'currentStage': 'awaiting_approval',
        'approvalPolicy': 'safe_auto',
      },
      'currentStage': 'awaiting_approval',
      'currentTask': {
        'id': 18,
        'title': '修复移动端终端连接',
        'status': 'running',
        'agentName': 'codex',
      },
      'recentMessages': [
        {
          'id': 8,
          'role': 'agent',
          'content': '准备执行高风险操作',
          'createdAt': '2026-07-31T10:00:00Z',
        },
      ],
      'previews': [
        {
          'id': 2,
          'title': 'Web',
          'url': 'https://example.test',
          'status': 'ready',
        },
      ],
      'timelineEvents': [
        {
          'id': 4,
          'title': '等待审批',
          'stage': 'awaiting_approval',
          'status': 'info',
        },
      ],
      'changedFiles': ['lib/main.dart'],
      'pendingApproval': {
        'id': 6,
        'sessionId': 12,
        'title': '审批: git push',
        'content': 'git push',
        'riskLevel': 'high',
        'status': 'pending',
      },
    });

    expect(state.session.agentName, 'codex');
    expect(state.session.approvalPolicy, 'safe_auto');
    expect(state.currentTask?.title, '修复移动端终端连接');
    expect(state.recentMessages.single.text, '准备执行高风险操作');
    expect(state.previews.single.status, 'ready');
    expect(state.pendingApproval?.riskLevel, 'high');
    expect(state.changedFiles, ['lib/main.dart']);
  });

  test('parses Code workspace structure and file content', () {
    final structure = CodeStructureResult.fromJson({
      'path': 'lib',
      'truncated': true,
      'entries': [
        {
          'name': 'features',
          'path': 'lib/features',
          'isDir': true,
          'extension': '',
        },
        {
          'name': 'main.dart',
          'path': 'lib/main.dart',
          'isDir': false,
          'extension': 'dart',
        },
      ],
    });
    final file = CodeSessionFile.fromJson({
      'path': 'lib/main.dart',
      'content': 'void main() {}',
      'extension': 'dart',
      'size': 14,
      'version': 'sha256',
    });

    expect(structure.path, 'lib');
    expect(structure.truncated, isTrue);
    expect(structure.entries.first.isDirectory, isTrue);
    expect(structure.entries.last.extension, 'dart');
    expect(file.content, 'void main() {}');
    expect(file.size, 14);
  });

  test('parses Code delivery progress and conflicts', () {
    final delivery = CodeDeliveryJob.fromJson({
      'id': 21,
      'sessionId': 12,
      'status': 'conflict',
      'stage': 'merging',
      'progress': 42,
      'attempt': 2,
      'queuePosition': 0,
      'targetBranch': 'main',
      'conflictFiles': ['lib/main.dart'],
      'repositories': [
        {
          'repositoryName': 'client',
          'status': 'conflict',
          'targetBranch': 'main',
          'conflictFiles': ['lib/main.dart'],
        },
      ],
    });

    expect(delivery.progress, 42);
    expect(delivery.canRetry, isTrue);
    expect(delivery.conflictFiles, ['lib/main.dart']);
    expect(delivery.repositories.single.repositoryName, 'client');
  });
}
