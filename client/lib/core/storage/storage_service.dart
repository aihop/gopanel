import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import '../../shared/models/server_connection.dart';

/// 统一存储服务，封装对 SharedPreferences 的调用
/// 遵循 PROMPT.md 规范：不让业务代码直接依赖存储实现
class StorageService {
  static late SharedPreferences _prefs;

  /// 初始化存储服务，在 main() 中调用
  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
  }

  // ==========================================
  // 服务器连接列表 (聚合页需要展示的全部服务器)
  // ==========================================
  static const String _keyServerList = 'server_list';

  /// 获取保存的所有服务器列表
  static List<ServerConnection> getServerList() {
    final String? jsonStr = _prefs.getString(_keyServerList);
    if (jsonStr == null || jsonStr.isEmpty) return [];

    try {
      final List<dynamic> jsonList = jsonDecode(jsonStr);
      return jsonList.map((e) => ServerConnection.fromJson(e)).toList();
    } catch (e) {
      return [];
    }
  }

  /// 新增或更新一个服务器到列表中
  static Future<bool> saveServerConnection(ServerConnection server) async {
    final list = getServerList();
    final index = list.indexWhere((e) => e.url == server.url);
    if (index >= 0) {
      // 存在相同 URL 的服务器则覆盖更新
      list[index] = server;
    } else {
      // 否则插入新服务器
      list.add(server);
    }
    final jsonStr = jsonEncode(list.map((e) => e.toJson()).toList());
    return await _prefs.setString(_keyServerList, jsonStr);
  }

  /// 从列表中移除某个服务器
  static Future<bool> removeServerConnection(String url) async {
    final list = getServerList();
    list.removeWhere((e) => e.url == url);
    final jsonStr = jsonEncode(list.map((e) => e.toJson()).toList());
    return await _prefs.setString(_keyServerList, jsonStr);
  }

  // ==========================================
  // 当前激活的服务器连接信息 (BaseURL & Token & Cookie)
  // ==========================================
  static const String _keyActiveServerUrl = 'active_server_url';
  static const String _keyActiveServerToken = 'active_server_token';
  static const String _keyActiveServerCookie = 'active_server_cookie';

  /// 获取当前激活的服务器面板地址
  static String? get activeServerUrl => _prefs.getString(_keyActiveServerUrl);

  /// 设置当前激活的服务器面板地址
  static Future<bool> setActiveServerUrl(String url) async {
    return await _prefs.setString(_keyActiveServerUrl, url);
  }

  /// 获取当前激活的服务器登录 Token
  static String? get activeServerToken => _prefs.getString(_keyActiveServerToken);

  /// 设置当前激活的服务器登录 Token
  static Future<bool> setActiveServerToken(String token) async {
    return await _prefs.setString(_keyActiveServerToken, token);
  }

  /// 获取当前激活的服务器 Cookie
  static String? get activeServerCookie => _prefs.getString(_keyActiveServerCookie);

  /// 设置当前激活的服务器 Cookie
  static Future<bool> setActiveServerCookie(String cookie) async {
    return await _prefs.setString(_keyActiveServerCookie, cookie);
  }

  /// 登出当前激活的服务器（退回到聚合页时调用）
  static Future<void> clearActiveServer() async {
    await _prefs.remove(_keyActiveServerUrl);
    await _prefs.remove(_keyActiveServerToken);
    await _prefs.remove(_keyActiveServerCookie);
  }
}
