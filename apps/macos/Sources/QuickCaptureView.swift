import SwiftUI

struct QuickCaptureWindowView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                Text("Quick Capture")
                    .font(.system(size: 24, weight: .bold, design: .serif))
                Spacer()
                if model.isSavingCapture {
                    ProgressView()
                }
            }

            Text("Drop in a quote, link, fragment, or half-formed idea. It will land in your inbox first.")
                .font(.system(size: 13))
                .foregroundStyle(.secondary)

            captureComposer

            HStack(spacing: 12) {
                Button {
                    model.prefillCaptureFromClipboard()
                } label: {
                    Label("Paste Clipboard", systemImage: "doc.on.clipboard")
                }
                .buttonStyle(.bordered)

                Button {
                    model.selectedSection = .inbox
                    openWindow(id: "main")
                    Task { await model.loadInbox() }
                } label: {
                    Label("Open Inbox", systemImage: "tray.full")
                }
                .buttonStyle(.bordered)
            }

            Spacer()
        }
        .padding(22)
        .background(
            LinearGradient(
                colors: [
                    Color(red: 0.97, green: 0.95, blue: 0.9),
                    Color(red: 0.88, green: 0.92, blue: 0.96)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
    }

    private var captureComposer: some View {
        VStack(alignment: .leading, spacing: 12) {
            TextField("Capture anything worth revisiting...", text: $model.captureDraft, axis: .vertical)
                .textFieldStyle(.roundedBorder)

            TextField("Source title", text: $model.captureSourceTitle)
                .textFieldStyle(.roundedBorder)

            TextField("Source URL", text: $model.captureSourceURL)
                .textFieldStyle(.roundedBorder)

            TextField("Source app", text: $model.captureSourceApp)
                .textFieldStyle(.roundedBorder)

            Button {
                Task { await model.createCapture() }
            } label: {
                Text("Save to Inbox")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
        }
        .padding(16)
        .background(Color.white.opacity(0.72), in: RoundedRectangle(cornerRadius: 18))
    }
}

struct QuickCaptureMenuView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Quick Capture")
                .font(.system(size: 18, weight: .bold, design: .serif))

            TextField("Paste a quote, link, or fragment...", text: $model.captureDraft, axis: .vertical)
                .textFieldStyle(.roundedBorder)

            HStack(spacing: 10) {
                Button("Paste") {
                    model.prefillCaptureFromClipboard()
                }
                .buttonStyle(.bordered)

                Button("Save") {
                    Task { await model.createCapture() }
                }
                .buttonStyle(.borderedProminent)
            }

            Divider()

            Button {
                openWindow(id: "quick-capture")
            } label: {
                Label("Open Quick Capture Window", systemImage: "macwindow")
            }
            .buttonStyle(.plain)

            Button {
                model.selectedSection = .inbox
                openWindow(id: "main")
                Task { await model.loadInbox() }
            } label: {
                Label("Open Inbox", systemImage: "tray.full")
            }
            .buttonStyle(.plain)
        }
        .padding(14)
    }
}

struct CaptureCommands: Commands {
    @ObservedObject var model: AppModel
    @Environment(\.openWindow) private var openWindow

    var body: some Commands {
        CommandMenu("Capture") {
            Button("Quick Capture") {
                openWindow(id: "quick-capture")
            }
            .keyboardShortcut("n", modifiers: [.command, .shift])

            Button("Open Inbox") {
                model.selectedSection = .inbox
                openWindow(id: "main")
                Task { await model.loadInbox() }
            }
            .keyboardShortcut("i", modifiers: [.command, .shift])

            Button("Capture Clipboard") {
                model.prefillCaptureFromClipboard()
                openWindow(id: "quick-capture")
            }
            .keyboardShortcut("v", modifiers: [.command, .shift, .option])
        }
    }
}
