/// 已安装应用操作类型常量 (与 GoPanel 后端 constant.AppOperate 对齐)
class AppOp {
  static const String start = 'start';
  static const String stop = 'stop';
  static const String restart = 'restart';
  static const String rebuild = 'rebuild';
  static const String upgrade = 'upgrade';
  static const String delete = 'delete';
  static const String sync = 'sync';
}

/// 已安装应用信息模型
class AppInstallInfo {
  final int id; // installId
  final String name; // 应用名称 (如 mysql)
  final String version; // 当前安装版本
  final String status; // 状态 (Running, Stopped, Error 等)
  final String description; // 描述
  final String icon; // 图标 URL 或相对路径
  final String createdAt;

  AppInstallInfo({
    required this.id,
    required this.name,
    required this.version,
    required this.status,
    required this.description,
    required this.icon,
    required this.createdAt,
  });

  factory AppInstallInfo.fromJson(Map<String, dynamic> json) {
    return AppInstallInfo(
      id: (json['id'] as num?)?.toInt() ?? 0,
      name: (json['name'] ?? '').toString(),
      version: (json['version'] ?? '').toString(),
      status: (json['status'] ?? 'Unknown').toString(),
      description: (json['appName'] ?? json['message'] ?? '').toString(),
      icon: (json['icon'] ?? '').toString(),
      createdAt: (json['createdAt'] ?? '').toString(),
    );
  }

  bool get isRunning => status.toLowerCase() == 'running';
  bool get isStopped => status.toLowerCase() == 'stopped';
}
