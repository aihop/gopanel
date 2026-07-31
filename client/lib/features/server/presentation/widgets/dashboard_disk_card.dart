import 'package:flutter/material.dart';

import '../../models/system_info.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../../shared/widgets/panel/panel_progress_row.dart';

class DashboardDiskCard extends StatelessWidget {
  final List<DiskInfo> disks;
  final bool expanded;
  final VoidCallback? onToggleExpanded;

  const DashboardDiskCard({
    super.key,
    required this.disks,
    required this.expanded,
    this.onToggleExpanded,
  });

  @override
  Widget build(BuildContext context) {
    final canToggle = disks.length > 3 && onToggleExpanded != null;
    final list = expanded || disks.length <= 3 ? disks : disks.take(3).toList();

    return PanelCard(
      title: const Text('磁盘'),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '${disks.length} 块设备',
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: Color(0xFF64748B),
            ),
          ),
          if (canToggle) ...[
            const SizedBox(width: 8),
            TextButton(
              onPressed: onToggleExpanded,
              child: Text(expanded ? '收起' : '展开'),
            ),
          ],
        ],
      ),
      child: Column(
        children: [
          if (disks.isEmpty)
            const Align(
              alignment: Alignment.centerLeft,
              child: Text(
                '暂无磁盘数据',
                style: TextStyle(color: Color(0xFF64748B)),
              ),
            )
          else
            for (int i = 0; i < list.length; i++) ...[
              _diskRow(list[i]),
              if (i != list.length - 1) const SizedBox(height: 14),
            ],
        ],
      ),
    );
  }

  Widget _diskRow(DiskInfo d) {
    final percent = (d.usedPercent / 100.0).clamp(0.0, 1.0);
    final usedGb = (d.used / 1024 / 1024 / 1024).toStringAsFixed(1);
    final totalGb = (d.total / 1024 / 1024 / 1024).toStringAsFixed(1);
    final text = '${d.usedPercent.toStringAsFixed(1)}% ($usedGb/$totalGb GB)';
    return PanelProgressRow(label: d.path, percent: percent, valueText: text);
  }
}

