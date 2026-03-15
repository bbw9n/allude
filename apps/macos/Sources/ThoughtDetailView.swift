import SwiftUI

struct ThoughtDetailView: View {
    @EnvironmentObject private var model: AppModel
    let thought: Thought?

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Thought Evolution")
                .font(.system(size: 24, weight: .bold))

            if let thought {
                Text(thought.currentVersion.content)
                    .font(.system(size: 16, weight: .regular, design: .serif))

                FlowLayout(items: thought.concepts.map(\.name)) { item in
                    Button(item) {
                        if let concept = thought.concepts.first(where: { $0.name == item }) {
                            Task { await model.loadConcept(id: concept.id) }
                            model.selectedSection = .concept
                        }
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }

                VStack(alignment: .leading, spacing: 10) {
                    Text("Timeline")
                        .font(.system(size: 13, weight: .semibold))

                    if let versions = thought.versions {
                        ForEach(versions) { version in
                            VStack(alignment: .leading, spacing: 4) {
                                Text("v\(version.version) • \(version.createdAt)")
                                    .font(.system(size: 12, weight: .bold))
                                Text(version.content)
                                    .font(.system(size: 12))
                                    .foregroundStyle(.secondary)
                            }
                            .padding(10)
                            .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
                        }
                    }
                }

                VStack(alignment: .leading, spacing: 10) {
                    Text("Related ideas")
                        .font(.system(size: 13, weight: .semibold))

                    ForEach(thought.relatedThoughts ?? []) { related in
                        Button {
                            model.selectThought(related)
                        } label: {
                            Text(related.currentVersion.content)
                                .font(.system(size: 12))
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(10)
                                .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
                        }
                        .buttonStyle(.plain)
                    }
                }
            } else {
                Text("Select a thought to inspect versions, concepts, and nearby links.")
                    .foregroundStyle(.secondary)
            }

            Spacer()
        }
        .padding(24)
        .background(Color.white.opacity(0.55), in: RoundedRectangle(cornerRadius: 24))
    }
}
