import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_status.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';
import '../../models/database_info.dart';
import '../controllers/database_controller.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class DatabaseDetailScreen extends ConsumerWidget {
  final DatabaseInfo database;

  const DatabaseDetailScreen({super.key, required this.database});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('数据库详情')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            PanelCard(
              title: Text(database.name),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _kv('类型', database.type.isEmpty ? '-' : database.type),
                  const SizedBox(height: 8),
                  _kv('服务端', database.server.isEmpty ? '-' : database.server),
                  const SizedBox(height: 8),
                  _kv('编码', database.encoding.isEmpty ? '-' : database.encoding),
                ],
              ),
            ),
            const SizedBox(height: 16),
            PanelCard(
              title: const Text('操作'),
              child: SizedBox(
                width: double.infinity,
                child: ElevatedButton.icon(
                  icon: const Icon(Icons.sync_rounded),
                  onPressed:
                      database.serverId > 0 ? () => _sync(context, ref) : null,
                  label: const Text('同步服务端数据库'),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _sync(BuildContext context, WidgetRef ref) async {
    try {
      final repo = ref.read(databaseRepositoryProvider);
      await repo.syncServer(database.serverId);
      final now = DateTime.now();
      final task = TaskEntity(
        id: 'dbSync:${database.serverId}:${now.millisecondsSinceEpoch}',
        title: '同步数据库服务端 #${database.serverId}',
        type: TaskType.other,
        status: TaskStatus.success,
        startedAt: now,
        updatedAt: now,
        summary: database.server,
      );
      ref.read(taskCenterControllerProvider.notifier).addLocalTask(task);
      ref.read(mainScaffoldIndexProvider.notifier).setIndex(2);
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
}
