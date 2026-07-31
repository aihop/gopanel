import '../../../core/network/api_client.dart';
import '../models/website_info.dart';

const websiteListPath = '/api/website/list';

List<WebsiteInfo> parseWebsiteList(Map<String, dynamic>? data) {
  final items = data?['items'];
  if (items is! List) return [];
  return items
      .whereType<Map>()
      .map((item) => WebsiteInfo.fromJson(item.cast<String, dynamic>()))
      .toList();
}

class WebsiteRepository {
  final ApiClient _apiClient;

  WebsiteRepository(this._apiClient);

  Future<List<WebsiteInfo>> listWebsites() async {
    final res = await _apiClient.post<Map<String, dynamic>>(
      websiteListPath,
      data: {},
    );
    return parseWebsiteList(res.data);
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
