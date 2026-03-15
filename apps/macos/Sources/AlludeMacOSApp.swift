import SwiftUI

@main
struct AlludeMacOSApp: App {
    @StateObject private var appModel = AppModel()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(appModel)
                .frame(minWidth: 1220, minHeight: 780)
        }
        .windowStyle(.hiddenTitleBar)
    }
}
