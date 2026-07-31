import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'ai_workspace_controller.dart';

final pendingAiApprovalCountProvider = FutureProvider<int>((ref) async {
  final repo = ref.watch(aiWorkspaceRepositoryProvider);
  final approvals = await repo.getApprovals(status: 'pending', limit: 200);
  return approvals.length;
});
