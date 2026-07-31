import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../ai_workspace/presentation/controllers/ai_approval_controller.dart';
import '../controllers/dashboard_controller.dart';
import '../../models/system_info.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../widgets/dashboard_base_card.dart';
import '../widgets/dashboard_ai_summary_card.dart';
import '../widgets/dashboard_disk_card.dart';
import '../widgets/dashboard_metric_card.dart';
import '../widgets/dashboard_monitor_card.dart';

/// 真正的 Dashboard (类似 GoPanel 的 StatusCard 概览)
/// 作为 MainScaffoldScreen 的第一个 Tab 内容
class DashboardScreen extends ConsumerStatefulWidget {
  const DashboardScreen({super.key});

  @override
  ConsumerState<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends ConsumerState<DashboardScreen> {
  bool _diskExpanded = false;
  MonitorTab _monitorTab = MonitorTab.traffic;

  @override
  Widget build(BuildContext context) {
    final dashboardState = ref.watch(dashboardControllerProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('服务器概览'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(dashboardControllerProvider.notifier).refresh();
              ref.read(taskCenterControllerProvider.notifier).refresh();
              ref.invalidate(pendingAiApprovalCountProvider);
            },
          ),
        ],
      ),
      body: _buildBody(context, dashboardState),
    );
  }

  Widget _buildBody(BuildContext context, DashboardState state) {
    if (state.isLoading &&
        (state.osInfo == null || state.currentInfo == null)) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.errorMessage != null &&
        (state.osInfo == null || state.currentInfo == null)) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 64, color: Colors.red),
            const SizedBox(height: 16),
            Text(
              state.errorMessage!,
              style: const TextStyle(color: Colors.red),
            ),
          ],
        ),
      );
    }

    if (state.osInfo == null || state.currentInfo == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final osInfo = state.osInfo!;
    final currentInfo = state.currentInfo!;

    final memTotalGb = (currentInfo.memoryTotal / 1024 / 1024 / 1024)
        .toStringAsFixed(1);
    final memAvailGb = (currentInfo.memoryAvailable / 1024 / 1024 / 1024)
        .toStringAsFixed(1);
    final memUsedGb = (currentInfo.memoryUsed / 1024 / 1024 / 1024)
        .toStringAsFixed(1);
    final memPercent = currentInfo.memoryTotal > 0
        ? currentInfo.memoryUsed / currentInfo.memoryTotal
        : 0.0;
    final loadPercent = currentInfo.loadUsagePercent > 0
        ? (currentInfo.loadUsagePercent / 100.0).clamp(0.0, 1.0)
        : 0.0;

    final disks = _sortDisksForDisplay(currentInfo.diskData);
    final monitor = state.monitor;

    return RefreshIndicator(
      onRefresh: () async {
        // 下拉手动触发刷新
        await Future.wait([
          ref.read(dashboardControllerProvider.notifier).refresh(),
          ref.read(taskCenterControllerProvider.notifier).refresh(),
        ]);
        ref.invalidate(pendingAiApprovalCountProvider);
      },
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        physics: const AlwaysScrollableScrollPhysics(),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            DashboardBaseCard(osInfo: osInfo, currentInfo: currentInfo),
            const SizedBox(height: 16),
            const DashboardAiSummaryCard(),
            const SizedBox(height: 16),
            LayoutBuilder(
              builder: (context, constraints) {
                final wide = constraints.maxWidth >= 520;

                final cpuCard = DashboardMetricCard(
                  title: 'CPU',
                  valueText: '${currentInfo.percent.toStringAsFixed(0)}%',
                  percent: (currentInfo.percent / 100.0).clamp(0.0, 1.0),
                  subtitle: '总使用率',
                );
                final memCard = DashboardMetricCard(
                  title: '内存',
                  valueText: '$memUsedGb GB',
                  percent: memPercent,
                  subtitle: '总数 $memTotalGb GB  可用 $memAvailGb GB',
                );
                final loadCard = DashboardMetricCard(
                  title: '负载',
                  valueText: currentInfo.loadUsagePercent > 0
                      ? '${currentInfo.loadUsagePercent.toStringAsFixed(1)}%'
                      : '-',
                  percent: loadPercent,
                  subtitle:
                      '1m ${currentInfo.load1.toStringAsFixed(2)}  5m ${currentInfo.load5.toStringAsFixed(2)}  15m ${currentInfo.load15.toStringAsFixed(2)}',
                );

                if (wide) {
                  return Row(
                    children: [
                      Expanded(child: cpuCard),
                      const SizedBox(width: 12),
                      Expanded(child: memCard),
                      const SizedBox(width: 12),
                      Expanded(child: loadCard),
                    ],
                  );
                }

                return Column(
                  children: [
                    cpuCard,
                    const SizedBox(height: 12),
                    memCard,
                    const SizedBox(height: 12),
                    loadCard,
                  ],
                );
              },
            ),
            const SizedBox(height: 16),
            DashboardDiskCard(
              disks: disks,
              expanded: _diskExpanded,
              onToggleExpanded: disks.length > 3
                  ? () {
                      setState(() {
                        _diskExpanded = !_diskExpanded;
                      });
                    }
                  : null,
            ),
            const SizedBox(height: 16),
            DashboardMonitorCard(
              monitor: monitor,
              tab: _monitorTab,
              onTabChanged: (t) {
                setState(() {
                  _monitorTab = t;
                });
              },
            ),
          ],
        ),
      ),
    );
  }

  List<DiskInfo> _sortDisksForDisplay(List<DiskInfo> disks) {
    if (disks.isEmpty) return const [];
    DiskInfo? systemDisk;
    for (final d in disks) {
      if (d.path == '/') {
        systemDisk = d;
        break;
      }
    }
    systemDisk ??= (List<DiskInfo>.from(
      disks,
    )..sort((a, b) => a.path.length.compareTo(b.path.length))).first;

    final rest = disks.where((d) => d.path != systemDisk!.path).toList()
      ..sort((a, b) => a.path.compareTo(b.path));
    return [systemDisk, ...rest];
  }
}
