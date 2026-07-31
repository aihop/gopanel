import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/app_theme.dart';
import '../../../features/server/presentation/screens/dashboard_screen.dart';
import '../../../features/settings/presentation/screens/settings_screen.dart';
import '../../../features/ai_workspace/presentation/screens/code_hub_screen.dart';
import '../../../features/resources/presentation/screens/resources_screen.dart';
import '../controllers/main_scaffold_controller.dart';

/// App 的主导航框架 (带有底部 NavigationBar)
/// 遵循 PROMPT.md 规范：采用扁平、低噪声的蓝色系底栏
class MainScaffoldScreen extends ConsumerWidget {
  const MainScaffoldScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final currentIndex = ref.watch(mainScaffoldIndexProvider);
    const pages = [
      DashboardScreen(),
      CodeHubScreen(),
      ResourcesScreen(),
      SettingsScreen(),
    ];
    return Scaffold(
      body: IndexedStack(index: currentIndex, children: pages),
      bottomNavigationBar: SafeArea(
        minimum: const EdgeInsets.fromLTRB(14, 0, 14, 10),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(24),
            border: Border.all(color: AppTheme.border.withValues(alpha: 0.8)),
            boxShadow: [
              BoxShadow(
                color: AppTheme.textPrimary.withValues(alpha: 0.08),
                blurRadius: 24,
                offset: const Offset(0, 8),
              ),
            ],
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(24),
            child: NavigationBar(
              height: 68,
              elevation: 0,
              backgroundColor: Colors.white,
              indicatorColor: AppTheme.primaryBlueLight,
              labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
              selectedIndex: currentIndex,
              onDestinationSelected: (index) {
                ref.read(mainScaffoldIndexProvider.notifier).setIndex(index);
              },
              destinations: const [
                NavigationDestination(
                  icon: Icon(Icons.dashboard_outlined),
                  selectedIcon: Icon(Icons.dashboard_rounded),
                  label: '概览',
                ),
                NavigationDestination(
                  icon: Icon(Icons.terminal_outlined),
                  selectedIcon: Icon(Icons.terminal_rounded),
                  label: '开发',
                ),
                NavigationDestination(
                  icon: Icon(Icons.widgets_outlined),
                  selectedIcon: Icon(Icons.widgets_rounded),
                  label: '资源',
                ),
                NavigationDestination(
                  icon: Icon(Icons.settings_outlined),
                  selectedIcon: Icon(Icons.settings_rounded),
                  label: '设置',
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
