# GraphQL — ELI5

> GraphQL is a smart waiter at a restaurant who lets you order EXACTLY what you want and brings back exactly that — no more, no less.

## Prerequisites

It helps a tiny bit to know three things, but if you don't, that is fine. We will explain all three right here so you can keep going.

**JSON.** JSON is just text that looks like a list of "name: thing" pairs, with curly braces around them. When a computer wants to send another computer some structured information, it usually packs it as JSON. Here is what JSON looks like:

```json
{
  "name": "Alice",
  "age": 30,
  "pets": ["cat", "dog"]
}
```

That is it. Curly braces hold a record. Square brackets hold a list. Strings have quotes. Numbers do not. JSON is what GraphQL servers send back when you ask them for data.

**HTTP.** HTTP is the way your web browser talks to websites. When you type `google.com` into your browser and press Enter, your browser sends an HTTP request to Google's computer. Google's computer sends back an HTTP response. The request says "I want this thing." The response says "Here you go" plus the actual thing. HTTP requests have a method (GET, POST, PUT, DELETE), a path (`/users/123`), some headers, and sometimes a body. HTTP responses have a status code (200 means "ok," 404 means "not found," 500 means "the server broke") and a body.

**APIs.** API stands for "Application Programming Interface." That is a fancy phrase for "a way one program talks to another program." If you have a program that needs to know the weather, it might ask the weather company's program. The way it asks is called an API. Most modern APIs use HTTP. The program sends an HTTP request, the other program sends back an HTTP response, and inside that response is some JSON describing the answer.

If any of that is still fuzzy, here is the only thing you really need to remember: a server is a computer that waits for questions. A client is a program that asks the server questions. The questions and answers travel as text over HTTP. GraphQL is one way of writing those questions and answers.

## What Even Is GraphQL

### The big idea, in one sentence

GraphQL is a language you use to ask a server for **exactly the data you want, in exactly the shape you want it, from one single web address.**

That sentence has a lot in it. Let's slow down.

### The smart waiter

Imagine you walk into a restaurant. The old way of ordering food is like a fixed menu. The menu says "Combo #3: burger, fries, drink, salad, dessert." You wanted just the burger. But Combo #3 is what the kitchen makes, so you get the burger AND the fries AND the drink AND the salad AND the dessert. You eat the burger. You throw the rest away. The restaurant cooked food you didn't want. You waited longer for it. You wasted it.

That is how the old style of web APIs works. You ask for `/users/123` and the server sends back the whole user record. Name. Email. Address. Phone. Birthday. List of past orders. List of payment cards. List of friends. List of pets. You wanted the name. You got fifty fields. You throw the rest away.

Now imagine a smart waiter instead. You sit down. You say, "I'd like just a hamburger, no bun, with cheese on the side, and a glass of water with no ice." The waiter writes that down, walks to the kitchen, comes back with **exactly that.** Hamburger, no bun, cheese on the side, water with no ice. Nothing else. No fries you didn't ask for. No surprise pickle.

That is GraphQL. You write a little request that says exactly what fields you want, and the server gives back exactly those fields, in the shape you asked for.

### A picture of the difference

```
OLD-STYLE REST API
==================

YOU:    GET /users/123
        (give me user 123)

SERVER: {
          "id": 123,
          "name": "Alice",
          "email": "alice@example.com",
          "phone": "555-1234",
          "birthday": "1995-04-12",
          "address": { "street":..., "city":..., "zip":... },
          "orders": [ ... 47 orders ... ],
          "payment_cards": [ ... 3 cards ... ],
          "friends": [ ... 200 friends ... ],
          "pets": [ ... 4 pets ... ],
          "preferences": { ... 30 settings ... },
          "loyalty_points": 4500,
          "last_login": "2026-04-26T10:00:00Z"
        }

YOU:    "I just wanted the name. I'm throwing all this away."


GRAPHQL API
===========

YOU:    POST /graphql
        body: { user(id: 123) { name } }
        (give me user 123, but only the name)

SERVER: {
          "data": {
            "user": { "name": "Alice" }
          }
        }

YOU:    "Perfect. That is exactly what I asked for."
```

The OLD-STYLE picture is like ordering Combo #3 and throwing away everything except the burger. The GRAPHQL picture is like the smart waiter who only brings the burger.

### Where the name comes from

The "Graph" in GraphQL is because the data on a server is shaped like a graph. A graph in this sense is not a chart. A graph is a bunch of dots (called nodes) with lines connecting them (called edges). Alice is a dot. Alice's order is a dot. Alice's pet is a dot. There is a line from Alice to her order. There is a line from Alice to her pet. The whole world of data on a server is one giant graph of dots and lines.

The "QL" in GraphQL is short for "Query Language." A query language is a way of writing a question. SQL is a query language for databases. GraphQL is a query language for servers.

So GraphQL is "a language for asking questions about a graph of data." When you ask, you start at one dot, and you walk along the lines, picking up the things you want as you go. "Start at Alice. Get her name. Walk to her orders. For each order, get the total. Walk to the items in each order. For each item, get the title."

### A tiny GraphQL question

Here is what a GraphQL question (called a "query") looks like:

```graphql
{
  user(id: 123) {
    name
    email
    orders {
      total
      items {
        title
      }
    }
  }
}
```

That says: "Give me the user with ID 123. From that user, give me the name and the email. Also give me the orders. For each order, give me the total and the items. For each item, give me the title."

The server reads that, walks the graph, gathers the data, and sends back JSON shaped exactly like the question:

```json
{
  "data": {
    "user": {
      "name": "Alice",
      "email": "alice@example.com",
      "orders": [
        {
          "total": 42.50,
          "items": [{ "title": "Notebook" }, { "title": "Pen" }]
        },
        {
          "total": 9.99,
          "items": [{ "title": "Sticker" }]
        }
      ]
    }
  }
}
```

Notice: the JSON has the same shape as the query. Where the query said `name`, the JSON has `name`. Where the query said `orders`, the JSON has `orders`. The query is a template. The response fills in the template with real data.

### Who made it

Facebook built GraphQL in 2012 because their mobile app was wasting a lot of phone battery and bandwidth fetching data it didn't need. They open-sourced it in 2015. Since 2018, GraphQL has been managed by the **GraphQL Foundation**, which is part of the **Linux Foundation** (a non-profit that takes care of important open-source projects). The official spec lives at `graphql.org`. The October 2021 version of the spec is the most recent stable one. There is also a "working draft" that adds new features (like `@defer` and `@stream`, which we'll meet later).

### Three things GraphQL is NOT

GraphQL is not a database. The graph it talks about is a shape, not a place where data is stored. The actual data still lives wherever you keep it: PostgreSQL, MySQL, MongoDB, Redis, files on disk, other web services. GraphQL is a layer on top.

GraphQL is not a replacement for HTTP. It rides on HTTP. A GraphQL request is still an HTTP POST request to one URL.

GraphQL is not magic. You still have to write code on the server that knows how to fetch data. That code is called a "resolver." We'll meet resolvers soon.

## vs REST

### What REST looks like

REST stands for "Representational State Transfer," which is a fancy phrase that mostly means "use HTTP the normal way, with one URL per kind of thing, and use the HTTP methods (GET, POST, PUT, DELETE) for what they were meant for." A REST API for a blog might look like this:

```
GET    /posts             -> list of posts
GET    /posts/42          -> the post with id 42
POST   /posts             -> create a new post
PUT    /posts/42          -> update post 42
DELETE /posts/42          -> delete post 42
GET    /posts/42/comments -> the comments on post 42
GET    /users/7           -> user 7
GET    /users/7/posts     -> posts by user 7
```

Every "thing" gets a URL. Every URL returns one shape. If you want a post and its author, you might have to call `/posts/42` (which gives you `author_id: 7`) and then call `/users/7` to get the author's name. Two round trips. That is called the "n+1 round-trip problem at the client."

### Four big differences

GraphQL is different in four important ways. Let's go through each.

**Difference 1: One endpoint vs many endpoints.**

REST has a different URL for every kind of thing. `/users/7`, `/posts/42`, `/comments/99`, `/orders/123`. Hundreds of URLs.

GraphQL has ONE URL. Usually it is `/graphql`. Every request goes to that same URL. The body of the request says what you want. The URL never changes.

```
REST:    GET /users/7
         GET /posts/42
         GET /orders/123

GRAPHQL: POST /graphql  body: { user(id: 7)   { ... } }
         POST /graphql  body: { post(id: 42)  { ... } }
         POST /graphql  body: { order(id: 123){ ... } }
```

This is sometimes confusing for people coming from REST because HTTP caching, status codes, and tooling are all built around different URLs meaning different things. GraphQL throws that away on purpose.

**Difference 2: Server-specified shape vs client-specified shape.**

In REST, the server decides what fields come back. If `/users/7` returns thirty fields, you get thirty fields. If you want twenty-eight, tough.

In GraphQL, the client decides. You ask for `name` and `email`, you get exactly `name` and `email`. You ask for nothing extra. Nothing extra is sent.

This is sometimes called "over-fetching" (REST sends too much) and "under-fetching" (REST sends too little, and you have to call again). GraphQL fixes both.

**Difference 3: Untyped vs typed schema.**

REST does not require a schema. The server returns JSON. The fields of the JSON are whatever the server decided to send today. Tomorrow it might send different fields. There is no rule. There are tools (like OpenAPI / Swagger) that bolt a schema on top of REST, but the schema is optional.

GraphQL **requires** a schema. The schema is a written description of every type, every field, every argument, and what each one returns. You cannot have a GraphQL server without a schema. The server checks every query against the schema before running it. If you ask for a field that does not exist, the server says no. If you pass the wrong type of argument, the server says no.

This is huge. It means tools can read the schema and help you. Editors can autocomplete field names. Linters can catch typos before you ship. Code generators can build TypeScript types automatically.

**Difference 4: Introspection.**

Because GraphQL servers carry their own schema, they can describe themselves. You can ask a GraphQL server: "What types do you have?" and it will tell you. You can ask: "What fields does the User type have?" and it will tell you. This is called **introspection.** It is a built-in part of every GraphQL server.

REST has nothing like this. With REST you read the docs (if there are docs) and hope they match the actual server.

```
INTROSPECTION QUERY
===================

YOU:    POST /graphql
        body: { __schema { types { name } } }

SERVER: {
          "data": {
            "__schema": {
              "types": [
                { "name": "User" },
                { "name": "Order" },
                { "name": "Item" },
                { "name": "Query" },
                { "name": "Mutation" },
                { "name": "ID" },
                { "name": "String" },
                ...
              ]
            }
          }
        }
```

A heads-up about introspection in production: it is great for development, but in a public production server it lets attackers map your whole API for free. Most teams **disable introspection in production** for that reason. We will come back to that under "Caching" and "Common Confusions."

### A side-by-side example

Let's say you want a user's name and the titles of their last 3 orders. Here is REST:

```bash
# Step 1: get the user
$ curl https://api.example.com/users/7
{ "id": 7, "name": "Alice", "email": "alice@example.com", "phone": "555-1234", ... }

# Step 2: get the orders
$ curl 'https://api.example.com/users/7/orders?limit=3'
[
  { "id": 100, "total": 42.50, "status": "shipped", "created": "...", "items_count": 3 },
  { "id": 101, "total": 9.99,  "status": "shipped", "created": "...", "items_count": 1 },
  { "id": 102, "total": 19.95, "status": "pending", "created": "...", "items_count": 2 }
]

# Step 3: ...the orders don't have titles. Get each order's items.
$ curl https://api.example.com/orders/100/items
$ curl https://api.example.com/orders/101/items
$ curl https://api.example.com/orders/102/items

# Step 4: piece all that together. Three round trips. Lots of fields you don't need.
```

