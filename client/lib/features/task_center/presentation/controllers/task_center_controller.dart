import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/task_repository.dart';
import '../../models/task_entity.dart';
import '../../models/task_status.dart';

final taskRepositoryProvider = Provider<TaskRepository>((ref) {
  return TaskRepository(ApiClient());
});

class TaskCenterState {
  final bool isLoading;
  final String? errorMessage;
  final List<TaskEntity> tasks;
  final List<TaskEntity> localTasks;
  final TaskStatus? filter;
  final bool attentionOnly;

  const TaskCenterState({
    this.isLoading = true,
    this.errorMessage,
    this.tasks = const [],
    this.localTasks = const [],
    this.filter,
    this.attentionOnly = false,
  });

  TaskCenterState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<TaskEntity>? tasks,
    List<TaskEntity>? localTasks,
    TaskStatus? filter,
    bool clearFilter = false,
    bool? attentionOnly,
  }) {
    return TaskCenterState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      tasks: tasks ?? this.tasks,
      localTasks: localTasks ?? this.localTasks,
      filter: clearFilter ? null : (filter ?? this.filter),
      attentionOnly: attentionOnly ?? this.attentionOnly,
    );
  }

  List<TaskEntity> get visibleTasks {
    final list = [...localTasks, ...tasks];
    if (attentionOnly) {
      return list.where((task) => task.requiresAttention).toList();
    }
    if (filter == null) return list;
    return list.where((t) => t.status == filter).toList();
  }
}

final taskCenterControllerProvider =
    NotifierProvider<TaskCenterController, TaskCenterState>(
      TaskCenterController.new,
    );

class TaskCenterController extends Notifier<TaskCenterState> {
  late TaskRepository _repo;

  @override
  TaskCenterState build() {
    _repo = ref.watch(taskRepositoryProvider);
    Future.microtask(_load);
    return const TaskCenterState();
  }

  Future<void> refresh() async {
    await _load();
  }

  void addLocalTask(TaskEntity task) {
    final list = [task, ...state.localTasks.where((t) => t.id != task.id)];
    state = state.copyWith(localTasks: list, errorMessage: null);
  }

  void updateLocalTask({
    required String id,
    TaskStatus? status,
    String? error,
  }) {
    final idx = state.localTasks.indexWhere((t) => t.id == id);
    if (idx < 0) return;
    final old = state.localTasks[idx];
    final next = TaskEntity(
      id: old.id,
      title: old.title,
      type: old.type,
      status: status ?? old.status,
      progress: status == TaskStatus.success ? 1 : old.progress,
      startedAt: old.startedAt,
      updatedAt: DateTime.now(),
      summary: old.summary,
      error: error ?? old.error,
      meta: old.meta,
      attention: old.attention,
    );
    final list = [...state.localTasks];
    list[idx] = next;
    state = state.copyWith(localTasks: list);
  }

  void setFilter(TaskStatus? status) {
    if (status == null) {
      state = state.copyWith(
        clearFilter: true,
        attentionOnly: false,
        errorMessage: null,
      );
      return;
    }
    state = state.copyWith(
      filter: status,
      attentionOnly: false,
      errorMessage: null,
    );
  }

  void showAttentionOnly() {
    state = state.copyWith(
      clearFilter: true,
      attentionOnly: true,
      errorMessage: null,
    );
  }

  Future<void> _load() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      final list = await _repo.list();
      state = state.copyWith(isLoading: false, tasks: list, errorMessage: null);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: '无法加载任务: $e');
    }
  }
}
