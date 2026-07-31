import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/apps_repository.dart';
import '../../models/app_install_info.dart';

final appsRepositoryProvider = Provider<AppsRepository>((ref) {
  return AppsRepository(ApiClient());
});

class AppsListState {
  final bool isLoading;
  final String? errorMessage;
  final List<AppInstallInfo> apps;
  final String filterStatus; // 'all', 'Running', 'Stopped'

  const AppsListState({
    this.isLoading = true,
    this.errorMessage,
    this.apps = const [],
    this.filterStatus = 'all',
  });

  AppsListState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<AppInstallInfo>? apps,
    String? filterStatus,
  }) {
    return AppsListState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      apps: apps ?? this.apps,
      filterStatus: filterStatus ?? this.filterStatus,
    );
  }
}

final appsControllerProvider = NotifierProvider<AppsController, AppsListState>(
  AppsController.new,
);

class AppsController extends Notifier<AppsListState> {
  late AppsRepository _repo;

  @override
  AppsListState build() {
    _repo = ref.watch(appsRepositoryProvider);
    _loadApps();
    return const AppsListState();
  }

  /// 获取或刷新应用列表
  Future<void> _loadApps({String? status}) async {
    final targetStatus = status ?? state.filterStatus;
    state = state.copyWith(
      isLoading: true,
      errorMessage: null,
      filterStatus: targetStatus,
    );

    try {
      final list = await _repo.getInstalledApps(
        page: 1,
        pageSize: 100, // 移动端暂且取前 100 个
        status: targetStatus,
      );
      state = state.copyWith(isLoading: false, apps: list);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: '无法获取已安装应用列表: $e');
    }
  }

  /// 外部手动刷新
  Future<void> refresh() async {
    await _loadApps();
  }

  /// 修改过滤状态并刷新列表
  Future<void> setFilter(String status) async {
    if (state.filterStatus == status) return;
    await _loadApps(status: status);
  }

  /// 对应用执行操作（启停/重启/重建）
  /// GoPanel 的应用操作通常耗时较长（docker compose 执行），所以这里同样需要捕获并展示异常
  Future<bool> operateApp(int installId, String operation) async {
    try {
      await _repo.operateApp(installId: installId, operation: operation);

      // 成功后重新刷新列表获取最新状态（可能是 Installing 或 Stopped 等状态流转）
      await refresh();
      return true;
    } catch (e) {
      throw Exception('应用操作失败: $e');
    }
  }
}
