import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'dart:async';

import '../../../../core/theme/app_theme.dart';
import '../../../ai_workspace/models/ai_dev_session.dart';
import '../../../ai_workspace/models/ai_session_state_info.dart';
import '../../../ai_workspace/presentation/controllers/ai_workspace_controller.dart';
import '../../../ai_workspace/presentation/screens/code_terminal_screen.dart';
import '../../models/task_entity.dart';
import '../../models/task_log.dart';
import '../../models/task_status.dart';
import '../../models/task_type.dart';
import '../../models/task_attention.dart';
import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../controllers/task_center_controller.dart';
import '../widgets/task_ai_session_cards.dart';
import '../../../../shared/widgets/panel/panel_card.dart';

class TaskDetailScreen extends ConsumerStatefulWidget {
  final TaskEntity task;

  const TaskDetailScreen({super.key, required this.task});

  @override
  ConsumerState<TaskDetailScreen> createState() => _TaskDetailScreenState();
}

class _TaskDetailScreenState extends ConsumerState<TaskDetailScreen> {
  bool _loading = true;
  String? _error;
  TaskLog? _log;
  bool _autoRefresh = true;
  bool _fetching = false;
  Timer? _timer;
  final ScrollController _scrollController = ScrollController();
  late TaskStatus _status;
  String? _taskError;
  List<AiPreview> _previews = const [];
  AiSessionStateInfo? _aiSessionState;
  TaskAttention? _attention;
  bool _attentionLoading = false;

  @override
  void initState() {
    super.initState();
    _status = widget.task.status;
    _taskError = widget.task.error;
    _attention = widget.task.attention;
    _load();
    _startAutoRefresh();
  }

