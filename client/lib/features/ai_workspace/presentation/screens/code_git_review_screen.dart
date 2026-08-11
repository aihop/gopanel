import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/ai_dev_session.dart';
import '../../models/code_git_review.dart';
import '../code_git_review_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_git_review_controller.dart';
import '../widgets/code_git_review_cards.dart';
import 'code_git_diff_screen.dart';

class CodeGitReviewScreen extends ConsumerStatefulWidget {
  const CodeGitReviewScreen({super.key, required this.session});

  final AiDevSession session;

  @override
  ConsumerState<CodeGitReviewScreen> createState() =>
      _CodeGitReviewScreenState();
}

class _CodeGitReviewScreenState extends ConsumerState<CodeGitReviewScreen> {
  late final CodeGitReviewController _controller;

  bool get _canSaveSession =>
      widget.session.worktreeBranch.isNotEmpty ||
      widget.session.isolationMode == 'multi_worktree';

  @override
  void initState() {
    super.initState();
    _controller = CodeGitReviewController(
      repository: ref.read(aiWorkspaceRepositoryProvider),
      sessionId: widget.session.id,
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
            title: Text(CodeGitReviewText.t(context, 'title')),
            actions: [
              IconButton(
                onPressed: state.isLoading || state.isSaving
                    ? null
                    : _controller.load,
                icon: const Icon(Icons.refresh_rounded),
              ),
            ],
          ),
          body: RefreshIndicator(
            onRefresh: _controller.load,
            child: _buildBody(state),
          ),
          bottomNavigationBar: _buildSaveBar(state),
        );
      },
    );
  }

  Widget _buildBody(CodeGitReviewState state) {
    if (state.isLoading && state.status == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 180),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (state.errorMessage != null && state.status == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 72),
          const Icon(Icons.source_outlined, size: 48),
          const SizedBox(height: 12),
          Text(
            CodeGitReviewText.t(context, 'loadFailed'),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(state.errorMessage!, textAlign: TextAlign.center),
        ],
      );
    }
    final status = state.status;
    if (status == null || !status.available) {
      return _Empty(message: CodeGitReviewText.t(context, 'noRepository'));
    }
    if (!status.hasChanges) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        children: [
          if (state.errorMessage != null) ...[
            _InlineError(message: state.errorMessage!),
            const SizedBox(height: 12),
          ],
          CodeGitSummaryCard(status: status),
          const SizedBox(height: 72),
          Center(child: Text(CodeGitReviewText.t(context, 'noChanges'))),
        ],
      );
    }
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: [
        if (state.isLoading || state.isSaving)
          const LinearProgressIndicator(minHeight: 2),
        if (state.errorMessage != null) ...[
          _InlineError(message: state.errorMessage!),
          const SizedBox(height: 12),
        ],
        CodeGitSummaryCard(status: status),
        const SizedBox(height: 16),
        for (var index = 0; index < status.repositories.length; index++) ...[
          CodeGitRepositoryCard(
            repository: status.repositories[index],
            onOpenDiff: (file, kind) =>
                _openDiff(status.repositories[index], file, kind),
          ),
          if (index != status.repositories.length - 1)
            const SizedBox(height: 16),
        ],
      ],
    );
  }

  Widget? _buildSaveBar(CodeGitReviewState state) {
    final hasChanges = state.status?.hasChanges == true;
    if (!hasChanges) return null;
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (!_canSaveSession)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Text(
                  CodeGitReviewText.t(context, 'saveUnavailable'),
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontSize: 12,
                  ),
                ),
              ),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: !_canSaveSession || state.isSaving
                    ? null
                    : _confirmSave,
                icon: state.isSaving
                    ? const SizedBox.square(
                        dimension: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.save_outlined),
                label: Text(CodeGitReviewText.t(context, 'save')),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _openDiff(
    CodeGitRepositoryStatus repository,
    CodeGitFile file,
    String kind,
  ) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeGitDiffScreen(
          sessionId: widget.session.id,
          repositoryId: repository.id,
          path: file.path,
          kind: kind,
        ),
      ),
    );
  }

  Future<void> _confirmSave() async {
    final messageController = TextEditingController();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(CodeGitReviewText.t(dialogContext, 'saveTitle')),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(CodeGitReviewText.t(dialogContext, 'saveHint')),
            const SizedBox(height: 16),
            TextField(
              controller: messageController,
              maxLength: 500,
              decoration: InputDecoration(
                labelText: CodeGitReviewText.t(dialogContext, 'message'),
                hintText: CodeGitReviewText.t(dialogContext, 'messageHint'),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(CodeGitReviewText.t(dialogContext, 'cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(CodeGitReviewText.t(dialogContext, 'confirmSave')),
          ),
        ],
      ),
    );
    final message = messageController.text;
    messageController.dispose();
    if (confirmed != true || !mounted) return;
    final result = await _controller.save(message);
    if (!mounted) return;
    if (result != null) {
      await ref
          .read(aiWorkspaceControllerProvider.notifier)
          .refreshCurrentSession(showLoading: true);
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          CodeGitReviewText.t(
            context,
            result == null ? 'saveFailed' : 'saveSuccess',
          ),
        ),
      ),
    );
  }
}

class _Empty extends StatelessWidget {
  const _Empty({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        const SizedBox(height: 120),
        const Center(child: Icon(Icons.source_outlined, size: 48)),
        const SizedBox(height: 12),
        Center(child: Text(message)),
      ],
    );
  }
}

class _InlineError extends StatelessWidget {
  const _InlineError({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF1F2),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(message, style: const TextStyle(color: AppTheme.error)),
    );
  }
}
