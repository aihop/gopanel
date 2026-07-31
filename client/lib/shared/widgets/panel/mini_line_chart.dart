import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';

class MiniLineChart extends StatelessWidget {
  final List<double> seriesA;
  final List<double> seriesB;
  final Color colorA;
  final Color colorB;
  final double height;

  const MiniLineChart({
    super.key,
    required this.seriesA,
    required this.seriesB,
    this.colorA = AppTheme.primaryBlue,
    this.colorB = const Color(0xFF6366F1),
    this.height = 140,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: height,
      width: double.infinity,
      child: CustomPaint(
        painter: _MiniLineChartPainter(
          seriesA: seriesA,
          seriesB: seriesB,
          colorA: colorA,
          colorB: colorB,
        ),
      ),
    );
  }
}

class _MiniLineChartPainter extends CustomPainter {
  final List<double> seriesA;
  final List<double> seriesB;
  final Color colorA;
  final Color colorB;

  _MiniLineChartPainter({
    required this.seriesA,
    required this.seriesB,
    required this.colorA,
    required this.colorB,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final padding = const EdgeInsets.fromLTRB(6, 10, 6, 12);
    final rect = padding.deflateRect(Offset.zero & size);
    if (rect.width <= 0 || rect.height <= 0) return;

    final maxV = _maxValue(seriesA, seriesB);
    if (maxV <= 0) {
      _drawGrid(canvas, rect);
      return;
    }

    _drawGrid(canvas, rect);
    _drawSeries(canvas, rect, seriesA, colorA.withValues(alpha: 0.9));
    _drawSeries(canvas, rect, seriesB, colorB.withValues(alpha: 0.9));
  }

  void _drawGrid(Canvas canvas, Rect rect) {
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1
      ..color = AppTheme.primaryBlueBorder.withValues(alpha: 0.55);

    const lines = 5;
    for (int i = 1; i <= lines; i++) {
      final y = rect.top + rect.height * (i / (lines + 1));
      canvas.drawLine(Offset(rect.left, y), Offset(rect.right, y), paint);
    }
  }

  void _drawSeries(Canvas canvas, Rect rect, List<double> series, Color color) {
    if (series.length < 2) return;
    final maxV = series.reduce(math.max);
    if (maxV <= 0) return;

    final path = Path();
    for (int i = 0; i < series.length; i++) {
      final x = rect.left + rect.width * (i / (series.length - 1));
      final v = (series[i] / maxV).clamp(0.0, 1.0);
      final y = rect.bottom - rect.height * v;
      if (i == 0) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
    }

    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round
      ..color = color;

    canvas.drawPath(path, paint);
  }

  double _maxValue(List<double> a, List<double> b) {
    double m = 0;
    for (final v in a) {
      if (v > m) m = v;
    }
    for (final v in b) {
      if (v > m) m = v;
    }
    return m;
  }

  @override
  bool shouldRepaint(covariant _MiniLineChartPainter oldDelegate) {
    return oldDelegate.seriesA != seriesA ||
        oldDelegate.seriesB != seriesB ||
        oldDelegate.colorA != colorA ||
        oldDelegate.colorB != colorB;
  }
}
