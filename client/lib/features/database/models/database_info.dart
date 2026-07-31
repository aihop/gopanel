class DatabaseInfo {
  final String type;
  final String name;
  final String server;
  final int serverId;
  final String encoding;
  final String comment;

  DatabaseInfo({
    required this.type,
    required this.name,
    required this.server,
    required this.serverId,
    required this.encoding,
    required this.comment,
  });

  factory DatabaseInfo.fromJson(Map<String, dynamic> json) {
    return DatabaseInfo(
      type: (json['type'] ?? '').toString(),
      name: (json['name'] ?? '').toString(),
      server: (json['server'] ?? '').toString(),
      serverId: (json['serverId'] as num?)?.toInt() ?? 0,
      encoding: (json['encoding'] ?? '').toString(),
      comment: (json['comment'] ?? '').toString(),
    );
  }
}

