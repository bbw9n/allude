import SwiftUI

struct ConceptPageView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            if let concept = model.selectedConcept {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        conceptHeader(concept)

                        HStack(alignment: .top, spacing: 24) {
                            VStack(alignment: .leading, spacing: 18) {
                                conceptThoughtSection(
                                    title: "Top Thoughts",
                                    subtitle: "The strongest current thoughts anchored to this concept.",
                                    thoughts: concept.topThoughts ?? []
                                )

                                conceptThoughtSection(
                                    title: "Contradictions",
                                    subtitle: "Where this concept produces disagreement, tension, or critique.",
                                    thoughts: concept.contradictionThoughts ?? []
                                )
                            }

                            VStack(alignment: .leading, spacing: 18) {
                                relatedConceptsCard(concept)
                                aliasesCard(concept)
                                guidanceCard
                            }
                            .frame(width: 300)
                        }
                    }
                }

                Spacer(minLength: 0)
            } else {
                VStack(alignment: .leading, spacing: 10) {
                    Text("No concept selected")
                        .font(.system(size: 26, weight: .bold))
                    Text("Pick a concept from a thought to open its dynamic knowledge hub.")
                        .foregroundStyle(.secondary)
                }
                Spacer()
            }
        }
        .padding(32)
    }

    @ViewBuilder
    private func conceptHeader(_ concept: Concept) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(concept.name.capitalized)
                .font(.system(size: 36, weight: .black, design: .serif))

            Text("Living concept page")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(.secondary)

            if let description = concept.description, !description.isEmpty {
                Text(description)
                    .font(.system(size: 15, weight: .regular, design: .serif))
                    .foregroundStyle(.primary.opacity(0.84))
            } else {
                Text("This concept page is built dynamically from linked thoughts, related concepts, and tensions in the idea graph.")
                    .font(.system(size: 15, weight: .regular, design: .serif))
                    .foregroundStyle(.primary.opacity(0.84))
            }

            HStack(spacing: 10) {
                ConceptMetricPill(label: "Thoughts", value: "\(concept.thoughtCount ?? concept.topThoughts?.count ?? 0)")
                ConceptMetricPill(label: "Related", value: "\(concept.relatedConcepts?.count ?? 0)")
                if let conceptType = concept.conceptType, !conceptType.isEmpty {
                    ConceptMetricPill(label: "Type", value: conceptType.capitalized)
                }
            }
        }
        .padding(20)
        .background(Color.white.opacity(0.7), in: RoundedRectangle(cornerRadius: 24))
    }

    @ViewBuilder
    private func conceptThoughtSection(title: String, subtitle: String, thoughts: [Thought]) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.system(size: 18, weight: .bold))

            Text(subtitle)
                .font(.system(size: 12))
                .foregroundStyle(.secondary)

            if thoughts.isEmpty {
                Text("Nothing here yet.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .padding(16)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.white.opacity(0.55), in: RoundedRectangle(cornerRadius: 18))
            } else {
                ForEach(thoughts) { thought in
                    Button {
                        model.selectThought(thought)
                        model.selectedSection = .constellation
                    } label: {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(thought.currentVersion.content)
                                .font(.system(size: 14, weight: .medium, design: .serif))
                                .frame(maxWidth: .infinity, alignment: .leading)

                            if !thought.concepts.isEmpty {
                                FlowLayout(items: Array(thought.concepts.prefix(4)).map(\.name))
                            }
                        }
                        .padding(16)
                        .background(Color.white.opacity(0.68), in: RoundedRectangle(cornerRadius: 18))
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    @ViewBuilder
    private func relatedConceptsCard(_ concept: Concept) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Related Concepts")
                .font(.system(size: 18, weight: .bold))

            if let related = concept.relatedConcepts, !related.isEmpty {
                FlowLayout(items: related.map(\.name)) { item in
                    Button(item) {
                        if let next = related.first(where: { $0.name == item }) {
                            Task { await model.loadConcept(id: next.id) }
                        }
                    }
                    .buttonStyle(.bordered)
                }
            } else {
                Text("Nearby concepts will appear here as the graph thickens.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            }
        }
        .padding(16)
        .background(Color.white.opacity(0.62), in: RoundedRectangle(cornerRadius: 18))
    }

    @ViewBuilder
    private func aliasesCard(_ concept: Concept) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Aliases")
                .font(.system(size: 18, weight: .bold))

            if let aliases = concept.aliases, !aliases.isEmpty {
                FlowLayout(items: aliases.map(\.alias))
            } else {
                Text("Canonical naming is still sparse for this concept.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            }
        }
        .padding(16)
        .background(Color.white.opacity(0.62), in: RoundedRectangle(cornerRadius: 18))
    }

    private var guidanceCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Jump back into the graph")
                .font(.system(size: 18, weight: .bold))

            Text("Pick any thought from this page to reopen its version history, AI notes, and constellation neighborhood.")
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
        }
        .padding(16)
        .background(Color.white.opacity(0.62), in: RoundedRectangle(cornerRadius: 18))
    }
}

private struct ConceptMetricPill: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label.uppercased())
                .font(.system(size: 9, weight: .bold))
                .foregroundStyle(.secondary)
            Text(value)
                .font(.system(size: 12, weight: .semibold))
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(Color.black.opacity(0.05), in: Capsule())
    }
}