Here is GraphQL:

```bash
$ curl -H 'Content-Type: application/json' -X POST https://api.example.com/graphql \
       -d '{"query":"{ user(id:\"7\") { name orders(last:3) { items { title } } } }"}'
{
  "data": {
    "user": {
      "name": "Alice",
      "orders": [
        { "items": [{ "title": "Notebook" }, { "title": "Pen" }, { "title": "Eraser" }] },
        { "items": [{ "title": "Sticker" }] },
        { "items": [{ "title": "Cup" }, { "title": "Mug" }] }
      ]
    }
  }
}
```

One request. Exactly the fields you want. No round trips. No wasted bytes. That is the GraphQL pitch in one breath.

## The Schema Definition Language (SDL)

### What the SDL is

The SDL is the language you use to write a GraphQL schema. SDL stands for "Schema Definition Language." A schema is a description of the shape of your data. Every GraphQL server has a schema. The SDL is how you write it down.

The SDL is a tiny language. It has types, fields, arguments, and directives. That is it. Once you learn those four things, you can read any GraphQL schema.

### Writing a tiny schema

Here is a complete (small) schema:

```graphql
type User {
  id: ID!
  name: String!
  email: String
  age: Int
  isAdmin: Boolean!
  orders: [Order!]!
}

type Order {
  id: ID!
  total: Float!
  items: [Item!]!
  status: OrderStatus!
}

type Item {
  title: String!
  price: Float!
}

enum OrderStatus {
  PENDING
  SHIPPED
  DELIVERED
  CANCELLED
}

type Query {
  user(id: ID!): User
  order(id: ID!): Order
}

type Mutation {
  createOrder(userId: ID!, items: [ItemInput!]!): Order!
}

input ItemInput {
  title: String!
  price: Float!
}
```

That is a real GraphQL schema. Let's read it line by line.

### Types

A **type** is a kind of thing. `type User { ... }` says "there is a kind of thing called a User, and here are its fields."

The fields go inside the curly braces. Each field has a name, a colon, and a return type. `name: String!` means "there is a field called `name` and it returns a String."

GraphQL has six kinds of types:

1. **Object types** — like `User`, `Order`, `Item` above. They have fields.
2. **Scalars** — leaf types that hold a single value. Built-in scalars are `Int`, `Float`, `String`, `Boolean`, and `ID`. You can define **custom scalars** like `DateTime`, `UUID`, or `JSON`.
3. **Enums** — a fixed list of allowed string values. `OrderStatus` is an enum.
4. **Interfaces** — a contract that other types can implement. "Anything with `id` and `name` is a Named." We'll see this below.
5. **Unions** — "this field can be one of these types." Like "the search result is either a User or an Order."
6. **Input types** — types that go INTO a query as arguments. `input ItemInput { ... }` is an input type.

### Scalars

Scalars are the leaves of the graph. They are the actual values, not records of more values. Built-in scalars:

- **`Int`** — a 32-bit signed integer.
- **`Float`** — a double-precision floating point number.
- **`String`** — a UTF-8 string.
- **`Boolean`** — `true` or `false`.
- **`ID`** — a unique identifier, serialized as a string. It looks like a String, but it carries the meaning "this is an ID, do not display it as a number, never do math on it." A bank account number that happens to start with zeros should be an `ID`, not an `Int`.

You can define **custom scalars** for things like dates and UUIDs:

```graphql
scalar DateTime
scalar UUID
scalar JSON

type Event {
  id: UUID!
  startsAt: DateTime!
  metadata: JSON
}
```

Custom scalars need a "scalar definition" on the server side that tells GraphQL how to serialize and parse them. The schema only declares "there is a custom scalar called `DateTime`." The server code says "and here is how to turn it into a string and back."

### Lists and non-null

Two little marks change a type:

- **`[T]`** means "a list of T." `[Order]` is "a list of orders."
- **`T!`** means "this is non-null. The server promises this will always have a value, never null."

You can stack them. Here are the four common shapes:

- **`[Order]`** — list might be null; items in the list might be null. Worst case.
- **`[Order!]`** — list might be null; items in the list are never null.
- **`[Order]!`** — list is never null (might be empty `[]`); items might be null.
- **`[Order!]!`** — list is never null; items are never null. The strongest contract.

Pick the strongest one you can honestly promise. Lists of things you control should usually be `[T!]!`. If your server can guarantee the list always exists and the items are always present, say so. The client can stop writing null-checks.

### Enums

An enum is a closed set of allowed values:

```graphql
enum OrderStatus {
  PENDING
  SHIPPED
  DELIVERED
  CANCELLED
}
```

If a field is `OrderStatus!`, the server promises it will return exactly one of those four values. The client can write a `switch` over them and know it's complete.

Enums travel as strings on the wire. They are NOT integers. `PENDING` is sent as the string `"PENDING"`.

### Interfaces

An interface is a contract. Other types implement it. Example:

```graphql
interface Named {
  id: ID!
  name: String!
}

type User implements Named {
  id: ID!
  name: String!
  email: String
}

type Pet implements Named {
  id: ID!
  name: String!
  species: String!
}
```

