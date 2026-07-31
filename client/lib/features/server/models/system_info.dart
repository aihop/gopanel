class OsInfo {
  final String version;
  final String kernelVersion;
  final String platform;
  final String hostname;

  OsInfo({
    required this.version,
    required this.kernelVersion,
    required this.platform,
    required this.hostname,
  });

  factory OsInfo.fromJson(Map<String, dynamic> json) {
    final platform = (json['kernelArch'] ?? json['platform'])?.toString() ?? '';
    return OsInfo(
      version: (json['version'] ?? json['platformVersion'])?.toString() ?? '',
      kernelVersion: (json['kernelVersion'])?.toString() ?? '',
      platform: platform,
      hostname: (json['hostname'])?.toString() ?? 'Unknown Host',
    );
  }
}

class SystemCurrentInfo {
  final int uptime; // 运行时间（秒）
  final int procs;
  final double percent; // CPU 总使用率 (0-100)
  final double load1; // 1分钟负载
  final double load5;
  final double load15;
  final double loadUsagePercent;

  final int memoryTotal;
  final int memoryAvailable;
  final int memoryUsed;

  final List<DiskInfo> diskData;

  SystemCurrentInfo({
    required this.uptime,
    required this.procs,
    required this.percent,
    required this.load1,
    required this.load5,
    required this.load15,
    required this.loadUsagePercent,
    required this.memoryTotal,
    required this.memoryAvailable,
    required this.memoryUsed,
    required this.diskData,
  });

  factory SystemCurrentInfo.fromJson(Map<String, dynamic> json) {
    final diskList = json['diskData'] as List<dynamic>? ?? [];

    double percent = 0.0;
    if (json['cpuUsedPercent'] != null) {
      percent = (json['cpuUsedPercent'] as num?)?.toDouble() ?? 0.0;
    } else {
      final cpuData = json['cpuData'] as List<dynamic>? ?? [];
      if (cpuData.isNotEmpty) {
        final firstCpu = cpuData[0] as Map<String, dynamic>;
        percent = (firstCpu['percent'] as num?)?.toDouble() ?? 0.0;
      }
    }

    final memoryTotal =
        (json['memoryTotal'] as num?)?.toInt() ??
        ((json['memData'] as Map<String, dynamic>?)?['total'] as num?)
            ?.toInt() ??
        0;
    final memoryAvailable =
        (json['memoryAvailable'] as num?)?.toInt() ??
        ((json['memData'] as Map<String, dynamic>?)?['available'] as num?)
            ?.toInt() ??
        0;
    final memoryUsed =
        (json['memoryUsed'] as num?)?.toInt() ??
        ((json['memData'] as Map<String, dynamic>?)?['used'] as num?)
            ?.toInt() ??
        0;

    return SystemCurrentInfo(
      uptime: (json['uptime'] as num?)?.toInt() ?? 0,
      procs: (json['procs'] as num?)?.toInt() ?? 0,
      percent: percent,
      load1: (json['load1'] as num?)?.toDouble() ?? 0.0,
      load5: (json['load5'] as num?)?.toDouble() ?? 0.0,
      load15: (json['load15'] as num?)?.toDouble() ?? 0.0,
      loadUsagePercent:
          (json['loadUsagePercent'] as num?)?.toDouble() ??
          ((json['loadUsage'] as num?)?.toDouble() ?? 0.0),
      memoryTotal: memoryTotal,
      memoryAvailable: memoryAvailable,
      memoryUsed: memoryUsed,
      diskData: diskList.map((e) => DiskInfo.fromJson(e)).toList(),
    );
  }
}

class DiskInfo {
  final String path;
  final String type;
  final int total;
  final int free;
  final int used;
  final double usedPercent;

  DiskInfo({
    required this.path,
    required this.type,
    required this.total,
    required this.free,
    required this.used,
    required this.usedPercent,
  });

  factory DiskInfo.fromJson(Map<String, dynamic> json) {
    return DiskInfo(
      path: json['path'] ?? '/',
      type: json['type'] ?? '',
      total: json['total'] as int? ?? 0,
      free: json['free'] as int? ?? 0,
      used: json['used'] as int? ?? 0,
      usedPercent: (json['usedPercent'] as num?)?.toDouble() ?? 0.0,
    );
  }
}
