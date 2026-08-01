/// 容器操作类型常量 (与 GoPanel 后端对齐)
class ContainerOp {
  static const String start = 'start';
  static const String stop = 'stop';
  static const String restart = 'restart';
  static const String kill = 'kill';
  static const String pause = 'pause';
  static const String unpause = 'unpause';
  static const String remove = 'remove';
}

/// 容器信息模型 (对应 GoPanel 后端的 dto.ContainerInfo)
class ContainerInfo {
  final String id;
  final String name;
  final String image;
  final String state; // running, exited, paused 等
  final String status; // 例如 "Up 2 days", "Exited (0) 5 days ago"
  final String created;

  ContainerInfo({
    required this.id,
    required this.name,
    required this.image,
    required this.state,
    required this.status,
    required this.created,
  });

  factory ContainerInfo.fromJson(Map<String, dynamic> json) {
    return ContainerInfo(
      id: (json['containerID'] ?? json['id'] ?? '').toString(),
      name: (json['name'] ?? '').toString(),
      image: (json['imageName'] ?? json['image'] ?? '').toString(),
      state: (json['state'] ?? 'unknown').toString(),
      status: (json['runTime'] ?? json['status'] ?? '').toString(),
      created: (json['createTime'] ?? json['created'] ?? '').toString(),
    );
  }

  bool get isRunning => state == 'running';
}
