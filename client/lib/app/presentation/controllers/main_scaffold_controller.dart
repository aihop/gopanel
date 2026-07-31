import 'package:flutter_riverpod/flutter_riverpod.dart';

final mainScaffoldIndexProvider =
    NotifierProvider<MainScaffoldIndexController, int>(
      MainScaffoldIndexController.new,
    );

class MainScaffoldIndexController extends Notifier<int> {
  static const overviewIndex = 0;
  static const codeIndex = 1;
  static const resourcesIndex = 2;
  static const settingsIndex = 3;

  @override
  int build() => overviewIndex;

  void setIndex(int index) {
    state = index;
  }
}
