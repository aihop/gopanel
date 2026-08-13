import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/dashboard_repository.dart';
import '../../models/security_risk.dart';
import 'dashboard_controller.dart';

class SecurityRiskState {
  final bool isLoading;
  final String? errorMessage;
  final List<SecurityRisk> risks;

  const SecurityRiskState({
    this.isLoading = true,
    this.errorMessage,
    this.risks = const [],
  });
}

final securityRiskControllerProvider =
    NotifierProvider<SecurityRiskController, SecurityRiskState>(
      SecurityRiskController.new,
    );

class SecurityRiskController extends Notifier<SecurityRiskState> {
  late DashboardRepository _repository;

  @override
  SecurityRiskState build() {
    _repository = ref.watch(dashboardRepositoryProvider);
    Future.microtask(refresh);
    return const SecurityRiskState();
  }

  Future<void> refresh() async {
    state = SecurityRiskState(isLoading: true, risks: state.risks);
    try {
      final risks = await _repository.getSecurityRisks();
      state = SecurityRiskState(risks: risks);
    } catch (error) {
      state = SecurityRiskState(
        errorMessage: '安全风险加载失败：$error',
        risks: state.risks,
      );
    }
  }
}
