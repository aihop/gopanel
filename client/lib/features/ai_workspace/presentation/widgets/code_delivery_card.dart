import 'package:flutter/material.dart';

import '../../models/ai_dev_session.dart';
import '../../models/code_delivery_job.dart';
import '../code_workspace_text.dart';

class CodeDeliveryCard extends StatelessWidget {
  const CodeDeliveryCard({
    super.key,
    required this.session,
    required this.delivery,
    required this.loading,
    required this.errorMessage,
    required this.onStart,
  });

  final AiDevSession session;
  final CodeDeliveryJob? delivery;
  final bool loading;
  final String? errorMessage;
  final VoidCallback onStart;

  bool get _managed =>
      session.isolationMode == 'multi_worktree' ||
      (session.sourceWorkDir.isNotEmpty && session.worktreeBranch.isNotEmpty);

  @override
  Widget build(BuildContext context) {
    final job = delivery;
    final accent = _accent(job);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF111827),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: accent.withValues(alpha: 0.24)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.rocket_launch_outlined, color: accent, size: 19),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  CodeWorkspaceText.t(context, 'delivery.title'),
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              if (job != null) _StatusBadge(job: job, color: accent),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            _description(context, job),
            style: const TextStyle(color: Colors.white70, height: 1.45),
          ),
          if (job?.isActive == true) ...[
            const SizedBox(height: 12),
            LinearProgressIndicator(
              value: job!.progress.clamp(0, 100) / 100,
              minHeight: 6,
              borderRadius: BorderRadius.circular(99),
              color: accent,
              backgroundColor: Colors.white12,
            ),
            const SizedBox(height: 8),
            Text(
              _progressMeta(context, job),
              style: const TextStyle(color: Colors.white54, fontSize: 12),
            ),
          ],
          if (job?.conflictFiles.isNotEmpty == true) ...[
            const SizedBox(height: 10),
            for (final file in job!.conflictFiles.take(8))
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  '• $file',
                  style: const TextStyle(
                    color: Colors.orangeAccent,
                    fontFamily: 'monospace',
                    fontSize: 12,
                  ),
                ),
              ),
          ],
          if (errorMessage?.isNotEmpty == true) ...[
            const SizedBox(height: 10),
            Text(
              errorMessage!,
              style: const TextStyle(color: Colors.redAccent, fontSize: 12),
            ),
          ],
          if (_showAction(job)) ...[
            const SizedBox(height: 12),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton.icon(
                onPressed: loading ? null : onStart,
                icon: loading
                    ? const SizedBox.square(
                        dimension: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.rocket_launch_outlined, size: 18),
                label: Text(
                  CodeWorkspaceText.t(
                    context,
                    job?.canRetry == true ? 'delivery.retry' : 'delivery.start',
                  ),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  bool _showAction(CodeDeliveryJob? job) {
    if (!_managed || session.status == 'delivered') return false;
    return job == null || job.canRetry;
  }

  String _description(BuildContext context, CodeDeliveryJob? job) {
    if (!_managed) return CodeWorkspaceText.t(context, 'delivery.unavailable');
    if (job == null) return CodeWorkspaceText.t(context, 'delivery.empty');
    if (job.status == 'conflict' || job.status == 'partial') {
      return CodeWorkspaceText.t(context, 'delivery.conflictHint');
    }
    if (job.status == 'failed') {
      return job.errorMessage.isEmpty
          ? CodeWorkspaceText.t(context, 'delivery.failed')
          : job.errorMessage;
    }
    if (job.isCompleted) {
      final commit = job.resultCommit.length > 10
          ? job.resultCommit.substring(0, 10)
          : job.resultCommit;
      final suffix = commit.isEmpty ? '' : ' · $commit';
      return '${CodeWorkspaceText.t(context, 'delivery.completed')}$suffix';
    }
    return _stageLabel(context, job.stage);
  }

  String _progressMeta(BuildContext context, CodeDeliveryJob job) {
    final parts = <String>['${job.progress}%'];
    if (job.queuePosition > 0) {
      parts.add(
        CodeWorkspaceText.format(context, 'delivery.queuePosition', {
          'position': job.queuePosition,
        }),
      );
    }
    if (job.attempt > 0) {
      parts.add(
        CodeWorkspaceText.format(context, 'delivery.attempt', {
          'attempt': job.attempt,
        }),
      );
    }
    return parts.join(' · ');
  }

  Color _accent(CodeDeliveryJob? job) {
    if (job?.isCompleted == true) return Colors.greenAccent;
    if (job?.status == 'failed') return Colors.redAccent;
    if (job?.status == 'conflict' || job?.status == 'partial') {
      return Colors.orangeAccent;
    }
    return Colors.lightBlueAccent;
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.job, required this.color});

  final CodeDeliveryJob job;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final key = switch (job.status) {
      'completed' => 'delivery.completed',
      'failed' => 'delivery.failed',
      'conflict' || 'partial' => 'delivery.conflict',
      'queued' => 'delivery.queued',
      _ => 'delivery.running',
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(99),
      ),
      child: Text(
        CodeWorkspaceText.t(context, key),
        style: TextStyle(color: color, fontSize: 11),
      ),
    );
  }
}

String _stageLabel(BuildContext context, String stage) {
  const keys = <String, String>{
    'queued': 'delivery.queued',
    'stopping_terminal': 'delivery.stageStoppingTerminal',
    'syncing': 'delivery.stageSyncing',
    'merging': 'delivery.stageMerging',
    'quality_check': 'delivery.stageQualityCheck',
    'pushing': 'delivery.stagePushing',
    'verifying': 'delivery.stageVerifying',
    'cleaning': 'delivery.stageCleaning',
  };
  return CodeWorkspaceText.t(context, keys[stage] ?? 'delivery.running');
}
