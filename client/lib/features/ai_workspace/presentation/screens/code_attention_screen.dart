import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../task_center/models/task_attention.dart';
import '../code_workspace_text.dart';
import '../controllers/ai_workspace_controller.dart';
import '../controllers/code_attention_controller.dart';
import 'ai_chat_screen.dart';

class CodeAttentionScreen extends ConsumerWidget {
  const CodeAttentionScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(codeAttentionControllerProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(CodeWorkspaceText.t(context, 'attention.title')),
        actions: [
          IconButton(
            tooltip: CodeWorkspaceText.t(context, 'action.refresh'),
            onPressed: state.isLoading
                ? null
                : ref.read(codeAttentionControllerProvider.notifier).load,
            icon: const Icon(Icons.refresh_rounded),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: ref.read(codeAttentionControllerProvider.notifier).load,
        child: _buildBody(context, ref, state),
      ),
    );
  }

  Widget _buildBody(
    BuildContext context,
    WidgetRef ref,
    CodeAttentionState state,
  ) {
    if (state.isLoading && state.items.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 180),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (state.errorMessage != null && state.items.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(20),
        children: [
          const SizedBox(height: 80),
          const Icon(Icons.cloud_off_outlined, size: 48),
          const SizedBox(height: 12),
          Text(
            CodeWorkspaceText.t(context, 'attention.loadFailed'),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            state.errorMessage!,
            textAlign: TextAlign.center,
            style: const TextStyle(color: AppTheme.textSecondary),
          ),
        ],
      );
    }
    if (state.items.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          const SizedBox(height: 112),
          const Icon(
            Icons.verified_user_outlined,
            size: 48,
            color: AppTheme.success,
          ),
          const SizedBox(height: 12),
          Center(child: Text(CodeWorkspaceText.t(context, 'attention.empty'))),
        ],
      );
    }

    return ListView.separated(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      itemCount: state.items.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (context, index) =>
          _itemCard(context, ref, state, state.items[index]),
    );
  }

  Widget _itemCard(
    BuildContext context,
    WidgetRef ref,
    CodeAttentionState state,
    TaskAttention attention,
  ) {
    return PanelCard(
      title: Text(attention.title),
      trailing: Icon(
        attention.severity == 'error'
            ? Icons.error_outline_rounded
            : Icons.warning_amber_rounded,
        color: attention.severity == 'error' ? AppTheme.error : Colors.orange,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (attention.summary.isNotEmpty)
            Text(
              attention.summary,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                height: 1.45,
              ),
            ),
          const SizedBox(height: 12),
          Text(
            CodeWorkspaceText.format(context, 'attention.session', {
              'id': attention.sessionId,
            }),
            style: const TextStyle(color: AppTheme.textLight, fontSize: 12),
          ),
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: attention.actions
                .map(
                  (action) => OutlinedButton.icon(
                    onPressed: state.actionKey == null
                        ? () => _handleAction(context, ref, attention, action)
                        : null,
                    icon: state.actionKey == '${attention.id}:${action.type}'
                        ? const SizedBox.square(
                            dimension: 15,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Icon(_actionIcon(action.type), size: 18),
                    label: Text(action.label),
                  ),
                )
                .toList(),
          ),
        ],
      ),
    );
  }

  Future<void> _handleAction(
    BuildContext context,
    WidgetRef ref,
    TaskAttention attention,
    TaskAttentionAction action,
  ) async {
    if (action.type == 'open_session') {
      await _openSession(context, ref, attention.sessionId);
      return;
    }
    if (action.requiresConfirmation) {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          title: Text(action.label),
          content: Text(
            CodeWorkspaceText.t(dialogContext, 'attention.confirmHint'),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(false),
              child: Text(
                CodeWorkspaceText.t(dialogContext, 'attention.cancel'),
              ),
            ),
            FilledButton(
              onPressed: () => Navigator.of(dialogContext).pop(true),
              child: Text(
                CodeWorkspaceText.t(dialogContext, 'attention.confirm'),
              ),
            ),
          ],
        ),
      );
      if (confirmed != true || !context.mounted) return;
    }
    final success = await ref
        .read(codeAttentionControllerProvider.notifier)
        .execute(attention, action);
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          success
              ? CodeWorkspaceText.t(context, 'attention.actionSuccess')
              : CodeWorkspaceText.t(context, 'attention.actionFailed'),
        ),
      ),
    );
  }

  Future<void> _openSession(
    BuildContext context,
    WidgetRef ref,
    int sessionId,
  ) async {
    try {
      final repository = ref.read(aiWorkspaceRepositoryProvider);
      final sessionState = await repository.getSessionState(sessionId);
      await ref
          .read(aiWorkspaceControllerProvider.notifier)
          .selectSession(sessionState.session);
      if (!context.mounted) return;
      await Navigator.of(
        context,
      ).push(MaterialPageRoute(builder: (_) => const AiChatScreen()));
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${CodeWorkspaceText.t(context, 'attention.openFailed')}：$error',
          ),
        ),
      );
    }
  }

  IconData _actionIcon(String type) => switch (type) {
    'approve' => Icons.check_rounded,
    'reject' => Icons.close_rounded,
    'retry_initialization' => Icons.refresh_rounded,
    _ => Icons.open_in_new_rounded,
  };
}
