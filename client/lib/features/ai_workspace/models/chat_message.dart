class ChatMessage {
  final String id;
  final String text;
  final bool isUser; // true: 用户发送, false: AI 回复
  final DateTime timestamp;

  ChatMessage({
    required this.id,
    required this.text,
    required this.isUser,
    required this.timestamp,
  });

  factory ChatMessage.fromAiMessageJson(Map<String, dynamic> json) {
    final role = (json['role'] ?? '').toString().toLowerCase();
    return ChatMessage(
      id: (json['id'] ?? '').toString(),
      text: (json['content'] ?? '').toString(),
      isUser: role == 'user',
      timestamp:
          DateTime.tryParse((json['createdAt'] ?? '').toString()) ??
          DateTime.now(),
    );
  }
}
