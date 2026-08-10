import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/models/ai_dev_session.dart';
import 'package:gopanel/features/ai_workspace/models/code_task.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_task_sessions.dart';

AiDevSession session({
  required int id,
  required int projectId,
  required int lastTaskId,
}) {
  return AiDevSession(
    id: id,
    userId: 1,
    projectId: projectId,
    title: 'session $id',
    agentName: 'codex',
    workDir: '/work/$id',
    sourceWorkDir: '',
    worktreeBranch: '',
    isolationMode: '',
    status: 'active',
    currentStage: 'idle',
    approvalPolicy: 'safe_auto',
    lastTaskId: lastTaskId,
    currentTaskTitle: '',
    createdAt: null,
    updatedAt: null,
    lastInstructionAt: null,
  );
}

CodeTask task({
  required int id,
  required int sessionId,
  required int projectId,
  String agentName = 'codex',
}) {
  return CodeTask(
    id: id,
    sessionId: sessionId,
    projectId: projectId,
    title: 'task $id',
    agentName: agentName,
    workDir: '/work/$sessionId',
    status: 'completed',
    createdAt: null,
    updatedAt: null,
  );
}

void main() {
  test('aligns project history to real code tasks in task order', () {
    final sessions = [
      session(id: 1, projectId: 7, lastTaskId: 11),
      session(id: 2, projectId: 7, lastTaskId: 12),
      session(id: 3, projectId: 7, lastTaskId: 0),
      session(id: 4, projectId: 8, lastTaskId: 14),
    ];
    final tasks = [
      task(id: 12, sessionId: 2, projectId: 7),
      task(id: 11, sessionId: 1, projectId: 7),
      task(id: 14, sessionId: 4, projectId: 8),
    ];

    final result = alignCodeTaskSessions(sessions, tasks, projectId: 7);

    expect(result.map((item) => item.id), [2, 1]);
  });

  test('filters terminal and stale task-to-session relationships', () {
    final sessions = [
      session(id: 1, projectId: 7, lastTaskId: 11),
      session(id: 2, projectId: 7, lastTaskId: 99),
    ];
    final tasks = [
      task(id: 11, sessionId: 1, projectId: 7, agentName: 'terminal'),
      task(id: 12, sessionId: 2, projectId: 7),
    ];

    expect(alignCodeTaskSessions(sessions, tasks, projectId: 7), isEmpty);
  });
}
