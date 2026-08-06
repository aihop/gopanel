import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/shared/models/server_connection.dart';

void main() {
  test('server list serialization excludes credentials', () {
    final connection = ServerConnection(
      id: 'server-1',
      name: 'demo',
      url: 'https://example.com',
      token: 'secret-token',
      lastConnectedAt: DateTime.utc(2026),
    );

    expect(connection.toJson().containsKey('token'), isFalse);
  });

  test('legacy server list entries still load for credential migration', () {
    final connection = ServerConnection.fromJson({
      'id': 'server-1',
      'name': 'demo',
      'url': 'https://example.com',
      'token': 'legacy-token',
      'lastConnectedAt': '2026-01-01T00:00:00.000Z',
    });

    expect(connection.token, 'legacy-token');
  });
}
