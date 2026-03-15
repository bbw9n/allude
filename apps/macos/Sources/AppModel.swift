import Foundation
import SwiftUI

@MainActor
final class AppModel: ObservableObject {
    @Published var selectedSection: SidebarSection = .composer
    @Published var thoughts: [Thought] = []
    @Published var selectedThought: Thought?
    @Published var selectedConcept: Concept?
    @Published var graph: GraphNeighborhood?
    @Published var telescopeQuery = ""
    @Published var searchResult = SearchThoughtsResult(thoughts: [], clusters: [])
    @Published var draft = ""
    @Published var isSaving = false
    @Published var isSearching = false
    @Published var isRefreshingGraph = false
    @Published var errorMessage: String?

    private let client = GraphQLClient()

    func saveDraft() async {
        let trimmed = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        isSaving = true
        defer { isSaving = false }

        do {
            if let selectedThought {
                let data = try await client.send(
                    query: AlludeAPI.updateThought,
                    variables: ["thoughtId": selectedThought.id, "content": trimmed],
                    data: UpdateThoughtData.self
                )
                self.selectedThought = data.updateThought
            } else {
                let data = try await client.send(
                    query: AlludeAPI.createThought,
                    variables: ["content": trimmed],
                    data: CreateThoughtData.self
                )
                self.selectedThought = data.createThought
                self.selectedSection = .constellation
            }

            draft = ""
            try? await Task.sleep(for: .milliseconds(150))
            await refreshSelectedThought()
        } catch {
            errorMessage = "Unable to save thought. Start the API at http://127.0.0.1:4000 and try again."
        }
    }

    func search() async {
        let trimmed = telescopeQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        isSearching = true
        defer { isSearching = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.searchThoughts,
                variables: ["query": trimmed],
                data: SearchThoughtsData.self
            )
            searchResult = data.searchThoughts
            thoughts = data.searchThoughts.thoughts
            if let first = data.searchThoughts.thoughts.first {
                selectedThought = first
                selectedSection = .telescope
            }
        } catch {
            errorMessage = "Search failed. Make sure the API is running."
        }
    }

    func refreshSelectedThought() async {
        guard let selectedThought else { return }
        do {
            let data = try await client.send(
                query: AlludeAPI.thought,
                variables: ["id": selectedThought.id],
                data: ThoughtData.self
            )
            if let thought = data.thought {
                self.selectedThought = thought
                self.thoughts = upserting(thought: thought, into: thoughts)
                if let firstConcept = thought.concepts.first {
                    await loadConcept(id: firstConcept.id)
                }
                await loadGraph(centerThoughtId: thought.id)
            }
        } catch {
            errorMessage = "Unable to refresh the current thought."
        }
    }

    func loadGraph(centerThoughtId: String) async {
        isRefreshingGraph = true
        defer { isRefreshingGraph = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.graph,
                variables: ["centerThoughtId": centerThoughtId, "distance": 2, "limit": 12],
                data: GraphData.self
            )
            graph = data.graph
        } catch {
            errorMessage = "Unable to load the idea constellation."
        }
    }

    func loadConcept(id: String) async {
        do {
            let data = try await client.send(
                query: AlludeAPI.concept,
                variables: ["id": id],
                data: ConceptData.self
            )
            selectedConcept = data.concept
        } catch {
            errorMessage = "Unable to load the concept page."
        }
    }

    func selectThought(_ thought: Thought) {
        selectedThought = thought
        draft = thought.currentVersion.content
        Task {
            await refreshSelectedThought()
        }
    }

    func startNewThought() {
        selectedThought = nil
        draft = ""
        selectedSection = .composer
    }

    private func upserting(thought: Thought, into list: [Thought]) -> [Thought] {
        var next = list.filter { $0.id != thought.id }
        next.insert(thought, at: 0)
        return next
    }
}
