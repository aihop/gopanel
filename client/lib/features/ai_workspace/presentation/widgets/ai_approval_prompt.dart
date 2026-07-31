import 'package:flutter/material.dart';

import '../../models/ai_dev_session.dart';

class AiApprovalPrompt extends StatefulWidget {
  const AiApprovalPrompt({
    super.key,
    required this.approval,
    required this.loading,
    required this.onDecision,
  });

  final AiApproval approval;
  final bool loading;
  final Future<void> Function(bool approved, String reason) onDecision;

  @override
  State<AiApprovalPrompt> createState() => _AiApprovalPromptState();
}

class _AiApprovalPromptState extends State<AiApprovalPrompt> {
  final _reasonController = TextEditingController();

  @override
  void didUpdateWidget(covariant AiApprovalPrompt oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.approval.id != widget.approval.id) {
      _reasonController.clear();
    }
  }

  @override
  void dispose() {
    _reasonController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF3B2612),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.orangeAccent.withValues(alpha: 0.4)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.gpp_maybe_rounded, color: Colors.orangeAccent),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  widget.approval.title.isEmpty
                      ? '高风险操作待确认'
                      : widget.approval.title,
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            widget.approval.content,
            style: const TextStyle(color: Colors.white70, height: 1.5),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _reasonController,
            style: const TextStyle(color: Colors.white),
            decoration: const InputDecoration(
              labelText: '处理说明（可选）',
              labelStyle: TextStyle(color: Colors.white60),
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: widget.loading
                      ? null
                      : () => widget.onDecision(false, _reasonController.text),
                  child: const Text('拒绝'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton(
                  onPressed: widget.loading
                      ? null
                      : () => widget.onDecision(true, _reasonController.text),
                  child: Text(widget.loading ? '处理中...' : '允许执行'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
