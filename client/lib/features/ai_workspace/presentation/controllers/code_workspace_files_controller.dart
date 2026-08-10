import 'package:flutter/foundation.dart';

import '../../data/ai_workspace_repository.dart';
import '../../models/code_workspace_file.dart';

class CodeWorkspaceFilesState {
  final bool isLoading;
  final String? errorMessage;
  final String currentPath;
  final List<CodeStructureEntry> entries;
  final CodeSessionFile? openFile;
  final bool truncated;
  final String draftContent;
  final bool isSaving;

  const CodeWorkspaceFilesState({
    this.isLoading = false,
    this.errorMessage,
    this.currentPath = '',
    this.entries = const [],
    this.openFile,
    this.truncated = false,
    this.draftContent = '',
    this.isSaving = false,
  });

  bool get isDirty => openFile != null && draftContent != openFile!.content;

  CodeWorkspaceFilesState copyWith({
    bool? isLoading,
    String? errorMessage,
    String? currentPath,
    List<CodeStructureEntry>? entries,
    CodeSessionFile? openFile,
    bool? truncated,
    String? draftContent,
    bool? isSaving,
    bool clearError = false,
    bool closeFile = false,
  }) {
    return CodeWorkspaceFilesState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      currentPath: currentPath ?? this.currentPath,
      entries: entries ?? this.entries,
      openFile: closeFile ? null : (openFile ?? this.openFile),
      truncated: truncated ?? this.truncated,
      draftContent: closeFile ? '' : (draftContent ?? this.draftContent),
      isSaving: isSaving ?? this.isSaving,
    );
  }
}

class CodeWorkspaceFilesController extends ChangeNotifier {
  CodeWorkspaceFilesController({
    required AiWorkspaceRepository repository,
    required this.sessionId,
  }) : _repository = repository;

  final AiWorkspaceRepository _repository;
  final int sessionId;
  CodeWorkspaceFilesState _state = const CodeWorkspaceFilesState();
  int _requestVersion = 0;
  bool _disposed = false;

  CodeWorkspaceFilesState get state => _state;

  Future<void> loadDirectory([String path = '']) async {
    final requestVersion = ++_requestVersion;
    _setState(
      _state.copyWith(isLoading: true, clearError: true, closeFile: true),
    );
    try {
      final result = await _repository.getSessionStructure(
        sessionId,
        path: path,
      );
      if (requestVersion != _requestVersion) return;
      _setState(
        _state.copyWith(
          isLoading: false,
          currentPath: result.path,
          entries: result.entries,
          truncated: result.truncated,
          clearError: true,
          closeFile: true,
        ),
      );
    } catch (error) {
      if (requestVersion != _requestVersion) return;
      _setState(
        _state.copyWith(
          isLoading: false,
          errorMessage: error.toString(),
          closeFile: true,
        ),
      );
    }
  }

  Future<void> openEntry(CodeStructureEntry entry) async {
    if (entry.isDirectory) {
      await loadDirectory(entry.path);
      return;
    }
    final requestVersion = ++_requestVersion;
    _setState(_state.copyWith(isLoading: true, clearError: true));
    try {
      final file = await _repository.getSessionFile(sessionId, entry.path);
      if (requestVersion != _requestVersion) return;
      _setState(
        _state.copyWith(
          isLoading: false,
          openFile: file,
          draftContent: file.content,
          clearError: true,
        ),
      );
    } catch (error) {
      if (requestVersion != _requestVersion) return;
      _setState(
        _state.copyWith(isLoading: false, errorMessage: error.toString()),
      );
    }
  }

  Future<void> refresh() {
    final file = _state.openFile;
    if (file != null) {
      return openEntry(
        CodeStructureEntry(
          name: file.path.split('/').last,
          path: file.path,
          isDirectory: false,
          extension: file.extension,
        ),
      );
    }
    return loadDirectory(_state.currentPath);
  }

  void updateDraft(String content) {
    if (_state.openFile == null || content == _state.draftContent) return;
    _setState(_state.copyWith(draftContent: content, clearError: true));
  }

  Future<bool> saveOpenFile() async {
    final file = _state.openFile;
    if (file == null || !_state.isDirty || _state.isSaving) return false;
    _setState(_state.copyWith(isSaving: true, clearError: true));
    try {
      final saved = await _repository.saveSessionFile(
        sessionId: sessionId,
        file: file,
        content: _state.draftContent,
      );
      _setState(
        _state.copyWith(
          openFile: saved,
          draftContent: saved.content,
          isSaving: false,
          clearError: true,
        ),
      );
      return true;
    } catch (error) {
      _setState(
        _state.copyWith(isSaving: false, errorMessage: error.toString()),
      );
      return false;
    }
  }

  void closeFile() {
    _requestVersion++;
    _setState(_state.copyWith(closeFile: true, clearError: true));
  }

  Future<void> openParent() {
    final segments = _state.currentPath
        .split('/')
        .where((segment) => segment.isNotEmpty)
        .toList();
    if (segments.isNotEmpty) segments.removeLast();
    return loadDirectory(segments.join('/'));
  }

  void _setState(CodeWorkspaceFilesState nextState) {
    if (_disposed) return;
    _state = nextState;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _requestVersion++;
    super.dispose();
  }
}
