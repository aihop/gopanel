import 'dart:convert';

class SecurityRisk {
  final int id;
  final String sourceType;
  final String sourceName;
  final String eventType;
  final String level;
  final String summary;
  final String aiConclusion;
  final int confidence;
  final DateTime? lastSeenAt;
  final List<SecurityRiskEvidence> evidence;
  final List<SecurityRiskAction> actions;

  const SecurityRisk({
    required this.id,
    required this.sourceType,
    required this.sourceName,
    required this.eventType,
    required this.level,
    required this.summary,
    required this.aiConclusion,
    required this.confidence,
    required this.lastSeenAt,
    required this.evidence,
    required this.actions,
  });

  bool get requiresApproval => actions.any((action) => action.requiresApproval);

  factory SecurityRisk.fromJson(Map<String, dynamic> json) {
    return SecurityRisk(
      id: _asInt(json['id']),
      sourceType: json['sourceType']?.toString() ?? '',
      sourceName: json['sourceName']?.toString() ?? '',
      eventType: json['eventType']?.toString() ?? '',
      level: json['level']?.toString() ?? 'info',
      summary: json['summary']?.toString() ?? '',
      aiConclusion: json['aiConclusion']?.toString() ?? '',
      confidence: _asInt(json['confidence']),
      lastSeenAt: DateTime.tryParse(json['lastSeenAt']?.toString() ?? ''),
      evidence: _decodeList(json['evidence'])
          .map(SecurityRiskEvidence.fromJson)
          .toList(),
      actions: _decodeList(json['suggestedActions'])
          .map(SecurityRiskAction.fromJson)
          .toList(),
    );
  }
}

class SecurityRiskEvidence {
  final String description;
  final int count;

  const SecurityRiskEvidence({required this.description, required this.count});

  factory SecurityRiskEvidence.fromJson(Map<String, dynamic> json) {
    return SecurityRiskEvidence(
      description: json['description']?.toString() ?? '',
      count: _asInt(json['count']),
    );
  }
}

class SecurityRiskAction {
  final String action;
  final bool requiresApproval;

  const SecurityRiskAction({
    required this.action,
    required this.requiresApproval,
  });

  factory SecurityRiskAction.fromJson(Map<String, dynamic> json) {
    return SecurityRiskAction(
      action: json['action']?.toString() ?? '',
      requiresApproval: json['requiresApproval'] == true,
    );
  }
}

List<Map<String, dynamic>> _decodeList(dynamic value) {
  try {
    final decoded = value is String ? jsonDecode(value) : value;
    if (decoded is! List) return const [];
    return decoded.whereType<Map>().map((item) => item.map(
      (key, entry) => MapEntry(key.toString(), entry),
    )).toList();
  } catch (_) {
    return const [];
  }
}

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  return int.tryParse(value?.toString() ?? '') ?? 0;
}
