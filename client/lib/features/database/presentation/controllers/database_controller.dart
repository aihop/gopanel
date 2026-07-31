import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/database_repository.dart';
import '../../models/database_info.dart';

final databaseRepositoryProvider = Provider<DatabaseRepository>((ref) {
  return DatabaseRepository(ApiClient());
});

class DatabaseListState {
  final bool isLoading;
  final String? errorMessage;
  final List<DatabaseInfo> items;

  const DatabaseListState({
    this.isLoading = true,
    this.errorMessage,
    this.items = const [],
  });

  DatabaseListState copyWith({
    bool? isLoading,
    String? errorMessage,
    List<DatabaseInfo>? items,
  }) {
    return DatabaseListState(
      isLoading: isLoading ?? this.isLoading,
      errorMessage: errorMessage,
      items: items ?? this.items,
    );
  }
}

final databaseControllerProvider =
    NotifierProvider<DatabaseController, DatabaseListState>(
  DatabaseController.new,
);

class DatabaseController extends Notifier<DatabaseListState> {
  late DatabaseRepository _repo;

  @override
  DatabaseListState build() {
    _repo = ref.watch(databaseRepositoryProvider);
    Future.microtask(_load);
    return const DatabaseListState();
  }

  Future<void> refresh() async {
    await _load();
  }

  Future<void> _load() async {
    state = state.copyWith(isLoading: true, errorMessage: null);
    try {
      final list = await _repo.listDatabases();
      state = state.copyWith(isLoading: false, items: list, errorMessage: null);
    } catch (e) {
      state = state.copyWith(isLoading: false, errorMessage: e.toString());
    }
  }
}
