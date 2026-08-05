import '../../../core/network/api_client.dart';
import '../models/task_attention.dart';

const taskAttentionListPath = '/api/code/attention';

class TaskAttentionRepository {
  final ApiClient _apiClient;

  TaskAttentionRepository(this._apiClient);

  Future<List<TaskAttention>> list() async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      taskAttentionListPath,
      queryParameters: {'limit': 100},
    );
    return parseTaskAttentionList(response.data);
  }

  Future<void> execute(TaskAttentionAction action) async {
    if (action.method != 'POST' || !action.path.startsWith('/api/code/')) {
      throw StateError('当前动作不能直接执行');
    }
    await _apiClient.post<Map<String, dynamic>>(
      action.path,
      data: const <String, dynamic>{},
    );
  }
}
