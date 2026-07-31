class SslInfo {
  final int id;
  final String primaryDomain;
  final String type;
  final String provider;
  final bool autoRenew;
  final String status;
  final String message;
  final DateTime? startDate;
  final DateTime? expireDate;

  SslInfo({
    required this.id,
    required this.primaryDomain,
    required this.type,
    required this.provider,
    required this.autoRenew,
    required this.status,
    required this.message,
    required this.startDate,
    required this.expireDate,
  });

  factory SslInfo.fromJson(Map<String, dynamic> json) {
    return SslInfo(
      id: (json['id'] as num?)?.toInt() ?? 0,
      primaryDomain: (json['primaryDomain'] ?? '').toString(),
      type: (json['type'] ?? '').toString(),
      provider: (json['provider'] ?? '').toString(),
      autoRenew: (json['autoRenew'] as bool?) ?? false,
      status: (json['status'] ?? '').toString(),
      message: (json['message'] ?? '').toString(),
      startDate: DateTime.tryParse((json['startDate'] ?? '').toString()),
      expireDate: DateTime.tryParse((json['expireDate'] ?? '').toString()),
    );
  }
}

