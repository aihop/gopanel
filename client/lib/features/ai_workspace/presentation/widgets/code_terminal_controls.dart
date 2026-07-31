import 'package:flutter/material.dart';

class CodeTerminalProjectLabel extends StatelessWidget {
  const CodeTerminalProjectLabel({super.key, required this.projectName});

  final String projectName;

  @override
  Widget build(BuildContext context) {
    if (projectName.isEmpty) return const SizedBox.shrink();
    return Tooltip(
      message: projectName,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 72),
        child: Text(
          projectName,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          textAlign: TextAlign.right,
          style: const TextStyle(fontSize: 11, color: Color(0xFF94A3B8)),
        ),
      ),
    );
  }
}

class CodeTerminalConnectionDot extends StatelessWidget {
  const CodeTerminalConnectionDot({
    super.key,
    required this.connected,
    required this.reconnecting,
  });

  final bool connected;
  final bool reconnecting;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 6),
      child: Tooltip(
        message: connected
            ? '已连接'
            : reconnecting
            ? '正在重连'
            : '未连接',
        child: Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: connected
                ? const Color(0xFF4ADE80)
                : reconnecting
                ? const Color(0xFFFBBF24)
                : const Color(0xFF64748B),
          ),
        ),
      ),
    );
  }
}

class CodeTerminalShortcutBar extends StatelessWidget {
  const CodeTerminalShortcutBar({
    super.key,
    required this.enabled,
    required this.onShortcut,
    required this.onEnter,
  });

  final bool enabled;
  final ValueChanged<String> onShortcut;
  final VoidCallback onEnter;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Container(
        height: 40,
        decoration: const BoxDecoration(
          color: Color(0xFF020617),
          border: Border(top: BorderSide(color: Color(0xFF1E293B))),
        ),
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
        child: ListView(
          scrollDirection: Axis.horizontal,
          children: [
            for (final shortcut in const [
              ('Esc', '\x1b'),
              ('Tab', '\t'),
              ('^C', '\x03'),
              ('←', '\x1b[D'),
              ('↑', '\x1b[A'),
              ('↓', '\x1b[B'),
              ('→', '\x1b[C'),
              ('⌫', '\x7f'),
            ]) ...[
              _TerminalKey(
                label: shortcut.$1,
                enabled: enabled,
                danger: shortcut.$1 == '^C',
                onPressed: () => onShortcut(shortcut.$2),
              ),
              const SizedBox(width: 4),
            ],
            _TerminalKey(
              label: '↵',
              enabled: enabled,
              accent: true,
              onPressed: onEnter,
            ),
          ],
        ),
      ),
    );
  }
}

class _TerminalKey extends StatelessWidget {
  const _TerminalKey({
    required this.label,
    required this.enabled,
    required this.onPressed,
    this.accent = false,
    this.danger = false,
  });

  final String label;
  final bool enabled;
  final VoidCallback onPressed;
  final bool accent;
  final bool danger;

  @override
  Widget build(BuildContext context) {
    final foreground = accent
        ? const Color(0xFF93C5FD)
        : danger
        ? const Color(0xFFFCA5A5)
        : const Color(0xFFCBD5E1);
    return Material(
      color: accent
          ? const Color(0xFF1E3A5F)
          : danger
          ? const Color(0xFF3F1D26)
          : const Color(0xFF111827),
      borderRadius: BorderRadius.circular(7),
      child: InkWell(
        onTap: enabled ? onPressed : null,
        borderRadius: BorderRadius.circular(7),
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 36),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Center(
              child: Text(
                label,
                style: TextStyle(
                  color: enabled
                      ? foreground
                      : foreground.withValues(alpha: 0.3),
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
