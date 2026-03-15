import SwiftUI

struct RootView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        NavigationSplitView {
            SidebarView()
        } detail: {
            ZStack {
                LinearGradient(
                    colors: [
                        Color(red: 0.96, green: 0.95, blue: 0.9),
                        Color(red: 0.82, green: 0.88, blue: 0.95)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .ignoresSafeArea()

                switch model.selectedSection {
                case .composer:
                    ComposerView()
                case .telescope:
                    TelescopeView()
                case .constellation:
                    ConstellationView()
                case .concept:
                    ConceptPageView()
                }
            }
            .overlay(alignment: .bottomTrailing) {
                if let message = model.errorMessage {
                    Text(message)
                        .font(.system(size: 12, weight: .medium))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)
                        .background(.ultraThinMaterial, in: Capsule())
                        .padding()
                }
            }
        }
        .task {
            if model.selectedThought == nil {
                model.startNewThought()
            }
        }
    }
}
