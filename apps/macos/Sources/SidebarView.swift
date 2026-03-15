import SwiftUI

struct SidebarView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Allude")
                .font(.system(size: 32, weight: .black, design: .serif))
                .padding(.top, 12)

            Text("AI-native idea network")
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(.secondary)

            Button(action: model.startNewThought) {
                Label("New Thought", systemImage: "sparkles.rectangle.stack")
            }
            .buttonStyle(.borderedProminent)

            VStack(alignment: .leading, spacing: 8) {
                ForEach(SidebarSection.allCases) { section in
                    Button {
                        model.selectedSection = section
                    } label: {
                        HStack {
                            Text(section.rawValue)
                            Spacer()
                            if model.selectedSection == section {
                                Image(systemName: "arrow.right.circle.fill")
                            }
                        }
                    }
                    .buttonStyle(.plain)
                    .padding(10)
                    .background(model.selectedSection == section ? Color.black.opacity(0.08) : Color.clear, in: RoundedRectangle(cornerRadius: 12))
                }
            }

            Divider()

            Text("Recent Thoughts")
                .font(.system(size: 13, weight: .semibold))

            if model.thoughts.isEmpty {
                Text("Create a thought, then explore its concept graph and versions here.")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 10) {
                        ForEach(model.thoughts) { thought in
                            Button {
                                model.selectThought(thought)
                            } label: {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(thought.currentVersion.content)
                                        .lineLimit(3)
                                        .font(.system(size: 12, weight: .medium))
                                    Text(thought.processingStatus)
                                        .font(.system(size: 10, weight: .bold))
                                        .foregroundStyle(.secondary)
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(10)
                                .background(Color.white.opacity(0.58), in: RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }

            Spacer()
        }
        .padding(20)
        .frame(minWidth: 280)
        .background(Color.white.opacity(0.4))
    }
}
