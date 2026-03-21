import Foundation

struct User: Decodable, Hashable {
    let id: String
    let username: String
    let bio: String?
    let interests: [String]
}

struct ThoughtVersion: Decodable, Hashable, Identifiable {
    let id: String
    let thoughtId: String
    let version: Int
    let content: String
    let createdAt: String
}

struct Concept: Decodable, Hashable, Identifiable {
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

struct ConceptAlias: Decodable, Hashable, Identifiable {
    let id: String
    let conceptId: String
    let alias: String
    let normalizedAlias: String
}

struct ThoughtLink: Decodable, Hashable, Identifiable {
    let id: String
    let sourceThoughtId: String
    let targetThoughtId: String
    let relationType: String
    let score: Double
    let origin: String
    let createdAt: String
}

struct Thought: Decodable, Hashable, Identifiable {
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

struct CollectionItem: Decodable, Hashable, Identifiable {
    let collectionId: String
    let thoughtId: String
    let position: Int?
    let addedAt: String?
    let thought: Thought?

    var id: String { "\(collectionId):\(thoughtId)" }
}

struct Collection: Decodable, Hashable, Identifiable {
    let id: String
    let curatorId: String
    let title: String
    let description: String?
    let visibility: String?
    let items: [CollectionItem]?
    let createdAt: String
    let updatedAt: String
}

struct SearchCluster: Decodable, Hashable {
    let label: String
    let concepts: [Concept]
    let thoughtIds: [String]
}

struct SearchThoughtsResult: Decodable, Hashable {
    let thoughts: [Thought]
    let clusters: [SearchCluster]
}

struct DraftSuggestions: Decodable, Hashable {
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

struct GraphNode: Decodable, Hashable, Identifiable {
    let thought: Thought
    let x: Double
    let y: Double
    let distance: Int

    var id: String { thought.id }
}

struct GraphEdge: Decodable, Hashable, Identifiable {
    let link: ThoughtLink

    var id: String { link.id }
}

struct GraphNeighborhood: Decodable, Hashable {
    let center: GraphNode
    let nodes: [GraphNode]
    let edges: [GraphEdge]
}

enum SidebarSection: String, CaseIterable, Identifiable {
    case composer = "Composer"
    case telescope = "Telescope"
    case constellation = "Constellation"
    case concept = "Concept"
    case collections = "Collections"

    var id: String { rawValue }
}
