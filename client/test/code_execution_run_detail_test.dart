import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/ai_workspace/data/ai_workspace_repository.dart';
import 'package:gopanel/features/ai_workspace/models/code_execution_run_detail.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_execution_run_controller.dart';
import 'package:gopanel/features/ai_workspace/presentation/screens/code_execution_run_screen.dart';

class _FakeRunRepository extends AiWorkspaceRepository {
  _FakeRunRepository() : super(ApiClient());

  CodeExecutionRunDetail detail = _detail();
  Object? error;

  @override
  Future<CodeExecutionRunDetail> getExecutionRun(int runId) async {
    if (error != null) throw error!;
    return detail;
  }
}

CodeExecutionRunDetail _detail() {
  return CodeExecutionRunDetail.fromJson({
    'id': 41,
    'sessionId': 12,
    'taskId': 13,
    'instructionId': 14,
    'executorId': 'codex',
    'model': 'gpt-5.1-codex',
    'prompt': '修复构建错误',
    'output': '已修复构建错误',
    'rawOutput': '{"type":"result","text":"done"}',
    'status': 'completed',
    'exitCode': 0,
    'durationMs': 2300,
    'inputTokens': 100,
    'outputTokens': 40,
    'cachedInputTokens': 20,
    'reasoningTokens': 10,
    'totalTokens': 150,
    'startedAt': '2026-08-11T08:00:00Z',
    'completedAt': '2026-08-11T08:00:02Z',
  });
}

void main() {
  test('parses raw execution details and builds diagnostics', () {
    final detail = _detail();

    expect(detail.id, 41);
    expect(detail.executorId, 'codex');
    expect(detail.rawOutput, contains('result'));
    expect(detail.hasTokenUsage, isTrue);
    expect(detail.totalTokens, 150);
    expect(detail.diagnosticText, contains('Run #41'));
    expect(detail.diagnosticText, contains('Raw output:'));
    expect(detail.diagnosticText, contains('gpt-5.1-codex'));
  });

  test('loads a run and preserves it when refresh fails', () async {
    final repository = _FakeRunRepository();
    final controller = CodeExecutionRunController(
      repository: repository,
      runId: 41,
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(controller.state.run?.id, 41);
    expect(controller.state.errorMessage, isNull);

    repository.error = StateError('run unavailable');
    await controller.load();
    expect(controller.state.run?.id, 41);
    expect(controller.state.errorMessage, contains('run unavailable'));
    expect(controller.state.isLoading, isFalse);
  });

  testWidgets('shows run metadata and token usage', (tester) async {
    tester.binding.platformDispatcher.localeTestValue = const Locale('zh');
    addTearDown(tester.binding.platformDispatcher.clearLocaleTestValue);

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: CodeExecutionRunOverview(run: _detail()),
          ),
        ),
      ),
    );

    expect(find.text('运行概览'), findsOneWidget);
    expect(find.text('状态: 已完成'), findsOneWidget);
    expect(find.text('执行器: codex'), findsOneWidget);
    expect(find.text('模型: gpt-5.1-codex'), findsOneWidget);
    expect(find.text('总 Tokens: 150'), findsOneWidget);
    expect(find.text('缓存输入 Tokens: 20'), findsOneWidget);
  });
}
