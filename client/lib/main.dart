import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/router/app_router.dart';
import 'core/storage/storage_service.dart';
import 'core/theme/app_theme.dart';

void main() async {
  // 确保 Flutter 绑定初始化完成
  WidgetsFlutterBinding.ensureInitialized();

  // 挂载必要的初始化逻辑，如 SharedPreferences
  await StorageService.init();

  // 使用 ProviderScope 包裹整个应用以开启 Riverpod
  runApp(const ProviderScope(child: GoPanelApp()));
}

class GoPanelApp extends StatelessWidget {
  const GoPanelApp({super.key});

  @override
  Widget build(BuildContext context) {
    // 注入主题和 GoRouter
    return MaterialApp.router(
      title: 'GoPanel',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      // 如果后续需要深色模式，可补 darkTheme: AppTheme.darkTheme,
      routerConfig: AppRouter.router,
    );
  }
}
