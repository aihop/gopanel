import 'task_status.dart';
import 'task_type.dart';
import 'task_attention.dart';

class TaskEntity {
  final String id;
  final String title;
  final TaskType type;
  final TaskStatus status;
  final double? progress;
  final DateTime? startedAt;
  final DateTime? updatedAt;
  final String? summary;
  final String? error;
  final Map<String, String> meta;
  final TaskAttention? attention;

  const TaskEntity({
    required this.id,
    required this.title,
    required this.type,
    required this.status,
    this.progress,
    this.startedAt,
    this.updatedAt,
    this.summary,
    this.error,
    this.meta = const {},
    this.attention,
  });

  bool get requiresAttention => attention != null;
}
