import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../models/ai_dev_session.dart';
import '../code_workspace_text.dart';

class CodeHubHero extends StatelessWidget {
  const CodeHubHero({
    super.key,
    required this.sessionCount,
    required this.activeCount,
    required this.onCreate,
  });

  final int sessionCount;
  final int activeCount;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF172554), Color(0xFF1E3A8A)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(28),
        boxShadow: [
          BoxShadow(
            color: AppTheme.primaryBlue.withValues(alpha: 0.18),
            blurRadius: 28,
            offset: const Offset(0, 12),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 46,
            height: 46,
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(16),
            ),
            child: const Icon(Icons.terminal_rounded, color: Colors.white),
          ),
          const SizedBox(height: 22),
          Text(
            CodeWorkspaceText.t(context, 'hub.heroTitle'),
            style: const TextStyle(
              color: Colors.white,
              fontSize: 22,
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            CodeWorkspaceText.t(context, 'hub.heroDescription'),
            style: TextStyle(
              color: Colors.white.withValues(alpha: 0.72),
              fontSize: 14,
              height: 1.55,
            ),
          ),
          const SizedBox(height: 22),
          Row(
            children: [
              _HeroMetric(
                label: CodeWorkspaceText.t(context, 'hub.all'),
                value: '$sessionCount',
              ),
              const SizedBox(width: 12),
              _HeroMetric(
                label: CodeWorkspaceText.t(context, 'hub.active'),
                value: '$activeCount',
              ),
              const Spacer(),
              FilledButton.icon(
                onPressed: onCreate,
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.white,
                  foregroundColor: AppTheme.primaryBlue,
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 13,
                  ),
                ),
                icon: const Icon(Icons.add_rounded, size: 18),
                label: Text(CodeWorkspaceText.t(context, 'hub.create')),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _HeroMetric extends StatelessWidget {
  const _HeroMetric({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 9),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Text(
            value,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(width: 5),
          Text(
            label,
            style: TextStyle(
              color: Colors.white.withValues(alpha: 0.64),
              fontSize: 11,
            ),
          ),
        ],
      ),
    );
  }
}

class CodeSessionListCard extends StatelessWidget {
  const CodeSessionListCard({
    super.key,
    required this.session,
    required this.projectName,
    required this.onTap,
  });

  final AiDevSession session;
  final String projectName;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final stage = session.currentStage.isEmpty
        ? session.status
        : session.currentStage;
    final title = session.title.isEmpty ? '开发 #${session.id}' : session.title;
    final updatedAt =
        session.lastInstructionAt ?? session.updatedAt ?? session.createdAt;

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(24),
      child: PanelCard(
        title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
        trailing: _StageBadge(stage: stage),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(
                  Icons.folder_outlined,
                  size: 17,
                  color: AppTheme.primaryBlue,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    projectName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      color: AppTheme.primaryBlue,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Text(
              session.workDir.isEmpty ? '未绑定工作目录' : session.workDir,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                fontFamily: 'monospace',
                fontSize: 12,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                const Icon(
                  Icons.smart_toy_outlined,
                  size: 15,
                  color: AppTheme.textSecondary,
                ),
                const SizedBox(width: 5),
                Text(
                  session.agentName.isEmpty ? '开发' : session.agentName,
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontSize: 12,
                  ),
                ),
                const Spacer(),
                Text(
                  _formatTime(updatedAt),
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontSize: 12,
                  ),
                ),
                const SizedBox(width: 2),
                const Icon(Icons.chevron_right_rounded, size: 18),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class CodeHubErrorCard extends StatelessWidget {
  const CodeHubErrorCard({
    super.key,
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return CodeHubEmptyCard(
      icon: Icons.error_outline_rounded,
      title: '加载失败',
      description: message,
      actionLabel: '重试',
      onAction: onRetry,
    );
  }
}

class CodeHubEmptyCard extends StatelessWidget {
  const CodeHubEmptyCard({
    super.key,
    required this.icon,
    required this.title,
    required this.description,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String description;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 36),
        child: Center(
          child: Column(
            children: [
              Icon(icon, size: 46, color: AppTheme.textSecondary),
              const SizedBox(height: 16),
              Text(title, style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 8),
              Text(
                description,
                textAlign: TextAlign.center,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  height: 1.5,
                ),
              ),
              if (actionLabel != null && onAction != null) ...[
                const SizedBox(height: 22),
                FilledButton(onPressed: onAction, child: Text(actionLabel!)),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _StageBadge extends StatelessWidget {
  const _StageBadge({required this.stage});

  final String stage;

  @override
  Widget build(BuildContext context) {
    final normalized = stage.toLowerCase();
    final failed = normalized == 'failed';
    final completed =
        normalized == 'completed' || normalized == 'preview_ready';
    final foreground = failed
        ? AppTheme.error
        : completed
        ? AppTheme.success
        : AppTheme.primaryBlue;
    final background = failed
        ? const Color(0xFFFFF1F2)
        : completed
        ? const Color(0xFFECFDF5)
        : AppTheme.primaryBlueLight;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(99),
      ),
      child: Text(
        _stageLabel(stage),
        style: TextStyle(
          color: foreground,
          fontSize: 11,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

String _stageLabel(String stage) => switch (stage) {
  'instruction_queued' => '排队中',
  'awaiting_approval' => '待审批',
  'executing' => '执行中',
  'running' => '运行中',
  'completed' => '已完成',
  'preview_ready' => '预览就绪',
  'failed' => '失败',
  'idle' => '空闲',
  _ => stage.isEmpty ? '会话中' : stage,
};

String _formatTime(DateTime? value) {
  if (value == null) return '-';
  final difference = DateTime.now().difference(value);
  if (difference.inMinutes < 1) return '刚刚';
  if (difference.inHours < 1) return '${difference.inMinutes} 分钟前';
  if (difference.inDays < 1) return '${difference.inHours} 小时前';
  if (difference.inDays < 7) return '${difference.inDays} 天前';
  return '${value.month}/${value.day}';
}
