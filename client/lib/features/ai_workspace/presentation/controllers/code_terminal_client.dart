import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:xterm/xterm.dart';

import '../../../../core/storage/storage_service.dart';

class CodeTerminalClient extends ChangeNotifier {
  CodeTerminalClient({
    required this.terminal,
    required this.sessionId,
    required this.nativeProtocol,
  }) {
    terminal.onOutput = sendInput;
    terminal.onResize = _handleResize;
  }

  final Terminal terminal;
  final int sessionId;
  final bool nativeProtocol;

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _reconnectTimer;
  Timer? _pingTimer;
  bool _connected = false;
  bool _reconnecting = false;
  bool _hasControl = false;
  bool _closing = false;
  bool _disposed = false;
  int _lastSequence = 0;
  int _resyncRequest = 0;
  int _cols = 80;
  int _rows = 24;
  String _pendingResyncId = '';

  bool get connected => _connected;
  bool get reconnecting => _reconnecting;
  bool get hasControl => _hasControl;
  bool get canInput => _connected && _hasControl;

  Uri? _terminalUri() {
    final server = StorageService.activeServerUrl;
    final token = StorageService.activeServerToken;
    if (server == null || server.isEmpty || token == null || token.isEmpty) {
      return null;
    }
    final base = Uri.parse(server);
    return base.replace(
      scheme: base.scheme == 'https' ? 'wss' : 'ws',
      path: '/api/code/terminal',
      queryParameters: {
        'token': token,
        'session_id': sessionId.toString(),
        'cols': _cols.toString(),
        'rows': _rows.toString(),
        if (_lastSequence > 0) 'after_sequence': _lastSequence.toString(),
      },
    );
  }

  Future<void> connect() async {
    if (_disposed || _closing) return;
    final uri = _terminalUri();
    if (uri == null) {
      _closing = true;
      _writeSystem('缺少服务器连接信息，无法打开终端。', error: true);
      _notify();
      return;
    }

    final previousChannel = _channel;
    _channel = null;
    await _subscription?.cancel();
    _subscription = null;
    previousChannel?.sink.close();

    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    _subscription = channel.stream.listen(
      (event) => _handleMessage(channel, event),
      onError: (Object error) {
        if (_channel != channel) return;
        _writeSystem('终端连接失败：$error', error: true);
        _handleDisconnect(channel);
      },
      onDone: () => _handleDisconnect(channel),
      cancelOnError: true,
    );
    try {
      await channel.ready;
      if (_disposed || _channel != channel) return;
      _connected = true;
      _reconnecting = false;
      if (!nativeProtocol) _hasControl = true;
      _startPing();
      _sendResize();
      _notify();
    } catch (error) {
      if (_channel != channel) return;
      _writeSystem('终端握手失败：$error', error: true);
      _handleDisconnect(channel);
    }
  }

  void _handleMessage(WebSocketChannel channel, dynamic event) {
    if (_channel != channel || _disposed) return;
    if (!_connected) {
      _connected = true;
      _reconnecting = false;
      _notify();
    }
    final text = event is List<int> ? utf8.decode(event) : event.toString();
    try {
      final message = jsonDecode(text) as Map<String, dynamic>;
      switch ((message['type'] ?? '').toString()) {
        case 'baseline':
          _handleBaseline(message);
        case 'output':
          _handleOutput(message);
        case 'control':
          _updateControl(message['hasControl'] == true);
          final reason = (message['controlReason'] ?? '').toString();
          if (reason.isNotEmpty) _writeSystem(reason);
        case 'resync_required':
          _requestResync();
        case 'closed':
          _closing = true;
          _connected = false;
          _updateControl(false);
          _writeSystem('会话已结束。');
          _notify();
        case 'error':
          _closing = true;
          _connected = false;
          _updateControl(false);
          _writeSystem((message['data'] ?? '终端发生错误。').toString(), error: true);
          _notify();
        case 'cmd':
          terminal.write((message['data'] ?? '').toString());
          if (!nativeProtocol) _updateControl(true);
        case 'meta':
        case 'pong':
          break;
        default:
          terminal.write(text);
      }
    } catch (_) {
      terminal.write(text);
    }
  }

