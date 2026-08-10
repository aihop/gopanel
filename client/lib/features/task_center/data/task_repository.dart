import '../../../core/network/api_client.dart';
import '../../ai_workspace/models/ai_dev_session.dart';
import '../../ai_workspace/models/ai_session_state_info.dart';
import '../../ai_workspace/models/code_task.dart';
import 'task_ai_utils.dart';
import 'task_attention_repository.dart';
import '../models/task_entity.dart';
import '../models/task_attention.dart';
import '../models/task_log.dart';
import '../models/task_status.dart';
import '../models/task_type.dart';

const pipelineListPath = '/api/pipeline/list';

class TaskRepository {
  final ApiClient _apiClient;

  TaskRepository(this._apiClient);

  Future<void> executeAttentionAction(TaskAttentionAction action) {
    return TaskAttentionRepository(_apiClient).execute(action);
  }

  Future<List<TaskEntity>> list({
    int maxPipelines = 10,
    int recordsPerPipeline = 3,
  }) async {
    final pipelinesRes = await _apiClient.get<Map<String, dynamic>>(
      pipelineListPath,
      queryParameters: {'page': 1, 'pageSize': maxPipelines},
    );
    final pipelinesData = pipelinesRes.data ?? const <String, dynamic>{};
    final pipelineItems =
        (pipelinesData['items'] as List<dynamic>? ?? const []);
    final pipelines = pipelineItems
        .map((e) => _Pipeline.fromJson((e as Map).cast<String, dynamic>()))
        .toList();

    final tasksNested = await _mapWithConcurrency<_Pipeline, List<TaskEntity>>(
      pipelines,
      concurrency: 5,
      mapper: (p) async {
        if (p.id <= 0) return const [];
        final recordsRes = await _apiClient.get<Map<String, dynamic>>(
          '/api/pipeline/records',
          queryParameters: {
            'pipelineId': p.id,
            'page': 1,
            'pageSize': recordsPerPipeline,
          },
        );
        final recordsData = recordsRes.data ?? const <String, dynamic>{};
        final recordItems =
            (recordsData['items'] as List<dynamic>? ?? const []);
        final records = recordItems
            .map(
              (e) =>
                  _PipelineRecord.fromJson((e as Map).cast<String, dynamic>()),
            )
            .toList();

        return records
            .map(
              (r) => TaskEntity(
                id: 'pipeline:${r.id}',
                title: '${p.name} #${r.id}',
                type: TaskType.pipeline,
                status: _mapPipelineStatus(r.status),
                progress: _pipelineProgress(r.status),
                startedAt: r.createdAt,
                updatedAt: r.updatedAt,
                summary: r.version.isNotEmpty
                    ? 'version ${r.version}'
                    : r.status,
                error: r.errorMessage.isNotEmpty ? r.errorMessage : null,
              ),
            )
            .toList();
      },
    );

    final tasks = <TaskEntity>[];
    for (final group in tasksNested) {
      tasks.addAll(group);
    }

    final aiTasksRes = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/tasks',
      queryParameters: {'page': 1, 'limit': 100, 'includeGit': false},
    );
    final aiTasksData = aiTasksRes.data ?? const <String, dynamic>{};
    final aiTaskItems = (aiTasksData['items'] as List<dynamic>? ?? const [])
        .whereType<Map>()
        .map((item) => CodeTask.fromJson(item.cast<String, dynamic>()))
        .where((task) => task.id > 0 && task.agentName != 'terminal')
        .toList();

    final attentionItems = await TaskAttentionRepository(_apiClient).list();
    final attentionByTask = {
      for (final attention in attentionItems)
        if (attention.taskId > 0) attention.taskId: attention,
    };

    final codeTasks = await _mapWithConcurrency<CodeTask, TaskEntity>(
      aiTaskItems,
      concurrency: 4,
      mapper: (task) async {
        AiSessionStateInfo? state;
        if (task.sessionId > 0) {
          try {
            state = await getAiSessionState(task.sessionId);
          } catch (_) {
            state = null;
          }
        }

        return buildAiTaskEntity(
          taskId: task.id,
          sessionId: task.sessionId,
          projectId: task.projectId,
          agentName: task.agentName,
          title: task.title,
          workDir: task.workDir,
          status: task.status,
          currentStage: state?.currentStage ?? '',
          createdAt: task.createdAt,
          updatedAt: task.updatedAt,
          state: state,
          attention: attentionByTask[task.id],
        );
      },
    );

