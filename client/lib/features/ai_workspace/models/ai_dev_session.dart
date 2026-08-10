class AiDevSession {
  final int id;
  final int userId;
  final int projectId;
  final String title;
  final String agentName;
  final String workDir;
  final String sourceWorkDir;
  final String worktreeBranch;
  final String isolationMode;
  final String status;
  final String currentStage;
  final String approvalPolicy;
  final int lastTaskId;
  final String currentTaskTitle;
  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? lastInstructionAt;

  const AiDevSession({
    required this.id,
    required this.userId,
    required this.projectId,
    required this.title,
    required this.agentName,
    required this.workDir,
    required this.sourceWorkDir,
    required this.worktreeBranch,
    required this.isolationMode,
    required this.status,
    required this.currentStage,
    required this.approvalPolicy,
    required this.lastTaskId,
    required this.currentTaskTitle,
    required this.createdAt,
    required this.updatedAt,
    required this.lastInstructionAt,
  });

  factory AiDevSession.fromJson(Map<String, dynamic> json) {
    return AiDevSession(
      id: (json['id'] as num?)?.toInt() ?? 0,
      userId: (json['userId'] as num?)?.toInt() ?? 0,
      projectId: (json['projectId'] as num?)?.toInt() ?? 0,
      title: (json['title'] ?? '').toString(),
      agentName: (json['agentName'] ?? '').toString(),
      workDir: (json['workDir'] ?? '').toString(),
      sourceWorkDir: (json['sourceWorkDir'] ?? '').toString(),
      worktreeBranch: (json['worktreeBranch'] ?? '').toString(),
      isolationMode: (json['isolationMode'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      currentStage: (json['currentStage'] ?? '').toString(),
      approvalPolicy: (json['approvalPolicy'] ?? 'safe_auto').toString(),
      lastTaskId: (json['lastTaskId'] as num?)?.toInt() ?? 0,
      currentTaskTitle: (json['currentTaskTitle'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
      lastInstructionAt: DateTime.tryParse(
        (json['lastInstructionAt'] ?? '').toString(),
      ),
    );
  }
}

class CodeProject {
  final int id;
  final String name;
  final String description;
  final String workDir;
  final List<String> sourceDirs;

  const CodeProject({
    required this.id,
    required this.name,
    required this.description,
    required this.workDir,
    required this.sourceDirs,
  });

  factory CodeProject.fromJson(Map<String, dynamic> json) {
    return CodeProject(
      id: (json['id'] as num?)?.toInt() ?? 0,
      name: (json['name'] ?? '').toString(),
      description: (json['description'] ?? '').toString(),
      workDir: (json['workDir'] ?? '').toString(),
      sourceDirs: (json['sourceDirs'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .where((item) => item.isNotEmpty)
          .toList(),
    );
  }
}

class CodeWorktreeCapability {
  final bool available;
  final String reason;
  final List<String> sourceDirs;
  final int repositoryCount;
  final List<String> dirtyRepositories;
  final bool snapshotSupported;

  const CodeWorktreeCapability({
    required this.available,
    required this.reason,
    required this.sourceDirs,
    required this.repositoryCount,
    required this.dirtyRepositories,
    required this.snapshotSupported,
  });

  bool get canCreate => available && dirtyRepositories.isEmpty;

  factory CodeWorktreeCapability.fromJson(Map<String, dynamic> json) {
    return CodeWorktreeCapability(
      available: json['available'] == true,
      reason: (json['reason'] ?? '').toString(),
      sourceDirs: (json['sourceDirs'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .where((item) => item.isNotEmpty)
          .toList(),
      repositoryCount: (json['repositoryCount'] as num?)?.toInt() ?? 0,
      dirtyRepositories:
          (json['dirtyRepositories'] as List<dynamic>? ?? const [])
              .map((item) => item.toString())
              .where((item) => item.isNotEmpty)
              .toList(),
      snapshotSupported: json['snapshotSupported'] == true,
    );
  }
}

class CodeExecutor {
  final String id;
  final String name;
  final String description;
  final bool available;
  final String version;
  final String reason;
  final bool nativeTerminal;
  final List<String> capabilities;
  final List<String> approvalPolicies;

  const CodeExecutor({
    required this.id,
    required this.name,
    required this.description,
    required this.available,
    required this.version,
    required this.reason,
    required this.nativeTerminal,
    required this.capabilities,
    required this.approvalPolicies,
  });

  bool get supportsAutomation => capabilities.contains('automation');

  factory CodeExecutor.fromJson(Map<String, dynamic> json) {
    return CodeExecutor(
      id: (json['id'] ?? '').toString(),
      name: (json['name'] ?? '').toString(),
      description: (json['description'] ?? '').toString(),
      available: json['available'] == true,
      version: (json['version'] ?? '').toString(),
      reason: (json['reason'] ?? '').toString(),
      nativeTerminal: json['nativeTerminal'] == true,
      capabilities: (json['capabilities'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .toList(),
      approvalPolicies:
          (json['approvalPolicies'] as List<dynamic>? ?? const ['full_auto'])
              .map((item) => item.toString())
              .toList(),
    );
  }
}

class AiInstruction {
  final int id;
  final int sessionId;
  final int taskId;
  final String content;
  final String status;
  final bool allowCode;
  final bool autoPreview;
  final bool requireApproval;
  final bool analysisOnly;
  final DateTime? createdAt;

  const AiInstruction({
    required this.id,
    required this.sessionId,
    required this.taskId,
    required this.content,
    required this.status,
    required this.allowCode,
    required this.autoPreview,
    required this.requireApproval,
    required this.analysisOnly,
    required this.createdAt,
  });

  factory AiInstruction.fromJson(Map<String, dynamic> json) {
    return AiInstruction(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      taskId: (json['taskId'] as num?)?.toInt() ?? 0,
      content: (json['content'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      allowCode: json['allowCode'] == true,
      autoPreview: json['autoPreview'] == true,
      requireApproval: json['requireApproval'] == true,
      analysisOnly: json['analysisOnly'] == true,
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
    );
  }
}

class AiTaskSummary {
  final int id;
  final String title;
  final String status;
  final String workDir;
  final String agentName;

  const AiTaskSummary({
    required this.id,
    required this.title,
    required this.status,
    required this.workDir,
    required this.agentName,
  });

  factory AiTaskSummary.fromJson(Map<String, dynamic> json) {
    return AiTaskSummary(
      id: (json['id'] as num?)?.toInt() ?? 0,
      title: (json['title'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      workDir: (json['workDir'] ?? '').toString(),
      agentName: (json['agentName'] ?? '').toString(),
    );
  }
}

class AiPreview {
  final int id;
  final int sessionId;
  final int taskId;
  final int instructionId;
  final String previewType;
  final String source;
  final String title;
  final String url;
  final String status;
  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? lastCheckedAt;

  const AiPreview({
    required this.id,
    required this.sessionId,
    required this.taskId,
    required this.instructionId,
    required this.previewType,
    required this.source,
    required this.title,
    required this.url,
    required this.status,
    required this.createdAt,
    required this.updatedAt,
    required this.lastCheckedAt,
  });

  factory AiPreview.fromJson(Map<String, dynamic> json) {
    return AiPreview(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      taskId: (json['taskId'] as num?)?.toInt() ?? 0,
      instructionId: (json['instructionId'] as num?)?.toInt() ?? 0,
      previewType: (json['previewType'] ?? '').toString(),
      source: (json['source'] ?? '').toString(),
      title: (json['title'] ?? '').toString(),
      url: (json['url'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
      lastCheckedAt: DateTime.tryParse(
        (json['lastCheckedAt'] ?? '').toString(),
      ),
    );
  }
}

class AiTimelineEvent {
  final int id;
  final int sessionId;
  final int taskId;
  final int instructionId;
  final String eventType;
  final String stage;
  final String title;
  final String content;
  final String status;
  final String meta;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const AiTimelineEvent({
    required this.id,
    required this.sessionId,
    required this.taskId,
    required this.instructionId,
    required this.eventType,
    required this.stage,
    required this.title,
    required this.content,
    required this.status,
    required this.meta,
    required this.createdAt,
    required this.updatedAt,
  });

  factory AiTimelineEvent.fromJson(Map<String, dynamic> json) {
    return AiTimelineEvent(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      taskId: (json['taskId'] as num?)?.toInt() ?? 0,
      instructionId: (json['instructionId'] as num?)?.toInt() ?? 0,
      eventType: (json['eventType'] ?? '').toString(),
      stage: (json['stage'] ?? '').toString(),
      title: (json['title'] ?? '').toString(),
      content: (json['content'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      meta: (json['meta'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
    );
  }
}

class AiApproval {
  final int id;
  final int sessionId;
  final int taskId;
  final int instructionId;
  final int requestUserId;
  final int approveUserId;
  final String title;
  final String content;
  final String riskLevel;
  final String status;
  final String decision;
  final String decisionReason;
  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? decisionAt;

  const AiApproval({
    required this.id,
    required this.sessionId,
    required this.taskId,
    required this.instructionId,
    required this.requestUserId,
    required this.approveUserId,
    required this.title,
    required this.content,
    required this.riskLevel,
    required this.status,
    required this.decision,
    required this.decisionReason,
    required this.createdAt,
    required this.updatedAt,
    required this.decisionAt,
  });

  factory AiApproval.fromJson(Map<String, dynamic> json) {
    return AiApproval(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sessionId: (json['sessionId'] as num?)?.toInt() ?? 0,
      taskId: (json['taskId'] as num?)?.toInt() ?? 0,
      instructionId: (json['instructionId'] as num?)?.toInt() ?? 0,
      requestUserId: (json['requestUserId'] as num?)?.toInt() ?? 0,
      approveUserId: (json['approveUserId'] as num?)?.toInt() ?? 0,
      title: (json['title'] ?? '').toString(),
      content: (json['content'] ?? '').toString(),
      riskLevel: (json['riskLevel'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      decision: (json['decision'] ?? '').toString(),
      decisionReason: (json['decisionReason'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
      decisionAt: DateTime.tryParse((json['decisionAt'] ?? '').toString()),
    );
  }
}
