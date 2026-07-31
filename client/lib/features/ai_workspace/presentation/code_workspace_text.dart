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
      'hub.title': '开发指挥台',
      'hub.sessions': '开发会话',
      'hub.sessionCount': '{count} 个',
      'hub.heroTitle': '随时掌握开发进度',
      'hub.heroDescription': '发指令、看过程、做审批，复杂操作留给桌面端。',
      'hub.all': '全部',
      'hub.active': '进行中',
      'hub.create': '新建',
      'stop.title': '停止当前开发任务？',
      'stop.description': '正在执行和排队中的指令会被停止，已产生的文件变更会保留。',
      'stop.cancel': '继续执行',
      'stop.confirm': '确认停止',
      'stop.success': '已发送停止请求',
      'delivery.title': '统一交付',
      'delivery.empty': '将当前会话变更合并到项目主分支并统一推送。',
      'delivery.unavailable': '当前会话没有受管隔离工作区，请新建支持 Git Worktree 的项目会话。',
      'delivery.start': '开始统一交付',
      'delivery.retry': '重新交付',
      'delivery.confirmTitle': '开始统一交付？',
      'delivery.confirmDescription':
          '系统会停止宿主终端，完成同步、合并、质量检查和推送。交付封口后不能继续修改当前会话。',
      'delivery.cancel': '暂不交付',
      'delivery.confirm': '确认交付',
      'delivery.started': '统一交付已进入队列',
      'delivery.failed': '交付失败',
      'delivery.conflict': '检测到合并冲突',
      'delivery.conflictHint': '请先在电脑端打开该会话的隔离工作区，解决以下冲突并提交，再返回重试。',
      'delivery.completed': '已完成合并与推送',
      'delivery.queued': '等待交付',
      'delivery.running': '正在交付',
      'delivery.queuePosition': '队列第 {position} 位',
      'delivery.attempt': '第 {attempt} 次尝试',
      'delivery.stageStoppingTerminal': '正在停止宿主终端',
      'delivery.stageSyncing': '正在同步远程分支',
      'delivery.stageMerging': '正在合并会话变更',
      'delivery.stageQualityCheck': '正在执行质量检查',
      'delivery.stagePushing': '正在推送主分支',
      'delivery.stageVerifying': '正在核验远端提交',
      'delivery.stageCleaning': '正在清理会话工作区',
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
      'hub.title': 'Development command center',
      'hub.sessions': 'Development sessions',
      'hub.sessionCount': '{count}',
      'hub.heroTitle': 'Stay on top of development',
      'hub.heroDescription':
          'Send instructions, follow progress, and approve actions. Keep complex work on desktop.',
      'hub.all': 'All',
      'hub.active': 'Active',
      'hub.create': 'New',
      'stop.title': 'Stop the current Code task?',
      'stop.description':
          'Running and queued instructions will stop. Existing file changes are preserved.',
      'stop.cancel': 'Keep running',
      'stop.confirm': 'Stop',
      'stop.success': 'Stop request sent',
      'delivery.title': 'Unified delivery',
      'delivery.empty':
          'Merge this session into the project branch and push it through the managed delivery flow.',
      'delivery.unavailable':
          'This session has no managed worktree. Create a project session with Git Worktree support.',
      'delivery.start': 'Start delivery',
      'delivery.retry': 'Retry delivery',
      'delivery.confirmTitle': 'Start unified delivery?',
      'delivery.confirmDescription':
          'Host terminals stop before sync, merge, quality checks, and push. The session is sealed after delivery starts.',
      'delivery.cancel': 'Not now',
      'delivery.confirm': 'Deliver',
      'delivery.started': 'Delivery queued',
      'delivery.failed': 'Delivery failed',
      'delivery.conflict': 'Merge conflicts detected',
      'delivery.conflictHint':
          'Open this session worktree on desktop, resolve and commit these conflicts, then retry here.',
      'delivery.completed': 'Merge and push completed',
      'delivery.queued': 'Waiting for delivery',
      'delivery.running': 'Delivering',
      'delivery.queuePosition': 'Queue position {position}',
      'delivery.attempt': 'Attempt {attempt}',
      'delivery.stageStoppingTerminal': 'Stopping host terminals',
      'delivery.stageSyncing': 'Syncing the remote branch',
      'delivery.stageMerging': 'Merging session changes',
      'delivery.stageQualityCheck': 'Running quality checks',
      'delivery.stagePushing': 'Pushing the main branch',
      'delivery.stageVerifying': 'Verifying the remote commit',
      'delivery.stageCleaning': 'Cleaning the session workspace',
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
