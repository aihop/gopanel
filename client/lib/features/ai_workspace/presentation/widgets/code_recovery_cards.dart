import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../models/code_session_recovery.dart';
import '../code_workspace_text.dart';

class CodeInitializationCard extends StatelessWidget {
  const CodeInitializationCard({
    super.key,
    required this.initialization,
    required this.isRetrying,
    required this.onRetry,
  });

  final CodeSessionInitialization initialization;
  final bool isRetrying;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final failed = initialization.isFailed;
    final initializing = initialization.isInitializing;
    final color = failed
        ? AppTheme.error
        : initializing
        ? Colors.orange
        : AppTheme.success;
    return PanelCard(
      title: Text(CodeWorkspaceText.t(context, 'recovery.initialization')),
      trailing: _StatusBadge(
        label: CodeWorkspaceText.t(
          context,
          failed
              ? 'recovery.initializationFailed'
              : initializing
              ? 'recovery.initializing'
              : 'recovery.initialized',
        ),
        color: color,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            initialization.errorMessage.isNotEmpty
                ? initialization.errorMessage
                : CodeWorkspaceText.t(
                    context,
                    failed
                        ? 'recovery.initializationFailedHint'
                        : initializing
                        ? 'recovery.initializingHint'
                        : 'recovery.initializedHint',
                  ),
            style: const TextStyle(color: AppTheme.textSecondary, height: 1.45),
          ),
          if (initialization.canRetry) ...[
            const SizedBox(height: 14),
            FilledButton.icon(
              onPressed: isRetrying ? null : onRetry,
              icon: isRetrying
                  ? const SizedBox.square(
                      dimension: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.refresh_rounded),
              label: Text(
                CodeWorkspaceText.t(context, 'recovery.retryInitialization'),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class CodeHistoryMessagesCard extends StatelessWidget {
  const CodeHistoryMessagesCard({super.key, required this.messages});

  final List<CodeHistoryMessage> messages;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Text(CodeWorkspaceText.t(context, 'recovery.conversation')),
      trailing: Text('${messages.length}'),
      child: messages.isEmpty
          ? Text(
              CodeWorkspaceText.t(context, 'recovery.noMessages'),
              style: const TextStyle(color: AppTheme.textSecondary),
            )
          : Column(
              children: [
                for (var index = 0; index < messages.length; index++) ...[
                  _HistoryMessage(message: messages[index]),
                  if (index != messages.length - 1) const Divider(height: 24),
                ],
              ],
            ),
    );
  }
}

class CodeExecutionRunsCard extends StatelessWidget {
  const CodeExecutionRunsCard({
    super.key,
    required this.runs,
    required this.total,
    required this.isLoadingMore,
    required this.retryingInstructionId,
    required this.retriedInstructionIds,
    required this.onRetry,
    required this.onLoadMore,
  });

  final List<CodeExecutionRun> runs;
  final int total;
  final bool isLoadingMore;
  final int? retryingInstructionId;
  final Set<int> retriedInstructionIds;
  final ValueChanged<CodeExecutionRun> onRetry;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Text(CodeWorkspaceText.t(context, 'recovery.runs')),
      trailing: Text('${runs.length}/$total'),
      child: runs.isEmpty
          ? Text(
              CodeWorkspaceText.t(context, 'recovery.noRuns'),
              style: const TextStyle(color: AppTheme.textSecondary),
            )
          : Column(
              children: [
                for (var index = 0; index < runs.length; index++) ...[
                  _ExecutionRunItem(
                    run: runs[index],
                    isRetrying:
                        retryingInstructionId == runs[index].instructionId,
                    wasRetried: retriedInstructionIds.contains(
                      runs[index].instructionId,
                    ),
                    onRetry: () => onRetry(runs[index]),
                  ),
                  if (index != runs.length - 1) const Divider(height: 24),
                ],
                if (runs.length < total) ...[
                  const SizedBox(height: 14),
                  OutlinedButton.icon(
                    onPressed: isLoadingMore ? null : onLoadMore,
                    icon: isLoadingMore
                        ? const SizedBox.square(
                            dimension: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.expand_more_rounded),
                    label: Text(
                      CodeWorkspaceText.t(context, 'recovery.loadMore'),
                    ),
                  ),
                ],
              ],
            ),
    );
  }
}

class _HistoryMessage extends StatelessWidget {
  const _HistoryMessage({required this.message});

