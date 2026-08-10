import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/task_center/models/task_attention.dart';
import 'package:gopanel/features/task_center/models/task_entity.dart';
import 'package:gopanel/features/task_center/models/task_status.dart';
import 'package:gopanel/features/task_center/models/task_type.dart';
import 'package:gopanel/features/task_center/presentation/controllers/task_center_controller.dart';

void main() {
  const attention = TaskAttention(
    id: 'approval:8',
    type: 'approval',
    severity: 'warning',
    title: '等待你确认',
    summary: 'git push origin main',
    sessionId: 12,
    taskId: 18,
    approvalId: 8,
    updatedAt: null,
    actions: [],
  );
  const regularTask = TaskEntity(
    id: 'pipeline:1',
    title: '构建',
    type: TaskType.pipeline,
    status: TaskStatus.success,
  );
  const attentionTask = TaskEntity(
    id: 'aiSession:12',
    title: '开发会话',
    type: TaskType.ai,
    status: TaskStatus.running,
    attention: attention,
  );
  const attentionOnlyTask = TaskEntity(
    id: 'aiSession:14',
    title: '初始化失败会话',
    type: TaskType.ai,
    status: TaskStatus.failed,
    attention: attention,
    attentionOnly: true,
  );

  test('attention tasks include server and local tasks', () {
    const state = TaskCenterState(
      tasks: [regularTask],
      localTasks: [attentionTask],
    );

    expect(state.allTasks, [attentionTask, regularTask]);
    expect(state.attentionTasks, [attentionTask]);
  });

  test('attention filter hides regular tasks', () {
    const state = TaskCenterState(
      tasks: [regularTask, attentionTask],
      attentionOnly: true,
    );

    expect(state.visibleTasks, [attentionTask]);
  });

  test('default list hides attention-only synthetic entries', () {
    const state = TaskCenterState(tasks: [regularTask, attentionOnlyTask]);

    expect(state.visibleTasks, [regularTask]);
    expect(state.attentionTasks, [attentionOnlyTask]);
  });

  test('server tasks replace local tasks with the same identity', () {
    const local = TaskEntity(
      id: 'pipeline:1',
      title: '本地构建',
      type: TaskType.pipeline,
      status: TaskStatus.running,
    );
    const state = TaskCenterState(tasks: [regularTask], localTasks: [local]);

    expect(state.allTasks, [regularTask]);
  });
}
