import 'package:flutter/material.dart';

import '../../models/system_info.dart';
import '../../../../shared/widgets/panel/info_tile.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class DashboardBaseCard extends StatelessWidget {
  final OsInfo osInfo;
  final SystemCurrentInfo currentInfo;

  const DashboardBaseCard({
    super.key,
    required this.osInfo,
    required this.currentInfo,
  });

  @override
  Widget build(BuildContext context) {
    final bootAt = DateTime.now().subtract(
      Duration(seconds: currentInfo.uptime),
    );
    final uptimeText = _formatDuration(Duration(seconds: currentInfo.uptime));
    final bootAtText = _formatDateTime(bootAt);

    return PanelCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            osInfo.hostname,
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 6),
          Text(
            '${osInfo.platform} · ${osInfo.version}'.trim(),
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: Color(0xFF64748B),
            ),
          ),
          const SizedBox(height: 16),
          LayoutBuilder(
            builder: (context, constraints) {
              final wide = constraints.maxWidth >= 480;
              final tiles = [
                InfoTile(
                  label: '启动时间',
                  value: bootAtText,
                  leading: const Icon(Icons.schedule_rounded),
                ),
                InfoTile(
                  label: '运行时间',
                  value: uptimeText,
                  leading: const Icon(Icons.timer_outlined),
                ),
                InfoTile(
                  label: '进程',
                  value: currentInfo.procs > 0 ? '${currentInfo.procs}' : '-',
                  leading: const Icon(Icons.tune_rounded),
                ),
                InfoTile(
                  label: '内核版本',
                  value: osInfo.kernelVersion.isEmpty
                      ? '-'
                      : osInfo.kernelVersion,
                  leading: const Icon(Icons.memory_rounded),
                ),
              ];

              if (wide) {
                return Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: tiles
                      .map(
                        (t) => ConstrainedBox(
                          constraints: const BoxConstraints(minWidth: 200),
                          child: t,
                        ),
                      )
                      .toList(),
                );
              }

              return Column(
                children: [
                  for (int i = 0; i < tiles.length; i++) ...[
                    SizedBox(width: double.infinity, child: tiles[i]),
                    if (i != tiles.length - 1) const SizedBox(height: 12),
                  ],
                ],
              );
            },
          ),
        ],
      ),
    );
  }

  String _formatDuration(Duration d) {
    final days = d.inDays;
    final hours = d.inHours.remainder(24);
    final minutes = d.inMinutes.remainder(60);
    final seconds = d.inSeconds.remainder(60);
    if (days > 0) return '$days天$hours小时$minutes分';
    if (hours > 0) return '$hours小时$minutes分';
    if (minutes > 0) return '$minutes分$seconds秒';
    return '$seconds秒';
  }

  String _formatDateTime(DateTime t) {
    String p2(int v) => v.toString().padLeft(2, '0');
    return '${t.year}-${p2(t.month)}-${p2(t.day)} ${p2(t.hour)}:${p2(t.minute)}:${p2(t.second)}';
  }
}
