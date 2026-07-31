import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/dashboard_repository.dart';
import '../../models/monitor_info.dart';
import '../../models/system_info.dart';

final dashboardRepositoryProvider = Provider<DashboardRepository>((ref) {
  return DashboardRepository(ApiClient());
});

class DashboardState {
  final bool isLoading;
  final String? errorMessage;
  final OsInfo? osInfo;
  final SystemCurrentInfo? currentInfo;
  final MonitorSeries monitor;

  const DashboardState({
    this.isLoading = true,
    this.errorMessage,
    this.osInfo,
    this.currentInfo,
    this.monitor = MonitorSeries.empty,
  });

  DashboardState copyWith({
    bool? isLoading,
    String? errorMessage,
    OsInfo? osInfo,
    SystemCurrentInfo? currentInfo,
    MonitorSeries? monitor,
  }) {
    return DashboardState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage, // null 代表没有错误
      osInfo: osInfo ?? this.osInfo,
      currentInfo: currentInfo ?? this.currentInfo,
      monitor: monitor ?? this.monitor,
    );
  }
}

final dashboardControllerProvider = NotifierProvider<DashboardController, DashboardState>(DashboardController.new);

class DashboardController extends Notifier<DashboardState> {
  late DashboardRepository _repo;
  Timer? _timer;
  IoNetInfo? _prevIoNet;
  DateTime? _prevIoNetAt;
  List<double> _netUpSeries = const [];
  List<double> _netDownSeries = const [];
  List<double> _ioReadSeries = const [];
  List<double> _ioWriteSeries = const [];

  @override
  DashboardState build() {
    _repo = ref.watch(dashboardRepositoryProvider);
    // 初次加载数据
    _loadInitialData();

    // 页面销毁时清理定时器
    ref.onDispose(() {
      _timer?.cancel();
    });

    return const DashboardState();
  }

  Future<void> _loadInitialData() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      // 并发请求 OS 信息和实时状态
      final results = await Future.wait([
        _repo.getOsInfo(),
        _repo.getCurrentInfo(),
        _repo.getIoNetInfo(),
      ]);

      final ioNet = results[2] as IoNetInfo;
      final now = DateTime.now();
      _prevIoNet = ioNet;
      _prevIoNetAt = now;
      _netUpSeries = const [];
      _netDownSeries = const [];
      _ioReadSeries = const [];
      _ioWriteSeries = const [];

      state = state.copyWith(
        isLoading: false,
        osInfo: results[0] as OsInfo,
        currentInfo: results[1] as SystemCurrentInfo,
        monitor: state.monitor.copyWith(
          totalSentBytes: ioNet.netBytesSent,
          totalRecvBytes: ioNet.netBytesRecv,
          totalReadBytes: ioNet.ioReadBytes,
          totalWriteBytes: ioNet.ioWriteBytes,
        ),
      );

      // 开启每 3 秒一次的定时刷新以获取实时状态
      _startPolling();
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: '无法加载服务器状态: $e',
      );
    }
  }

  void _startPolling() {
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 3), (timer) async {
      try {
        final results = await Future.wait([
          _repo.getCurrentInfo(),
          _repo.getIoNetInfo(),
        ]);
        final current = results[0] as SystemCurrentInfo;
        final ioNet = results[1] as IoNetInfo;

        final now = DateTime.now();
        final prev = _prevIoNet;
        final prevAt = _prevIoNetAt;
        double upBps = 0;
        double downBps = 0;
        double readBps = 0;
        double writeBps = 0;
        if (prev != null && prevAt != null) {
          final dt = now.difference(prevAt).inMilliseconds / 1000.0;
          if (dt > 0) {
            final dSent = (ioNet.netBytesSent - prev.netBytesSent);
            final dRecv = (ioNet.netBytesRecv - prev.netBytesRecv);
            final dRead = (ioNet.ioReadBytes - prev.ioReadBytes);
            final dWrite = (ioNet.ioWriteBytes - prev.ioWriteBytes);
            if (dSent > 0) upBps = dSent / dt;
            if (dRecv > 0) downBps = dRecv / dt;
            if (dRead > 0) readBps = dRead / dt;
            if (dWrite > 0) writeBps = dWrite / dt;
          }
        }

        _prevIoNet = ioNet;
        _prevIoNetAt = now;

        _netUpSeries = _appendPoint(_netUpSeries, upBps);
        _netDownSeries = _appendPoint(_netDownSeries, downBps);
        _ioReadSeries = _appendPoint(_ioReadSeries, readBps);
        _ioWriteSeries = _appendPoint(_ioWriteSeries, writeBps);

        // 仅在不报错且仍在屏幕时更新，避免无脑覆盖加载态
        state = state.copyWith(
          currentInfo: current,
          errorMessage: null,
          monitor: state.monitor.copyWith(
            netUpBps: upBps,
            netDownBps: downBps,
            ioReadBps: readBps,
            ioWriteBps: writeBps,
            totalSentBytes: ioNet.netBytesSent,
            totalRecvBytes: ioNet.netBytesRecv,
            totalReadBytes: ioNet.ioReadBytes,
            totalWriteBytes: ioNet.ioWriteBytes,
            netUpSeries: _netUpSeries,
            netDownSeries: _netDownSeries,
            ioReadSeries: _ioReadSeries,
            ioWriteSeries: _ioWriteSeries,
          ),
        );
      } catch (e) {
        // 如果长连接断开或 token 过期，可以在此统一抛出
        // 为避免频繁弹窗，只在 log 中输出，或更新小范围报错状态
      }
    });
  }

  /// 手动触发刷新（如下拉刷新时调用）
  Future<void> refresh() async {
    _timer?.cancel();
    await _loadInitialData();
  }

  List<double> _appendPoint(List<double> series, double value) {
    final list = [...series, value];
    const maxPoints = 30;
    if (list.length <= maxPoints) return list;
    return list.sublist(list.length - maxPoints);
  }
}
