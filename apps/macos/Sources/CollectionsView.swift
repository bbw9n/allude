import SwiftUI

struct CollectionsView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        HStack(alignment: .top, spacing: 24) {
            VStack(alignment: .leading, spacing: 18) {
                HStack {
                    Text("Collections")
                        .font(.system(size: 34, weight: .bold, design: .serif))
                    if model.isRefreshingCollections || model.isSavingCollection {
                        ProgressView()
                    }
                }

                VStack(alignment: .leading, spacing: 12) {
                    Text("Create Collection")
                        .font(.system(size: 18, weight: .bold))

                    TextField("Collection title", text: $model.newCollectionTitle)
                        .textFieldStyle(.roundedBorder)

                    TextField("Description", text: $model.newCollectionDescription, axis: .vertical)
                        .textFieldStyle(.roundedBorder)

                    Button {
                        Task { await model.createCollection() }
                    } label: {
                        Text("Create Collection")
                    }
                    .buttonStyle(.borderedProminent)
                }
                .padding(18)
                .background(Color.white.opacity(0.65), in: RoundedRectangle(cornerRadius: 20))

                if let selectedThought = model.selectedThought {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Save Current Thought")
                            .font(.system(size: 18, weight: .bold))

                        Text(selectedThought.currentVersion.content)
                            .font(.system(size: 14, weight: .regular, design: .serif))
                            .lineLimit(4)
                            .foregroundStyle(.primary.opacity(0.85))

                        if model.collections.isEmpty {
                            Text("Create a collection first, then you can add this thought to it.")
                                .font(.system(size: 12))
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(model.collections) { collection in
                                Button {
                                    Task { await model.addSelectedThoughtToCollection(collection) }
                                } label: {
                                    HStack {
                                        VStack(alignment: .leading, spacing: 4) {
                                            Text(collection.title)
                                                .font(.system(size: 13, weight: .semibold))
                                            Text("\(collection.items?.count ?? 0) saved thoughts")
                                                .font(.system(size: 11))
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Image(systemName: "plus.circle")
                                    }
                                    .padding(12)
                                    .background(Color.white.opacity(0.58), in: RoundedRectangle(cornerRadius: 14))
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(18)
                    .background(Color.white.opacity(0.65), in: RoundedRectangle(cornerRadius: 20))
                }

                Spacer()
            }
            .frame(width: 360)

            VStack(alignment: .leading, spacing: 16) {
                Text("Library")
                    .font(.system(size: 22, weight: .bold))

                if model.collections.isEmpty {
                    Text("No collections yet. Start one to gather related thoughts into reusable idea sets.")
                        .foregroundStyle(.secondary)
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 14) {
                            ForEach(model.collections) { collection in
                                Button {
                                    model.selectCollection(collection)
                                } label: {
                                    VStack(alignment: .leading, spacing: 8) {
                                        HStack {
                                            Text(collection.title)
                                                .font(.system(size: 16, weight: .bold))
                                            Spacer()
                                            Text("\(collection.items?.count ?? 0)")
                                                .font(.system(size: 12, weight: .semibold))
                                                .foregroundStyle(.secondary)
                                        }

                                        if let description = collection.description, !description.isEmpty {
                                            Text(description)
                                                .font(.system(size: 12))
                                                .foregroundStyle(.secondary)
                                        }

                                        if let items = collection.items, !items.isEmpty {
                                            ForEach(items.prefix(3)) { item in
                                                if let thought = item.thought {
                                                    Text(thought.currentVersion.content)
                                                        .font(.system(size: 12, weight: .medium, design: .serif))
                                                        .lineLimit(2)
                                                        .frame(maxWidth: .infinity, alignment: .leading)
                                                }
                                            }
                                        } else {
                                            Text("Empty collection")
                                                .font(.system(size: 12))
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                    .padding(16)
                                    .background(
                                        (model.selectedCollection?.id == collection.id ? Color.black.opacity(0.08) : Color.white.opacity(0.65)),
                                        in: RoundedRectangle(cornerRadius: 18)
                                    )
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }

            CollectionDetailView(collection: model.selectedCollection)
                .frame(width: 380)
        }
        .padding(32)
        .task {
            if model.collections.isEmpty {
                await model.loadCollections()
            }
        }
    }
}

private struct CollectionDetailView: View {
    @EnvironmentObject private var model: AppModel
    let collection: Collection?

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Collection Detail")
                .font(.system(size: 24, weight: .bold))

            if let collection {
                Text(collection.title)
                    .font(.system(size: 22, weight: .bold, design: .serif))

                if let description = collection.description, !description.isEmpty {
                    Text(description)
                        .foregroundStyle(.secondary)
                }

                Text("\(collection.items?.count ?? 0) saved thoughts")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(.secondary)

                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        ForEach(collection.items ?? []) { item in
                            if let thought = item.thought {
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
                                    .padding(14)
                                    .background(Color.white.opacity(0.62), in: RoundedRectangle(cornerRadius: 16))
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            } else {
                Text("Pick a collection to inspect its saved thoughts.")
                    .foregroundStyle(.secondary)
            }

            Spacer()
        }
        .padding(24)
        .background(Color.white.opacity(0.55), in: RoundedRectangle(cornerRadius: 24))
    }
}
