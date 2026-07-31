import 'package:flutter/material.dart';

import '../../models/ai_dev_session.dart';

class AiTimelinePanel extends StatelessWidget {
  const AiTimelinePanel({
    super.key,
    required this.timelineEvents,
    required this.errorSummary,
    required this.changedFiles,
  });

  final List<AiTimelineEvent> timelineEvents;
  final String errorSummary;
  final List<String> changedFiles;

  @override
  Widget build(BuildContext context) {
    final hasSummary =
        errorSummary.trim().isNotEmpty || changedFiles.isNotEmpty;
    final hasTimeline = timelineEvents.isNotEmpty;
    if (!hasSummary && !hasTimeline) {
      return const SizedBox.shrink();
    }

    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF111827),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '过程摘要',
            style: TextStyle(
              color: Colors.white,
              fontSize: 14,
              fontWeight: FontWeight.w700,
            ),
          ),
          if (errorSummary.trim().isNotEmpty) ...[
            const SizedBox(height: 12),
            _SummaryBlock(
              icon: Icons.error_outline_rounded,
              iconColor: Colors.redAccent,
              title: '错误摘要',
              child: Text(
                errorSummary.trim(),
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 12,
                  height: 1.5,
                ),
              ),
            ),
          ],
          if (changedFiles.isNotEmpty) ...[
            const SizedBox(height: 12),
            _SummaryBlock(
              icon: Icons.edit_note_rounded,
              iconColor: Colors.lightBlueAccent,
              title: '涉及文件',
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: changedFiles
                    .take(6)
                    .map(
                      (file) => Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 6,
                        ),
                        decoration: BoxDecoration(
                          color: Colors.white.withValues(alpha: 0.05),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          file,
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.85),
                            fontSize: 11,
                          ),
                        ),
                      ),
                    )
                    .toList(),
              ),
            ),
          ],
          if (timelineEvents.isNotEmpty) ...[
            const SizedBox(height: 12),
            const Text(
              '时间线',
              style: TextStyle(
                color: Colors.white70,
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 10),
            ...timelineEvents
                .take(4)
                .map(
                  (event) => Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: _TimelineItem(event: event),
                  ),
                ),
          ],
        ],
      ),
    );
  }
}

class _SummaryBlock extends StatelessWidget {
  const _SummaryBlock({
    required this.icon,
    required this.iconColor,
    required this.title,
    required this.child,
  });

  final IconData icon;
  final Color iconColor;
  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.04),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: iconColor, size: 16),
              const SizedBox(width: 8),
              Text(
                title,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          child,
        ],
      ),
    );
  }
}

class _TimelineItem extends StatelessWidget {
  const _TimelineItem({required this.event});

  final AiTimelineEvent event;

  @override
  Widget build(BuildContext context) {
    final style = _styleForStatus(event.status);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 10,
          height: 10,
          margin: const EdgeInsets.only(top: 4),
          decoration: BoxDecoration(
            color: style.$1,
            borderRadius: BorderRadius.circular(999),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.04),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        event.title.isEmpty ? '过程事件' : event.title,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    Text(
                      style.$2,
                      style: TextStyle(
                        color: style.$1,
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
                if (event.content.trim().isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(
                    event.content.trim(),
                    style: TextStyle(
                      color: Colors.white.withValues(alpha: 0.78),
                      fontSize: 12,
                      height: 1.45,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }

  (Color, String) _styleForStatus(String value) {
    switch (value) {
      case 'success':
        return (Colors.greenAccent, '完成');
      case 'error':
        return (Colors.redAccent, '异常');
      case 'running':
        return (Colors.orangeAccent, '进行中');
      default:
        return (Colors.lightBlueAccent, '记录');
    }
  }
}
