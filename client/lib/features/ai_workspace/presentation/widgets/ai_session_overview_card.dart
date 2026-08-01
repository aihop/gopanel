import 'package:flutter/material.dart';

import '../../models/ai_dev_session.dart';

class AiSessionOverviewCard extends StatelessWidget {
  const AiSessionOverviewCard({
    super.key,
    required this.workspace,
    required this.currentSession,
    required this.currentTask,
    required this.stage,
  });

  final String? workspace;
  final AiDevSession? currentSession;
  final AiTaskSummary? currentTask;
  final String stage;

  @override
  Widget build(BuildContext context) {
    final activeStage = stage.isEmpty
        ? (currentSession?.currentStage.isEmpty ?? true
              ? 'idle'
              : currentSession!.currentStage)
        : stage;

    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF111827),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(
                Icons.terminal_rounded,
                color: Colors.greenAccent,
                size: 18,
              ),
              const SizedBox(width: 8),
              const Expanded(
                child: Text(
                  '开发会话',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              _StatusChip(stage: activeStage),
            ],
          ),
          const SizedBox(height: 12),
          _InfoRow(
            label: '工作区',
            value: workspace == null || workspace!.isEmpty
                ? '尚未选择'
                : workspace!,
            valueColor: workspace == null ? Colors.orangeAccent : Colors.white,
          ),
          _InfoRow(
            label: '会话',
            value: currentSession == null
                ? '尚未创建'
                : '#${currentSession!.id} · ${currentSession!.title}',
          ),
          _InfoRow(
            label: '任务',
            value: currentTask == null
                ? '等待首条指令'
                : '#${currentTask!.id} · ${currentTask!.title}',
          ),
          if ((currentSession?.agentName.isNotEmpty ?? false))
            _InfoRow(label: 'Agent', value: currentSession!.agentName),
        ],
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({
    required this.label,
    required this.value,
    this.valueColor = Colors.white70,
  });

  final String label;
  final String value;
  final Color valueColor;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 48,
            child: Text(
              label,
              style: TextStyle(
                color: Colors.white.withValues(alpha: 0.45),
                fontSize: 12,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              value,
              style: TextStyle(color: valueColor, fontSize: 12, height: 1.5),
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.stage});

  final String stage;

  @override
  Widget build(BuildContext context) {
    final style = _styleForStage(stage);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: style.$1.withValues(alpha: 0.16),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: style.$1.withValues(alpha: 0.24)),
      ),
      child: Text(
        style.$2,
        style: TextStyle(
          color: style.$1,
          fontSize: 11,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  (Color, String) _styleForStage(String value) {
    switch (value) {
      case 'preview_ready':
        return (Colors.lightBlueAccent, '预览就绪');
      case 'completed':
        return (Colors.greenAccent, '已完成');
      case 'failed':
        return (Colors.redAccent, '失败');
      case 'executing':
      case 'running':
        return (Colors.orangeAccent, '执行中');
      case 'instruction_queued':
      case 'queued':
      case 'task_ready':
        return (Colors.amberAccent, '排队中');
      default:
        return (Colors.white70, value.isEmpty ? '空闲' : value);
    }
  }
}
