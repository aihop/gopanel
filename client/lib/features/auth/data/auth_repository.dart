import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';

/// 认证仓库
/// 负责处理所有与登录、鉴权相关的网络请求，对接 GoPanel 后端真实接口
class AuthRepository {
  final ApiClient _apiClient;

  AuthRepository(this._apiClient);

  /// 验证服务器面板地址和安全入口是否正确
  Future<void> checkServerEntrance(String fullUrl) async {
    final dio = Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 10),
      ),
    );

    try {
      // 访问用户输入的地址（可能带入口，也可能只输入了域名）
      final response = await dio.get(fullUrl);
      final data = response.data;
      if (data is String &&
          data.contains('安全入口登录') &&
          data.contains('暂时无法访问')) {
        throw Exception('当前服务器已开启安全入口，请输入包含入口路径的完整面板地址');
      }
    } on DioException catch (e) {
      if (e.response?.data is String) {
        final data = e.response!.data as String;
        if (data.contains('安全入口登录') && data.contains('暂时无法访问')) {
          throw Exception('当前服务器已开启安全入口，请输入包含入口路径的完整面板地址');
        }
      }
      // 如果不是入口错误，我们假定它可能是因为跨域或其他原因被拦截，允许继续，或者向外抛出连接异常
      if (e.type == DioExceptionType.connectionTimeout ||
          e.type == DioExceptionType.connectionError) {
        throw Exception('无法连接到服务器，请检查网络或地址拼写');
      }
    } catch (e) {
      if (e.toString().contains('当前服务器已开启安全入口')) {
        rethrow;
      }
      throw Exception('验证服务器地址失败: $e');
    }
  }

  /// 登录接口 (对应 GoPanel: /api/auth/signin)
  /// GoPanel 的请求体支持 email 或 mobile+isdCode，以及 password
  /// 响应体中的 data 通常包含 { "xAuth": "xxx" }
  Future<String> login({
    required String name, // App 端统一叫 name，这里我们当作 email 处理，或者你需要手机号登录可自行调整
    required String password,
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/auth/signin',
      data: {
        'email': name, // 后端实际预期的是 email 字段
        'password': password,
      },
    );

    // 提取 Token
    final data = response.data;
    if (data != null && data.containsKey('xAuth')) {
      return data['xAuth'] as String;
    }

    throw Exception('登录成功，但未获取到 Token');
  }

  /// 获取当前登录用户信息 (对应 GoPanel: POST /api/user/info)
  /// 可以用这个接口来“验证 Token 是否有效”
  Future<Map<String, dynamic>> getUserInfo() async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/user/info',
    );
    return response.data ?? {};
  }
}
