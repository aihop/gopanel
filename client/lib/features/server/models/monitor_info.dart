class IoNetInfo {
  final int ioReadBytes;
  final int ioWriteBytes;
  final int netBytesSent;
  final int netBytesRecv;
  final DateTime? shotTime;

  IoNetInfo({
    required this.ioReadBytes,
    required this.ioWriteBytes,
    required this.netBytesSent,
    required this.netBytesRecv,
    required this.shotTime,
  });

  factory IoNetInfo.fromJson(Map<String, dynamic> json) {
    return IoNetInfo(
      ioReadBytes: (json['ioReadBytes'] as num?)?.toInt() ?? 0,
      ioWriteBytes: (json['ioWriteBytes'] as num?)?.toInt() ?? 0,
      netBytesSent: (json['netBytesSent'] as num?)?.toInt() ?? 0,
      netBytesRecv: (json['netBytesRecv'] as num?)?.toInt() ?? 0,
      shotTime: DateTime.tryParse((json['shotTime'] ?? '').toString()),
    );
  }
}

class MonitorSeries {
  final double netUpBps;
  final double netDownBps;
  final double ioReadBps;
  final double ioWriteBps;
  final int totalSentBytes;
  final int totalRecvBytes;
  final int totalReadBytes;
  final int totalWriteBytes;
  final List<double> netUpSeries;
  final List<double> netDownSeries;
  final List<double> ioReadSeries;
  final List<double> ioWriteSeries;

  const MonitorSeries({
    required this.netUpBps,
    required this.netDownBps,
    required this.ioReadBps,
    required this.ioWriteBps,
    required this.totalSentBytes,
    required this.totalRecvBytes,
    required this.totalReadBytes,
    required this.totalWriteBytes,
    required this.netUpSeries,
    required this.netDownSeries,
    required this.ioReadSeries,
    required this.ioWriteSeries,
  });

  MonitorSeries copyWith({
    double? netUpBps,
    double? netDownBps,
    double? ioReadBps,
    double? ioWriteBps,
    int? totalSentBytes,
    int? totalRecvBytes,
    int? totalReadBytes,
    int? totalWriteBytes,
    List<double>? netUpSeries,
    List<double>? netDownSeries,
    List<double>? ioReadSeries,
    List<double>? ioWriteSeries,
  }) {
    return MonitorSeries(
      netUpBps: netUpBps ?? this.netUpBps,
      netDownBps: netDownBps ?? this.netDownBps,
      ioReadBps: ioReadBps ?? this.ioReadBps,
      ioWriteBps: ioWriteBps ?? this.ioWriteBps,
      totalSentBytes: totalSentBytes ?? this.totalSentBytes,
      totalRecvBytes: totalRecvBytes ?? this.totalRecvBytes,
      totalReadBytes: totalReadBytes ?? this.totalReadBytes,
      totalWriteBytes: totalWriteBytes ?? this.totalWriteBytes,
      netUpSeries: netUpSeries ?? this.netUpSeries,
      netDownSeries: netDownSeries ?? this.netDownSeries,
      ioReadSeries: ioReadSeries ?? this.ioReadSeries,
      ioWriteSeries: ioWriteSeries ?? this.ioWriteSeries,
    );
  }

  static const empty = MonitorSeries(
    netUpBps: 0,
    netDownBps: 0,
    ioReadBps: 0,
    ioWriteBps: 0,
    totalSentBytes: 0,
    totalRecvBytes: 0,
    totalReadBytes: 0,
    totalWriteBytes: 0,
    netUpSeries: [],
    netDownSeries: [],
    ioReadSeries: [],
    ioWriteSeries: [],
  );
}

