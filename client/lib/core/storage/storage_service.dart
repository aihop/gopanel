import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../shared/models/server_connection.dart';

class StorageService {
  static late SharedPreferences _prefs;
  static const FlutterSecureStorage _secureStorage = FlutterSecureStorage();
  static final Map<String, String> _serverTokens = {};
  static String? _activeServerToken;
  static String? _activeServerCookie;

  static const String _keyServerList = 'server_list';
  static const String _keyActiveServerUrl = 'active_server_url';
  static const String _keyActiveServerToken = 'active_server_token';
  static const String _keyActiveServerCookie = 'active_server_cookie';
  static const String _secureActiveToken = 'secure_active_server_token';
  static const String _secureActiveCookie = 'secure_active_server_cookie';

  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
    await _migrateActiveCredentials();
    _activeServerToken = await _secureStorage.read(key: _secureActiveToken);
    _activeServerCookie = await _secureStorage.read(key: _secureActiveCookie);
    await _loadServerTokens();
  }

  static List<ServerConnection> getServerList() {
    final entries = _readServerEntries();
    return entries.map((entry) {
      final server = ServerConnection.fromJson(entry);
      return ServerConnection(
        id: server.id,
        name: server.name,
        url: server.url,
        token: _serverTokens[server.id] ?? '',
        lastConnectedAt: server.lastConnectedAt,
      );
    }).toList();
  }

  static Future<bool> saveServerConnection(ServerConnection server) async {
    final list = getServerList();
    final index = list.indexWhere((item) => item.url == server.url);
    final saved = index >= 0
        ? ServerConnection(
            id: list[index].id,
            name: server.name,
            url: server.url,
            token: server.token,
            lastConnectedAt: server.lastConnectedAt,
          )
        : server;
    await _secureStorage.write(
      key: _serverTokenKey(saved.id),
      value: saved.token,
    );
    _serverTokens[saved.id] = saved.token;
    if (index >= 0) {
      list[index] = saved;
    } else {
      list.add(saved);
    }
    return _writeServerList(list);
  }

  static Future<bool> removeServerConnection(String url) async {
    final list = getServerList();
    final removed = list.where((item) => item.url == url).toList();
    list.removeWhere((item) => item.url == url);
    for (final server in removed) {
      await _secureStorage.delete(key: _serverTokenKey(server.id));
      _serverTokens.remove(server.id);
    }
    return _writeServerList(list);
  }

  static String? get activeServerUrl => _prefs.getString(_keyActiveServerUrl);

  static Future<bool> setActiveServerUrl(String url) {
    return _prefs.setString(_keyActiveServerUrl, url);
  }

  static String? get activeServerToken => _activeServerToken;

  static Future<bool> setActiveServerToken(String token) async {
    await _secureStorage.write(key: _secureActiveToken, value: token);
    _activeServerToken = token;
    return true;
  }

  static String? get activeServerCookie => _activeServerCookie;

  static Future<bool> setActiveServerCookie(String cookie) async {
    await _secureStorage.write(key: _secureActiveCookie, value: cookie);
    _activeServerCookie = cookie;
    return true;
  }

  static Future<void> clearActiveServer() async {
    await _prefs.remove(_keyActiveServerUrl);
    await _secureStorage.delete(key: _secureActiveToken);
    await _secureStorage.delete(key: _secureActiveCookie);
    _activeServerToken = null;
    _activeServerCookie = null;
  }

  static Future<void> _migrateActiveCredentials() async {
    await _migratePreference(_keyActiveServerToken, _secureActiveToken);
    await _migratePreference(_keyActiveServerCookie, _secureActiveCookie);
  }

  static Future<void> _migratePreference(
    String preferenceKey,
    String secureKey,
  ) async {
    final legacyValue = _prefs.getString(preferenceKey);
    if (legacyValue == null || legacyValue.isEmpty) return;
    final secureValue = await _secureStorage.read(key: secureKey);
    if (secureValue == null || secureValue.isEmpty) {
      await _secureStorage.write(key: secureKey, value: legacyValue);
    }
    await _prefs.remove(preferenceKey);
  }

  static Future<void> _loadServerTokens() async {
    final entries = _readServerEntries();
    var needsRewrite = false;
    for (final entry in entries) {
      final id = entry['id'] as String;
      final legacyToken = entry['token'] as String?;
      var token = await _secureStorage.read(key: _serverTokenKey(id));
      if ((token == null || token.isEmpty) &&
          legacyToken != null &&
          legacyToken.isNotEmpty) {
        await _secureStorage.write(
          key: _serverTokenKey(id),
          value: legacyToken,
        );
        token = legacyToken;
      }
      if (token != null && token.isNotEmpty) {
        _serverTokens[id] = token;
      }
      if (entry.containsKey('token')) {
        entry.remove('token');
        needsRewrite = true;
      }
    }
    if (needsRewrite) {
      await _prefs.setString(_keyServerList, jsonEncode(entries));
    }
  }

  static List<Map<String, dynamic>> _readServerEntries() {
    final jsonString = _prefs.getString(_keyServerList);
    if (jsonString == null || jsonString.isEmpty) return [];
    try {
      final values = jsonDecode(jsonString) as List<dynamic>;
      return values
          .whereType<Map<String, dynamic>>()
          .map((value) => Map<String, dynamic>.from(value))
          .toList();
    } catch (_) {
      return [];
    }
  }

  static Future<bool> _writeServerList(List<ServerConnection> servers) {
    return _prefs.setString(
      _keyServerList,
      jsonEncode(servers.map((server) => server.toJson()).toList()),
    );
  }

  static String _serverTokenKey(String id) => 'secure_server_token_$id';
}
