import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../ai_workspace/presentation/controllers/ai_approval_controller.dart';
import '../../../task_center/models/task_attention.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_center_screen.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';

class DashboardTaskCenterButton extends ConsumerWidget {
  const DashboardTaskCenterButton({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref.watch(
      taskCenterControllerProvider.select(
        (state) => state.attentionTasks.length,
      ),
    );
    return IconButton(
      tooltip: '任务中心',
      onPressed: () {
        ref.read(taskCenterControllerProvider.notifier).showAll();
        Navigator.of(
          context,
        ).push(MaterialPageRoute(builder: (_) => const TaskCenterScreen()));
      },
      icon: Badge.count(
        count: count,
        isLabelVisible: count > 0,
        backgroundColor: AppTheme.error,
        child: const Icon(Icons.task_alt_rounded),
      ),
    );
  }
}

class DashboardAttentionCard extends ConsumerStatefulWidget {
  const DashboardAttentionCard({super.key});

  @override
  ConsumerState<DashboardAttentionCard> createState() =>
      _DashboardAttentionCardState();
}

class _DashboardAttentionCardState
    extends ConsumerState<DashboardAttentionCard> {
  String? _actionKey;

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(taskCenterControllerProvider);
    final items = state.attentionTasks.take(3).toList();

    return PanelCard(
      title: Row(
        children: [
          const Icon(
            Icons.notification_important_outlined,
            color: AppTheme.error,
            size: 21,
          ),
          const SizedBox(width: 8),
          const Text('待我处理'),
          if (state.attentionTasks.isNotEmpty) ...[
            const SizedBox(width: 8),
            _countBadge(state.attentionTasks.length),
          ],
        ],
      ),
      trailing: TextButton(
        onPressed: state.isLoading && state.tasks.isEmpty
            ? null
            : () => _openTaskCenter(attentionOnly: true),
        child: const Text('查看全部'),
      ),
      child: _buildBody(state, items),
    );
  }

  Widget _buildBody(TaskCenterState state, List<TaskEntity> items) {
    if (state.isLoading && state.tasks.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 12),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (state.errorMessage != null && state.tasks.isEmpty) {
      return _messageBody(
        icon: Icons.cloud_off_outlined,
        message: '待处理事项加载失败',
        actionLabel: '重试',
        onAction: ref.read(taskCenterControllerProvider.notifier).refresh,
      );
    }
    if (items.isEmpty) {
      return _messageBody(
        icon: Icons.check_circle_outline_rounded,
        message: '当前没有需要你处理的事项',
      );
    }

    return Column(
      children: [
        for (var index = 0; index < items.length; index++) ...[
          _attentionItem(items[index]),
          if (index != items.length - 1) const Divider(height: 24),
        ],
      ],
    );
  }

  Widget _attentionItem(TaskEntity task) {
    final attention = task.attention!;
    return InkWell(
      onTap: () => _openTask(task),
      borderRadius: BorderRadius.circular(14),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    attention.title.isEmpty ? task.title : attention.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontWeight: FontWeight.w800),
                  ),
                ),
                const SizedBox(width: 8),
                const Icon(
                  Icons.chevron_right_rounded,
                  color: AppTheme.textLight,
                ),
              ],
            ),
            if (attention.summary.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(
                attention.summary,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  height: 1.4,
                ),
              ),
            ],
            if (attention.actions.isNotEmpty) ...[
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: attention.actions
                    .take(3)
                    .map((action) => _actionButton(task, action))
                    .toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _actionButton(TaskEntity task, TaskAttentionAction action) {
    final key = '${task.attention!.id}:${action.type}';
    final loading = _actionKey == key;
    return OutlinedButton.icon(
      onPressed: _actionKey == null
          ? () => _handleAction(task, action, key)
          : null,
      icon: loading
          ? const SizedBox.square(
              dimension: 14,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : Icon(_actionIcon(action.type), size: 17),
      label: Text(action.label),
      style: OutlinedButton.styleFrom(
        foregroundColor: action.type == 'reject'
            ? AppTheme.error
            : AppTheme.primaryBlue,
        visualDensity: VisualDensity.compact,
      ),
    );
  }

  Future<void> _handleAction(
    TaskEntity task,
    TaskAttentionAction action,
    String key,
  ) async {
    if (action.type == 'open_session') {
      await _openTask(task);
      return;
    }
    if (action.requiresConfirmation) {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(action.label),
          content: Text('确认${action.label}？该操作会立即影响当前开发会话。'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('取消'),
            ),
            ElevatedButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: const Text('确认'),
            ),
          ],
        ),
      );
      if (confirmed != true || !mounted) return;
    }

    setState(() => _actionKey = key);
    try {
      await ref.read(taskRepositoryProvider).executeAttentionAction(action);
      await ref.read(taskCenterControllerProvider.notifier).refresh();
      ref.invalidate(pendingAiApprovalCountProvider);
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('${action.label}成功')));
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('${action.label}失败：$error')));
    } finally {
      if (mounted) setState(() => _actionKey = null);
    }
  }

  Future<void> _openTask(TaskEntity task) async {
    await Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)));
    if (!mounted) return;
    await ref.read(taskCenterControllerProvider.notifier).refresh();
  }

  Future<void> _openTaskCenter({required bool attentionOnly}) async {
    if (attentionOnly) {
      ref.read(taskCenterControllerProvider.notifier).showAttentionOnly();
    }
    await Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => const TaskCenterScreen()));
    if (!mounted) return;
    await ref.read(taskCenterControllerProvider.notifier).refresh();
  }

  Widget _messageBody({
    required IconData icon,
    required String message,
    String? actionLabel,
    VoidCallback? onAction,
  }) {
    return Row(
      children: [
        Icon(icon, color: AppTheme.textLight),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            message,
            style: const TextStyle(
              color: AppTheme.textSecondary,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        if (actionLabel != null && onAction != null)
          TextButton(onPressed: onAction, child: Text(actionLabel)),
      ],
    );
  }

  Widget _countBadge(int count) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF1F2),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        '$count',
        style: const TextStyle(
          color: AppTheme.error,
          fontSize: 12,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }

  IconData _actionIcon(String type) {
    switch (type) {
      case 'approve':
        return Icons.check_rounded;
      case 'reject':
        return Icons.close_rounded;
      case 'retry_initialization':
        return Icons.refresh_rounded;
      default:
        return Icons.open_in_new_rounded;
    }
  }
}
