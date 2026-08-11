import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../task_center/data/task_attention_repository.dart';
import '../../../task_center/models/task_attention.dart';

final codeAttentionRepositoryProvider = Provider<TaskAttentionRepository>((
  ref,
) {
  return TaskAttentionRepository(ApiClient());
});

class CodeAttentionState {
  final bool isLoading;
  final String? errorMessage;
  final List<TaskAttention> items;
  final String? actionKey;

  const CodeAttentionState({
    this.isLoading = true,
    this.errorMessage,
    this.items = const [],
    this.actionKey,
  });

  CodeAttentionState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<TaskAttention>? items,
    String? actionKey,
    bool clearError = false,
    bool clearAction = false,
  }) {
    return CodeAttentionState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      items: items ?? this.items,
      actionKey: clearAction ? null : (actionKey ?? this.actionKey),
    );
  }
}

final codeAttentionControllerProvider =
    NotifierProvider<CodeAttentionController, CodeAttentionState>(
      CodeAttentionController.new,
    );

class CodeAttentionController extends Notifier<CodeAttentionState> {
  late TaskAttentionRepository _repository;

  @override
  CodeAttentionState build() {
    _repository = ref.watch(codeAttentionRepositoryProvider);
    Future.microtask(load);
    return const CodeAttentionState();
  }

  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final items = await _repository.list();
      state = state.copyWith(isLoading: false, items: items);
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<bool> execute(
    TaskAttention attention,
    TaskAttentionAction action,
  ) async {
    if (state.actionKey != null) return false;
    final key = '${attention.id}:${action.type}';
    state = state.copyWith(actionKey: key, clearError: true);
    try {
      await _repository.execute(action);
      state = state.copyWith(
        items: state.items.where((item) => item.id != attention.id).toList(),
        clearAction: true,
      );
      await load();
      return true;
    } catch (error) {
      state = state.copyWith(errorMessage: error.toString(), clearAction: true);
      return false;
    }
  }
}
