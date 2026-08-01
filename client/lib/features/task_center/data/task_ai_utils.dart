import '../../ai_workspace/models/ai_session_state_info.dart';
import '../models/task_entity.dart';
import '../models/task_log.dart';
import '../models/task_status.dart';
import '../models/task_type.dart';

TaskEntity buildAiTaskEntity({
  required int sessionId,
  required String title,
  required String workDir,
  required String status,
  required String currentStage,
  required DateTime? createdAt,
  required DateTime? updatedAt,
  AiSessionStateInfo? state,
}) {
  final resolvedStage = state?.currentStage ?? currentStage;
  final resolvedStatus = state?.session.status ?? status;
  final previewCount = state?.previews.length ?? 0;
  final latestInstruction = state?.latestInstruction?.content.trim() ?? '';
  final latestEventTitle = state?.timelineEvents.isNotEmpty == true
      ? state!.timelineEvents.first.title.trim()
      : '';
  final errorSummary = state?.errorSummary.trim() ?? '';
  final changedFiles = state?.changedFiles ?? const <String>[];

  final summary = () {
    if (errorSummary.isNotEmpty) return errorSummary;
    if (latestInstruction.isNotEmpty) return latestInstruction;
    if (latestEventTitle.isNotEmpty) return latestEventTitle;
    final parts = <String>[
      aiStageLabel(resolvedStage),
      if (previewCount > 0) '$previewCount 个预览',
      if (workDir.isNotEmpty) workDir,
    ];
    return parts.join(' · ');
  }();

  return TaskEntity(
    id: 'aiSession:$sessionId',
    title: title.isEmpty ? '开发会话 #$sessionId' : title,
    type: TaskType.ai,
    status: mapAiSessionStatus(resolvedStatus, resolvedStage),
    progress: aiSessionProgress(resolvedStage),
    startedAt: createdAt,
    updatedAt: updatedAt,
    summary: summary,
    error: errorSummary.isNotEmpty
        ? errorSummary
        : (resolvedStage == 'failed' ? '开发会话执行失败' : null),
    meta: {
      'sessionId': sessionId.toString(),
      'currentStage': resolvedStage,
      'currentStageLabel': aiStageLabel(resolvedStage),
      'previewCount': previewCount.toString(),
      'workDir': workDir,
      if (latestInstruction.isNotEmpty) 'latestInstruction': latestInstruction,
      if (latestEventTitle.isNotEmpty) 'latestEventTitle': latestEventTitle,
      if (errorSummary.isNotEmpty) 'errorSummary': errorSummary,
      if (changedFiles.isNotEmpty) 'changedFiles': changedFiles.join('\n'),
    },
  );
}

TaskLog buildAiTaskLog({
  required String taskId,
  required AiSessionStateInfo state,
}) {
  final lines = <String>[
    '会话 #${state.session.id}',
    '阶段: ${state.currentStage.isEmpty ? '-' : aiStageLabel(state.currentStage)}',
  ];

  if (state.currentTask != null) {
    lines.add('任务 #${state.currentTask!.id}');
  }
  if (state.errorSummary.trim().isNotEmpty) {
    lines
      ..add('')
      ..add('错误摘要:')
      ..add(state.errorSummary.trim());
  }
  if (state.changedFiles.isNotEmpty) {
    lines
      ..add('')
      ..add('涉及文件:');
    for (final file in state.changedFiles) {
      lines.add('- $file');
    }
  }
  if (state.timelineEvents.isNotEmpty) {
    final recentTimeline = state.timelineEvents.take(6).toList().reversed;
    lines
      ..add('')
      ..add('时间线:');
    for (final event in recentTimeline) {
      final title = event.title.trim().isEmpty ? event.eventType : event.title.trim();
      lines.add('- [$title] ${event.content.trim()}'.trim());
    }
  }
  if (state.recentOutput.trim().isNotEmpty) {
    lines
      ..add('')
      ..add('最近输出:')
      ..add(state.recentOutput.trim());
  }

  return TaskLog(
    taskId: taskId,
    lines: lines,
    status: mapAiSessionStatus(state.session.status, state.currentStage),
    meta: {
      'sessionId': state.session.id.toString(),
      'currentStage': state.currentStage,
      'currentStageLabel': aiStageLabel(state.currentStage),
      'previewCount': state.previews.length.toString(),
      if (state.errorSummary.trim().isNotEmpty)
        'errorSummary': state.errorSummary.trim(),
      if (state.changedFiles.isNotEmpty)
        'changedFiles': state.changedFiles.join('\n'),
      if (state.timelineEvents.isNotEmpty)
        'latestEventTitle': state.timelineEvents.first.title.trim(),
    },
  );
}

TaskStatus mapAiSessionStatus(String status, String currentStage) {
  final stage = currentStage.toLowerCase();
  final normalized = status.toLowerCase();
  if (stage == 'failed' || normalized == 'failed') return TaskStatus.failed;
  if (stage == 'completed' || stage == 'preview_ready') return TaskStatus.success;
  return TaskStatus.running;
}

double? aiSessionProgress(String currentStage) {
  switch (currentStage) {
    case 'idle':
      return 0.05;
    case 'instruction_queued':
      return 0.15;
    case 'executing':
      return 0.65;
    case 'completed':
    case 'preview_ready':
      return 1.0;
    case 'failed':
      return null;
    default:
      return 0.35;
  }
}

String aiStageLabel(String currentStage) {
  switch (currentStage) {
    case 'instruction_queued':
      return '指令排队中';
    case 'executing':
      return '执行中';
    case 'preview_ready':
      return '预览已生成';
    case 'completed':
      return '已完成';
    case 'failed':
      return '执行失败';
    case 'idle':
      return '空闲';
    default:
      return currentStage.isEmpty ? '会话中' : currentStage;
  }
}
