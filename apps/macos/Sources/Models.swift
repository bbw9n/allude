import Foundation

struct User: Decodable, Hashable, Sendable {
    let id: String
    let username: String
    let bio: String?
    let interests: [String]
}

struct ThoughtVersion: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let thoughtId: String
    let version: Int
    let content: String
    let createdAt: String
}

struct Concept: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let name: String
    let slug: String?
    let description: String?
    let conceptType: String?
    let thoughtCount: Int?
    let aliases: [ConceptAlias]?
    let createdAt: String
    let relatedConcepts: [Concept]?
    let topThoughts: [Thought]?
    let contradictionThoughts: [Thought]?
}

struct ConceptAlias: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let conceptId: String
    let alias: String
    let normalizedAlias: String
}

struct ThoughtLink: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let sourceThoughtId: String
    let targetThoughtId: String
    let relationType: String
    let score: Double
    let origin: String
    let createdAt: String
}

struct Thought: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let author: User?
    let currentVersion: ThoughtVersion
    let versions: [ThoughtVersion]?
    let concepts: [Concept]
    let relatedThoughts: [Thought]?
    let links: [ThoughtLink]?
    let collections: [Collection]?
    let processingStatus: String
    let processingNotes: [String]
    let createdAt: String
    let updatedAt: String
}

struct CollectionItem: Decodable, Hashable, Identifiable, Sendable {
    let collectionId: String
    let thoughtId: String
    let position: Int?
    let addedAt: String?
    let thought: Thought?

    var id: String { "\(collectionId):\(thoughtId)" }
}

struct Collection: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let curatorId: String
    let title: String
    let description: String?
    let visibility: String?
    let items: [CollectionItem]?
    let createdAt: String
    let updatedAt: String
}

struct CaptureItem: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let authorId: String
    let content: String
    let sourceType: String?
    let sourceTitle: String?
    let sourceUrl: String?
    let sourceApp: String?
    let status: String
    let promotedThoughtId: String?
    let promotedThought: Thought?
    let createdAt: String
    let updatedAt: String
}

struct IdeaCurrent: Decodable, Hashable, Identifiable, Sendable {
    let id: String
    let title: String
    let summary: String?
    let clusterKey: String?
    let freshnessScore: Double
    let qualityScore: Double
    let concepts: [Concept]?
    let thoughts: [Thought]?
    let createdAt: String
    let updatedAt: String
}

struct SearchCluster: Decodable, Hashable, Sendable {
    let label: String
    let concepts: [Concept]
    let thoughtIds: [String]
}

struct SearchThoughtsResult: Decodable, Hashable, Sendable {
    let thoughts: [Thought]
    let clusters: [SearchCluster]
}

struct DraftSuggestions: Decodable, Hashable, Sendable {
    let relatedConcepts: [String]
    let supportingThoughts: [Thought]
    let counterThoughts: [Thought]
    let reframes: [String]
    let notes: [String]

    static let empty = DraftSuggestions(
        relatedConcepts: [],
        supportingThoughts: [],
        counterThoughts: [],
        reframes: [],
        notes: []
    )
}

struct GraphNode: Decodable, Hashable, Identifiable, Sendable {
    let thought: Thought
    let x: Double
    let y: Double
    let distance: Int

    var id: String { thought.id }
}

struct GraphEdge: Decodable, Hashable, Identifiable, Sendable {
    let link: ThoughtLink

    var id: String { link.id }
}

struct GraphNeighborhood: Decodable, Hashable, Sendable {
    let center: GraphNode
    let nodes: [GraphNode]
    let edges: [GraphEdge]
}

struct ThinkingMapConceptNode: Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let count: Int
    let x: Double
    let y: Double
    let thoughtIDs: [String]
}

struct ThinkingMapEdge: Identifiable, Hashable, Sendable {
    let id: String
    let sourceID: String
    let targetID: String
    let weight: Double
}

struct PersonalThinkingMap: Hashable, Sendable {
    let concepts: [ThinkingMapConceptNode]
    let edges: [ThinkingMapEdge]
}

struct TelescopeJump: Decodable, Hashable, Sendable {
    let label: String
    let query: String
    let reason: String?
    let thoughtIds: [String]?
}

struct TelescopeResult: Decodable, Hashable, Sendable {
    let query: String
    let intent: String
    let seedConcepts: [Concept]
    let seedThoughts: [Thought]
    let graph: GraphNeighborhood?
    let clusters: [SearchCluster]
    let narrative: String
    let suggestedJumps: [TelescopeJump]
    let relatedCurrents: [IdeaCurrent]

    static let empty = TelescopeResult(
        query: "",
        intent: "explore",
        seedConcepts: [],
        seedThoughts: [],
        graph: nil,
        clusters: [],
        narrative: "Search across your idea graph to surface concept clusters, related thoughts, and promising next jumps.",
        suggestedJumps: [],
        relatedCurrents: []
    )
}

enum SidebarSection: String, CaseIterable, Identifiable, Sendable {
    case composer = "Composer"
    case inbox = "Inbox"
    case telescope = "Telescope"
    case constellation = "Constellation"
    case concept = "Concept"
    case collections = "Collections"
    case map = "Thinking Map"

    var id: String { rawValue }
}
