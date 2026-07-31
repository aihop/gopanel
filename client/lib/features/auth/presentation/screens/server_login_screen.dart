import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../controllers/auth_controller.dart';

/// 登录与服务器连接界面
/// 遵循 PROMPT.md 规范：使用简约白底卡片、蓝色品牌色进行设计
class ServerLoginScreen extends ConsumerStatefulWidget {
  const ServerLoginScreen({super.key});

  @override
  ConsumerState<ServerLoginScreen> createState() => _ServerLoginScreenState();
}

class _ServerLoginScreenState extends ConsumerState<ServerLoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _urlController = TextEditingController();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void dispose() {
    _urlController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _handleConnect() async {
    if (_formKey.currentState?.validate() ?? false) {
      // 收起键盘
      FocusScope.of(context).unfocus();

      final authController = ref.read(authControllerProvider.notifier);

      final success = await authController.connectAndLogin(
        serverUrl: _urlController.text.trim(),
        username: _usernameController.text.trim(),
        password: _passwordController.text.trim(),
      );

      if (mounted) {
        if (success) {
          // 登录成功，跳转单个服务器 Dashboard
          context.go('/dashboard');
        } else {
          // 登录失败，显示错误（读取最新 state 里的 errorMessage）
          final errorMsg =
              ref.read(authControllerProvider).errorMessage ?? '连接失败';
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(errorMsg), backgroundColor: Colors.red),
          );
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    // 监听状态，提取 isLoading 用于控制按钮交互
    final authState = ref.watch(authControllerProvider);
    final isLoading = authState.isLoading;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                // LOGO 或标题区
                const Icon(
                  Icons.terminal_rounded,
                  size: 64,
                  color: Color(0xFF2563EB),
                ),
                const SizedBox(height: 16),
                const Text(
                  'GoPanel',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.bold,
                    letterSpacing: 1.2,
                  ),
                ),
                const SizedBox(height: 8),
                const Text(
                  '连接远程服务器管理面板',
                  textAlign: TextAlign.center,
                  style: TextStyle(fontSize: 15, color: Color(0xFF64748B)),
                ),
                const SizedBox(height: 48),

                // 登录卡片
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(24.0),
                    child: Form(
                      key: _formKey,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          const Text(
                            '面板地址',
                            style: TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                              color: Color(0xFF0F172A),
                            ),
                          ),
                          const SizedBox(height: 8),
                          TextFormField(
                            controller: _urlController,
                            decoration: const InputDecoration(
                              hintText: 'https://demo.gopanel.run',
                              prefixIcon: Icon(Icons.link_rounded),
                            ),
                            validator: (value) {
                              if (value == null || value.isEmpty) {
                                return '请输入面板地址';
                              }
                              if (!value.startsWith('http')) {
                                return '请输入带有 http/https 的完整地址';
                              }
                              return null;
                            },
                          ),
                          const SizedBox(height: 20),
                          const Text(
                            '用户名',
                            style: TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                              color: Color(0xFF0F172A),
                            ),
                          ),
                          const SizedBox(height: 8),
                          TextFormField(
                            controller: _usernameController,
                            decoration: const InputDecoration(
                              hintText: '请输入用户名',
                              prefixIcon: Icon(Icons.person_rounded),
                            ),
                            validator: (value) {
                              if (value == null || value.isEmpty) {
                                return '请输入用户名';
                              }
                              return null;
                            },
                          ),
                          const SizedBox(height: 20),
                          const Text(
                            '密码',
                            style: TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                              color: Color(0xFF0F172A),
                            ),
                          ),
                          const SizedBox(height: 8),
                          TextFormField(
                            controller: _passwordController,
                            obscureText: true,
                            decoration: const InputDecoration(
                              hintText: '请输入密码',
                              prefixIcon: Icon(Icons.key_rounded),
                            ),
                            validator: (value) {
                              if (value == null || value.isEmpty) {
                                return '请输入密码';
                              }
                              return null;
                            },
                          ),
                          const SizedBox(height: 32),
                          ElevatedButton(
                            onPressed: isLoading ? null : _handleConnect,
                            child: isLoading
                                ? const SizedBox(
                                    width: 20,
                                    height: 20,
                                    child: CircularProgressIndicator(
                                      color: Colors.white,
                                      strokeWidth: 2,
                                    ),
                                  )
                                : const Text('连 接'),
                          ),
                          const SizedBox(height: 16),
                          OutlinedButton.icon(
                            onPressed: isLoading
                                ? null
                                : () => context.push('/qr_scanner'),
                            icon: const Icon(Icons.qr_code_scanner_rounded),
                            label: const Text('扫描二维码授权'),
                            style: OutlinedButton.styleFrom(
                              padding: const EdgeInsets.symmetric(vertical: 14),
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
