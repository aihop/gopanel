class TaskAttentionAction {
  final String type;
  final String label;
  final String method;
  final String path;
  final bool requiresConfirmation;

  const TaskAttentionAction({
    required this.type,
    required this.label,
    required this.method,
    required this.path,
    required this.requiresConfirmation,
  });

  factory TaskAttentionAction.fromJson(Map<String, dynamic> json) {
    return TaskAttentionAction(
      type: (json['type'] ?? '').toString(),
      label: (json['label'] ?? '').toString(),
      method: (json['method'] ?? '').toString(),
      path: (json['path'] ?? '').toString(),
      requiresConfirmation: json['requiresConfirmation'] == true,
    );
  }
}

class TaskAttention {
  final String id;
  final String type;
  final String severity;
  final String title;
  final String summary;
  final int sessionId;
  final int taskId;
  final int approvalId;
  final DateTime? updatedAt;
  final List<TaskAttentionAction> actions;

  const TaskAttention({
    required this.id,
    required this.type,
    required this.severity,
    required this.title,
    required this.summary,
    required this.sessionId,
    required this.taskId,
    required this.approvalId,
    required this.updatedAt,
    required this.actions,
  });

  factory TaskAttention.fromJson(Map<String, dynamic> json) {
    final actions = json['actions'] as List<dynamic>? ?? const [];
    return TaskAttention(
      id: (json['id'] ?? '').toString(),
      type: (json['type'] ?? '').toString(),
      severity: (json['severity'] ?? '').toString(),
      title: (json['title'] ?? '').toString(),
      summary: (json['summary'] ?? '').toString(),
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      taskId: (json['taskId'] as num?)?.toInt() ?? 0,
      approvalId: (json['approvalId'] as num?)?.toInt() ?? 0,
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
      actions: actions
          .whereType<Map>()
          .map(
            (item) =>
                TaskAttentionAction.fromJson(item.cast<String, dynamic>()),
          )
          .toList(),
    );
  }
}

List<TaskAttention> parseTaskAttentionList(Map<String, dynamic>? data) {
  final items = data?['items'] as List<dynamic>? ?? const [];
  return items
      .whereType<Map>()
      .map((item) => TaskAttention.fromJson(item.cast<String, dynamic>()))
      .where((item) => item.sessionId > 0)
      .toList();
}
