import 'package:flutter/foundation.dart';

import '../../data/ai_workspace_repository.dart';
import '../../models/code_session_recovery.dart';

class CodeSessionRecoveryState {
  final bool isLoading;
  final bool isLoadingMore;
  final String? errorMessage;
  final List<CodeHistoryMessage> messages;
  final List<CodeExecutionRun> runs;
  final int totalRuns;
  final int page;
  final CodeSessionInitialization? initialization;
  final int? retryingInstructionId;
  final Set<int> retriedInstructionIds;
  final bool isRetryingInitialization;

  const CodeSessionRecoveryState({
    this.isLoading = false,
    this.isLoadingMore = false,
    this.errorMessage,
    this.messages = const [],
    this.runs = const [],
    this.totalRuns = 0,
    this.page = 0,
    this.initialization,
    this.retryingInstructionId,
    this.retriedInstructionIds = const {},
    this.isRetryingInitialization = false,
  });

  bool get canLoadMore => runs.length < totalRuns;

  CodeSessionRecoveryState copyWith({
    bool? isLoading,
    bool? isLoadingMore,
    String? errorMessage,
    List<CodeHistoryMessage>? messages,
    List<CodeExecutionRun>? runs,
    int? totalRuns,
    int? page,
    CodeSessionInitialization? initialization,
    int? retryingInstructionId,
    Set<int>? retriedInstructionIds,
    bool? isRetryingInitialization,
    bool clearError = false,
    bool clearRetryingInstruction = false,
  }) {
    return CodeSessionRecoveryState(
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      messages: messages ?? this.messages,
      runs: runs ?? this.runs,
      totalRuns: totalRuns ?? this.totalRuns,
      page: page ?? this.page,
      initialization: initialization ?? this.initialization,
      retryingInstructionId: clearRetryingInstruction
          ? null
          : (retryingInstructionId ?? this.retryingInstructionId),
      retriedInstructionIds:
          retriedInstructionIds ?? this.retriedInstructionIds,
      isRetryingInitialization:
          isRetryingInitialization ?? this.isRetryingInitialization,
    );
  }
}

class CodeSessionRecoveryController extends ChangeNotifier {
  CodeSessionRecoveryController({
    required AiWorkspaceRepository repository,
    required this.sessionId,
    this.pageSize = 20,
  }) : _repository = repository;

  final AiWorkspaceRepository _repository;
  final int sessionId;
  final int pageSize;
  CodeSessionRecoveryState _state = const CodeSessionRecoveryState();
  int _requestVersion = 0;
  bool _disposed = false;

  CodeSessionRecoveryState get state => _state;

  Future<void> load() async {
    final request = ++_requestVersion;
    _setState(
      _state.copyWith(isLoading: true, isLoadingMore: false, clearError: true),
    );
    try {
      final results = await Future.wait([
        _repository.getSessionHistory(sessionId, page: 1, limit: pageSize),
        _repository.getSessionInitialization(sessionId),
      ]);
      if (request != _requestVersion) return;
      final history = results[0] as CodeSessionHistory;
      _setState(
        _state.copyWith(
          isLoading: false,
          messages: history.messages,
          runs: history.runs,
          totalRuns: history.total,
          page: history.page,
          initialization: results[1] as CodeSessionInitialization,
          clearError: true,
        ),
      );
    } catch (error) {
      if (request != _requestVersion) return;
      _setState(
        _state.copyWith(isLoading: false, errorMessage: error.toString()),
      );
    }
  }

  Future<void> loadMore() async {
    if (_state.isLoading || _state.isLoadingMore || !_state.canLoadMore) return;
    final request = _requestVersion;
    final nextPage = _state.page + 1;
    _setState(_state.copyWith(isLoadingMore: true, clearError: true));
    try {
      final history = await _repository.getSessionHistory(
        sessionId,
        page: nextPage,
        limit: pageSize,
      );
      if (request != _requestVersion) return;
      final existingIds = _state.runs.map((run) => run.id).toSet();
      _setState(
        _state.copyWith(
          isLoadingMore: false,
          runs: [
            ..._state.runs,
            ...history.runs.where((run) => !existingIds.contains(run.id)),
          ],
          totalRuns: history.total,
          page: history.page,
          clearError: true,
        ),
      );
    } catch (error) {
      if (request != _requestVersion) return;
      _setState(
        _state.copyWith(isLoadingMore: false, errorMessage: error.toString()),
      );
    }
  }

  Future<bool> retryInstruction(CodeExecutionRun run) async {
    if (!run.canRetry ||
        _state.retriedInstructionIds.contains(run.instructionId) ||
        _state.retryingInstructionId != null) {
      return false;
    }
    _setState(
      _state.copyWith(
        retryingInstructionId: run.instructionId,
        clearError: true,
      ),
    );
    try {
      await _repository.retryInstruction(run.instructionId);
      _setState(
        _state.copyWith(
          retriedInstructionIds: {
            ..._state.retriedInstructionIds,
            run.instructionId,
          },
          clearRetryingInstruction: true,
        ),
      );
      return true;
    } catch (error) {
      _setState(
        _state.copyWith(
          errorMessage: error.toString(),
          clearRetryingInstruction: true,
        ),
      );
      return false;
    }
  }

  Future<bool> retryInitialization() async {
    if (_state.initialization?.canRetry != true ||
        _state.isRetryingInitialization) {
      return false;
    }
    _setState(
      _state.copyWith(isRetryingInitialization: true, clearError: true),
    );
    try {
      final initialization = await _repository.retrySessionInitialization(
        sessionId,
      );
      _setState(
        _state.copyWith(
          initialization: initialization,
          isRetryingInitialization: false,
        ),
      );
      return true;
    } catch (error) {
      _setState(
        _state.copyWith(
          isRetryingInitialization: false,
          errorMessage: error.toString(),
        ),
      );
      return false;
    }
  }

  void _setState(CodeSessionRecoveryState next) {
    if (_disposed) return;
    _state = next;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _requestVersion++;
    super.dispose();
  }
}
