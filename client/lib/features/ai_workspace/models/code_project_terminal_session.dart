class CodeProjectTerminalSession {
  final int id;
  final String status;
  final String shell;
  final String workDir;
  final String errorMessage;

  const CodeProjectTerminalSession({
    required this.id,
    required this.status,
    required this.shell,
    required this.workDir,
    required this.errorMessage,
  });

  factory CodeProjectTerminalSession.fromJson(Map<String, dynamic> json) {
    return CodeProjectTerminalSession(
      id: (json['id'] as num?)?.toInt() ?? 0,
      status: (json['status'] ?? '').toString(),
      shell: (json['shell'] ?? '').toString(),
      workDir: (json['workDir'] ?? '').toString(),
      errorMessage: (json['errorMessage'] ?? '').toString(),
    );
  }
}
