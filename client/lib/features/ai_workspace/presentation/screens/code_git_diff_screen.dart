import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../code_git_review_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_git_diff_controller.dart';

class CodeGitDiffScreen extends ConsumerStatefulWidget {
  const CodeGitDiffScreen({
    super.key,
    required this.sessionId,
    required this.repositoryId,
    required this.path,
    required this.kind,
  });

  final int sessionId;
  final String repositoryId;
  final String path;
  final String kind;

  @override
  ConsumerState<CodeGitDiffScreen> createState() => _CodeGitDiffScreenState();
}

class _CodeGitDiffScreenState extends ConsumerState<CodeGitDiffScreen> {
  late final CodeGitDiffController _controller;

  @override
  void initState() {
    super.initState();
    _controller = CodeGitDiffController(
      repository: ref.read(aiWorkspaceRepositoryProvider),
      sessionId: widget.sessionId,
      repositoryId: widget.repositoryId,
      path: widget.path,
      kind: widget.kind,
    )..load();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) {
        final state = _controller.state;
        return Scaffold(
          appBar: AppBar(
            title: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(CodeGitReviewText.t(context, 'diffTitle')),
                Text(
                  widget.path,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 11),
                ),
              ],
            ),
            actions: [
              IconButton(
                tooltip: CodeGitReviewText.t(context, 'copyDiff'),
                onPressed: state.diff?.content.isNotEmpty == true
                    ? () => _copy(state.diff!.content)
                    : null,
                icon: const Icon(Icons.content_copy_rounded),
              ),
              IconButton(
                onPressed: state.isLoading ? null : _controller.load,
                icon: const Icon(Icons.refresh_rounded),
              ),
            ],
          ),
          body: RefreshIndicator(
            onRefresh: _controller.load,
            child: _buildBody(state),
          ),
        );
      },
    );
  }

  Widget _buildBody(CodeGitDiffState state) {
    if (state.isLoading && state.diff == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 180),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (state.errorMessage != null && state.diff == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 72),
          const Icon(Icons.difference_outlined, size: 48),
          const SizedBox(height: 12),
          Text(
            CodeGitReviewText.t(context, 'diffLoadFailed'),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(state.errorMessage!, textAlign: TextAlign.center),
          const SizedBox(height: 14),
          Center(
            child: OutlinedButton(
              onPressed: _controller.load,
              child: const Icon(Icons.refresh_rounded),
            ),
          ),
        ],
      );
    }
    final diff = state.diff;
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(12),
      children: [
        if (state.isLoading) const LinearProgressIndicator(minHeight: 2),
        if (state.errorMessage != null) ...[
          _Notice(message: state.errorMessage!, error: true),
          const SizedBox(height: 10),
        ],
        if (diff?.truncated == true) ...[
          _Notice(message: CodeGitReviewText.t(context, 'diffTruncated')),
          const SizedBox(height: 10),
        ],
        if (diff == null || diff.content.isEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 80),
            child: Center(
              child: Text(CodeGitReviewText.t(context, 'diffEmpty')),
            ),
          )
        else
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: const Color(0xFF0F172A),
              borderRadius: BorderRadius.circular(12),
            ),
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: SelectableText(
                diff.content,
                style: const TextStyle(
                  color: Color(0xFFCBD5E1),
                  fontFamily: 'monospace',
                  fontSize: 11,
                  height: 1.5,
                ),
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _copy(String content) async {
    await Clipboard.setData(ClipboardData(text: content));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(CodeGitReviewText.t(context, 'diffCopied'))),
    );
  }
}

class _Notice extends StatelessWidget {
  const _Notice({required this.message, this.error = false});

  final String message;
  final bool error;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: error ? const Color(0xFFFFF1F2) : const Color(0xFFFFFBEB),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        message,
        style: TextStyle(color: error ? AppTheme.error : Colors.orange),
      ),
    );
  }
}
