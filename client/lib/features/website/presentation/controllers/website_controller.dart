import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/website_repository.dart';
import '../../models/website_info.dart';

final websiteRepositoryProvider = Provider<WebsiteRepository>((ref) {
  return WebsiteRepository(ApiClient());
});

class WebsiteListState {
  final bool isLoading;
  final String? errorMessage;
  final List<WebsiteInfo> items;

  const WebsiteListState({
    this.isLoading = true,
    this.errorMessage,
    this.items = const [],
  });

  WebsiteListState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<WebsiteInfo>? items,
  }) {
    return WebsiteListState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      items: items ?? this.items,
    );
  }
}

final websiteControllerProvider =
    NotifierProvider<WebsiteController, WebsiteListState>(WebsiteController.new);

class WebsiteController extends Notifier<WebsiteListState> {
  late WebsiteRepository _repo;

  @override
  WebsiteListState build() {
    _repo = ref.watch(websiteRepositoryProvider);
    _load();
    return const WebsiteListState();
  }

  Future<void> refresh() async {
    await _load();
  }

  Future<void> _load() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      final list = await _repo.listWebsites();
      state = state.copyWith(isLoading: false, items: list, errorMessage: null);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: e.toString());
    }
  }
}

