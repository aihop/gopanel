import '../../../core/network/api_client.dart';
import '../models/database_info.dart';

const databaseListPath = '/api/database/list';

List<DatabaseInfo> parseDatabaseList(Map<String, dynamic>? data) {
  final items = data?['items'];
  if (items is! List) return [];
  return items
      .whereType<Map>()
      .map((item) => DatabaseInfo.fromJson(item.cast<String, dynamic>()))
      .toList();
}

class DatabaseRepository {
  final ApiClient _apiClient;

  DatabaseRepository(this._apiClient);

  Future<List<DatabaseInfo>> listDatabases() async {
    final res = await _apiClient.post<Map<String, dynamic>>(
      databaseListPath,
      data: {'page': 1, 'limit': 100},
    );
    return parseDatabaseList(res.data);
  }

  Future<void> syncServer(int serverId) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/database/server/sync',
      data: {'id': serverId},
    );
  }
}
