enum TaskType {
  ai,
  pipeline,
  appInstall,
  backup,
  ssl,
  systemUpgrade,
  other,
}

extension TaskTypeX on TaskType {
  String get label {
    switch (this) {
      case TaskType.ai:
        return 'AI 开发';
      case TaskType.pipeline:
        return '流水线';
      case TaskType.appInstall:
        return '应用安装';
      case TaskType.backup:
        return '备份';
      case TaskType.ssl:
        return '证书';
      case TaskType.systemUpgrade:
        return '升级';
      case TaskType.other:
        return '任务';
    }
  }
}
