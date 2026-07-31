import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/ai_dev_session.dart';
import '../controllers/ai_workspace_controller.dart';
import '../widgets/code_hub_cards.dart';
import 'code_session_sheet.dart';
import 'code_terminal_screen.dart';

class CodeHubScreen extends ConsumerStatefulWidget {
  const CodeHubScreen({super.key});

  @override
  ConsumerState<CodeHubScreen> createState() => _CodeHubScreenState();
}

class _CodeHubScreenState extends ConsumerState<CodeHubScreen> {
  Future<void> _refresh() async {
    await ref.read(aiWorkspaceControllerProvider.notifier).loadWorkspace();
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
    final workspace = ref.read(aiWorkspaceControllerProvider);
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CodeTerminalScreen(
          session: workspace.currentSession ?? session,
          task: workspace.currentTask,
          nativeProtocol: _usesNativeProtocol(
            workspace.executors,
            session.agentName,
          ),
          projectName: _projectName(workspace.projects, session.projectId),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final workspace = ref.watch(aiWorkspaceControllerProvider);

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
          children: _buildSessionList(workspace),
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _openSessionCreator,
        icon: const Icon(Icons.add_rounded),
        label: const Text('新建会话'),
      ),
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
