import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/code_workspace_file.dart';
import '../code_workspace_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_workspace_files_controller.dart';
import '../widgets/code_workspace_editor.dart';

class CodeWorkspaceFilesScreen extends ConsumerStatefulWidget {
  const CodeWorkspaceFilesScreen({
    super.key,
    required this.sessionId,
    required this.sessionTitle,
  });

  final int sessionId;
  final String sessionTitle;

  @override
  ConsumerState<CodeWorkspaceFilesScreen> createState() =>
      _CodeWorkspaceFilesScreenState();
}

class _CodeWorkspaceFilesScreenState
    extends ConsumerState<CodeWorkspaceFilesScreen> {
  late final CodeWorkspaceFilesController _controller;

  @override
  void initState() {
    super.initState();
    _controller = CodeWorkspaceFilesController(
      repository: ref.read(aiWorkspaceRepositoryProvider),
      sessionId: widget.sessionId,
    );
    _controller.loadDirectory();
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
        final file = state.openFile;
        return PopScope<void>(
          canPop: file == null,
          onPopInvokedWithResult: (didPop, _) {
            if (!didPop) _handleBack();
          },
          child: Scaffold(
            backgroundColor: const Color(0xFF0F172A),
            appBar: AppBar(
              backgroundColor: const Color(0xFF1E293B),
              foregroundColor: Colors.white,
              titleSpacing: 0,
              title: _ScreenTitle(
                title: file == null
                    ? CodeWorkspaceText.t(context, 'files.title')
                    : file.path.split('/').last,
                subtitle: widget.sessionTitle,
              ),
              leading: IconButton(
                tooltip: CodeWorkspaceText.t(context, 'action.back'),
                onPressed: _handleBack,
                icon: const Icon(Icons.arrow_back_rounded),
              ),
              actions: [
                if (file != null)
                  IconButton(
                    tooltip: CodeWorkspaceText.t(context, 'files.copy'),
                    onPressed: () => _copyFile(context, file),
                    icon: const Icon(Icons.content_copy_rounded),
                  ),
                IconButton(
                  tooltip: CodeWorkspaceText.t(context, 'action.refresh'),
                  onPressed: state.isLoading || state.isSaving
                      ? null
                      : _refresh,
                  icon: const Icon(Icons.refresh_rounded),
                ),
              ],
            ),
            body: Column(
              children: [
                if (state.isLoading)
                  const LinearProgressIndicator(minHeight: 2),
                if (state.errorMessage != null)
                  _FilesErrorBanner(
                    message: state.errorMessage!,
                    onRetry: file != null && state.isDirty ? _save : _refresh,
                  ),
                Expanded(
                  child: file == null
                      ? _DirectoryView(
                          state: state,
                          onOpen: _controller.openEntry,
                          onOpenParent: _controller.openParent,
                          onRefresh: _controller.refresh,
                        )
                      : CodeWorkspaceEditor(
                          key: ValueKey('${file.path}:${file.version}'),
                          file: file,
                          content: state.draftContent,
                          isDirty: state.isDirty,
                          isSaving: state.isSaving,
                          onChanged: _controller.updateDraft,
                          onSave: _save,
                        ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  Future<void> _copyFile(BuildContext context, CodeSessionFile _) async {
    await Clipboard.setData(
      ClipboardData(text: _controller.state.draftContent),
    );
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(CodeWorkspaceText.t(context, 'files.copied'))),
    );
  }

  Future<void> _save() async {
    final saved = await _controller.saveOpenFile();
    if (!mounted || !saved) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(CodeWorkspaceText.t(context, 'files.saveSuccess')),
      ),
    );
  }

  Future<void> _refresh() async {
    if (_controller.state.isDirty && !await _confirmDiscard()) return;
    await _controller.refresh();
  }

  Future<void> _handleBack() async {
    final state = _controller.state;
    if (state.isSaving) return;
    if (state.openFile == null) {
      Navigator.of(context).pop();
      return;
    }
    if (state.isDirty && !await _confirmDiscard()) return;
    _controller.closeFile();
  }

  Future<bool> _confirmDiscard() async {
    return await showDialog<bool>(
          context: context,
          builder: (dialogContext) => AlertDialog(
            title: Text(
              CodeWorkspaceText.t(dialogContext, 'files.unsavedTitle'),
            ),
            content: Text(
              CodeWorkspaceText.t(dialogContext, 'files.unsavedHint'),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(false),
                child: Text(
                  CodeWorkspaceText.t(dialogContext, 'files.keepEditing'),
                ),
              ),
              FilledButton(
                onPressed: () => Navigator.of(dialogContext).pop(true),
                child: Text(
                  CodeWorkspaceText.t(dialogContext, 'files.discard'),
                ),
              ),
            ],
          ),
        ) ??
        false;
  }
}

class _ScreenTitle extends StatelessWidget {
  const _ScreenTitle({required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
        ),
        Text(
          subtitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 11, color: Colors.white54),
        ),
      ],
    );
  }
}

