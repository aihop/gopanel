import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../code_workspace_text.dart';

TextEditingValue insertCodeTerminalText(TextEditingValue value, String text) {
  final selection = value.selection.isValid
      ? value.selection
      : TextSelection.collapsed(offset: value.text.length);
  final nextText = value.text.replaceRange(
    selection.start,
    selection.end,
    text,
  );
  return TextEditingValue(
    text: nextText,
    selection: TextSelection.collapsed(offset: selection.start + text.length),
  );
}

class CodeTerminalCommandBar extends StatefulWidget {
  const CodeTerminalCommandBar({
    super.key,
    required this.connected,
    required this.hasControl,
    required this.supportsControl,
    required this.onTakeControl,
    required this.onReleaseControl,
    required this.onSend,
    required this.onShortcut,
    required this.onTerminalKeyboard,
  });

  final bool connected;
  final bool hasControl;
  final bool supportsControl;
  final VoidCallback onTakeControl;
  final VoidCallback onReleaseControl;
  final ValueChanged<String> onSend;
  final ValueChanged<String> onShortcut;
  final VoidCallback onTerminalKeyboard;

  @override
  State<CodeTerminalCommandBar> createState() => _CodeTerminalCommandBarState();
}

class _CodeTerminalCommandBarState extends State<CodeTerminalCommandBar> {
  final _controller = TextEditingController();
  final _focusNode = FocusNode();
  final List<String> _history = [];
  int _historyIndex = -1;

