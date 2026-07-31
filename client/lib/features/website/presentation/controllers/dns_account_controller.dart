import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/dns_repository.dart';
import '../../models/dns_account.dart';

final dnsRepositoryProvider = Provider<DnsRepository>((ref) {
  return DnsRepository(ApiClient());
});

class DnsAccountState {
  final bool isLoading;
  final String? errorMessage;
  final List<DnsAccount> accounts;

  const DnsAccountState({
    this.isLoading = false,
    this.errorMessage,
    this.accounts = const [],
  });

  DnsAccountState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<DnsAccount>? accounts,
  }) {
    return DnsAccountState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage, // null 则清除
      accounts: accounts ?? this.accounts,
    );
  }
}

final dnsAccountControllerProvider = NotifierProvider<DnsAccountController, DnsAccountState>(DnsAccountController.new);

class DnsAccountController extends Notifier<DnsAccountState> {
  late DnsRepository _repo;

  @override
  DnsAccountState build() {
    _repo = ref.watch(dnsRepositoryProvider);
    // 初始化时自动加载
    Future.microtask(() => loadAccounts());
    return const DnsAccountState();
  }

  Future<void> loadAccounts() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      final listData = await _repo.getDnsAccounts();
      final accounts = listData.map((e) => DnsAccount.fromJson(e)).toList();
      state = state.copyWith(isLoading: false, accounts: accounts);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: '加载 DNS 账户失败: $e');
    }
  }

  /// 添加 DNS 账户
  /// 根据类型不同，拼接成后端的 Authorization JSON
  Future<bool> addAccount({
    required String name,
    required String type,
    String? accessKey,
    String? secretKey,
    String? apiToken,
    String? rawJson,
  }) async {
    state = state.copyWith(isLoading: true, errorMessage: null);

    try {
      String authStr = '{}';

      if (type == 'aliyun' || type == 'tencentcloud' || type == 'volcengine') {
        authStr = jsonEncode({
          'accessKey': accessKey ?? '',
          'secretKey': secretKey ?? '',
        });
      } else if (type == 'cloudflare') {
        authStr = jsonEncode({
          'token': apiToken ?? '',
        });
      } else if (type == 'huaweicloud' || type == 'other') {
        // 自定义或华为云，直接透传 JSON
        authStr = rawJson ?? '{}';
        // 尝试验证是否为合法 JSON
        jsonDecode(authStr);
      }

      await _repo.addDnsAccount(
        name: name,
        type: type,
        authorizationStr: authStr,
      );
      
      // 添加成功后重新刷新列表
      await loadAccounts();
      return true;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: '添加 DNS 账户失败: $e',
      );
      return false;
    }
  }

  /// 删除 DNS 账户
  Future<bool> deleteAccount(int id) async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      await _repo.deleteDnsAccount(id);
      await loadAccounts();
      return true;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: '删除 DNS 账户失败: $e',
      );
      return false;
    }
  }
}
