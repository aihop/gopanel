import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/ai_workspace_repository.dart';
import '../../models/ai_dev_session.dart';
import '../controllers/ai_approval_controller.dart';
import '../controllers/ai_workspace_controller.dart';

class AiApprovalListScreen extends ConsumerStatefulWidget {
  const AiApprovalListScreen({super.key});

  @override
  ConsumerState<AiApprovalListScreen> createState() =>
      _AiApprovalListScreenState();
}

class _AiApprovalListScreenState extends ConsumerState<AiApprovalListScreen> {
  bool _loading = true;
  String? _error;
  List<AiApproval> _approvals = const [];
  final Set<int> _submittingIds = <int>{};

  AiWorkspaceRepository get _repo => ref.read(aiWorkspaceRepositoryProvider);

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final approvals = await _repo.getApprovals();
      ref.invalidate(pendingAiApprovalCountProvider);
      if (!mounted) return;
      setState(() {
        _approvals = approvals;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  Future<void> _decideApproval(AiApproval approval, bool approve) async {
    setState(() {
      _submittingIds.add(approval.id);
    });
    try {
      if (approve) {
        await _repo.approveApproval(approval.id);
      } else {
        await _repo.rejectApproval(approval.id);
      }
      if (!mounted) return;
      setState(() {
        _approvals = _approvals
            .where((item) => item.id != approval.id)
            .toList();
      });
      ref.invalidate(pendingAiApprovalCountProvider);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(approve ? '已允许执行' : '已拒绝执行')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('处理失败: $e')));
    } finally {
      if (mounted) {
        setState(() {
          _submittingIds.remove(approval.id);
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('待审批'),
        actions: [
          IconButton(icon: const Icon(Icons.refresh_rounded), onPressed: _load),
        ],
      ),
      body: RefreshIndicator(onRefresh: _load, child: _buildBody()),
    );
  }

  Widget _buildBody() {
    if (_loading && _approvals.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null && _approvals.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              children: [
                const SizedBox(height: 80),
                const Icon(Icons.gpp_bad_outlined, size: 44),
                const SizedBox(height: 12),
                const Text('审批列表加载失败'),
                const SizedBox(height: 8),
                Text(_error!, textAlign: TextAlign.center),
              ],
            ),
          ),
        ],
      );
    }

    if (_approvals.isEmpty) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 96),
          Center(child: Icon(Icons.verified_user_outlined, size: 44)),
          SizedBox(height: 12),
          Center(child: Text('当前没有待审批的危险操作')),
        ],
      );
    }

    return ListView.separated(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      itemCount: _approvals.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (context, index) {
        final approval = _approvals[index];
        final submitting = _submittingIds.contains(approval.id);
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        approval.title.isEmpty ? '危险操作审批' : approval.title,
                        style: const TextStyle(fontWeight: FontWeight.w700),
                      ),
                    ),
                    const SizedBox(width: 12),
                    _RiskChip(level: approval.riskLevel),
                  ],
                ),
                const SizedBox(height: 10),
                Text(
                  approval.content,
                  style: const TextStyle(color: Color(0xFF334155), height: 1.5),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Text(
                      '会话 #${approval.sessionId}',
                      style: const TextStyle(
                        color: Color(0xFF64748B),
                        fontSize: 12,
                      ),
                    ),
                    if (approval.taskId > 0) ...[
                      const SizedBox(width: 8),
                      Text(
                        '任务 #${approval.taskId}',
                        style: const TextStyle(
                          color: Color(0xFF64748B),
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton(
                        onPressed: submitting
                            ? null
                            : () => _decideApproval(approval, false),
                        child: const Text('拒绝'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: FilledButton(
                        onPressed: submitting
                            ? null
                            : () => _decideApproval(approval, true),
                        child: Text(submitting ? '处理中...' : '允许执行'),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _RiskChip extends StatelessWidget {
  const _RiskChip({required this.level});

  final String level;

  @override
  Widget build(BuildContext context) {
    final normalized = level.isEmpty ? 'medium' : level;
    Color color;
    switch (normalized) {
      case 'high':
        color = Colors.red;
        break;
      case 'low':
        color = Colors.orange;
        break;
      default:
        color = Colors.deepOrange;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        normalized,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}