  @override
  void dispose() {
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final canSend = widget.connected && widget.hasControl;
    return SafeArea(
      top: false,
      child: DecoratedBox(
        decoration: const BoxDecoration(
          color: Color(0xFF020617),
          border: Border(top: BorderSide(color: Color(0xFF1E293B))),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(7, 7, 7, 6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (widget.connected && !widget.hasControl) ...[
                    Padding(
                      padding: const EdgeInsets.only(left: 4, bottom: 5),
                      child: Text(
                        CodeWorkspaceText.t(
                          context,
                          'terminal.takeControlHint',
                        ),
                        style: const TextStyle(
                          color: Color(0xFFFBBF24),
                          fontSize: 11,
                        ),
                      ),
                    ),
                  ],
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      if (widget.supportsControl) ...[
                        _ControlButton(
                          connected: widget.connected,
                          hasControl: widget.hasControl,
                          onPressed: widget.hasControl
                              ? widget.onReleaseControl
                              : widget.onTakeControl,
                        ),
                        const SizedBox(width: 6),
                      ],
                      Expanded(
                        child: TextField(
                          controller: _controller,
                          focusNode: _focusNode,
                          enabled: widget.connected,
                          minLines: 1,
                          maxLines: 3,
                          keyboardType: TextInputType.text,
                          textInputAction: TextInputAction.send,
                          autocorrect: false,
                          enableSuggestions: false,
                          smartDashesType: SmartDashesType.disabled,
                          smartQuotesType: SmartQuotesType.disabled,
                          onTap: _requestControl,
                          onSubmitted: (_) => _submit(),
                          style: const TextStyle(
                            color: Colors.white,
                            fontFamily: 'monospace',
                            fontSize: 14,
                          ),
                          decoration: InputDecoration(
                            isDense: true,
                            filled: true,
                            fillColor: const Color(0xFF111827),
                            hintText: widget.connected
                                ? CodeWorkspaceText.t(
                                    context,
                                    'terminal.commandHint',
                                  )
                                : CodeWorkspaceText.t(
                                    context,
                                    'terminal.disconnected',
                                  ),
                            hintStyle: const TextStyle(
                              color: Color(0xFF64748B),
                              fontSize: 13,
                            ),
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 12,
                              vertical: 11,
                            ),
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(11),
                              borderSide: const BorderSide(
                                color: Color(0xFF334155),
                              ),
                            ),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(11),
                              borderSide: const BorderSide(
                                color: Color(0xFF334155),
                              ),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(11),
                              borderSide: const BorderSide(
                                color: Color(0xFF60A5FA),
                              ),
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: 6),
                      IconButton.filled(
                        tooltip: CodeWorkspaceText.t(
                          context,
                          'terminal.sendCommand',
                        ),
                        onPressed: canSend ? _submit : null,
                        style: IconButton.styleFrom(
                          backgroundColor: const Color(0xFF1D4ED8),
                          disabledBackgroundColor: const Color(0xFF111827),
                          foregroundColor: Colors.white,
                        ),
                        icon: const Icon(Icons.send_rounded, size: 19),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            SizedBox(
              height: 42,
              child: ListView(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.fromLTRB(7, 2, 7, 5),
                children: [
                  _key('Esc', () => _shortcut('\x1b'), enabled: canSend),
                  _key('Tab', () => _shortcut('\t'), enabled: canSend),
                  _key(
                    '^C',
                    () => _shortcut('\x03'),
                    enabled: canSend,
                    danger: true,
                  ),
                  _key('^D', () => _shortcut('\x04'), enabled: canSend),
                  _key(
                    CodeWorkspaceText.t(context, 'terminal.historyPrevious'),
                    () => _historyMove(-1),
                    enabled: _history.isNotEmpty,
                  ),
                  _key(
                    CodeWorkspaceText.t(context, 'terminal.historyNext'),
                    () => _historyMove(1),
                    enabled: _history.isNotEmpty,
                  ),
                  for (final symbol in const ['/', '-', '_', '.', '~', '|'])
                    _key(
                      symbol,
                      () => _insert(symbol),
                      enabled: widget.connected,
                      accent: true,
                    ),
                  _key(
                    CodeWorkspaceText.t(context, 'terminal.paste'),
                    _paste,
                    enabled: widget.connected,
                    icon: Icons.content_paste_rounded,
                  ),
                  _key(
                    CodeWorkspaceText.t(context, 'terminal.nativeKeyboard'),
                    widget.onTerminalKeyboard,
                    enabled: canSend,
                    icon: Icons.keyboard_alt_outlined,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _key(
    String label,
    VoidCallback onPressed, {
    required bool enabled,
    bool accent = false,
    bool danger = false,
    IconData? icon,
  }) {
    final color = danger
        ? const Color(0xFFFCA5A5)
        : accent
        ? const Color(0xFF93C5FD)
        : const Color(0xFFCBD5E1);
    return Padding(
      padding: const EdgeInsets.only(right: 5),
      child: OutlinedButton.icon(
        onPressed: enabled ? onPressed : null,
        icon: icon == null ? const SizedBox.shrink() : Icon(icon, size: 15),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(38, 34),
          padding: const EdgeInsets.symmetric(horizontal: 9),
          visualDensity: VisualDensity.compact,
          foregroundColor: color,
          side: BorderSide(
            color: enabled ? color.withValues(alpha: 0.25) : Colors.white10,
          ),
          textStyle: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 11,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
    );
  }

  void _requestControl() {
    if (widget.connected && !widget.hasControl && widget.supportsControl) {
      widget.onTakeControl();
    }
  }

  void _submit() {
    final command = _controller.text;
    if (!widget.hasControl || command.trim().isEmpty) return;
    widget.onSend('$command\r');
    if (_history.isEmpty || _history.last != command) {
      _history.add(command);
      if (_history.length > 30) _history.removeAt(0);
    }
    _historyIndex = -1;
    _controller.clear();
    HapticFeedback.selectionClick();
    _focusNode.requestFocus();
    setState(() {});
  }

  void _shortcut(String value) {
    widget.onShortcut(value);
    HapticFeedback.selectionClick();
    _focusNode.requestFocus();
  }

  void _insert(String text) {
    _controller.value = insertCodeTerminalText(_controller.value, text);
    _focusNode.requestFocus();
  }

  Future<void> _paste() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final text = data?.text;
    if (text == null || text.isEmpty || !mounted) return;
    _insert(text.replaceAll('\r\n', '\n').replaceAll('\n', ' '));
  }

  void _historyMove(int direction) {
    if (_history.isEmpty) return;
    if (_historyIndex < 0) {
      _historyIndex = _history.length - 1;
    } else {
      _historyIndex = (_historyIndex + direction).clamp(0, _history.length - 1);
    }
    _controller.value = TextEditingValue(
      text: _history[_historyIndex],
      selection: TextSelection.collapsed(
        offset: _history[_historyIndex].length,
      ),
    );
    _focusNode.requestFocus();
    setState(() {});
  }
}

class _ControlButton extends StatelessWidget {
  const _ControlButton({
    required this.connected,
    required this.hasControl,
    required this.onPressed,
  });

  final bool connected;
  final bool hasControl;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return IconButton.outlined(
      tooltip: CodeWorkspaceText.t(
        context,
        hasControl ? 'terminal.releaseControl' : 'terminal.takeControl',
      ),
      onPressed: connected ? onPressed : null,
      style: IconButton.styleFrom(
        foregroundColor: hasControl
            ? const Color(0xFF4ADE80)
            : const Color(0xFF93C5FD),
        side: BorderSide(
          color: hasControl ? const Color(0x664ADE80) : const Color(0x6693C5FD),
        ),
      ),
      icon: Icon(
        hasControl ? Icons.keyboard_hide_rounded : Icons.keyboard_alt_outlined,
      ),
    );
  }
}
