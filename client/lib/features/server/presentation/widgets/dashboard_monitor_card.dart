import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/monitor_info.dart';
import '../../../../shared/widgets/panel/info_tile.dart';
import '../../../../shared/widgets/panel/glass_tabs.dart';
import '../../../../shared/widgets/panel/mini_line_chart.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

enum MonitorTab { traffic, diskIo }

class DashboardMonitorCard extends StatelessWidget {
  final MonitorSeries monitor;
  final MonitorTab tab;
  final ValueChanged<MonitorTab> onTabChanged;

  const DashboardMonitorCard({
    super.key,
    required this.monitor,
    required this.tab,
    required this.onTabChanged,
  });

  @override
  Widget build(BuildContext context) {
    final isTraffic = tab == MonitorTab.traffic;

    final aTitle = isTraffic ? '上行' : '读取';
    final bTitle = isTraffic ? '下行' : '写入';
    final aNow = _formatRate(isTraffic ? monitor.netUpBps : monitor.ioReadBps);
    final bNow = _formatRate(isTraffic ? monitor.netDownBps : monitor.ioWriteBps);

    final aTotal = isTraffic
        ? _formatBytes(monitor.totalSentBytes)
        : _formatBytes(monitor.totalReadBytes);
    final bTotal = isTraffic
        ? _formatBytes(monitor.totalRecvBytes)
        : _formatBytes(monitor.totalWriteBytes);

    final aSeries = isTraffic ? monitor.netUpSeries : monitor.ioReadSeries;
    final bSeries = isTraffic ? monitor.netDownSeries : monitor.ioWriteSeries;

    return PanelCard(
      title: const Text('监控'),
      trailing: _tabBar(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 10,
            runSpacing: 10,
            children: [
              InfoTile(
                label: '$aTitle: $aNow',
                value: '总${_totalLabel(isTraffic, true)}: $aTotal',
                leading: const Icon(Icons.arrow_upward_rounded),
              ),
              InfoTile(
                label: '$bTitle: $bNow',
                value: '总${_totalLabel(isTraffic, false)}: $bTotal',
                leading: const Icon(Icons.arrow_downward_rounded),
              ),
            ],
          ),
          const SizedBox(height: 14),
          MiniLineChart(seriesA: aSeries, seriesB: bSeries),
          const SizedBox(height: 8),
          Row(
            children: [
              _legendDot(color: AppTheme.primaryBlue),
              Text(
                aTitle,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: AppTheme.textSecondary,
                ),
              ),
              const SizedBox(width: 14),
              _legendDot(color: const Color(0xFF6366F1)),
              Text(
                bTitle,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: AppTheme.textSecondary,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _tabBar() {
    return SizedBox(
      width: 180,
      child: GlassTabs<MonitorTab>(
        items: const [
          GlassTabItem(value: MonitorTab.traffic, label: '流量'),
          GlassTabItem(value: MonitorTab.diskIo, label: '磁盘 IO'),
        ],
        selected: tab,
        onChanged: onTabChanged,
      ),
    );
  }

  Widget _legendDot({required Color color}) {
    return Container(
      width: 8,
      height: 8,
      margin: const EdgeInsets.only(right: 6),
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }

  String _formatBytes(int bytes) {
    final b = bytes.toDouble();
    if (b >= 1024 * 1024 * 1024) {
      return '${(b / 1024 / 1024 / 1024).toStringAsFixed(2)} GB';
    }
    if (b >= 1024 * 1024) {
      return '${(b / 1024 / 1024).toStringAsFixed(2)} MB';
    }
    if (b >= 1024) {
      return '${(b / 1024).toStringAsFixed(2)} KB';
    }
    return '$bytes B';
  }

  String _formatRate(double bps) {
    if (bps <= 0) return '0 B/s';
    final b = bps;
    if (b >= 1024 * 1024) {
      return '${(b / 1024 / 1024).toStringAsFixed(2)} MB/s';
    }
    if (b >= 1024) {
      return '${(b / 1024).toStringAsFixed(2)} KB/s';
    }
    return '${b.toStringAsFixed(0)} B/s';
  }

  String _totalLabel(bool isTraffic, bool a) {
    if (isTraffic) return a ? '发送' : '接收';
    return a ? '读' : '写';
  }
}
