import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/container_repository.dart';
import '../../models/container_info.dart';

final containerRepositoryProvider = Provider<ContainerRepository>((ref) {
  return ContainerRepository(ApiClient());
});

class ContainerListState {
  final bool isLoading;
  final String? errorMessage;
  final List<ContainerInfo> containers;
  final String filterState; // 'all', 'running', 'exited' 等

  const ContainerListState({
    this.isLoading = true,
    this.errorMessage,
    this.containers = const [],
    this.filterState = 'all',
  });

  ContainerListState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<ContainerInfo>? containers,
    String? filterState,
  }) {
    return ContainerListState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      containers: containers ?? this.containers,
      filterState: filterState ?? this.filterState,
    );
  }
}

final containerControllerProvider = NotifierProvider<ContainerController, ContainerListState>(ContainerController.new);

class ContainerController extends Notifier<ContainerListState> {
  late ContainerRepository _repo;

  @override
  ContainerListState build() {
    _repo = ref.watch(containerRepositoryProvider);
    // 首次进入加载列表
    _loadContainers();
    return const ContainerListState();
  }

  /// 获取或刷新容器列表
  Future<void> _loadContainers({String? filter}) async {
    final targetFilter = filter ?? state.filterState;
    state = state.copyWith(isLoading: true, errorMessage: null, filterState: targetFilter);

    try {
      final list = await _repo.getContainerList(
        page: 1,
        pageSize: 100, // 移动端暂且获取前100个容器，后续可加懒加载
        state: targetFilter,
      );
      state = state.copyWith(isLoading: false, containers: list);
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: '无法获取容器列表: $e',
      );
    }
  }

  /// 供外部调用的刷新方法
  Future<void> refresh() async {
    await _loadContainers();
  }

  /// 更改过滤状态并重新加载
  Future<void> setFilter(String filterState) async {
    if (state.filterState == filterState) return;
    await _loadContainers(filter: filterState);
  }

  /// 操作容器 (启动/停止/重启)
  /// 操作成功后会自动刷新一次列表，保证状态同步
  Future<bool> operateContainer(String containerName, String operation) async {
    // 操作过程中不整体展示全屏 loading，只依赖外层反馈，防止打断滚动
    try {
      await _repo.operateContainers(
        names: [containerName],
        operation: operation,
      );
      
      // 成功后重新刷新当前列表获取最新状态
      await refresh();
      return true;
    } catch (e) {
      // 若操作失败，抛出错误供 UI 显示 SnackBar
      throw Exception('容器操作失败: $e');
    }
  }
}
