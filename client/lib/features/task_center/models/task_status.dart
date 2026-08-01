enum TaskStatus {
  running,
  success,
  failed,
}

extension TaskStatusX on TaskStatus {
  String get label {
    switch (this) {
      case TaskStatus.running:
        return '运行中';
      case TaskStatus.success:
        return '成功';
      case TaskStatus.failed:
        return '失败';
    }
  }
}

