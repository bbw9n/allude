import SwiftUI

struct ConceptPageView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            if let concept = model.selectedConcept {
                Text(concept.name.capitalized)
                    .font(.system(size: 36, weight: .black, design: .serif))

                Text("Living concept page")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(.secondary)

                HStack(alignment: .top, spacing: 24) {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Top Thoughts")
                            .font(.system(size: 18, weight: .bold))

                        ForEach(concept.topThoughts ?? []) { thought in
                            Button {
                                model.selectThought(thought)
                                model.selectedSection = .constellation
                            } label: {
                                Text(thought.currentVersion.content)
                                    .font(.system(size: 14, weight: .medium, design: .serif))
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(16)
                                    .background(Color.white.opacity(0.68), in: RoundedRectangle(cornerRadius: 18))
                            }
                            .buttonStyle(.plain)
                        }
                    }

                    VStack(alignment: .leading, spacing: 14) {
                        Text("Related Concepts")
                            .font(.system(size: 18, weight: .bold))

                        FlowLayout(items: concept.relatedConcepts?.map(\.name) ?? []) { item in
                            Button(item) {
                                if let next = concept.relatedConcepts?.first(where: { $0.name == item }) {
                                    Task { await model.loadConcept(id: next.id) }
                                }
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                    .frame(width: 280)
                }

                Spacer()
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
}
