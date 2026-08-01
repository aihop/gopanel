import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../controllers/dns_account_controller.dart';
import 'add_dns_account_sheet.dart';

/// DNS 账户列表页面
class DnsAccountListScreen extends ConsumerStatefulWidget {
  const DnsAccountListScreen({super.key});

  @override
  ConsumerState<DnsAccountListScreen> createState() => _DnsAccountListScreenState();
}

class _DnsAccountListScreenState extends ConsumerState<DnsAccountListScreen> {
  @override
  Widget build(BuildContext context) {
    final state = ref.watch(dnsAccountControllerProvider);
    final controller = ref.read(dnsAccountControllerProvider.notifier);

    // 显示错误提示
    ref.listen(dnsAccountControllerProvider, (previous, next) {
      if (next.errorMessage != null && next.errorMessage != previous?.errorMessage) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(next.errorMessage!), backgroundColor: Colors.red),
        );
      }
    });

    return Scaffold(
      backgroundColor: const Color(0xFFF1F5F9), // 蓝色系极浅背景
      appBar: AppBar(
        title: const Text('DNS 账户授权'),
        backgroundColor: Colors.white,
        foregroundColor: const Color(0xFF0F172A),
        elevation: 0.5,
        actions: [
          IconButton(
            icon: const Icon(Icons.add_circle_outline_rounded, color: Color(0xFF2563EB)),
            tooltip: '添加 DNS 账户',
            onPressed: () => _showAddAccountSheet(context),
          ),
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            tooltip: '刷新',
            onPressed: () => controller.loadAccounts(),
          ),
        ],
      ),
      body: state.isLoading && state.accounts.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : state.accounts.isEmpty
              ? _buildEmptyState()
              : RefreshIndicator(
                  onRefresh: () => controller.loadAccounts(),
                  child: ListView.separated(
                    padding: const EdgeInsets.all(16),
                    itemCount: state.accounts.length,
                    separatorBuilder: (context, index) => const SizedBox(height: 12),
                    itemBuilder: (context, index) {
                      final account = state.accounts[index];
                      return _buildAccountCard(account, controller);
                    },
                  ),
                ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.dns_rounded, size: 64, color: Colors.blueGrey.shade200),
          const SizedBox(height: 16),
          const Text(
            '暂无 DNS 账户',
            style: TextStyle(fontSize: 16, color: Colors.black54),
          ),
          const SizedBox(height: 8),
          const Text(
            '用于申请和续期 Let\'s Encrypt 泛域名 SSL 证书',
            style: TextStyle(fontSize: 13, color: Colors.black38),
          ),
          const SizedBox(height: 24),
          ElevatedButton.icon(
            onPressed: () => _showAddAccountSheet(context),
            icon: const Icon(Icons.add),
            label: const Text('添加账户'),
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF2563EB),
              foregroundColor: Colors.white,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAccountCard(dynamic account, DnsAccountController controller) {
    // 这里简单映射下提供商图标
    IconData providerIcon = Icons.cloud_queue_rounded;
    Color iconColor = Colors.blueGrey;
    String typeLabel = account.type;

    if (account.type == 'aliyun') {
      typeLabel = '阿里云';
      iconColor = Colors.orange;
    } else if (account.type == 'tencentcloud') {
      typeLabel = '腾讯云';
      iconColor = Colors.blue;
    } else if (account.type == 'cloudflare') {
      typeLabel = 'Cloudflare';
      iconColor = Colors.orangeAccent;
    }

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: Colors.blueGrey.shade100, width: 1),
      ),
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        leading: CircleAvatar(
          backgroundColor: iconColor.withValues(alpha: 0.1),
          child: Icon(providerIcon, color: iconColor),
        ),
        title: Text(
          account.name,
          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16),
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4.0),
          child: Text('服务商: $typeLabel'),
        ),
        trailing: IconButton(
          icon: const Icon(Icons.delete_outline_rounded, color: Colors.redAccent),
          onPressed: () => _confirmDelete(context, account, controller),
        ),
      ),
    );
  }

  void _confirmDelete(BuildContext context, dynamic account, DnsAccountController controller) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('删除确认'),
        content: Text('确定要删除 DNS 账户 "${account.name}" 吗？这可能导致相关证书无法自动续期。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(context);
              controller.deleteAccount(account.id);
            },
            child: const Text('删除', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
  }

  void _showAddAccountSheet(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (context) => const AddDnsAccountSheet(),
    );
  }
}
