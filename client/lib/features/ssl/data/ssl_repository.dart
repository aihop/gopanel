import '../../../core/network/api_client.dart';
import '../models/ssl_info.dart';

class SslRepository {
  final ApiClient _apiClient;

  SslRepository(this._apiClient);

  Future<List<SslInfo>> list() async {
    final res = await _apiClient.post<List<dynamic>>(
      '/api/ssl/list',
      data: {},
    );
    final data = res.data ?? const [];
    return data
        .whereType<Map>()
        .map((e) => SslInfo.fromJson(e.cast<String, dynamic>()))
        .toList();
  }

  Future<void> renew(int id) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/ssl/renew',
      data: {'id': id},
    );
  }

  Future<List<String>> getLogs(int id) async {
    return _apiClient.getSseDataLines(
      '/api/ssl/$id/logs',
      timeout: const Duration(seconds: 25),
    );
  }
}

