class CodeTask {
  final int id;
  final int sessionId;
  final int projectId;
  final String title;
  final String agentName;
  final String workDir;
  final String status;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const CodeTask({
    required this.id,
    required this.sessionId,
    required this.projectId,
    required this.title,
    required this.agentName,
    required this.workDir,
    required this.status,
    required this.createdAt,
    required this.updatedAt,
  });

  factory CodeTask.fromJson(Map<String, dynamic> json) {
    return CodeTask(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      projectId: (json['projectId'] as num?)?.toInt() ?? 0,
      title: (json['title'] ?? '').toString(),
      agentName: (json['agentName'] ?? '').toString(),
      workDir: (json['workDir'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
    );
  }
}
