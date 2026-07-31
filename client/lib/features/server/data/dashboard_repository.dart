import '../../../core/network/api_client.dart';
import '../models/monitor_info.dart';
import '../models/system_info.dart';

/// 仪表盘与系统资源仓库
/// 负责处理 GoPanel 获取主机信息、CPU、内存等资源状态的请求
class DashboardRepository {
  final ApiClient _apiClient;

  DashboardRepository(this._apiClient);

  /// 获取基础操作系统信息
  /// GET /api/dashboard/base/os (注意 GoPanel 路由里可能是带 v1 前缀的)
  Future<OsInfo> getOsInfo() async {
    final response = await _apiClient.get<Map<String, dynamic>>(
      '/api/dashboard/base/os',
    );
    return OsInfo.fromJson(response.data ?? {});
  }

  /// 获取实时资源数据（CPU使用率、内存、磁盘进度）
  /// POST /api/dashboard/current
  Future<SystemCurrentInfo> getCurrentInfo() async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/dashboard/current',
    );
    return SystemCurrentInfo.fromJson(response.data ?? {});
  }

  /// 获取 IO / 网络累计数据（用于计算速率与绘制趋势）
  /// POST /api/dashboard/current {scope: ioNet}
  Future<IoNetInfo> getIoNetInfo({
    String ioOption = 'all',
    String netOption = 'all',
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/dashboard/current',
      data: {'scope': 'ioNet', 'ioOption': ioOption, 'netOption': netOption},
    );
    return IoNetInfo.fromJson(response.data ?? {});
  }
}
