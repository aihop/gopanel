import 'package:flutter/foundation.dart';

import '../../data/ai_workspace_repository.dart';
import '../../models/code_execution_run_detail.dart';

class CodeExecutionRunState {
  final bool isLoading;
  final String? errorMessage;
  final CodeExecutionRunDetail? run;

  const CodeExecutionRunState({
    this.isLoading = false,
    this.errorMessage,
    this.run,
  });
}

class CodeExecutionRunController extends ChangeNotifier {
  CodeExecutionRunController({
    required AiWorkspaceRepository repository,
    required this.runId,
  }) : _repository = repository;

  final AiWorkspaceRepository _repository;
  final int runId;
  CodeExecutionRunState _state = const CodeExecutionRunState();
  int _requestVersion = 0;
  bool _disposed = false;

  CodeExecutionRunState get state => _state;

  Future<void> load() async {
    final request = ++_requestVersion;
    _setState(CodeExecutionRunState(isLoading: true, run: _state.run));
    try {
      final run = await _repository.getExecutionRun(runId);
      if (request != _requestVersion) return;
      _setState(CodeExecutionRunState(run: run));
    } catch (error) {
      if (request != _requestVersion) return;
      _setState(
        CodeExecutionRunState(errorMessage: error.toString(), run: _state.run),
      );
    }
  }

  void _setState(CodeExecutionRunState next) {
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
