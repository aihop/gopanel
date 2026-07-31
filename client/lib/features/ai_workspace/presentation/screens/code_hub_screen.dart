import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../shared/widgets/panel/glass_tabs.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';
import '../../models/ai_dev_session.dart';
import '../controllers/ai_workspace_controller.dart';
import '../widgets/code_hub_cards.dart';
import 'code_session_sheet.dart';
import 'code_terminal_screen.dart';

enum _CodeListMode { sessions, systemTasks }

class CodeHubScreen extends ConsumerStatefulWidget {
  const CodeHubScreen({super.key});

  @override
  ConsumerState<CodeHubScreen> createState() => _CodeHubScreenState();
}

class _CodeHubScreenState extends ConsumerState<CodeHubScreen> {
  _CodeListMode _mode = _CodeListMode.sessions;

  Future<void> _refresh() async {
    await Future.wait([
      ref.read(aiWorkspaceControllerProvider.notifier).loadWorkspace(),
      ref.read(taskCenterControllerProvider.notifier).refresh(),
    ]);
  }

  void _openSessionCreator() {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: false,
      builder: (_) => const CodeSessionSheet(),
    );
  }

  Future<void> _openSession(AiDevSession session) async {
    await ref
        .read(aiWorkspaceControllerProvider.notifier)
        .selectSession(session);
    if (!mounted) return;
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeTerminalScreen(
          session: session,
          nativeProtocol: _usesNativeProtocol(
            ref.read(aiWorkspaceControllerProvider).executors,
            session.agentName,
          ),
          projectName: _projectName(
            ref.read(aiWorkspaceControllerProvider).projects,
            session.projectId,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final workspace = ref.watch(aiWorkspaceControllerProvider);
    final taskCenter = ref.watch(taskCenterControllerProvider);
    final systemTasks = [
      ...taskCenter.localTasks,
      ...taskCenter.tasks,
    ].where((task) => task.type != TaskType.ai).toList();

    return Scaffold(
      appBar: AppBar(
        title: const Text('开发'),
        actions: [
          IconButton(
            tooltip: '新建开发会话',
            onPressed: _openSessionCreator,
            icon: const Icon(Icons.add_rounded),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(16),
          children: [
            GlassTabs<_CodeListMode>(
              items: const [
                GlassTabItem(value: _CodeListMode.sessions, label: '开发会话'),
                GlassTabItem(value: _CodeListMode.systemTasks, label: '系统任务'),
              ],
              selected: _mode,
              onChanged: (mode) => setState(() => _mode = mode),
            ),
            const SizedBox(height: 16),
            if (_mode == _CodeListMode.sessions)
              ..._buildSessionList(workspace)
            else
              ..._buildTaskList(taskCenter, systemTasks),
          ],
        ),
      ),
      floatingActionButton: _mode == _CodeListMode.sessions
          ? FloatingActionButton.extended(
              onPressed: _openSessionCreator,
              icon: const Icon(Icons.add_rounded),
              label: const Text('新建会话'),
            )
          : null,
    );
  }

  List<Widget> _buildSessionList(AiWorkspaceState state) {
    if (state.isLoading && state.sessions.isEmpty) {
      return const [
        SizedBox(
          height: 180,
          child: Center(child: CircularProgressIndicator()),
        ),
      ];
    }
    if (state.errorMessage != null && state.sessions.isEmpty) {
      return [
        CodeHubErrorCard(
          message: state.errorMessage!,
          onRetry: ref
              .read(aiWorkspaceControllerProvider.notifier)
              .loadWorkspace,
        ),
      ];
    }
    if (state.sessions.isEmpty) {
      return [
        CodeHubEmptyCard(
          icon: Icons.terminal_rounded,
          title: '还没有开发会话',
          description: '创建会话后，可以在手机上查看执行过程、发送指令并处理审批。',
          actionLabel: '创建会话',
          onAction: _openSessionCreator,
        ),
      ];
    }

    return state.sessions
        .map(
          (session) => Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: CodeSessionListCard(
              session: session,
              projectName: _projectName(state.projects, session.projectId),
              onTap: () => _openSession(session),
            ),
          ),
        )
        .toList();
  }

  List<Widget> _buildTaskList(TaskCenterState state, List<TaskEntity> tasks) {
    if (state.isLoading && tasks.isEmpty) {
      return const [
        SizedBox(
          height: 180,
          child: Center(child: CircularProgressIndicator()),
        ),
      ];
    }
    if (state.errorMessage != null && tasks.isEmpty) {
      return [
        CodeHubErrorCard(
          message: state.errorMessage!,
          onRetry: ref.read(taskCenterControllerProvider.notifier).refresh,
        ),
      ];
    }
    if (tasks.isEmpty) {
      return const [
        CodeHubEmptyCard(
          icon: Icons.playlist_add_check_rounded,
          title: '暂无系统任务',
          description: '网站部署、流水线、应用安装等任务会显示在这里。',
        ),
      ];
    }

    return tasks
        .map(
          (task) => Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: CodeSystemTaskCard(
              task: task,
              onTap: () {
                Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => TaskDetailScreen(task: task),
                  ),
                );
              },
            ),
          ),
        )
        .toList();
  }

  String _projectName(List<CodeProject> projects, int projectId) {
    for (final project in projects) {
      if (project.id == projectId) return project.name;
    }
    return '开发项目';
  }

  bool _usesNativeProtocol(List<CodeExecutor> executors, String executorId) {
    for (final executor in executors) {
      if (executor.id == executorId) return executor.nativeTerminal;
    }
    return executorId != 'terminal';
  }
}
