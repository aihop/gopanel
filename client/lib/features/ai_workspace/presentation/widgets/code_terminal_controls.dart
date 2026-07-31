import 'package:flutter/material.dart';

class CodeTerminalSessionHeader extends StatelessWidget {
  const CodeTerminalSessionHeader({
    super.key,
    required this.projectName,
    required this.agentName,
    required this.stage,
    required this.connected,
    required this.reconnecting,
    required this.hasControl,
  });

  final String projectName;
  final String agentName;
  final String stage;
  final bool connected;
  final bool reconnecting;
  final bool hasControl;

  @override
  Widget build(BuildContext context) {
    final statusText = connected
        ? hasControl
              ? '可输入'
              : '只读'
        : reconnecting
        ? '重连中'
        : '未连接';
    final statusColor = connected
        ? hasControl
              ? const Color(0xFF4ADE80)
              : const Color(0xFFFBBF24)
        : const Color(0xFF71717A);
    return Container(
      color: const Color(0xFF181818),
      padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: const Color(0xFF083344),
              borderRadius: BorderRadius.circular(14),
            ),
            child: const Icon(Icons.terminal_rounded, color: Color(0xFF22D3EE)),
          ),
          const SizedBox(width: 11),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  projectName,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  '${agentName.isEmpty ? 'Code' : agentName} · ${_stageLabel(stage)}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: Color(0xFF71717A),
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: statusColor.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(999),
              border: Border.all(color: statusColor.withValues(alpha: 0.28)),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 7,
                  height: 7,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: statusColor,
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  statusText,
                  style: TextStyle(
                    color: statusColor,
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
        ],
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
    required this.onKeyboard,
  });

  final bool enabled;
  final ValueChanged<String> onShortcut;
  final VoidCallback onEnter;
  final VoidCallback onKeyboard;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Container(
        color: const Color(0xFF171717),
        padding: const EdgeInsets.fromLTRB(8, 8, 8, 7),
        child: Column(
          children: [
            _ShortcutRow(
              enabled: enabled,
              items: const [
                ('esc', '\x1b'),
                ('tab', '\t'),
                ('↑', '\x1b[A'),
                ('⌫', '\x7f'),
              ],
              trailing: _TerminalKey(
                label: '↵',
                enabled: enabled,
                accent: true,
                onPressed: onEnter,
              ),
              onShortcut: onShortcut,
            ),
            const SizedBox(height: 7),
            _ShortcutRow(
              enabled: enabled,
              items: const [
                ('^C', '\x03'),
                ('ctrl', '\x00'),
                ('shft', '\x00'),
                ('alt', '\x00'),
                ('/', '/'),
                ('←', '\x1b[D'),
                ('↓', '\x1b[B'),
                ('→', '\x1b[C'),
              ],
              trailing: _TerminalKey(
                label: '⌨',
                enabled: enabled,
                onPressed: onKeyboard,
              ),
              onShortcut: onShortcut,
            ),
          ],
        ),
      ),
    );
  }
}

class _ShortcutRow extends StatelessWidget {
  const _ShortcutRow({
    required this.enabled,
    required this.items,
    required this.trailing,
    required this.onShortcut,
  });

  final bool enabled;
  final List<(String, String)> items;
  final Widget trailing;
  final ValueChanged<String> onShortcut;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 42,
      child: ListView(
        scrollDirection: Axis.horizontal,
        children: [
          for (final item in items) ...[
            _TerminalKey(
              label: item.$1,
              enabled: enabled && item.$2 != '\x00',
              danger: item.$1 == '^C',
              onPressed: () => onShortcut(item.$2),
            ),
            const SizedBox(width: 7),
          ],
          trailing,
        ],
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
    final background = accent
        ? const Color(0xFF083344)
        : danger
        ? const Color(0xFF451A1A)
        : const Color(0xFF27272A);
    final foreground = accent
        ? const Color(0xFF22D3EE)
        : danger
        ? const Color(0xFFF87171)
        : const Color(0xFFE4E4E7);
    return SizedBox(
      height: 42,
      child: Material(
        color: background,
        borderRadius: BorderRadius.circular(11),
        child: InkWell(
          onTap: enabled ? onPressed : null,
          borderRadius: BorderRadius.circular(11),
          child: ConstrainedBox(
            constraints: const BoxConstraints(minWidth: 48),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 13),
              child: Center(
                child: Text(
                  label,
                  style: TextStyle(
                    color: enabled
                        ? foreground
                        : foreground.withValues(alpha: 0.3),
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

String _stageLabel(String stage) => switch (stage) {
  'instruction_queued' => '排队中',
  'awaiting_approval' => '待审批',
  'executing' => '执行中',
  'completed' => '已完成',
  'preview_ready' => '预览就绪',
  'failed' => '失败',
  'idle' => '空闲',
  _ => stage.isEmpty ? '会话中' : stage,
};
