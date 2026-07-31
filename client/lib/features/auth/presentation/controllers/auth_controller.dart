import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/storage/storage_service.dart';
import '../../../../shared/models/server_connection.dart';
import '../../data/auth_repository.dart';

/// 注入 AuthRepository 的 Provider
final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(ApiClient());
});

/// AuthController 的状态定义
class AuthState {
  final bool isLoading;
  final String? errorMessage;
  final bool isAuthenticated;

  const AuthState({
    this.isLoading = false,
    this.errorMessage,
    this.isAuthenticated = false,
  });

  AuthState copyWith({
    bool? isLoading,
    String? errorMessage,
    bool? isAuthenticated,
  }) {
    return AuthState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage, // 故意不兜底，传 null 就能清空 error
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
    );
  }
}

/// 注入 AuthController 的 NotifierProvider
final authControllerProvider = NotifierProvider<AuthController, AuthState>(
  AuthController.new,
);

/// 处理整个连接和登录过程的控制器
class AuthController extends Notifier<AuthState> {
  late AuthRepository _authRepository;

  @override
  AuthState build() {
    _authRepository = ref.watch(authRepositoryProvider);
    _checkInitialAuth();
    return const AuthState();
  }

  /// App 启动时检查是否有缓存的 Token 并验证
  Future<void> _checkInitialAuth() async {
    final token = StorageService.activeServerToken;
    final baseUrl = StorageService.activeServerUrl;

    if (token != null && baseUrl != null) {
      state = state.copyWith(isLoading: true);
      try {
        // 请求用户信息以验证 Token 是否过期
        await _authRepository.getUserInfo();
        state = state.copyWith(isLoading: false, isAuthenticated: true);
      } catch (e) {
        // 如果验证失败，清理过期状态
        await logout();
      }
    }
  }

  /// 建立连接并登录
  /// 用户在 ServerLoginScreen 填写的往往是 GoPanel 的面板 URL、账号、密码
  Future<bool> connectAndLogin({
    required String serverUrl,
    required String username,
    required String password,
  }) async {
    state = state.copyWith(isLoading: true, errorMessage: null);

    try {
      // 0. 验证地址和安全入口是否匹配
      await _authRepository.checkServerEntrance(serverUrl);

      // 1. 解析 BaseUrl (只提取 scheme + host + port)
      final uri = Uri.parse(serverUrl);
      final baseUrl =
          '${uri.scheme}://${uri.host}${uri.hasPort ? ':${uri.port}' : ''}';

      // 2. 先临时保存 BaseUrl，拦截器会自动读取并拼接到后续请求
      // 注意：这里保存的是纯粹的 baseUrl，不带入口路径，以便 /api/... 能正确拼接
      await StorageService.setActiveServerUrl(baseUrl);

      // 3. 调用真实的 GoPanel 登录接口
      final token = await _authRepository.login(
        name: username,
        password: password,
      );

      // 4. 成功后保存 Token，拦截器会自动把它塞进 xAuth Header 中
      await StorageService.setActiveServerToken(token);

      // 保存到本地聚合服务器列表中
      await StorageService.saveServerConnection(
        ServerConnection(
          id: DateTime.now().millisecondsSinceEpoch.toString(),
          name: username, // 默认别名为用户名
          url: serverUrl, // 列表里展示完整的带入口 URL
          token: token,
          lastConnectedAt: DateTime.now(),
        ),
      );

      // 5. 更新状态为已登录
      state = state.copyWith(isLoading: false, isAuthenticated: true);
      return true;
    } catch (e) {
      // 失败后清空可能写了一半的配置
      await StorageService.clearActiveServer();

      state = state.copyWith(isLoading: false, errorMessage: '连接或登录失败: $e');
      return false;
    }
  }

  /// 扫码授权登录：二维码内容中直接包含 Token 与 URL
  /// 假设二维码格式为 JSON：{"url": "https://demo.gopanel.run", "token": "xxxx"}
  /// 或者用某种特定协议拼装的字符串
  Future<bool> connectWithQrToken({
    required String serverUrl,
    required String token,
  }) async {
    state = state.copyWith(isLoading: true, errorMessage: null);

    try {
      // 0. 验证地址和安全入口是否匹配
      await _authRepository.checkServerEntrance(serverUrl);

      // 1. 解析 BaseUrl (只提取 scheme + host + port)
      final uri = Uri.parse(serverUrl);
      final baseUrl =
          '${uri.scheme}://${uri.host}${uri.hasPort ? ':${uri.port}' : ''}';

      // 2. 先临时保存 BaseUrl 和 Token，拦截器会自动读取
      await StorageService.setActiveServerUrl(baseUrl);
      await StorageService.setActiveServerToken(token);

      // 3. 直接调用 getUserInfo 验证这个 Token 的有效性
      await _authRepository.getUserInfo();

      // 4. 验证成功，保存到服务器列表并更新状态为已登录
      await StorageService.saveServerConnection(
        ServerConnection(
          id: DateTime.now().millisecondsSinceEpoch.toString(),
          name: 'QR 授权',
          url: serverUrl,
          token: token,
          lastConnectedAt: DateTime.now(),
        ),
      );

      state = state.copyWith(isLoading: false, isAuthenticated: true);
      return true;
    } catch (e) {
      // 验证失败，清理缓存
      await StorageService.clearActiveServer();

      state = state.copyWith(isLoading: false, errorMessage: '扫码授权验证失败: $e');
      return false;
    }
  }

  /// 登出
  Future<void> logout() async {
    await StorageService.clearActiveServer();
    state = const AuthState(isAuthenticated: false);
  }
}
