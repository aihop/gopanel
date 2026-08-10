import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/task_entity.dart';
import '../../models/task_status.dart';
import '../../models/task_type.dart';
import '../controllers/task_center_controller.dart';
import 'task_detail_screen.dart';
import '../../../../shared/widgets/panel/glass_tabs.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class TaskCenterScreen extends ConsumerWidget {
  const TaskCenterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(taskCenterControllerProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('任务')),
      body: _buildBody(context, ref, state),
    );
  }

  Widget _buildBody(
    BuildContext context,
    WidgetRef ref,
    TaskCenterState state,
  ) {
    if (state.isLoading && state.tasks.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.errorMessage != null && state.tasks.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 56, color: AppTheme.error),
              const SizedBox(height: 12),
              Text(
                state.errorMessage!,
                textAlign: TextAlign.center,
                style: const TextStyle(color: AppTheme.error),
              ),
              const SizedBox(height: 12),
              ElevatedButton(
                onPressed: () {
                  ref.read(taskCenterControllerProvider.notifier).refresh();
                },
                child: const Text('重试'),
              ),
            ],
          ),
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(taskCenterControllerProvider.notifier).refresh();
      },
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          PanelCard(
            title: const Text('筛选'),
            trailing: TextButton.icon(
              onPressed: () {
                ref
                    .read(taskCenterControllerProvider.notifier)
                    .showAttentionOnly();
              },
              icon: Icon(
                Icons.notification_important_outlined,
                color: state.attentionOnly
                    ? AppTheme.error
                    : AppTheme.textSecondary,
              ),
              label: Text(
                '待我处理 ${state.attentionTasks.length}',
                style: TextStyle(
                  color: state.attentionOnly
                      ? AppTheme.error
                      : AppTheme.textSecondary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            child: GlassTabs<TaskStatus?>(
              items: const [
                GlassTabItem(value: null, label: '全部'),
                GlassTabItem(value: TaskStatus.running, label: '运行中'),
                GlassTabItem(value: TaskStatus.failed, label: '失败'),
                GlassTabItem(value: TaskStatus.success, label: '成功'),
              ],
              selected: state.attentionOnly ? null : state.filter,
              onChanged: (v) {
                ref.read(taskCenterControllerProvider.notifier).setFilter(v);
              },
            ),
          ),
          const SizedBox(height: 16),
          if (state.visibleTasks.isEmpty)
            const PanelCard(
              title: Text('任务中心'),
              child: Text(
                '暂无任务',
                style: TextStyle(
                  color: AppTheme.textSecondary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            )
          else
            ...state.visibleTasks.map(
              (t) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _taskCard(context, t),
              ),
            ),
        ],
      ),
    );
  }

  Widget _taskCard(BuildContext context, TaskEntity t) {
    return InkWell(
      onTap: () {
        Navigator.of(
          context,
        ).push(MaterialPageRoute(builder: (_) => TaskDetailScreen(task: t)));
      },
      borderRadius: BorderRadius.circular(20),
      child: PanelCard(
        title: Text(t.title),
        trailing: _statusBadge(t.status),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              t.type.label,
              style: TextStyle(
                color: t.type == TaskType.ai
                    ? AppTheme.primaryBlue
                    : AppTheme.textSecondary,
                fontWeight: FontWeight.w700,
              ),
            ),
            if (t.type == TaskType.ai) ...[
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if ((t.meta['currentStageLabel'] ?? '').isNotEmpty)
                    _metaTag(
                      t.meta['currentStageLabel']!,
                      color: AppTheme.primaryBlue,
                      background: AppTheme.primaryBlueLight,
                    ),
                  if ((t.meta['previewCount'] ?? '0') != '0')
                    _metaTag(
                      '${t.meta['previewCount']} 个预览',
                      color: AppTheme.success,
                      background: const Color(0xFFECFDF5),
                    ),
                  if (t.requiresAttention)
                    _metaTag(
                      '待我处理',
                      color: AppTheme.error,
                      background: const Color(0xFFFFF1F2),
                    ),
                ],
              ),
            ],
            if (t.attention != null) ...[
              const SizedBox(height: 10),
              Text(
                t.attention!.title,
                style: const TextStyle(
                  color: AppTheme.error,
                  fontWeight: FontWeight.w800,
                ),
              ),
              if (t.attention!.summary.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(
                  t.attention!.summary,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ],
            if (t.summary != null && t.summary!.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                t.summary!,
                maxLines: t.type == TaskType.ai ? 2 : null,
                overflow: t.type == TaskType.ai ? TextOverflow.ellipsis : null,
                style: TextStyle(
                  color: t.type == TaskType.ai
                      ? const Color(0xFF0F172A)
                      : AppTheme.textSecondary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
            if (t.type == TaskType.ai &&
                (t.meta['workDir'] ?? '').isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                t.meta['workDir']!,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
            if (t.error != null && t.error!.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                t.error!,
                style: const TextStyle(
                  color: AppTheme.error,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
            if (t.progress != null) ...[
              const SizedBox(height: 12),
              LinearProgressIndicator(
                value: t.progress!.clamp(0.0, 1.0),
                backgroundColor: AppTheme.primaryBlueLight,
                color: _statusColor(t.status),
                minHeight: 6,
                borderRadius: BorderRadius.circular(3),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _metaTag(
    String text, {
    required Color color,
    required Color background,
  }) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.18)),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }

  Widget _statusBadge(TaskStatus s) {
    final bg = s == TaskStatus.running
        ? AppTheme.primaryBlueLight
        : s == TaskStatus.success
        ? const Color(0xFFECFDF5)
        : const Color(0xFFFFF1F2);
    final fg = _statusColor(s);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: fg.withValues(alpha: 0.25)),
      ),
      child: Text(
        s.label,
        style: TextStyle(fontWeight: FontWeight.w800, color: fg, fontSize: 12),
      ),
    );
  }

  Color _statusColor(TaskStatus s) {
    switch (s) {
      case TaskStatus.running:
        return AppTheme.primaryBlue;
      case TaskStatus.success:
        return AppTheme.success;
      case TaskStatus.failed:
        return AppTheme.error;
    }
  }
}
