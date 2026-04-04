# GraphQL (Query Language for APIs)

Design, query, and serve flexible APIs using GraphQL's type system, schema definition language, and resolver architecture to let clients request exactly the data they need.

## Schema Definition Language (SDL)

### Type Definitions

```graphql
type User {
  id: ID!
  email: String!
  name: String
  role: Role!
  posts(first: Int = 10, after: String): PostConnection!
  createdAt: DateTime!
}

enum Role {
  ADMIN
  EDITOR
  VIEWER
}

scalar DateTime

interface Node {
  id: ID!
}

type Post implements Node {
  id: ID!
  title: String!
  body: String!
  author: User!
  tags: [String!]!
  publishedAt: DateTime
}

union SearchResult = User | Post | Comment

input CreatePostInput {
  title: String!
  body: String!
  tags: [String!]
}
```

### Root Types

```graphql
type Query {
  user(id: ID!): User
  users(filter: UserFilter, first: Int, after: String): UserConnection!
  post(id: ID!): Post
  search(query: String!): [SearchResult!]!
}

type Mutation {
  createPost(input: CreatePostInput!): CreatePostPayload!
  updatePost(id: ID!, input: UpdatePostInput!): UpdatePostPayload!
  deletePost(id: ID!): DeletePostPayload!
}

type Subscription {
  postCreated: Post!
  commentAdded(postId: ID!): Comment!
}

schema {
  query: Query
  mutation: Mutation
  subscription: Subscription
}
```

### Relay-Style Pagination

```graphql
type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type PostEdge {
  cursor: String!
  node: Post!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

## Queries and Operations

### Basic Queries

```graphql
# Simple field selection
query GetUser {
  user(id: "123") {
    name
    email
    role
  }
}

# Nested queries with arguments
query GetUserPosts {
  user(id: "123") {
    name
    posts(first: 5) {
      edges {
        node {
          title
          publishedAt
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}

# Variables
query GetUser($userId: ID!) {
  user(id: $userId) {
    name
    email
  }
}

# Fragments for reuse
fragment UserFields on User {
  id
  name
  email
  role
}

query TwoUsers {
  alice: user(id: "1") { ...UserFields }
  bob: user(id: "2") { ...UserFields }
}
```

### Mutations

```graphql
mutation CreatePost($input: CreatePostInput!) {
  createPost(input: $input) {
    post {
      id
      title
    }
    errors {
      field
      message
    }
  }
}

# Variables:
# {
#   "input": {
#     "title": "Hello World",
#     "body": "First post content",
#     "tags": ["intro", "hello"]
#   }
# }
```

### Introspection

```graphql
# Full schema introspection
query IntrospectionQuery {
  __schema {
    types {
      name
      kind
      fields {
        name
        type { name kind }
      }
    }
  }
}

# Single type introspection
query TypeInfo {
  __type(name: "User") {
    name
    fields {
      name
      type {
        name
        kind
        ofType { name }
      }
    }
  }
}
```

## Server Implementation

### Resolver Pattern (Node.js / Apollo)

```javascript
const resolvers = {
  Query: {
    user: (parent, { id }, context) => context.db.users.findById(id),
    users: (parent, { filter, first, after }, context) => {
      return context.db.users.paginate({ filter, first, after });
    },
  },
  User: {
    posts: (user, { first, after }, context) => {
      return context.loaders.userPosts.load({
        userId: user.id, first, after
      });
    },
  },
  Mutation: {
    createPost: async (parent, { input }, context) => {
      const post = await context.db.posts.create(input);
      context.pubsub.publish('POST_CREATED', { postCreated: post });
      return { post, errors: [] };
    },
  },
  Subscription: {
    postCreated: {
      subscribe: (parent, args, context) =>
        context.pubsub.asyncIterator(['POST_CREATED']),
    },
  },
};
```

### DataLoader (N+1 Solution)

```javascript
const DataLoader = require('dataloader');

// Batch function: receives array of keys, returns array of results
const userLoader = new DataLoader(async (userIds) => {
  const users = await db.users.findByIds(userIds);
  // Must return in same order as input keys
  const userMap = new Map(users.map(u => [u.id, u]));
  return userIds.map(id => userMap.get(id) || null);
});

// In resolver -- each call is batched automatically
const resolvers = {
  Post: {
    author: (post, args, context) => context.loaders.user.load(post.authorId),
  },
};
```

## CLI and Development Tools

### GraphQL Queries via curl

```bash
# Simple query
curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "{ user(id: \"123\") { name email } }"}'

# Query with variables
curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query GetUser($id: ID!) { user(id: $id) { name } }",
    "variables": { "id": "123" }
  }'

# Introspection for schema discovery
curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { types { name } } }"}'
```

### Schema Validation and Codegen

```bash
# GraphQL Code Generator (TypeScript types from schema)
npm install -D @graphql-codegen/cli
npx graphql-codegen init
npx graphql-codegen --config codegen.yml

# Rover (Apollo Federation CLI)
npm install -g @apollo/rover
rover graph introspect https://api.example.com/graphql
rover subgraph check my-graph@prod --schema schema.graphql
rover supergraph compose --config supergraph.yaml

# graphql-inspector (breaking change detection)
npx graphql-inspector diff old.graphql new.graphql
npx graphql-inspector validate 'src/**/*.graphql' schema.graphql
```

## Federation (Apollo)

### Subgraph Schema

```graphql
# Users subgraph
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

# Posts subgraph
type Post @key(fields: "id") {
  id: ID!
  title: String!
  author: User!
}

extend type User @key(fields: "id") {
  id: ID! @external
  posts: [Post!]!
}
```

### Supergraph Composition

```yaml
# supergraph.yaml
federation_version: =2.7.0
subgraphs:
  users:
    routing_url: http://users-service:4001/graphql
    schema:
      file: ./schemas/users.graphql
  posts:
    routing_url: http://posts-service:4002/graphql
    schema:
      file: ./schemas/posts.graphql
```

## Tips

- Always use DataLoader to batch and deduplicate database queries -- the N+1 problem is the number one GraphQL performance killer
- Disable introspection in production by setting `introspection: false` in Apollo Server config to prevent schema leakage
- Use persisted queries (query hashing) to reduce payload size and prevent arbitrary query execution in production
- Set query depth and complexity limits to block abusive deeply-nested queries that can DoS your server
- Design mutations with input types and payload types following the Relay mutation convention for consistent error handling
- Use `@defer` and `@stream` directives (when supported) to progressively deliver large responses
- Prefer cursor-based pagination over offset pagination for stable results across concurrent writes
- Version your schema by deprecating fields (`@deprecated(reason: "Use newField")`) rather than introducing /v2 endpoints
- Run `graphql-inspector diff` in CI to catch unintentional breaking schema changes before merge
- Use union types for search results and error types rather than returning null with a separate error field
- Co-locate fragments next to the components that consume them to keep queries maintainable

## See Also

- rest-api, openapi, grpc, websockets, apollo, relay

## References

- [GraphQL Specification](https://spec.graphql.org/October2021/)
- [Apollo Server Documentation](https://www.apollographql.com/docs/apollo-server/)
- [DataLoader GitHub](https://github.com/graphql/dataloader)
- [Relay Specification](https://relay.dev/docs/guides/graphql-server-specification/)
- [Apollo Federation](https://www.apollographql.com/docs/federation/)
- [GraphQL Code Generator](https://the-guild.dev/graphql/codegen)
