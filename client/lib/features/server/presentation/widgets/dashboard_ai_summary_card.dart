import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_status.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';

class DashboardAiSummaryCard extends ConsumerWidget {
  const DashboardAiSummaryCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(taskCenterControllerProvider);
    final allTasks = [...state.localTasks, ...state.tasks];
    final aiTasks = allTasks.where((task) => task.type == TaskType.ai).toList()
      ..sort((a, b) {
        final at =
            a.updatedAt ?? a.startedAt ?? DateTime.fromMillisecondsSinceEpoch(0);
        final bt =
            b.updatedAt ?? b.startedAt ?? DateTime.fromMillisecondsSinceEpoch(0);
        return bt.compareTo(at);
      });

    final runningTask = aiTasks.cast<TaskEntity?>().firstWhere(
          (task) => task?.status == TaskStatus.running,
          orElse: () => null,
        );
    final previewTask = aiTasks.cast<TaskEntity?>().firstWhere(
          (task) => int.tryParse(task?.meta['previewCount'] ?? '0') != null &&
              int.parse(task!.meta['previewCount'] ?? '0') > 0,
          orElse: () => null,
        );
    final latestEventTask = aiTasks.cast<TaskEntity?>().firstWhere(
          (task) => (task?.meta['latestEventTitle'] ?? '').isNotEmpty,
          orElse: () => null,
        );
    final failedTask = aiTasks.cast<TaskEntity?>().firstWhere(
          (task) => task?.status == TaskStatus.failed,
          orElse: () => null,
        );

    return PanelCard(
      title: const Text('AI 开发'),
      trailing: TextButton(
        onPressed: () {
          ref
              .read(mainScaffoldIndexProvider.notifier)
              .setIndex(MainScaffoldIndexController.codeIndex);
        },
        child: const Text('开发'),
      ),
      child: Column(
        children: [
          _buildSummaryRow(
            context,
            label: '进行中',
            task: runningTask,
            emptyText: state.isLoading ? '加载中...' : '暂无进行中任务',
            color: const Color(0xFF2563EB),
          ),
          const SizedBox(height: 12),
          _buildSummaryRow(
            context,
            label: '最近过程',
            task: latestEventTask,
            emptyText: '暂无过程事件',
            color: const Color(0xFF7C3AED),
            summaryOverride: latestEventTask == null
                ? null
                : latestEventTask.meta['latestEventTitle'],
          ),
          const SizedBox(height: 12),
          _buildSummaryRow(
            context,
            label: '最近预览',
            task: previewTask,
            emptyText: '暂无可用预览',
            color: const Color(0xFF059669),
            suffixText: previewTask == null
                ? null
                : '${previewTask.meta['previewCount'] ?? '0'} 个预览',
          ),
          const SizedBox(height: 12),
          _buildSummaryRow(
            context,
            label: '最近失败',
            task: failedTask,
            emptyText: '暂无失败任务',
            color: const Color(0xFFDC2626),
          ),
        ],
      ),
    );
  }

  Widget _buildSummaryRow(
    BuildContext context, {
    required String label,
    required TaskEntity? task,
    required String emptyText,
    required Color color,
    String? suffixText,
    String? summaryOverride,
  }) {
    final content = task == null
        ? Text(
            emptyText,
            style: const TextStyle(
              color: Color(0xFF64748B),
              fontWeight: FontWeight.w500,
            ),
          )
        : InkWell(
            onTap: () {
              Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)),
              );
            },
            borderRadius: BorderRadius.circular(12),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 2),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    task.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    summaryOverride ??
                        task.meta['errorSummary'] ??
                        task.summary ??
                        task.meta['currentStageLabel'] ??
                        '',
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      color: Color(0xFF475569),
                      fontSize: 13,
                      height: 1.45,
                    ),
                  ),
                  if ((task.meta['workDir'] ?? '').isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      task.meta['workDir']!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Color(0xFF64748B),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          );

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 10,
          height: 10,
          margin: const EdgeInsets.only(top: 6),
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(999),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text(
                    label,
                    style: TextStyle(
                      color: color,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (suffixText != null && suffixText.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    Text(
                      suffixText,
                      style: const TextStyle(
                        color: Color(0xFF64748B),
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ],
              ),
              const SizedBox(height: 6),
              content,
            ],
          ),
        ),
      ],
    );
  }
}