class _DirectoryView extends StatelessWidget {
  const _DirectoryView({
    required this.state,
    required this.onOpen,
    required this.onOpenParent,
    required this.onRefresh,
  });

  final CodeWorkspaceFilesState state;
  final ValueChanged<CodeStructureEntry> onOpen;
  final VoidCallback onOpenParent;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        _PathBar(path: state.currentPath),
        if (state.truncated)
          Container(
            width: double.infinity,
            color: Colors.amber.withValues(alpha: 0.12),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Text(
              CodeWorkspaceText.t(context, 'files.truncated'),
              style: const TextStyle(color: Colors.amber, fontSize: 12),
            ),
          ),
        Expanded(
          child: RefreshIndicator(
            onRefresh: onRefresh,
            child: ListView.separated(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(12, 8, 12, 24),
              itemCount: state.entries.isEmpty
                  ? (state.currentPath.isEmpty ? 1 : 2)
                  : state.entries.length + (state.currentPath.isEmpty ? 0 : 1),
              separatorBuilder: (_, _) =>
                  const Divider(height: 1, color: Color(0xFF243247)),
              itemBuilder: (context, index) {
                if (state.currentPath.isNotEmpty && index == 0) {
                  return _FileRow(
                    icon: Icons.drive_file_move_outline,
                    title: '..',
                    subtitle: CodeWorkspaceText.t(context, 'action.parent'),
                    onTap: onOpenParent,
                  );
                }
                if (state.entries.isEmpty) {
                  return _EmptyFilesHint(
                    text: CodeWorkspaceText.t(context, 'files.empty'),
                  );
                }
                final offset = state.currentPath.isEmpty ? index : index - 1;
                final entry = state.entries[offset];
                return _FileRow(
                  icon: entry.isDirectory
                      ? Icons.folder_rounded
                      : _iconForExtension(entry.extension),
                  title: entry.name,
                  subtitle: entry.isDirectory
                      ? null
                      : entry.extension.toUpperCase(),
                  onTap: () => onOpen(entry),
                );
              },
            ),
          ),
        ),
      ],
    );
  }

  IconData _iconForExtension(String extension) {
    return switch (extension) {
      'md' => Icons.article_outlined,
      'json' || 'yaml' || 'yml' => Icons.data_object_rounded,
      'png' || 'jpg' || 'jpeg' || 'gif' || 'webp' => Icons.image_outlined,
      _ => Icons.description_outlined,
    };
  }
}

class _PathBar extends StatelessWidget {
  const _PathBar({required this.path});

  final String path;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      color: const Color(0xFF111827),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          const Icon(Icons.account_tree_outlined, color: Colors.blueAccent),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              path.isEmpty ? CodeWorkspaceText.t(context, 'files.root') : path,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: Colors.white70,
                fontFamily: 'monospace',
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _FileRow extends StatelessWidget {
  const _FileRow({
    required this.icon,
    required this.title,
    required this.onTap,
    this.subtitle,
  });

  final IconData icon;
  final String title;
  final String? subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 8),
      leading: Icon(icon, color: Colors.lightBlueAccent),
      title: Text(title, style: const TextStyle(color: Colors.white)),
      subtitle: subtitle == null
          ? null
          : Text(subtitle!, style: const TextStyle(color: Colors.white38)),
      trailing: const Icon(Icons.chevron_right_rounded, color: Colors.white38),
      onTap: onTap,
    );
  }
}

class _EmptyFilesHint extends StatelessWidget {
  const _EmptyFilesHint({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 64),
      child: Column(
        children: [
          const Icon(
            Icons.folder_open_rounded,
            color: Colors.white24,
            size: 44,
          ),
          const SizedBox(height: 12),
          Text(text, style: const TextStyle(color: Colors.white54)),
        ],
      ),
    );
  }
}

class _FilesErrorBanner extends StatelessWidget {
  const _FilesErrorBanner({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return MaterialBanner(
      backgroundColor: const Color(0xFF451A1A),
      content: Text(
        '${CodeWorkspaceText.t(context, 'files.operationFailed')}：$message',
        style: const TextStyle(color: Colors.white),
      ),
      actions: [
        TextButton(
          onPressed: onRetry,
          child: Text(CodeWorkspaceText.t(context, 'action.retry')),
        ),
      ],
    );
  }
}
