import 'package:flutter/material.dart';

import '../../models/code_workspace_file.dart';
import '../code_workspace_text.dart';

class CodeWorkspaceEditor extends StatefulWidget {
  const CodeWorkspaceEditor({
    super.key,
    required this.file,
    required this.content,
    required this.isDirty,
    required this.isSaving,
    required this.onChanged,
    required this.onSave,
  });

  final CodeSessionFile file;
  final String content;
  final bool isDirty;
  final bool isSaving;
  final ValueChanged<String> onChanged;
  final VoidCallback onSave;

  @override
  State<CodeWorkspaceEditor> createState() => _CodeWorkspaceEditorState();
}

class _CodeWorkspaceEditorState extends State<CodeWorkspaceEditor> {
  late final TextEditingController _textController;
  final _verticalScrollController = ScrollController();
  final _horizontalScrollController = ScrollController();
  final _focusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _textController = TextEditingController(text: widget.content);
  }

  @override
  void dispose() {
    _textController.dispose();
    _verticalScrollController.dispose();
    _horizontalScrollController.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          width: double.infinity,
          color: const Color(0xFF111827),
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  '${CodeWorkspaceText.t(context, 'files.editing')} · ${_formatSize(widget.file.size)} · ${widget.file.path}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(color: Colors.white54, fontSize: 11),
                ),
              ),
              const SizedBox(width: 8),
              Text(
                CodeWorkspaceText.t(
                  context,
                  widget.isDirty ? 'files.unsaved' : 'files.saved',
                ),
                style: TextStyle(
                  color: widget.isDirty
                      ? const Color(0xFFFBBF24)
                      : const Color(0xFF4ADE80),
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: Scrollbar(
            controller: _verticalScrollController,
            child: SingleChildScrollView(
              controller: _verticalScrollController,
              padding: const EdgeInsets.all(12),
              child: Scrollbar(
                controller: _horizontalScrollController,
                notificationPredicate: (notification) =>
                    notification.metrics.axis == Axis.horizontal,
                child: SingleChildScrollView(
                  controller: _horizontalScrollController,
                  scrollDirection: Axis.horizontal,
                  child: SizedBox(
                    width: MediaQuery.sizeOf(context).width < 900
                        ? 900
                        : MediaQuery.sizeOf(context).width - 24,
                    child: TextField(
                      controller: _textController,
                      focusNode: _focusNode,
                      onChanged: widget.onChanged,
                      maxLines: null,
                      minLines: 24,
                      keyboardType: TextInputType.multiline,
                      textInputAction: TextInputAction.newline,
                      autocorrect: false,
                      enableSuggestions: false,
                      smartDashesType: SmartDashesType.disabled,
                      smartQuotesType: SmartQuotesType.disabled,
                      style: const TextStyle(
                        color: Color(0xFFE2E8F0),
                        fontFamily: 'monospace',
                        fontSize: 13,
                        height: 1.55,
                      ),
                      cursorColor: const Color(0xFF60A5FA),
                      decoration: const InputDecoration(
                        isCollapsed: true,
                        filled: false,
                        border: InputBorder.none,
                        enabledBorder: InputBorder.none,
                        focusedBorder: InputBorder.none,
                        contentPadding: EdgeInsets.zero,
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
        SafeArea(
          top: false,
          child: Container(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 10),
            decoration: const BoxDecoration(
              color: Color(0xFF0F172A),
              border: Border(top: BorderSide(color: Color(0xFF243247))),
            ),
            child: SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: widget.isDirty && !widget.isSaving
                    ? widget.onSave
                    : null,
                icon: widget.isSaving
                    ? const SizedBox.square(
                        dimension: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.save_outlined),
                label: Text(CodeWorkspaceText.t(context, 'files.save')),
              ),
            ),
          ),
        ),
      ],
    );
  }

  String _formatSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / 1024 / 1024).toStringAsFixed(1)} MB';
  }
}
