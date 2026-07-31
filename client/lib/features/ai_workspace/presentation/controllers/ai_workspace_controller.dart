import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/ai_workspace_repository.dart';
import '../../models/ai_dev_session.dart';
import '../../models/chat_message.dart';

final aiWorkspaceRepositoryProvider = Provider<AiWorkspaceRepository>((ref) {
  return AiWorkspaceRepository(ApiClient());
});

class AiWorkspaceState {
  final bool isLoading;
  final bool isSending;
  final bool isActionLoading;
  final String? errorMessage;
  final List<CodeProject> projects;
  final List<CodeExecutor> executors;
  final List<AiDevSession> sessions;
  final List<ChatMessage> chatHistory;
  final AiDevSession? currentSession;
  final AiTaskSummary? currentTask;
  final AiInstruction? latestInstruction;
  final AiApproval? pendingApproval;
  final String currentStage;
  final String recentOutput;
  final List<AiPreview> previews;
  final List<AiTimelineEvent> timelineEvents;
  final String errorSummary;
  final List<String> changedFiles;

  const AiWorkspaceState({
    this.isLoading = false,
    this.isSending = false,
    this.isActionLoading = false,
    this.errorMessage,
    this.projects = const [],
    this.executors = const [],
    this.sessions = const [],
    this.chatHistory = const [],
    this.currentSession,
    this.currentTask,
    this.latestInstruction,
    this.pendingApproval,
    this.currentStage = 'idle',
    this.recentOutput = '',
    this.previews = const [],
    this.timelineEvents = const [],
    this.errorSummary = '',
    this.changedFiles = const [],
  });

  String? get selectedWorkspacePath => currentSession?.workDir;
  bool get isAiThinking => isSending || _activeStages.contains(currentStage);

  AiWorkspaceState copyWith({
    bool? isLoading,
    bool? isSending,
    bool? isActionLoading,
    String? errorMessage,
    List<CodeProject>? projects,
    List<CodeExecutor>? executors,
    List<AiDevSession>? sessions,
    List<ChatMessage>? chatHistory,
    AiDevSession? currentSession,
    AiTaskSummary? currentTask,
    AiInstruction? latestInstruction,
    AiApproval? pendingApproval,
    String? currentStage,
    String? recentOutput,
    List<AiPreview>? previews,
    List<AiTimelineEvent>? timelineEvents,
    String? errorSummary,
    List<String>? changedFiles,
    bool clearError = false,
    bool clearSession = false,
    bool clearTask = false,
    bool clearInstruction = false,
    bool clearApproval = false,
  }) {
    return AiWorkspaceState(
      isLoading: isLoading ?? this.isLoading,
      isSending: isSending ?? this.isSending,
      isActionLoading: isActionLoading ?? this.isActionLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      projects: projects ?? this.projects,
      executors: executors ?? this.executors,
      sessions: sessions ?? this.sessions,
      chatHistory: chatHistory ?? this.chatHistory,
      currentSession: clearSession
          ? null
          : (currentSession ?? this.currentSession),
      currentTask: clearTask ? null : (currentTask ?? this.currentTask),
      latestInstruction: clearInstruction
          ? null
          : (latestInstruction ?? this.latestInstruction),
      pendingApproval: clearApproval
          ? null
          : (pendingApproval ?? this.pendingApproval),
      currentStage: currentStage ?? this.currentStage,
      recentOutput: recentOutput ?? this.recentOutput,
      previews: previews ?? this.previews,
      timelineEvents: timelineEvents ?? this.timelineEvents,
      errorSummary: errorSummary ?? this.errorSummary,
      changedFiles: changedFiles ?? this.changedFiles,
    );
  }
}

const _activeStages = {
  'interactive',
  'task_ready',
  'instruction_queued',
  'executing',
  'running',
};

final aiWorkspaceControllerProvider =
    NotifierProvider<AiWorkspaceController, AiWorkspaceState>(
      AiWorkspaceController.new,
    );

class AiWorkspaceController extends Notifier<AiWorkspaceState> {
  late AiWorkspaceRepository _repo;
  Timer? _refreshTimer;
  bool _refreshing = false;

  @override
  AiWorkspaceState build() {
    _repo = ref.watch(aiWorkspaceRepositoryProvider);
    ref.onDispose(() => _refreshTimer?.cancel());
    Future.microtask(loadWorkspace);
    return AiWorkspaceState(chatHistory: [_welcomeMessage()]);
  }

