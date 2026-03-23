import Combine
import Foundation
import SwiftUI

@MainActor
final class AppModel: ObservableObject {
    @Published var selectedSection: SidebarSection = .composer
    @Published var thoughts: [Thought] = []
    @Published var selectedThought: Thought?
    @Published var selectedConcept: Concept?
    @Published var captures: [CaptureItem] = []
    @Published var selectedCapture: CaptureItem?
    @Published var collections: [Collection] = []
    @Published var selectedCollection: Collection?
    @Published var personalMapThoughts: [Thought] = []
    @Published var personalThinkingMap = PersonalThinkingMap(concepts: [], edges: [])
    @Published var graph: GraphNeighborhood?
    @Published var telescopeQuery = ""
    @Published var telescopeResult = TelescopeResult.empty
    @Published var searchResult = SearchThoughtsResult(thoughts: [], clusters: [])
    @Published var selectedSearchClusterLabel: String?
    @Published var draftSuggestions = DraftSuggestions.empty
    @Published var draft = ""
    @Published var captureDraft = ""
    @Published var captureSourceTitle = ""
    @Published var captureSourceURL = ""
    @Published var captureSourceApp = ""
    @Published var newCollectionTitle = ""
    @Published var newCollectionDescription = ""
    @Published var isSaving = false
    @Published var isSearching = false
    @Published var isRefreshingGraph = false
    @Published var isRefreshingDraftSuggestions = false
    @Published var isRefreshingCollections = false
    @Published var isRefreshingInbox = false
    @Published var isSavingCollection = false
    @Published var isSavingCapture = false
    @Published var isRefreshingThinkingMap = false
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
                query: AlludeAPI.telescope,
                variables: ["query": trimmed],
                data: TelescopeData.self
            )
            telescopeResult = data.telescope
            searchResult = SearchThoughtsResult(
                thoughts: data.telescope.seedThoughts,
                clusters: data.telescope.clusters
            )
            thoughts = mergeThoughts(data.telescope.seedThoughts, into: thoughts)
            graph = data.telescope.graph
            selectedSearchClusterLabel = data.telescope.clusters.first?.label
            if let first = filteredSearchThoughts.first {
                selectedThought = first
                selectedSection = .telescope
            } else {
                selectedThought = nil
            }
            if let firstConcept = data.telescope.seedConcepts.first {
                selectedConcept = firstConcept
                await loadConcept(id: firstConcept.id)
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

    func loadInbox() async {
        isRefreshingInbox = true
        defer { isRefreshingInbox = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.inbox,
                variables: ["limit": 50],
                data: InboxData.self
            )
            captures = data.inbox
            if selectedCapture == nil {
                selectedCapture = data.inbox.first
            } else if let selectedCapture {
                self.selectedCapture = data.inbox.first(where: { $0.id == selectedCapture.id }) ?? data.inbox.first
            }
        } catch {
            errorMessage = "Unable to load inbox."
        }
    }

    func createCapture() async {
        let content = captureDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }

        isSavingCapture = true
        defer { isSavingCapture = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.createCapture,
                variables: [
                    "content": AnyEncodable(content),
                    "sourceType": AnyEncodable(inferredCaptureSourceType(from: content)),
                    "sourceTitle": AnyEncodable(captureSourceTitle.isEmpty ? nil as String? : captureSourceTitle),
                    "sourceUrl": AnyEncodable(captureSourceURL.isEmpty ? nil as String? : captureSourceURL),
                    "sourceApp": AnyEncodable(captureSourceApp.isEmpty ? nil as String? : captureSourceApp)
                ],
                data: CreateCapturePayload.self
            )
            captureDraft = ""
            captureSourceTitle = ""
            captureSourceURL = ""
            captureSourceApp = ""
            selectedCapture = data.createCapture
            await loadInbox()
        } catch {
            errorMessage = "Unable to save capture."
        }
    }

    func archiveSelectedCapture() async {
        guard let selectedCapture else { return }
        await archiveCapture(selectedCapture)
    }

    func archiveCapture(_ capture: CaptureItem) async {
        isSavingCapture = true
        defer { isSavingCapture = false }

        do {
            _ = try await client.send(
                query: AlludeAPI.archiveCapture,
                variables: ["captureId": capture.id],
                data: ArchiveCapturePayload.self
            )
            if self.selectedCapture?.id == capture.id {
                self.selectedCapture = nil
            }
            await loadInbox()
        } catch {
            errorMessage = "Unable to archive capture."
        }
    }

    func promoteSelectedCapture() async {
        guard let selectedCapture else { return }
        await promoteCapture(selectedCapture)
    }

    func promoteCapture(_ capture: CaptureItem) async {
        isSavingCapture = true
        defer { isSavingCapture = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.promoteCapture,
                variables: ["captureId": capture.id],
                data: PromoteCapturePayload.self
            )
            selectedCapture = data.promoteCapture
            if let thought = data.promoteCapture.promotedThought {
                selectedThought = thought
                thoughts = mergeThoughts([thought], into: thoughts)
                selectedSection = .constellation
                await refreshSelectedThought()
            } else if let thoughtID = data.promoteCapture.promotedThoughtId {
                let thoughtData = try await client.send(
                    query: AlludeAPI.thought,
                    variables: ["id": thoughtID],
                    data: ThoughtData.self
                )
                if let thought = thoughtData.thought {
                    selectedThought = thought
                    thoughts = mergeThoughts([thought], into: thoughts)
                    selectedSection = .constellation
                    await refreshSelectedThought()
                }
            }
            await loadInbox()
        } catch {
            errorMessage = "Unable to promote capture."
        }
    }

    func selectCapture(_ capture: CaptureItem) {
        selectedCapture = capture
        selectedSection = .inbox
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

    func loadPersonalThinkingMap() async {
        isRefreshingThinkingMap = true
        defer { isRefreshingThinkingMap = false }

        do {
            let data = try await client.send(
                query: AlludeAPI.myThoughts,
                variables: ["limit": 50],
                data: MyThoughtsData.self
            )
            personalMapThoughts = data.myThoughts
            thoughts = mergeThoughts(data.myThoughts, into: thoughts)
            personalThinkingMap = buildThinkingMap(from: data.myThoughts)
        } catch {
            errorMessage = "Unable to load your thinking map."
        }
    }

    func selectThought(_ thought: Thought) {
        selectedThought = thought
        draft = thought.currentVersion.content
        scheduleDraftSuggestions()
        Task {
            await refreshSelectedThought()
        }
    }

    func selectSearchCluster(label: String?) {
        selectedSearchClusterLabel = label
        if let filteredFirst = filteredSearchThoughts.first {
            selectedThought = filteredFirst
        } else {
            selectedThought = telescopeResult.seedThoughts.first ?? searchResult.thoughts.first
        }
    }

    func applyTelescopeSuggestion(_ query: String) {
        telescopeQuery = query
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

    private func mergeThoughts(_ incoming: [Thought], into existing: [Thought]) -> [Thought] {
        var merged = existing
        for thought in incoming {
            merged = upserting(thought: thought, into: merged)
        }
        return merged
    }

    private func buildThinkingMap(from thoughts: [Thought]) -> PersonalThinkingMap {
        struct ConceptAccumulator {
            var count: Int
            var thoughtIDs: Set<String>
        }

        var conceptStats: [String: ConceptAccumulator] = [:]
        var conceptNames: [String: String] = [:]
        var pairWeights: [String: Double] = [:]

        for thought in thoughts {
            let conceptList = thought.concepts
            for concept in conceptList {
                conceptNames[concept.id] = concept.name
                var accumulator = conceptStats[concept.id] ?? ConceptAccumulator(count: 0, thoughtIDs: [])
                accumulator.count += 1
                accumulator.thoughtIDs.insert(thought.id)
                conceptStats[concept.id] = accumulator
            }

            let conceptIDs = Array(Set(conceptList.map(\.id))).sorted()
            for leftIndex in conceptIDs.indices {
                for rightIndex in conceptIDs.indices where rightIndex > leftIndex {
                    let left = conceptIDs[leftIndex]
                    let right = conceptIDs[rightIndex]
                    let key = "\(left)|\(right)"
                    pairWeights[key, default: 0] += 1
                }
            }
        }

        let sortedConceptIDs = conceptStats.keys.sorted {
            let left = conceptStats[$0]?.count ?? 0
            let right = conceptStats[$1]?.count ?? 0
            if left == right {
                return (conceptNames[$0] ?? "") < (conceptNames[$1] ?? "")
            }
            return left > right
        }

        let limitedIDs = Array(sortedConceptIDs.prefix(14))
        let radius = 180.0
        let concepts = limitedIDs.enumerated().map { index, id in
            let angle = Double(index) / Double(max(limitedIDs.count, 1)) * .pi * 2
            let count = conceptStats[id]?.count ?? 0
            return ThinkingMapConceptNode(
                id: id,
                name: conceptNames[id] ?? id,
                count: count,
                x: cos(angle) * radius,
                y: sin(angle) * radius,
                thoughtIDs: Array(conceptStats[id]?.thoughtIDs ?? [])
            )
        }

        let conceptSet = Set(limitedIDs)
        let edges = pairWeights.compactMap { key, weight -> ThinkingMapEdge? in
            let parts = key.components(separatedBy: "|")
            guard parts.count == 2 else { return nil }
            guard conceptSet.contains(parts[0]), conceptSet.contains(parts[1]) else { return nil }
            return ThinkingMapEdge(id: key, sourceID: parts[0], targetID: parts[1], weight: weight)
        }
        .sorted { $0.weight > $1.weight }

        return PersonalThinkingMap(concepts: concepts, edges: Array(edges.prefix(28)))
    }

    var filteredSearchThoughts: [Thought] {
        guard let selectedSearchClusterLabel,
              let cluster = telescopeResult.clusters.first(where: { $0.label == selectedSearchClusterLabel }) ?? searchResult.clusters.first(where: { $0.label == selectedSearchClusterLabel })
        else {
            if !telescopeResult.seedThoughts.isEmpty {
                return telescopeResult.seedThoughts
            }
            return searchResult.thoughts
        }

        let clusterThoughtIDs = Set(cluster.thoughtIds)
        let telescopeFiltered = telescopeResult.seedThoughts.filter { clusterThoughtIDs.contains($0.id) }
        if !telescopeFiltered.isEmpty {
            return telescopeFiltered
        }
        let fallbackFiltered = searchResult.thoughts.filter { clusterThoughtIDs.contains($0.id) }
        if !fallbackFiltered.isEmpty {
            return fallbackFiltered
        }
        return telescopeResult.seedThoughts.isEmpty ? searchResult.thoughts : telescopeResult.seedThoughts
    }

    var telescopeNarrative: String {
        if !telescopeResult.narrative.isEmpty, telescopeResult.query == telescopeQuery.trimmingCharacters(in: .whitespacesAndNewlines) {
            return telescopeResult.narrative
        }
        return TelescopeResult.empty.narrative
    }

    var telescopeSuggestedJumps: [TelescopeJump] {
        if !telescopeResult.suggestedJumps.isEmpty {
            return telescopeResult.suggestedJumps
        }
        let query = telescopeQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        if query.isEmpty {
            return [
                TelescopeJump(label: "Bridge philosophy and startups", query: "connections between stoicism and startup culture", reason: "Explore a concrete example of cross-domain idea search.", thoughtIds: []),
                TelescopeJump(label: "Boredom and creativity", query: "ideas related to boredom and creativity", reason: "Follow a familiar conceptual tension.", thoughtIds: []),
                TelescopeJump(label: "Disagreement on stoicism", query: "thinkers who disagree with stoicism", reason: "See how contradiction flows through the graph.", thoughtIds: [])
            ]
        }
        return [
            TelescopeJump(label: "Find contradictions", query: "contradictions in \(query)", reason: "Surface objections and edge cases.", thoughtIds: []),
            TelescopeJump(label: "Find examples", query: "examples of \(query)", reason: "Turn the cluster into concrete cases.", thoughtIds: []),
            TelescopeJump(label: "Explore adjacency", query: "adjacent ideas to \(query)", reason: "Branch into nearby concepts.", thoughtIds: [])
        ]
    }

    var telescopeClusters: [SearchCluster] {
        telescopeResult.clusters.isEmpty ? searchResult.clusters : telescopeResult.clusters
    }

    var telescopeRelatedCurrents: [IdeaCurrent] {
        telescopeResult.relatedCurrents
    }

    var telescopeGraph: GraphNeighborhood? {
        telescopeResult.graph ?? graph
    }

    private func inferredCaptureSourceType(from content: String) -> String {
        if content.contains("http://") || content.contains("https://") {
            return "link"
        }
        if content.contains("\"") || content.contains("“") {
            return "quote"
        }
        return "note"
    }
}
