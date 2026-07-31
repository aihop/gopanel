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
}
