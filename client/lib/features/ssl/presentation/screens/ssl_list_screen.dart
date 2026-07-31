import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../../models/ssl_info.dart';
import '../controllers/ssl_controller.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_status.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class SslListScreen extends ConsumerWidget {
  final bool embedded;

  const SslListScreen({super.key, this.embedded = false});
  const SslListScreen.embedded({super.key}) : embedded = true;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(sslControllerProvider);
    if (embedded) {
      return Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    'SSL 证书',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.refresh_rounded),
                  onPressed: () {
                    ref.read(sslControllerProvider.notifier).refresh();
                  },
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(child: _body(context, ref, state)),
        ],
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('SSL 证书'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(sslControllerProvider.notifier).refresh();
            },
          ),
        ],
      ),
      body: _body(context, ref, state),
    );
  }

  Widget _body(BuildContext context, WidgetRef ref, SslListState state) {
    if (state.isLoading && state.items.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (state.errorMessage != null && state.items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            state.errorMessage!,
            style: const TextStyle(color: AppTheme.error),
            textAlign: TextAlign.center,
          ),
        ),
      );
    }
    if (state.items.isEmpty) {
      return const Center(
        child: Text('暂无证书', style: TextStyle(color: AppTheme.textSecondary)),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(sslControllerProvider.notifier).refresh();
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: state.items.length,
        separatorBuilder: (_, _) => const SizedBox(height: 12),
        itemBuilder: (context, index) => _card(context, ref, state.items[index]),
      ),
    );
  }

  Widget _card(BuildContext context, WidgetRef ref, SslInfo s) {
    return PanelCard(
      title: Text(s.primaryDomain.isEmpty ? '证书 #${s.id}' : s.primaryDomain),
      trailing: _statusBadge(s.status),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 10,
            runSpacing: 8,
            children: [
              _kv('类型', s.type.isEmpty ? '-' : s.type),
              _kv('Provider', s.provider.isEmpty ? '-' : s.provider),
              _kv('自动续期', s.autoRenew ? '开启' : '关闭'),
              _kv('到期', _formatDate(s.expireDate)),
            ],
          ),
          if (s.message.isNotEmpty) ...[
            const SizedBox(height: 10),
            Text(
              s.message,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: ElevatedButton(
                  onPressed: () async {
                    await _renew(context, ref, s.id);
                  },
                  child: const Text('续期'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _renew(BuildContext context, WidgetRef ref, int id) async {
    try {
      final repo = ref.read(sslRepositoryProvider);
      await repo.renew(id);
      if (!context.mounted) return;
      final now = DateTime.now();
      final task = TaskEntity(
        id: 'ssl:$id',
        title: '续期证书 #$id',
        type: TaskType.ssl,
        status: TaskStatus.running,
        startedAt: now,
        updatedAt: now,
        summary: 'SSL Renew',
      );
      ref.read(taskCenterControllerProvider.notifier).addLocalTask(task);
      ref
          .read(mainScaffoldIndexProvider.notifier)
          .setIndex(MainScaffoldIndexController.codeIndex);
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)),
      );
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已触发续期，请在日志中查看进度')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.toString()), backgroundColor: AppTheme.error),
      );
    }
  }

  String _formatDate(DateTime? d) {
    if (d == null) return '-';
    String p2(int v) => v.toString().padLeft(2, '0');
    return '${d.year}-${p2(d.month)}-${p2(d.day)}';
  }

  Widget _kv(String k, String v) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: AppTheme.primaryBlueLight,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppTheme.primaryBlueBorder),
      ),
      child: Text(
        '$k: $v',
        style: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w700,
          color: AppTheme.textSecondary,
        ),
      ),
    );
  }

  Widget _statusBadge(String s) {
    final lower = s.toLowerCase();
    final good = lower == 'valid' || lower == 'success';
    final bad = lower == 'failed' || lower == 'error' || lower == 'invalid';
    final color = good
        ? AppTheme.success
        : bad
            ? AppTheme.error
            : AppTheme.textSecondary;
    final bg = good
        ? const Color(0xFFECFDF5)
        : bad
            ? const Color(0xFFFFF1F2)
            : AppTheme.primaryBlueLight;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Text(
        s.isEmpty ? '-' : s,
        style: TextStyle(
          fontWeight: FontWeight.w800,
          color: color,
          fontSize: 12,
        ),
      ),
    );
  }
}
