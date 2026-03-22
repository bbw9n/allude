import SwiftUI

struct TelescopeView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            Text("Idea Telescope")
                .font(.system(size: 34, weight: .bold, design: .serif))

            HStack {
                TextField("Ask Allude", text: $model.telescopeQuery)
                    .textFieldStyle(.roundedBorder)
                    .font(.system(size: 16))
                    .onSubmit {
                        Task { await model.search() }
                    }

                Button {
                    Task { await model.search() }
                } label: {
                    if model.isSearching {
                        ProgressView()
                    } else {
                        Text("Search")
                    }
                }
                .buttonStyle(.borderedProminent)
            }

            VStack(alignment: .leading, spacing: 12) {
                Text("Search readout")
                    .font(.system(size: 18, weight: .bold))

                Text(model.telescopeNarrative)
                    .font(.system(size: 14, weight: .regular, design: .serif))
                    .foregroundStyle(.primary.opacity(0.84))

                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 10) {
                        ForEach(model.telescopeSuggestedJumps, id: \.query) { jump in
                            Button(jump.label) {
                                model.applyTelescopeSuggestion(jump.query)
                                Task { await model.search() }
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        }
                    }
                    .padding(.vertical, 2)
                }

                if !model.telescopeResult.seedConcepts.isEmpty {
                    FlowLayout(items: model.telescopeResult.seedConcepts.map(\.name))
                }
            }
            .padding(18)
            .background(Color.white.opacity(0.64), in: RoundedRectangle(cornerRadius: 20))

            HStack(alignment: .top, spacing: 24) {
                VStack(alignment: .leading, spacing: 16) {
                    HStack {
                        Text("Cluster signals")
                            .font(.system(size: 18, weight: .bold))
                        Spacer()
                        if !model.telescopeClusters.isEmpty {
                            Button("All") {
                                model.selectSearchCluster(label: nil)
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        }
                    }

                    if model.telescopeClusters.isEmpty {
                        Text("Try queries like “connections between stoicism and startup culture”.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(model.telescopeClusters, id: \.label) { cluster in
                            Button {
                                model.selectSearchCluster(label: cluster.label)
                                if let firstConcept = cluster.concepts.first {
                                    Task { await model.loadConcept(id: firstConcept.id) }
                                }
                            } label: {
                                VStack(alignment: .leading, spacing: 8) {
                                    HStack {
                                        Text(cluster.label)
                                            .font(.system(size: 16, weight: .semibold))
                                        Spacer()
                                        if model.selectedSearchClusterLabel == cluster.label {
                                            Image(systemName: "scope")
                                        }
                                    }
                                    FlowLayout(items: cluster.concepts.map(\.name))
                                    Text("\(cluster.thoughtIds.count) connected thoughts")
                                        .font(.system(size: 12))
                                        .foregroundStyle(.secondary)
                                }
                                .padding(14)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(
                                    (model.selectedSearchClusterLabel == cluster.label ? Color.black.opacity(0.08) : Color.white.opacity(0.58)),
                                    in: RoundedRectangle(cornerRadius: 16)
                                )
                            }
                            .buttonStyle(.plain)
                        }
                    }

                    if !model.telescopeRelatedCurrents.isEmpty {
                        Divider()
                            .padding(.vertical, 4)

                        Text("Related currents")
                            .font(.system(size: 16, weight: .bold))

                        ForEach(model.telescopeRelatedCurrents) { current in
                            VStack(alignment: .leading, spacing: 6) {
                                Text(current.title)
                                    .font(.system(size: 14, weight: .semibold))
                                if let summary = current.summary, !summary.isEmpty {
                                    Text(summary)
                                        .font(.system(size: 12, weight: .regular, design: .serif))
                                        .foregroundStyle(.secondary)
                                }
                            }
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 14))
                        }
                    }
                }
                .frame(width: 300)

                VStack(alignment: .leading, spacing: 14) {
                    HStack {
                        Text("Telescope results")
                            .font(.system(size: 18, weight: .bold))
                        Spacer()
                        Text("\(model.filteredSearchThoughts.count)")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(.secondary)
                    }

                    if model.filteredSearchThoughts.isEmpty {
                        Text("Search results will appear here.")
                            .foregroundStyle(.secondary)
                    } else {
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                ForEach(model.filteredSearchThoughts) { thought in
                                    Button {
                                        model.selectThought(thought)
                                        model.selectedSection = .telescope
                                    } label: {
                                        VStack(alignment: .leading, spacing: 8) {
                                            Text(thought.currentVersion.content)
                                                .font(.system(size: 14, weight: .medium, design: .serif))
                                            FlowLayout(items: thought.concepts.map(\.name))
                                            if let related = thought.relatedThoughts, !related.isEmpty {
                                                Text("Leads to \(related.count) adjacent thought\(related.count == 1 ? "" : "s")")
                                                    .font(.system(size: 11, weight: .medium))
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .padding(16)
                                        .background(Color.white.opacity(0.7), in: RoundedRectangle(cornerRadius: 18))
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                    }

                    if let graph = model.telescopeGraph {
                        Divider()
                            .padding(.vertical, 4)

                        VStack(alignment: .leading, spacing: 10) {
                            Text("Graph preview")
                                .font(.system(size: 16, weight: .bold))

                            Text("Centered on \(graph.center.thought.currentVersion.content)")
                                .font(.system(size: 13, weight: .regular, design: .serif))
                                .lineLimit(2)

                            ForEach(Array(graph.nodes.prefix(4))) { node in
                                Button {
                                    model.selectThought(node.thought)
                                    model.selectedSection = .constellation
                                } label: {
                                    HStack(alignment: .top, spacing: 8) {
                                        Text("•")
                                        Text(node.thought.currentVersion.content)
                                            .font(.system(size: 12, weight: .regular, design: .serif))
                                            .lineLimit(2)
                                    }
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(14)
                        .background(Color.white.opacity(0.56), in: RoundedRectangle(cornerRadius: 16))
                    }
                }

                ThoughtDetailView(thought: model.selectedThought)
                    .frame(width: 380)
            }

            Spacer()
        }
        .padding(32)
    }
}
