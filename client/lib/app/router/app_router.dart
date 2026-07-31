import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../features/auth/presentation/screens/qr_scanner_screen.dart';
import '../../features/auth/presentation/screens/server_login_screen.dart';
import '../../features/server/presentation/screens/server_list_screen.dart';
import '../presentation/screens/main_scaffold_screen.dart';

/// 全局路由配置
/// 使用 GoRouter 进行强类型、声明式路由管理
class AppRouter {
  // 路由路径常量
  static const String serverListPath = '/';
  static const String loginPath = '/login';
  static const String qrScannerPath = '/qr_scanner';
  static const String dashboardPath = '/dashboard';

  // 路由实例
  static final GoRouter router = GoRouter(
    initialLocation: serverListPath,
    routes: <RouteBase>[
      GoRoute(
        path: serverListPath,
        builder: (BuildContext context, GoRouterState state) {
          // 全局真正的入口首页：服务器聚合列表
          return const ServerListScreen();
        },
      ),
      GoRoute(
        path: loginPath,
        builder: (BuildContext context, GoRouterState state) {
          return const ServerLoginScreen();
        },
      ),
      GoRoute(
        path: qrScannerPath,
        builder: (BuildContext context, GoRouterState state) {
          return const QRScannerScreen();
        },
      ),
      GoRoute(
        path: dashboardPath,
        builder: (BuildContext context, GoRouterState state) {
          // 选中单个服务器后，进入该服务器的概览/管理骨架
          return const MainScaffoldScreen();
        },
      ),
    ],
  );
}
