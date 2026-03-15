import SwiftUI

struct FlowLayout<Content: View>: View {
    let items: [String]
    let content: (String) -> Content

    init(items: [String], @ViewBuilder content: @escaping (String) -> Content = { item in
        Text(item)
            .font(.system(size: 11, weight: .semibold))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color.black.opacity(0.08), in: Capsule())
    }) {
        self.items = items
        self.content = content
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            let rows = makeRows()
            ForEach(rows.indices, id: \.self) { index in
                HStack {
                    ForEach(rows[index], id: \.self) { item in
                        content(item)
                    }
                    Spacer(minLength: 0)
                }
            }
        }
    }

    private func makeRows() -> [[String]] {
        var rows: [[String]] = [[]]
        var currentCount = 0

        for item in items {
            if currentCount > 0 && currentCount + item.count > 24 {
                rows.append([item])
                currentCount = item.count
            } else {
                rows[rows.count - 1].append(item)
                currentCount += item.count
            }
        }
        return rows.filter { !$0.isEmpty }
    }
}
