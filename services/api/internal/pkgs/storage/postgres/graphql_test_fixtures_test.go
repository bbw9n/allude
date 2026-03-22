package postgres_test

const (
	mutationCreateThought = `
mutation CreateThought($content: String!) {
  createThought(content: $content) {
    id
    currentVersion {
      id
      version
      content
    }
    processingStatus
  }
}`

	mutationEditThought = `
mutation EditThought($thoughtId: ID!, $content: String!) {
  editThought(thoughtId: $thoughtId, content: $content) {
    id
    currentVersion {
      version
      content
    }
    versions {
      id
      version
    }
  }
}`

	mutationCreateCollection = `
mutation CreateCollection($title: String!, $description: String) {
  createCollection(title: $title, description: $description) {
    id
    title
    description
  }
}`

	mutationAddThoughtToCollection = `
mutation AddThoughtToCollection($collectionId: ID!, $thoughtId: ID!) {
  addThoughtToCollection(collectionId: $collectionId, thoughtId: $thoughtId) {
    id
    items {
      thought {
        id
      }
    }
  }
}`

	mutationRecordEngagement = `
mutation RecordEngagement($entityType: String!, $entityId: ID!, $actionType: String!, $dwellMs: Int) {
  recordEngagement(entityType: $entityType, entityId: $entityId, actionType: $actionType, dwellMs: $dwellMs) {
    id
    entityId
    actionType
    dwellMs
  }
}`

	queryThoughtLifecycle = `
query ThoughtLifecycle($id: ID!) {
  thought(id: $id) {
    id
    concepts {
      canonicalName
    }
    relatedThoughts {
      id
    }
    versions {
      id
      version
    }
    collections {
      id
    }
  }
  relatedThoughts(thoughtId: $id, limit: 8) {
    id
  }
  listThoughtVersions(thoughtId: $id) {
    id
    version
  }
}`

	queryDiscoverySearch = `
query Search($query: String!) {
  search(query: $query) {
    thoughts {
      id
    }
    clusters {
      label
    }
  }
  searchThoughts(query: $query) {
    thoughts {
      id
    }
  }
}`

	queryDiscoveryGraph = `
query Graph($thoughtId: ID!) {
  graph(centerThoughtId: $thoughtId, distance: 2, limit: 12) {
    center {
      thought {
        id
      }
    }
    nodes {
      thought {
        id
      }
    }
    edges {
      link {
        id
        relationType
      }
    }
  }
}`

	queryDiscoveryConcept = `
query Concept($name: String!) {
  concept(name: $name) {
    id
    canonicalName
    slug
    thoughtCount
    topThoughts {
      id
    }
    relatedConcepts {
      id
    }
    contradictionThoughts {
      id
    }
    aliases {
      alias
    }
  }
}`

	queryDiscoveryTelescope = `
query Discovery($query: String!, $thoughtId: ID!) {
  draftSuggestions(content: $query, thoughtId: $thoughtId) {
    relatedConcepts
    reframes
    supportingThoughts {
      id
    }
    counterThoughts {
      id
    }
  }
  telescope(query: $query) {
    query
    intent
    narrative
    seedThoughts {
      id
    }
    seedConcepts {
      canonicalName
    }
    graph {
      center {
        thought {
          id
        }
      }
    }
    suggestedJumps {
      label
      query
    }
    relatedCurrents {
      id
    }
  }
}`

	queryPersonalization = `
query ViewerAndHome($collectionId: ID!) {
  me {
    id
    username
    interests
  }
  viewer {
    id
    username
    interests
  }
  viewerInterests(limit: 6) {
    conceptId
    affinityScore
    concept {
      canonicalName
    }
  }
  myThoughts(limit: 10) {
    id
  }
  currents(limit: 4) {
    id
    title
    summary
    clusterKey
    thoughts {
      id
    }
    concepts {
      canonicalName
    }
  }
  home(limit: 4) {
    viewer {
      id
      interests
    }
    currents {
      id
    }
    recommendedThoughts {
      id
    }
    recommendedCollections {
      id
    }
  }
  collection(id: $collectionId) {
    id
    title
    items {
      thought {
        id
      }
    }
  }
  collections {
    id
    title
  }
}`
)
