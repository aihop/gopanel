import 'package:flutter/material.dart';

import '../code_workspace_text.dart';

class AiChatErrorBanner extends StatelessWidget {
  const AiChatErrorBanner({
    super.key,
    required this.message,
    required this.onRetry,
    required this.onDismiss,
  });

  final String message;
  final VoidCallback onRetry;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    return MaterialBanner(
      backgroundColor: const Color(0xFF451A1A),
      content: Text(message, style: const TextStyle(color: Colors.white)),
      leading: const Icon(Icons.error_outline_rounded, color: Colors.redAccent),
      actions: [
        TextButton(
          onPressed: onDismiss,
          child: Text(CodeWorkspaceText.t(context, 'action.dismiss')),
        ),
        TextButton(
          onPressed: onRetry,
          child: Text(CodeWorkspaceText.t(context, 'action.retry')),
        ),
      ],
    );
  }
}

class AiChatEmptyWorkspace extends StatelessWidget {
  const AiChatEmptyWorkspace({super.key, required this.onOpenSessions});

  final VoidCallback onOpenSessions;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.terminal_rounded, color: Colors.white54, size: 52),
            const SizedBox(height: 16),
            Text(
              CodeWorkspaceText.t(context, 'chat.emptyTitle'),
              style: const TextStyle(
                color: Colors.white,
                fontSize: 18,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              CodeWorkspaceText.t(context, 'chat.emptyHint'),
              textAlign: TextAlign.center,
              style: const TextStyle(color: Colors.white60, height: 1.5),
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: onOpenSessions,
              icon: const Icon(Icons.add_rounded),
              label: Text(CodeWorkspaceText.t(context, 'chat.chooseSession')),
            ),
          ],
        ),
      ),
    );
  }
}

class AiChatExecutionIndicator extends StatelessWidget {
  const AiChatExecutionIndicator({
    super.key,
    required this.isStopping,
    required this.onStop,
  });

  final bool isStopping;
  final VoidCallback onStop;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 8, 8),
      child: Row(
        children: [
          const SizedBox.square(
            dimension: 16,
            child: CircularProgressIndicator(
              color: Colors.greenAccent,
              strokeWidth: 2,
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              CodeWorkspaceText.t(context, 'chat.executing'),
              style: const TextStyle(color: Colors.greenAccent),
            ),
          ),
          TextButton.icon(
            onPressed: isStopping ? null : onStop,
            icon: isStopping
                ? const SizedBox.square(
                    dimension: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.stop_circle_outlined, size: 18),
            label: Text(CodeWorkspaceText.t(context, 'action.stop')),
          ),
        ],
      ),
    );
  }
}
