import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/ai_workspace/data/ai_workspace_repository.dart';
import 'package:gopanel/features/ai_workspace/models/code_git_review.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_git_diff_controller.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_git_review_controller.dart';
import 'package:gopanel/features/ai_workspace/presentation/widgets/code_git_review_cards.dart';

class _FakeGitRepository extends AiWorkspaceRepository {
  _FakeGitRepository() : super(ApiClient());

  CodeGitStatus status = _changedStatus();
  CodeGitStatus? afterSaveStatus;
  CodeGitDiff diff = _diff();
  Object? statusError;
  Object? diffError;
  Object? saveError;
  String savedMessage = '';

  @override
  Future<CodeGitStatus> getGitStatus(int sessionId) async {
    if (statusError != null) throw statusError!;
    return status;
  }

  @override
  Future<CodeGitDiff> getGitDiff({
    required int sessionId,
    required String repositoryId,
    required String path,
    required String kind,
  }) async {
    if (diffError != null) throw diffError!;
    return diff;
  }

  @override
  Future<CodeGitSaveResult> saveGitChanges(
    int sessionId,
    String message,
  ) async {
    if (saveError != null) throw saveError!;
    savedMessage = message;
    status = afterSaveStatus ?? status;
    return CodeGitSaveResult.fromJson({
      'status': 'saved',
      'commit': 'abc123',
      'branch': 'code/session-12',
    });
  }
}

Map<String, dynamic> _statusJson({bool changed = true}) {
  return {
    'available': true,
    'files': changed ? 1 : 0,
    'staged': changed ? 1 : 0,
    'changed': changed ? 1 : 0,
    'untracked': 0,
    'additions': changed ? 5 : 0,
    'deletions': changed ? 3 : 0,
    'stagedAdditions': changed ? 2 : 0,
    'stagedDeletions': changed ? 1 : 0,
    'repositories': [
      {
        'id': 'repository-1',
        'name': 'gopanel',
        'branch': 'code/session-12',
        'stagedCount': changed ? 1 : 0,
        'changedCount': changed ? 1 : 0,
        'untrackedCount': 0,
        'additions': changed ? 5 : 0,
        'deletions': changed ? 3 : 0,
        'stagedAdditions': changed ? 2 : 0,
        'stagedDeletions': changed ? 1 : 0,
        'truncated': changed,
        'isolated': true,
        'savedCommits': 2,
        'headCommit': 'abc12345',
        'files': changed
            ? [
                {
                  'path': 'lib/app.dart',
                  'oldPath': 'lib/old_app.dart',
                  'workspacePath': 'source/lib/app.dart',
                  'indexStatus': 'M',
                  'worktreeStatus': 'M',
                  'staged': true,
                  'changed': true,
                  'untracked': false,
                },
              ]
            : <Map<String, dynamic>>[],
      },
    ],
  };
}

CodeGitStatus _changedStatus() => CodeGitStatus.fromJson(_statusJson());

CodeGitStatus _cleanStatus() =>
    CodeGitStatus.fromJson(_statusJson(changed: false));

CodeGitDiff _diff() => CodeGitDiff.fromJson({
  'repositoryId': 'repository-1',
  'path': 'lib/app.dart',
  'kind': 'working',
  'content': '@@ -1 +1 @@\n-old\n+new',
  'truncated': true,
});

void main() {
  test('parses Git totals, repository state, and dual file states', () {
    final status = _changedStatus();
    final repository = status.repositories.single;
    final file = repository.files.single;

    expect(status.hasChanges, isTrue);
    expect(status.totalAdditions, 7);
    expect(status.totalDeletions, 4);
    expect(repository.isolated, isTrue);
    expect(repository.truncated, isTrue);
    expect(repository.savedCommits, 2);
    expect(repository.headCommit, 'abc12345');
    expect(file.staged, isTrue);
    expect(file.changed, isTrue);
    expect(file.workspacePath, 'source/lib/app.dart');
  });

  test('parses commits from a multi-repository save result', () {
    final result = CodeGitSaveResult.fromJson({
      'status': 'saved',
      'repositories': [
        {'commit': 'abc123'},
        {'commit': ''},
        {'commit': 'def456'},
      ],
    });

    expect(result.commits, ['abc123', 'def456']);
  });

  test('loads status and preserves it when refresh fails', () async {
    final repository = _FakeGitRepository();
    final controller = CodeGitReviewController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(controller.state.status?.files, 1);

    repository.statusError = StateError('status unavailable');
    await controller.load();
    expect(controller.state.status?.files, 1);
    expect(controller.state.errorMessage, contains('status unavailable'));
    expect(controller.state.isLoading, isFalse);
  });

  test('saves changes then reloads a clean status', () async {
    final repository = _FakeGitRepository()..afterSaveStatus = _cleanStatus();
    final controller = CodeGitReviewController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.load();
    final result = await controller.save('feat: mobile review');

    expect(result?.commit, 'abc123');
    expect(repository.savedMessage, 'feat: mobile review');
    expect(controller.state.status?.hasChanges, isFalse);
    expect(controller.state.errorMessage, isNull);
  });

  test('preserves status and exposes an error when save fails', () async {
    final repository = _FakeGitRepository()
      ..saveError = StateError('save unavailable');
    final controller = CodeGitReviewController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(await controller.save(''), isNull);
    expect(controller.state.status?.hasChanges, isTrue);
    expect(controller.state.errorMessage, contains('save unavailable'));
    expect(controller.state.isSaving, isFalse);
  });

  test('loads a diff and preserves it when refresh fails', () async {
    final repository = _FakeGitRepository();
    final controller = CodeGitDiffController(
      repository: repository,
      sessionId: 12,
      repositoryId: 'repository-1',
      path: 'lib/app.dart',
      kind: 'working',
    );
    addTearDown(controller.dispose);

    await controller.load();
    expect(controller.state.diff?.content, contains('+new'));

    repository.diffError = StateError('diff unavailable');
    await controller.load();
    expect(controller.state.diff?.content, contains('+new'));
    expect(controller.state.errorMessage, contains('diff unavailable'));
  });

  testWidgets('shows totals and both staged and working diff actions', (
    tester,
  ) async {
    tester.binding.platformDispatcher.localeTestValue = const Locale('zh');
    addTearDown(tester.binding.platformDispatcher.clearLocaleTestValue);
    final openedKinds = <String>[];
    final status = _changedStatus();

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: Column(
              children: [
                CodeGitSummaryCard(status: status),
                CodeGitRepositoryCard(
                  repository: status.repositories.single,
                  onOpenDiff: (file, kind) => openedKinds.add(kind),
                ),
              ],
            ),
          ),
        ),
      ),
    );

    expect(find.text('+7'), findsOneWidget);
    expect(find.text('-4'), findsOneWidget);
    expect(find.text('文件较多，列表仅展示前 500 项'), findsOneWidget);

    await tester.tap(find.text('查看已暂存差异'));
    await tester.tap(find.text('查看未暂存差异'));
    expect(openedKinds, ['staged', 'working']);
  });
}
