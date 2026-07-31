import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_terminal_socket.dart';

void main() {
  test('terminal handshake reuses JWT and security entrance cookie', () {
    final headers = buildCodeTerminalHeaders(
      token: 'jwt-token',
      cookie: 'Entrance=secure-entry; session=active',
    );

    expect(headers['X-Auth'], 'jwt-token');
    expect(headers['Cookie'], 'Entrance=secure-entry; session=active');
  });

  test('terminal handshake omits an empty cookie', () {
    final headers = buildCodeTerminalHeaders(token: 'jwt-token', cookie: '  ');

    expect(headers, {'X-Auth': 'jwt-token'});
  });

  test('builds AI session terminal websocket URL', () {
    final uri = buildCodeTerminalUri(
      server: 'https://panel.example.com/entrance',
      token: 'jwt-token',
      sessionId: 12,
      mode: CodeTerminalMode.aiSession,
      cols: 100,
      rows: 30,
      lastSequence: 8,
    );

    expect(uri.scheme, 'wss');
    expect(uri.path, '/api/code/terminal');
    expect(uri.queryParameters['session_id'], '12');
    expect(uri.queryParameters['after_sequence'], '8');
  });

  test('builds project terminal websocket URL', () {
    final uri = buildCodeTerminalUri(
      server: 'http://panel.example.com',
      token: 'jwt-token',
      sessionId: 27,
      mode: CodeTerminalMode.projectTerminal,
      cols: 80,
      rows: 24,
    );

    expect(uri.scheme, 'ws');
    expect(uri.path, '/api/code/project-terminal/27/ws');
    expect(uri.queryParameters, {'token': 'jwt-token'});
  });
}
