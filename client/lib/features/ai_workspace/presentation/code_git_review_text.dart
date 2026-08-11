import 'package:flutter/widgets.dart';

abstract final class CodeGitReviewText {
  static const _values = <String, Map<String, String>>{
    'zh': {
      'title': '变更审查',
      'summary': '{files} 个文件',
      'noRepository': '当前会话没有可审查的 Git 仓库',
      'noChanges': '当前工作区没有未保存修改',
      'loadFailed': 'Git 变更加载失败',
      'repository': '仓库',
      'detached': '未绑定分支',
      'staged': '已暂存',
      'changed': '未暂存',
      'untracked': '新文件',
      'savedCommits': '已保存 {count} 个提交',
      'truncated': '文件较多，列表仅展示前 500 项',
      'workingDiff': '查看未暂存差异',
      'stagedDiff': '查看已暂存差异',
      'diffTitle': '文件差异',
      'diffLoadFailed': '文件差异加载失败',
      'diffEmpty': '该文件没有可展示的文本差异',
      'diffTruncated': '差异内容过大，当前仅展示前 1 MB',
      'copyDiff': '复制差异',
      'diffCopied': '差异内容已复制',
      'save': '保存修改',
      'saveUnavailable': '仅受管隔离工作区可以从手机保存修改',
      'saveTitle': '保存当前全部修改？',
      'saveHint': '系统会检查文件安全性，并将当前会话工作区的全部修改保存为 Git 提交。',
      'message': '提交说明',
      'messageHint': '留空则使用默认提交说明',
      'cancel': '取消',
      'confirmSave': '确认保存',
      'saveSuccess': '修改已保存为 Git 提交',
      'saveFailed': '保存失败，请检查错误后重试',
    },
    'en': {
      'title': 'Review changes',
      'summary': '{files} files',
      'noRepository': 'No Git repository is available for this session',
      'noChanges': 'The workspace has no unsaved changes',
      'loadFailed': 'Failed to load Git changes',
      'repository': 'Repository',
      'detached': 'Detached HEAD',
      'staged': 'Staged',
      'changed': 'Unstaged',
      'untracked': 'New file',
      'savedCommits': '{count} saved commits',
      'truncated': 'Only the first 500 changed files are shown',
      'workingDiff': 'View unstaged diff',
      'stagedDiff': 'View staged diff',
      'diffTitle': 'File diff',
      'diffLoadFailed': 'Failed to load file diff',
      'diffEmpty': 'No text diff is available for this file',
      'diffTruncated': 'The diff is too large; only the first 1 MB is shown',
      'copyDiff': 'Copy diff',
      'diffCopied': 'Diff copied',
      'save': 'Save changes',
      'saveUnavailable':
          'Only managed isolated workspaces can be saved from mobile',
      'saveTitle': 'Save all current changes?',
      'saveHint':
          'GoPanel checks file safety and saves every current workspace change in a Git commit.',
      'message': 'Commit message',
      'messageHint': 'Leave blank to use the default message',
      'cancel': 'Cancel',
      'confirmSave': 'Save',
      'saveSuccess': 'Changes saved in a Git commit',
      'saveFailed': 'Save failed. Review the error and try again.',
    },
  };

  static String t(BuildContext context, String key) {
    final language = View.of(context).platformDispatcher.locale.languageCode;
    return _values[language]?[key] ?? _values['en']![key] ?? key;
  }

  static String format(
    BuildContext context,
    String key,
    Map<String, Object> values,
  ) {
    var text = t(context, key);
    for (final entry in values.entries) {
      text = text.replaceAll('{${entry.key}}', entry.value.toString());
    }
    return text;
  }
}
