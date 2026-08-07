class CodeDeliveryRepositoryResult {
  final String repositoryName;
  final String status;
  final String targetBranch;
  final String commit;
  final String errorMessage;
  final List<String> conflictFiles;

  const CodeDeliveryRepositoryResult({
    required this.repositoryName,
    required this.status,
    required this.targetBranch,
    required this.commit,
    required this.errorMessage,
    required this.conflictFiles,
  });

  factory CodeDeliveryRepositoryResult.fromJson(Map<String, dynamic> json) {
    return CodeDeliveryRepositoryResult(
      repositoryName: (json['repositoryName'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      targetBranch: (json['targetBranch'] ?? '').toString(),
      commit: (json['commit'] ?? '').toString(),
      errorMessage: (json['errorMessage'] ?? '').toString(),
      conflictFiles: (json['conflictFiles'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .where((item) => item.isNotEmpty)
          .toList(),
    );
  }
}

class CodeDeliveryJob {
  final int id;
  final int sessionId;
  final String status;
  final String stage;
  final int progress;
  final int attempt;
  final int queuePosition;
  final String targetBranch;
  final String resultCommit;
  final String resultType;
  final String errorMessage;
  final bool hasPendingChanges;
  final bool hasPendingCommits;
  final bool hasUncommittedChanges;
  final List<String> conflictFiles;
  final List<CodeDeliveryRepositoryResult> repositories;

  const CodeDeliveryJob({
    required this.id,
    required this.sessionId,
    required this.status,
    required this.stage,
    required this.progress,
    required this.attempt,
    required this.queuePosition,
    required this.targetBranch,
    required this.resultCommit,
    required this.resultType,
    required this.errorMessage,
    required this.hasPendingChanges,
    required this.hasPendingCommits,
    required this.hasUncommittedChanges,
    required this.conflictFiles,
    required this.repositories,
  });

  bool get isActive => status == 'queued' || status == 'running';
  bool get isCompleted => status == 'completed';
  bool get canRetry =>
      status == 'failed' || status == 'conflict' || status == 'partial';
  bool get canDeliverPending =>
      isCompleted && hasPendingCommits && !hasUncommittedChanges;

  factory CodeDeliveryJob.fromJson(Map<String, dynamic> json) {
    return CodeDeliveryJob(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      status: (json['status'] ?? '').toString(),
      stage: (json['stage'] ?? '').toString(),
      progress: (json['progress'] as num?)?.toInt() ?? 0,
      attempt: (json['attempt'] as num?)?.toInt() ?? 0,
      queuePosition: (json['queuePosition'] as num?)?.toInt() ?? 0,
      targetBranch: (json['targetBranch'] ?? '').toString(),
      resultCommit: (json['resultCommit'] ?? '').toString(),
      resultType: (json['resultType'] ?? '').toString(),
      errorMessage: (json['errorMessage'] ?? '').toString(),
      hasPendingChanges: json['hasPendingChanges'] == true,
      hasPendingCommits: json['hasPendingCommits'] == true,
      hasUncommittedChanges: json['hasUncommittedChanges'] == true,
      conflictFiles: (json['conflictFiles'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .where((item) => item.isNotEmpty)
          .toList(),
      repositories: (json['repositories'] as List<dynamic>? ?? const [])
          .whereType<Map>()
          .map(
            (item) => CodeDeliveryRepositoryResult.fromJson(
              item.cast<String, dynamic>(),
            ),
          )
          .toList(),
    );
  }
}
