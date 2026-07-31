import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../controllers/ai_workspace_controller.dart';
import '../code_workspace_text.dart';
import '../widgets/ai_approval_prompt.dart';
import '../widgets/ai_chat_message_item.dart';
import '../widgets/ai_preview_strip.dart';
import '../widgets/ai_session_overview_card.dart';
import '../widgets/ai_timeline_panel.dart';
import 'ai_preview_detail_screen.dart';
import 'ai_preview_list_screen.dart';
import 'code_session_sheet.dart';
import 'code_workspace_files_screen.dart';

class AiChatScreen extends ConsumerStatefulWidget {
  const AiChatScreen({super.key});

  @override
  ConsumerState<AiChatScreen> createState() => _AiChatScreenState();
}

class _AiChatScreenState extends ConsumerState<AiChatScreen> {
  final _inputController = TextEditingController();
  final _scrollController = ScrollController();
  final _focusNode = FocusNode();

  @override
  void dispose() {
    _inputController.dispose();
    _scrollController.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _sendMessage() {
    final text = _inputController.text.trim();
    if (text.isEmpty) return;
    _inputController.clear();
    ref.read(aiWorkspaceControllerProvider.notifier).sendMessage(text);
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 250),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _openSessionSheet() {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: false,
      builder: (_) => const CodeSessionSheet(),
    );
  }

  void _openFiles() {
    final session = ref.read(aiWorkspaceControllerProvider).currentSession;
    if (session == null) return;
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeWorkspaceFilesScreen(
          sessionId: session.id,
          sessionTitle: session.title.isEmpty
              ? '开发 #${session.id}'
              : session.title,
        ),
      ),
    );
  }

  Future<void> _confirmStop() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(CodeWorkspaceText.t(dialogContext, 'stop.title')),
        content: Text(CodeWorkspaceText.t(dialogContext, 'stop.description')),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(CodeWorkspaceText.t(dialogContext, 'stop.cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(CodeWorkspaceText.t(dialogContext, 'stop.confirm')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final stopped = await ref
        .read(aiWorkspaceControllerProvider.notifier)
        .stopCurrentSession();
    if (!mounted || !stopped) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(CodeWorkspaceText.t(context, 'stop.success'))),
    );
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(aiWorkspaceControllerProvider);
    final controller = ref.read(aiWorkspaceControllerProvider.notifier);
    final session = state.currentSession;

    ref.listen(aiWorkspaceControllerProvider, (previous, next) {
      if (previous?.chatHistory.length != next.chatHistory.length) {
        _scrollToBottom();
      }
    });

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF1E293B),
        foregroundColor: Colors.white,
        title: const Text(
          'GoPanel 开发',
          style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
        ),
        actions: [
          if (session != null)
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'files.title'),
              onPressed: _openFiles,
              icon: const Icon(Icons.folder_open_rounded),
            ),
          IconButton(
            tooltip: '刷新状态',
            onPressed: session == null
                ? controller.loadWorkspace
                : () => controller.refreshCurrentSession(showLoading: true),
            icon: const Icon(Icons.refresh_rounded),
          ),
          IconButton(
            tooltip: '选择或创建会话',
            onPressed: _openSessionSheet,
            icon: const Icon(Icons.add_to_photos_outlined),
          ),
        ],
      ),
      body: Column(
        children: [
          if (state.errorMessage != null)
            _ErrorBanner(
              message: state.errorMessage!,
              onRetry: session == null
                  ? controller.loadWorkspace
                  : () => controller.refreshCurrentSession(showLoading: true),
              onDismiss: controller.clearError,
            ),
          if (state.isLoading) const LinearProgressIndicator(minHeight: 2),
          Expanded(
            child: session == null
                ? _EmptyWorkspace(onOpenSessions: _openSessionSheet)
                : ListView(
                    controller: _scrollController,
                    children: [
                      AiSessionOverviewCard(
                        workspace: session.workDir,
                        currentSession: session,
                        currentTask: state.currentTask,
                        stage: state.currentStage,
                      ),
                      if (state.pendingApproval != null)
                        AiApprovalPrompt(
                          approval: state.pendingApproval!,
                          loading: state.isActionLoading,
                          onDecision: (approved, reason) =>
                              controller.decideApproval(
                                approved: approved,
                                reason: reason,
                              ),
                        ),
                      AiTimelinePanel(
                        timelineEvents: state.timelineEvents,
                        errorSummary: state.errorSummary,
                        changedFiles: state.changedFiles,
                      ),
                      AiPreviewStrip(
                        previews: state.previews,
                        onOpenPreview: (preview) {
                          Navigator.of(context).push(
                            MaterialPageRoute(
                              builder: (_) =>
                                  AiPreviewDetailScreen(preview: preview),
                            ),
                          );
                        },
                        onViewAll: () {
                          Navigator.of(context).push(
                            MaterialPageRoute(
                              builder: (_) => AiPreviewListScreen(
                                sessionId: session.id,
                                title: '会话预览',
                              ),
                            ),
                          );
                        },
                      ),
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 4, 16, 16),
                        child: Column(
                          children: [
                            for (
                              var index = 0;
                              index < state.chatHistory.length;
                              index++
                            ) ...[
                              if (index > 0) const SizedBox(height: 16),
                              AiChatMessageItem(
                                message: state.chatHistory[index],
                              ),
                            ],
                          ],
                        ),
                      ),
                    ],
                  ),
          ),
          if (session != null && state.isAiThinking)
            _ExecutionIndicator(
              isStopping: state.isActionLoading,
              onStop: _confirmStop,
            ),
          _CommandComposer(
            controller: _inputController,
            focusNode: _focusNode,
            enabled: session != null && !state.isSending,
            onSend: _sendMessage,
          ),
        ],
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({
    required this.message,
    required this.onRetry,
    required this.onDismiss,
  });

  final String message;
  final VoidCallback onRetry;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    return MaterialBanner(
      backgroundColor: const Color(0xFF451A1A),
      content: Text(message, style: const TextStyle(color: Colors.white)),
      leading: const Icon(Icons.error_outline_rounded, color: Colors.redAccent),
      actions: [
        TextButton(onPressed: onDismiss, child: const Text('关闭')),
        TextButton(onPressed: onRetry, child: const Text('重试')),
      ],
    );
  }
}

