import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../ai_workspace/models/ai_dev_session.dart';
import '../../../ai_workspace/models/ai_session_state_info.dart';
import '../../../ai_workspace/presentation/screens/ai_preview_detail_screen.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class TaskAiSessionSummaryCard extends StatelessWidget {
  const TaskAiSessionSummaryCard({
    super.key,
    required this.state,
  });

  final AiSessionStateInfo state;

  @override
  Widget build(BuildContext context) {
    final latestEvent = state.timelineEvents.isNotEmpty
        ? state.timelineEvents.first
        : null;

    return PanelCard(
      title: const Text('开发摘要'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _kv('阶段', state.currentStage.isEmpty ? '-' : state.currentStage),
          if (state.currentTask != null) ...[
            const SizedBox(height: 8),
            _kv('任务', '#${state.currentTask!.id} · ${state.currentTask!.title}'),
          ],
          if (latestEvent != null) ...[
            const SizedBox(height: 12),
            _SectionBlock(
              label: '最新事件',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    latestEvent.title.isEmpty ? latestEvent.eventType : latestEvent.title,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  if (latestEvent.content.trim().isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      latestEvent.content.trim(),
                      style: const TextStyle(
                        color: AppTheme.textSecondary,
                        fontSize: 13,
                        height: 1.45,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
          if (state.errorSummary.trim().isNotEmpty) ...[
            const SizedBox(height: 12),
            _SectionBlock(
              label: '错误摘要',
              child: Text(
                state.errorSummary.trim(),
                style: const TextStyle(
                  color: AppTheme.error,
                  fontWeight: FontWeight.w700,
                  height: 1.45,
                ),
              ),
            ),
          ],
          if (state.changedFiles.isNotEmpty) ...[
            const SizedBox(height: 12),
            _SectionBlock(
              label: '涉及文件',
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: state.changedFiles
                    .take(8)
                    .map(
                      (file) => Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 6,
                        ),
                        decoration: BoxDecoration(
                          color: const Color(0xFFF1F5F9),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          file,
                          style: const TextStyle(fontSize: 12),
                        ),
                      ),
                    )
                    .toList(),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class TaskAiPreviewCard extends StatelessWidget {
  const TaskAiPreviewCard({
    super.key,
    required this.previews,
  });

  final List<AiPreview> previews;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: const Text('预览'),
      trailing: Text('${previews.length} 项'),
      child: Column(
        children: previews
            .map(
              (preview) => Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: _PreviewItem(preview: preview),
              ),
            )
            .toList(),
      ),
    );
  }
}

class _SectionBlock extends StatelessWidget {
  const _SectionBlock({
    required this.label,
    required this.child,
  });

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(
            color: AppTheme.textSecondary,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 6),
        child,
      ],
    );
  }
}

class _PreviewItem extends StatelessWidget {
  const _PreviewItem({required this.preview});

  final AiPreview preview;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => AiPreviewDetailScreen(preview: preview),
          ),
        );
      },
      borderRadius: BorderRadius.circular(12),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: const Color(0xFFF8FAFC),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: const Color(0xFFE2E8F0)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    preview.title.isEmpty ? '开发预览' : preview.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                ),
                const SizedBox(width: 12),
                Text(
                  preview.status,
                  style: TextStyle(
                    color: _previewStatusColor(preview.status),
                    fontWeight: FontWeight.w700,
                    fontSize: 12,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              preview.url,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                height: 1.4,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Color _previewStatusColor(String status) {
    switch (status) {
      case 'ready':
        return AppTheme.success;
      case 'local':
        return const Color(0xFFD97706);
      case 'invalid':
      case 'unreachable':
      case 'failed':
        return AppTheme.error;
      default:
        return AppTheme.primaryBlue;
    }
  }
}

Widget _kv(String k, String v) {
  return Row(
    mainAxisAlignment: MainAxisAlignment.spaceBetween,
    children: [
      Text(
        k,
        style: const TextStyle(
          color: AppTheme.textSecondary,
          fontWeight: FontWeight.w600,
        ),
      ),
      const SizedBox(width: 12),
      Expanded(
        child: Text(
          v,
          textAlign: TextAlign.right,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontWeight: FontWeight.w700),
        ),
      ),
    ],
  );
}
