import '../../../core/network/api_client.dart';
import '../models/app_install_info.dart';

/// 应用商店与已安装应用操作仓库
/// 对接 GoPanel 真实的 AppInstall 接口
class AppsRepository {
  final ApiClient _apiClient;

  AppsRepository(this._apiClient);

  /// 分页搜索已安装应用
  /// POST /api/apps/installed/search
  Future<List<AppInstallInfo>> getInstalledApps({
    int page = 1,
    int pageSize = 50,
    String? name,
    String? status, // 例如 running, stopped
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/apps/installed/search',
      data: {
        'page': page,
        'pageSize': pageSize,
        if (name != null && name.isNotEmpty) 'name': name,
        if (status != null && status.isNotEmpty && status != 'all')
          'status': status,
      },
    );

    final data = response.data;
    if (data != null && data['items'] is List) {
      final items = data['items'] as List;
      return items
          .map((e) => AppInstallInfo.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return [];
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