class _EmptyWorkspace extends StatelessWidget {
  const _EmptyWorkspace({required this.onOpenSessions});

  final VoidCallback onOpenSessions;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.terminal_rounded, color: Colors.white54, size: 52),
            const SizedBox(height: 16),
            const Text(
              '开始一个开发会话',
              style: TextStyle(
                color: Colors.white,
                fontSize: 18,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              '绑定已有项目与服务器执行器，在手机上发指令、看过程、开预览并处理审批。',
              textAlign: TextAlign.center,
              style: TextStyle(color: Colors.white60, height: 1.5),
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: onOpenSessions,
              icon: const Icon(Icons.add_rounded),
              label: const Text('选择或创建会话'),
            ),
          ],
        ),
      ),
    );
  }
}

class _ExecutionIndicator extends StatelessWidget {
  const _ExecutionIndicator({required this.isStopping, required this.onStop});

  final bool isStopping;
  final VoidCallback onStop;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 8, 8),
      child: Row(
        children: [
          const SizedBox.square(
            dimension: 16,
            child: CircularProgressIndicator(
              color: Colors.greenAccent,
              strokeWidth: 2,
            ),
          ),
          const SizedBox(width: 10),
          const Expanded(
            child: Text(
              '开发任务正在执行，状态会自动刷新',
              style: TextStyle(color: Colors.greenAccent),
            ),
          ),
          TextButton.icon(
            onPressed: isStopping ? null : onStop,
            icon: isStopping
                ? const SizedBox.square(
                    dimension: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.stop_circle_outlined, size: 18),
            label: Text(CodeWorkspaceText.t(context, 'action.stop')),
          ),
        ],
      ),
    );
  }
}

class _CommandComposer extends StatelessWidget {
  const _CommandComposer({
    required this.controller,
    required this.focusNode,
    required this.enabled,
    required this.onSend,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final bool enabled;
  final VoidCallback onSend;

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF1E293B),
      padding: EdgeInsets.fromLTRB(
        16,
        10,
        8,
        MediaQuery.paddingOf(context).bottom + 10,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          const Padding(
            padding: EdgeInsets.only(bottom: 12, right: 8),
            child: Text(
              '\$',
              style: TextStyle(
                color: Colors.greenAccent,
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          Expanded(
            child: TextField(
              controller: controller,
              focusNode: focusNode,
              enabled: enabled,
              maxLines: 5,
              minLines: 1,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => onSend(),
              style: const TextStyle(color: Colors.white),
              decoration: InputDecoration(
                hintText: enabled ? '输入开发指令或补充要求...' : '请先选择开发会话',
                hintStyle: const TextStyle(color: Colors.white38),
                border: InputBorder.none,
              ),
            ),
          ),
          IconButton(
            onPressed: enabled ? onSend : null,
            icon: const Icon(Icons.send_rounded),
            color: Colors.blueAccent,
          ),
        ],
      ),
    );
  }
}
