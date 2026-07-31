class DnsAccount {
  final int id;
  final String name;
  final String type;
  final String authorization;
  final String createdAt;

  DnsAccount({
    required this.id,
    required this.name,
    required this.type,
    required this.authorization,
    required this.createdAt,
  });

  factory DnsAccount.fromJson(Map<String, dynamic> json) {
    return DnsAccount(
      id: json['id'] as int? ?? 0,
      name: json['name'] as String? ?? '',
      type: json['type'] as String? ?? '',
      authorization: json['authorization'] as String? ?? '',
      createdAt: json['createdAt'] as String? ?? '',
    );
  }
}
