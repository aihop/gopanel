class CodeStructureEntry {
  final String name;
  final String path;
  final bool isDirectory;
  final String extension;

  const CodeStructureEntry({
    required this.name,
    required this.path,
    required this.isDirectory,
    required this.extension,
  });

  factory CodeStructureEntry.fromJson(Map<String, dynamic> json) {
    return CodeStructureEntry(
      name: (json['name'] ?? '').toString(),
      path: (json['path'] ?? '').toString(),
      isDirectory: json['isDir'] == true,
      extension: (json['extension'] ?? '').toString(),
    );
  }
}

class CodeStructureResult {
  final String path;
  final List<CodeStructureEntry> entries;
  final bool truncated;

  const CodeStructureResult({
    required this.path,
    required this.entries,
    required this.truncated,
  });

  factory CodeStructureResult.fromJson(Map<String, dynamic> json) {
    return CodeStructureResult(
      path: (json['path'] ?? '').toString(),
      entries: (json['entries'] as List<dynamic>? ?? const [])
          .whereType<Map>()
          .map(
            (item) => CodeStructureEntry.fromJson(item.cast<String, dynamic>()),
          )
          .toList(),
      truncated: json['truncated'] == true,
    );
  }
}

class CodeSessionFile {
  final String path;
  final String content;
  final String extension;
  final int size;
  final String version;

  const CodeSessionFile({
    required this.path,
    required this.content,
    required this.extension,
    required this.size,
    required this.version,
  });

  factory CodeSessionFile.fromJson(Map<String, dynamic> json) {
    return CodeSessionFile(
      path: (json['path'] ?? '').toString(),
      content: (json['content'] ?? '').toString(),
      extension: (json['extension'] ?? '').toString(),
      size: (json['size'] as num?)?.toInt() ?? 0,
      version: (json['version'] ?? '').toString(),
    );
  }
}
