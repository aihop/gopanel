import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/ai_workspace/data/ai_workspace_repository.dart';
import 'package:gopanel/features/ai_workspace/models/ai_dev_session.dart';
import 'package:gopanel/features/ai_workspace/models/code_session_recovery.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_session_recovery_controller.dart';
import 'package:gopanel/features/ai_workspace/presentation/widgets/code_recovery_cards.dart';

class _FakeRecoveryRepository extends AiWorkspaceRepository {
  _FakeRecoveryRepository() : super(ApiClient());

  final Map<int, CodeSessionHistory> histories = {};
  CodeSessionInitialization initialization = const CodeSessionInitialization(
    id: 12,
    status: 'active',
    currentStage: 'idle',
    errorMessage: '',
  );
  Object? historyError;
  Object? retryError;
  Object? initializationRetryError;
  final List<int> retriedInstructions = [];
  var initializationRetries = 0;

  @override
  Future<CodeSessionHistory> getSessionHistory(
    int sessionId, {
    int page = 1,
    int limit = 20,
  }) async {
    if (historyError != null) throw historyError!;
    return histories[page] ?? _history(page: page);
  }

  @override
  Future<CodeSessionInitialization> getSessionInitialization(
    int sessionId,
  ) async {
    return initialization;
  }

  @override
  Future<AiInstruction> retryInstruction(int instructionId) async {
    if (retryError != null) throw retryError!;
    retriedInstructions.add(instructionId);
    return AiInstruction.fromJson({
      'id': instructionId,
      'sessionId': 12,
      'status': 'queued',
    });
  }

  @override
  Future<CodeSessionInitialization> retrySessionInitialization(
    int sessionId,
  ) async {
    if (initializationRetryError != null) throw initializationRetryError!;
    initializationRetries++;
    initialization = const CodeSessionInitialization(
      id: 12,
      status: 'initializing',
      currentStage: 'syncing_base',
      errorMessage: '',
    );
    return initialization;
  }
}

const _sessionJson = <String, dynamic>{
  'id': 12,
  'title': 'Mobile recovery',
  'status': 'active',
};

CodeExecutionRun _run({
  required int id,
  required int instructionId,
  String status = 'failed',
}) {
  return CodeExecutionRun(
    id: id,
    instructionId: instructionId,
    prompt: 'Fix the build',
    output: '',
    status: status,
    errorMessage: status == 'failed' ? 'build failed' : '',
    durationMs: 1200,
    totalTokens: 42,
    createdAt: DateTime.utc(2026, 8, 11),
  );
}

CodeSessionHistory _history({
  int page = 1,
  int total = 0,
  List<CodeExecutionRun> runs = const [],
}) {
  return CodeSessionHistory(
    session: AiDevSession.fromJson(_sessionJson),
    messages: const [],
    runs: runs,
    total: total,
    page: page,
    limit: 2,
  );
}

