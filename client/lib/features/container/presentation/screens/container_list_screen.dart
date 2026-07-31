import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/container_info.dart';
import '../controllers/container_controller.dart';

/// 容器列表页面
/// 作为 MainScaffoldScreen 的第二个 Tab 内容
class ContainerListScreen extends ConsumerStatefulWidget {
  final bool embedded;

  const ContainerListScreen({super.key, this.embedded = false});
  const ContainerListScreen.embedded({super.key}) : embedded = true;

  @override
  ConsumerState<ContainerListScreen> createState() =>
      _ContainerListScreenState();
}

class _ContainerListScreenState extends ConsumerState<ContainerListScreen> {
  // 当前正在操作的容器（防重复点击）
  String? _operatingContainerName;

  Future<void> _handleOperation(
    ContainerInfo container,
    String operation,
  ) async {
    // 危险操作或停止操作需要二次确认
    if (operation == ContainerOp.stop ||
        operation == ContainerOp.restart ||
        operation == ContainerOp.remove) {
      final confirmed = await _showConfirmDialog(
        context,
        operation,
        container.name,
      );
      if (confirmed != true) return;
    }

    if (!mounted) return;

    setState(() {
      _operatingContainerName = container.name;
    });

    try {
      final controller = ref.read(containerControllerProvider.notifier);
      await controller.operateContainer(container.name, operation);

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('操作成功: $operation'),
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
          _operatingContainerName = null;
        });
      }
    }
  }

  Future<bool?> _showConfirmDialog(
    BuildContext context,
    String operation,
    String containerName,
  ) {
    String title = '确认操作';
    String content = '确定要对容器 "$containerName" 执行 $operation 操作吗？';

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
    final state = ref.watch(containerControllerProvider);

    if (widget.embedded) {
      return Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 8, 4),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    '容器',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
                  ),
                ),
                PopupMenuButton<String>(
                  icon: const Icon(Icons.filter_list_rounded),
                  iconSize: 20,
                  padding: EdgeInsets.zero,
                  onSelected: (value) {
                    ref
                        .read(containerControllerProvider.notifier)
                        .setFilter(value);
                  },
                  itemBuilder: (context) => [
                    _buildPopupMenuItem('all', '全部', state.filterState),
                    _buildPopupMenuItem('running', '运行中', state.filterState),
                    _buildPopupMenuItem('exited', '已停止', state.filterState),
                  ],
                ),
                IconButton(
                  icon: const Icon(Icons.refresh_rounded),
                  iconSize: 20,
                  visualDensity: VisualDensity.compact,
                  onPressed: () {
                    ref.read(containerControllerProvider.notifier).refresh();
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
        title: const Text('容器'),
        actions: [
          // 状态过滤菜单
          PopupMenuButton<String>(
            icon: const Icon(Icons.filter_list_rounded),
            onSelected: (value) {
              ref.read(containerControllerProvider.notifier).setFilter(value);
            },
            itemBuilder: (context) => [
              _buildPopupMenuItem('all', '全部', state.filterState),
              _buildPopupMenuItem('running', '运行中', state.filterState),
              _buildPopupMenuItem('exited', '已停止', state.filterState),
            ],
          ),
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () {
              ref.read(containerControllerProvider.notifier).refresh();
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
    String currentState,
  ) {
    return PopupMenuItem(
      value: value,
      child: Row(
        children: [
          Icon(
            Icons.check_rounded,
            color: currentState == value ? Colors.blue : Colors.transparent,
            size: 18,
          ),
          const SizedBox(width: 8),
          Text(text),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context, ContainerListState state) {
    if (state.isLoading && state.containers.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.errorMessage != null && state.containers.isEmpty) {
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

    if (state.containers.isEmpty) {
      return const Center(
        child: Text('没有找到符合条件的容器', style: TextStyle(color: Colors.grey)),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(containerControllerProvider.notifier).refresh();
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: state.containers.length,
        separatorBuilder: (context, index) => const SizedBox(height: 16),
        itemBuilder: (context, index) {
          return _buildContainerCard(state.containers[index]);
        },
      ),
    );
  }

  Widget _buildContainerCard(ContainerInfo container) {
    final isRunning = container.isRunning;
    final isOperating = _operatingContainerName == container.name;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                // 状态指示点
                Container(
                  width: 12,
                  height: 12,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: isRunning ? Colors.green : Colors.red,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    container.name,
                    style: const TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.bold,
                      color: Color(0xFF0F172A),
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            // 镜像与时间
            Row(
              children: [
                const Icon(Icons.layers_rounded, size: 16, color: Colors.grey),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    container.image,
                    style: const TextStyle(
                      color: Color(0xFF64748B),
                      fontSize: 13,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                const Icon(
                  Icons.access_time_rounded,
                  size: 16,
                  color: Colors.grey,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    container.status,
                    style: const TextStyle(
                      color: Color(0xFF64748B),
                      fontSize: 13,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
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
                        _handleOperation(container, ContainerOp.restart);
                      },
                    ),
                    const SizedBox(width: 12),
                    _buildActionButton(
                      Icons.stop_rounded,
                      '停止',
                      Colors.red,
                      () {
                        _handleOperation(container, ContainerOp.stop);
                      },
                    ),
                  ] else ...[
                    _buildActionButton(
                      Icons.play_arrow_rounded,
                      '启动',
                      Colors.green,
                      () {
                        _handleOperation(container, ContainerOp.start);
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
