import 'ai_dev_session.dart';
import 'chat_message.dart';

class AiSessionStateInfo {
  final AiDevSession session;
  final AiTaskSummary? currentTask;
  final AiInstruction? latestInstruction;
  final String currentStage;
  final String recentOutput;
  final List<ChatMessage> recentMessages;
  final List<AiPreview> previews;
  final List<AiTimelineEvent> timelineEvents;
  final String errorSummary;
  final List<String> changedFiles;
  final AiApproval? pendingApproval;

  const AiSessionStateInfo({
    required this.session,
    required this.currentTask,
    required this.latestInstruction,
    required this.currentStage,
    required this.recentOutput,
    required this.recentMessages,
    required this.previews,
    required this.timelineEvents,
    required this.errorSummary,
    required this.changedFiles,
    required this.pendingApproval,
  });

  factory AiSessionStateInfo.fromJson(Map<String, dynamic> json) {
    final recentMessagesRaw =
        json['recentMessages'] as List<dynamic>? ?? const [];
    final previewsRaw = json['previews'] as List<dynamic>? ?? const [];
    final timelineRaw = json['timelineEvents'] as List<dynamic>? ?? const [];

    return AiSessionStateInfo(
      session: AiDevSession.fromJson(
        (json['session'] as Map? ?? const {}).cast<String, dynamic>(),
      ),
      currentTask: json['currentTask'] is Map<String, dynamic>
          ? AiTaskSummary.fromJson(json['currentTask'] as Map<String, dynamic>)
          : json['currentTask'] is Map
          ? AiTaskSummary.fromJson(
              (json['currentTask'] as Map).cast<String, dynamic>(),
            )
          : null,
      latestInstruction: json['latestInstruction'] is Map<String, dynamic>
          ? AiInstruction.fromJson(
              json['latestInstruction'] as Map<String, dynamic>,
            )
          : json['latestInstruction'] is Map
          ? AiInstruction.fromJson(
              (json['latestInstruction'] as Map).cast<String, dynamic>(),
            )
          : null,
      currentStage: (json['currentStage'] ?? '').toString(),
      recentOutput: (json['recentOutput'] ?? '').toString(),
      recentMessages: recentMessagesRaw
          .whereType<Map>()
          .map(
            (item) =>
                ChatMessage.fromAiMessageJson(item.cast<String, dynamic>()),
          )
          .toList(),
      previews: previewsRaw
          .whereType<Map>()
          .map((item) => AiPreview.fromJson(item.cast<String, dynamic>()))
          .toList(),
      timelineEvents: timelineRaw
          .whereType<Map>()
          .map((item) => AiTimelineEvent.fromJson(item.cast<String, dynamic>()))
          .toList(),
      errorSummary: (json['errorSummary'] ?? '').toString(),
      changedFiles: (json['changedFiles'] as List<dynamic>? ?? const [])
          .map((item) => item.toString())
          .where((item) => item.trim().isNotEmpty)
          .toList(),
      pendingApproval: json['pendingApproval'] is Map<String, dynamic>
          ? AiApproval.fromJson(json['pendingApproval'] as Map<String, dynamic>)
          : json['pendingApproval'] is Map
          ? AiApproval.fromJson(
              (json['pendingApproval'] as Map).cast<String, dynamic>(),
            )
          : null,
    );
  }
}

class AiInstructionSendResult {
  final AiDevSession session;
  final AiInstruction instruction;
  final AiTaskSummary? task;
  final AiApproval? approval;

  const AiInstructionSendResult({
    required this.session,
    required this.instruction,
    required this.task,
    required this.approval,
  });

  factory AiInstructionSendResult.fromJson(Map<String, dynamic> json) {
    return AiInstructionSendResult(
      session: AiDevSession.fromJson(
        (json['session'] as Map? ?? const {}).cast<String, dynamic>(),
      ),
      instruction: AiInstruction.fromJson(
        (json['instruction'] as Map? ?? const {}).cast<String, dynamic>(),
      ),
      task: json['task'] is Map<String, dynamic>
          ? AiTaskSummary.fromJson(json['task'] as Map<String, dynamic>)
          : json['task'] is Map
          ? AiTaskSummary.fromJson(
              (json['task'] as Map).cast<String, dynamic>(),
            )
          : null,
      approval: json['approval'] is Map<String, dynamic>
          ? AiApproval.fromJson(json['approval'] as Map<String, dynamic>)
          : json['approval'] is Map
          ? AiApproval.fromJson(
              (json['approval'] as Map).cast<String, dynamic>(),
            )
          : null,
    );
  }
}
