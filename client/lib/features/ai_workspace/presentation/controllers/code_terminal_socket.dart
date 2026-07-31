import 'package:web_socket_channel/web_socket_channel.dart';

import 'code_terminal_socket_fallback.dart'
    if (dart.library.io) 'code_terminal_socket_io.dart'
    as platform;

Map<String, dynamic> buildCodeTerminalHeaders({
  required String token,
  String? cookie,
}) {
  final normalizedCookie = cookie?.trim() ?? '';
  return {
    'X-Auth': token,
    if (normalizedCookie.isNotEmpty) 'Cookie': normalizedCookie,
  };
}

WebSocketChannel connectCodeTerminalSocket(
  Uri uri, {
  required String token,
  String? cookie,
}) {
  return platform.connectCodeTerminalSocket(
    uri,
    headers: buildCodeTerminalHeaders(token: token, cookie: cookie),
  );
}
