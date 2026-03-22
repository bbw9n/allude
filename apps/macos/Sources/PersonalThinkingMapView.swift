import SwiftUI

struct PersonalThinkingMapView: View {
    @EnvironmentObject private var model: AppModel
    @State private var selectedConceptID: String?

    private var selectedConceptNode: ThinkingMapConceptNode? {
        let fallback = model.personalThinkingMap.concepts.first
        guard let selectedConceptID else { return fallback }
        return model.personalThinkingMap.concepts.first(where: { $0.id == selectedConceptID }) ?? fallback
    }

    private var relatedThoughts: [Thought] {
        guard let selectedConceptNode else { return [] }
        let thoughtIDSet = Set(selectedConceptNode.thoughtIDs)
        return model.personalMapThoughts.filter { thoughtIDSet.contains($0.id) }
    }

    var body: some View {
        HStack(spacing: 24) {
            VStack(alignment: .leading, spacing: 18) {
                HStack {
                    Text("Personal Thinking Map")
                        .font(.system(size: 34, weight: .bold, design: .serif))

                    if model.isRefreshingThinkingMap {
                        ProgressView()
                    }
                }

                Text("A live map of the concepts most present in your current thought history.")
                    .font(.system(size: 14))
                    .foregroundStyle(.secondary)

                if model.personalThinkingMap.concepts.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("No map yet")
                            .font(.system(size: 20, weight: .semibold))
                        Text("Write a few thoughts and Allude will begin to cluster your recurring ideas here.")
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 24))
                } else {
                    ThinkingMapCanvas(
                        map: model.personalThinkingMap,
                        selectedConceptID: selectedConceptID,
                        onSelectConcept: { conceptID in
                            selectedConceptID = conceptID
                        }
                    )
                }
            }

            VStack(alignment: .leading, spacing: 16) {
                Text("Concept focus")
                    .font(.system(size: 24, weight: .bold))

                if let selectedConceptNode {
                    VStack(alignment: .leading, spacing: 10) {
                        Text(selectedConceptNode.name.capitalized)
                            .font(.system(size: 22, weight: .bold, design: .serif))

                        HStack(spacing: 10) {
                            MapMetricPill(label: "Thoughts", value: "\(selectedConceptNode.count)")
                            MapMetricPill(label: "Links", value: "\(connectedEdgeCount(for: selectedConceptNode.id))")
                        }

                        Button {
                            Task {
                                await model.loadConcept(id: selectedConceptNode.id)
                                model.selectedSection = .concept
                            }
                        } label: {
                            Label("Open Concept Page", systemImage: "point.topleft.down.curvedto.point.bottomright.up")
                        }
                        .buttonStyle(.bordered)
                    }
                    .padding(18)
                    .background(Color.white.opacity(0.68), in: RoundedRectangle(cornerRadius: 20))

                    VStack(alignment: .leading, spacing: 12) {
                        Text("Thoughts in this cluster")
                            .font(.system(size: 18, weight: .bold))

                        if relatedThoughts.isEmpty {
                            Text("No mapped thoughts yet.")
                                .foregroundStyle(.secondary)
                        } else {
                            ScrollView {
                                VStack(alignment: .leading, spacing: 12) {
                                    ForEach(relatedThoughts) { thought in
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
                                            .background(Color.white.opacity(0.58), in: RoundedRectangle(cornerRadius: 16))
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                        }
                    }
                } else {
                    Text("Pick a concept node to inspect how your thoughts cluster around it.")
                        .foregroundStyle(.secondary)
                }

                Spacer()
            }
            .frame(width: 380)
        }
        .padding(32)
        .task {
            if model.personalThinkingMap.concepts.isEmpty {
                await model.loadPersonalThinkingMap()
                selectedConceptID = model.personalThinkingMap.concepts.first?.id
            }
        }
        .onChange(of: model.personalThinkingMap.concepts) { _, concepts in
            if selectedConceptID == nil {
                selectedConceptID = concepts.first?.id
            }
        }
    }

    private func connectedEdgeCount(for conceptID: String) -> Int {
        model.personalThinkingMap.edges.filter { $0.sourceID == conceptID || $0.targetID == conceptID }.count
    }
}

private struct ThinkingMapCanvas: View {
    let map: PersonalThinkingMap
    let selectedConceptID: String?
    let onSelectConcept: (String) -> Void

    var body: some View {
        GeometryReader { proxy in
            Canvas { context, size in
                let center = CGPoint(x: size.width / 2, y: size.height / 2)
                let positions = Dictionary(uniqueKeysWithValues: map.concepts.map { concept in
                    (concept.id, CGPoint(x: center.x + concept.x, y: center.y + concept.y))
                })

                for edge in map.edges {
                    guard let source = positions[edge.sourceID], let target = positions[edge.targetID] else { continue }
                    var path = Path()
                    path.move(to: source)
                    path.addLine(to: target)
                    context.stroke(path, with: .color(Color.black.opacity(min(0.32, 0.08 + edge.weight * 0.04))), lineWidth: 1.2 + edge.weight * 0.35)
                }

                for concept in map.concepts {
                    guard let point = positions[concept.id] else { continue }
                    let isSelected = selectedConceptID == concept.id
                    let width = CGFloat(90 + min(concept.count, 6) * 8)
                    let rect = CGRect(x: point.x - width / 2, y: point.y - 28, width: width, height: 56)
                    let path = RoundedRectangle(cornerRadius: 18, style: .continuous).path(in: rect)
                    context.fill(path, with: .color(isSelected ? Color.black.opacity(0.82) : Color.white.opacity(0.85)))
                    context.stroke(path, with: .color(Color.black.opacity(0.18)), lineWidth: 1)

                    var text = context.resolve(
                        Text(concept.name)
                            .font(.system(size: 11, weight: .semibold))
                    )
                    text.shading = .color(isSelected ? .white : .black)
                    context.draw(text, in: rect.insetBy(dx: 8, dy: 8))
                }
            }
            .background(
                RoundedRectangle(cornerRadius: 28, style: .continuous)
                    .fill(Color.white.opacity(0.45))
            )
            .overlay {
                ForEach(map.concepts) { concept in
                    Button {
                        onSelectConcept(concept.id)
                    } label: {
                        Color.clear
                    }
                    .buttonStyle(.plain)
                    .frame(width: 140, height: 72)
                    .position(x: proxy.size.width / 2 + concept.x, y: proxy.size.height / 2 + concept.y)
                }
            }
        }
    }
}

private struct MapMetricPill: View {
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
