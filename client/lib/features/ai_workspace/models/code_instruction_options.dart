class CodeInstructionOptions {
  final bool autoPreview;
  final bool requireApproval;

  const CodeInstructionOptions({
    this.autoPreview = true,
    this.requireApproval = false,
  });

  CodeInstructionOptions copyWith({bool? autoPreview, bool? requireApproval}) {
    return CodeInstructionOptions(
      autoPreview: autoPreview ?? this.autoPreview,
      requireApproval: requireApproval ?? this.requireApproval,
    );
  }
}
