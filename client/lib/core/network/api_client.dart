import 'package:dio/dio.dart';
import 'dart:async';
import 'dart:convert';

import 'api_exception.dart';
import 'api_response.dart';
import 'interceptors/auth_interceptor.dart';

/// 全局统一的网络请求客户端
/// 遵循 PROMPT.md 规范：不让网络请求直接散落在页面或子 Widget
class ApiClient {
  // 单例模式，保证全局网络配置一致
  static final ApiClient _instance = ApiClient._internal();
  factory ApiClient() => _instance;

  late final Dio _dio;

  ApiClient._internal() {
    _dio = Dio(
      BaseOptions(
        // 默认的超时时间配置，依据 GoPanel 可能的连接慢情况
        connectTimeout: const Duration(seconds: 15),
        receiveTimeout: const Duration(seconds: 15),
        sendTimeout: const Duration(seconds: 15),
        responseType: ResponseType.json,
        contentType: 'application/json',
      ),
    );

    // 添加处理 URL 和 Token 的认证拦截器
    _dio.interceptors.add(AuthInterceptor());
  }

  /// 封装标准的 GET 请求
  Future<ApiResponse<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    try {
      final response = await _dio.get(
        path,
        queryParameters: queryParameters,
        options: options,
      );
      return _handleResponse<T>(response);
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    } catch (e) {
      throw ApiException(msg: e.toString());
    }
  }

  /// 封装标准的 POST 请求
  Future<ApiResponse<T>> post<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    try {
      final response = await _dio.post(
        path,
        data: data,
        queryParameters: queryParameters,
        options: options,
      );
      return _handleResponse<T>(response);
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    } catch (e) {
      throw ApiException(msg: e.toString());
    }
  }

  /// 封装标准的 PUT 请求
  Future<ApiResponse<T>> put<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    try {
      final response = await _dio.put(
        path,
        data: data,
        queryParameters: queryParameters,
        options: options,
      );
      return _handleResponse<T>(response);
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    } catch (e) {
      throw ApiException(msg: e.toString());
    }
  }

  /// 封装标准的 DELETE 请求
  Future<ApiResponse<T>> delete<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    try {
      final response = await _dio.delete(
        path,
        data: data,
        queryParameters: queryParameters,
        options: options,
      );
      return _handleResponse<T>(response);
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    } catch (e) {
      throw ApiException(msg: e.toString());
    }
  }

  /// GET 获取 SSE 的 data 行（text/event-stream）
  /// 仅用于后端返回非标准 JSON 的 SSE 日志流（如 /api/pipeline/logs）
  Future<List<String>> getSseDataLines(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
    Duration timeout = const Duration(seconds: 20),
    Duration maxWait = const Duration(seconds: 3),
    int maxLines = 2000,
  }) async {
    final res = await _dio.get<ResponseBody>(
      path,
      queryParameters: queryParameters,
      options: (options ?? Options()).copyWith(
        responseType: ResponseType.stream,
      ),
    );

    final body = res.data;
    if (body == null) {
      throw ApiException(msg: 'SSE 响应为空');
    }

    final lines = <String>[];
    final completer = Completer<List<String>>();
    StreamSubscription<String>? sub;
    Timer? timer;

    Future<void> finish() async {
      if (!completer.isCompleted) {
        completer.complete(lines);
      }
      timer?.cancel();
      timer = null;
      await sub?.cancel();
      sub = null;
    }

    timer = Timer(maxWait, () {
      finish();
    });

    sub = body.stream
        .cast<List<int>>()
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen(
          (line) {
            if (!line.startsWith('data:')) return;
            final data = line.substring(5).trimLeft();
            if (data == 'EOF') {
              finish();
              return;
            }
            if (data.isEmpty) return;
            lines.add(data);
            if (lines.length >= maxLines) {
              finish();
            }
          },
          onError: (e) {
            if (!completer.isCompleted) {
              completer.completeError(ApiException(msg: e.toString()));
            }
            timer?.cancel();
          },
          onDone: () {
            finish();
          },
          cancelOnError: true,
        );

    return completer.future.timeout(
      timeout,
      onTimeout: () async {
        await finish();
        return lines;
      },
    );
  }

  /// 处理 Dio 的 Response 对象，转换为项目的统一 ApiResponse
  ApiResponse<T> _handleResponse<T>(Response response) {
    final responseData = response.data;

    // 如果服务端返回格式确实是 GoPanel 标准 JSON: {code: xxx, msg: xxx, data: xxx}
    if (responseData is Map<String, dynamic>) {
      final apiResponse = ApiResponse<T>.fromJson(responseData);
      // 可在此判断业务级错误，如果接口虽然 http=200 但业务报错
      if (!apiResponse.isSuccess) {
        throw ApiException(code: apiResponse.code, msg: apiResponse.msg);
      }
      return apiResponse;
    }

    // 如果接口直接返回非包装数据或列表
    throw ApiException(msg: '未识别的服务器响应格式');
  }
}
