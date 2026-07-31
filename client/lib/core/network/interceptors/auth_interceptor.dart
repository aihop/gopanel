import 'package:dio/dio.dart';
import '../../storage/storage_service.dart';

/// 全局认证与动态连接拦截器
/// 遵循 PROMPT.md：
/// 1. 不把 baseUrl 写死，移动端支持连接多台远程面板，URL 需要动态读取
/// 2. 不在页面里散落 Token 拼接逻辑，统一在拦截器处理鉴权头
class AuthInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    // 动态获取当前激活服务器的信息，而不是全局硬编码写死
    final token = StorageService.activeServerToken;
    final baseUrl = StorageService.activeServerUrl;

    // 1. 如果请求的 URL 尚未补全域名（相对路径），且本地已有选中的服务端 URL，则覆盖 baseUrl
    if (!options.path.startsWith('http') &&
        baseUrl != null &&
        baseUrl.isNotEmpty) {
      options.baseUrl = baseUrl;
    }

    // 2. 如果存在激活的 Token，统一追加认证请求头
    // 根据 GoPanel 后端 middleware/jwt.go 的规范，Token 通常放在 xAuth 头中
    if (token != null && token.isNotEmpty) {
      options.headers['X-Auth'] = token;
    }

    // 3. 如果存在激活的 Cookie (如 Entrance, session)，自动携带
    final cookie = StorageService.activeServerCookie;
    if (cookie != null && cookie.isNotEmpty) {
      options.headers['Cookie'] = cookie;
    }

    super.onRequest(options, handler);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    // 拦截并保存后端下发的 Set-Cookie (如 Entrance, gin_session 等)
    final setCookies = response.headers['set-cookie'];
    if (setCookies != null && setCookies.isNotEmpty) {
      // 简单合并多个 Cookie
      final cookieStr = setCookies.map((str) => str.split(';')[0]).join('; ');
      StorageService.setActiveServerCookie(cookieStr);
    }

    // 在这里可以统一处理特殊的响应业务状态码（比如 code != 200 抛异常）
    // 当前保留基础拦截流转，具体根据 ApiResponse 的 `code` 在 ApiClient 中处理
    super.onResponse(response, handler);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    // 统一处理 401，通常表示 Token 失效，可能需要跳转登录或清理本地连接状态
    if (err.response?.statusCode == 401) {
      // 可以在此处发送全局事件触发强制退出 / 返回服务器列表页
      // e.g. AppEventBus.fire(EventUserUnauthorized());
    }

    super.onError(err, handler);
  }
}
