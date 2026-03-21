import SwiftUI

struct ThoughtDetailView: View {
    @EnvironmentObject private var model: AppModel
    let thought: Thought?

    @State private var selectedVersionID: String?

    private var sortedVersions: [ThoughtVersion] {
        guard let thought, let versions = thought.versions, !versions.isEmpty else { return [] }
        return versions.sorted { $0.version > $1.version }
    }

    private var selectedVersion: ThoughtVersion? {
        guard let selectedVersionID else { return sortedVersions.first }
        return sortedVersions.first(where: { $0.id == selectedVersionID }) ?? sortedVersions.first
    }

    private var previousVersion: ThoughtVersion? {
        guard let selectedVersion else { return nil }
        return sortedVersions
            .filter { $0.version < selectedVersion.version }
            .sorted { $0.version > $1.version }
            .first
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Thought Evolution")
                .font(.system(size: 24, weight: .bold))

            if let thought {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        ThoughtSummaryCard(thought: thought)

                        versionPicker

                        if let selectedVersion {
                            versionCard(
                                title: selectedVersion.id == thought.currentVersion.id ? "Current text" : "Selected version",
                                subtitle: "v\(selectedVersion.version) • \(selectedVersion.createdAt)",
                                content: selectedVersion.content,
                                emphasized: selectedVersion.id == thought.currentVersion.id
                            )
                        }

                        revisionDiffSection

                        conceptSection(thought: thought)
                        relatedIdeasSection(thought: thought)
                        timelineSection(thought: thought)
                    }
                }
            } else {
                Text("Select a thought to inspect versions, concepts, and nearby links.")
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 0)
        }
        .padding(24)
        .background(Color.white.opacity(0.55), in: RoundedRectangle(cornerRadius: 24))
        .onAppear {
            selectedVersionID = sortedVersions.first?.id
        }
        .onChange(of: thought?.id) { _, _ in
            selectedVersionID = sortedVersions.first?.id
        }
    }

    @ViewBuilder
    private var versionPicker: some View {
        if !sortedVersions.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text("Version focus")
                    .font(.system(size: 13, weight: .semibold))

                Picker("Version focus", selection: Binding(
                    get: { selectedVersionID ?? sortedVersions.first?.id ?? "" },
                    set: { selectedVersionID = $0 }
                )) {
                    ForEach(sortedVersions) { version in
                        Text("v\(version.version)")
                            .tag(version.id)
                    }
                }
                .pickerStyle(.segmented)
            }
        }
    }

    @ViewBuilder
    private var revisionDiffSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Revision diff")
                .font(.system(size: 13, weight: .semibold))

            if let selectedVersion, let previousVersion {
                let diff = RevisionDiff.compare(from: previousVersion.content, to: selectedVersion.content)

                VStack(alignment: .leading, spacing: 12) {
                    Text("v\(previousVersion.version) → v\(selectedVersion.version)")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(.secondary)

                    if diff.added.isEmpty && diff.removed.isEmpty {
                        Text("No material line-level change detected between these versions.")
                            .font(.system(size: 12))
                            .foregroundStyle(.secondary)
                    } else {
                        if !diff.added.isEmpty {
                            DiffBucketView(title: "Added", items: diff.added, tint: Color.green.opacity(0.18))
                        }
                        if !diff.removed.isEmpty {
                            DiffBucketView(title: "Removed", items: diff.removed, tint: Color.red.opacity(0.16))
                        }
                    }
                }
                .padding(14)
                .background(Color.white.opacity(0.56), in: RoundedRectangle(cornerRadius: 16))
            } else {
                Text("Save another revision to see how this thought evolves over time.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .padding(14)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 16))
            }
        }
    }

    @ViewBuilder
    private func conceptSection(thought: Thought) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Concept links")
                .font(.system(size: 13, weight: .semibold))

            if thought.concepts.isEmpty {
                Text("Concept nodes will appear here after enrichment finishes.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            } else {
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
            }
        }
    }

    @ViewBuilder
    private func relatedIdeasSection(thought: Thought) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Related ideas")
                .font(.system(size: 13, weight: .semibold))

            if let relatedThoughts = thought.relatedThoughts, !relatedThoughts.isEmpty {
                ForEach(relatedThoughts) { related in
                    Button {
                        model.selectThought(related)
                    } label: {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(related.currentVersion.content)
                                .font(.system(size: 12, weight: .medium))
                                .frame(maxWidth: .infinity, alignment: .leading)
                            if !related.concepts.isEmpty {
                                FlowLayout(items: Array(related.concepts.prefix(3)).map(\.name))
                            }
                        }
                        .padding(12)
                        .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
                    }
                    .buttonStyle(.plain)
                }
            } else {
                Text("Linked neighboring thoughts will appear here as the graph fills in.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            }
        }
    }

    @ViewBuilder
    private func timelineSection(thought: Thought) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Timeline")
                .font(.system(size: 13, weight: .semibold))

            ForEach(sortedVersions) { version in
                Button {
                    selectedVersionID = version.id
                } label: {
                    HStack(alignment: .top, spacing: 12) {
                        Circle()
                            .fill(version.id == selectedVersionID ? Color.black : Color.black.opacity(0.25))
                            .frame(width: 10, height: 10)
                            .padding(.top, 4)

                        VStack(alignment: .leading, spacing: 4) {
                            Text("v\(version.version) • \(version.createdAt)")
                                .font(.system(size: 12, weight: .bold))
                            Text(version.content)
                                .font(.system(size: 12))
                                .foregroundStyle(.secondary)
                                .lineLimit(version.id == selectedVersionID ? 6 : 2)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(12)
                    .background(
                        (version.id == selectedVersionID ? Color.black.opacity(0.08) : Color.white.opacity(0.45)),
                        in: RoundedRectangle(cornerRadius: 14)
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }

    @ViewBuilder
    private func versionCard(title: String, subtitle: String, content: String, emphasized: Bool) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(size: 13, weight: .semibold))
            Text(subtitle)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(.secondary)
            Text(content)
                .font(.system(size: 15, weight: .regular, design: .serif))
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(16)
        .background(
            emphasized ? Color.white.opacity(0.74) : Color.white.opacity(0.54),
            in: RoundedRectangle(cornerRadius: 18)
        )
    }
}

private struct ThoughtSummaryCard: View {
    let thought: Thought

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(thought.currentVersion.content)
                .font(.system(size: 16, weight: .regular, design: .serif))

            HStack(spacing: 10) {
                SummaryPill(label: "Current", value: "v\(thought.currentVersion.version)")
                SummaryPill(label: "Versions", value: "\(thought.versions?.count ?? 1)")
                SummaryPill(label: "Concepts", value: "\(thought.concepts.count)")
                SummaryPill(label: "Links", value: "\(thought.relatedThoughts?.count ?? 0)")
            }
        }
        .padding(18)
        .background(Color.white.opacity(0.72), in: RoundedRectangle(cornerRadius: 20))
    }
}

