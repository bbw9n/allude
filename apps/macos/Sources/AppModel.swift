import Foundation
import SwiftUI

@MainActor
final class AppModel: ObservableObject {
    @Published var selectedSection: SidebarSection = .composer
    @Published var thoughts: [Thought] = []
    @Published var selectedThought: Thought?
    @Published var selectedConcept: Concept?
    @Published var collections: [Collection] = []
    @Published var selectedCollection: Collection?
    @Published var graph: GraphNeighborhood?
    @Published var telescopeQuery = ""
    @Published var searchResult = SearchThoughtsResult(thoughts: [], clusters: [])
    @Published var draftSuggestions = DraftSuggestions.empty
    @Published var draft = ""
    @Published var newCollectionTitle = ""
    @Published var newCollectionDescription = ""
    @Published var isSaving = false
    @Published var isSearching = false
    @Published var isRefreshingGraph = false
    @Published var isRefreshingDraftSuggestions = false
    @Published var isRefreshingCollections = false
    @Published var isSavingCollection = false
    @Published var errorMessage: String?

    private let client = GraphQLClient()
    private var draftSuggestionTask: Task<Void, Never>?

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
            draftSuggestions = .empty
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
                variables: [
                    "centerThoughtId": AnyEncodable(centerThoughtId),
                    "distance": AnyEncodable(2),
                    "limit": AnyEncodable(12)
                ],
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

    func loadCollections() async {
        isRefreshingCollections = true
        defer { isRefreshingCollections = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.collections,
                variables: EmptyVariables(),
                data: CollectionsData.self
            )
            collections = data.collections
            if selectedCollection == nil {
                selectedCollection = data.collections.first
            } else if let selectedCollection {
                self.selectedCollection = data.collections.first(where: { $0.id == selectedCollection.id }) ?? data.collections.first
            }
        } catch {
            errorMessage = "Unable to load collections."
        }
    }

    func createCollection() async {
        let title = newCollectionTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty else { return }

        isSavingCollection = true
        defer { isSavingCollection = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.createCollection,
                variables: [
                    "title": AnyEncodable(title),
                    "description": AnyEncodable(newCollectionDescription.isEmpty ? nil as String? : newCollectionDescription)
                ],
                data: CreateCollectionPayload.self
            )
            newCollectionTitle = ""
            newCollectionDescription = ""
            selectedCollection = data.createCollection
            await loadCollections()
        } catch {
            errorMessage = "Unable to create collection."
        }
    }

    func addSelectedThoughtToCollection(_ collection: Collection) async {
        guard let selectedThought else { return }

        isSavingCollection = true
        defer { isSavingCollection = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.addThoughtToCollection,
                variables: [
                    "collectionId": AnyEncodable(collection.id),
                    "thoughtId": AnyEncodable(selectedThought.id)
                ],
                data: AddThoughtToCollectionPayload.self
            )
            selectedCollection = data.addThoughtToCollection
            collections = collections.map { $0.id == data.addThoughtToCollection.id ? data.addThoughtToCollection : $0 }
            await refreshSelectedThought()
        } catch {
            errorMessage = "Unable to add the thought to this collection."
        }
    }

    func selectCollection(_ collection: Collection) {
        selectedCollection = collection
        selectedSection = .collections
    }

    func selectThought(_ thought: Thought) {
        selectedThought = thought
        draft = thought.currentVersion.content
        scheduleDraftSuggestions()
        Task {
            await refreshSelectedThought()
        }
    }

    func startNewThought() {
        selectedThought = nil
        draft = ""
        draftSuggestions = .empty
        selectedSection = .composer
    }

    func scheduleDraftSuggestions() {
        draftSuggestionTask?.cancel()

        let currentDraft = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard currentDraft.count >= 12 else {
            draftSuggestions = .empty
            isRefreshingDraftSuggestions = false
            return
        }

        let thoughtID = selectedThought?.id
        draftSuggestionTask = Task { @MainActor [client, currentDraft, thoughtID] in
            try? await Task.sleep(for: .milliseconds(350))
            guard !Task.isCancelled else { return }
            isRefreshingDraftSuggestions = true
            do {
                let data = try await client.send(
                    query: AlludeAPI.draftSuggestions,
                    variables: [
                        "content": AnyEncodable(currentDraft),
                        "thoughtId": AnyEncodable(thoughtID)
                    ],
                    data: DraftSuggestionsData.self
                )
                guard !Task.isCancelled else { return }
                draftSuggestions = data.draftSuggestions
                isRefreshingDraftSuggestions = false
            } catch {
                guard !Task.isCancelled else { return }
                draftSuggestions = .empty
                isRefreshingDraftSuggestions = false
            }
        }
    }

    private func upserting(thought: Thought, into list: [Thought]) -> [Thought] {
        var next = list.filter { $0.id != thought.id }
        next.insert(thought, at: 0)
        return next
    }
}
