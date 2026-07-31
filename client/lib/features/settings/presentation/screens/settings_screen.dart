import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/storage/storage_service.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../website/presentation/screens/dns_account_list_screen.dart';

/// 设置页面
/// 作为 MainScaffoldScreen 的第四个 Tab 内容
/// 负责管理当前服务器连接信息、退出登录等操作
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  Future<void> _handleDisconnect(BuildContext context, WidgetRef ref) async {
    // 调用 AuthController 清理当前激活的连接态
    final authController = ref.read(authControllerProvider.notifier);
    await authController.logout();
    
    if (context.mounted) {
      // 登出后，通过 goRouter 返回到全局真正的聚合首页（ServerListScreen）
      context.go('/');
    }
  }

  Future<void> _handleDeleteServer(BuildContext context, WidgetRef ref) async {
    final currentUrl = StorageService.activeServerUrl;
    if (currentUrl == null) return;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('移除服务器'),
        content: const Text('确定要从本地列表中移除当前服务器吗？(移除后需重新登录/扫码)'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('取消', style: TextStyle(color: Colors.grey)),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('移除'),
          ),
        ],
      ),
    );

    if (confirmed == true) {
      // 从本地服务器列表中删除
      await StorageService.removeServerConnection(currentUrl);
      // 清除当前激活态并退回列表页
      final authController = ref.read(authControllerProvider.notifier);
      await authController.logout();
      if (context.mounted) {
        context.go('/');
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final currentUrl = StorageService.activeServerUrl ?? '未知地址';

    return Scaffold(
      appBar: AppBar(
        title: const Text('设置与管理'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // 当前服务器卡片
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Row(
                    children: [
                      Icon(Icons.computer_rounded, color: Color(0xFF2563EB)),
                      SizedBox(width: 8),
                      Text(
                        '当前连接的服务器',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                          color: Color(0xFF0F172A),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  Text(
                    currentUrl,
                    style: const TextStyle(
                      fontSize: 15,
                      color: Color(0xFF64748B),
                    ),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),

          // 业务功能区
          const Padding(
            padding: EdgeInsets.only(left: 8, bottom: 8),
            child: Text(
              '网站与证书',
              style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold, color: Colors.black54),
            ),
          ),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.dns_rounded, color: Color(0xFF2563EB)),
                  title: const Text('DNS 账户授权'),
                  subtitle: const Text('用于自动申请与续期 Let\'s Encrypt 证书'),
                  trailing: const Icon(Icons.chevron_right_rounded),
                  onTap: () {
                    Navigator.push(
                      context,
                      MaterialPageRoute(builder: (context) => const DnsAccountListScreen()),
                    );
                  },
                ),
                // 未来可以在这里继续增加: 网站列表、SSL 证书列表等入口
              ],
            ),
          ),
          const SizedBox(height: 24),
          
          // 操作列表区
          const Padding(
            padding: EdgeInsets.only(left: 8, bottom: 8),
            child: Text(
              '连接操作',
              style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold, color: Colors.black54),
            ),
          ),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.logout_rounded, color: Colors.orange),
                  title: const Text('断开连接'),
                  subtitle: const Text('返回服务器选择列表'),
                  trailing: const Icon(Icons.chevron_right_rounded),
                  onTap: () => _handleDisconnect(context, ref),
                ),
                const Divider(height: 1, indent: 56),
                ListTile(
                  leading: const Icon(Icons.delete_outline_rounded, color: Colors.red),
                  title: const Text('移除服务器', style: TextStyle(color: Colors.red)),
                  subtitle: const Text('从本地记录中彻底删除'),
                  onTap: () => _handleDeleteServer(context, ref),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
