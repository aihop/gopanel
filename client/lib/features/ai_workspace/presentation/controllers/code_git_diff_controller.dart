import 'package:flutter/foundation.dart';

import '../../data/ai_workspace_repository.dart';
import '../../models/code_git_review.dart';

class CodeGitDiffState {
  final bool isLoading;
  final String? errorMessage;
  final CodeGitDiff? diff;

  const CodeGitDiffState({
    this.isLoading = false,
    this.errorMessage,
    this.diff,
  });
}

class CodeGitDiffController extends ChangeNotifier {
  CodeGitDiffController({
    required AiWorkspaceRepository repository,
    required this.sessionId,
    required this.repositoryId,
    required this.path,
    required this.kind,
  }) : _repository = repository;

  final AiWorkspaceRepository _repository;
  final int sessionId;
  final String repositoryId;
  final String path;
  final String kind;
  CodeGitDiffState _state = const CodeGitDiffState();
  int _requestVersion = 0;
  bool _disposed = false;

  CodeGitDiffState get state => _state;

  Future<void> load() async {
    final request = ++_requestVersion;
    _setState(CodeGitDiffState(isLoading: true, diff: _state.diff));
    try {
      final diff = await _repository.getGitDiff(
        sessionId: sessionId,
        repositoryId: repositoryId,
        path: path,
        kind: kind,
      );
      if (request != _requestVersion) return;
      _setState(CodeGitDiffState(diff: diff));
    } catch (error) {
      if (request != _requestVersion) return;
      _setState(
        CodeGitDiffState(errorMessage: error.toString(), diff: _state.diff),
      );
    }
  }

  void _setState(CodeGitDiffState next) {
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
