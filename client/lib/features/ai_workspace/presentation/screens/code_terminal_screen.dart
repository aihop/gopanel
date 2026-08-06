import 'package:flutter/material.dart';
import 'package:xterm/xterm.dart';

import '../../models/ai_dev_session.dart';
import '../../models/code_project_terminal_session.dart';
import '../code_workspace_text.dart';
import '../controllers/code_terminal_client.dart';
import '../controllers/code_terminal_socket.dart';
import '../widgets/code_terminal_controls.dart';
import 'ai_chat_screen.dart';
import 'code_workspace_files_screen.dart';

const _terminalTheme = TerminalTheme(
  cursor: Color(0xFF60A5FA),
  selection: Color(0x663B82F6),
  foreground: Color(0xFFD4D4D8),
  background: Color(0xFF0B1020),
  black: Color(0xFF0B1020),
  red: Color(0xFFEF4444),
  green: Color(0xFF22C55E),
  yellow: Color(0xFFF59E0B),
  blue: Color(0xFF3B82F6),
  magenta: Color(0xFFA855F7),
  cyan: Color(0xFF06B6D4),
  white: Color(0xFFE4E4E7),
  brightBlack: Color(0xFF64748B),
  brightRed: Color(0xFFF87171),
  brightGreen: Color(0xFF4ADE80),
  brightYellow: Color(0xFFFBBF24),
  brightBlue: Color(0xFF60A5FA),
  brightMagenta: Color(0xFFC084FC),
  brightCyan: Color(0xFF22D3EE),
  brightWhite: Color(0xFFFAFAFA),
  searchHitBackground: Color(0xFFF59E0B),
  searchHitBackgroundCurrent: Color(0xFF3B82F6),
  searchHitForeground: Color(0xFF0B1020),
);

class CodeTerminalScreen extends StatefulWidget {
  const CodeTerminalScreen({
    super.key,
    required this.session,
    required this.terminalId,
    required this.task,
    required this.projectName,
    required this.nativeProtocol,
  }) : terminalMode = CodeTerminalMode.aiSession,
       projectTerminal = null;

  const CodeTerminalScreen.project({
    super.key,
    required this.projectTerminal,
    required this.terminalId,
    required this.projectName,
  }) : session = null,
       task = null,
       nativeProtocol = true,
       terminalMode = CodeTerminalMode.projectTerminal;

  final AiDevSession? session;
  final AiTaskSummary? task;
  final String projectName;
  final bool nativeProtocol;
  final int terminalId;
  final CodeTerminalMode terminalMode;
  final CodeProjectTerminalSession? projectTerminal;

  @override
  State<CodeTerminalScreen> createState() => _CodeTerminalScreenState();
}

class _CodeTerminalScreenState extends State<CodeTerminalScreen> {
  late final Terminal _terminal;
  late final TerminalController _terminalController;
  late final FocusNode _terminalFocusNode;
  final _terminalViewKey = GlobalKey<TerminalViewState>();
  late final CodeTerminalClient _client;

