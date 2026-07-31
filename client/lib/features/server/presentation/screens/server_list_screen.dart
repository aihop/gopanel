import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/storage/storage_service.dart';
import '../../../../shared/models/server_connection.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';

/// 全局真正的首页：服务器聚合页
/// 遵循 PROMPT.md：展示多个保存的服务器，点击才进入单个服务器的 Dashboard
class ServerListScreen extends ConsumerStatefulWidget {
  const ServerListScreen({super.key});

  @override
  ConsumerState<ServerListScreen> createState() => _ServerListScreenState();
}

class _ServerListScreenState extends ConsumerState<ServerListScreen> {
  List<ServerConnection> _servers = [];

  @override
  void initState() {
    super.initState();
    _loadServers();
  }

  void _loadServers() {
    setState(() {
      _servers = StorageService.getServerList();
    });
  }

  Future<void> _handleConnect(ServerConnection server) async {
    // 设置当前激活服务器
    await StorageService.setActiveServerUrl(server.url);
    await StorageService.setActiveServerToken(server.token);

    // 重新触发一遍 authController 的状态验证
    final authController = ref.read(authControllerProvider.notifier);
    // 这里借用一下其内部静默刷新的逻辑（在 auth_controller.dart 里封装一个公开方法更优，这里用简易实现）
    final success = await authController.connectWithQrToken(
      serverUrl: server.url,
      token: server.token,
    );

    if (mounted) {
      if (success) {
        context.push('/dashboard');
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('服务器凭证已失效，请重新登录'), backgroundColor: Colors.red),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('服务器列表'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '添加服务器',
            onPressed: () => context.push('/login'),
          ),
        ],
      ),
      body: _servers.isEmpty
          ? Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.dns_outlined, size: 80, color: Colors.blue.withValues(alpha: 0.3)),
                  const SizedBox(height: 16),
                  const Text('暂无已连接的服务器', style: TextStyle(color: Colors.grey, fontSize: 16)),
                  const SizedBox(height: 24),
                  ElevatedButton.icon(
                    onPressed: () => context.push('/login'),
                    icon: const Icon(Icons.add),
                    label: const Text('添加服务器'),
                  )
                ],
              ),
            )
          : ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: _servers.length,
              separatorBuilder: (context, index) => const SizedBox(height: 16),
              itemBuilder: (context, index) {
                final server = _servers[index];
                return Card(
                  child: ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                    leading: const CircleAvatar(
                      backgroundColor: Color(0xFFEFF6FF),
                      child: Icon(Icons.dns_rounded, color: Color(0xFF2563EB)),
                    ),
                    title: Text(
                      server.name,
                      style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                    ),
                    subtitle: Padding(
                      padding: const EdgeInsets.only(top: 8.0),
                      child: Text(server.url, style: const TextStyle(color: Colors.grey)),
                    ),
                    trailing: const Icon(Icons.chevron_right_rounded),
                    onTap: () => _handleConnect(server),
                  ),
                );
              },
            ),
    );
  }
}
