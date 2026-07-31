import '../../../core/network/api_client.dart';
import '../models/database_info.dart';

class DatabaseRepository {
  final ApiClient _apiClient;

  DatabaseRepository(this._apiClient);

  Future<List<DatabaseInfo>> listDatabases() async {
    final res = await _apiClient.post<List<dynamic>>(
      '/api/database/list',
      data: {},
    );
    final data = res.data ?? const [];
    return data
        .whereType<Map>()
        .map((e) => DatabaseInfo.fromJson(e.cast<String, dynamic>()))
        .toList();
  }

  Future<void> syncServer(int serverId) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/database/server/sync',
      data: {'id': serverId},
    );
  }
}