  @override
  void initState() {
    super.initState();
    _terminal = Terminal(maxLines: 5000);
    _terminalController = TerminalController();
    _terminalFocusNode = FocusNode();
    _client = CodeTerminalClient(
      terminal: _terminal,
      sessionId: widget.terminalId,
      nativeProtocol: widget.nativeProtocol,
      mode: widget.terminalMode,
    )..addListener(_refresh);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _client.connect();
    });
  }

  @override
  void dispose() {
    _client
      ..removeListener(_refresh)
      ..dispose();
    _terminalController.dispose();
    _terminalFocusNode.dispose();
    super.dispose();
  }

  void _refresh() {
    if (mounted) setState(() {});
  }

  void _openSessionStatus() {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => const AiChatScreen()));
  }

  void _openFiles() {
    final session = widget.session;
    if (session == null) return;
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeWorkspaceFilesScreen(
          sessionId: session.id,
          sessionTitle: session.title.isEmpty
              ? '开发 #${session.id}'
              : session.title,
        ),
      ),
    );
  }

  void _sendShortcut(String value) {
    _client.sendInput(value);
    _terminalFocusNode.requestFocus();
  }

  void _showKeyboard() {
    if (!_client.canInput) return;
    _terminalFocusNode.requestFocus();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _terminalViewKey.currentState?.requestKeyboard();
    });
  }

  void _toggleControl() {
    if (_client.hasControl) {
      _client.releaseControl();
    } else {
      _client.takeControl();
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = widget.session;
    final projectTerminal = widget.projectTerminal;
    final sessionTitle = session == null
        ? CodeWorkspaceText.t(context, 'terminal.projectTitle')
        : session.title.isEmpty
        ? '开发 #${session.id}'
        : session.title;
    final taskTitle = widget.task == null
        ? CodeWorkspaceText.t(context, 'terminal.noTask')
        : widget.task!.title.isEmpty
        ? '任务 #${widget.task!.id}'
        : widget.task!.title;
    final title = projectTerminal == null
        ? CodeWorkspaceText.format(context, 'terminal.taskTitle', {
            'task': taskTitle,
          })
        : widget.projectName;
    final subtitle = projectTerminal?.workDir ?? sessionTitle;
    return Scaffold(
      backgroundColor: const Color(0xFF0B1020),
      appBar: AppBar(
        toolbarHeight: 44,
        leadingWidth: 44,
        backgroundColor: const Color(0xFF0F172A),
        foregroundColor: Colors.white,
        titleSpacing: 0,
        iconTheme: const IconThemeData(size: 20),
        title: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
            ),
            Text(
              subtitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 10, color: Color(0xFF94A3B8)),
            ),
          ],
        ),
        actions: [
          CodeTerminalProjectLabel(projectName: widget.projectName),
          CodeTerminalConnectionDot(
            connected: _client.connected,
            reconnecting: _client.reconnecting,
          ),
          if (_client.connectionFailed)
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'terminal.reconnect'),
              onPressed: _client.reconnect,
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints.tightFor(width: 36, height: 36),
              icon: const Icon(Icons.refresh_rounded, color: Color(0xFFFBBF24)),
            ),
          if (widget.nativeProtocol)
            IconButton(
              tooltip: _client.hasControl
                  ? CodeWorkspaceText.t(context, 'terminal.releaseControl')
                  : CodeWorkspaceText.t(context, 'terminal.takeControl'),
              onPressed: _client.connected ? _toggleControl : null,
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints.tightFor(width: 36, height: 36),
              icon: Icon(
                _client.hasControl
                    ? Icons.lock_open_rounded
                    : Icons.lock_outline_rounded,
                color: _client.hasControl
                    ? const Color(0xFF60A5FA)
                    : Colors.white54,
              ),
            ),
          if (session != null) ...[
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'terminal.sessionStatus'),
              onPressed: _openSessionStatus,
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints.tightFor(width: 36, height: 36),
              icon: const Icon(Icons.timeline_rounded),
            ),
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'terminal.files'),
              onPressed: _openFiles,
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints.tightFor(width: 36, height: 36),
              icon: const Icon(Icons.folder_outlined),
            ),
          ],
          const SizedBox(width: 4),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: TerminalView(
              _terminal,
              key: _terminalViewKey,
              controller: _terminalController,
              focusNode: _terminalFocusNode,
              autofocus: true,
              theme: _terminalTheme,
              textStyle: const TerminalStyle(
                fontSize: 12,
                height: 1.2,
                fontFamily: 'Menlo',
                fontFamilyFallback: [
                  'PingFang SC',
                  'Noto Sans Mono CJK SC',
                  'Noto Sans CJK SC',
                  'Noto Sans SC',
                  'Microsoft YaHei',
                  'sans-serif',
                ],
              ),
              padding: const EdgeInsets.fromLTRB(8, 6, 6, 4),
              readOnly: !_client.canInput,
              deleteDetection: true,
              cursorType: TerminalCursorType.verticalBar,
              keyboardType: TextInputType.text,
              keyboardAppearance: Brightness.dark,
            ),
          ),
          CodeTerminalShortcutBar(
            enabled: _client.canInput,
            onShortcut: _sendShortcut,
            onEnter: () => _sendShortcut('\r'),
            onKeyboard: _showKeyboard,
          ),
        ],
      ),
    );
  }
}
