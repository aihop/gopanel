import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/storage/storage_service.dart';
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

  test('macOS data protection keychain is reserved for release builds', () {
    expect(
      buildMacOsSecureStorageOptions(
        releaseMode: false,
      ).usesDataProtectionKeychain,
      isFalse,
    );
    expect(
      buildMacOsSecureStorageOptions(
        releaseMode: true,
      ).usesDataProtectionKeychain,
      isTrue,
    );
  });

  test('Code instruction drafts are isolated by server and session', () {
    final firstServer = buildCodeInstructionDraftKey(
      'https://one.example.com',
      12,
    );
    final secondServer = buildCodeInstructionDraftKey(
      'https://two.example.com',
      12,
    );

    expect(firstServer, isNot(secondServer));
    expect(firstServer, isNot(buildCodeInstructionDraftKey(null, 12)));
    expect(
      firstServer,
      isNot(buildCodeInstructionDraftKey('https://one.example.com', 13)),
    );
    expect(
      firstServer,
      buildCodeInstructionDraftKey(' HTTPS://ONE.EXAMPLE.COM ', 12),
    );
  });
}
