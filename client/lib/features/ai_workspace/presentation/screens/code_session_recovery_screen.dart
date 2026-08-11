import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../models/code_session_recovery.dart';
import '../code_workspace_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_session_recovery_controller.dart';
import '../widgets/code_recovery_cards.dart';
import 'code_execution_run_screen.dart';

class CodeSessionRecoveryScreen extends ConsumerStatefulWidget {
  const CodeSessionRecoveryScreen({
    super.key,
    required this.sessionId,
    required this.sessionTitle,
  });

  final int sessionId;
  final String sessionTitle;

  @override
  ConsumerState<CodeSessionRecoveryScreen> createState() =>
      _CodeSessionRecoveryScreenState();
}

class _CodeSessionRecoveryScreenState
    extends ConsumerState<CodeSessionRecoveryScreen> {
  late final CodeSessionRecoveryController _controller;

  @override
  void initState() {
    super.initState();
    _controller = CodeSessionRecoveryController(
      repository: ref.read(aiWorkspaceRepositoryProvider),
      sessionId: widget.sessionId,
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
                Text(CodeWorkspaceText.t(context, 'recovery.title')),
                Text(
                  widget.sessionTitle,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontSize: 11,
                  ),
                ),
              ],
            ),
            actions: [
              IconButton(
                tooltip: CodeWorkspaceText.t(context, 'action.refresh'),
                onPressed: state.isLoading ? null : _controller.load,
                icon: const Icon(Icons.refresh_rounded),
              ),
            ],
          ),
          body: RefreshIndicator(
            onRefresh: _controller.load,
            child: _buildBody(context, state),
          ),
        );
      },
    );
  }

  Widget _buildBody(BuildContext context, CodeSessionRecoveryState state) {
    if (state.isLoading && state.messages.isEmpty && state.runs.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 180),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (state.errorMessage != null &&
        state.messages.isEmpty &&
        state.runs.isEmpty &&
        state.initialization == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 72),
          const Icon(Icons.history_toggle_off_rounded, size: 48),
          const SizedBox(height: 12),
          Text(
            CodeWorkspaceText.t(context, 'recovery.loadFailed'),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            state.errorMessage!,
            textAlign: TextAlign.center,
            style: const TextStyle(color: AppTheme.textSecondary),
          ),
          const SizedBox(height: 14),
          Center(
            child: OutlinedButton(
              onPressed: _controller.load,
              child: Text(CodeWorkspaceText.t(context, 'action.retry')),
            ),
          ),
        ],
      );
    }

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: [
        if (state.isLoading) const LinearProgressIndicator(minHeight: 2),
        if (state.errorMessage != null) ...[
          _InlineError(message: state.errorMessage!),
          const SizedBox(height: 12),
        ],
        if (state.initialization != null) ...[
          CodeInitializationCard(
            initialization: state.initialization!,
            isRetrying: state.isRetryingInitialization,
            onRetry: _retryInitialization,
          ),
          const SizedBox(height: 16),
        ],
        CodeExecutionRunsCard(
          runs: state.runs,
          total: state.totalRuns,
          isLoadingMore: state.isLoadingMore,
          retryingInstructionId: state.retryingInstructionId,
          retriedInstructionIds: state.retriedInstructionIds,
          onRetry: _retryInstruction,
          onOpenDetail: _openRunDetail,
          onLoadMore: _controller.loadMore,
        ),
        const SizedBox(height: 16),
        CodeHistoryMessagesCard(messages: state.messages),
      ],
    );
  }

  Future<void> _retryInstruction(CodeExecutionRun run) async {
    final confirmed = await _confirm(
      title: CodeWorkspaceText.t(context, 'recovery.retryInstruction'),
      content: CodeWorkspaceText.t(context, 'recovery.retryInstructionHint'),
    );
    if (!confirmed || !mounted) return;
    final success = await _controller.retryInstruction(run);
    if (!mounted) return;
    if (success) {
      await ref
          .read(aiWorkspaceControllerProvider.notifier)
          .refreshCurrentSession(showLoading: true);
    }
    _showResult(success);
  }

  void _openRunDetail(CodeExecutionRun run) {
    Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => CodeExecutionRunScreen(runId: run.id)),
    );
  }

  Future<void> _retryInitialization() async {
    final confirmed = await _confirm(
      title: CodeWorkspaceText.t(context, 'recovery.retryInitialization'),
      content: CodeWorkspaceText.t(context, 'recovery.retryInitializationHint'),
    );
    if (!confirmed || !mounted) return;
    final success = await _controller.retryInitialization();
    if (!mounted) return;
    _showResult(success);
  }

  Future<bool> _confirm({
    required String title,
    required String content,
  }) async {
    return await showDialog<bool>(
          context: context,
          builder: (dialogContext) => AlertDialog(
            title: Text(title),
            content: Text(content),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(false),
                child: Text(
                  CodeWorkspaceText.t(dialogContext, 'recovery.cancel'),
                ),
              ),
              FilledButton(
                onPressed: () => Navigator.of(dialogContext).pop(true),
                child: Text(
                  CodeWorkspaceText.t(dialogContext, 'recovery.confirm'),
                ),
              ),
            ],
          ),
        ) ??
        false;
  }

  void _showResult(bool success) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          CodeWorkspaceText.t(
            context,
            success ? 'recovery.actionSuccess' : 'recovery.actionFailed',
          ),
        ),
      ),
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
