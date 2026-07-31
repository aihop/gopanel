import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/ssl_repository.dart';
import '../../models/ssl_info.dart';

final sslRepositoryProvider = Provider<SslRepository>((ref) {
  return SslRepository(ApiClient());
});

class SslListState {
  final bool isLoading;
  final String? errorMessage;
  final List<SslInfo> items;

  const SslListState({
    this.isLoading = true,
    this.errorMessage,
    this.items = const [],
  });

  SslListState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<SslInfo>? items,
  }) {
    return SslListState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      items: items ?? this.items,
    );
  }
}

final sslControllerProvider =
    NotifierProvider<SslController, SslListState>(SslController.new);

class SslController extends Notifier<SslListState> {
  late SslRepository _repo;

  @override
  SslListState build() {
    _repo = ref.watch(sslRepositoryProvider);
    _load();
    return const SslListState();
  }

  Future<void> refresh() async {
    await _load();
  }

  Future<void> _load() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      final list = await _repo.list();
      state = state.copyWith(isLoading: false, items: list, errorMessage: null);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: e.toString());
    }
  }
}

