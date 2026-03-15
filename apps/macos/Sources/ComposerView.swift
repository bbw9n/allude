import SwiftUI

struct ComposerView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        HStack(spacing: 24) {
            VStack(alignment: .leading, spacing: 16) {
                Text(model.selectedThought == nil ? "Write Thought" : "Revise Thought")
                    .font(.system(size: 34, weight: .bold, design: .serif))

                TextEditor(text: $model.draft)
                    .font(.system(size: 16, weight: .regular, design: .serif))
                    .scrollContentBackground(.hidden)
                    .padding(18)
                    .background(
                        RoundedRectangle(cornerRadius: 24, style: .continuous)
                            .fill(Color.white.opacity(0.72))
                    )
                    .overlay(alignment: .topLeading) {
                        if model.draft.isEmpty {
                            Text("Capture an observation, question, argument, or quote...")
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 24)
                                .padding(.vertical, 26)
                        }
                    }

                HStack {
                    Button {
                        Task { await model.saveDraft() }
                    } label: {
                        if model.isSaving {
                            ProgressView()
                        } else {
                            Text(model.selectedThought == nil ? "Save Thought" : "Save Revision")
                        }
                    }
                    .buttonStyle(.borderedProminent)

                    if let thought = model.selectedThought {
                        Text("Current version v\(thought.currentVersion.version)")
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(.secondary)
                    }
                }
            }

            AIInsightPanel(thought: model.selectedThought)
                .frame(width: 360)
        }
        .padding(32)
    }
}

struct AIInsightPanel: View {
    let thought: Thought?

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("AI Suggestions")
                .font(.system(size: 22, weight: .bold))

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Related concepts")
                        .font(.system(size: 13, weight: .semibold))
                    if let thought, !thought.concepts.isEmpty {
                        FlowLayout(items: thought.concepts.map(\.name))
                    } else {
                        Text("Save a thought to extract concept nodes.")
                            .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Counterarguments")
                        .font(.system(size: 13, weight: .semibold))
                    if let thought, let related = thought.relatedThoughts, !related.isEmpty {
                        ForEach(related.prefix(3)) { candidate in
                            Text(candidate.currentVersion.content)
                                .font(.system(size: 12))
                        }
                    } else {
                        Text("Contrasting thoughts will appear here once the graph has neighbors.")
                            .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Analysis notes")
                        .font(.system(size: 13, weight: .semibold))
                    if let thought, !thought.processingNotes.isEmpty {
                        ForEach(thought.processingNotes, id: \.self) { note in
                            Text("• \(note)")
                                .font(.system(size: 12))
                        }
                    } else {
                        Text("The backend marks processing state here after concept extraction and linking.")
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer()
        }
        .padding(24)
        .background(Color.white.opacity(0.6), in: RoundedRectangle(cornerRadius: 26))
    }
}
