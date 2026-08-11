import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../models/code_git_review.dart';
import '../code_git_review_text.dart';

class CodeGitSummaryCard extends StatelessWidget {
  const CodeGitSummaryCard({super.key, required this.status});

  final CodeGitStatus status;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Text(CodeGitReviewText.t(context, 'title')),
      trailing: Text(
        CodeGitReviewText.format(context, 'summary', {'files': status.files}),
      ),
      child: Wrap(
        spacing: 10,
        runSpacing: 10,
        children: [
          _CountBadge(
            label: '+${status.totalAdditions}',
            color: AppTheme.success,
          ),
          _CountBadge(
            label: '-${status.totalDeletions}',
            color: AppTheme.error,
          ),
          if (status.staged > 0)
            _CountBadge(
              label:
                  '${CodeGitReviewText.t(context, 'staged')} ${status.staged}',
              color: AppTheme.primaryBlue,
            ),
          if (status.changed > 0)
            _CountBadge(
              label:
                  '${CodeGitReviewText.t(context, 'changed')} ${status.changed}',
              color: Colors.orange,
            ),
          if (status.untracked > 0)
            _CountBadge(
              label:
                  '${CodeGitReviewText.t(context, 'untracked')} ${status.untracked}',
              color: Colors.purple,
            ),
        ],
      ),
    );
  }
}

class CodeGitRepositoryCard extends StatelessWidget {
  const CodeGitRepositoryCard({
    super.key,
    required this.repository,
    required this.onOpenDiff,
  });

  final CodeGitRepositoryStatus repository;
  final void Function(CodeGitFile file, String kind) onOpenDiff;

  @override
  Widget build(BuildContext context) {
    return PanelCard(
      title: Text(repository.name),
      trailing: Text(
        repository.branch.isEmpty
            ? CodeGitReviewText.t(context, 'detached')
            : repository.branch,
        style: const TextStyle(color: AppTheme.textSecondary, fontSize: 11),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (repository.savedCommits > 0) ...[
            Text(
              CodeGitReviewText.format(context, 'savedCommits', {
                'count': repository.savedCommits,
              }),
              style: const TextStyle(color: AppTheme.success, fontSize: 12),
            ),
            const SizedBox(height: 10),
          ],
          if (repository.truncated) ...[
            _Warning(message: CodeGitReviewText.t(context, 'truncated')),
            const SizedBox(height: 10),
          ],
          for (var index = 0; index < repository.files.length; index++) ...[
            _FileItem(file: repository.files[index], onOpenDiff: onOpenDiff),
            if (index != repository.files.length - 1) const Divider(height: 20),
          ],
        ],
      ),
    );
  }
}

class _FileItem extends StatelessWidget {
  const _FileItem({required this.file, required this.onOpenDiff});

  final CodeGitFile file;
  final void Function(CodeGitFile file, String kind) onOpenDiff;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            _FileStatus(file: file),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                file.path,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontWeight: FontWeight.w700),
              ),
            ),
          ],
        ),
        if (file.oldPath.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(
            file.oldPath,
            style: const TextStyle(color: AppTheme.textLight, fontSize: 11),
          ),
        ],
        const SizedBox(height: 6),
        Wrap(
          spacing: 6,
          runSpacing: 4,
          children: [
            if (file.staged)
              TextButton.icon(
                onPressed: () => onOpenDiff(file, 'staged'),
                icon: const Icon(Icons.difference_outlined, size: 16),
                label: Text(CodeGitReviewText.t(context, 'stagedDiff')),
              ),
            if (file.changed || file.untracked)
              TextButton.icon(
                onPressed: () => onOpenDiff(file, 'working'),
                icon: const Icon(Icons.difference_outlined, size: 16),
                label: Text(CodeGitReviewText.t(context, 'workingDiff')),
              ),
          ],
        ),
      ],
    );
  }
}

class _FileStatus extends StatelessWidget {
  const _FileStatus({required this.file});

  final CodeGitFile file;

  @override
  Widget build(BuildContext context) {
    final label = file.untracked
        ? 'U'
        : file.worktreeStatus.trim().isNotEmpty
        ? file.worktreeStatus
        : file.indexStatus;
    final color = file.untracked
        ? Colors.purple
        : file.staged && !file.changed
        ? AppTheme.primaryBlue
        : Colors.orange;
    return _CountBadge(label: label, color: color);
  }
}

class _CountBadge extends StatelessWidget {
  const _CountBadge({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _Warning extends StatelessWidget {
  const _Warning({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: const Color(0xFFFFFBEB),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(message, style: const TextStyle(color: Colors.orange)),
    );
  }
}