  @override
  void dispose() {
    _timer?.cancel();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _load({bool keepLoadingState = false}) async {
    if (_fetching) return;
    _fetching = true;
    setState(() {
      if (!keepLoadingState) {
        _loading = true;
      }
      _error = null;
    });
    try {
      final repo = ref.read(taskRepositoryProvider);
      final log = await repo.getLog(widget.task.id);
      List<AiPreview> previews = const [];
      AiSessionStateInfo? aiSessionState;
      final aiSessionId = _aiSessionId;
      if (aiSessionId != null) {
        aiSessionState = await repo.getAiSessionState(aiSessionId);
        previews = await repo.getAiSessionPreviews(aiSessionId);
      }
      if (!mounted) return;
      setState(() {
        _log = log;
        _previews = previews;
        _aiSessionState = aiSessionState;
        _loading = false;
      });
      if (log.status != null && log.status != _status) {
        setState(() {
          _status = log.status!;
          if (_status == TaskStatus.success) {
            _taskError = null;
          }
        });
        ref
            .read(taskCenterControllerProvider.notifier)
            .updateLocalTask(id: widget.task.id, status: log.status);
      }
      _tailToBottom();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    } finally {
      _fetching = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = widget.task;
    final meta = _log?.meta ?? const <String, String>{};
    final isWebsiteDeploy = t.id.startsWith('websiteDeploy:');
    final isDbSync = t.id.startsWith('dbSync:');
    final isAiSession = _aiSessionId != null;
    return Scaffold(
      appBar: AppBar(
        title: const Text('任务详情'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () => _load(keepLoadingState: true),
          ),
          IconButton(
            icon: Icon(
              _autoRefresh ? Icons.pause_rounded : Icons.play_arrow_rounded,
            ),
            onPressed: () {
              setState(() {
                _autoRefresh = !_autoRefresh;
              });
              if (_autoRefresh) {
                _startAutoRefresh();
              } else {
                _timer?.cancel();
              }
            },
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            PanelCard(
              title: Text(t.title),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _kv('类型', t.type.label),
                  const SizedBox(height: 8),
                  _kv('状态', _status.label),
                  if (isWebsiteDeploy) ...[
                    const SizedBox(height: 12),
                    _kv(
                      'Deploy 状态',
                      meta['deployStatus']?.isNotEmpty == true
                          ? meta['deployStatus']!
                          : '-',
                    ),
                    const SizedBox(height: 8),
                    _kv(
                      '版本号',
                      meta['deployVersion']?.isNotEmpty == true
                          ? meta['deployVersion']!
                          : '-',
                    ),
                    const SizedBox(height: 8),
                    _kv('是否生效', meta['deployActive'] == 'true' ? '是' : '否'),
                  ],
                  if (t.progress != null || _status == TaskStatus.success) ...[
                    const SizedBox(height: 12),
                    LinearProgressIndicator(
                      value:
                          (_status == TaskStatus.success
                                  ? 1.0
                                  : (t.progress ?? 0.0))
                              .clamp(0.0, 1.0),
                      backgroundColor: AppTheme.primaryBlueLight,
                      color: _statusColor(_status),
                      minHeight: 6,
                      borderRadius: BorderRadius.circular(3),
                    ),
                  ],
                  if (_taskError != null && _taskError!.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Text(
                      _taskError!,
                      style: const TextStyle(
                        color: AppTheme.error,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                  if (isDbSync && _status == TaskStatus.success) ...[
                    const SizedBox(height: 14),
                    Row(
                      children: [
                        Expanded(
                          child: ElevatedButton(
                            onPressed: () {
                              ref
                                  .read(mainScaffoldIndexProvider.notifier)
                                  .setIndex(
                                    MainScaffoldIndexController.resourcesIndex,
                                  );
                              Navigator.of(context).pop();
                            },
                            child: const Text('返回资源页'),
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 16),
            if (_attention != null) ...[
              _buildAttentionCard(),
              const SizedBox(height: 16),
            ],
            if (isAiSession && _aiSessionState != null) ...[
              TaskAiSessionSummaryCard(state: _aiSessionState!),
              const SizedBox(height: 16),
            ],
            if (isAiSession && _previews.isNotEmpty) ...[
              TaskAiPreviewCard(previews: _previews),
              const SizedBox(height: 16),
            ],
            PanelCard(
              title: const Text('日志'),
              trailing: TextButton(
                onPressed: _copyLog,
                child: const Text('复制'),
              ),
              child: _buildLogBody(),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildLogBody() {
    if (_loading) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 20),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (_error != null) {
      return Text(_error!, style: const TextStyle(color: AppTheme.error));
    }
    final lines = _log?.lines ?? const [];
    if (lines.isEmpty) {
      return const Text(
        '暂无日志',
        style: TextStyle(color: AppTheme.textSecondary),
      );
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFF0B1220),
        borderRadius: BorderRadius.circular(12),
      ),
      child: SingleChildScrollView(
        controller: _scrollController,
        child: Text(
          lines.join('\n'),
          style: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 12,
            height: 1.4,
            color: Color(0xFFE2E8F0),
          ),
        ),
      ),
    );
  }

  Widget _buildAttentionCard() {
    final attention = _attention!;
    return PanelCard(
      title: const Text('待我处理'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            attention.title,
            style: const TextStyle(
              color: AppTheme.error,
              fontWeight: FontWeight.w800,
            ),
          ),
          if (attention.summary.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              attention.summary,
              style: const TextStyle(
                color: AppTheme.textSecondary,
                height: 1.45,
              ),
            ),
          ],
          const SizedBox(height: 14),
          Wrap(
            spacing: 10,
            runSpacing: 10,
            children: attention.actions
                .map(
                  (action) => ElevatedButton(
                    onPressed: _attentionLoading
                        ? null
                        : () => _handleAttentionAction(action),
                    child: Text(action.label),
                  ),
                )
                .toList(),
          ),
        ],
      ),
    );
  }

  Future<void> _handleAttentionAction(TaskAttentionAction action) async {
    if (action.type == 'open_session') {
      final session = _aiSessionState?.session;
      if (session == null) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('暂时无法打开开发会话，请刷新后重试')));
        return;
      }
      await ref
          .read(aiWorkspaceControllerProvider.notifier)
          .selectSession(session);
      if (!mounted) return;
      final workspace = ref.read(aiWorkspaceControllerProvider);
      final executor = workspace.executors
          .where((item) => item.id == session.agentName)
          .firstOrNull;
      final project = workspace.projects
          .where((item) => item.id == session.projectId)
          .firstOrNull;
      await Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => CodeTerminalScreen(
            session: workspace.currentSession ?? session,
            task: workspace.currentTask,
            nativeProtocol:
                executor?.nativeTerminal ?? session.agentName != 'terminal',
            projectName: project?.name ?? '开发项目',
          ),
        ),
      );
      return;
    }
    if (action.requiresConfirmation) {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(action.label),
          content: Text('确认${action.label}？该操作会立即影响当前开发会话。'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('取消'),
            ),
            ElevatedButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: const Text('确认'),
            ),
          ],
        ),
      );
      if (confirmed != true || !mounted) return;
    }
    setState(() => _attentionLoading = true);
    try {
      await ref.read(taskRepositoryProvider).executeAttentionAction(action);
      if (!mounted) return;
      setState(() {
        _attention = null;
        _attentionLoading = false;
      });
      await ref.read(taskCenterControllerProvider.notifier).refresh();
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('${action.label}成功')));
      await _load(keepLoadingState: true);
    } catch (error) {
      if (!mounted) return;
      setState(() => _attentionLoading = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('${action.label}失败：$error')));
    }
  }

  Widget _kv(String k, String v) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          k,
          style: const TextStyle(
            color: AppTheme.textSecondary,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            v,
            textAlign: TextAlign.right,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
        ),
      ],
    );
  }

  Color _statusColor(TaskStatus s) {
    switch (s) {
      case TaskStatus.running:
        return AppTheme.primaryBlue;
      case TaskStatus.success:
        return AppTheme.success;
      case TaskStatus.failed:
        return AppTheme.error;
    }
  }

  Future<void> _copyLog() async {
    final text = _log?.lines.join('\n') ?? '';
    if (text.isEmpty) return;
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('日志已复制')));
  }

  void _startAutoRefresh() {
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 4), (_) async {
      if (!_autoRefresh || !mounted) return;
      await _load(keepLoadingState: true);
    });
  }

  void _tailToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOut,
      );
    });
  }

  int? get _aiSessionId {
    if (!widget.task.id.startsWith('aiSession:')) return null;
    final id = int.tryParse(widget.task.id.substring('aiSession:'.length));
    if (id == null || id <= 0) return null;
    return id;
  }
}
