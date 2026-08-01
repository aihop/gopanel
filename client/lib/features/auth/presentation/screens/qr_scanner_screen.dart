import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../controllers/auth_controller.dart';

/// 扫码授权页面
class QRScannerScreen extends ConsumerStatefulWidget {
  const QRScannerScreen({super.key});

  @override
  ConsumerState<QRScannerScreen> createState() => _QRScannerScreenState();
}

class _QRScannerScreenState extends ConsumerState<QRScannerScreen> {
  final MobileScannerController _scannerController = MobileScannerController(
    detectionSpeed: DetectionSpeed.noDuplicates,
  );
  bool _isProcessing = false;

  @override
  void dispose() {
    _scannerController.dispose();
    super.dispose();
  }

  Future<void> _handleBarcode(BarcodeCapture capture) async {
    if (_isProcessing) return;

    final List<Barcode> barcodes = capture.barcodes;
    if (barcodes.isEmpty) return;

    final String? codeValue = barcodes.first.rawValue;
    if (codeValue == null || codeValue.isEmpty) return;

    setState(() {
      _isProcessing = true;
    });

    try {
      // 假设 GoPanel 面板生成的二维码是一个 JSON 字符串，例如：
      // {"url": "https://demo.gopanel.run", "token": "xxx..."}
      // 或者特定协议 gpanel://connect?url=xxx&token=xxx
      final Map<String, dynamic> data = jsonDecode(codeValue);
      final url = data['url'] as String?;
      final token = data['token'] as String?;

      if (url == null || token == null || !url.startsWith('http')) {
        _showErrorAndResume('二维码格式不正确，缺少面板地址或凭证');
        return;
      }

      // 暂停扫描器，等待网络验证结果
      _scannerController.stop();

      final authController = ref.read(authControllerProvider.notifier);
      final success = await authController.connectWithQrToken(
        serverUrl: url,
        token: token,
      );

      if (mounted) {
        if (success) {
          // 成功连接，进入选中服务器的概览页
          context.go('/dashboard');
        } else {
          final errorMsg =
              ref.read(authControllerProvider).errorMessage ?? '连接验证失败';
          _showErrorAndResume(errorMsg);
        }
      }
    } catch (e) {
      // JSON 解析失败或其他错误
      _showErrorAndResume('无法识别的二维码内容');
    }
  }

  void _showErrorAndResume(String errorMsg) {
    if (!mounted) return;

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(errorMsg), backgroundColor: Colors.red),
    );

    // 延迟恢复扫描，避免重复报错
    Future.delayed(const Duration(seconds: 2), () {
      if (mounted) {
        setState(() {
          _isProcessing = false;
        });
        _scannerController.start();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final isLoading = ref.watch(authControllerProvider).isLoading;

    return Scaffold(
      appBar: AppBar(
        title: const Text('扫描控制面板二维码'),
        actions: [
          IconButton(
            icon: const Icon(Icons.flash_on),
            onPressed: () => _scannerController.toggleTorch(),
          ),
          IconButton(
            icon: const Icon(Icons.flip_camera_ios),
            onPressed: () => _scannerController.switchCamera(),
          ),
        ],
      ),
      body: Stack(
        children: [
          MobileScanner(
            controller: _scannerController,
            onDetect: _handleBarcode,
          ),

          // 扫描框装饰
          Center(
            child: Container(
              width: 250,
              height: 250,
              decoration: BoxDecoration(
                border: Border.all(color: Colors.blue, width: 2),
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),

          // 底部提示文字
          const Positioned(
            bottom: 40,
            left: 0,
            right: 0,
            child: Text(
              '请将面板端生成的二维码对准框内',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.w500,
                shadows: [Shadow(color: Colors.black54, blurRadius: 4)],
              ),
            ),
          ),

          // 网络验证中的 Loading 遮罩
          if (isLoading || _isProcessing)
            Container(
              color: Colors.black.withValues(alpha: 0.6),
              child: const Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(color: Colors.white),
                    SizedBox(height: 16),
                    Text(
                      '正在验证并连接服务器...',
                      style: TextStyle(color: Colors.white, fontSize: 16),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
