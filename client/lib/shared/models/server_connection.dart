class ServerConnection {
  final String id;
  final String name; // 别名，比如 "香港测试服务器"
  final String url;
  final String token;
  final DateTime lastConnectedAt;

  ServerConnection({
    required this.id,
    required this.name,
    required this.url,
    required this.token,
    required this.lastConnectedAt,
  });

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'url': url,
      'token': token,
      'lastConnectedAt': lastConnectedAt.toIso8601String(),
    };
  }

  factory ServerConnection.fromJson(Map<String, dynamic> json) {
    return ServerConnection(
      id: json['id'] as String,
      name: json['name'] as String,
      url: json['url'] as String,
      token: json['token'] as String,
      lastConnectedAt: DateTime.parse(json['lastConnectedAt'] as String),
    );
  }
}
