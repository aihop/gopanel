import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/storage/storage_service.dart';
import '../../models/code_instruction_options.dart';
import '../controllers/ai_workspace_controller.dart';
import '../code_workspace_text.dart';
import '../widgets/ai_approval_prompt.dart';
import '../widgets/ai_chat_message_item.dart';
import '../widgets/ai_chat_state_widgets.dart';
import '../widgets/ai_preview_strip.dart';
import '../widgets/ai_session_overview_card.dart';
import '../widgets/ai_timeline_panel.dart';
import '../widgets/code_delivery_card.dart';
import '../widgets/code_instruction_composer.dart';
import 'ai_preview_detail_screen.dart';
import 'ai_preview_list_screen.dart';
import 'code_git_review_screen.dart';
import 'code_session_sheet.dart';
import 'code_session_recovery_screen.dart';
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
  Timer? _draftTimer;
  int? _draftSessionId;
  bool _restoringDraft = false;
  CodeInstructionOptions _instructionOptions = const CodeInstructionOptions();

  @override
  void initState() {
    super.initState();
    _inputController.addListener(_scheduleDraftSave);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _activateDraftSession(
        ref.read(aiWorkspaceControllerProvider).currentSession?.id,
      );
    });
  }

  @override
  void dispose() {
    _draftTimer?.cancel();
    final sessionId = _draftSessionId;
    if (sessionId != null) {
      unawaited(
        StorageService.saveCodeInstructionDraft(
          sessionId,
          _inputController.text,
        ),
      );
    }
    _inputController.removeListener(_scheduleDraftSave);
    _inputController.dispose();
    _scrollController.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  Future<void> _sendMessage() async {
    final text = _inputController.text.trim();
    if (text.isEmpty) return;
    final sessionId = _draftSessionId;
    final sent = await ref
        .read(aiWorkspaceControllerProvider.notifier)
        .sendMessage(text, options: _instructionOptions);
    if (!mounted || !sent || _draftSessionId != sessionId) return;
    _setInputText('');
    if (sessionId != null) {
      await StorageService.saveCodeInstructionDraft(sessionId, '');
    }
    _scrollToBottom();
  }

  void _scheduleDraftSave() {
    if (_restoringDraft || _draftSessionId == null) return;
    _draftTimer?.cancel();
    final sessionId = _draftSessionId!;
    _draftTimer = Timer(const Duration(milliseconds: 400), () {
      if (_draftSessionId != sessionId) return;
      unawaited(
        StorageService.saveCodeInstructionDraft(
          sessionId,
          _inputController.text,
        ),
      );
    });
  }

  void _activateDraftSession(int? sessionId) {
    if (_draftSessionId == sessionId) return;
    _draftTimer?.cancel();
    final previousSessionId = _draftSessionId;
    if (previousSessionId != null) {
      unawaited(
        StorageService.saveCodeInstructionDraft(
          previousSessionId,
          _inputController.text,
        ),
      );
    }
    _draftSessionId = sessionId;
    _setInputText(
      sessionId == null
          ? ''
          : StorageService.getCodeInstructionDraft(sessionId),
    );
  }

  void _setInputText(String text) {
    _restoringDraft = true;
    _inputController.value = TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
    );
    _restoringDraft = false;
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
              ? CodeWorkspaceText.format(context, 'chat.sessionFallback', {
                  'id': session.id,
                })
              : session.title,
        ),
      ),
    );
  }

  void _openGitReview() {
    final session = ref.read(aiWorkspaceControllerProvider).currentSession;
    if (session == null) return;
    Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => CodeGitReviewScreen(session: session)),
    );
  }

  void _openRecovery() {
    final session = ref.read(aiWorkspaceControllerProvider).currentSession;
    if (session == null) return;
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeSessionRecoveryScreen(
          sessionId: session.id,
          sessionTitle: session.title.isEmpty
              ? CodeWorkspaceText.format(context, 'chat.sessionFallback', {
                  'id': session.id,
                })
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

  Future<void> _confirmDelivery() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(
          CodeWorkspaceText.t(dialogContext, 'delivery.confirmTitle'),
        ),
        content: Text(
          CodeWorkspaceText.t(dialogContext, 'delivery.confirmDescription'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(CodeWorkspaceText.t(dialogContext, 'delivery.cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(CodeWorkspaceText.t(dialogContext, 'delivery.confirm')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final started = await ref
        .read(aiWorkspaceControllerProvider.notifier)
        .startDelivery();
    if (!mounted || !started) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(CodeWorkspaceText.t(context, 'delivery.started'))),
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
      if (previous?.currentSession?.id != next.currentSession?.id) {
        _activateDraftSession(next.currentSession?.id);
      }
    });

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF1E293B),
        foregroundColor: Colors.white,
        title: Text(
          CodeWorkspaceText.t(context, 'chat.title'),
          style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
        ),
        actions: [
          if (session != null)
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'recovery.title'),
              onPressed: _openRecovery,
              icon: const Icon(Icons.history_rounded),
            ),
          if (session != null)
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'chat.gitReview'),
              onPressed: _openGitReview,
              icon: const Icon(Icons.source_outlined),
            ),
          if (session != null)
            IconButton(
              tooltip: CodeWorkspaceText.t(context, 'files.title'),
              onPressed: _openFiles,
              icon: const Icon(Icons.folder_open_rounded),
            ),
          IconButton(
            tooltip: CodeWorkspaceText.t(context, 'chat.refresh'),
            onPressed: session == null
                ? controller.loadWorkspace
                : () => controller.refreshCurrentSession(showLoading: true),
            icon: const Icon(Icons.refresh_rounded),
          ),
          IconButton(
            tooltip: CodeWorkspaceText.t(context, 'chat.chooseSession'),
            onPressed: _openSessionSheet,
            icon: const Icon(Icons.add_to_photos_outlined),
          ),
        ],
      ),
      body: Column(
        children: [
          if (state.errorMessage != null)
            AiChatErrorBanner(
              message: state.errorMessage!,
              onRetry: session == null
                  ? controller.loadWorkspace
                  : () => controller.refreshCurrentSession(showLoading: true),
              onDismiss: controller.clearError,
            ),
          if (state.isLoading) const LinearProgressIndicator(minHeight: 2),
          Expanded(
            child: session == null
                ? AiChatEmptyWorkspace(onOpenSessions: _openSessionSheet)
                : ListView(
                    controller: _scrollController,
                    children: [
                      AiSessionOverviewCard(
                        workspace: session.workDir,
                        currentSession: session,
                        currentTask: state.currentTask,
                        stage: state.currentStage,
                      ),
                      CodeDeliveryCard(
                        session: session,
                        delivery: state.delivery,
                        loading: state.isDeliveryLoading,
                        errorMessage: state.deliveryError,
                        onStart: _confirmDelivery,
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
                                title: CodeWorkspaceText.t(
                                  context,
                                  'chat.previewTitle',
                                ),
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
            AiChatExecutionIndicator(
              isStopping: state.isActionLoading,
              onStop: _confirmStop,
            ),
          CodeInstructionComposer(
            controller: _inputController,
            focusNode: _focusNode,
            enabled:
                session != null &&
                !state.isSending &&
                !state.isDevelopmentClosed,
            closed: state.isDevelopmentClosed,
            options: _instructionOptions,
            onOptionsChanged: (options) {
              setState(() => _instructionOptions = options);
            },
            onSend: _sendMessage,
          ),
        ],
      ),
    );
  }
}
