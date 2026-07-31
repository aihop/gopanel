import '../../../core/network/api_client.dart';
import '../models/container_info.dart';

/// 容器操作仓库
/// 对接 GoPanel 真实的 Docker 接口
class ContainerRepository {
  final ApiClient _apiClient;

  ContainerRepository(this._apiClient);

  /// 搜索并分页获取容器列表
  /// POST /api/container/search (此处用 /api/containers 根据常规推断，若后端不是这个路径需调整)
  /// GoPanel 的路由定义为 /container/search
  Future<List<ContainerInfo>> getContainerList({
    int page = 1,
    int pageSize = 50,
    String? name,
    String? state = "all",
    String? order = 'null',
    String? orderBy = 'created_at',
  }) async {
    // 参照 GoPanel，通常搜索接口支持分页和筛选
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/container/search',
      data: {
        'page': page,
        'pageSize': pageSize,
        if (name != null && name.isNotEmpty) 'name': name,
        if (state != null && state.isNotEmpty) 'state': state,
        if (order != null && order.isNotEmpty) 'order': order,
        if (orderBy != null && orderBy.isNotEmpty) 'orderBy': orderBy,
      },
    );

    final data = response.data;
    if (data != null && data['items'] is List) {
      final items = data['items'] as List;
      return items
          .map((e) => ContainerInfo.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return [];
  }

  /// 对容器执行操作（支持批量）
  /// POST /api/container/operate
  Future<bool> operateContainers({
    required List<String> names, // GoPanel 传的是 names (字符串数组)
    required String operation, // start, stop, restart 等
  }) async {
    await _apiClient.post<dynamic>(
      '/api/container/operate',
      data: {'names': names, 'operation': operation},
    );
    // 未抛异常则视为成功
    return true;
  }
}
