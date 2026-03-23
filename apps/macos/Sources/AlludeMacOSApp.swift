import SwiftUI

@main
struct AlludeMacOSApp: App {
    @StateObject private var appModel = AppModel()

    var body: some Scene {
        WindowGroup("Allude", id: "main") {
            RootView()
                .environmentObject(appModel)
                .frame(minWidth: 1220, minHeight: 780)
        }
        .windowStyle(.hiddenTitleBar)
        .commands {
            CaptureCommands(model: appModel)
        }

        WindowGroup("Quick Capture", id: "quick-capture") {
            QuickCaptureWindowView()
                .environmentObject(appModel)
                .frame(minWidth: 420, idealWidth: 480, maxWidth: 560, minHeight: 360, idealHeight: 420, maxHeight: 520)
        }
        .windowStyle(.hiddenTitleBar)
        .defaultPosition(.center)

        MenuBarExtra("Capture", systemImage: "tray.and.arrow.down.fill") {
            QuickCaptureMenuView()
                .environmentObject(appModel)
                .frame(width: 360)
        }
    }
}
