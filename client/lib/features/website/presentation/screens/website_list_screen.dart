import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/website_info.dart';
import '../controllers/website_controller.dart';
import 'website_detail_screen.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class WebsiteListScreen extends ConsumerWidget {
  final bool embedded;

  const WebsiteListScreen({super.key, this.embedded = false});
  const WebsiteListScreen.embedded({super.key}) : embedded = true;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(websiteControllerProvider);
    if (embedded) {
      return Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    '网站',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.refresh_rounded),
                  onPressed: () {
                    ref.read(websiteControllerProvider.notifier).refresh();
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
        title: const Text('网站'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(websiteControllerProvider.notifier).refresh();
            },
          ),
        ],
      ),
      body: _body(context, ref, state),
    );
  }

  Widget _body(BuildContext context, WidgetRef ref, WebsiteListState state) {
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
          '暂无网站',
          style: TextStyle(color: AppTheme.textSecondary),
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(websiteControllerProvider.notifier).refresh();
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: state.items.length,
        separatorBuilder: (_, _) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          final item = state.items[index];
          return _card(context, item);
        },
      ),
    );
  }

  Widget _card(BuildContext context, WebsiteInfo w) {
    return InkWell(
      borderRadius: BorderRadius.circular(20),
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(builder: (_) => WebsiteDetailScreen(website: w)),
        );
      },
      child: PanelCard(
        title: Text(w.alias.isNotEmpty ? w.alias : w.primaryDomain),
        trailing: _statusBadge(w.status),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (w.primaryDomain.isNotEmpty)
              Text(
                w.primaryDomain,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 10,
              runSpacing: 8,
              children: [
                _kv('类型', w.type.isEmpty ? '-' : w.type),
                _kv('应用', w.appName.isEmpty ? '-' : w.appName),
                _kv('流水线', w.pipelineId > 0 ? '#${w.pipelineId}' : '-'),
              ],
            ),
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
        style: TextStyle(
          fontWeight: FontWeight.w800,
          color: color,
          fontSize: 12,
        ),
      ),
    );
  }
}
