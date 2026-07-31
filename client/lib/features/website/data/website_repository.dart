import '../../../core/network/api_client.dart';
import '../models/website_info.dart';

class WebsiteRepository {
  final ApiClient _apiClient;

  WebsiteRepository(this._apiClient);

  Future<List<WebsiteInfo>> listWebsites() async {
    final res = await _apiClient.post<List<dynamic>>(
      '/api/website/list',
      data: {},
    );
    final data = res.data ?? const [];
    return data
        .whereType<Map>()
        .map((e) => WebsiteInfo.fromJson(e.cast<String, dynamic>()))
        .toList();
  }

  Future<int> runPipeline(int pipelineId) async {
    final res = await _apiClient.post<Map<String, dynamic>>(
      '/api/pipeline/run',
      data: {'id': pipelineId},
    );
    final data = res.data ?? const <String, dynamic>{};
    return (data['recordId'] as num?)?.toInt() ?? 0;
  }

  Future<void> deployTrigger({
    required int websiteId,
    String zipPath = '',
    String imageTag = '',
  }) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/website/deploy/trigger',
      data: {'websiteId': websiteId, 'zipPath': zipPath, 'imageTag': imageTag},
    );
  }
}
