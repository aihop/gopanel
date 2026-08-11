class CodeExecutionRunDetail {
  final int id;
  final int sessionId;
  final int taskId;
  final int instructionId;
  final String executorId;
  final String model;
  final String prompt;
  final String output;
  final String rawOutput;
  final String status;
  final int exitCode;
  final int durationMs;
  final int inputTokens;
  final int outputTokens;
  final int cachedInputTokens;
  final int reasoningTokens;
  final int totalTokens;
  final String errorMessage;
  final DateTime? startedAt;
  final DateTime? completedAt;
  final DateTime? createdAt;

  const CodeExecutionRunDetail({
    required this.id,
    required this.sessionId,
    required this.taskId,
    required this.instructionId,
    required this.executorId,
    required this.model,
    required this.prompt,
    required this.output,
    required this.rawOutput,
    required this.status,
    required this.exitCode,
    required this.durationMs,
    required this.inputTokens,
    required this.outputTokens,
    required this.cachedInputTokens,
    required this.reasoningTokens,
    required this.totalTokens,
    required this.errorMessage,
    required this.startedAt,
    required this.completedAt,
    required this.createdAt,
  });

  bool get hasTokenUsage =>
      inputTokens > 0 ||
      outputTokens > 0 ||
      cachedInputTokens > 0 ||
      reasoningTokens > 0 ||
      totalTokens > 0;

  String get diagnosticText => [
    'Run #$id',
    'Status: $status',
    'Session: $sessionId',
    'Task: $taskId',
    'Instruction: $instructionId',
    'Executor: $executorId',
    if (model.isNotEmpty) 'Model: $model',
    'Exit code: $exitCode',
    'Duration: $durationMs ms',
    'Tokens: $totalTokens',
    if (errorMessage.isNotEmpty) 'Error:\n$errorMessage',
    if (prompt.isNotEmpty) 'Prompt:\n$prompt',
    if (output.isNotEmpty) 'Output:\n$output',
    if (rawOutput.isNotEmpty) 'Raw output:\n$rawOutput',
  ].join('\n\n');

  factory CodeExecutionRunDetail.fromJson(Map<String, dynamic> json) {
    int number(String key) => (json[key] as num?)?.toInt() ?? 0;
    DateTime? time(String key) =>
        DateTime.tryParse((json[key] ?? '').toString());
    return CodeExecutionRunDetail(
      id: number('id'),
      sessionId: number('sessionId'),
      taskId: number('taskId'),
      instructionId: number('instructionId'),
      executorId: (json['executorId'] ?? '').toString(),
      model: (json['model'] ?? '').toString(),
      prompt: (json['prompt'] ?? '').toString(),
      output: (json['output'] ?? '').toString(),
      rawOutput: (json['rawOutput'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      exitCode: number('exitCode'),
      durationMs: number('durationMs'),
      inputTokens: number('inputTokens'),
      outputTokens: number('outputTokens'),
      cachedInputTokens: number('cachedInputTokens'),
      reasoningTokens: number('reasoningTokens'),
      totalTokens: number('totalTokens'),
      errorMessage: (json['errorMessage'] ?? '').toString(),
      startedAt: time('startedAt'),
      completedAt: time('completedAt'),
      createdAt: time('createdAt'),
    );
  }
}