private struct SummaryPill: View {
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

private struct DiffBucketView: View {
    let title: String
    let items: [String]
    let tint: Color

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(size: 12, weight: .bold))
            ForEach(items, id: \.self) { item in
                Text(item)
                    .font(.system(size: 12))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(10)
                    .background(tint, in: RoundedRectangle(cornerRadius: 10))
            }
        }
    }
}

private struct RevisionDiff {
    let added: [String]
    let removed: [String]

    static func compare(from old: String, to new: String) -> RevisionDiff {
        let oldSegments = segments(in: old)
        let newSegments = segments(in: new)
        let oldNormalized = Set(oldSegments.map(normalize))
        let newNormalized = Set(newSegments.map(normalize))

        let added = newSegments.filter { !oldNormalized.contains(normalize($0)) }
        let removed = oldSegments.filter { !newNormalized.contains(normalize($0)) }
        return RevisionDiff(added: added, removed: removed)
    }

    private static func segments(in value: String) -> [String] {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return [] }

        let raw = trimmed
            .replacingOccurrences(of: "\n", with: ". ")
            .components(separatedBy: ". ")
            .flatMap { segment in
                segment.components(separatedBy: "\n")
            }

        return raw
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }

    private static func normalize(_ value: String) -> String {
        value
            .lowercased()
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
