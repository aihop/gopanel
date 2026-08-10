import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/core/network/api_client.dart';
import 'package:gopanel/features/ai_workspace/data/ai_workspace_repository.dart';
import 'package:gopanel/features/ai_workspace/models/code_workspace_file.dart';
import 'package:gopanel/features/ai_workspace/presentation/controllers/code_workspace_files_controller.dart';

class _FakeAiWorkspaceRepository extends AiWorkspaceRepository {
  _FakeAiWorkspaceRepository({this.saveError}) : super(ApiClient());

  final Object? saveError;
  CodeSessionFile? savedFile;

  @override
  Future<CodeSessionFile> getSessionFile(int sessionId, String path) async {
    return const CodeSessionFile(
      path: 'lib/main.dart',
      content: 'void main() {}',
      extension: 'dart',
      size: 14,
      version: 'v1',
    );
  }

  @override
  Future<CodeSessionFile> saveSessionFile({
    required int sessionId,
    required CodeSessionFile file,
    required String content,
  }) async {
    savedFile = file;
    if (saveError != null) throw saveError!;
    return CodeSessionFile(
      path: file.path,
      content: content,
      extension: file.extension,
      size: content.length,
      version: 'v2',
    );
  }
}

const _entry = CodeStructureEntry(
  name: 'main.dart',
  path: 'lib/main.dart',
  isDirectory: false,
  extension: 'dart',
);

void main() {
  test('saving updates file version and clears dirty state', () async {
    final repository = _FakeAiWorkspaceRepository();
    final controller = CodeWorkspaceFilesController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.openEntry(_entry);
    controller.updateDraft('void main() => print("mobile");');
    expect(controller.state.isDirty, isTrue);

    expect(await controller.saveOpenFile(), isTrue);
    expect(repository.savedFile?.version, 'v1');
    expect(controller.state.openFile?.version, 'v2');
    expect(controller.state.isDirty, isFalse);
    expect(controller.state.errorMessage, isNull);
  });

  test('save failure preserves the mobile draft', () async {
    final repository = _FakeAiWorkspaceRepository(
      saveError: StateError('文件已被其他操作修改'),
    );
    final controller = CodeWorkspaceFilesController(
      repository: repository,
      sessionId: 12,
    );
    addTearDown(controller.dispose);

    await controller.openEntry(_entry);
    controller.updateDraft('unsaved mobile change');

    expect(await controller.saveOpenFile(), isFalse);
    expect(controller.state.draftContent, 'unsaved mobile change');
    expect(controller.state.isDirty, isTrue);
    expect(controller.state.errorMessage, contains('文件已被其他操作修改'));
  });
}
