import 'dart:ui';

import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';

class GlassTabItem<T> {
  final T value;
  final String label;
  final IconData? icon;

  const GlassTabItem({required this.value, required this.label, this.icon});
}

class GlassTabs<T> extends StatelessWidget {
  final List<GlassTabItem<T>> items;
  final T selected;
  final ValueChanged<T> onChanged;
  final EdgeInsetsGeometry outerPadding;
  final EdgeInsetsGeometry tabPadding;
  final double borderRadius;
  final double tabRadius;
  final double blurSigma;

  const GlassTabs({
    super.key,
    required this.items,
    required this.selected,
    required this.onChanged,
    this.outerPadding = const EdgeInsets.all(6),
    this.tabPadding = const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
    this.borderRadius = 18,
    this.tabRadius = 14,
    this.blurSigma = 14,
  });

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(borderRadius),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma),
        child: Container(
          padding: outerPadding,
          decoration: BoxDecoration(
            color: Colors.white.withValues(alpha: 0.55),
            borderRadius: BorderRadius.circular(borderRadius),
            border: Border.all(
              color: AppTheme.primaryBlueBorder.withValues(alpha: 0.9),
            ),
            boxShadow: [
              BoxShadow(
                color: AppTheme.textPrimary.withValues(alpha: 0.06),
                blurRadius: 18,
                offset: const Offset(0, 10),
              ),
            ],
          ),
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            physics: const BouncingScrollPhysics(),
            child: Row(
              children: [
                for (int i = 0; i < items.length; i++) ...[
                  _tab(items[i]),
                  if (i != items.length - 1) const SizedBox(width: 6),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _tab(GlassTabItem<T> item) {
    final isSelected = item.value == selected;
    return InkWell(
      onTap: () => onChanged(item.value),
      borderRadius: BorderRadius.circular(tabRadius),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 180),
        curve: Curves.easeOut,
        padding: tabPadding,
        decoration: BoxDecoration(
          color: isSelected ? Colors.white : Colors.transparent,
          borderRadius: BorderRadius.circular(tabRadius),
          border: isSelected
              ? Border.all(color: AppTheme.primaryBlueBorder)
              : Border.all(color: Colors.transparent),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (item.icon != null) ...[
              Icon(
                item.icon,
                size: 16,
                color: isSelected ? AppTheme.primaryBlue : AppTheme.textSecondary,
              ),
              const SizedBox(width: 8),
            ],
            Text(
              item.label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w800,
                color: isSelected ? AppTheme.primaryBlue : AppTheme.textSecondary,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
