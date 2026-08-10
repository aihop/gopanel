import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/ai_workspace/data/ai_workspace_repository.dart';

void main() {
  test('builds the supported mobile Code instruction contract', () {
    final request = buildCodeInstructionRequest(
      command: '启动预览并检查登录页',
      autoPreview: false,
      requireApproval: true,
    );

    expect(request, {
      'content': '启动预览并检查登录页',
      'allowCode': true,
      'autoPreview': false,
      'requireApproval': true,
      'analysisOnly': false,
    });
  });
}
