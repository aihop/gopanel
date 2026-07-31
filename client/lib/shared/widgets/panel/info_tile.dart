import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';

class InfoTile extends StatelessWidget {
  final String label;
  final String value;
  final Widget? leading;

  const InfoTile({
    super.key,
    required this.label,
    required this.value,
    this.leading,
  });

  @override
  Widget build(BuildContext context) {
    final border = AppTheme.primaryBlueBorder;
    final bg = AppTheme.primaryBlueLight;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (leading != null) ...[
            IconTheme(
              data: const IconThemeData(size: 18, color: AppTheme.primaryBlue),
              child: leading!,
            ),
            const SizedBox(width: 10),
          ],
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: AppTheme.textSecondary,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                value,
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                  color: AppTheme.textPrimary,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