  final CodeHistoryMessage message;

  @override
  Widget build(BuildContext context) {
    final user = message.role.toLowerCase() == 'user';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(
              user ? Icons.person_outline : Icons.smart_toy_outlined,
              size: 17,
              color: user ? AppTheme.primaryBlue : AppTheme.success,
            ),
            const SizedBox(width: 6),
            Text(
              CodeWorkspaceText.t(
                context,
                user ? 'recovery.user' : 'recovery.agent',
              ),
              style: const TextStyle(fontWeight: FontWeight.w800),
            ),
            const Spacer(),
            Text(
              _formatTime(message.createdAt),
              style: const TextStyle(color: AppTheme.textLight, fontSize: 11),
            ),
          ],
        ),
        const SizedBox(height: 8),
        SelectableText(
          message.content,
          style: const TextStyle(height: 1.5, color: AppTheme.textSecondary),
        ),
      ],
    );
  }
}

class _ExecutionRunItem extends StatelessWidget {
  const _ExecutionRunItem({
    required this.run,
    required this.isRetrying,
    required this.wasRetried,
    required this.onRetry,
  });

  final CodeExecutionRun run;
  final bool isRetrying;
  final bool wasRetried;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final color = _runColor(run.status);
    final summary = run.errorMessage.isNotEmpty
        ? run.errorMessage
        : run.output.isNotEmpty
        ? run.output
        : run.prompt;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            _StatusBadge(label: _runLabel(context, run.status), color: color),
            const Spacer(),
            Text(
              _formatTime(run.createdAt),
              style: const TextStyle(color: AppTheme.textLight, fontSize: 11),
            ),
          ],
        ),
        if (run.prompt.isNotEmpty) ...[
          const SizedBox(height: 8),
          Text(
            run.prompt,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
        ],
        if (summary.isNotEmpty && summary != run.prompt) ...[
          const SizedBox(height: 6),
          Text(
            summary,
            maxLines: 4,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: run.status == 'failed'
                  ? AppTheme.error
                  : AppTheme.textSecondary,
              height: 1.4,
            ),
          ),
        ],
        const SizedBox(height: 8),
        Wrap(
          spacing: 10,
          runSpacing: 6,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            Text(
              CodeWorkspaceText.format(context, 'recovery.duration', {
                'duration': run.durationMs,
              }),
              style: const TextStyle(color: AppTheme.textLight, fontSize: 11),
            ),
            if (run.totalTokens > 0)
              Text(
                CodeWorkspaceText.format(context, 'recovery.tokens', {
                  'tokens': run.totalTokens,
                }),
                style: const TextStyle(color: AppTheme.textLight, fontSize: 11),
              ),
            if (run.canRetry && !wasRetried)
              TextButton.icon(
                onPressed: isRetrying ? null : onRetry,
                icon: isRetrying
                    ? const SizedBox.square(
                        dimension: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.replay_rounded, size: 17),
                label: Text(
                  CodeWorkspaceText.t(context, 'recovery.retryInstruction'),
                ),
              ),
            if (wasRetried)
              Text(
                CodeWorkspaceText.t(context, 'recovery.retryQueued'),
                style: const TextStyle(
                  color: AppTheme.success,
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                ),
              ),
          ],
        ),
      ],
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

Color _runColor(String status) => switch (status) {
  'completed' => AppTheme.success,
  'failed' => AppTheme.error,
  'cancelled' || 'stopped' => Colors.orange,
  'running' => AppTheme.primaryBlue,
  'awaiting_approval' || 'pending_approval' => Colors.amber,
  _ => AppTheme.textSecondary,
};

String _runLabel(BuildContext context, String status) {
  final key = switch (status) {
    'completed' => 'recovery.statusCompleted',
    'failed' => 'recovery.statusFailed',
    'cancelled' || 'stopped' => 'recovery.statusStopped',
    'running' => 'recovery.statusRunning',
    'awaiting_approval' ||
    'pending_approval' => 'recovery.statusAwaitingApproval',
    'queued' => 'recovery.statusQueued',
    _ => 'recovery.statusUnknown',
  };
  return CodeWorkspaceText.t(context, key);
}

String _formatTime(DateTime? value) {
  if (value == null) return '-';
  final local = value.toLocal();
  String two(int number) => number.toString().padLeft(2, '0');
  return '${local.month}/${local.day} ${two(local.hour)}:${two(local.minute)}';
}