    tasks.addAll(codeTasks);
    final loadedTaskIds = aiTaskItems.map((task) => task.id).toSet();
    tasks.addAll(
      attentionItems
          .where(
            (attention) =>
                attention.taskId <= 0 ||
                !loadedTaskIds.contains(attention.taskId),
          )
          .map(_buildAttentionOnlyTask),
    );

    tasks.sort((a, b) {
      final at =
          a.updatedAt ?? a.startedAt ?? DateTime.fromMillisecondsSinceEpoch(0);
      final bt =
          b.updatedAt ?? b.startedAt ?? DateTime.fromMillisecondsSinceEpoch(0);
      return bt.compareTo(at);
    });
    return tasks;
  }

  Future<TaskLog> getLog(String taskId) async {
    if (taskId.startsWith('pipeline:')) {
      final recordId = int.tryParse(taskId.substring('pipeline:'.length));
      if (recordId == null || recordId <= 0) {
        return const TaskLog(taskId: '', lines: []);
      }
      final lines = await _apiClient.getSseDataLines(
        '/api/pipeline/logs',
        queryParameters: {'recordId': recordId},
        timeout: const Duration(seconds: 25),
      );
      return TaskLog(taskId: taskId, lines: lines);
    }

    if (taskId.startsWith('ssl:')) {
      final id = int.tryParse(taskId.substring('ssl:'.length));
      if (id == null || id <= 0) {
        return const TaskLog(taskId: '', lines: []);
      }
      final lines = await _apiClient.getSseDataLines(
        '/api/ssl/$id/logs',
        timeout: const Duration(seconds: 25),
      );
      return TaskLog(taskId: taskId, lines: lines);
    }

    if (taskId.startsWith('appInstall:')) {
      final name = taskId.substring('appInstall:'.length);
      if (name.isEmpty) return const TaskLog(taskId: '', lines: []);
      final lines = await _apiClient.getSseDataLines(
        '/api/apps/install/$name/logs',
        timeout: const Duration(seconds: 25),
      );
      return TaskLog(taskId: taskId, lines: lines);
    }

    if (taskId.startsWith('upgrade:')) {
      final log = taskId.substring('upgrade:'.length);
      if (log.isEmpty) return const TaskLog(taskId: '', lines: []);
      final lines = await _apiClient.getSseDataLines(
        '/api/setting/system/upgrade/logs',
        queryParameters: {'log': log},
        timeout: const Duration(seconds: 25),
      );
      return TaskLog(taskId: taskId, lines: lines);
    }

    if (taskId.startsWith('websiteDeploy:')) {
      final websiteId = int.tryParse(taskId.substring('websiteDeploy:'.length));
      if (websiteId == null || websiteId <= 0) {
        return const TaskLog(taskId: '', lines: []);
      }
      final res = await _apiClient.post<List<dynamic>>(
        '/api/website/deploy/list',
        data: {'websiteId': websiteId},
      );
      final list = res.data ?? const [];
      if (list.isEmpty) {
        return const TaskLog(
          taskId: '',
          lines: ['暂无部署记录'],
          status: TaskStatus.running,
        );
      }
      final first = (list.first as Map).cast<String, dynamic>();
      final statusText = (first['status'] ?? '').toString();
      final version = (first['version'] ?? '').toString();
      final isActive = (first['isActive'] as bool?) ?? false;
      final logText = (first['logText'] ?? '').toString();
      final lines = logText.isEmpty ? const <String>[] : logText.split('\n');
      return TaskLog(
        taskId: taskId,
        lines: lines,
        status: _mapWebsiteDeployStatus(statusText),
        meta: {
          'deployStatus': statusText,
          'deployVersion': version,
          'deployActive': isActive ? 'true' : 'false',
        },
      );
    }

    if (taskId.startsWith('dbSync:')) {
      return const TaskLog(taskId: '', lines: ['该操作暂无服务端日志流，状态以接口返回为准。']);
    }

    if (taskId.startsWith('aiSession:')) {
      final sessionId = int.tryParse(taskId.substring('aiSession:'.length));
      if (sessionId == null || sessionId <= 0) {
        return const TaskLog(taskId: '', lines: []);
      }
      final state = await getAiSessionState(sessionId);
      return buildAiTaskLog(taskId: taskId, state: state);
    }

    return TaskLog(taskId: taskId, lines: const ['暂无可用日志源']);
  }

  Future<AiSessionStateInfo> getAiSessionState(int sessionId) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/state',
    );
    return AiSessionStateInfo.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<List<AiPreview>> getAiSessionPreviews(int sessionId) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/previews',
    );
    final data = response.data;
    if (data != null && data['items'] is List) {
      final items = data['items'] as List;
      return items
          .whereType<Map>()
          .map((item) => AiPreview.fromJson(item.cast<String, dynamic>()))
          .toList();
    }
    return [];
  }
}

