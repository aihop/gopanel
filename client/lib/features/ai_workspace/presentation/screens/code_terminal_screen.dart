import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../../core/storage/storage_service.dart';
import '../../models/ai_dev_session.dart';
import '../widgets/code_terminal_controls.dart';
import 'ai_chat_screen.dart';
import 'code_workspace_files_screen.dart';

class CodeTerminalScreen extends StatefulWidget {
  const CodeTerminalScreen({
    super.key,
    required this.session,
    required this.projectName,
    required this.nativeProtocol,
  });

  final AiDevSession session;
  final String projectName;
  final bool nativeProtocol;

  @override
  State<CodeTerminalScreen> createState() => _CodeTerminalScreenState();
}

class _CodeTerminalScreenState extends State<CodeTerminalScreen> {
  static const _maxOutputLength = 120000;

  final _commandController = TextEditingController();
  final _scrollController = ScrollController();
  final _commandFocusNode = FocusNode();
  final _output = StringBuffer();
  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _reconnectTimer;
  Timer? _pingTimer;
  bool _connected = false;
  bool _reconnecting = false;
  bool _hasControl = false;
  bool _closing = false;
  int _lastSequence = 0;
  int _resyncRequest = 0;
  String _pendingResyncId = '';

  @override
  void initState() {
    super.initState();
    _connect();
  }

