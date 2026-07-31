import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel connectCodeTerminalSocket(
  Uri uri, {
  required Map<String, dynamic> headers,
}) {
  return WebSocketChannel.connect(uri);
}
