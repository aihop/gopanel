import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/panel/panel_card.dart';
import '../../../../shared/widgets/panel/ring_gauge.dart';

class DashboardMetricCard extends StatelessWidget {
  final String title;
  final String valueText;
  final String? subtitle;
  final double percent;
  final Color? color;

  const DashboardMetricCard({
    super.key,
    required this.title,
    required this.valueText,
    required this.percent,
    this.subtitle,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    final p = percent.clamp(0.0, 1.0);
    final c = color ?? _colorForPercent(p);

    return PanelCard(
      title: Text(title),
      padding: const EdgeInsets.all(18),
      child: Row(
        children: [
          SizedBox(
            width: 72,
            height: 72,
            child: Stack(
              alignment: Alignment.center,
              children: [
                RingGauge(value: p, color: c, strokeWidth: 10),
                Text(
                  '${(p * 100).toStringAsFixed(0)}%',
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w800,
                    color: AppTheme.textPrimary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  valueText,
                  style: const TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.w800,
                    color: AppTheme.textPrimary,
                  ),
                ),
                if (subtitle != null) ...[
                  const SizedBox(height: 6),
                  Text(
                    subtitle!,
                    style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: AppTheme.textSecondary,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }

  Color _colorForPercent(double p) {
    if (p > 0.9) return AppTheme.error;
    if (p > 0.8) return AppTheme.warning;
    return AppTheme.primaryBlue;
  }
}

