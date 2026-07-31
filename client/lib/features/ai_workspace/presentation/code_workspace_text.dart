import 'package:flutter/widgets.dart';

abstract final class CodeWorkspaceText {
  static const _values = <String, Map<String, String>>{
    'zh': {
      'files.title': '项目文件',
      'files.root': '项目根目录',
      'files.empty': '当前目录没有可查看的文件',
      'files.truncated': '文件较多，仅展示前 500 项',
      'files.readOnly': '只读查看',
      'files.copy': '复制内容',
      'files.copied': '文件内容已复制',
      'files.loadFailed': '文件加载失败',
      'action.back': '返回',
      'action.parent': '上一级',
      'action.refresh': '刷新',
      'action.retry': '重试',
      'action.stop': '停止执行',
      'stop.title': '停止当前开发任务？',
      'stop.description': '正在执行和排队中的指令会被停止，已产生的文件变更会保留。',
      'stop.cancel': '继续执行',
      'stop.confirm': '确认停止',
      'stop.success': '已发送停止请求',
    },
    'en': {
      'files.title': 'Project files',
      'files.root': 'Project root',
      'files.empty': 'No viewable files in this directory',
      'files.truncated': 'Only the first 500 items are shown',
      'files.readOnly': 'Read only',
      'files.copy': 'Copy content',
      'files.copied': 'File content copied',
      'files.loadFailed': 'Failed to load files',
      'action.back': 'Back',
      'action.parent': 'Parent folder',
      'action.refresh': 'Refresh',
      'action.retry': 'Retry',
      'action.stop': 'Stop execution',
      'stop.title': 'Stop the current Code task?',
      'stop.description':
          'Running and queued instructions will stop. Existing file changes are preserved.',
      'stop.cancel': 'Keep running',
      'stop.confirm': 'Stop',
      'stop.success': 'Stop request sent',
    },
  };

  static String t(BuildContext context, String key) {
    final language = View.of(context).platformDispatcher.locale.languageCode;
    return _values[language]?[key] ?? _values['en']![key] ?? key;
  }
}
