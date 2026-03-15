import Foundation

enum GraphQLClientError: Error {
    case invalidResponse
    case graphQLErrors([String])
}

struct GraphQLRequestBody<V: Encodable>: Encodable {
    let query: String
    let variables: V
}

struct GraphQLResponse<D: Decodable>: Decodable {
    let data: D?
    let errors: [GraphQLErrorPayload]?
}

struct GraphQLErrorPayload: Decodable {
    let message: String
}

final class GraphQLClient {
    let endpoint: URL

    init(endpoint: URL = URL(string: "http://127.0.0.1:4000/")!) {
        self.endpoint = endpoint
    }

    func send<V: Encodable, D: Decodable>(query: String, variables: V, data: D.Type) async throws -> D {
        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        request.addValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(GraphQLRequestBody(query: query, variables: variables))

        let (payload, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, (200 ..< 300).contains(http.statusCode) else {
            throw GraphQLClientError.invalidResponse
        }

        let decoded = try JSONDecoder.graphQL.decode(GraphQLResponse<D>.self, from: payload)
        if let errors = decoded.errors, !errors.isEmpty {
            throw GraphQLClientError.graphQLErrors(errors.map(\.message))
        }
        guard let data = decoded.data else {
            throw GraphQLClientError.invalidResponse
        }
        return data
    }
}

extension JSONDecoder {
    static var graphQL: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return decoder
    }
}
