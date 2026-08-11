import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_attention_controller.dart';
import 'package:gopanel/features/task_center/data/task_attention_repository.dart';
import 'package:gopanel/features/task_center/models/task_attention.dart';

void main() {
  const attention = TaskAttention(
    id: 'approval:8',
    type: 'approval',
    severity: 'warning',
    title: '等待确认',
    summary: 'git push origin main',
    sessionId: 12,
    taskId: 18,
    approvalId: 8,
    updatedAt: null,
    actions: [
      TaskAttentionAction(
        type: 'approve',
        label: '允许执行',
        method: 'POST',
        path: '/api/code/approvals/8/approve',
        requiresConfirmation: true,
      ),
    ],
  );

  test('loads Code attention items', () async {
    final repository = _FakeAttentionRepository(items: [attention]);
    final container = ProviderContainer(
      overrides: [
        codeAttentionRepositoryProvider.overrideWithValue(repository),
      ],
    );
    addTearDown(container.dispose);

    await container.read(codeAttentionControllerProvider.notifier).load();

    final state = container.read(codeAttentionControllerProvider);
    expect(state.isLoading, isFalse);
    expect(state.items, [attention]);
    expect(state.errorMessage, isNull);
  });

  test('executes an attention action and refreshes the list', () async {
    final repository = _FakeAttentionRepository(items: [attention]);
    final container = ProviderContainer(
      overrides: [
        codeAttentionRepositoryProvider.overrideWithValue(repository),
      ],
    );
    addTearDown(container.dispose);
    await container.read(codeAttentionControllerProvider.notifier).load();
    repository.items = const [];

    final success = await container
        .read(codeAttentionControllerProvider.notifier)
        .execute(attention, attention.actions.single);

    expect(success, isTrue);
    expect(repository.executed, [attention.actions.single]);
    expect(container.read(codeAttentionControllerProvider).items, isEmpty);
  });

  test('preserves the item and exposes action errors', () async {
    final repository = _FakeAttentionRepository(
      items: [attention],
      executeError: StateError('denied'),
    );
    final container = ProviderContainer(
      overrides: [
        codeAttentionRepositoryProvider.overrideWithValue(repository),
      ],
    );
    addTearDown(container.dispose);
    await container.read(codeAttentionControllerProvider.notifier).load();

    final success = await container
        .read(codeAttentionControllerProvider.notifier)
        .execute(attention, attention.actions.single);

    final state = container.read(codeAttentionControllerProvider);
    expect(success, isFalse);
    expect(state.items, [attention]);
    expect(state.errorMessage, contains('denied'));
    expect(state.actionKey, isNull);
  });
}

class _FakeAttentionRepository extends TaskAttentionRepository {
  _FakeAttentionRepository({required this.items, this.executeError})
    : super(ApiClient());

  List<TaskAttention> items;
  final Object? executeError;
  final List<TaskAttentionAction> executed = [];

  @override
  Future<List<TaskAttention>> list() async => items;

  @override
  Future<void> execute(TaskAttentionAction action) async {
    if (executeError != null) throw executeError!;
    executed.add(action);
  }
}
