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

            HStack(alignment: .top, spacing: 24) {
                VStack(alignment: .leading, spacing: 16) {
                    Text("Cluster signals")
                        .font(.system(size: 18, weight: .bold))

                    if model.searchResult.clusters.isEmpty {
                        Text("Try queries like “connections between stoicism and startup culture”.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(model.searchResult.clusters, id: \.label) { cluster in
                            VStack(alignment: .leading, spacing: 8) {
                                Text(cluster.label)
                                    .font(.system(size: 16, weight: .semibold))
                                FlowLayout(items: cluster.concepts.map(\.name))
                                Text("\(cluster.thoughtIds.count) connected thoughts")
                                    .font(.system(size: 12))
                                    .foregroundStyle(.secondary)
                            }
                            .padding(14)
                            .background(Color.white.opacity(0.58), in: RoundedRectangle(cornerRadius: 16))
                        }
                    }
                }
                .frame(width: 300)

                VStack(alignment: .leading, spacing: 14) {
                    Text("Matching thoughts")
                        .font(.system(size: 18, weight: .bold))

                    if model.searchResult.thoughts.isEmpty {
                        Text("Search results will appear here.")
                            .foregroundStyle(.secondary)
                    } else {
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                ForEach(model.searchResult.thoughts) { thought in
                                    Button {
                                        model.selectThought(thought)
                                        model.selectedSection = .constellation
                                    } label: {
                                        VStack(alignment: .leading, spacing: 8) {
                                            Text(thought.currentVersion.content)
                                                .font(.system(size: 14, weight: .medium, design: .serif))
                                            FlowLayout(items: thought.concepts.map(\.name))
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
                }
            }

            Spacer()
        }
        .padding(32)
    }
}
