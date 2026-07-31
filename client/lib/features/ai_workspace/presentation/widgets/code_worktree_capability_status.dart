import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/ai_dev_session.dart';
import '../code_workspace_text.dart';

class CodeWorktreeCapabilityStatus extends StatelessWidget {
  const CodeWorktreeCapabilityStatus({
    super.key,
    required this.capability,
    required this.loading,
    required this.error,
    required this.onRetry,
  });

  final CodeWorktreeCapability? capability;
  final bool loading;
  final String? error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    if (loading) {
      return _StatusPanel(
        icon: Icons.sync_rounded,
        color: AppTheme.primaryBlue,
        title: CodeWorkspaceText.t(context, 'session.worktreeChecking'),
        child: const LinearProgressIndicator(minHeight: 2),
      );
    }
    if (error != null) {
      return _StatusPanel(
        icon: Icons.error_outline_rounded,
        color: AppTheme.error,
        title: CodeWorkspaceText.t(context, 'session.worktreeCheckFailed'),
        child: Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: onRetry,
            icon: const Icon(Icons.refresh_rounded, size: 17),
            label: Text(CodeWorkspaceText.t(context, 'session.worktreeRetry')),
          ),
        ),
      );
    }
    final value = capability;
    if (value == null) return const SizedBox.shrink();
    if (value.dirtyRepositories.isNotEmpty) {
      return _StatusPanel(
        icon: Icons.warning_amber_rounded,
        color: AppTheme.warning,
        title: CodeWorkspaceText.t(context, 'session.worktreeDirty'),
        child: Text(
          CodeWorkspaceText.format(context, 'session.worktreeDirtyHint', {
            'repositories': value.dirtyRepositories.join(', '),
          }),
        ),
      );
    }
    if (!value.available) {
      final reasonKey = switch (value.reason) {
        'not_git' => 'session.worktreeReason_not_git',
        'source_unavailable' => 'session.worktreeReason_source_unavailable',
        _ => 'session.worktreeReason_unknown',
      };
      return _StatusPanel(
        icon: Icons.block_rounded,
        color: AppTheme.error,
        title: CodeWorkspaceText.t(context, 'session.worktreeUnavailable'),
        child: Text(CodeWorkspaceText.t(context, reasonKey)),
      );
    }
    return _StatusPanel(
      icon: Icons.account_tree_outlined,
      color: AppTheme.success,
      title: CodeWorkspaceText.t(context, 'session.worktreeReady'),
      child: Text(CodeWorkspaceText.t(context, 'session.worktreeReadyHint')),
    );
  }
}

class _StatusPanel extends StatelessWidget {
  const _StatusPanel({
    required this.icon,
    required this.color,
    required this.title,
    required this.child,
  });

  final IconData icon;
  final Color color;
  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: color.withValues(alpha: 0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: color),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  title,
                  style: TextStyle(color: color, fontWeight: FontWeight.w700),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          DefaultTextStyle.merge(
            style: const TextStyle(
              color: AppTheme.textSecondary,
              fontSize: 12,
              height: 1.45,
            ),
            child: child,
          ),
        ],
      ),
    );
  }
}