  void _handleBaseline(Map<String, dynamic> message) {
    final requestId = (message['requestId'] ?? '').toString();
    if (_pendingResyncId.isNotEmpty && requestId != _pendingResyncId) return;
    final chunkIndex = (message['chunkIndex'] as num?)?.toInt() ?? 0;
    final chunkCount = (message['chunkCount'] as num?)?.toInt() ?? 1;
    if (message['truncated'] == true && chunkIndex == 0) {
      terminal.buffer.clear();
      terminal.buffer.setCursor(0, 0);
      _writeSystem('较早的终端输出已截断。');
    }
    terminal.write((message['data'] ?? '').toString());
    if (chunkIndex != chunkCount - 1) return;
    _lastSequence = (message['sequence'] as num?)?.toInt() ?? 0;
    _pendingResyncId = '';
    _updateControl(message['hasControl'] == true);
    _send('ack', _lastSequence.toString());
  }

  void _handleOutput(Map<String, dynamic> message) {
    if (_pendingResyncId.isNotEmpty) return;
    final sequence = (message['sequence'] as num?)?.toInt() ?? 0;
    if (_lastSequence > 0 && sequence != _lastSequence + 1) {
      _requestResync();
      return;
    }
    terminal.write((message['data'] ?? '').toString());
    _lastSequence = sequence;
    _send('ack', sequence.toString());
  }

  void _handleDisconnect(WebSocketChannel channel) {
    if (_channel != channel) return;
    _pingTimer?.cancel();
    _channel = null;
    _subscription = null;
    _connected = false;
    _hasControl = false;
    _pendingResyncId = '';
    _notify();
    if (_closing || _disposed || _reconnectTimer != null) return;
    _reconnecting = true;
    _notify();
    _reconnectTimer = Timer(const Duration(milliseconds: 1500), () {
      _reconnectTimer = null;
      connect();
    });
  }

  void _startPing() {
    _pingTimer?.cancel();
    _pingTimer = Timer.periodic(
      const Duration(seconds: 15),
      (_) => _send('ping', ''),
    );
  }

  void _requestResync() {
    if (!_connected || _pendingResyncId.isNotEmpty) return;
    _pendingResyncId =
        '${DateTime.now().millisecondsSinceEpoch}-${++_resyncRequest}';
    _send(
      'resync',
      jsonEncode({'sequence': _lastSequence, 'requestId': _pendingResyncId}),
    );
  }

  void _send(String type, String data) {
    if (!_connected) return;
    _channel?.sink.add(jsonEncode({'type': type, 'data': data}));
  }

  void sendInput(String data) {
    if (!canInput || data.isEmpty) return;
    _send('cmd', data);
  }

  void takeControl() => _send('take_control', '');

  void releaseControl() => _send('release_control', '');

  void _handleResize(int cols, int rows, int pixelWidth, int pixelHeight) {
    _cols = cols;
    _rows = rows;
    _sendResize();
  }

  void _sendResize() {
    if (!canInput) return;
    _send('resize', jsonEncode({'cols': _cols, 'rows': _rows}));
  }

  void _updateControl(bool value) {
    if (_hasControl == value) return;
    _hasControl = value;
    if (value) _sendResize();
    _notify();
  }

  void _writeSystem(String message, {bool error = false}) {
    final color = error ? '31' : '33';
    terminal.write('\r\n\x1b[${color}m[GoPanel] $message\x1b[0m\r\n');
  }

  void _notify() {
    if (!_disposed) notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _closing = true;
    _reconnectTimer?.cancel();
    _pingTimer?.cancel();
    _subscription?.cancel();
    _channel?.sink.close();
    terminal.onOutput = null;
    terminal.onResize = null;
    super.dispose();
  }
}
