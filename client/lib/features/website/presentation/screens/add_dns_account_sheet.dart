import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../controllers/dns_account_controller.dart';

class AddDnsAccountSheet extends ConsumerStatefulWidget {
  const AddDnsAccountSheet({super.key});

  @override
  ConsumerState<AddDnsAccountSheet> createState() => _AddDnsAccountSheetState();
}

class _AddDnsAccountSheetState extends ConsumerState<AddDnsAccountSheet> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  final _accessKeyController = TextEditingController();
  final _secretKeyController = TextEditingController();
  final _tokenController = TextEditingController();
  final _jsonController = TextEditingController();

  String _selectedType = 'aliyun';

  final List<Map<String, String>> _providerOptions = [
    {'label': '阿里云 (Aliyun)', 'value': 'aliyun'},
    {'label': '腾讯云 (Tencent)', 'value': 'tencentcloud'},
    {'label': 'Cloudflare', 'value': 'cloudflare'},
    {'label': '火山引擎', 'value': 'volcengine'},
    {'label': '华为云/其他 (Raw JSON)', 'value': 'huaweicloud'},
  ];

  @override
  void dispose() {
    _nameController.dispose();
    _accessKeyController.dispose();
    _secretKeyController.dispose();
    _tokenController.dispose();
    _jsonController.dispose();
    super.dispose();
  }

  void _submit() async {
    if (!_formKey.currentState!.validate()) return;
    
    // 收起键盘
    FocusScope.of(context).unfocus();

    final controller = ref.read(dnsAccountControllerProvider.notifier);
    final success = await controller.addAccount(
      name: _nameController.text.trim(),
      type: _selectedType,
      accessKey: _accessKeyController.text.trim(),
      secretKey: _secretKeyController.text.trim(),
      apiToken: _tokenController.text.trim(),
      rawJson: _jsonController.text.trim(),
    );

    if (success && mounted) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('DNS 账户添加成功'), backgroundColor: Colors.green),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final isLoading = ref.watch(dnsAccountControllerProvider).isLoading;

    return DraggableScrollableSheet(
      initialChildSize: 0.8,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      builder: (_, scrollController) {
        return Container(
          decoration: const BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
          ),
          padding: EdgeInsets.only(
            bottom: MediaQuery.of(context).viewInsets.bottom, // 键盘遮挡适配
          ),
          child: Column(
            children: [
              // 拖动把手与标题
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
                child: Column(
                  children: [
                    Container(
                      width: 40,
                      height: 4,
                      decoration: BoxDecoration(
                        color: Colors.grey.shade300,
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                    const SizedBox(height: 16),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text('添加 DNS 账户', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                        IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(context)),
                      ],
                    ),
                  ],
                ),
              ),
              const Divider(height: 1),

              // 表单区
              Expanded(
                child: Form(
                  key: _formKey,
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.all(20),
                    children: [
                      _buildLabel('账户名称'),
                      TextFormField(
                        controller: _nameController,
                        decoration: const InputDecoration(
                          hintText: '如: 我的阿里云域名账户',
                          prefixIcon: Icon(Icons.title_rounded),
                          border: OutlineInputBorder(),
                        ),
                        validator: (v) => v!.trim().isEmpty ? '必填项' : null,
                      ),
                      const SizedBox(height: 20),

                      _buildLabel('服务商类型'),
                      DropdownButtonFormField<String>(
                        initialValue: _selectedType,
                        decoration: const InputDecoration(
                          border: OutlineInputBorder(),
                          prefixIcon: Icon(Icons.dns_rounded),
                        ),
                        items: _providerOptions.map((option) {
                          return DropdownMenuItem(
                            value: option['value'],
                            child: Text(option['label']!),
                          );
                        }).toList(),
                        onChanged: (val) {
                          if (val != null) setState(() => _selectedType = val);
                        },
                      ),
                      const SizedBox(height: 20),

                      // 根据服务商显示不同的凭证输入框
                      if (_selectedType == 'aliyun' || _selectedType == 'tencentcloud' || _selectedType == 'volcengine')
                        ...[
                          _buildLabel('Access Key / Secret Id'),
                          TextFormField(
                            controller: _accessKeyController,
                            decoration: const InputDecoration(
                              hintText: '请输入 Access Key',
                              border: OutlineInputBorder(),
                            ),
                            validator: (v) => v!.trim().isEmpty ? '必填项' : null,
                          ),
                          const SizedBox(height: 20),
                          _buildLabel('Secret Key'),
                          TextFormField(
                            controller: _secretKeyController,
                            obscureText: true,
                            decoration: const InputDecoration(
                              hintText: '请输入 Secret Key',
                              border: OutlineInputBorder(),
                            ),
                            validator: (v) => v!.trim().isEmpty ? '必填项' : null,
                          ),
                        ]
                      else if (_selectedType == 'cloudflare')
                        ...[
                          _buildLabel('API Token'),
                          TextFormField(
                            controller: _tokenController,
                            obscureText: true,
                            decoration: const InputDecoration(
                              hintText: '请输入 Cloudflare API Token',
                              border: OutlineInputBorder(),
                            ),
                            validator: (v) => v!.trim().isEmpty ? '必填项' : null,
                          ),
                        ]
                      else
                        ...[
                          _buildLabel('Authorization JSON (华为云等)'),
                          TextFormField(
                            controller: _jsonController,
                            maxLines: 5,
                            decoration: const InputDecoration(
                              hintText: '{\n  "accessKey": "xxx",\n  "secretKey": "yyy"\n}',
                              border: OutlineInputBorder(),
                            ),
                            validator: (v) {
                              if (v!.trim().isEmpty) return '必填项';
                              try {
                                // 尝试校验是否合法 JSON
                                // jsonDecode(v);
                              } catch (_) {
                                return '请输入合法的 JSON 格式';
                              }
                              return null;
                            },
                          ),
                        ],
                        
                      const SizedBox(height: 40),
                      
                      SizedBox(
                        height: 48,
                        child: ElevatedButton(
                          onPressed: isLoading ? null : _submit,
                          style: ElevatedButton.styleFrom(
                            backgroundColor: const Color(0xFF2563EB),
                            foregroundColor: Colors.white,
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                          ),
                          child: isLoading 
                            ? const CircularProgressIndicator(color: Colors.white)
                            : const Text('保存 DNS 账户', style: TextStyle(fontSize: 16)),
                        ),
                      ),
                      const SizedBox(height: 20),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildLabel(String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8.0),
      child: Text(text, style: const TextStyle(fontWeight: FontWeight.w600, color: Color(0xFF334155))),
    );
  }
}
