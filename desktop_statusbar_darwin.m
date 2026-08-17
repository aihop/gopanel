//go:build desktop && darwin

#import <Cocoa/Cocoa.h>

extern void GoPanelOpenCodeWorkspace(void);
extern void GoPanelShowMainWindow(void);
extern void GoPanelQuitApplication(void);

@interface GoPanelStatusBarController : NSObject
@property(nonatomic, strong) NSStatusItem *statusItem;
@property(nonatomic, strong) NSMenuItem *summaryItem;
@end

@implementation GoPanelStatusBarController

- (void)openCodeWorkspace:(id)sender {
    GoPanelOpenCodeWorkspace();
}

- (void)showMainWindow:(id)sender {
    GoPanelShowMainWindow();
}

- (void)quitApplication:(id)sender {
    GoPanelQuitApplication();
}

@end

static GoPanelStatusBarController *gopanelStatusBarController;

static void gopanel_run_on_main(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
        return;
    }
    dispatch_async(dispatch_get_main_queue(), block);
}

void gopanel_statusbar_start(void) {
    gopanel_run_on_main(^{
        if (gopanelStatusBarController != nil) {
            return;
        }

        gopanelStatusBarController = [GoPanelStatusBarController new];
        gopanelStatusBarController.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];

        NSStatusBarButton *button = gopanelStatusBarController.statusItem.button;
        if (@available(macOS 11.0, *)) {
            button.image = [NSImage imageWithSystemSymbolName:@"terminal" accessibilityDescription:@"GoPanel Code"];
            button.image.template = YES;
            button.imagePosition = NSImageLeft;
        } else {
            button.title = @"G";
        }
        button.toolTip = @"GoPanel Code";

        NSMenu *menu = [NSMenu new];
        NSMenuItem *openCode = [[NSMenuItem alloc] initWithTitle:@"打开 Code 工作台"
                                                          action:@selector(openCodeWorkspace:)
                                                   keyEquivalent:@""];
        openCode.target = gopanelStatusBarController;
        [menu addItem:openCode];

        gopanelStatusBarController.summaryItem = [[NSMenuItem alloc] initWithTitle:@"暂无需要处理的 Code 事项"
                                                                             action:nil
                                                                      keyEquivalent:@""];
        gopanelStatusBarController.summaryItem.enabled = NO;
        [menu addItem:gopanelStatusBarController.summaryItem];
        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *showWindow = [[NSMenuItem alloc] initWithTitle:@"显示 GoPanel"
                                                            action:@selector(showMainWindow:)
                                                     keyEquivalent:@""];
        showWindow.target = gopanelStatusBarController;
        [menu addItem:showWindow];
        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *quitApplication = [[NSMenuItem alloc] initWithTitle:@"退出 GoPanel"
                                                                 action:@selector(quitApplication:)
                                                          keyEquivalent:@""];
        quitApplication.target = gopanelStatusBarController;
        [menu addItem:quitApplication];
        gopanelStatusBarController.statusItem.menu = menu;
    });
}

void gopanel_statusbar_update(int attention, int running, int queued) {
    gopanel_run_on_main(^{
        if (gopanelStatusBarController == nil) {
            return;
        }

        NSStatusBarButton *button = gopanelStatusBarController.statusItem.button;
        button.title = attention > 99 ? @" 99+" : (attention > 0 ? [NSString stringWithFormat:@" %d", attention] : @"");
        button.toolTip = [NSString stringWithFormat:@"GoPanel Code：需处理 %d，运行中 %d，排队 %d", attention, running, queued];
        gopanelStatusBarController.summaryItem.title = [NSString stringWithFormat:@"需处理 %d · 运行中 %d · 排队 %d", attention, running, queued];
        [NSApp dockTile].badgeLabel = attention > 0 ? [NSString stringWithFormat:@"%d", attention] : @"";
    });
}

void gopanel_statusbar_stop(void) {
    gopanel_run_on_main(^{
        [NSApp dockTile].badgeLabel = @"";
        if (gopanelStatusBarController != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:gopanelStatusBarController.statusItem];
            gopanelStatusBarController = nil;
        }
    });
}
