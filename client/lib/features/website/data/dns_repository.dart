import '../../../core/network/api_client.dart';

/// DNS 账户仓库
/// 对接 GoPanel 真实的 DNS 接口，用于证书申请的授权
class DnsRepository {
  final ApiClient _apiClient;

  DnsRepository(this._apiClient);

  /// 获取 DNS 账户列表
  /// POST /api/website/dns/search
  Future<List<Map<String, dynamic>>> getDnsAccounts() async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/website/dns/search',
      data: {
        'page': 1,
        'pageSize': 100,
        // 这里可以根据后端的 search 参数格式补充
      },
    );

    // 假设返回的数据结构是 { code: 0, msg: "success", data: { items: [...] } }
    final data = response.data;
    if (data != null && data.containsKey('items')) {
      final items = data['items'] as List;
      return items.cast<Map<String, dynamic>>();
    }
    return [];
  }

  /// 添加 DNS 账户
  /// POST /api/website/dns
  Future<bool> addDnsAccount({
    required String name,
    required String type,
    required String authorizationStr, // JSON 字符串格式的鉴权信息
  }) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/website/dns',
      data: {'name': name, 'type': type, 'authorization': authorizationStr},
    );
    // 没有抛出异常说明添加成功
    return true;
  }

  /// 删除 DNS 账户
  /// POST /api/website/dns/del
  Future<bool> deleteDnsAccount(int id) async {
    await _apiClient.post<Map<String, dynamic>>(
      '/api/website/dns/del',
      data: {'id': id},
    );
    return true;
  }
}
