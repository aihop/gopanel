import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/database_info.dart';
import '../controllers/database_controller.dart';
import 'database_detail_screen.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class DatabaseListScreen extends ConsumerWidget {
  final bool embedded;

  const DatabaseListScreen({super.key, this.embedded = false});
  const DatabaseListScreen.embedded({super.key}) : embedded = true;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(databaseControllerProvider);
    if (embedded) {
      return Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 8, 4),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    '数据库',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.refresh_rounded),
                  iconSize: 20,
                  visualDensity: VisualDensity.compact,
                  onPressed: () {
                    ref.read(databaseControllerProvider.notifier).refresh();
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
        title: const Text('数据库'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(databaseControllerProvider.notifier).refresh();
            },
          ),
        ],
      ),
      body: _body(context, ref, state),
    );
  }

  Widget _body(BuildContext context, WidgetRef ref, DatabaseListState state) {
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
        child: Text(
          '暂无数据库',
          style: TextStyle(color: AppTheme.textSecondary),
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(databaseControllerProvider.notifier).refresh();
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: state.items.length,
        separatorBuilder: (_, _) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          return _card(context, state.items[index]);
        },
      ),
    );
  }

  Widget _card(BuildContext context, DatabaseInfo d) {
    return InkWell(
      borderRadius: BorderRadius.circular(20),
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(builder: (_) => DatabaseDetailScreen(database: d)),
        );
      },
      child: PanelCard(
        title: Text(d.name),
        trailing: _typeBadge(d.type),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Wrap(
              spacing: 10,
              runSpacing: 8,
              children: [
                _kv('服务端', d.server.isEmpty ? '-' : d.server),
                _kv('编码', d.encoding.isEmpty ? '-' : d.encoding),
              ],
            ),
            if (d.comment.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(
                d.comment,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ],
        ),
      ),
    );
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

  Widget _typeBadge(String type) {
    final t = type.isEmpty ? '-' : type;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: AppTheme.primaryBlueLight,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppTheme.primaryBlueBorder),
      ),
      child: Text(
        t,
        style: const TextStyle(
          fontWeight: FontWeight.w800,
          color: AppTheme.textSecondary,
          fontSize: 12,
        ),
      ),
    );
  }
}