Now any field that returns `Named` could return either a User or a Pet. The client gets back something with at least `id` and `name`, and can ask for type-specific extras using a fragment (which we'll see later).

### Unions

A union is "one of these types, no shared fields required":

```graphql
union SearchResult = User | Order | Pet

type Query {
  search(query: String!): [SearchResult!]!
}
```

The client gets back a list of things that could be users, orders, or pets. To pick fields, you use **inline fragments**:

```graphql
{
  search(query: "alice") {
    ... on User { name email }
    ... on Order { total }
    ... on Pet { species }
  }
}
```

`... on User` means "if this thing is a User, get these fields."

### Input types

You cannot pass an output type as an argument. You have to define a separate **input type**:

```graphql
input ItemInput {
  title: String!
  price: Float!
}

type Mutation {
  createOrder(userId: ID!, items: [ItemInput!]!): Order!
}
```

Input types use the keyword `input`, not `type`. They cannot have nested object types as fields, only scalars, enums, lists, and other input types. This rule keeps inputs simple and validatable.

### Directives

A directive is an annotation that changes how something behaves. Directives start with `@`. Built-in ones:

- **`@skip(if: Boolean!)`** — used in queries: "skip this field if the condition is true."
- **`@include(if: Boolean!)`** — used in queries: "include this field only if the condition is true."
- **`@deprecated(reason: String)`** — marks a field or enum value as deprecated. Tools warn when you use it.
- **`@specifiedBy(url: String!)`** — points at the spec for a custom scalar.
- **`@oneOf`** — on input types, "exactly one of these fields must be set." (Working draft, gaining adoption.)

You can also define **custom directives** for your own purposes:

```graphql
directive @auth(requires: Role!) on FIELD_DEFINITION

type Query {
  secretStuff: String! @auth(requires: ADMIN)
}
```

The server reads the directive and decides what to do. We'll see auth directives below.

### The three root types

Every schema has up to three special types: **Query**, **Mutation**, and **Subscription**.

- `Query` is for reading.
- `Mutation` is for writing.
- `Subscription` is for getting pushed updates.

These are normal object types — they are not magic — but the GraphQL server treats them as the entry points. When you send a `query { ... }` request, the server starts resolving fields on the `Query` type. When you send a `mutation { ... }`, it starts on the `Mutation` type.

You don't have to have all three. A read-only API has only `Query`. A simple service might skip subscriptions.

## Query (read), Mutation (write), Subscription (push via WebSocket)

### Query

A query is how you ask for data. You start with the keyword `query` (or you can leave it off; bare braces default to `query`):

```graphql
query GetUserAndOrders {
  user(id: "7") {
    name
    orders {
      total
    }
  }
}
```

The name (`GetUserAndOrders`) is optional but helpful — it shows up in logs and tooling. Best practice is to name every query.

Queries can be sent as HTTP POST with a JSON body, or as HTTP GET with the query in a query parameter (less common, useful for caching). Most servers accept POST.

```bash
$ curl -H 'Content-Type: application/json' -X POST http://api/graphql \
       -d '{"query":"query GetUserAndOrders { user(id:\"7\"){ name orders { total }} }"}'
```

### Mutation

A mutation is how you change data: create, update, delete. Mutations look almost identical to queries, but you start with the keyword `mutation`:

```graphql
mutation CreateOrder($userId: ID!, $items: [ItemInput!]!) {
  createOrder(userId: $userId, items: $items) {
    id
    total
    status
  }
}
```

Mutations are still HTTP POST. They are NOT the only writes — that is a common confusion. The HTTP method is always POST for mutations; the keyword in the body is what tells the server it's a mutation. Nothing about HTTP changes.

A subtle but real rule: **mutations at the same level run in order, one at a time**, while query fields can be resolved in parallel. That's because writes need to happen in a deterministic order. If you send a mutation that says "deduct money, then add item to cart," you want them in that order.

### Subscription

A subscription is how the server pushes data to you over time, instead of you pulling. The classic example is a chat: when somebody sends a new message, every other client should receive it without polling.

```graphql
subscription NewMessages($roomId: ID!) {
  messageAdded(roomId: $roomId) {
    id
    text
    author { name }
  }
}
```

Subscriptions are NOT a single HTTP request. They run over a long-lived connection. The two common transports:

- **WebSocket**, using the `graphql-ws` protocol (and the older deprecated `subscriptions-transport-ws`).
- **Server-Sent Events** (SSE), using the `graphql-sse` protocol.

We'll explore the transport in its own section.

### A picture of the three

```
QUERY (read)
============
  Client --POST--> Server
  Client <--JSON-- Server
  (one request, one response, done)


MUTATION (write)
================
  Client --POST--> Server
  Client <--JSON-- Server
  (one request, one response, done — but server changed something)


SUBSCRIPTION (push)
===================
  Client --upgrade to WebSocket--> Server
  Client <--ack-- Server
  Client --subscribe-- Server
  Client <--data-- Server (whenever something happens)
  Client <--data-- Server
  Client <--data-- Server
  ... (forever, until either side closes)
```

## Resolvers

### What a resolver is

A resolver is a function on the server that knows how to get one piece of data. There is one resolver per field on every type. When the server processes a query, it walks the query, and at each field, it calls the resolver for that field.

If your schema has:

```graphql
type Query {
  user(id: ID!): User
}

type User {
  name: String!
  orders: [Order!]!
}
```

Then on the server, you write:

```javascript
const resolvers = {
  Query: {
    user: (parent, args, context, info) => {
      return db.users.find(u => u.id === args.id)
    }
  },
  User: {
    name:   (parent, args, context, info) => parent.name,
    orders: (parent, args, context, info) => db.orders.find(o => o.userId === parent.id)
  }
}
```

That is JavaScript, but every GraphQL framework in every language follows the same pattern: name the type, name the field, give a function that returns the value.

### The four arguments to every resolver

Every resolver receives four arguments. They are sometimes called `parent, args, context, info` or `root, args, ctx, info`. Different libraries use different names. The shape is the same.

1. **`parent`** — the value returned by the parent field. For `User.name`, the parent is the User. For `Query.user`, the parent is `null` (top-level fields have no parent).
2. **`args`** — the arguments passed to this field. For `user(id: "7")`, args is `{ id: "7" }`.
3. **`context`** — a per-request bag the server gives you. Usually it has the database connection, the current user, the request headers, your loggers, etc. The context is your safe place to put cross-cutting things.
4. **`info`** — info about the current execution: the field name, the path through the query, and the parsed selection set. Most resolvers ignore `info`. Advanced resolvers use it to look at "what fields did the client actually ask for?" and skip work.

### A picture of resolution

```
QUERY:
  { user(id: "7") { name orders { total } } }


SERVER WALK:

  1. Start: type Query, field "user"
     -> call Query.user resolver, args = { id: "7" }
     -> returns the User record { id: "7", name: "Alice", ... }

  2. For each subfield of the result:

     2a. type User, field "name"
         -> call User.name resolver, parent = the User
         -> returns "Alice"

     2b. type User, field "orders"
         -> call User.orders resolver, parent = the User
         -> returns [order100, order101]

         For each order in the list, walk subfields:

           type Order, field "total"
             -> call Order.total resolver, parent = orderN
             -> returns 42.50

  3. Stitch all the returned values into the response shape.
     Send back JSON.
```

### Default resolvers

Most fields don't need a resolver. If you don't write one, GraphQL uses a default: "look at `parent.<fieldName>` and return that." So `User.name` doesn't need a resolver if the parent object already has a `name` property. You only write resolvers when:

- The field needs to fetch from a database or service (almost every top-level field).
- The field name on the server differs from the schema name.
- The field needs a transformation (formatting, computing).

### Async resolvers

Real resolvers are almost always async — they make a database call or an HTTP call. Modern GraphQL libraries handle promises automatically:

```javascript
const resolvers = {
  Query: {
    user: async (parent, args, context) => {
      return await context.db.query('SELECT * FROM users WHERE id = $1', [args.id])
    }
  }
}
```

The server waits for the promise to resolve before walking deeper.

## The N+1 Problem and DataLoader

### What N+1 means

The N+1 problem is a famous performance trap in GraphQL. It is also a famous performance trap in classical ORMs (object-relational mappers). It comes from this question:

> What if I ask for a list of N things, and for each one I ask for a related thing?

Example query:

```graphql
{
  users {
    name
    orders {
      total
    }
  }
}
```

Naive resolvers:

```javascript
{
  Query: {
    users: () => db.query('SELECT * FROM users')
  },
  User: {
    orders: (user) => db.query('SELECT * FROM orders WHERE user_id = $1', [user.id])
  }
}
```

If there are 100 users, here is what happens:

```
Query 1:    SELECT * FROM users
            -> 100 users
Query 2:    SELECT * FROM orders WHERE user_id = 1
Query 3:    SELECT * FROM orders WHERE user_id = 2
Query 4:    SELECT * FROM orders WHERE user_id = 3
...
Query 101:  SELECT * FROM orders WHERE user_id = 100

Total queries: 1 + 100 = 101.
That is the N+1 problem: one query for the parents, and N queries for the children.
```

If your database is fast and the rows are cached this might be fine. If not, your API just turned a single GraphQL request into 101 database round-trips. With 1000 users, 1001 round-trips.

### A picture of N+1

```
WITHOUT DATALOADER (N+1)
========================

Client: { users { orders { total } } }

  resolver Query.users
      |
      v
  DB:  SELECT * FROM users
       (1 query)
      |
      v
  100 user records
      |
      +---> resolver User.orders for user 1
      |        |
      |        v
      |     DB: SELECT * FROM orders WHERE user_id=1   (query 2)
      |
      +---> resolver User.orders for user 2
      |        |
      |        v
      |     DB: SELECT * FROM orders WHERE user_id=2   (query 3)
      |
      +---> resolver User.orders for user 3
      |        |
      |        v
      |     DB: SELECT * FROM orders WHERE user_id=3   (query 4)
      |
      ... 97 more queries ...

  Total: 101 database queries for one API request.


WITH DATALOADER
===============

Client: { users { orders { total } } }

  resolver Query.users
      |
      v
  DB: SELECT * FROM users   (1 query)
      |
      v
  100 user records
      |
      +---> resolver User.orders for user 1 -> loader.load(1)
      +---> resolver User.orders for user 2 -> loader.load(2)
      +---> resolver User.orders for user 3 -> loader.load(3)
      ... 97 more queues ...

  All loader.load() calls in the same tick batch up.

  loader fires once:
    DB: SELECT * FROM orders WHERE user_id IN (1,2,3,...,100)   (query 2)

  Splits the result back into per-user lists, hands each list
  to the right resolver.

  Total: 2 database queries for one API request.
```

### What DataLoader is

**DataLoader** is a tiny library invented at Facebook in 2015 to fix the N+1 problem. It does two things:

1. **Batching.** It collects all the IDs requested in the same tick, sends ONE database query for all of them, then splits the results.
2. **Caching.** Within a single request, if the same ID is requested twice, the second request gets the cached result.

You write a "batch function" that takes a list of keys and returns a list of values:

```javascript
const DataLoader = require('dataloader')

const ordersByUserId = new DataLoader(async (userIds) => {
  const rows = await db.query(
    'SELECT * FROM orders WHERE user_id = ANY($1)',
    [userIds]
  )
  // Group rows by user_id, return in same order as userIds
  return userIds.map(id => rows.filter(r => r.user_id === id))
})

const resolvers = {
  User: {
    orders: (user, args, context) => context.ordersByUserId.load(user.id)
  }
}
```

Now, no matter how many users come through, only ONE batched query goes to the database for orders. N+1 becomes 2.

### When to create a DataLoader

You create a NEW DataLoader **per request**. Never share one between requests. The cache is meant to live for the lifetime of one HTTP request and then die. Sharing across requests would mean Alice gets Bob's cached order data — a security bug.

Most servers create DataLoaders in the per-request `context` setup:

```javascript
const server = new ApolloServer({
  schema,
  context: () => ({
    db,
    ordersByUserId: new DataLoader(/* batch fn */),
    usersById: new DataLoader(/* batch fn */),
  })
})
```

## Aliases / Fragments / Variables / Directives

### Aliases

By default, the response field name matches the query field name. If you ask for `name`, the response has `name`. But what if you ask for the same field twice with different arguments? Like, "get me user 1's name AND user 2's name in the same query"?

You can't have two top-level fields named `user`. You'd get a conflict. Aliases fix that:

```graphql
{
  alice: user(id: "1") { name }
  bob:   user(id: "2") { name }
}
```

Response:

```json
{
  "data": {
    "alice": { "name": "Alice" },
    "bob":   { "name": "Bob" }
  }
}
```

The alias `alice` and `bob` rename the response keys. This works on any field at any depth.

### Fragments

A fragment is a reusable chunk of fields. You define it once and use it in multiple places:

```graphql
fragment UserBasics on User {
  id
  name
  email
}

query Two {
  alice: user(id: "1") { ...UserBasics }
  bob:   user(id: "2") { ...UserBasics }
}
```

The `...UserBasics` is called a "fragment spread." It expands to the fields inside the fragment definition.

Fragments are great for **deduplicating** common field selections. They are not for abstraction — they don't hide anything from the server. They are a copy-paste alias.

There are also **inline fragments**, which we saw above for unions:

```graphql
{
  search(query: "alice") {
    ... on User { name email }
    ... on Order { total }
  }
}
```

The `... on User` is an inline fragment. It says "in the case where this thing is a User, ask for these fields."

### Variables

Hardcoding values into a query string is bad. It tangles up code. It also opens you up to injection if you build the string by gluing strings together. Use **variables**:

```graphql
query GetUser($id: ID!, $includeOrders: Boolean!) {
  user(id: $id) {
    name
    orders @include(if: $includeOrders) {
      total
    }
  }
}
```

The variables are sent in a separate JSON field of the request body:

```bash
$ curl -H 'Content-Type: application/json' -X POST http://api/graphql -d '
{
  "query": "query GetUser($id: ID!, $includeOrders: Boolean!) { user(id: $id) { name orders @include(if: $includeOrders) { total } } }",
  "variables": { "id": "7", "includeOrders": true }
}'
```

Variables are **typed** — they have to match the schema. `$id: ID!` means "this variable is an ID and is required." If you forget it, the server says: `Variable "$id" of required type "ID!" was not provided`.

Variables are NOT string interpolation. The query is a template that the server parses ONCE; variables fill in the holes after parsing. There is no SQL-injection-shaped attack on a properly parameterized GraphQL query.

### Built-in directives

Directives modify execution. Two built-ins for queries:

- **`@skip(if: Boolean!)`** — skip this field if true.
- **`@include(if: Boolean!)`** — include this field only if true.

```graphql
query Profile($withOrders: Boolean!) {
  user(id: "7") {
    name
    orders @include(if: $withOrders) {
      total
    }
    secretStuff @skip(if: true)   # always skipped
  }
}
```

Both `@skip` and `@include` take a Boolean argument. They are mutually opposite. By convention, prefer `@include` because the positive form is easier to read.

### Custom directives

You can define your own directives in the schema. Common ones:

- `@auth(requires: Role!)` — restrict by role.
- `@cost(complexity: Int!)` — annotate the cost of a field for query-cost limiting.
- `@formatDate(format: String!)` — transform output.

The server has to actually do something with the directive — directives are inert until code reads them. We'll see how `@auth` directives work below.

### `@defer` and `@stream` (June 2023 spec)

Two newer directives, in the **June 2023 incremental delivery RFC**, are gaining adoption:

- **`@defer`** — "send the rest of the response back later." Useful for slow fields. The client gets the fast parts immediately, then the slow parts as a follow-up patch.
- **`@stream`** — for list fields: "send list items one at a time as they're ready."

```graphql
query SlowFields {
  user(id: "7") {
    name                          # fast
    expensiveStats @defer {       # slow — server can send this later
      monthlyTotal
    }
  }
}
```

Servers stream `@defer`/`@stream` responses using the `multipart/mixed` HTTP response format. Apollo Server, GraphQL Yoga, and the GraphQL Working Draft have stable implementations. Apollo Federation 2 supports it. Older clients fall back gracefully.

## Error Handling

### The shape of errors

Every GraphQL response has at most two top-level fields: `data` and `errors`. A successful response has `data` and no `errors`. A fully failed response has `errors` and no `data` (or `data: null`). A **partial failure** has BOTH — some fields succeeded, some failed.

```json
{
  "data": {
    "user": {
      "name": "Alice",
      "orders": null
    }
  },
  "errors": [
    {
      "message": "Database connection lost while fetching orders",
      "path": ["user", "orders"],
      "locations": [{ "line": 4, "column": 5 }],
      "extensions": {
        "code": "INTERNAL_SERVER_ERROR",
        "exception": { /* server-only details */ }
      }
    }
  ]
}
```

The fields of an error:

- **`message`** — human-readable text. The client should NOT parse this for logic. It can change.
- **`path`** — the JSON path to the field that failed: `["user", "orders"]` means `data.user.orders`.
- **`locations`** — line/column in the original query string where the bad field was. Useful for tooling.
- **`extensions`** — any extra info the server wants to share. Conventionally has a `code` like `"UNAUTHENTICATED"`, `"FORBIDDEN"`, `"BAD_USER_INPUT"`, `"INTERNAL_SERVER_ERROR"`.

The HTTP status code stays 200 OK for partial failures. That is unusual compared to REST, where errors are 4xx/5xx. In GraphQL, the body is what matters.

### Field-level errors

If a non-nullable field's resolver throws, the error "bubbles up" to the nearest nullable parent. Example: if `user.orders` is `[Order!]!` and an order's resolver fails, the entire `orders` list becomes the bubble target. If `orders` itself is non-null, it bubbles further up to `user`. The whole `user` becomes null.

This is why you should think hard about non-null. `[Order!]!` is a strong promise. If you can't keep it under errors, downgrade to `[Order!]` so a single error doesn't black-hole the whole list.

### Custom error codes

Most servers expose a way to throw structured errors. In Apollo Server 4:

```javascript
import { GraphQLError } from 'graphql'

throw new GraphQLError('Not authenticated', {
  extensions: { code: 'UNAUTHENTICATED', http: { status: 401 } }
})
```

Common error codes you'll see:

- `UNAUTHENTICATED` — the request had no valid login.
- `FORBIDDEN` — logged in, but not allowed.
- `BAD_USER_INPUT` — the inputs were invalid.
- `NOT_FOUND` — the thing wasn't found.
- `INTERNAL_SERVER_ERROR` — something on the server broke.
- `PERSISTED_QUERY_NOT_FOUND` / `PERSISTED_QUERY_NOT_SUPPORTED` — APQ-related.

## Auth Patterns

There are three common places to do auth in GraphQL. From outside in:

### Pattern 1: Auth in context

The HTTP middleware reads the `Authorization` header, looks up the user, and attaches them to the per-request `context`:

```javascript
const server = new ApolloServer({ schema })

await server.start()
app.use('/graphql', expressMiddleware(server, {
  context: async ({ req }) => {
    const token = req.headers.authorization?.replace('Bearer ', '')
    const user  = await verifyToken(token)
    return { user, db }
  }
}))
```

Now every resolver gets `context.user`.

### Pattern 2: Resolver-level checks

Each resolver decides whether to allow the call:

```javascript
const resolvers = {
  Query: {
    secretStuff: (parent, args, context) => {
      if (!context.user)               throw new GraphQLError('Not signed in',  { extensions: { code: 'UNAUTHENTICATED' }})
      if (!context.user.isAdmin)       throw new GraphQLError('Forbidden',      { extensions: { code: 'FORBIDDEN' }})
      return db.secrets.findAll()
    }
  }
}
```

Pros: explicit, easy to reason about. Cons: copy-paste; easy to forget on a new field.

### Pattern 3: Field-level directives

Mark fields in the schema with a custom directive, and have the server enforce it:

```graphql
directive @auth(requires: Role = USER) on FIELD_DEFINITION

enum Role { USER ADMIN }

type Query {
  publicStuff: String
  secretStuff: String @auth(requires: ADMIN)
}
```

The server reads the directive and inserts a wrapper resolver that checks the user's role before running the real one. Pros: declarative; visible in the schema; harder to forget. Cons: needs a directive transformer; not as flexible as explicit checks.

### Don't put auth in the schema TEXT

A common mistake is to assume the SDL itself enforces auth. It does not. The SDL is a description. The directive `@auth(requires: ADMIN)` is just an annotation. The server has to read it and act on it. If your server doesn't have a directive transformer for `@auth`, it does nothing.

## Caching

### Why HTTP caching is hard with GraphQL

REST caching is straightforward: each URL has a unique response, so HTTP caches (browsers, CDNs, proxies) can cache by URL. `GET /users/7` returns the same body, so cache it.

GraphQL has ONE URL. Every request is a POST. POSTs are not cached by default. Even if you switched to GET, the body is different every time, so the URL+body combo is rarely the same twice in a row. HTTP-level caching of GraphQL is effectively dead unless you do something special.

### Solution 1: Persisted Queries / APQ

A **persisted query** is a query string that the server stores by hash. Instead of sending the whole query text every time, the client sends a SHA-256 hash:

```bash
# First time: the client doesn't know the hash, sends the query plus its hash.
$ curl -X POST http://api/graphql \
       -d '{"query":"...full query...","extensions":{"persistedQuery":{"version":1,"sha256Hash":"abc..."}}}'

# Server stores it under hash abc..., returns the data.

# Second time: client just sends the hash.
$ curl -X POST http://api/graphql \
       -d '{"extensions":{"persistedQuery":{"version":1,"sha256Hash":"abc..."}}}'

# Server looks up the query by hash, runs it.
```

This is **APQ — Automatic Persisted Queries.** It reduces request size, which is great on mobile. It also enables HTTP-level caching: the URL `+hash` is a stable cache key.

A stronger version is **trusted documents** (sometimes called a "query whitelist"): the server **only** accepts queries whose hash is in a pre-registered list. That blocks unknown queries and is a defense against attack queries. Persisted queries are NOT just for security on their own — security comes from the trusted-document mode.

If a client sends a hash the server doesn't know, the response is `Persisted query not found`, and the client retries with the full query.

### Solution 2: Client-side normalized cache

Every serious GraphQL client has a **normalized cache.** Instead of caching by URL, the client looks at returned objects, finds their unique IDs, and stores each entity in a flat key-value map. Then queries are answered from that map when possible.

Apollo Client uses `InMemoryCache`. Relay has its own store. urql has Graphcache.

```
Server response:                   Normalized cache:
{
  "user": {                          users:
    "id": "7",                         "7": { id: "7", name: "Alice", orders: [order:100, order:101] }
    "name": "Alice",
    "orders": [                      orders:
      { "id": "100", "total": 42 },    "100": { id: "100", total: 42 }
      { "id": "101", "total": 9 }      "101": { id: "101", total: 9 }
    ]
  }
}
```

When a mutation updates user 7, the cache patches the entry, and every component subscribed to user 7 re-renders. This is one of the best parts of GraphQL.

### Solution 3: CDN caching with Stellate (or similar)

A few services (**Stellate**, formerly **GraphCDN**) sit in front of your GraphQL server and cache responses by hashing the operation. They speak GraphQL natively and invalidate on mutations. If your data is mostly read-heavy, this is huge. (Tyk Gateway and Apollo Router have similar features.)

### `@defer` / `@stream` and partial caching

When using `@defer` and `@stream`, the server sends the response in chunks. Clients fill the cache progressively. Most normalized caches handle this transparently — `@defer` payloads patch the existing cache entry as they arrive.

## Federation / Stitching

### The problem at scale

A single team can run a single GraphQL server happily. A company with twenty teams cannot. Each team owns part of the schema. They want to deploy independently. They don't want to run one giant monolith.

### Stitching (legacy)

The original answer was **schema stitching**: combine multiple GraphQL schemas into one big schema in a gateway. The gateway delegates to backend services. This worked, but the developer experience was rough — too many edge cases, messy ownership.

### Apollo Federation v1

Federation v1 (2019) was Apollo's structured take. It introduced:

- **Subgraph** — a GraphQL service owned by one team.
- **Supergraph** — the composed view of all subgraphs.
- **Gateway** — the runtime that routes requests to subgraphs.
- **`@key`** — a directive on a type that tells the gateway "this type can be resolved by ID across subgraphs."

It worked, but had performance issues at scale.

### Apollo Federation v2 (GA 2022)

**Federation v2** is a rewrite that is now the default. Key directives:

- `@key(fields: "id")` — declares an entity's lookup field. An "entity" is a type that can be referenced from multiple subgraphs.
- `@external` — "this field exists in another subgraph."
- `@requires(fields: "x y")` — "to compute this field, I need these other fields fetched first."
- `@provides(fields: "name")` — "I can supply these fields without a roundtrip."
- `@shareable` — "more than one subgraph defines this field; that's fine."
- `@inaccessible` — "hide this from the public schema."
- `@tag(name: "...")` — tag fields for tooling and routing rules.

The runtime is the **Apollo Router**, written in Rust. It is much faster than the older JavaScript gateway.

```
FEDERATION
==========

         +-------------------+
         |   Apollo Router   |   (Rust, fast, single endpoint)
         +-------------------+
            |     |      |
   +--------+     |      +--------+
   |              |               |
+------+      +------+         +------+
|Users |      |Orders|         |Items |
|sub-  |      |sub-  |         |sub-  |
|graph |      |graph |         |graph |
+------+      +------+         +------+
   |             |                |
  DB           DB               DB

Each subgraph owns part of the schema. The router composes them
into a single schema (the supergraph) and routes each query path
to the right subgraph.
```

### Alternatives to Apollo Federation

- **GraphQL Mesh** — composes REST, gRPC, OpenAPI, and GraphQL sources into one schema. Good for legacy bridges.
- **Hive** by The Guild — schema registry + composition + monitoring; works with vendor-neutral federation.
- **WunderGraph** — alternative federation runtime.
- **Hot Chocolate Fusion** — federation in the .NET ecosystem.

The federation specification has been opened up so multiple gateway vendors can implement it. The original is Apollo's, but the spec is portable.

### Stitching vs Federation in one breath

Stitching is "throw schemas in a blender, hope it works." Federation is "every subgraph declares its entities and relationships; the router uses that to plan queries." Federation is the modern answer.

## Subscriptions Transport

### Why subscriptions need a different transport

Queries and mutations are request/response. Subscriptions need to push data over time. HTTP doesn't naturally support that without polling. So GraphQL subscriptions ride one of two transports.

### WebSocket via `graphql-ws`

A WebSocket is an HTTP upgrade that converts a normal HTTP connection into a long-lived bidirectional channel. The browser opens a connection and keeps it open. The server can push messages whenever it wants.

The WebSocket protocol for GraphQL is called **`graphql-transport-ws`** (the protocol name on the wire), and the most popular library implementing it is also called **`graphql-ws`**. They are NOT the same as the older `subscriptions-transport-ws` (SUBT-WS), which is deprecated and has security and protocol bugs. **Use `graphql-ws` for new work; only use `subscriptions-transport-ws` to maintain old clients.**

The `graphql-ws` connection lifecycle:

```
SUBSCRIPTION WEBSOCKET LIFECYCLE
================================

  Client                                   Server
    |                                        |
    |  HTTP GET /graphql                     |
    |  Upgrade: websocket                    |
    |  Sec-WebSocket-Protocol:               |
    |       graphql-transport-ws             |
    |--------------------------------------->|
    |                                        |
    |  HTTP 101 Switching Protocols          |
    |<---------------------------------------|
    |                                        |
    |  ConnectionInit (with auth payload)    |
    |--------------------------------------->|
    |                                        |
    |  ConnectionAck                         |
    |<---------------------------------------|
    |                                        |
    |  Subscribe id=1                        |
    |   { messageAdded(roomId: "abc")        |
    |       { id text } }                    |
    |--------------------------------------->|
    |                                        |
    |  Next id=1   { messageAdded: {...}}    |
    |<---------------------------------------|
    |                                        |
    |  Next id=1   { messageAdded: {...}}    |
    |<---------------------------------------|
    |                                        |
    |  Next id=1   { messageAdded: {...}}    |
    |<---------------------------------------|
    |                                        |
    |  Complete id=1                         |
    |--------------------------------------->|
    |                                        |
    |  ConnectionClose                       |
    |<--------------------------------------->|
```

You speak it from the command line with `wscat`:

```bash
$ wscat -c 'wss://api.example.com/graphql' -H 'Sec-WebSocket-Protocol: graphql-transport-ws'
> {"type":"connection_init","payload":{}}
< {"type":"connection_ack"}
> {"id":"1","type":"subscribe","payload":{"query":"subscription { messageAdded(roomId:\"abc\") { id text } }"}}
< {"id":"1","type":"next","payload":{"data":{"messageAdded":{"id":"42","text":"hi"}}}}
< {"id":"1","type":"next","payload":{"data":{"messageAdded":{"id":"43","text":"there"}}}}
> {"id":"1","type":"complete"}
```

### Server-Sent Events via `graphql-sse`

SSE is a simpler alternative: it's a long-lived HTTP GET (or POST) that the server keeps open and writes events to as text. Browsers have a built-in `EventSource` API for it. It is one-directional (server -> client). For most subscription needs, that's plenty.

`graphql-sse` (from The Guild) is the standardized GraphQL-over-SSE library. It is stable and supported by GraphQL Yoga, Apollo Server, and Mercurius. SSE works through HTTP/2 multiplexing better than WebSockets do, and is friendlier to corporate firewalls.

### Pick one

For most apps, `graphql-ws` over WebSocket is the default. SSE is great if you don't need bidirectional, or if your hosting setup makes WebSockets painful (some serverless platforms).

## Tooling

GraphQL's tooling is one of its best features. Some highlights:

- **GraphiQL 2.x** — the original in-browser IDE. Open-source, embeddable, comes with most servers at `/graphql`. Modern version has tabs, persistence, and theme support.
- **Apollo Sandbox** — Apollo's hosted explorer at `studio.apollographql.com/sandbox`. Schema docs, query editor, history, and federation-aware.
- **GraphQL Playground (deprecated)** — older Apollo IDE. Use GraphiQL 2 or Sandbox instead.
- **Insomnia** — REST/GraphQL desktop client; great GraphQL support including variables and headers.
- **Altair** — open-source desktop GraphQL client. Excellent for subscriptions and file uploads.
- **GraphQL Voyager** — visualize the schema as an interactive graph diagram. Beautiful for onboarding.
- **graphql-doctor** — schema linter focused on best practices.
- **graphql-eslint** — ESLint plugin that lints both schemas and queries against your code.
- **graphql-schema-linter** — older but still useful schema linter.
- **GraphQL Code Generator** — generates TypeScript types, hooks, SDKs from your schema and operations. The `codegen.ts` config file. We'll see it in Hands-On.
- **graphql-faker** — mock a schema for early development.
- **GraphQL Inspector** — diff schemas, find breaking changes, validate operations against a schema.
- **graphql-config** — standardized way to point tools at your schema.
- **GraphQL Armor** — security middleware: cost limit, depth limit, alias limit, disable introspection, etc.

## Common Server Frameworks

You can pick from at least one mature framework in every major language.

- **Apollo Server (4.x)** — JavaScript/TypeScript. The flagship. Full rewrite in 2022 with `@apollo/server` package; runs as middleware on Express, Fastify, AWS Lambda, etc. Federation-ready.
- **GraphQL Yoga (3.x+)** — JavaScript/TypeScript by The Guild. Light, plugin-driven via Envelop, supports SSE and `@defer` natively.
- **Mercurius** — Fastify plugin, JavaScript/TypeScript. Famously fast.
- **Helix / Envelop** — building blocks for custom GraphQL HTTP setups.
- **express-graphql (deprecated)** — old Express middleware. Use Yoga or Apollo Server now.
- **Hot Chocolate** — .NET / C#. Excellent. Full federation support via Hot Chocolate Fusion.
- **Graphene** — Python. Long-standing.
- **Strawberry** — Python. Newer, type-hint-driven, modern.
- **Ariadne** — Python. Schema-first.
- **gqlgen** — Go. Code-generated from schema. Performance-friendly.
- **graphql-go** — Go. Hand-written resolvers.
- **Juniper** — Rust. Sync.
- **async-graphql** — Rust. Async-first, very fast.
- **Sangria** — Scala.
- **graphql-ruby** — Ruby. Mature.

There are also "instant GraphQL" servers that build a schema for you from a database:

- **Hasura** — gives you a GraphQL API on top of PostgreSQL/MySQL/etc with row-level permissions, in minutes.
- **PostGraphile** — same idea, PostgreSQL-only, Node-based.
- **Dgraph** — a graph database that speaks GraphQL natively.
- **Fauna** — managed serverless database with GraphQL.

## Common Client Libraries

- **Apollo Client** — JavaScript/TypeScript. Has the best React integration. Normalized cache. Full federation support. Subscription support via `graphql-ws`.
- **Relay** — Facebook's client. Heavier, opinionated, very fast. Requires schema follows the Relay connection spec (cursors, edges, nodes).
- **urql** — JavaScript/TypeScript. Light, plugin-based. Graphcache plugin gives Apollo-like normalization.
- **graphql-request** — micro-library: just a `request()` function that sends a query and returns data. Perfect for scripts and serverless.
- **swr + graphql-request** or **react-query + graphql-request** — popular pairing for "I just want data fetching, not a full GraphQL stack."
- **Apollo iOS** — native iOS client.
- **Apollo Android** (now **Apollo Kotlin**) — Kotlin/Java client.

## Common Errors

Here are errors you will see often, verbatim, with the canonical fix.

1. **`Cannot query field "X" on type "Y".`**
   You typed a field name the schema doesn't have. Check spelling. Run an introspection. Look at the docs in GraphiQL.

2. **`Field "X" of type "Y!" must have a sub selection.`**
   You asked for an object-type field without saying which fields you want. Add `{ ... }` and pick fields. `user` -> `user { name }`.

3. **`Variable "$X" of required type "Y!" was not provided.`**
   The query says `($id: ID!)` but you didn't pass `variables: { id: ... }`. Add the variable to the request body's `variables` field.

4. **`Variable "$X" got invalid value Z; Expected type "Y".`**
   The variable's value didn't match the expected type. `ID!` expected a string, you sent a number. Coerce or fix.

5. **`Syntax Error: Expected Name, found ...`**
   Malformed GraphQL query. Often a missing brace or comma. Pretty-print and look for the unbalanced bracket.

6. **`FetchError: failed to fetch`** (or **`Network error`**)
   The HTTP request didn't reach the server. Wrong URL, server down, CORS blocked, TLS error.

7. **`Persisted query not found`**
   The server doesn't have that hash on file. Client should retry with the full `query` text plus the hash so the server can register it.

8. **`Operation must include a name when there are multiple operations.`**
   Your document has more than one `query`/`mutation`/`subscription`. Either name them and pick one with `operationName`, or send only one.

9. **`Query depth limit exceeded.`**
   Your nesting is too deep. Either flatten the query or raise the depth limit on the server. Triggered by `graphql-armor` or `graphql-depth-limit`.

10. **`Cannot return null for non-nullable field X.Y.`**
    A resolver returned null for a field declared `!`. Either fix the resolver to return a value, or change the schema to make the field nullable.

11. **`Subscription is not supported.`**
    The transport you used doesn't accept subscriptions, or the server doesn't have the subscription transport enabled. Switch to a WebSocket connection or enable `graphql-sse`.

12. **`Schema validation failed: type X is missing field Y.`**
    Composition failure (often in federation). One subgraph references a field of another subgraph that doesn't exist. Re-publish the changed subgraph.

13. **`Anonymous queries are not allowed.`**
    Some servers (especially APQ-strict ones) require you to name every operation. Add `query MyName { ... }`.

14. **`PERSISTED_QUERY_NOT_SUPPORTED`**
    Server has APQ disabled. Disable APQ on the client or enable on the server.

15. **`UNAUTHENTICATED`** / **`FORBIDDEN`**
    No or wrong credentials. Re-auth or check the user's roles.

## Hands-On

You will need:

- `curl` (every Mac/Linux machine has it).
- A GraphQL server you can hit. For these examples, we'll show outputs from a hypothetical `https://countries.trevorblades.com/` (a public GraphQL service) and a local `http://localhost:4000/graphql` you would run yourself.
- Optional: `node` and `npm` for client tools.
- Optional: `wscat` (`npm install -g wscat`) for WebSocket subscriptions.

### 1. Hello, GraphQL — your first query with curl

```bash
$ curl -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"{ continent(code: \"EU\") { name countries { name } } }"}'
{"data":{"continent":{"name":"Europe","countries":[{"name":"Andorra"},{"name":"Albania"},{"name":"Austria"},{"name":"Aland Islands"},{"name":"Bosnia and Herzegovina"},{"name":"Belgium"},{"name":"Bulgaria"},{"name":"Belarus"},{"name":"Switzerland"},{"name":"Czechia"},{"name":"Germany"},{"name":"Denmark"},{"name":"Estonia"},{"name":"Spain"},{"name":"Finland"},{"name":"Faroe Islands"},{"name":"France"},{"name":"United Kingdom"},{"name":"Guernsey"},{"name":"Greece"},{"name":"Croatia"},{"name":"Hungary"},{"name":"Ireland"},{"name":"Isle of Man"},{"name":"Iceland"},{"name":"Italy"},{"name":"Jersey"},{"name":"Liechtenstein"},{"name":"Lithuania"},{"name":"Luxembourg"},{"name":"Latvia"},{"name":"Monaco"},{"name":"Moldova"},{"name":"Montenegro"},{"name":"North Macedonia"},{"name":"Malta"},{"name":"Netherlands"},{"name":"Norway"},{"name":"Poland"},{"name":"Portugal"},{"name":"Romania"},{"name":"Serbia"},{"name":"Russia"},{"name":"Sweden"},{"name":"Slovenia"},{"name":"Svalbard and Jan Mayen"},{"name":"Slovakia"},{"name":"San Marino"},{"name":"Turkey"},{"name":"Ukraine"},{"name":"Holy See (Vatican City State)"}]}}}
```

You sent JSON containing one field, `query`. The server returned JSON wrapping a `data` object with the same shape as your query.

### 2. Variables — same query, parameterized

```bash
$ curl -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"query GetContinent($code: ID!) { continent(code: $code) { name } }","variables":{"code":"AS"}}'
{"data":{"continent":{"name":"Asia"}}}
```

You named the operation `GetContinent`, declared one variable `$code`, and passed `{"code":"AS"}` in `variables`.

### 3. Pretty-print with jq

```bash
$ curl -s -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"{ country(code: \"JP\") { name capital currency } }"}' | jq
{
  "data": {
    "country": {
      "name": "Japan",
      "capital": "Tokyo",
      "currency": "JPY"
    }
  }
}
```

`jq` is a tiny JSON tool. `jq` with no argument pretty-prints. `jq '.data.country.name'` extracts a single field.

### 4. Aliases in one request

```bash
$ curl -s -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"{ japan: country(code:\"JP\"){ name } france: country(code:\"FR\"){ name } }"}' | jq
{
  "data": {
    "japan":  { "name": "Japan" },
    "france": { "name": "France" }
  }
}
```

Two `country` calls in one request, distinguished by the aliases `japan` and `france`.

### 5. Introspection — list all type names

```bash
$ curl -s -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"{ __schema { types { name } } }"}' | jq '.data.__schema.types[].name' | head -10
"Continent"
"ContinentFilterInput"
"StringQueryOperatorInput"
"Country"
"CountryFilterInput"
"Language"
"LanguageFilterInput"
"State"
"Subdivision"
"Query"
```

The double underscore `__schema` is the introspection root. **Disable introspection in production** to avoid leaking your schema.

### 6. Introspect one type

```bash
$ curl -s -H 'Content-Type: application/json' -X POST https://countries.trevorblades.com/graphql \
       -d '{"query":"{ __type(name: \"Country\") { fields { name type { name } } } }"}' | jq '.data.__type.fields[] | "\(.name): \(.type.name)"' | head -10
"code: null"
"name: null"
"native: null"
"phone: null"
"continent: null"
"capital: null"
"currency: null"
"languages: null"
"emoji: null"
"emojiU: null"
```

(`null` here just means the type is wrapped — the underlying name lives one level deeper. `code: ID!` shows up wrapped in a NonNull/Named pair.)

### 7. Stand up a local server with Apollo Server

```bash
$ mkdir gql-hello && cd gql-hello
$ npm init -y > /dev/null
$ npm install @apollo/server graphql > /dev/null

added 51 packages in 2s
```

Save this as `index.mjs`:

```javascript
import { ApolloServer } from '@apollo/server'
import { startStandaloneServer } from '@apollo/server/standalone'

const typeDefs = `#graphql
  type Query { hello(name: String = "world"): String! }
`
const resolvers = {
  Query: { hello: (_, { name }) => `hello, ${name}!` }
}

const server = new ApolloServer({ typeDefs, resolvers })
const { url } = await startStandaloneServer(server, { listen: { port: 4000 } })
console.log(`Ready at ${url}`)
```

```bash
$ node index.mjs
Ready at http://localhost:4000/
```

In another terminal:

```bash
$ curl -s -H 'Content-Type: application/json' -X POST http://localhost:4000/ \
       -d '{"query":"{ hello(name: \"Alice\") }"}' | jq
{ "data": { "hello": "hello, Alice!" } }
```

### 8. Stand up GraphQL Yoga (alternative)

```bash
$ npm install graphql-yoga graphql > /dev/null
```

```javascript
// yoga.mjs
import { createYoga, createSchema } from 'graphql-yoga'
import { createServer } from 'http'

const yoga = createYoga({
  schema: createSchema({
    typeDefs: `type Query { hello: String! }`,
    resolvers: { Query: { hello: () => 'hello yoga!' } }
  })
})

createServer(yoga).listen(4000, () => console.log('http://localhost:4000/graphql'))
```

```bash
$ node yoga.mjs
http://localhost:4000/graphql
```

### 9. Mutation example

Schema:

```graphql
type Query { hello: String! }
type Mutation { addUser(name: String!): User! }
type User { id: ID! name: String! }
```

```bash
$ curl -s -H 'Content-Type: application/json' -X POST http://localhost:4000/ \
       -d '{"query":"mutation { addUser(name: \"Alice\") { id name } }"}' | jq
{ "data": { "addUser": { "id": "1", "name": "Alice" } } }
```

### 10. Subscription with wscat

```bash
$ wscat -c 'ws://localhost:4000/graphql' -H 'Sec-WebSocket-Protocol: graphql-transport-ws'
Connected (press CTRL+C to quit)
> {"type":"connection_init"}
< {"type":"connection_ack"}
> {"id":"1","type":"subscribe","payload":{"query":"subscription { tick }"}}
< {"id":"1","type":"next","payload":{"data":{"tick":1}}}
< {"id":"1","type":"next","payload":{"data":{"tick":2}}}
< {"id":"1","type":"next","payload":{"data":{"tick":3}}}
```

`wscat` is a tiny WebSocket REPL. `npm install -g wscat`.

### 11. Persisted query example (APQ on Apollo)

```bash
# First request: client computes hash, sends both query and hash.
$ curl -s -X POST http://localhost:4000/ \
       -H 'Content-Type: application/json' \
       -d '{"query":"{ hello }","extensions":{"persistedQuery":{"version":1,"sha256Hash":"3b8a7..."}}}' | jq
{ "data": { "hello": "hello, world!" } }

# Subsequent requests: only hash, no query text.
$ curl -s -X POST http://localhost:4000/ \
       -H 'Content-Type: application/json' \
       -d '{"extensions":{"persistedQuery":{"version":1,"sha256Hash":"3b8a7..."}}}' | jq
{ "data": { "hello": "hello, world!" } }
```

If the hash isn't registered, the server returns `Persisted query not found`.

### 12. Health check

```bash
$ curl -s http://localhost:4000/.well-known/apollo/server-health
{"status":"pass"}
```

Apollo Server exposes a health endpoint at `/.well-known/apollo/server-health` you can hit from a load balancer.

### 13. Schema download with apollo CLI

```bash
$ npm install -g @apollo/cli
$ apollo schema:download --endpoint http://localhost:4000/ schema.json
loading from http://localhost:4000/
✔ Saving schema to schema.json
```

Now `schema.json` has the introspected schema. Useful for offline tooling.

### 14. Generate TypeScript types with GraphQL Code Generator

```bash
$ npm install -D @graphql-codegen/cli @graphql-codegen/typescript > /dev/null
```

`codegen.ts`:

```typescript
import type { CodegenConfig } from '@graphql-codegen/cli'
const config: CodegenConfig = {
  schema: 'http://localhost:4000/',
  generates: {
    './src/gql.ts': { plugins: ['typescript'] }
  }
}
export default config
```

```bash
$ npx graphql-codegen --config codegen.ts
✔ Parse Configuration
✔ Generate outputs
$ head -10 src/gql.ts
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
...
```

### 15. Schema diff / breaking change detection

```bash
$ npx graphql-inspector diff old-schema.graphql new-schema.graphql
✔ No breaking changes detected
✔ Detected 2 changes
  Field 'User.fullName' was added
  Field 'User.legacyName' was deprecated
```

Run this in CI before merging schema changes.

### 16. Validate a query against a schema

```bash
$ npx graphql-inspector validate 'src/**/*.graphql' http://localhost:4000/
Validation: PASSED
2 documents validated against schema
```

Catches `Cannot query field` errors at build time, not at run time.

### 17. apollo schema:check (registry mode)

```bash
$ apollo schema:check --endpoint=http://localhost:4000/
✔ Loaded schema
ℹ Validating local schema against published schema in studio
✔ No breaking changes found
```

Used in CI to gate merges with Apollo Studio.

### 18. apollo client:codegen

```bash
$ apollo client:codegen --target=typescript --localSchemaFile=schema.json src/operations
✔ Loaded schema
✔ Generated types for 3 operations
```

### 19. apollo client:check

```bash
$ apollo client:check --localSchemaFile=schema.json --queries='src/**/*.gql'
✔ Operations are compatible with the schema
```

### 20. mercurius starter (Fastify)

```bash
$ npm install fastify mercurius graphql > /dev/null
```

```javascript
import Fastify from 'fastify'
import mercurius from 'mercurius'

const app = Fastify()
app.register(mercurius, {
  schema: `type Query { hello: String! }`,
  resolvers: { Query: { hello: async () => 'mercurius hi' } },
  graphiql: true
})
await app.listen({ port: 4000 })
```

### 21. gqlgen (Go)

```bash
$ go install github.com/99designs/gqlgen@latest
$ gqlgen init
✔ Created gqlgen.yml
✔ Created schema.graphqls
✔ Generated server.go
$ go run server.go
2026/04/27 12:00:00 connect to http://localhost:8080/ for GraphQL playground
```

### 22. strawberry codegen (Python)

```bash
$ pip install strawberry-graphql[debug-server]
$ strawberry server schema:schema
[INFO] - Running strawberry on http://0.0.0.0:8000/graphql
```

### 23. hasura console

```bash
$ hasura console
INFO console running at: http://localhost:9695/
```

Hasura builds a GraphQL API from your Postgres schema in minutes.

### 24. postgraphile

```bash
$ npx postgraphile -c postgres://localhost/mydb --watch --enhance-graphiql
postgraphile v4.x server listening on port 5000
```

### 25. dgraph live load

```bash
$ dgraph live --files data.json --schema schema.graphql --alpha localhost:9080
[Edge - 0d 0h 0m 1s] Txns: 1 N-Quads: 100 N-Quads/s [last 5s]: 100 Aborts: 0
```

### 26. Kubernetes — port forward to a GraphQL service

```bash
$ kubectl port-forward svc/graphql 4000:4000
Forwarding from 127.0.0.1:4000 -> 4000
Forwarding from [::1]:4000 -> 4000
```

### 27. Tail the logs of a GraphQL deployment

```bash
$ kubectl logs -l app=graphql -f
[2026-04-27T12:00:00Z] POST /graphql 200 28ms { user(id: "7") { name } }
[2026-04-27T12:00:01Z] POST /graphql 200 14ms { hello }
```

### 28. Compose REST + GraphQL with GraphQL Mesh

```bash
$ npm install @graphql-mesh/cli @graphql-mesh/openapi @graphql-mesh/json-schema > /dev/null
```

`.meshrc.yaml`:

```yaml
sources:
  - name: PetStore
    handler:
      openapi:
        source: https://petstore.swagger.io/v2/swagger.json
```

```bash
$ npx mesh dev
🕸️  Mesh - PetStore Loading sources...
🕸️  Mesh - Server Starting GraphQLServer on port 4000
```

Mesh just turned a REST API into GraphQL.

### 29. Install GraphQL Armor (security middleware)

```bash
$ npm install @escape.tech/graphql-armor > /dev/null
```

```javascript
import { ApolloArmor } from '@escape.tech/graphql-armor'
const armor = new ApolloArmor({
  costLimit:    { maxCost: 5000 },
  maxDepth:     { n: 10 },
  maxAliases:   { n: 15 },
  maxDirectives:{ n: 50 },
  blockFieldSuggestion: { enabled: true },
  introspectionDisable: { enabled: process.env.NODE_ENV === 'production' }
})
const server = new ApolloServer({ typeDefs, resolvers, ...armor.protection })
```

### 30. Lint queries with graphql-eslint

```bash
$ npm install -D @graphql-eslint/eslint-plugin > /dev/null
$ npx eslint --ext .graphql src/
src/queries/user.graphql
  3:5  warning  Field 'fullName' is deprecated  @graphql-eslint/no-deprecated
✖ 1 problem (0 errors, 1 warning)
```

### 31. Test an Apollo Server with vitest/jest

```javascript
import { ApolloServer } from '@apollo/server'

test('hello', async () => {
  const server = new ApolloServer({ typeDefs, resolvers })
  const r = await server.executeOperation({ query: '{ hello }' })
  expect(r.body.singleResult.data).toEqual({ hello: 'hello, world!' })
})
```

```bash
$ npx vitest run
✓ hello (12ms)
1 passed
```

### 32. Schema linter

```bash
$ npx graphql-schema-linter schema.graphql
schema.graphql
  3:5  type-description-required  Type 'User' is missing a description
  ...
```

### 33. Visualize with GraphQL Voyager

```bash
$ npx graphql-voyager --schema-file schema.graphql
Voyager listening on http://localhost:8000
```

Open in a browser to see your schema as an interactive web of types.

## Common Confusions

1. **GraphQL is NOT a database.** The schema describes shapes the resolvers fetch. The actual data still lives in Postgres, MongoDB, Redis, S3, other APIs. The schema is a "what's available" map; the resolvers are the "how to get it."

2. **Subscription transport is over-the-wire WebSocket BUT the server library matters.** Two servers can both speak `graphql-transport-ws`, yet one might use `graphql-ws` and the other `subscriptions-transport-ws` (deprecated). The wire protocol name and the library name are both "graphql-ws" — confusing! Pin down which one your client and server speak before you debug.

3. **The N+1 problem and how DataLoader hides it.** Without DataLoader, fetching a list of N parents and their children causes 1+N database queries. DataLoader batches them into 2 (one for parents, one for children). Resolvers don't know they're being batched — that's the magic.

4. **Query batching at HTTP level vs at resolver level.** Two different things. HTTP batching is "send 5 queries in one HTTP request" (Apollo Server supports this with an array request body). Resolver batching is what DataLoader does. They're complementary.

5. **Persisted queries are NOT for security alone — they reduce request size.** APQ saves bandwidth, especially on mobile. Trusted-document mode (where the server only accepts known hashes) adds a security layer on top.

6. **Introspection in production is a security smell — disable it.** Introspection lets anyone download your full schema. In production, attackers can use it to find unused fields, deprecated paths, internal types you forgot to hide. Set the server to disable introspection in production.

7. **`@defer` for streaming is part of the June 2023 incremental delivery RFC.** Not in the 2018 or 2021 stable specs. Apollo Server 4.x and Yoga 3.x support it. Older clients ignore it.

8. **Mutations should still be POST, not GET.** Idempotency is a server contract. The HTTP method is always POST for mutations; the keyword `mutation` in the body is what tells the server it's a write. Sending a mutation as GET breaks HTTP semantics and most servers refuse.

9. **How variables differ from string interpolation.** Variables are typed and parsed separately. They are NOT glued into the query string. The server sees `$id` as a placeholder and substitutes the value at execution time. There is no GraphQL injection possible the way SQL injection is possible — as long as you use variables, not string concat.

10. **What does `!` mean.** Non-null. The server promises the field will never be null. If a resolver returns null for a `!` field, GraphQL throws and the error bubbles up. `!` is a contract; honor it.

11. **The SDL `Query` and `Mutation` are TYPES, not magic.** `type Query { ... }` is an ordinary object type that the GraphQL server has been told to use as the entry point. You could rename it to anything — though don't, because tools assume the conventional names.

12. **Where do auth checks belong?** In the **context** (parse the bearer token), then **resolvers** (verify role) — and **directives** can declaratively enforce. NOT in the SDL alone. The SDL is a description; only code can enforce.

13. **Fragments are for deduplication, not abstraction.** A fragment shares a chunk of fields between two queries. It does not hide data or polyfill behavior. The server sees the expanded query as if you wrote the fields out longhand.

14. **Cursor-based vs offset pagination.** Offset (`limit/offset`) is simple but breaks when items are inserted/deleted between requests. Cursor (`first/after`) returns a stable pointer per item; new items at the start don't shift your page. Relay's connection spec mandates cursors and is the GraphQL standard.

15. **What does Relay mean — both a client AND a spec.** "Relay" is Facebook's React GraphQL client. "Relay spec" is the contract (Connections, Edges, Nodes, PageInfo) that any GraphQL server can follow to be Relay-compatible. You can use the spec without using the client.

16. **APQ is automatic; trusted documents are stricter.** APQ negotiates the hash on first request. Trusted documents pre-register the hashes; unknown hashes are rejected.

17. **GraphQL doesn't require POST.** Servers can also accept GET for queries (with the query as a URL parameter). Most teams use POST anyway because URL length limits hurt long queries.

18. **The "graph" doesn't have to be cyclic.** GraphQL the language can describe trees, lattices, or arbitrary cyclic graphs. Resolvers handle the walk. Some servers detect cyclic queries and limit them to prevent infinite recursion.

19. **One endpoint vs one operation.** GraphQL has one endpoint (`/graphql`), but multiple operation types (Query / Mutation / Subscription). They all hit the same URL.

20. **Schema-first vs code-first.** Schema-first writes the SDL by hand; resolvers attach later. Code-first (TypeGraphQL, Strawberry) writes types in code; SDL is generated. Both are valid; pick one and stick with it.

## Vocabulary

(120+ terms, alphabetized loosely by topic.)

| Term | Meaning |
|------|---------|
| GraphQL | A query language and runtime for APIs that lets clients ask for exactly the data they need from a single endpoint over HTTP. |
| GraphQL Foundation | The Linux Foundation project (since 2018) that stewards the GraphQL spec and ecosystem. |
| schema | The full description of a GraphQL API: every type, every field, every argument, every directive. |
| SDL (Schema Definition Language) | The text format used to write schemas: `type User { ... }`. |
| type | A named shape with fields. The SDL has six kinds of types. |
| object type | A type with fields that can return scalars or other object types. `type User`. |
| scalar | A leaf type holding a single value: Int, Float, String, Boolean, ID, or custom. |
| Int | 32-bit signed integer scalar. |
| Float | Double-precision floating point scalar. |
| String | UTF-8 string scalar. |
| Boolean | true/false scalar. |
| ID | A unique-identifier scalar; serialized as a string but means "an identifier." |
| custom scalar | A user-defined scalar like `DateTime`, `UUID`, or `JSON`, with server code that parses and serializes. |
| interface | A contract a type can implement: "anything Named has id and name." |
| union | A type that's "one of these types," with no shared fields required. |
| input type | A type used only as an argument; cannot be returned. Use the `input` keyword. |
| enum | A fixed list of allowed string values: `enum Status { PENDING SHIPPED }`. |
| list type `[T]` | A list (array) of T values. |
| non-null type `T!` | A T that the server promises is never null. |
| field | A named member of a type, returning some type. |
| argument | A named parameter on a field: `user(id: ID!)`. |
| directive | An annotation prefixed with `@` that changes execution. |
| `@skip` | Built-in directive: skip this field if the condition is true. |
| `@include` | Built-in directive: include this field only if the condition is true. |
| `@deprecated` | Marks a field or enum value as deprecated; tools warn. |
| `@specifiedBy` | Points custom-scalar consumers at the formal spec URL. |
| `@oneOf` | Marks input types where exactly one field must be set (working draft, gaining adoption). |
| custom directive | A user-defined directive, with server code that interprets it. |
| query | An operation type for reading data. |
| mutation | An operation type for changing data. |
| subscription | An operation type for receiving pushed data over time. |
| root types | Query, Mutation, Subscription — the three entry points. |
| resolver | A server-side function that returns the value for a field. |
| field resolver | A resolver for a specific field of a specific type. |
| parent argument | The first resolver argument; the value returned by the parent field's resolver. |
| args argument | The second resolver argument; the field's arguments. |
| context argument | The third resolver argument; per-request shared state (db, user, loggers). |
| info argument | The fourth resolver argument; metadata about the field, including the parsed selection set. |
| executable schema | A schema that has been linked with resolvers and can run queries. |
| `makeExecutableSchema` | A function (graphql-tools) that combines typeDefs + resolvers into an executable schema. |
| graphql-js | The reference JavaScript implementation of the GraphQL spec. |
| graphql-tools | A toolkit for building GraphQL schemas and stitching schemas together. |
| schema stitching | The legacy way to combine multiple schemas into one; superseded by federation. |
| Apollo Federation v1 | First-generation federation spec from Apollo (2019); now legacy. |
| Federation v2 | Current federation spec (GA 2022); rewritten architecture, `@shareable`, `@inaccessible`, etc. |
| subgraph | A GraphQL service that's part of a federated supergraph. |
| supergraph | The composed schema across all subgraphs. |
| gateway | The runtime that routes federated requests; nowadays usually the Apollo Router. |
| router | A federation gateway. The Apollo Router is written in Rust for speed. |
| Apollo Router | Apollo's Rust-based federation gateway, replacing the older JS gateway. |
| entity | A type identified by a `@key` directive that can be referenced across subgraphs. |
| `@key` | Federation directive that declares the key fields of an entity. |
| `@external` | Federation directive: this field is owned by another subgraph. |
| `@requires` | Federation directive: this field needs other fields fetched first. |
| `@provides` | Federation directive: this subgraph can supply these fields without a roundtrip. |
| `@shareable` | Federation directive: more than one subgraph defines this field. |
| `@inaccessible` | Federation directive: hide this field from the public schema. |
| `@tag` | Federation directive: add a tag for routing or governance. |
| federation 2.x | Current federation generation, with the directives above. |
| schema composition | The build step that combines subgraphs into a supergraph. |
| schema design | The art of designing types/fields for evolution and consistency. |
| Relay-style pagination | The Relay connection spec: `edges`, `node`, `cursor`, `pageInfo`. |
| edges | The array of connection edges, each with a `node` and a `cursor`. |
| node | The actual entity inside an edge. |
| cursor | An opaque string pointer to a position in a connection. |
| pageInfo | A shape with `hasNextPage`, `hasPreviousPage`, `startCursor`, `endCursor`. |
| hasNextPage | Boolean: is there a page after this one? |
| hasPreviousPage | Boolean: is there a page before this one? |
| connection | A field shape that wraps `edges` and `pageInfo`. |
| after / before / first / last | The four standard pagination arguments in the Relay spec. |
| offset / limit | Alternative pagination using a numeric offset. Simpler, less stable under writes. |
| query complexity | A score per query measuring how expensive it is. Used to rate-limit attackers. |
| query depth | How deeply nested a query is. Used to prevent recursion attacks. |
| query batching | Sending multiple queries in one request body, at the HTTP level. |
| persisted queries | Queries pre-registered by hash; clients send the hash instead of the full text. |
| APQ (Automatic Persisted Queries) | Apollo's negotiation: clients send hash; if unknown, retry with full query. |
| trusted documents | Stricter mode: server only accepts known hashes; rejects all unknown. |
| query whitelist | Older name for trusted documents. |
| response cache | Server-side cache of responses keyed by operation. Used by Stellate / Apollo Router. |
| normalized cache | Client-side cache that splits objects by ID and stores them flatly. Apollo InMemoryCache, Relay store. |
| partial response | A response that has both `data` and `errors` — some fields succeeded, some failed. |
| errors array | The top-level `errors` field of a response. |
| error extensions | Custom metadata on an error: `code`, etc. |
| error path | The JSON path to the field that errored. |
| locations | Line/column in the query string where the error originated. |
| live queries | A push pattern where any change re-runs the query. Less standardized than subscriptions. |
| subscriptions | The push operation type. |
| graphql-ws | The modern subscription transport library and protocol. |
| graphql-sse | Subscription transport over Server-Sent Events. |
| graphql-transport-ws | The wire-protocol name for `graphql-ws`. |
| websocket protocol | The HTTP upgrade that turns a connection into a long-lived bidirectional channel. |
| server-sent events | A long-lived HTTP response that streams events. |
| multipart upload spec | The community spec for `multipart/form-data` file uploads to GraphQL. |
| file uploads | Sending files alongside a GraphQL request, usually via the multipart spec. |
| scalar Upload | The custom scalar used by upload-capable servers. |
| Apollo Server | Reference JavaScript GraphQL server (4.x as of 2022 rewrite). |
| GraphQL Yoga | Lightweight JS server by The Guild. |
| express-graphql (deprecated) | Older Express-only middleware. Use Yoga or Apollo Server. |
| Mercurius | Fastify GraphQL plugin. |
| Helix | A toolkit for building custom GraphQL HTTP layers. |
| Envelop | Plugin system for GraphQL servers (Yoga uses it). |
| ariadne | Python schema-first server. |
| strawberry | Python type-hint-first server. |
| graphene | Python class-based server. |
| gqlgen | Go code-generated server. |
| async-graphql | Async-first Rust server. |
| Juniper | Synchronous Rust server. |
| Hot Chocolate | .NET / C# server. |
| Sangria | Scala server. |
| Apollo Sandbox | Hosted browser GraphQL IDE at studio.apollographql.com/sandbox. |
| GraphiQL 2.x | Open-source in-browser IDE; modern rewrite of GraphiQL. |
| GraphQL Playground (deprecated) | Older Apollo IDE. Use Sandbox or GraphiQL 2. |
| Altair | Open-source desktop GraphQL client. |
| Insomnia GraphQL | GraphQL support inside Insomnia. |
| Postman GraphQL support | Postman's GraphQL panel. |
| GraphQL Voyager | Visualizes a schema as an interactive graph. |
| graphql-faker | Mocks a schema for early development. |
| graphql-doctor | Schema linter for best practices. |
| graphql-schema-linter | Older schema linter. |
| graphql-eslint | ESLint plugin that lints schemas and operations. |
| introspection | The GraphQL query that asks "what does this schema look like?" |
| `__schema` | The introspection root field. |
| `__type` | Introspect a single type. |
| `__typename` | Available on every object/interface/union; returns the runtime type name. |
| query introspection | The act of running `__schema` queries. |
| sdl-from-introspection | Reconstructing SDL by introspecting a server. |
| Hasura | Instant GraphQL server over Postgres/MySQL. |
| PostGraphile | Postgres-only instant GraphQL server. |
| Dgraph | Native graph database that speaks GraphQL. |
| Fauna | Managed serverless database with GraphQL. |
| Stellate | CDN for GraphQL responses; formerly GraphCDN. |
| GraphCDN (legacy) | Old name for Stellate. |
| Tyk Gateway | API gateway with native GraphQL support. |
| GraphOS | Apollo's hosted platform: schema registry, observability, federation. |
| Apollo Studio | Older name for parts of GraphOS. |
| Apollo schema registry | Central store of versioned schemas. |
| schema check | CI step that compares a proposed schema against the published one. |
| schema diff | Tool output showing additions, removals, deprecations between two schemas. |
| breaking change detection | Automated diffing for backwards-incompatible changes. |
| schema versioning vs evolution | Versioning means side-by-side `/v1`, `/v2`. Evolution means add-only changes. GraphQL prefers evolution. |
| GraphQL Armor | Security middleware: rate-limit, cost-limit, depth-limit, disable introspection. |
| QueryComplexity calculator | A library that scores each query's cost. |
| depth-limit middleware | A library that rejects queries deeper than N. |
| query.maxAliasCount | Limit on alias count per request, to block alias-overload attacks. |
| query.maxRootFields | Limit on top-level fields per request. |
| query.maxDirectives | Limit on directive count. |
| query.maxFieldsPerType | Limit on field count per type. |
| batching attacks | Abusing HTTP-level batching to exhaust the server. |
| query loop attacks | Recursive queries through interfaces or unions, intended to blow up cost. |
| alias overload | Sending the same expensive field many times under different aliases. |
| fragment cycles | Mutually-recursive fragment definitions; servers must detect and reject. |
| fragment-spread loops detection | Validation step that catches fragment cycles. |
| validation errors | Errors found during query validation, before execution. |
| syntax errors | Errors from parsing the query text. |
| lookup map for type resolution | A union/interface server hint that maps a value to its concrete type. |
| GraphQL Code Generator | Tool that generates types/hooks/SDKs from schema + operations. |
| codegen.ts / codegen.yml | Configuration files for GraphQL Code Generator. |
| generate types/hooks/SDK | What codegen produces for clients. |
| server-prepass codegen | Server-side codegen step (e.g. gqlgen, Hot Chocolate types). |
| GraphQL+REST hybrid | Architecture mixing GraphQL with classic REST endpoints. |
| JSON-API | A REST convention; a competitor to GraphQL for shape control. |
| OpenAPI | Schema spec for REST APIs; can be composed into GraphQL via Mesh. |
| gRPC | Google's RPC protocol; can be exposed as GraphQL via Mesh. |
| schema first | Write SDL by hand; bind resolvers later. |
| code first | Write types in code; generate SDL. |
| context-based auth | Auth done at the request boundary, set on context. |
| field-level auth | Per-field directive enforcement of auth. |
| trusted documents | (Repeat for emphasis.) The pre-registered persisted-query mode for security. |
| supergraph schema | The composed federation schema, including federation directives. |
| router config | YAML config for the Apollo Router (rate limits, plugins, etc.). |
| apollo-server-core (legacy) | Pre-4.x Apollo Server packaging. |
| `@apollo/server` (4.x) | Modern Apollo Server package. |
| `executableSchemaTransformer` | Function that wraps a schema with directive logic. |

## Try This

A short list of things to do once you've finished reading. None take more than a few minutes.

1. **Hit a public GraphQL server with curl.** Use `https://countries.trevorblades.com/graphql`. Send `{ continent(code: "EU") { name } }`. Read the response.
2. **Open GraphiQL.** Apollo Server starts GraphiQL at `http://localhost:4000/`. Type a query, hit Run, see the response.
3. **Run an introspection query.** `{ __schema { types { name } } }`. Read the type list. Pick one type and introspect it: `{ __type(name: "User") { fields { name } } }`.
4. **Add a custom scalar.** `scalar DateTime`. Add a field that uses it. Watch what happens if you forget to write the server-side parser/serializer.
5. **Make a non-null mistake on purpose.** Define a field as `String!`, then return `null` from the resolver. Watch the error bubble up.
6. **Add a DataLoader.** Take a "user.orders" resolver that fires one query per user. Wrap it in a DataLoader. Watch query count drop from N+1 to 2.
7. **Add `@auth` directive.** Define `directive @auth(role: Role!) on FIELD_DEFINITION`. Apply to one field. Write a directive transformer that throws when the user is not in role.
8. **Try a subscription with wscat.** Boot a Yoga server with a `tick` subscription. Connect with `wscat -c ws://localhost:4000/graphql -H 'Sec-WebSocket-Protocol: graphql-transport-ws'`. Send `connection_init`, then `subscribe`, watch the ticks.
9. **Run GraphQL Inspector against your schema.** `npx graphql-inspector validate 'src/**/*.graphql' http://localhost:4000/`.
10. **Disable introspection.** Set the server to disable introspection. Try the introspection query and see the error.

## Where to Go Next

You've now got the shape of GraphQL. Next stops:

- **`api/graphql`** — the deeper cheatsheet on GraphQL with more reference material.
- **`api/rest`** — the contrast point. Compare side-by-side.
- **`api/openapi`** — the schema spec for REST. Useful for hybrid setups.
- **`api/grpc`** — another typed RPC alternative. Different tradeoffs.
- **`networking/http`**, **`networking/http2`**, **`networking/http3`** — the transport GraphQL rides on.
- **`networking/websocket`** — the transport for subscriptions.
- **`ramp-up/websocket-eli5`** — easier intro to WebSockets.
- **`ramp-up/http3-quic-eli5`** — what HTTP/3 changes for any API.
- **`ramp-up/tls-eli5`**, **`ramp-up/tcp-eli5`** — the layers underneath.
- **`ramp-up/linux-kernel-eli5`** — if you want to keep going on the operating-system side.

## See Also

- [api/graphql](../api/graphql.md)
- [api/grpc](../api/grpc.md)
- [api/rest](../api/rest.md)
- [api/openapi](../api/openapi.md)
- [networking/http](../networking/http.md)
- [networking/http2](../networking/http2.md)
- [networking/http3](../networking/http3.md)
- [networking/websocket](../networking/websocket.md)
- [ramp-up/tcp-eli5](tcp-eli5.md)
- [ramp-up/tls-eli5](tls-eli5.md)
- [ramp-up/websocket-eli5](websocket-eli5.md)
- [ramp-up/http3-quic-eli5](http3-quic-eli5.md)
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md)

