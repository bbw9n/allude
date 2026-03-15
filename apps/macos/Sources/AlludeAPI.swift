import Foundation

enum AlludeAPI {
    static let thoughtFields = """
    id
    processingStatus
    processingNotes
    createdAt
    updatedAt
    currentVersion {
      id
      thoughtId
      version
      content
      createdAt
    }
    versions {
      id
      thoughtId
      version
      content
      createdAt
    }
    concepts {
      id
      name
      createdAt
    }
    relatedThoughts(limit: 8) {
      id
      processingStatus
      processingNotes
      createdAt
      updatedAt
      currentVersion {
        id
        thoughtId
        version
        content
        createdAt
      }
      concepts {
        id
        name
        createdAt
      }
    }
    links(limit: 8) {
      id
      sourceThoughtId
      targetThoughtId
      relationType
      score
      origin
      createdAt
    }
    """

    static let searchThoughts = """
    query SearchThoughts($query: String!) {
      searchThoughts(query: $query) {
        thoughts {
          \(thoughtFields)
        }
        clusters {
          label
          thoughtIds
          concepts {
            id
            name
            createdAt
          }
        }
      }
    }
    """

    static let createThought = """
    mutation CreateThought($content: String!) {
      createThought(content: $content) {
        \(thoughtFields)
      }
    }
    """

    static let updateThought = """
    mutation UpdateThought($thoughtId: ID!, $content: String!) {
      updateThought(thoughtId: $thoughtId, content: $content) {
        \(thoughtFields)
      }
    }
    """

    static let thought = """
    query Thought($id: ID!) {
      thought(id: $id) {
        \(thoughtFields)
      }
    }
    """

    static let graph = """
    query Graph($centerThoughtId: ID!, $distance: Int!, $limit: Int!) {
      graph(centerThoughtId: $centerThoughtId, distance: $distance, limit: $limit) {
        center {
          x
          y
          distance
          thought {
            \(thoughtFields)
          }
        }
        nodes {
          x
          y
          distance
          thought {
            \(thoughtFields)
          }
        }
        edges {
          link {
            id
            sourceThoughtId
            targetThoughtId
            relationType
            score
            origin
            createdAt
          }
        }
      }
    }
    """

    static let concept = """
    query Concept($id: ID) {
      concept(id: $id) {
        id
        name
        createdAt
        relatedConcepts(limit: 8) {
          id
          name
          createdAt
        }
        topThoughts(limit: 8) {
          \(thoughtFields)
        }
      }
    }
    """
}

struct SearchThoughtsData: Decodable {
    let searchThoughts: SearchThoughtsResult
}

struct ThoughtData: Decodable {
    let thought: Thought?
}

struct CreateThoughtData: Decodable {
    let createThought: Thought
}

struct UpdateThoughtData: Decodable {
    let updateThought: Thought
}

struct GraphData: Decodable {
    let graph: GraphNeighborhood
}

struct ConceptData: Decodable {
    let concept: Concept?
}
