import '../../../core/network/api_client.dart';
import '../models/ai_dev_session.dart';
import '../models/ai_session_state_info.dart';
import '../models/code_workspace_file.dart';
import '../models/code_delivery_job.dart';
import '../models/code_project_terminal_session.dart';

/// AI 工作区仓库
/// 负责读取服务器目录、执行远程 AI 任务等 API 交互
class AiWorkspaceRepository {
  final ApiClient _apiClient;

  AiWorkspaceRepository(this._apiClient);

  Future<List<CodeProject>> getProjects() async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/projects',
      queryParameters: {'page': 1, 'limit': 100},
    );
    return _items(response.data, CodeProject.fromJson);
  }

  Future<List<CodeExecutor>> getExecutors() async {
    final response = await _apiClient.get<List<dynamic>>('/api/code/executors');
    return (response.data ?? const [])
        .whereType<Map>()
        .map((item) => CodeExecutor.fromJson(item.cast<String, dynamic>()))
        .toList();
  }

  Future<CodeWorktreeCapability> getWorktreeCapability(int projectId) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/projects/$projectId/worktree-capability',
    );
    return CodeWorktreeCapability.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<CodeProjectTerminalSession> openProjectTerminal(int projectId) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/projects/$projectId/terminal',
      data: const <String, dynamic>{},
    );
    return CodeProjectTerminalSession.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<List<AiDevSession>> getSessions() async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions',
      queryParameters: {'page': 1, 'limit': 50},
    );
    return _items(response.data, AiDevSession.fromJson);
  }

  Future<AiDevSession> createSession({
    required int projectId,
    required String executorId,
    required String approvalPolicy,
    String title = '',
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/sessions',
      data: {
        'title': title,
        'workDir': '',
        'projectId': projectId,
        'executorId': executorId,
        'approvalPolicy': approvalPolicy,
        'isolated': false,
      },
    );

    return AiDevSession.fromJson(response.data ?? const <String, dynamic>{});
  }

  Future<AiSessionStateInfo> getSessionState(int sessionId) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/state',
    );
    return AiSessionStateInfo.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<CodeStructureResult> getSessionStructure(
    int sessionId, {
    String path = '',
  }) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/structure',
      queryParameters: {'path': path},
    );
    return CodeStructureResult.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<CodeSessionFile> getSessionFile(int sessionId, String path) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/file',
      queryParameters: {'path': path},
    );
    return CodeSessionFile.fromJson(response.data ?? const <String, dynamic>{});
  }

  Future<List<AiPreview>> getSessionPreviews(int sessionId) async {
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

  Future<List<AiApproval>> getApprovals({
    String status = 'pending',
    int limit = 50,
  }) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/approvals',
      queryParameters: {'status': status, 'limit': limit},
    );
    final data = response.data;
    if (data != null && data['items'] is List) {
      final items = data['items'] as List;
      return items
          .whereType<Map>()
          .map((item) => AiApproval.fromJson(item.cast<String, dynamic>()))
          .toList();
    }
    return [];
  }

  Future<AiApproval> approveApproval(int approvalId, {String? reason}) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/approvals/$approvalId/approve',
      data: {'reason': reason ?? ''},
    );
    final data = response.data ?? const <String, dynamic>{};
    return AiApproval.fromJson(
      (data['approval'] as Map? ?? const {}).cast<String, dynamic>(),
    );
  }

  Future<AiApproval> rejectApproval(int approvalId, {String? reason}) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/approvals/$approvalId/reject',
      data: {'reason': reason ?? ''},
    );
    final data = response.data ?? const <String, dynamic>{};
    return AiApproval.fromJson(
      (data['approval'] as Map? ?? const {}).cast<String, dynamic>(),
    );
  }

  Future<AiInstructionSendResult> sendAiCommand({
    required int sessionId,
    required String command,
    bool allowCode = true,
    bool autoPreview = true,
    bool requireApproval = false,
    bool analysisOnly = false,
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/instructions',
      data: {
        'content': command,
        'allowCode': allowCode,
        'autoPreview': autoPreview,
        'requireApproval': requireApproval,
        'analysisOnly': analysisOnly,
      },
    );
    return AiInstructionSendResult.fromJson(
      response.data ?? const <String, dynamic>{},
    );
  }

  Future<void> stopSession(int sessionId) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/stop',
      data: const <String, dynamic>{},
    );
  }

  Future<CodeDeliveryJob?> getDelivery(int sessionId) async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/delivery',
    );
    final data = response.data;
    return data == null ? null : CodeDeliveryJob.fromJson(data);
  }

  Future<CodeDeliveryJob> startDelivery(int sessionId) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/code/sessions/$sessionId/worktree/merge',
      data: const {'confirm': true},
    );
    return CodeDeliveryJob.fromJson(response.data ?? const <String, dynamic>{});
  }

  List<T> _items<T>(
    Map<String, dynamic>? data,
    T Function(Map<String, dynamic>) fromJson,
  ) {
    return (data?['items'] as List<dynamic>? ?? const [])
        .whereType<Map>()
        .map((item) => fromJson(item.cast<String, dynamic>()))
        .toList();
  }
}
