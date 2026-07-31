import '../../../core/network/api_client.dart';
import '../models/app_install_info.dart';

const installedAppsListPath = '/api/apps/installed/list';

List<AppInstallInfo> parseInstalledApps(
  Map<String, dynamic>? data, {
  String? status,
}) {
  final items = data?['items'];
  if (items is! List) return [];
  final apps = items
      .whereType<Map>()
      .map((item) => AppInstallInfo.fromJson(item.cast<String, dynamic>()))
      .toList();
  if (status == null || status.isEmpty || status == 'all') return apps;
  final normalizedStatus = status.toLowerCase();
  return apps
      .where((app) => app.status.toLowerCase() == normalizedStatus)
      .toList();
}

/// 应用商店与已安装应用操作仓库
/// 对接 GoPanel 真实的 AppInstall 接口
class AppsRepository {
  final ApiClient _apiClient;

  AppsRepository(this._apiClient);

  /// 分页搜索已安装应用
  /// POST /api/apps/installed/list
  Future<List<AppInstallInfo>> getInstalledApps({
    int page = 1,
    int pageSize = 50,
    String? name,
    String? status, // 例如 running, stopped
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      installedAppsListPath,
      data: {
        'page': page,
        'limit': pageSize,
        if (name != null && name.isNotEmpty) 'name': name,
      },
    );

    return parseInstalledApps(response.data, status: status);
  }

  /// 对已安装应用执行生命周期操作 (启停、重启等)
  /// POST /api/apps/installed/op
  /// body 格式: { "installId": uint, "operate": "start|stop|restart..." }
  Future<bool> operateApp({
    required int installId,
    required String operation,
  }) async {
    await _apiClient.post<dynamic>(
      '/api/apps/installed/op',
      data: {'installId': installId, 'operate': operation},
    );
    // 没抛异常就是操作成功
    return true;
  }
}
