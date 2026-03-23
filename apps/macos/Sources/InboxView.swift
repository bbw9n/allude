import SwiftUI

struct InboxView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        HStack(alignment: .top, spacing: 24) {
            VStack(alignment: .leading, spacing: 18) {
                HStack {
                    Text("Inbox")
                        .font(.system(size: 34, weight: .bold, design: .serif))
                    if model.isRefreshingInbox || model.isSavingCapture {
                        ProgressView()
                    }
                }

                VStack(alignment: .leading, spacing: 12) {
                    Text("Quick Capture")
                        .font(.system(size: 18, weight: .bold))

                    TextField("Paste a quote, link, fragment, or idea...", text: $model.captureDraft, axis: .vertical)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source title (optional)", text: $model.captureSourceTitle)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source URL (optional)", text: $model.captureSourceURL)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source app (optional)", text: $model.captureSourceApp)
                        .textFieldStyle(.roundedBorder)

                    Button {
                        Task { await model.createCapture() }
                    } label: {
                        Text("Save to Inbox")
                    }
                    .buttonStyle(.borderedProminent)
                }
                .padding(18)
                .background(Color.white.opacity(0.65), in: RoundedRectangle(cornerRadius: 20))

                Spacer()
            }
            .frame(width: 360)

            VStack(alignment: .leading, spacing: 16) {
                Text("Captured Items")
                    .font(.system(size: 22, weight: .bold))

                if model.captures.isEmpty {
                    Text("Nothing captured yet. Drop in scraps here first, then promote the ones worth developing.")
                        .foregroundStyle(.secondary)
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 14) {
                            ForEach(model.captures) { capture in
                                Button {
                                    model.selectCapture(capture)
                                } label: {
                                    VStack(alignment: .leading, spacing: 8) {
                                        HStack {
                                            Text(capture.sourceTitle ?? capture.sourceType?.capitalized ?? "Capture")
                                                .font(.system(size: 15, weight: .bold))
                                            Spacer()
                                            Text(capture.status.capitalized)
                                                .font(.system(size: 11, weight: .semibold))
                                                .foregroundStyle(.secondary)
                                        }

                                        Text(capture.content)
                                            .font(.system(size: 13, weight: .medium, design: .serif))
                                            .lineLimit(4)
                                            .frame(maxWidth: .infinity, alignment: .leading)

                                        HStack(spacing: 8) {
                                            if let app = capture.sourceApp, !app.isEmpty {
                                                inboxBadge(app)
                                            }
                                            if let type = capture.sourceType, !type.isEmpty {
                                                inboxBadge(type.capitalized)
                                            }
                                            if capture.promotedThoughtId != nil {
                                                inboxBadge("Promoted")
                                            }
                                        }
                                    }
                                    .padding(16)
                                    .background(
                                        (model.selectedCapture?.id == capture.id ? Color.black.opacity(0.08) : Color.white.opacity(0.65)),
                                        in: RoundedRectangle(cornerRadius: 18)
                                    )
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }

            InboxDetailView(capture: model.selectedCapture)
                .frame(width: 380)
        }
        .padding(32)
        .task {
            if model.captures.isEmpty {
                await model.loadInbox()
            }
        }
    }

    private func inboxBadge(_ label: String) -> some View {
        Text(label)
            .font(.system(size: 11, weight: .semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(Color.black.opacity(0.07), in: Capsule())
    }
}

private struct InboxDetailView: View {
    @EnvironmentObject private var model: AppModel
    let capture: CaptureItem?

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Capture Detail")
                .font(.system(size: 24, weight: .bold))

            if let capture {
                Text(capture.sourceTitle ?? "Untitled Capture")
                    .font(.system(size: 22, weight: .bold, design: .serif))

                Text(capture.content)
                    .font(.system(size: 14, weight: .regular, design: .serif))
                    .frame(maxWidth: .infinity, alignment: .leading)

                if let sourceURL = capture.sourceUrl, !sourceURL.isEmpty {
                    Text(sourceURL)
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                }

                HStack(spacing: 10) {
                    Button {
                        Task { await model.promoteSelectedCapture() }
                    } label: {
                        Text(capture.promotedThoughtId == nil ? "Promote to Thought" : "Open Thought")
                    }
                    .buttonStyle(.borderedProminent)

                    Button {
                        Task { await model.archiveSelectedCapture() }
                    } label: {
                        Text("Archive")
                    }
                    .buttonStyle(.bordered)
                }

                if let promotedThought = capture.promotedThought {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Promoted Thought")
                            .font(.system(size: 14, weight: .bold))

                        Button {
                            model.selectThought(promotedThought)
                            model.selectedSection = .constellation
                        } label: {
                            Text(promotedThought.currentVersion.content)
                                .font(.system(size: 13, weight: .medium, design: .serif))
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(14)
                                .background(Color.white.opacity(0.62), in: RoundedRectangle(cornerRadius: 16))
                        }
                        .buttonStyle(.plain)
                    }
                }
            } else {
                Text("Pick a capture to archive it, promote it into a thought, or jump back into the graph.")
                    .foregroundStyle(.secondary)
            }

            Spacer()
        }
        .padding(24)
        .background(Color.white.opacity(0.55), in: RoundedRectangle(cornerRadius: 24))
    }
}