void main() {
  test('parses complete history and recovery statuses', () {
    final history = CodeSessionHistory.fromJson({
      'session': _sessionJson,
      'messages': [
        {
          'id': 7,
          'runId': 8,
          'role': 'user',
          'content': '继续修复',
          'createdAt': '2026-08-11T08:00:00Z',
        },
      ],
      'runs': [
        {
          'id': 9,
          'instructionId': 10,
          'prompt': '修复构建',
          'status': 'cancelled',
          'durationMs': 1300,
          'totalTokens': 50,
        },
      ],
      'total': 3,
      'page': 2,
      'limit': 1,
    });
    final initialization = CodeSessionInitialization.fromJson({
      'id': 12,
      'status': 'failed',
      'currentStage': 'initialization_failed',
      'initializationError': 'worktree unavailable',
    });

    expect(history.session.id, 12);
    expect(history.messages.single.content, '继续修复');
    expect(history.runs.single.canRetry, isTrue);
    expect(history.total, 3);
    expect(initialization.isFailed, isTrue);
    expect(initialization.canRetry, isTrue);
    expect(initialization.errorMessage, 'worktree unavailable');
  });

  test('loads pages without duplicate execution runs', () async {
    final repository = _FakeRecoveryRepository()
      ..histories[1] = _history(
        total: 3,
        runs: [_run(id: 1, instructionId: 11), _run(id: 2, instructionId: 12)],
      )
      ..histories[2] = _history(
        page: 2,
        total: 3,
        runs: [_run(id: 2, instructionId: 12), _run(id: 3, instructionId: 13)],
      );
    final controller = CodeSessionRecoveryController(
      repository: repository,
      sessionId: 12,
      pageSize: 2,
    );
    addTearDown(controller.dispose);

    await controller.load();
    await controller.loadMore();

    expect(controller.state.runs.map((run) => run.id), [1, 2, 3]);
    expect(controller.state.page, 2);
    expect(controller.state.canLoadMore, isFalse);
  });

  test('refresh errors preserve the previously loaded history', () async {
    final repository = _FakeRecoveryRepository()
      ..histories[1] = _history(
        total: 1,
        runs: [_run(id: 1, instructionId: 11)],
      );
    final controller = CodeSessionRecoveryController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.load();
    repository.historyError = StateError('history unavailable');
    await controller.load();

    expect(controller.state.runs.single.id, 1);
    expect(controller.state.errorMessage, contains('history unavailable'));
    expect(controller.state.isLoading, isFalse);
  });

  test(
    'retries failed or stopped instructions without rewriting history',
    () async {
      final repository = _FakeRecoveryRepository();
      final controller = CodeSessionRecoveryController(
        repository: repository,
        sessionId: 12,
      );
      addTearDown(controller.dispose);
      final failed = _run(id: 1, instructionId: 11);
      final stopped = _run(id: 2, instructionId: 12, status: 'cancelled');

      expect(await controller.retryInstruction(failed), isTrue);
      expect(await controller.retryInstruction(stopped), isTrue);
      expect(await controller.retryInstruction(failed), isFalse);

      expect(repository.retriedInstructions, [11, 12]);
      expect(controller.state.retriedInstructionIds, {11, 12});
      expect(failed.status, 'failed');
      expect(stopped.status, 'cancelled');
    },
  );

  test('retry errors remain visible and allow another attempt', () async {
    final repository = _FakeRecoveryRepository()
      ..retryError = StateError('queue unavailable');
    final controller = CodeSessionRecoveryController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);
    final run = _run(id: 1, instructionId: 11);

    expect(await controller.retryInstruction(run), isFalse);
    expect(controller.state.errorMessage, contains('queue unavailable'));
    expect(controller.state.retriedInstructionIds, isEmpty);

    repository.retryError = null;
    expect(await controller.retryInstruction(run), isTrue);
  });

  test('retries failed initialization and updates its state', () async {
    final repository = _FakeRecoveryRepository()
      ..initialization = const CodeSessionInitialization(
        id: 12,
        status: 'failed',
        currentStage: 'initialization_failed',
        errorMessage: 'clone failed',
      );
    final controller = CodeSessionRecoveryController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(await controller.retryInitialization(), isTrue);

    expect(repository.initializationRetries, 1);
    expect(controller.state.initialization?.isInitializing, isTrue);
    expect(controller.state.errorMessage, isNull);
  });

  testWidgets('offers retry for a failed execution run', (tester) async {
    tester.binding.platformDispatcher.localeTestValue = const Locale('zh');
    addTearDown(tester.binding.platformDispatcher.clearLocaleTestValue);
    final failed = _run(id: 1, instructionId: 11);
    CodeExecutionRun? retried;
    CodeExecutionRun? opened;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: CodeExecutionRunsCard(
              runs: [failed],
              total: 1,
              isLoadingMore: false,
              retryingInstructionId: null,
              retriedInstructionIds: const {},
              onRetry: (run) => retried = run,
              onOpenDetail: (run) => opened = run,
              onLoadMore: () {},
            ),
          ),
        ),
      ),
    );

    expect(find.text('失败'), findsOneWidget);
    expect(find.text('查看详情'), findsOneWidget);
    await tester.tap(find.text('查看详情'));
    expect(opened?.id, 1);
    expect(find.text('重试指令'), findsOneWidget);
    await tester.tap(find.text('重试指令'));
    expect(retried?.instructionId, 11);
  });

  testWidgets('shows stopped state and disables an already retried run', (
    tester,
  ) async {
    tester.binding.platformDispatcher.localeTestValue = const Locale('zh');
    addTearDown(tester.binding.platformDispatcher.clearLocaleTestValue);
    final stopped = _run(id: 2, instructionId: 12, status: 'cancelled');

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: CodeExecutionRunsCard(
              runs: [stopped],
              total: 1,
              isLoadingMore: false,
              retryingInstructionId: null,
              retriedInstructionIds: const {12},
              onRetry: (_) {},
              onOpenDetail: (_) {},
              onLoadMore: () {},
            ),
          ),
        ),
      ),
    );

    expect(find.text('已停止'), findsOneWidget);
    expect(find.text('已重新加入队列'), findsOneWidget);
    expect(find.text('重试指令'), findsNothing);
  });
}
