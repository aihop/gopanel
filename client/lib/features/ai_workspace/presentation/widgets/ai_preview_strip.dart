import 'package:flutter/material.dart';

import '../../models/ai_dev_session.dart';

class AiPreviewStrip extends StatelessWidget {
  const AiPreviewStrip({
    super.key,
    required this.previews,
    required this.onOpenPreview,
    required this.onViewAll,
  });

  final List<AiPreview> previews;
  final ValueChanged<AiPreview> onOpenPreview;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) {
    if (previews.isEmpty) {
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
          Row(
            children: [
              const Text(
                '预览',
                style: TextStyle(
                  color: Colors.white,
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const Spacer(),
              TextButton(onPressed: onViewAll, child: const Text('查看全部')),
            ],
          ),
          const SizedBox(height: 8),
          ...previews
              .take(3)
              .map(
                (preview) => _PreviewItem(
                  preview: preview,
                  onTap: () => onOpenPreview(preview),
                ),
              ),
        ],
      ),
    );
  }
}

class _PreviewItem extends StatelessWidget {
  const _PreviewItem({required this.preview, required this.onTap});

  final AiPreview preview;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(10),
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.05),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              preview.title.isEmpty ? '开发预览' : preview.title,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              '状态: ${preview.status}',
              style: TextStyle(
                color: _statusColor(preview.status).withValues(alpha: 0.9),
                fontSize: 12,
              ),
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Expanded(
                  child: Text(
                    preview.url,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: Colors.lightBlueAccent.shade100,
                      fontSize: 12,
                      height: 1.4,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Icon(
                  Icons.open_in_new_rounded,
                  size: 16,
                  color: Colors.white.withValues(alpha: 0.6),
                ),
              ],
            ),
            if (preview.status == 'local') ...[
              const SizedBox(height: 6),
              Text(
                '本地地址通常需要代理后手机才能直接访问',
                style: TextStyle(
                  color: Colors.orange.withValues(alpha: 0.8),
                  fontSize: 11,
                  height: 1.4,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'ready':
      case 'available':
        return Colors.greenAccent;
      case 'local':
        return Colors.orangeAccent;
      case 'starting':
        return Colors.amberAccent;
      default:
        return Colors.lightBlueAccent;
    }
  }
}
