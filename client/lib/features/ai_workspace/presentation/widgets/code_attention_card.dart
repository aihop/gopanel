import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../task_center/models/task_attention.dart';
import '../code_workspace_text.dart';

class CodeAttentionCard extends StatelessWidget {
  const CodeAttentionCard({
    super.key,
    required this.isLoading,
    required this.errorMessage,
    required this.items,
    required this.onOpen,
    required this.onRetry,
  });

  final bool isLoading;
  final String? errorMessage;
  final List<TaskAttention> items;
  final VoidCallback onOpen;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Row(
        children: [
          const Icon(
            Icons.notification_important_outlined,
            size: 21,
            color: AppTheme.error,
          ),
          const SizedBox(width: 8),
          Text(CodeWorkspaceText.t(context, 'attention.title')),
          if (items.isNotEmpty) ...[
            const SizedBox(width: 8),
            Badge.count(count: items.length, backgroundColor: AppTheme.error),
          ],
        ],
      ),
      trailing: TextButton(
        onPressed: isLoading ? null : onOpen,
        child: Text(CodeWorkspaceText.t(context, 'attention.viewAll')),
      ),
      child: _buildBody(context),
    );
  }

  Widget _buildBody(BuildContext context) {
    if (isLoading && items.isEmpty) {
      return const Center(child: LinearProgressIndicator(minHeight: 2));
    }
    if (errorMessage != null && items.isEmpty) {
      return Row(
        children: [
          const Icon(Icons.cloud_off_outlined, color: AppTheme.textLight),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              CodeWorkspaceText.t(context, 'attention.loadFailed'),
              style: const TextStyle(color: AppTheme.textSecondary),
            ),
          ),
          TextButton(
            onPressed: onRetry,
            child: Text(CodeWorkspaceText.t(context, 'action.retry')),
          ),
        ],
      );
    }
    if (items.isEmpty) {
      return Row(
        children: [
          const Icon(Icons.check_circle_outline, color: AppTheme.success),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              CodeWorkspaceText.t(context, 'attention.empty'),
              style: const TextStyle(color: AppTheme.textSecondary),
            ),
          ),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (var index = 0; index < items.take(2).length; index++) ...[
          Text(
            items[index].title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontWeight: FontWeight.w800),
          ),
          if (items[index].summary.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              items[index].summary,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                height: 1.4,
              ),
            ),
          ],
          if (index < items.take(2).length - 1) const Divider(height: 24),
        ],
      ],
    );
  }
}
