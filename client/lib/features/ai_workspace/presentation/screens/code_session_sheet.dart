import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/ai_dev_session.dart';
import '../controllers/ai_workspace_controller.dart';

class CodeSessionSheet extends ConsumerStatefulWidget {
  const CodeSessionSheet({super.key});

  @override
  ConsumerState<CodeSessionSheet> createState() => _CodeSessionSheetState();
}

class _CodeSessionSheetState extends ConsumerState<CodeSessionSheet> {
  final _titleController = TextEditingController();
  int? _projectId;
  String? _executorId;
  String _approvalPolicy = 'safe_auto';

  @override
  void dispose() {
    _titleController.dispose();
    super.dispose();
  }

  Future<void> _createSession(
    List<CodeProject> projects,
    List<CodeExecutor> executors,
  ) async {
    final projectId = _selectedProjectId(projects);
    final executor = _selectedExecutor(executors);
    if (projectId == null || executor == null) return;
    final policy = executor.approvalPolicies.contains(_approvalPolicy)
        ? _approvalPolicy
        : executor.approvalPolicies.firstOrNull ?? 'full_auto';
    try {
      await ref
          .read(aiWorkspaceControllerProvider.notifier)
          .createSession(
            projectId: projectId,
            executorId: executor.id,
            approvalPolicy: policy,
            title: _titleController.text,
          );
      if (mounted) Navigator.of(context).pop();
    } catch (_) {}
  }

  int? _selectedProjectId(List<CodeProject> projects) {
    if (projects.any((item) => item.id == _projectId)) return _projectId;
    return projects.firstOrNull?.id;
  }

  CodeExecutor? _selectedExecutor(List<CodeExecutor> executors) {
    final available = executors
        .where((item) => item.available && item.supportsAutomation)
        .toList();
    return available.cast<CodeExecutor?>().firstWhere(
      (item) => item?.id == _executorId,
      orElse: () => available.firstOrNull,
    );
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(aiWorkspaceControllerProvider);
    final projects = state.projects;
    final availableExecutors = state.executors
        .where((item) => item.available && item.supportsAutomation)
        .toList();
    final selectedExecutor = _selectedExecutor(state.executors);
    final policies = selectedExecutor?.approvalPolicies ?? const <String>[];
    final selectedPolicy = policies.contains(_approvalPolicy)
        ? _approvalPolicy
        : policies.firstOrNull;

    return SafeArea(
      child: Padding(
        padding: EdgeInsets.only(
          left: 20,
          right: 20,
          top: 12,
          bottom: MediaQuery.viewInsetsOf(context).bottom + 20,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Colors.grey.shade300,
                    borderRadius: BorderRadius.circular(99),
                  ),
                ),
              ),
              const SizedBox(height: 18),
              Text('Code 会话', style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 12),
              if (state.isLoading && state.sessions.isEmpty)
                const Center(child: CircularProgressIndicator())
              else ...[
                _SessionList(
                  sessions: state.sessions,
                  selectedId: state.currentSession?.id,
                  onSelected: (session) async {
                    await ref
                        .read(aiWorkspaceControllerProvider.notifier)
                        .selectSession(session);
                    if (context.mounted) Navigator.of(context).pop();
                  },
                ),
                const Divider(height: 32),
                Text('新建会话', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 12),
                if (projects.isEmpty)
                  const _EmptyHint(text: '还没有 Code 项目，请先在管理台创建项目。')
                else if (availableExecutors.isEmpty)
                  const _EmptyHint(text: '服务器没有可用的 Code 执行器。')
                else ...[
                  DropdownButtonFormField<int>(
                    initialValue: _selectedProjectId(projects),
                    decoration: const InputDecoration(labelText: '项目'),
                    items: projects
                        .map(
                          (project) => DropdownMenuItem(
                            value: project.id,
                            child: Text(project.name),
                          ),
                        )
                        .toList(),
                    onChanged: (value) => setState(() => _projectId = value),
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    initialValue: selectedExecutor?.id,
                    decoration: const InputDecoration(labelText: '执行器'),
                    items: availableExecutors
                        .map(
                          (executor) => DropdownMenuItem(
                            value: executor.id,
                            child: Text(
                              executor.version.isEmpty
                                  ? executor.name
                                  : '${executor.name} · ${executor.version}',
                            ),
                          ),
                        )
                        .toList(),
                    onChanged: (value) {
                      setState(() {
                        _executorId = value;
                        final executor = availableExecutors.firstWhere(
                          (item) => item.id == value,
                        );
                        if (!executor.approvalPolicies.contains(
                          _approvalPolicy,
                        )) {
                          _approvalPolicy =
                              executor.approvalPolicies.firstOrNull ??
                              'full_auto';
                        }
                      });
                    },
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _titleController,
                    decoration: const InputDecoration(labelText: '会话名称（可选）'),
                  ),
                  const SizedBox(height: 16),
                  const Text('审批策略'),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: policies
                        .map(
                          (policy) => ChoiceChip(
                            label: Text(_policyLabel(policy)),
                            selected: policy == selectedPolicy,
                            onSelected: (_) {
                              setState(() => _approvalPolicy = policy);
                            },
                          ),
                        )
                        .toList(),
                  ),
                  if (selectedPolicy != null) ...[
                    const SizedBox(height: 8),
                    Text(
                      _policyDescription(selectedPolicy),
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                  const SizedBox(height: 8),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton.icon(
                      onPressed: state.isActionLoading
                          ? null
                          : () => _createSession(projects, state.executors),
                      icon: state.isActionLoading
                          ? const SizedBox.square(
                              dimension: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.add_rounded),
                      label: const Text('创建并打开'),
                    ),
                  ),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }

  String _policyLabel(String policy) => switch (policy) {
    'manual' => '手动确认',
    'safe_auto' => '安全自动',
    'full_auto' => '完全自动',
    _ => policy,
  };

  String _policyDescription(String policy) => switch (policy) {
    'manual' => '关键执行前等待确认',
    'safe_auto' => '仅高风险动作需要确认',
    'full_auto' => '不暂停执行，适合可信项目',
    _ => '',
  };
}

class _SessionList extends StatelessWidget {
  const _SessionList({
    required this.sessions,
    required this.selectedId,
    required this.onSelected,
  });

  final List<AiDevSession> sessions;
  final int? selectedId;
  final ValueChanged<AiDevSession> onSelected;

  @override
  Widget build(BuildContext context) {
    if (sessions.isEmpty) {
      return const _EmptyHint(text: '暂无历史会话，可以在下方创建。');
    }
    return ConstrainedBox(
      constraints: const BoxConstraints(maxHeight: 220),
      child: ListView.builder(
        shrinkWrap: true,
        itemCount: sessions.length,
        itemBuilder: (context, index) {
          final session = sessions[index];
          return ListTile(
            contentPadding: EdgeInsets.zero,
            selected: session.id == selectedId,
            leading: const Icon(Icons.terminal_rounded),
            title: Text(
              session.title.isEmpty ? '会话 #${session.id}' : session.title,
            ),
            subtitle: Text('${session.agentName} · ${session.currentStage}'),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => onSelected(session),
          );
        },
      ),
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.grey.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text),
    );
  }
}
