import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';

class RingGauge extends StatelessWidget {
  final double value;
  final Color color;
  final double strokeWidth;

  const RingGauge({
    super.key,
    required this.value,
    this.color = AppTheme.primaryBlue,
    this.strokeWidth = 10,
  });

  @override
  Widget build(BuildContext context) {
    final v = value.clamp(0.0, 1.0);
    return CustomPaint(
      painter: _RingGaugePainter(
        value: v,
        color: color,
        strokeWidth: strokeWidth,
      ),
      child: const SizedBox.expand(),
    );
  }
}

class _RingGaugePainter extends CustomPainter {
  final double value;
  final Color color;
  final double strokeWidth;

  _RingGaugePainter({
    required this.value,
    required this.color,
    required this.strokeWidth,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final shortest = math.min(size.width, size.height);
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: shortest,
      height: shortest,
    ).deflate(strokeWidth / 2);

    final bgPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round
      ..color = AppTheme.border;

    final fgPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round
      ..color = color;

    final startAngle = -math.pi / 2;
    canvas.drawArc(rect, startAngle, math.pi * 2, false, bgPaint);
    canvas.drawArc(rect, startAngle, math.pi * 2 * value, false, fgPaint);
  }

  @override
  bool shouldRepaint(covariant _RingGaugePainter oldDelegate) {
    return oldDelegate.value != value ||
        oldDelegate.color != color ||
        oldDelegate.strokeWidth != strokeWidth;
  }
}
