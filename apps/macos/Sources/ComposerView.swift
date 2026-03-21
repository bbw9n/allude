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
                    .onChange(of: model.draft) { _, _ in
                        model.scheduleDraftSuggestions()
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

            AIInsightPanel(
                thought: model.selectedThought,
                suggestions: model.draftSuggestions,
                isRefreshing: model.isRefreshingDraftSuggestions,
                openThought: { thought in model.selectThought(thought) }
            )
                .frame(width: 360)
        }
        .padding(32)
    }
}

struct AIInsightPanel: View {
    let thought: Thought?
    let suggestions: DraftSuggestions
    let isRefreshing: Bool
    let openThought: (Thought) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack {
                Text("AI Suggestions")
                    .font(.system(size: 22, weight: .bold))
                Spacer()
                if isRefreshing {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Related concepts")
                        .font(.system(size: 13, weight: .semibold))
                    if !suggestions.relatedConcepts.isEmpty {
                        FlowLayout(items: suggestions.relatedConcepts)
                    } else if let thought, !thought.concepts.isEmpty {
                        FlowLayout(items: thought.concepts.map(\.name))
                    } else {
                        Text("Keep drafting and concept links will appear here.")
                            .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Supporting thoughts")
                        .font(.system(size: 13, weight: .semibold))
                    if !suggestions.supportingThoughts.isEmpty {
                        ForEach(suggestions.supportingThoughts.prefix(3)) { candidate in
                            Button {
                                openThought(candidate)
                            } label: {
                                Text(candidate.currentVersion.content)
                                    .font(.system(size: 12))
                                    .frame(maxWidth: .infinity, alignment: .leading)
                            }
                            .buttonStyle(.plain)
                        }
                    } else {
                        Text("Nearby supporting thoughts will show up here.")
                        .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Counterarguments")
                        .font(.system(size: 13, weight: .semibold))
                    if !suggestions.counterThoughts.isEmpty {
                        ForEach(suggestions.counterThoughts.prefix(3)) { candidate in
                            Button {
                                openThought(candidate)
                            } label: {
                                Text(candidate.currentVersion.content)
                                    .font(.system(size: 12))
                                    .frame(maxWidth: .infinity, alignment: .leading)
                            }
                            .buttonStyle(.plain)
                        }
                    } else {
                        Text("Tensions and competing framings will appear here.")
                        .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Reframes")
                        .font(.system(size: 13, weight: .semibold))
                    if !suggestions.reframes.isEmpty {
                        ForEach(suggestions.reframes, id: \.self) { reframe in
                            Text("• \(reframe)")
                                .font(.system(size: 12))
                        }
                    } else {
                        Text("Revision suggestions will appear as you develop the draft.")
                            .foregroundStyle(.secondary)
                    }
                }
            }

            GroupBox {
                VStack(alignment: .leading, spacing: 10) {
                    Text("Analysis notes")
                        .font(.system(size: 13, weight: .semibold))
                    if !suggestions.notes.isEmpty {
                        ForEach(suggestions.notes, id: \.self) { note in
                            Text("• \(note)")
                                .font(.system(size: 12))
                        }
                    } else if let thought, !thought.processingNotes.isEmpty {
                        ForEach(thought.processingNotes, id: \.self) { note in
                            Text("• \(note)")
                                .font(.system(size: 12))
                        }
                    } else {
                        Text("The backend will annotate how it interpreted the draft.")
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
