import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/website/data/website_repository.dart';
import 'package:gopanel/features/website/models/website_info.dart';
import 'package:gopanel/features/website/presentation/controllers/website_controller.dart';

class _FakeWebsiteRepository extends WebsiteRepository {
  _FakeWebsiteRepository() : super(ApiClient());

  @override
  Future<List<WebsiteInfo>> listWebsites() async {
    return [
      WebsiteInfo(
        id: 1,
        alias: 'mobile',
        primaryDomain: 'mobile.example.test',
        type: 'static',
        status: 'running',
        pipelineId: 0,
        appName: '',
        updatedAt: null,
        expireDate: null,
      ),
    ];
  }
}

void main() {
  test('resource controller finishes initial loading', () async {
    final container = ProviderContainer(
      overrides: [
        websiteRepositoryProvider.overrideWithValue(_FakeWebsiteRepository()),
      ],
    );
    addTearDown(container.dispose);

    expect(container.read(websiteControllerProvider).isLoading, isTrue);
    await Future<void>.delayed(Duration.zero);

    final state = container.read(websiteControllerProvider);
    expect(state.isLoading, isFalse);
    expect(state.errorMessage, isNull);
    expect(state.items.single.alias, 'mobile');
  });
}