TaskEntity _buildAttentionOnlyTask(TaskAttention attention) {
  final failed = attention.severity == 'error';
  return TaskEntity(
    id: 'aiSession:${attention.sessionId}',
    title: '开发会话 #${attention.sessionId}',
    type: TaskType.ai,
    status: failed ? TaskStatus.failed : TaskStatus.running,
    updatedAt: attention.updatedAt,
    summary: attention.summary,
    error: failed && attention.summary.isNotEmpty ? attention.summary : null,
    attention: attention,
    attentionOnly: true,
    meta: {
      'sessionId': attention.sessionId.toString(),
      'currentStageLabel': attention.title,
      'previewCount': '0',
    },
  );
}

TaskStatus _mapWebsiteDeployStatus(String s) {
  final v = s.toLowerCase();
  if (v == 'running') return TaskStatus.success;
  if (v == 'failed') return TaskStatus.failed;
  return TaskStatus.running;
}

Future<List<R>> _mapWithConcurrency<T, R>(
  List<T> items, {
  required int concurrency,
  required Future<R> Function(T item) mapper,
}) async {
  if (items.isEmpty) return const [];
  final results = List<R?>.filled(items.length, null);
  var nextIndex = 0;

  Future<void> worker() async {
    while (true) {
      final i = nextIndex++;
      if (i >= items.length) return;
      results[i] = await mapper(items[i]);
    }
  }

  final workers = List.generate(
    concurrency.clamp(1, items.length),
    (_) => worker(),
  );
  await Future.wait(workers);
  return results.cast<R>();
}

class _Pipeline {
  final int id;
  final String name;
  final String description;
  final String version;

  const _Pipeline({
    required this.id,
    required this.name,
    required this.description,
    required this.version,
  });

  factory _Pipeline.fromJson(Map<String, dynamic> json) {
    return _Pipeline(
      id: (json['id'] as num?)?.toInt() ?? 0,
      name: (json['name'] ?? '').toString(),
      description: (json['description'] ?? '').toString(),
      version: (json['version'] ?? '').toString(),
    );
  }
}

class _PipelineRecord {
  final int id;
  final int pipelineId;
  final String status;
  final String version;
  final String errorMessage;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const _PipelineRecord({
    required this.id,
    required this.pipelineId,
    required this.status,
    required this.version,
    required this.errorMessage,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _PipelineRecord.fromJson(Map<String, dynamic> json) {
    return _PipelineRecord(
      id: (json['id'] as num?)?.toInt() ?? 0,
      pipelineId: (json['pipelineId'] as num?)?.toInt() ?? 0,
      status: (json['status'] ?? '').toString(),
      version: (json['version'] ?? '').toString(),
      errorMessage: (json['errorMessage'] ?? '').toString(),
      createdAt: DateTime.tryParse((json['createdAt'] ?? '').toString()),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
    );
  }
}

TaskStatus _mapPipelineStatus(String status) {
  switch (status) {
    case 'success':
      return TaskStatus.success;
    case 'failed':
      return TaskStatus.failed;
    default:
      return TaskStatus.running;
  }
}

double? _pipelineProgress(String status) {
  switch (status) {
    case 'pending':
      return 0.05;
    case 'cloning':
      return 0.25;
    case 'building':
      return 0.55;
    case 'deploying':
      return 0.8;
    case 'success':
      return 1;
    case 'failed':
      return null;
    default:
      return null;
  }
}
