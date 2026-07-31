import 'package:flutter/material.dart';

class PanelCard extends StatelessWidget {
  final Widget? title;
  final Widget? trailing;
  final Widget child;
  final EdgeInsetsGeometry padding;

  const PanelCard({
    super.key,
    this.title,
    this.trailing,
    required this.child,
    this.padding = const EdgeInsets.all(20),
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: padding,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (title != null || trailing != null)
              Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  if (title != null)
                    DefaultTextStyle.merge(
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                      ),
                      child: title!,
                    )
                  else
                    const SizedBox.shrink(),
                  if (trailing != null) trailing! else const SizedBox.shrink(),
                ],
              ),
            if (title != null || trailing != null) const SizedBox(height: 16),
            child,
          ],
        ),
      ),
    );
  }
}

