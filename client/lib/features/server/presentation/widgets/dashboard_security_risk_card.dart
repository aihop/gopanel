import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../models/security_risk.dart';
import '../controllers/security_risk_controller.dart';

class DashboardSecurityRiskCard extends ConsumerWidget {
  const DashboardSecurityRiskCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(securityRiskControllerProvider);
    return PanelCard(
      title: Row(
        children: [
          const Icon(Icons.security_rounded, color: AppTheme.error, size: 21),
          const SizedBox(width: 8),
          const Text('安全风险'),
          if (state.risks.isNotEmpty) ...[
            const SizedBox(width: 8),
            _riskBadge(state.risks.length),
          ],
        ],
      ),
      trailing: IconButton(
        tooltip: '刷新安全风险',
        onPressed: state.isLoading
            ? null
            : ref.read(securityRiskControllerProvider.notifier).refresh,
        icon: const Icon(Icons.refresh_rounded),
      ),
      child: _buildBody(ref, state),
    );
  }

  Widget _buildBody(WidgetRef ref, SecurityRiskState state) {
    if (state.isLoading && state.risks.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (state.errorMessage != null && state.risks.isEmpty) {
      return Row(
        children: [
          const Icon(Icons.cloud_off_outlined, color: AppTheme.textLight),
          const SizedBox(width: 10),
          const Expanded(child: Text('安全风险加载失败')),
          TextButton(
            onPressed: ref.read(securityRiskControllerProvider.notifier).refresh,
            child: const Text('重试'),
          ),
        ],
      );
    }
    if (state.risks.isEmpty) {
      return const Row(
        children: [
          Icon(Icons.verified_user_outlined, color: AppTheme.success),
          SizedBox(width: 10),
          Expanded(child: Text('当前未发现活动安全风险')),
        ],
      );
    }

    return Column(
      children: [
        for (var index = 0; index < state.risks.take(3).length; index++) ...[
          _riskItem(state.risks[index]),
          if (index < state.risks.take(3).length - 1)
            const Divider(height: 24),
        ],
      ],
    );
  }

  Widget _riskItem(SecurityRisk risk) {
    final color = _levelColor(risk.level);
    final evidenceCount = risk.evidence.fold<int>(0, (sum, item) => sum + item.count);
    final conclusion = risk.aiConclusion.isNotEmpty ? risk.aiConclusion : risk.summary;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                _levelLabel(risk.level),
                style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w800),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                risk.sourceName.isEmpty ? _sourceLabel(risk.sourceType) : risk.sourceName,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontWeight: FontWeight.w800),
              ),
            ),
          ],
        ),
        const SizedBox(height: 7),
        Text(
          conclusion,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(color: AppTheme.textSecondary, height: 1.4),
        ),
        const SizedBox(height: 9),
        Wrap(
          spacing: 8,
          runSpacing: 6,
          children: [
            _metaChip('${_sourceLabel(risk.sourceType)} · ${_eventLabel(risk.eventType)}'),
            if (evidenceCount > 0) _metaChip('$evidenceCount 条证据'),
            if (risk.aiConclusion.isNotEmpty) _metaChip('AI 置信度 ${risk.confidence}%'),
            if (risk.requiresApproval) _metaChip('处置需审批', warning: true),
          ],
        ),
      ],
    );
  }

  Widget _metaChip(String text, {bool warning = false}) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: warning ? const Color(0xFFFFF7ED) : const Color(0xFFF1F5F9),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: warning ? const Color(0xFFC2410C) : AppTheme.textSecondary,
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Widget _riskBadge(int count) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF1F2),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text('$count', style: const TextStyle(color: AppTheme.error, fontSize: 12, fontWeight: FontWeight.w800)),
    );
  }

  Color _levelColor(String level) {
    if (level == 'critical' || level == 'high') return AppTheme.error;
    if (level == 'medium') return const Color(0xFFD97706);
    return AppTheme.primaryBlue;
  }

  String _levelLabel(String level) => const {
    'critical': '严重', 'high': '高风险', 'medium': '中风险', 'low': '低风险',
  }[level] ?? '提示';

  String _sourceLabel(String source) => const {
    'website': '网站', 'ssh': 'SSH', 'panel': '面板', 'system': '系统',
  }[source] ?? source;

  String _eventLabel(String type) => const {
    'sqli': 'SQL 注入', 'xss': 'XSS', 'path_traversal': '路径穿越',
    'sensitive_path': '敏感路径扫描', 'request_flood': '高频请求',
    'not_found_scan': '404 扫描', 'server_error_spike': '5xx 异常',
    'ssh_brute_force': '暴力破解', 'ssh_failure_then_success': '失败后成功',
    'ssh_root_login': 'root 登录', 'ssh_new_source': '新来源登录',
    'panel_brute_force': '面板暴力破解',
  }[type] ?? type;
}
