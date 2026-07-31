import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';

class PanelProgressRow extends StatelessWidget {
  final String label;
  final double percent;
  final String valueText;

  const PanelProgressRow({
    super.key,
    required this.label,
    required this.percent,
    required this.valueText,
  });

  @override
  Widget build(BuildContext context) {
    final p = percent.clamp(0.0, 1.0);
    Color progressColor = AppTheme.primaryBlue;
    if (p > 0.9) {
      progressColor = AppTheme.error;
    } else if (p > 0.8) {
      progressColor = AppTheme.warning;
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppTheme.textSecondary,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            const SizedBox(width: 12),
            Text(
              valueText,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                fontWeight: FontWeight.w700,
                fontSize: 13,
                color: AppTheme.textPrimary,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        LinearProgressIndicator(
          value: p,
          backgroundColor: AppTheme.primaryBlueLight,
          color: progressColor,
          minHeight: 6,
          borderRadius: BorderRadius.circular(3),
        ),
      ],
    );
  }
}