## References

- **GraphQL Specification, October 2021.** The current stable spec. <https://spec.graphql.org/October2021/>
- **GraphQL Specification, Working Draft.** Includes `@defer` / `@stream` / `@oneOf`. <https://spec.graphql.org/draft/>
- **GraphQL over HTTP draft.** Standardizing how GraphQL rides HTTP (status codes, content negotiation). <https://graphql.github.io/graphql-over-http/>
- **graphql-ws specification.** Subscription transport. <https://github.com/enisdenjo/graphql-ws>
- **graphql-sse specification.** SSE-based subscription transport. <https://github.com/enisdenjo/graphql-sse>
- **graphql.org** — official documentation, learning resources.
- **Apollo Documentation** — <https://www.apollographql.com/docs/> — Apollo Server, Apollo Client, Federation, Router.
- **Eve Porcello, "Learning GraphQL" (O'Reilly, 2018).** The friendliest book on the language.
- **Marc-André Giroux, "Production Ready GraphQL" (Self-published, 2020).** The deepest book on running GraphQL at scale.
- **The Guild blog** — <https://the-guild.dev/blog> — Yoga, Code Generator, Inspector, Hive, Mesh.
- **Apollo Federation 2 docs** — <https://www.apollographql.com/docs/federation/>
- **GraphQL Foundation** — <https://graphql.org/foundation/>
- **June 2023 Incremental Delivery RFC** — `@defer` / `@stream` proposal. <https://github.com/graphql/defer-stream-wg>
- **GraphQL Armor** — <https://github.com/Escape-Technologies/graphql-armor>

## Version Notes

- **2015** — GraphQL open-sourced by Facebook.
- **2018** — Spec moves to the GraphQL Foundation under the Linux Foundation.
- **June 2018** spec.
- **October 2021** spec — current stable.
- **Apollo Server 4 (2022)** — full rewrite; package renamed `@apollo/server`; old `apollo-server` deprecated.
- **Apollo Federation 2** — went GA in 2022; new directives (`@shareable`, `@inaccessible`); composition rewrites.
- **Apollo Router (Rust)** — replaces the older JavaScript gateway for federation.
- **graphql-ws 5+** — replaces deprecated `subscriptions-transport-ws`.
- **graphql-sse stable** — alternative subscription transport using Server-Sent Events.
- **June 2023 Incremental Delivery RFC** — `@defer` / `@stream` standardization phase.
- **Working Draft additions** — `@oneOf` directive for input choice (gaining adoption).
- **Persisted Queries** — APQ negotiation now standard in Apollo Client; trusted-document mode is the strict cousin.
- **GraphiQL 2.x** — modern fork; replaces GraphQL Playground (deprecated).
- **GraphQL Yoga 3+** — built on Envelop plugin model.
- **Mercurius** — current maintained Fastify plugin.
- **Hot Chocolate Fusion** — federation in the .NET ecosystem.
- **Stellate** — current name for what was GraphCDN.
- **Hive** — schema registry from The Guild, vendor-neutral federation alternative.

That's it. You can now read a GraphQL schema, write a query, send it with curl, run a server, fix an N+1 with DataLoader, talk about subscriptions, and reason about caching and federation. Welcome to the typed, client-driven, single-endpoint world.
