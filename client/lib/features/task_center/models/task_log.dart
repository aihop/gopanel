import 'task_status.dart';

class TaskLog {
  final String taskId;
  final List<String> lines;
  final TaskStatus? status;
  final Map<String, String> meta;

  const TaskLog({
    required this.taskId,
    required this.lines,
    this.status,
    this.meta = const {},
  });
}
