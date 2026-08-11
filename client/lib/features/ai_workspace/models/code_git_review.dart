class CodeGitFile {
  final String path;
  final String oldPath;
  final String workspacePath;
  final String indexStatus;
  final String worktreeStatus;
  final bool staged;
  final bool changed;
  final bool untracked;

  const CodeGitFile({
    required this.path,
    required this.oldPath,
    required this.workspacePath,
    required this.indexStatus,
    required this.worktreeStatus,
    required this.staged,
    required this.changed,
    required this.untracked,
  });

  factory CodeGitFile.fromJson(Map<String, dynamic> json) {
    return CodeGitFile(
      path: (json['path'] ?? '').toString(),
      oldPath: (json['oldPath'] ?? '').toString(),
      workspacePath: (json['workspacePath'] ?? '').toString(),
      indexStatus: (json['indexStatus'] ?? '').toString(),
      worktreeStatus: (json['worktreeStatus'] ?? '').toString(),
      staged: json['staged'] == true,
      changed: json['changed'] == true,
      untracked: json['untracked'] == true,
    );
  }
}

class CodeGitRepositoryStatus {
  final String id;
  final String name;
  final String branch;
  final List<CodeGitFile> files;
  final int stagedCount;
  final int changedCount;
  final int untrackedCount;
  final int additions;
  final int deletions;
  final int stagedAdditions;
  final int stagedDeletions;
  final bool truncated;
  final bool isolated;
  final int savedCommits;
  final String headCommit;

  const CodeGitRepositoryStatus({
    required this.id,
    required this.name,
    required this.branch,
    required this.files,
    required this.stagedCount,
    required this.changedCount,
    required this.untrackedCount,
    required this.additions,
    required this.deletions,
    required this.stagedAdditions,
    required this.stagedDeletions,
    required this.truncated,
    required this.isolated,
    required this.savedCommits,
    required this.headCommit,
  });

  int get totalAdditions => additions + stagedAdditions;
  int get totalDeletions => deletions + stagedDeletions;

  factory CodeGitRepositoryStatus.fromJson(Map<String, dynamic> json) {
    int number(String key) => (json[key] as num?)?.toInt() ?? 0;
    return CodeGitRepositoryStatus(
      id: (json['id'] ?? '').toString(),
      name: (json['name'] ?? '').toString(),
      branch: (json['branch'] ?? '').toString(),
      files: (json['files'] as List<dynamic>? ?? const [])
          .whereType<Map>()
          .map((item) => CodeGitFile.fromJson(item.cast<String, dynamic>()))
          .toList(),
      stagedCount: number('stagedCount'),
      changedCount: number('changedCount'),
      untrackedCount: number('untrackedCount'),
      additions: number('additions'),
      deletions: number('deletions'),
      stagedAdditions: number('stagedAdditions'),
      stagedDeletions: number('stagedDeletions'),
      truncated: json['truncated'] == true,
      isolated: json['isolated'] == true,
      savedCommits: number('savedCommits'),
      headCommit: (json['headCommit'] ?? '').toString(),
    );
  }
}

class CodeGitStatus {
  final bool available;
  final String reason;
  final List<CodeGitRepositoryStatus> repositories;
  final int files;
  final int staged;
  final int changed;
  final int untracked;
  final int additions;
  final int deletions;
  final int stagedAdditions;
  final int stagedDeletions;

  const CodeGitStatus({
    required this.available,
    required this.reason,
    required this.repositories,
    required this.files,
    required this.staged,
    required this.changed,
    required this.untracked,
    required this.additions,
    required this.deletions,
    required this.stagedAdditions,
    required this.stagedDeletions,
  });

  bool get hasChanges => files > 0;
  int get totalAdditions => additions + stagedAdditions;
  int get totalDeletions => deletions + stagedDeletions;

  factory CodeGitStatus.fromJson(Map<String, dynamic> json) {
    int number(String key) => (json[key] as num?)?.toInt() ?? 0;
    return CodeGitStatus(
      available: json['available'] == true,
      reason: (json['reason'] ?? '').toString(),
      repositories: (json['repositories'] as List<dynamic>? ?? const [])
          .whereType<Map>()
          .map(
            (item) =>
                CodeGitRepositoryStatus.fromJson(item.cast<String, dynamic>()),
          )
          .toList(),
      files: number('files'),
      staged: number('staged'),
      changed: number('changed'),
      untracked: number('untracked'),
      additions: number('additions'),
      deletions: number('deletions'),
      stagedAdditions: number('stagedAdditions'),
      stagedDeletions: number('stagedDeletions'),
    );
  }
}

class CodeGitDiff {
  final String repositoryId;
  final String path;
  final String kind;
  final String content;
  final bool truncated;

  const CodeGitDiff({
    required this.repositoryId,
    required this.path,
    required this.kind,
    required this.content,
    required this.truncated,
  });

  factory CodeGitDiff.fromJson(Map<String, dynamic> json) {
    return CodeGitDiff(
      repositoryId: (json['repositoryId'] ?? '').toString(),
      path: (json['path'] ?? '').toString(),
      kind: (json['kind'] ?? '').toString(),
      content: (json['content'] ?? '').toString(),
      truncated: json['truncated'] == true,
    );
  }
}

class CodeGitSaveResult {
  final String status;
  final String commit;
  final String branch;
  final List<String> commits;

  const CodeGitSaveResult({
    required this.status,
    required this.commit,
    required this.branch,
    required this.commits,
  });

  factory CodeGitSaveResult.fromJson(Map<String, dynamic> json) {
    final repositories = json['repositories'] as List<dynamic>? ?? const [];
    return CodeGitSaveResult(
      status: (json['status'] ?? '').toString(),
      commit: (json['commit'] ?? '').toString(),
      branch: (json['branch'] ?? '').toString(),
      commits: repositories
          .whereType<Map>()
          .map((item) => (item['commit'] ?? '').toString())
          .where((item) => item.isNotEmpty)
          .toList(),
    );
  }
}
