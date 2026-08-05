import 'package:flutter_test/flutter_test.dart';
import 'package:gopanel/features/apps/data/apps_repository.dart';
import 'package:gopanel/features/container/data/container_repository.dart';
import 'package:gopanel/features/database/data/database_repository.dart';
import 'package:gopanel/features/website/data/website_repository.dart';
import 'package:gopanel/features/task_center/data/task_repository.dart';

void main() {
  test('resource repositories use registered list routes', () {
    expect(websiteListPath, '/api/website/list');
    expect(databaseListPath, '/api/database/list');
    expect(containerListPath, '/api/container/list');
    expect(installedAppsListPath, '/api/apps/installed/list');
    expect(pipelineListPath, '/api/pipeline/list');
  });

  test('website and database lists parse paginated responses', () {
    final websites = parseWebsiteList({
      'items': [
        {'id': 7, 'alias': '官网', 'primaryDomain': 'gopanel.cn'},
      ],
      'total': 1,
    });
    final databases = parseDatabaseList({
      'items': [
        {'type': 'mysql', 'name': 'panel', 'server': 'local'},
      ],
      'total': 1,
      'warnings': [],
    });

    expect(websites.single.primaryDomain, 'gopanel.cn');
    expect(databases.single.name, 'panel');
  });

  test('container list maps current backend field names', () {
    final containers = parseContainerList({
      'items': [
        {
          'containerID': 'abc123',
          'name': 'nginx',
          'imageName': 'nginx:latest',
          'state': 'running',
          'runTime': 'Up 2 hours',
          'createTime': '2026-08-01',
        },
      ],
    });

    expect(containers.single.image, 'nginx:latest');
    expect(containers.single.status, 'Up 2 hours');
    expect(containers.single.created, '2026-08-01');
  });

  test('installed apps map fields and filter status locally', () {
    final apps = parseInstalledApps({
      'items': [
        {
          'id': 3,
          'name': 'mysql',
          'version': '8.4',
          'status': 'Running',
          'appName': 'MySQL',
        },
        {'id': 4, 'name': 'redis', 'status': 'Stopped'},
      ],
    }, status: 'running');

    expect(apps, hasLength(1));
    expect(apps.single.name, 'mysql');
    expect(apps.single.description, 'MySQL');
  });
}
