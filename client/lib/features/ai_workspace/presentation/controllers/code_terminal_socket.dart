import 'package:web_socket_channel/web_socket_channel.dart';

import 'code_terminal_socket_fallback.dart'
    if (dart.library.io) 'code_terminal_socket_io.dart'
    as platform;

enum CodeTerminalMode { aiSession, projectTerminal }

Uri buildCodeTerminalUri({
  required String server,
  required String token,
  required int sessionId,
  required CodeTerminalMode mode,
  required int cols,
  required int rows,
  int lastSequence = 0,
}) {
  final base = Uri.parse(server);
  final projectTerminal = mode == CodeTerminalMode.projectTerminal;
  return base.replace(
    scheme: base.scheme == 'https' ? 'wss' : 'ws',
    path: projectTerminal
        ? '/api/code/project-terminal/$sessionId/ws'
        : '/api/code/terminal',
    queryParameters: {
      'token': token,
      if (!projectTerminal) 'session_id': sessionId.toString(),
      if (!projectTerminal) 'cols': cols.toString(),
      if (!projectTerminal) 'rows': rows.toString(),
      if (!projectTerminal && lastSequence > 0)
        'after_sequence': lastSequence.toString(),
    },
  );
}

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
