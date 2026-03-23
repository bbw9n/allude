package graphql_test

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

	mutationUpdateThought = `
mutation UpdateThought($thoughtId: ID!, $content: String!) {
  updateThought(thoughtId: $thoughtId, content: $content) {
    id
    currentVersion {
      version
      content
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

	mutationCreateCapture = `
mutation CreateCapture($content: String!, $sourceType: String, $sourceTitle: String, $sourceUrl: String, $sourceApp: String) {
  createCapture(content: $content, sourceType: $sourceType, sourceTitle: $sourceTitle, sourceUrl: $sourceUrl, sourceApp: $sourceApp) {
    id
    content
    sourceType
    sourceTitle
    sourceUrl
    sourceApp
    status
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
    entityType
    entityId
    actionType
    dwellMs
  }
}`

	mutationArchiveCapture = `
mutation ArchiveCapture($captureId: ID!) {
  archiveCapture(captureId: $captureId) {
    id
    status
  }
}`

	mutationPromoteCapture = `
mutation PromoteCapture($captureId: ID!) {
  promoteCapture(captureId: $captureId) {
    id
    status
    promotedThoughtId
    promotedThought {
      id
    }
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
      thoughtIds
    }
  }
  searchThoughts(query: $query) {
    thoughts {
      id
    }
  }
}`

	queryConceptByName = `
query ConceptByName($name: String!) {
  concept(name: $name) {
    id
    canonicalName
    topThoughts {
      id
    }
    relatedConcepts {
      id
    }
    thoughtCount
  }
}`

	queryConceptBySlug = `
query ConceptBySlug($slug: String!) {
  concept(slug: $slug) {
    id
    canonicalName
  }
}`

	queryDiscoveryGraph = `
query Graph($thoughtId: ID!) {
  graph(thoughtId: $thoughtId, hopCount: 2, limit: 12) {
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

	queryDiscoverySurface = `
query Discovery($query: String!, $thoughtId: ID!, $collectionId: ID!) {
  currents(limit: 4) {
    id
    title
    thoughts {
      id
    }
    concepts {
      id
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
  draftSuggestions(content: $query, thoughtId: $thoughtId) {
    relatedConcepts
    reframes
    supportingThoughts {
      id
    }
    counterThoughts {
      id
    }
    notes
  }
  telescope(query: $query) {
    query
    intent
    narrative
    seedConcepts {
      id
      canonicalName
    }
    seedThoughts {
      id
    }
    graph {
      center {
        thought {
          id
        }
      }
    }
    clusters {
      label
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

	queryViewerAndPersonalization = `
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
    thoughts {
      id
    }
    concepts {
      id
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

	queryInbox = `
query Inbox {
  inbox(limit: 20) {
    id
    content
    sourceType
    sourceTitle
    sourceUrl
    sourceApp
    status
    promotedThoughtId
  }
}`

	queryCurrentsRich = `
query Discovery {
  currents(limit: 4) {
    id
    title
    summary
    clusterKey
    freshnessScore
    qualityScore
    thoughts {
      id
    }
    concepts {
      canonicalName
    }
  }
}`
)
