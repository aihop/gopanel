import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_status.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';
import '../../models/website_info.dart';
import '../controllers/website_controller.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class WebsiteDetailScreen extends ConsumerWidget {
  final WebsiteInfo website;

  const WebsiteDetailScreen({super.key, required this.website});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('网站详情')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            PanelCard(
              title: Text(website.alias.isNotEmpty ? website.alias : website.primaryDomain),
              trailing: _statusBadge(website.status),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _kv('域名', website.primaryDomain.isEmpty ? '-' : website.primaryDomain),
                  const SizedBox(height: 8),
                  _kv('类型', website.type.isEmpty ? '-' : website.type),
                  const SizedBox(height: 8),
                  _kv('应用', website.appName.isEmpty ? '-' : website.appName),
                  const SizedBox(height: 8),
                  _kv('流水线', website.pipelineId > 0 ? '#${website.pipelineId}' : '-'),
                ],
              ),
            ),
            const SizedBox(height: 16),
            PanelCard(
              title: const Text('操作'),
              child: Column(
                children: [
                  SizedBox(
                    width: double.infinity,
                    child: ElevatedButton.icon(
                      icon: const Icon(Icons.rocket_launch_outlined),
                      onPressed: website.pipelineId > 0
                          ? () => _runPipeline(context, ref)
                          : null,
                      label: const Text('触发流水线部署'),
                    ),
                  ),
                  const SizedBox(height: 10),
                  SizedBox(
                    width: double.infinity,
                    child: OutlinedButton.icon(
                      icon: const Icon(Icons.publish_outlined),
                      onPressed: () => _deployTrigger(context, ref),
                      label: const Text('触发部署（无日志）'),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _runPipeline(BuildContext context, WidgetRef ref) async {
    try {
      final repo = ref.read(websiteRepositoryProvider);
      final recordId = await repo.runPipeline(website.pipelineId);
      if (recordId <= 0) {
        throw Exception('未获取到 recordId');
      }
      final now = DateTime.now();
      final task = TaskEntity(
        id: 'pipeline:$recordId',
        title: '${website.alias.isNotEmpty ? website.alias : website.primaryDomain} #$recordId',
        type: TaskType.pipeline,
        status: TaskStatus.running,
        startedAt: now,
        updatedAt: now,
        summary: 'pipelineId=${website.pipelineId}',
      );
      ref.read(taskCenterControllerProvider.notifier).addLocalTask(task);
      ref
          .read(mainScaffoldIndexProvider.notifier)
          .setIndex(MainScaffoldIndexController.codeIndex);
      if (!context.mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.error),
      );
    }
  }

  Future<void> _deployTrigger(BuildContext context, WidgetRef ref) async {
    try {
      final repo = ref.read(websiteRepositoryProvider);
      await repo.deployTrigger(websiteId: website.id);
      final now = DateTime.now();
      final task = TaskEntity(
        id: 'websiteDeploy:${website.id}',
        title: '网站部署 ${website.alias.isNotEmpty ? website.alias : website.primaryDomain}',
        type: TaskType.other,
        status: TaskStatus.running,
        startedAt: now,
        updatedAt: now,
        summary: 'deploy trigger',
      );
      ref.read(taskCenterControllerProvider.notifier).addLocalTask(task);
      ref
          .read(mainScaffoldIndexProvider.notifier)
          .setIndex(MainScaffoldIndexController.codeIndex);
      if (!context.mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)),
      );
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已触发部署（请稍后刷新网站状态）')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.error),
      );
    }
  }

  Widget _statusBadge(String s) {
    final lower = s.toLowerCase();
    final ok = lower == 'running' || lower == 'enable' || lower == 'enabled';
    final color = ok ? AppTheme.success : AppTheme.textSecondary;
    final bg = ok ? const Color(0xFFECFDF5) : AppTheme.primaryBlueLight;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Text(
        s.isEmpty ? '-' : s,
        style: TextStyle(fontWeight: FontWeight.w800, color: color, fontSize: 12),
      ),
    );
  }

  Widget _kv(String k, String v) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(k, style: const TextStyle(color: AppTheme.textSecondary, fontWeight: FontWeight.w600)),
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
}