  Future<void> loadWorkspace() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final results = await Future.wait([
        _repo.getProjects(),
        _repo.getExecutors(),
        _repo.getSessions(),
      ]);
      final projects = results[0] as List<CodeProject>;
      final executors = results[1] as List<CodeExecutor>;
      final sessions = results[2] as List<AiDevSession>;
      state = state.copyWith(
        isLoading: false,
        projects: projects,
        executors: executors,
        sessions: sessions,
      );
      if (state.currentSession == null && sessions.isNotEmpty) {
        await selectSession(sessions.first);
      }
    } catch (error) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: '开发工作区加载失败：$error',
      );
    }
  }

  Future<void> createSession({
    required int projectId,
    required String executorId,
    required String approvalPolicy,
    String title = '',
  }) async {
    state = state.copyWith(isActionLoading: true, clearError: true);
    try {
      final session = await _repo.createSession(
        projectId: projectId,
        executorId: executorId,
        approvalPolicy: approvalPolicy,
        title: title.trim(),
      );
      state = state.copyWith(
        isActionLoading: false,
        sessions: [session, ...state.sessions],
      );
      await selectSession(session);
    } catch (error) {
      state = state.copyWith(
        isActionLoading: false,
        errorMessage: '创建开发会话失败：$error',
      );
      rethrow;
    }
  }

  Future<void> selectSession(AiDevSession session) async {
    _refreshTimer?.cancel();
    state = state.copyWith(
      currentSession: session,
      currentStage: session.currentStage,
      chatHistory: [_welcomeMessage()],
      previews: const [],
      timelineEvents: const [],
      errorSummary: '',
      changedFiles: const [],
      clearTask: true,
      clearInstruction: true,
      clearApproval: true,
    );
    await refreshCurrentSession(showLoading: true);
    _refreshTimer = Timer.periodic(
      const Duration(seconds: 3),
      (_) => refreshCurrentSession(),
    );
  }

  Future<void> refreshCurrentSession({bool showLoading = false}) async {
    final sessionId = state.currentSession?.id;
    if (sessionId == null || _refreshing) return;
    _refreshing = true;
    if (showLoading) {
      state = state.copyWith(isLoading: true, clearError: true);
    }
    try {
      final sessionState = await _repo.getSessionState(sessionId);
      final messages = sessionState.recentMessages.isEmpty
          ? state.chatHistory
          : sessionState.recentMessages;
      final sessions = state.sessions
          .map(
            (item) => item.id == sessionState.session.id
                ? sessionState.session
                : item,
          )
          .toList();
      state = state.copyWith(
        isLoading: false,
        clearError: true,
        sessions: sessions,
        chatHistory: messages,
        currentSession: sessionState.session,
        currentTask: sessionState.currentTask,
        latestInstruction: sessionState.latestInstruction,
        pendingApproval: sessionState.pendingApproval,
        currentStage: sessionState.currentStage,
        recentOutput: sessionState.recentOutput,
        previews: sessionState.previews,
        timelineEvents: sessionState.timelineEvents,
        errorSummary: sessionState.errorSummary,
        changedFiles: sessionState.changedFiles,
        clearTask: sessionState.currentTask == null,
        clearInstruction: sessionState.latestInstruction == null,
        clearApproval: sessionState.pendingApproval == null,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: '会话状态刷新失败：$error');
    } finally {
      _refreshing = false;
    }
  }

  Future<void> sendMessage(String text) async {
    final content = text.trim();
    if (content.isEmpty || state.isSending) return;
    final session = state.currentSession;
    if (session == null) {
      state = state.copyWith(errorMessage: '请先创建或选择一个开发会话');
      return;
    }
    final userMessage = ChatMessage(
      id: 'local-${DateTime.now().microsecondsSinceEpoch}',
      text: content,
      isUser: true,
      timestamp: DateTime.now(),
    );
    state = state.copyWith(
      isSending: true,
      clearError: true,
      chatHistory: [...state.chatHistory, userMessage],
    );
    try {
      final result = await _repo.sendAiCommand(
        sessionId: session.id,
        command: content,
      );
      state = state.copyWith(
        currentSession: result.session,
        currentTask: result.task,
        latestInstruction: result.instruction,
        pendingApproval: result.approval,
        currentStage: result.approval == null
            ? 'instruction_queued'
            : 'awaiting_approval',
        clearApproval: result.approval == null,
      );
      await refreshCurrentSession();
    } catch (error) {
      state = state.copyWith(errorMessage: '指令发送失败：$error');
    } finally {
      state = state.copyWith(isSending: false);
    }
  }

  Future<void> decideApproval({
    required bool approved,
    String reason = '',
  }) async {
    final approval = state.pendingApproval;
    if (approval == null || state.isActionLoading) return;
    state = state.copyWith(isActionLoading: true, clearError: true);
    try {
      if (approved) {
        await _repo.approveApproval(approval.id, reason: reason);
      } else {
        await _repo.rejectApproval(approval.id, reason: reason);
      }
      state = state.copyWith(isActionLoading: false, clearApproval: true);
      await refreshCurrentSession();
    } catch (error) {
      state = state.copyWith(
        isActionLoading: false,
        errorMessage: '审批处理失败：$error',
      );
    }
  }

  Future<bool> stopCurrentSession() async {
    final sessionId = state.currentSession?.id;
    if (sessionId == null || state.isActionLoading) return false;
    state = state.copyWith(isActionLoading: true, clearError: true);
    try {
      await _repo.stopSession(sessionId);
      state = state.copyWith(isActionLoading: false);
      await refreshCurrentSession();
      return true;
    } catch (error) {
      state = state.copyWith(
        isActionLoading: false,
        errorMessage: error.toString(),
      );
      return false;
    }
  }

  void clearError() {
    state = state.copyWith(clearError: true);
  }

  ChatMessage _welcomeMessage() {
    return ChatMessage(
      id: 'welcome',
      text: '选择已有会话，或从开发项目创建会话，然后发送开发指令。',
      isUser: false,
      timestamp: DateTime.now(),
    );
  }
}
