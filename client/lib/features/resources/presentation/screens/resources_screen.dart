import 'package:flutter/material.dart';

import '../../../container/presentation/screens/container_list_screen.dart';
import '../../../apps/presentation/screens/apps_list_screen.dart';
import '../../../website/presentation/screens/website_list_screen.dart';
import '../../../database/presentation/screens/database_list_screen.dart';
import '../../../ssl/presentation/screens/ssl_list_screen.dart';
import '../../../../shared/widgets/panel/glass_tabs.dart';

enum ResourceTab { containers, apps, websites, databases, ssl }

class ResourcesScreen extends StatefulWidget {
  const ResourcesScreen({super.key});

  @override
  State<ResourcesScreen> createState() => _ResourcesScreenState();
}

class _ResourcesScreenState extends State<ResourcesScreen> {
  ResourceTab _tab = ResourceTab.containers;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        toolbarHeight: 72,
        titleSpacing: 16,
        title: Padding(
          padding: const EdgeInsets.only(top: 8, bottom: 8),
          child: _segmented(),
        ),
      ),
      body: Column(children: [Expanded(child: _buildTabBody())]),
    );
  }

  Widget _segmented() {
    return GlassTabs<ResourceTab>(
      items: const [
        GlassTabItem(value: ResourceTab.websites, label: '网站'),
        GlassTabItem(value: ResourceTab.databases, label: '数据库'),
        GlassTabItem(value: ResourceTab.ssl, label: 'SSL'),
        GlassTabItem(value: ResourceTab.containers, label: '容器'),
        GlassTabItem(value: ResourceTab.apps, label: '应用'),
      ],
      selected: _tab,
      onChanged: (v) {
        setState(() => _tab = v);
      },
      outerPadding: const EdgeInsets.all(4),
      tabPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      borderRadius: 16,
      tabRadius: 12,
      blurSigma: 12,
    );
  }

  Widget _buildTabBody() {
    switch (_tab) {
      case ResourceTab.containers:
        return const ContainerListScreen.embedded();
      case ResourceTab.apps:
        return const AppsListScreen.embedded();
      case ResourceTab.websites:
        return const WebsiteListScreen.embedded();
      case ResourceTab.databases:
        return const DatabaseListScreen.embedded();
      case ResourceTab.ssl:
        return const SslListScreen.embedded();
    }
  }
}
