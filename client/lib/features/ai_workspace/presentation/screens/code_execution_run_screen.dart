import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../models/code_execution_run_detail.dart';
import '../code_workspace_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_execution_run_controller.dart';

class CodeExecutionRunScreen extends ConsumerStatefulWidget {
  const CodeExecutionRunScreen({super.key, required this.runId});

  final int runId;

  @override
  ConsumerState<CodeExecutionRunScreen> createState() =>
      _CodeExecutionRunScreenState();
}

class _CodeExecutionRunScreenState
    extends ConsumerState<CodeExecutionRunScreen> {
  late final CodeExecutionRunController _controller;

  @override
  void initState() {
    super.initState();
    _controller = CodeExecutionRunController(
      repository: ref.read(aiWorkspaceRepositoryProvider),
      runId: widget.runId,
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
            title: Text(
              CodeWorkspaceText.format(context, 'runDetail.title', {
                'id': widget.runId,
              }),
            ),
            actions: [
              IconButton(
                tooltip: CodeWorkspaceText.t(context, 'runDetail.copy'),
                onPressed: state.run == null ? null : () => _copy(state.run!),
                icon: const Icon(Icons.content_copy_rounded),
              ),
              IconButton(
                tooltip: CodeWorkspaceText.t(context, 'action.refresh'),
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

  Widget _buildBody(CodeExecutionRunState state) {
    if (state.isLoading && state.run == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 180),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (state.errorMessage != null && state.run == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 72),
          const Icon(Icons.error_outline_rounded, size: 48),
          const SizedBox(height: 12),
          Text(
            CodeWorkspaceText.t(context, 'runDetail.loadFailed'),
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
    final run = state.run;
    if (run == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          const SizedBox(height: 120),
          Center(child: Text(CodeWorkspaceText.t(context, 'runDetail.empty'))),
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
        CodeExecutionRunOverview(run: run),
        const SizedBox(height: 16),
        _ContentCard(
          title: CodeWorkspaceText.t(context, 'runDetail.prompt'),
          content: run.prompt,
          emptyKey: 'runDetail.noPrompt',
        ),
        const SizedBox(height: 16),
        if (run.errorMessage.isNotEmpty) ...[
          _ContentCard(
            title: CodeWorkspaceText.t(context, 'runDetail.error'),
            content: run.errorMessage,
            error: true,
          ),
          const SizedBox(height: 16),
        ],
        _ContentCard(
          title: CodeWorkspaceText.t(context, 'runDetail.output'),
          content: run.output,
          emptyKey: 'runDetail.noOutput',
        ),
        const SizedBox(height: 16),
        _ContentCard(
          title: CodeWorkspaceText.t(context, 'runDetail.rawOutput'),
          content: run.rawOutput,
          emptyKey: 'runDetail.noRawOutput',
          code: true,
        ),
      ],
    );
  }

  Future<void> _copy(CodeExecutionRunDetail run) async {
    await Clipboard.setData(ClipboardData(text: run.diagnosticText));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(CodeWorkspaceText.t(context, 'runDetail.copied'))),
    );
  }
}

class CodeExecutionRunOverview extends StatelessWidget {
  const CodeExecutionRunOverview({super.key, required this.run});

  final CodeExecutionRunDetail run;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Text(CodeWorkspaceText.t(context, 'runDetail.overview')),
      child: Wrap(
        spacing: 10,
        runSpacing: 10,
        children: [
          _Meta(
            label: CodeWorkspaceText.t(context, 'runDetail.status'),
            value: _statusLabel(context, run.status),
          ),
          _Meta(
            label: CodeWorkspaceText.t(context, 'runDetail.executor'),
            value: run.executorId,
          ),
          if (run.model.isNotEmpty)
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.model'),
              value: run.model,
            ),
          _Meta(
            label: CodeWorkspaceText.t(context, 'runDetail.exitCode'),
            value: '${run.exitCode}',
          ),
          _Meta(
            label: CodeWorkspaceText.t(context, 'runDetail.durationLabel'),
            value: '${run.durationMs} ms',
          ),
          _Meta(
            label: CodeWorkspaceText.t(context, 'runDetail.session'),
            value: '#${run.sessionId}',
          ),
          if (run.taskId > 0)
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.task'),
              value: '#${run.taskId}',
            ),
          if (run.instructionId > 0)
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.instruction'),
              value: '#${run.instructionId}',
            ),
          if (run.startedAt != null)
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.startedAt'),
              value: _formatTime(run.startedAt!),
            ),
          if (run.completedAt != null)
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.completedAt'),
              value: _formatTime(run.completedAt!),
            ),
          if (run.hasTokenUsage) ...[
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.inputTokens'),
              value: '${run.inputTokens}',
            ),
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.outputTokens'),
              value: '${run.outputTokens}',
            ),
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.cachedTokens'),
              value: '${run.cachedInputTokens}',
            ),
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.reasoningTokens'),
              value: '${run.reasoningTokens}',
            ),
            _Meta(
              label: CodeWorkspaceText.t(context, 'runDetail.totalTokens'),
              value: '${run.totalTokens}',
            ),
          ],
        ],
      ),
    );
  }
}

class _ContentCard extends StatelessWidget {
  const _ContentCard({
    required this.title,
    required this.content,
    this.emptyKey,
    this.error = false,
    this.code = false,
  });

  final String title;
  final String content;
  final String? emptyKey;
  final bool error;
  final bool code;

  @override
  Widget build(BuildContext context) {
    final value = content.isEmpty && emptyKey != null
        ? CodeWorkspaceText.t(context, emptyKey!)
        : content;
    return PanelCard(
      title: Text(title),
      child: SelectableText(
        value,
        style: TextStyle(
          color: error ? AppTheme.error : AppTheme.textSecondary,
          height: 1.5,
          fontFamily: code ? 'monospace' : null,
          fontSize: code ? 12 : null,
        ),
      ),
    );
  }
}

class _Meta extends StatelessWidget {
  const _Meta({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: const Color(0xFFF1F5F9),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text('$label: ${value.isEmpty ? '-' : value}'),
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

String _statusLabel(BuildContext context, String status) {
  final key = switch (status) {
    'completed' => 'recovery.statusCompleted',
    'failed' => 'recovery.statusFailed',
    'cancelled' || 'stopped' => 'recovery.statusStopped',
    'running' => 'recovery.statusRunning',
    'awaiting_approval' ||
    'pending_approval' => 'recovery.statusAwaitingApproval',
    'queued' => 'recovery.statusQueued',
    _ => 'recovery.statusUnknown',
  };
  return CodeWorkspaceText.t(context, key);
}

String _formatTime(DateTime value) {
  final local = value.toLocal();
  String two(int number) => number.toString().padLeft(2, '0');
  return '${local.year}-${two(local.month)}-${two(local.day)} '
      '${two(local.hour)}:${two(local.minute)}:${two(local.second)}';
}
