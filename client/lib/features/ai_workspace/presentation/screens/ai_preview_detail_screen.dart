import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../models/ai_dev_session.dart';

class AiPreviewDetailScreen extends StatelessWidget {
  const AiPreviewDetailScreen({super.key, required this.preview});

  final AiPreview preview;

  bool get _isLocalPreview =>
      preview.status == 'local' ||
      preview.url.contains('localhost') ||
      preview.url.contains('127.0.0.1') ||
      preview.url.contains('0.0.0.0');

  Future<void> _openPreview(BuildContext context) async {
    final messenger = ScaffoldMessenger.of(context);
    final uri = Uri.tryParse(preview.url);
    if (uri == null) {
      messenger.showSnackBar(const SnackBar(content: Text('预览地址无效')));
      return;
    }

    final opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (!opened && context.mounted) {
      messenger.showSnackBar(const SnackBar(content: Text('暂时无法打开预览')));
    }
  }

  Future<void> _copyUrl(BuildContext context) async {
    await Clipboard.setData(ClipboardData(text: preview.url));
    if (context.mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('预览链接已复制')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(preview.title.isEmpty ? '开发预览' : preview.title),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_isLocalPreview)
            Container(
              padding: const EdgeInsets.all(12),
              margin: const EdgeInsets.only(bottom: 16),
              decoration: BoxDecoration(
                color: Colors.orange.withValues(alpha: 0.08),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.orange.withValues(alpha: 0.2)),
              ),
              child: const Text(
                '当前预览地址看起来是开发机本地地址，手机通常无法直接访问。后续建议接入代理地址或正式预览地址。',
                style: TextStyle(height: 1.5),
              ),
            ),
          _InfoCard(
            title: '状态',
            child: Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                _StatusChip(
                  label: preview.status.isEmpty ? 'unknown' : preview.status,
                ),
                if (preview.previewType.isNotEmpty)
                  _MetaChip(label: preview.previewType),
                if (preview.source.isNotEmpty) _MetaChip(label: preview.source),
              ],
            ),
          ),
          const SizedBox(height: 12),
          _InfoCard(
            title: '预览地址',
            child: SelectableText(
              preview.url,
              style: theme.textTheme.bodyMedium?.copyWith(height: 1.6),
            ),
          ),
          const SizedBox(height: 12),
          _InfoCard(
            title: '关联信息',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('会话 #${preview.sessionId}'),
                if (preview.taskId > 0) Text('任务 #${preview.taskId}'),
                if (preview.instructionId > 0)
                  Text('指令 #${preview.instructionId}'),
                if (preview.updatedAt != null)
                  Text('更新时间: ${preview.updatedAt}'),
              ],
            ),
          ),
        ],
      ),
      bottomNavigationBar: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
          child: Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => _copyUrl(context),
                  icon: const Icon(Icons.copy_rounded),
                  label: const Text('复制链接'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => _openPreview(context),
                  icon: const Icon(Icons.open_in_browser_rounded),
                  label: const Text('打开预览'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _InfoCard extends StatelessWidget {
  const _InfoCard({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: Theme.of(
                context,
              ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 12),
            child,
          ],
        ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (label) {
      case 'ready':
        color = Colors.green;
        break;
      case 'local':
        color = Colors.orange;
        break;
      case 'failed':
      case 'unreachable':
        color = Colors.red;
        break;
      default:
        color = Colors.blueGrey;
    }

    return Chip(
      label: Text(label),
      backgroundColor: color.withValues(alpha: 0.12),
      side: BorderSide(color: color.withValues(alpha: 0.25)),
      labelStyle: TextStyle(color: color),
      visualDensity: VisualDensity.compact,
    );
  }
}

class _MetaChip extends StatelessWidget {
  const _MetaChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(label: Text(label), visualDensity: VisualDensity.compact);
  }
}
