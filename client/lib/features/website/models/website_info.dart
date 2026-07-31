class WebsiteInfo {
  final int id;
  final String alias;
  final String primaryDomain;
  final String type;
  final String status;
  final int pipelineId;
  final String appName;
  final DateTime? updatedAt;
  final DateTime? expireDate;

  WebsiteInfo({
    required this.id,
    required this.alias,
    required this.primaryDomain,
    required this.type,
    required this.status,
    required this.pipelineId,
    required this.appName,
    required this.updatedAt,
    required this.expireDate,
  });

  factory WebsiteInfo.fromJson(Map<String, dynamic> json) {
    return WebsiteInfo(
      id: (json['id'] as num?)?.toInt() ?? 0,
      alias: (json['alias'] ?? '').toString(),
      primaryDomain: (json['primaryDomain'] ?? '').toString(),
      type: (json['type'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      pipelineId: (json['pipelineId'] as num?)?.toInt() ?? 0,
      appName: (json['appName'] ?? '').toString(),
      updatedAt: DateTime.tryParse((json['updatedAt'] ?? '').toString()),
      expireDate: DateTime.tryParse((json['expireDate'] ?? '').toString()),
    );
  }
}