  @override
  void dispose() {
    _closing = true;
    _reconnectTimer?.cancel();
    _pingTimer?.cancel();
    _subscription?.cancel();
    _channel?.sink.close();
    _commandController.dispose();
    _scrollController.dispose();
    _commandFocusNode.dispose();
    super.dispose();
  }

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
        'session_id': widget.session.id.toString(),
        'cols': '80',
        'rows': '24',
        if (_lastSequence > 0) 'after_sequence': _lastSequence.toString(),
      },
    );
  }

  Future<void> _connect() async {
    final uri = _terminalUri();
    if (uri == null) {
      _appendOutput('[GoPanel] 缺少服务器连接信息，无法打开终端。\n');
      return;
    }
    await _subscription?.cancel();
    _channel?.sink.close();
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    _subscription = channel.stream.listen(
      _handleMessage,
      onError: (Object error) {
        _appendOutput('[GoPanel] 终端连接失败：$error\n');
        _handleDisconnect();
      },
      onDone: _handleDisconnect,
      cancelOnError: true,
    );
    try {
      await channel.ready;
      if (!mounted || _channel != channel) return;
      setState(() {
        _connected = true;
        _reconnecting = false;
        if (!widget.nativeProtocol) _hasControl = true;
      });
      _startPing();
    } catch (error) {
      if (_channel != channel) return;
      _appendOutput('[GoPanel] 终端握手失败：$error\n');
      _handleDisconnect();
    }
  }

  void _handleMessage(dynamic event) {
    if (mounted && !_connected) {
      setState(() {
        _connected = true;
        _reconnecting = false;
      });
    }
    final text = event is List<int> ? utf8.decode(event) : event.toString();
    try {
      final message = jsonDecode(text) as Map<String, dynamic>;
      final type = (message['type'] ?? '').toString();
      switch (type) {
        case 'baseline':
          _handleBaseline(message);
        case 'output':
          _handleOutput(message);
        case 'control':
          _updateControl(message['hasControl'] == true);
          final reason = (message['controlReason'] ?? '').toString();
          if (reason.isNotEmpty) _appendOutput('\n[GoPanel] $reason\n');
        case 'resync_required':
          _requestResync();
        case 'closed':
          _closing = true;
          _updateControl(false);
          _appendOutput('\n[GoPanel] 会话已结束。\n');
        case 'cmd':
          _appendOutput((message['data'] ?? '').toString());
          if (!widget.nativeProtocol) _updateControl(true);
        case 'pong':
          break;
        default:
          _appendOutput(text);
      }
    } catch (_) {
      _appendOutput(text);
    }
  }

  void _handleBaseline(Map<String, dynamic> message) {
    final requestId = (message['requestId'] ?? '').toString();
    if (_pendingResyncId.isNotEmpty && requestId != _pendingResyncId) return;
    final chunkIndex = (message['chunkIndex'] as num?)?.toInt() ?? 0;
    final chunkCount = (message['chunkCount'] as num?)?.toInt() ?? 1;
    if (message['truncated'] == true && chunkIndex == 0) {
      _replaceOutput('[GoPanel] 较早的终端输出已截断。\n');
    }
    _appendOutput((message['data'] ?? '').toString());
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
    _appendOutput((message['data'] ?? '').toString());
    _lastSequence = sequence;
    _send('ack', sequence.toString());
  }

  void _handleDisconnect() {
    _pingTimer?.cancel();
    if (mounted) {
      setState(() {
        _connected = false;
        _hasControl = false;
      });
    }
    if (_closing || _reconnectTimer != null) return;
    if (mounted) setState(() => _reconnecting = true);
    _reconnectTimer = Timer(const Duration(milliseconds: 1500), () {
      _reconnectTimer = null;
      _connect();
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

  void _sendCommand() {
    final command = _commandController.text;
    if (command.trim().isEmpty || !_hasControl) return;
    _commandController.clear();
    _send('cmd', '$command\r');
    _commandFocusNode.requestFocus();
  }

  void _updateControl(bool value) {
    if (!mounted || _hasControl == value) return;
    setState(() => _hasControl = value);
  }

  void _appendOutput(String value) {
    if (value.isEmpty) return;
    final clean = _stripAnsi(
      value,
    ).replaceAll('\r\n', '\n').replaceAll('\r', '');
    _output.write(clean);
    if (_output.length > _maxOutputLength) {
      final content = _output.toString();
      _output
        ..clear()
        ..write(content.substring(content.length - _maxOutputLength));
    }
    if (!mounted) return;
    setState(() {});
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
    });
  }

  void _replaceOutput(String value) {
    _output
      ..clear()
      ..write(value);
  }

  String _stripAnsi(String value) {
    return value.replaceAll(
      RegExp(r'\x1B(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1B\\))'),
      '',
    );
  }

  void _openSessionStatus() {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => const AiChatScreen()));
  }

  void _openFiles() {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeWorkspaceFilesScreen(
          sessionId: widget.session.id,
          sessionTitle: widget.session.title.isEmpty
              ? '开发 #${widget.session.id}'
              : widget.session.title,
        ),
      ),
    );
  }

  void _sendShortcut(String value) {
    _send('cmd', value);
    _commandFocusNode.requestFocus();
  }

  void _handleEnterShortcut() {
    if (_commandController.text.trim().isEmpty) {
      _sendShortcut('\r');
      return;
    }
    _sendCommand();
  }

  @override
  Widget build(BuildContext context) {
    final title = widget.session.title.isEmpty
        ? '开发 #${widget.session.id}'
        : widget.session.title;
    return Scaffold(
      backgroundColor: const Color(0xFF0B1020),
      appBar: AppBar(
        toolbarHeight: 52,
        backgroundColor: const Color(0xFF0F172A),
        foregroundColor: Colors.white,
        titleSpacing: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
            ),
            Text(
              widget.projectName,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 10, color: Colors.white54),
            ),
          ],
        ),
        actions: [
          if (widget.nativeProtocol && !_hasControl)
            IconButton(
              tooltip: '接管终端',
              onPressed: _connected ? () => _send('take_control', '') : null,
              icon: const Icon(
                Icons.lock_outline_rounded,
                color: Colors.white54,
              ),
            ),
          IconButton(
            tooltip: '文件',
            onPressed: _openFiles,
            icon: const Icon(Icons.folder_outlined),
          ),
          IconButton(
            tooltip: '会话状态与指令',
            onPressed: _openSessionStatus,
            icon: const Icon(Icons.timeline_rounded),
          ),
          CodeTerminalConnectionDot(
            connected: _connected,
            reconnecting: _reconnecting,
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTap: () => _commandFocusNode.requestFocus(),
              child: SingleChildScrollView(
                controller: _scrollController,
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 16),
                child: SizedBox(
                  width: double.infinity,
                  child: SelectableText(
                    _output.isEmpty
                        ? _reconnecting
                              ? '正在重新连接终端…'
                              : '正在连接开发终端…'
                        : _output.toString(),
                    style: const TextStyle(
                      color: Color(0xFFD4D4D8),
                      fontFamily: 'monospace',
                      fontSize: 12,
                      height: 1.45,
                    ),
                  ),
                ),
              ),
            ),
          ),
          Container(
            color: const Color(0xFF0B1020),
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                const Text(
                  '❯',
                  style: TextStyle(
                    color: Color(0xFF60A5FA),
                    fontFamily: 'monospace',
                    fontSize: 20,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    controller: _commandController,
                    focusNode: _commandFocusNode,
                    enabled: _connected && _hasControl,
                    onSubmitted: (_) => _sendCommand(),
                    textInputAction: TextInputAction.send,
                    minLines: 1,
                    maxLines: 3,
                    style: const TextStyle(
                      color: Color(0xFFF4F4F5),
                      fontFamily: 'monospace',
                      fontSize: 14,
                    ),
                    decoration: InputDecoration(
                      hintText: _connected && _hasControl
                          ? ''
                          : _connected
                          ? '终端只读'
                          : '等待连接',
                      hintStyle: const TextStyle(
                        color: Color(0xFF64748B),
                        fontFamily: 'monospace',
                      ),
                      filled: false,
                      border: InputBorder.none,
                      enabledBorder: InputBorder.none,
                      disabledBorder: InputBorder.none,
                      focusedBorder: InputBorder.none,
                      contentPadding: const EdgeInsets.symmetric(vertical: 10),
                    ),
                  ),
                ),
              ],
            ),
          ),
          CodeTerminalShortcutBar(
            enabled: _connected && _hasControl,
            onShortcut: _sendShortcut,
            onEnter: _handleEnterShortcut,
          ),
        ],
      ),
    );
  }
}
