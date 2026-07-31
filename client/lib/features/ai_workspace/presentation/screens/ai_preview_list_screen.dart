import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/ai_dev_session.dart';
import '../controllers/ai_workspace_controller.dart';
import 'ai_preview_detail_screen.dart';

class AiPreviewListScreen extends ConsumerStatefulWidget {
  const AiPreviewListScreen({
    super.key,
    required this.sessionId,
    required this.title,
  });

  final int sessionId;
  final String title;

  @override
  ConsumerState<AiPreviewListScreen> createState() =>
      _AiPreviewListScreenState();
}

class _AiPreviewListScreenState extends ConsumerState<AiPreviewListScreen> {
  bool _loading = true;
  String? _error;
  List<AiPreview> _previews = const [];

  @override
  void initState() {
    super.initState();
    _loadPreviews();
  }

  Future<void> _loadPreviews() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final previews = await ref
          .read(aiWorkspaceRepositoryProvider)
          .getSessionPreviews(widget.sessionId);
      if (!mounted) return;
      setState(() {
        _previews = previews;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  void _openPreview(AiPreview preview) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => AiPreviewDetailScreen(preview: preview),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
        actions: [
          IconButton(
            tooltip: '刷新',
            onPressed: _loadPreviews,
            icon: const Icon(Icons.refresh_rounded),
          ),
        ],
      ),
      body: RefreshIndicator(onRefresh: _loadPreviews, child: _buildBody()),
    );
  }

  Widget _buildBody() {
    if (_loading && _previews.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null && _previews.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              children: [
                const SizedBox(height: 48),
                const Icon(Icons.link_off_rounded, size: 42),
                const SizedBox(height: 12),
                Text('加载预览失败', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 8),
                Text(_error!, textAlign: TextAlign.center),
              ],
            ),
          ),
        ],
      );
    }

    if (_previews.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 96),
          Center(child: Icon(Icons.desktop_windows_outlined, size: 44)),
          SizedBox(height: 12),
          Center(child: Text('当前还没有可展示的预览')),
        ],
      );
    }

    return ListView.separated(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      itemCount: _previews.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (context, index) {
        final preview = _previews[index];
        return Card(
          child: InkWell(
            onTap: () => _openPreview(preview),
            borderRadius: BorderRadius.circular(12),
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          preview.title.isEmpty ? '开发预览' : preview.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.titleSmall
                              ?.copyWith(fontWeight: FontWeight.w700),
                        ),
                      ),
                      const SizedBox(width: 12),
                      _PreviewStatusBadge(status: preview.status),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(
                    preview.url,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(
                      context,
                    ).textTheme.bodySmall?.copyWith(height: 1.5),
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      if (preview.taskId > 0) Text('任务 #${preview.taskId}'),
                      if (preview.taskId > 0 && preview.previewType.isNotEmpty)
                        const Text(' · '),
                      if (preview.previewType.isNotEmpty)
                        Text(preview.previewType),
                      const Spacer(),
                      const Icon(Icons.chevron_right_rounded),
                    ],
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

class _PreviewStatusBadge extends StatelessWidget {
  const _PreviewStatusBadge({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (status) {
      case 'ready':
        color = Colors.green;
        break;
      case 'local':
        color = Colors.orange;
        break;
      case 'unreachable':
      case 'failed':
      case 'invalid':
        color = Colors.red;
        break;
      default:
        color = Colors.blueGrey;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        status.isEmpty ? 'unknown' : status,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}
