import 'package:flutter_riverpod/flutter_riverpod.dart';

final mainScaffoldIndexProvider =
    NotifierProvider<MainScaffoldIndexController, int>(
  MainScaffoldIndexController.new,
);

class MainScaffoldIndexController extends Notifier<int> {
  @override
  int build() => 0;

  void setIndex(int index) {
    state = index;
  }
}
