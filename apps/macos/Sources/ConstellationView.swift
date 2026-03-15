import SwiftUI

struct ConstellationView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        HStack(spacing: 24) {
            VStack(alignment: .leading, spacing: 18) {
                HStack {
                    Text("Idea Constellation")
                        .font(.system(size: 34, weight: .bold, design: .serif))

                    if model.isRefreshingGraph {
                        ProgressView()
                    }
                }

                if let graph = model.graph {
                    GraphCanvasView(graph: graph) { node in
                        model.selectThought(node.thought)
                        model.selectedSection = .constellation
                    }
                } else {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("No constellation yet")
                            .font(.system(size: 20, weight: .semibold))
                        Text("Create or select a thought, then the graph will recenter around it.")
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(Color.white.opacity(0.5), in: RoundedRectangle(cornerRadius: 24))
                }
            }

            ThoughtDetailView(thought: model.selectedThought)
                .frame(width: 380)
        }
        .padding(32)
        .task(id: model.selectedThought?.id) {
            if let thought = model.selectedThought {
                await model.loadGraph(centerThoughtId: thought.id)
            }
        }
    }
}

struct GraphCanvasView: View {
    let graph: GraphNeighborhood
    let onSelect: (GraphNode) -> Void

    var body: some View {
        GeometryReader { proxy in
            Canvas { context, size in
                let centerPoint = CGPoint(x: size.width / 2, y: size.height / 2)
                let pointMap = Dictionary(uniqueKeysWithValues: graph.nodes.map { node in
                    (node.id, CGPoint(x: centerPoint.x + node.x, y: centerPoint.y + node.y))
                })

                for edge in graph.edges {
                    if let source = pointMap[edge.link.sourceThoughtId], let target = pointMap[edge.link.targetThoughtId] {
                        var path = Path()
                        path.move(to: source)
                        path.addLine(to: target)
                        context.stroke(path, with: .color(Color.black.opacity(0.15)), lineWidth: 1.4)
                    }
                }

                for node in graph.nodes {
                    let point = pointMap[node.id] ?? centerPoint
                    let rect = CGRect(x: point.x - 54, y: point.y - 28, width: 108, height: 56)
                    let shape = RoundedRectangle(cornerRadius: 18, style: .continuous).path(in: rect)
                    let fill = node.distance == 0 ? Color(red: 0.18, green: 0.22, blue: 0.30) : Color.white.opacity(0.82)
                    let label = String(node.thought.currentVersion.content.prefix(36))
                    context.fill(shape, with: .color(fill))
                    context.stroke(shape, with: .color(Color.black.opacity(0.2)), lineWidth: 1)
                    var text = context.resolve(
                        Text(label)
                            .font(.system(size: 11, weight: .semibold))
                    )
                    text.shading = .color(node.distance == 0 ? .white : .black)
                    context.draw(text, in: rect.insetBy(dx: 6, dy: 6))
                }
            }
            .background(
                RoundedRectangle(cornerRadius: 28, style: .continuous)
                    .fill(Color.white.opacity(0.45))
            )
            .overlay {
                ForEach(graph.nodes) { node in
                    let point = CGPoint(
                        x: proxy.size.width / 2 + node.x,
                        y: proxy.size.height / 2 + node.y
                    )
                    Button {
                        onSelect(node)
                    } label: {
                        Color.clear
                    }
                    .buttonStyle(.plain)
                    .frame(width: 120, height: 64)
                    .position(point)
                }
            }
        }
    }
}
