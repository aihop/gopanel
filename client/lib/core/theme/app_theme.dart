import 'package:flutter/material.dart';

/// 全局主题配置
/// 遵循 PROMPT.md 视觉约束：
/// - GoPanel 主色调为蓝色系
/// - 简约、清晰、扁平、低噪声
/// - 白底卡片、浅蓝强调、柔和阴影、弱边框
class AppTheme {
  // 定义核心色彩
  static const Color primaryBlue = Color(0xFF2563EB); // 类似 Tailwind blue-600
  static const Color primaryBlueLight = Color(
    0xFFEFF6FF,
  ); // 类似 Tailwind blue-50
  static const Color primaryBlueBorder = Color(
    0xFFDBEAFE,
  ); // 类似 Tailwind blue-100

  static const Color textPrimary = Color(0xFF0F172A); // slate-900
  static const Color textSecondary = Color(0xFF64748B); // slate-500
  static const Color textLight = Color(0xFF94A3B8); // slate-400

  static const Color background = Color(0xFFF8FAFC); // slate-50
  static const Color surface = Colors.white;
  static const Color border = Color(0xFFE2E8F0); // slate-200

  // 状态色
  static const Color success = Color(0xFF10B981); // amber / green 系根据需要
  static const Color warning = Color(0xFFF59E0B);
  static const Color error = Color(0xFFF43F5E); // rose-500

  /// 获取亮色主题
  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      colorScheme: ColorScheme.fromSeed(
        seedColor: primaryBlue,
        primary: primaryBlue,
        surface: background,
        onSurface: textPrimary,
        error: error,
      ),
      scaffoldBackgroundColor: background,

      // AppBar 样式（扁平、克制）
      appBarTheme: const AppBarTheme(
        backgroundColor: surface,
        foregroundColor: textPrimary,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: true,
        iconTheme: IconThemeData(color: textPrimary),
        titleTextStyle: TextStyle(
          color: textPrimary,
          fontSize: 18,
          fontWeight: FontWeight.w600,
        ),
      ),

      // 卡片样式（白底卡片、弱边框、柔和阴影、圆角）
      cardTheme: CardThemeData(
        color: surface,
        elevation: 1, // 柔和阴影
        shadowColor: textPrimary.withValues(alpha: 0.04),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(20), // 类似 StatusCard 的大圆角
          side: const BorderSide(color: border, width: 1),
        ),
        margin: EdgeInsets.zero,
      ),

      // 按钮样式
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: primaryBlue,
          foregroundColor: Colors.white,
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: primaryBlue,
          textStyle: const TextStyle(fontWeight: FontWeight.w600),
        ),
      ),

      // 输入框样式
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: surface,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 16,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: primaryBlue, width: 2),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: error),
        ),
        hintStyle: const TextStyle(color: textLight),
      ),
    );
  }
}
