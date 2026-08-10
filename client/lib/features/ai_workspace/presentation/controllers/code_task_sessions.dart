import '../../models/ai_dev_session.dart';
import '../../models/code_task.dart';

List<AiDevSession> alignCodeTaskSessions(
  List<AiDevSession> sessions,
  List<CodeTask> tasks, {
  required int projectId,
}) {
  final sessionsById = {for (final session in sessions) session.id: session};
  final result = <AiDevSession>[];
  final added = <int>{};
  for (final task in tasks) {
    if (task.projectId != projectId || task.agentName == 'terminal') continue;
    final session = sessionsById[task.sessionId];
    if (session == null ||
        session.lastTaskId != task.id ||
        !added.add(session.id)) {
      continue;
    }
    result.add(session);
  }
  return result;
}
