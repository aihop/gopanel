import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/ai_dev_session.dart';
import '../../models/code_project_terminal_session.dart';
import '../code_workspace_text.dart';
import '../controllers/ai_workspace_controller.dart';

typedef CodeProjectTerminalLaunch = ({
  CodeProject project,
  CodeProjectTerminalSession terminal,
});

class CodeProjectTerminalSheet extends ConsumerStatefulWidget {
  const CodeProjectTerminalSheet({super.key});

  @override
  ConsumerState<CodeProjectTerminalSheet> createState() =>
      _CodeProjectTerminalSheetState();
}

class _CodeProjectTerminalSheetState
    extends ConsumerState<CodeProjectTerminalSheet> {
  int? _openingProjectId;
  String? _errorMessage;

  Future<void> _openTerminal(CodeProject project) async {
    if (_openingProjectId != null) return;
    setState(() {
      _openingProjectId = project.id;
      _errorMessage = null;
    });
    try {
      final terminal = await ref
          .read(aiWorkspaceRepositoryProvider)
          .openProjectTerminal(project.id);
      if (!mounted) return;
      Navigator.of(context).pop((project: project, terminal: terminal));
    } catch (error) {
      if (!mounted) return;
      final message =
          '${CodeWorkspaceText.t(context, 'terminal.openFailed')}：$error';
      setState(() {
        _openingProjectId = null;
        _errorMessage = message;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(message)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(aiWorkspaceControllerProvider);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(99),
                ),
              ),
            ),
            const SizedBox(height: 20),
            Text(
              CodeWorkspaceText.t(context, 'terminal.chooseProject'),
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 8),
            Text(
              CodeWorkspaceText.t(context, 'terminal.chooseProjectHint'),
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
                height: 1.5,
              ),
            ),
            const SizedBox(height: 18),
            if (_errorMessage != null) ...[
              _TerminalSheetMessage(
                icon: Icons.error_outline_rounded,
                message: _errorMessage!,
                color: Theme.of(context).colorScheme.error,
              ),
              const SizedBox(height: 12),
            ],
            if (state.isLoading && state.projects.isEmpty)
              const SizedBox(
                height: 150,
                child: Center(child: CircularProgressIndicator()),
              )
            else if (state.errorMessage != null && state.projects.isEmpty)
              _TerminalSheetMessage(
                icon: Icons.cloud_off_rounded,
                message: state.errorMessage!,
                color: Theme.of(context).colorScheme.error,
                onRetry: ref
                    .read(aiWorkspaceControllerProvider.notifier)
                    .loadWorkspace,
              )
            else if (state.projects.isEmpty)
              _TerminalSheetMessage(
                icon: Icons.folder_off_outlined,
                message: CodeWorkspaceText.t(context, 'terminal.noProjects'),
              )
            else
              Flexible(
                child: ListView.separated(
                  shrinkWrap: true,
                  itemCount: state.projects.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 8),
                  itemBuilder: (context, index) {
                    final project = state.projects[index];
                    final opening = _openingProjectId == project.id;
                    return ListTile(
                      enabled: _openingProjectId == null,
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 14,
                        vertical: 6,
                      ),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(18),
                      ),
                      tileColor: Theme.of(
                        context,
                      ).colorScheme.surfaceContainerHighest,
                      leading: const Icon(Icons.terminal_rounded),
                      title: Text(
                        project.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      subtitle: Text(
                        project.description.isNotEmpty
                            ? project.description
                            : project.workDir,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      trailing: opening
                          ? const SizedBox.square(
                              dimension: 20,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.chevron_right_rounded),
                      onTap: () => _openTerminal(project),
                    );
                  },
                ),
              ),
            if (_openingProjectId != null) ...[
              const SizedBox(height: 14),
              Center(
                child: Text(
                  CodeWorkspaceText.t(context, 'terminal.opening'),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _TerminalSheetMessage extends StatelessWidget {
  const _TerminalSheetMessage({
    required this.icon,
    required this.message,
    this.color,
    this.onRetry,
  });

  final IconData icon;
  final String message;
  final Color? color;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: (color ?? Theme.of(context).colorScheme.primary).withValues(
          alpha: 0.08,
        ),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Column(
        children: [
          Icon(icon, color: color),
          const SizedBox(height: 10),
          Text(message, textAlign: TextAlign.center),
          if (onRetry != null) ...[
            const SizedBox(height: 12),
            TextButton(
              onPressed: onRetry,
              child: Text(CodeWorkspaceText.t(context, 'action.retry')),
            ),
          ],
        ],
      ),
    );
  }
}
