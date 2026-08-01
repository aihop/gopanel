import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/presentation/controllers/main_scaffold_controller.dart';
import '../../../task_center/models/task_entity.dart';
import '../../../task_center/models/task_status.dart';
import '../../../task_center/models/task_type.dart';
import '../../../task_center/presentation/controllers/task_center_controller.dart';
import '../../../task_center/presentation/screens/task_detail_screen.dart';
import '../../models/app_install_info.dart';
import '../controllers/apps_controller.dart';

/// 已安装应用列表页面
/// 作为 MainScaffoldScreen 的第三个 Tab 内容
class AppsListScreen extends ConsumerStatefulWidget {
  final bool embedded;

  const AppsListScreen({super.key, this.embedded = false});
  const AppsListScreen.embedded({super.key}) : embedded = true;

  @override
  ConsumerState<AppsListScreen> createState() => _AppsListScreenState();
}

class _AppsListScreenState extends ConsumerState<AppsListScreen> {
  int? _operatingAppId;

  Future<void> _handleOperation(AppInstallInfo app, String operation) async {
    // 危险操作二次确认
    if (operation == AppOp.stop ||
        operation == AppOp.restart ||
        operation == AppOp.rebuild) {
      final confirmed = await _showConfirmDialog(context, operation, app.name);
      if (confirmed != true) return;
    }

    if (!mounted) return;

    setState(() {
      _operatingAppId = app.id;
    });

    try {
      final controller = ref.read(appsControllerProvider.notifier);
      await controller.operateApp(app.id, operation);

      if (mounted) {
        if (operation == AppOp.rebuild || operation == AppOp.upgrade) {
          final now = DateTime.now();
          final task = TaskEntity(
            id: 'appInstall:${app.name}',
            title: '${app.name} ${operation == AppOp.rebuild ? '重建' : '升级'}',
            type: TaskType.appInstall,
            status: TaskStatus.running,
            startedAt: now,
            updatedAt: now,
            summary: operation,
          );
          ref.read(taskCenterControllerProvider.notifier).addLocalTask(task);
          ref
              .read(mainScaffoldIndexProvider.notifier)
              .setIndex(MainScaffoldIndexController.codeIndex);
          Navigator.of(context).push(
            MaterialPageRoute(builder: (_) => TaskDetailScreen(task: task)),
          );
        }
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('操作已发送: $operation'),
            backgroundColor: Colors.green,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString()), backgroundColor: Colors.red),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _operatingAppId = null;
        });
      }
    }
  }

  Future<bool?> _showConfirmDialog(
    BuildContext context,
    String operation,
    String appName,
  ) {
    String title = '确认操作';
    String content = '确定要对应用 "$appName" 执行 $operation 操作吗？(此操作可能耗时较长)';

    return showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('取消', style: TextStyle(color: Colors.grey)),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('确定'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(appsControllerProvider);

    if (widget.embedded) {
      return Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 8, 4),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    '已安装应用',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
                  ),
                ),
                PopupMenuButton<String>(
                  icon: const Icon(Icons.filter_list_rounded),
                  iconSize: 20,
                  padding: EdgeInsets.zero,
                  onSelected: (value) {
                    ref.read(appsControllerProvider.notifier).setFilter(value);
                  },
                  itemBuilder: (context) => [
                    _buildPopupMenuItem('all', '全部', state.filterStatus),
                    _buildPopupMenuItem('Running', '运行中', state.filterStatus),
                    _buildPopupMenuItem('Stopped', '已停止', state.filterStatus),
                  ],
                ),
                IconButton(
                  icon: const Icon(Icons.refresh_rounded),
                  iconSize: 20,
                  visualDensity: VisualDensity.compact,
                  onPressed: () {
                    ref.read(appsControllerProvider.notifier).refresh();
                  },
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(child: _buildBody(context, state)),
        ],
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('已安装应用'),
        actions: [
          PopupMenuButton<String>(
            icon: const Icon(Icons.filter_list_rounded),
            onSelected: (value) {
              ref.read(appsControllerProvider.notifier).setFilter(value);
            },
            itemBuilder: (context) => [
              _buildPopupMenuItem('all', '全部', state.filterStatus),
              _buildPopupMenuItem('Running', '运行中', state.filterStatus),
              _buildPopupMenuItem('Stopped', '已停止', state.filterStatus),
            ],
          ),
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(appsControllerProvider.notifier).refresh();
            },
          ),
        ],
      ),
      body: _buildBody(context, state),
    );
  }

  PopupMenuItem<String> _buildPopupMenuItem(
    String value,
    String text,
    String currentStatus,
  ) {
    return PopupMenuItem(
      value: value,
      child: Row(
        children: [
          Icon(
            Icons.check_rounded,
            color: currentStatus.toLowerCase() == value.toLowerCase()
                ? Colors.blue
                : Colors.transparent,
            size: 18,
          ),
          const SizedBox(width: 8),
          Text(text),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context, AppsListState state) {
    if (state.isLoading && state.apps.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.errorMessage != null && state.apps.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 64, color: Colors.red),
            const SizedBox(height: 16),
            Text(
              state.errorMessage!,
              style: const TextStyle(color: Colors.red),
            ),
          ],
        ),
      );
    }

    if (state.apps.isEmpty) {
      return const Center(
        child: Text('没有找到已安装的应用', style: TextStyle(color: Colors.grey)),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(appsControllerProvider.notifier).refresh();
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: state.apps.length,
        separatorBuilder: (context, index) => const SizedBox(height: 16),
        itemBuilder: (context, index) {
          return _buildAppCard(state.apps[index]);
        },
      ),
    );
  }

  Widget _buildAppCard(AppInstallInfo app) {
    final isRunning = app.isRunning;
    final isOperating = _operatingAppId == app.id;

    // 状态显示的颜色 (运行绿，停止红，其他黄)
    Color statusColor = Colors.orange;
    if (isRunning) statusColor = Colors.green;
    if (app.isStopped) statusColor = Colors.red;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                // 占位图标，实际 GoPanel 会传一个相对路径或完整 URL
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: const Color(0xFFEFF6FF),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: const Icon(
                    Icons.apps_rounded,
                    color: Color(0xFF2563EB),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        app.name,
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                          color: Color(0xFF0F172A),
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'v${app.version}',
                        style: const TextStyle(
                          color: Color(0xFF64748B),
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),
                // 右侧状态文本
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: statusColor.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    app.status,
                    style: TextStyle(
                      color: statusColor,
                      fontSize: 12,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            // 描述
            if (app.description.isNotEmpty) ...[
              Text(
                app.description,
                style: const TextStyle(color: Color(0xFF64748B), fontSize: 13),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 16),
            ],
            // 操作按钮区
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                if (isOperating)
                  const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                else ...[
                  if (isRunning) ...[
                    _buildActionButton(
                      Icons.restart_alt_rounded,
                      '重启',
                      Colors.orange,
                      () {
                        _handleOperation(app, AppOp.restart);
                      },
                    ),
                    const SizedBox(width: 12),
                    _buildActionButton(
                      Icons.stop_rounded,
                      '停止',
                      Colors.red,
                      () {
                        _handleOperation(app, AppOp.stop);
                      },
                    ),
                  ] else ...[
                    _buildActionButton(
                      Icons.build_rounded,
                      '重建',
                      Colors.blue,
                      () {
                        _handleOperation(app, AppOp.rebuild);
                      },
                    ),
                    const SizedBox(width: 12),
                    _buildActionButton(
                      Icons.play_arrow_rounded,
                      '启动',
                      Colors.green,
                      () {
                        _handleOperation(app, AppOp.start);
                      },
                    ),
                  ],
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButton(
    IconData icon,
    String label,
    Color color,
    VoidCallback onTap,
  ) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: color.withValues(alpha: 0.3)),
        ),
        child: Row(
          children: [
            Icon(icon, size: 16, color: color),
            const SizedBox(width: 4),
            Text(
              label,
              style: TextStyle(
                color: color,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
