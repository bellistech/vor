# MongoDB — ELI5

> MongoDB is a giant filing cabinet of folders, where each folder holds little JSON-shaped index cards instead of stiff spreadsheet rows.

## Prerequisites

(helps to know what JSON is)

You do not need to be a database person. You do not need to know SQL. You do not need to have ever set up a server. You only need to know what JSON is, and even that we will explain in a minute.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

If you see a `>` at the start of a line in a code block, that means "type the rest of this line into the **mongosh** prompt." That is a different kind of terminal we will meet shortly. The `>` is the prompt that mongosh prints at you. You do not type the `>`.

If a word looks weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

### Quick refresher: what is JSON?

JSON stands for **JavaScript Object Notation.** It is a way to write down data using curly braces and key-value pairs. It looks like this:

```json
{
  "name": "alice",
  "age": 30,
  "pets": ["cat", "dog"],
  "address": {
    "city": "Portland",
    "zip": "97201"
  }
}
```

Read it like a recipe card:

- `name` is the ingredient name. `"alice"` is the value.
- `age` is the ingredient name. `30` is the value.
- `pets` is the ingredient name. The value is a **list** of things, written with square brackets.
- `address` is the ingredient name. The value is **another whole recipe card**, nested inside.

That is it. JSON is just a card with labels on it, and the labels can hold strings, numbers, lists, or whole other cards inside. If you can read the card above, you know enough JSON to use MongoDB.

## What Even Is MongoDB / NoSQL Document DB

### The filing cabinet picture

Imagine a big metal filing cabinet in an office. The cabinet has drawers. Each drawer has a label like "USERS" or "ORDERS" or "PRODUCTS."

Open one drawer. Inside the drawer there are folders. Each folder is the same shape — a manila folder. But what is **inside** each folder is allowed to look different.

One folder might hold an index card that says:

```json
{ "name": "alice", "age": 30, "city": "Portland" }
```

Another folder in the **same drawer** might hold an index card that says:

```json
{ "name": "bob", "email": "bob@example.com", "joined": "2024-01-15" }
```

Notice: alice has an `age` and a `city`. Bob does not. Bob has an `email` and a `joined`. Alice does not.

In a regular spreadsheet (or a regular database), this would be illegal. Every row in a spreadsheet must have the **same columns** in the **same order.** If alice has an `age`, every row needs an `age`, even if it is empty. If bob has an `email`, every row needs an `email`.

MongoDB does not care. MongoDB lets every folder hold a card that looks however it wants. **That is the whole pitch.** That is the one big idea that makes MongoDB different from a spreadsheet-style database.

### The index card picture

Each card inside a folder is called a **document.**

Each drawer of folders is called a **collection.**

The whole cabinet is called a **database.**

So when somebody says "I have a MongoDB database with a `users` collection that holds 100 documents," what they mean in plain English is:

- There is a filing cabinet.
- One drawer is labeled `users`.
- Inside that drawer there are 100 cards.
- Each card has whatever fields make sense for that user.

### The "NoSQL" label

You will see MongoDB called a **NoSQL** database. That word is a tiny bit misleading. It does not mean "no structured query language at all." It means "not a SQL database in the traditional row-and-column sense." MongoDB has its own query language, which we will meet. The query language is JSON-shaped, just like the data.

There are several flavors of NoSQL: document databases (MongoDB), key-value stores (Redis), wide-column stores (Cassandra), and graph databases (Neo4j). MongoDB is the **document** kind. The "document" here is the JSON-shaped index card we just talked about.

### Why would I want this?

Three reasons people pick MongoDB:

1. **The shape of my data changes a lot.** New fields show up every week. Old fields go away. I do not want to alter a giant table every time.
2. **My data is naturally tree-shaped.** A user has an address. The address has a city and a zip. The user also has a list of orders. Each order has a list of items. Each item has a price. In a SQL database I would need five tables and four joins to read all of that back. In MongoDB I can put the whole tree on one card.
3. **I want to scale across many machines easily.** MongoDB is designed from the ground up to spread one logical collection across many physical servers. We will meet that idea later under "Sharding."

### Why might I not want this?

Two reasons people stay away from MongoDB:

1. **My data is highly relational.** If almost everything I do is "join table A to table B to table C," a SQL database is going to feel more natural.
2. **I need rock-solid multi-row transactions across many tables.** MongoDB has transactions now, but they have rules and limits we will cover. SQL databases were built around this from day one.

There is a long-running internet debate about which approach is "better." The honest answer is: **neither is universally better. They solve different problems.** Many large systems use both — MongoDB for the part that changes shape a lot, SQL for the part that needs rigid relationships.

## vs Relational

Let's put them side by side.

### A SQL (relational) database looks like this

```
+------------------+      +------------------+
|     users        |      |     orders       |
+------------------+      +------------------+
| id  | name | age |      | id | user_id | $ |
|-----|------|-----|      |----|---------|---|
| 1   | alice| 30  |      | 1  | 1       | 9 |
| 2   | bob  | 25  |      | 2  | 1       |12 |
+------------------+      | 3  | 2       | 7 |
                          +------------------+
```

Two stiff grids. Every row in `users` has exactly the same three columns. Every row in `orders` has exactly the same three columns. To get "alice's orders" you have to **join** the two tables on `user_id = id`.

### A MongoDB collection looks like this

```
collection: users
+----------------------------------------+
| { _id: 1, name: "alice", age: 30,      |
|   orders: [                            |
|     { amount: 9 },                     |
|     { amount: 12 }                     |
|   ]                                    |
| }                                      |
+----------------------------------------+
| { _id: 2, name: "bob", age: 25,        |
|   orders: [                            |
|     { amount: 7 }                      |
|   ]                                    |
| }                                      |
+----------------------------------------+
```

One drawer of cards. Each card holds the whole picture for one user, **including** that user's orders embedded right inside the card. No join needed to read alice and her orders — they were on the same card the whole time.

### What "no schema enforcement by default" really means

You will hear "MongoDB is schemaless." That phrase is **slightly wrong** in a way that confuses beginners. The truth has two parts:

1. **MongoDB does not require you to declare a schema** before you put cards in. You can `insertOne({ ... })` with whatever fields you want, and MongoDB will accept it.
2. **Your application still needs the data to have a consistent shape** or you will not be able to write code against it. If half your `users` cards have `email` and half have `e_mail`, your code is going to break.

So the real rule is: MongoDB does not enforce a schema **at the database level by default,** but you will end up enforcing one **at the application level,** and MongoDB also offers a feature called **Schema Validation** that lets you push enforcement back into the database when you want it.

### Rich types — way beyond text and numbers

A SQL column can usually hold a string, a number, a date, or a boolean. That's about it. To hold a list of pets you need a separate `pets` table. To hold a JSON blob you need a `jsonb` column.

A MongoDB field can hold:

- **Strings** — `"alice"`
- **Numbers** — `30` (32-bit int), `NumberLong(30000000000)` (64-bit int), `NumberDecimal("3.14")` (high-precision decimal)
- **Booleans** — `true` or `false`
- **Dates** — `ISODate("2025-01-15T10:00:00Z")`
- **Arrays** — `["cat", "dog"]`
- **Embedded documents** — `{ city: "Portland", zip: "97201" }`
- **ObjectId** — a special 12-byte unique identifier (we will meet this in detail)
- **Regex** — `/^al/i` to match strings starting with "al"
- **Geo coordinates** — `{ type: "Point", coordinates: [-122.6, 45.5] }`
- **Binary data** — `BinData(0, "...")`
- **Null** — `null`

Notice arrays and embedded documents. Those two types are why people pick document databases. You can model "a user with three addresses" as one card with a three-element array, instead of two tables joined on a foreign key.

## BSON

When MongoDB stores your JSON-shaped card on disk, it does not actually save the text. Saving text would be slow to parse every time. Instead MongoDB uses a binary format called **BSON** (pronounced "bee-son"), short for **Binary JSON.**

BSON is two things at once:

1. **A binary serialization** of JSON-shaped data — laid out so the database can skim through fields quickly without re-parsing text.
2. **A type-extended version of JSON** — it adds types JSON does not have, like `Date`, `ObjectId`, `Decimal128`, `BinData`, `MinKey`, `MaxKey`, and `Long`.

Why care? Three reasons:

- **Speed.** Reading and writing is faster than parsing JSON text every time.
- **Types.** A JSON `42` could be an int or a float. A BSON `42` is one or the other, exactly, and the database knows which.
- **Size.** For most documents, BSON is comparable in size to compact JSON, occasionally smaller.

You will rarely write BSON by hand. The shell `mongosh` lets you write JavaScript objects that look like JSON, and it converts them to BSON behind the scenes. The drivers in every language do the same.

```
JSON                            BSON                   on disk
{name:"alice",age:30}    -->    [type:string][len][...]  bytes
                                [type:int32][value]
```

You speak JSON. The database speaks BSON. The conversion is automatic.

## Documents, Collections, Databases

The MongoDB world is just three layers deep. Memorize this and you will not get lost.

### Document (the index card)

The smallest unit. A JSON-shaped card. Capped at **16 megabytes** of size. If you are tempted to store a whole video in a single document, do not — use **GridFS** for big binary blobs.

```json
{
  "_id": ObjectId("66af5a..."),
  "name": "alice",
  "age": 30
}
```

Every document has an `_id` field. Always. We will get to it next.

### Collection (the drawer)

A group of documents. By convention, documents in the same collection have a **similar** shape (a "users" collection holds user-like cards), but MongoDB does not force this.

Naming convention: lowercase, plural, no spaces. `users`, `orders`, `products`, `events`.

Special kinds of collections:

- **Capped collections** — fixed-size, oldest-document-rolls-out-when-full, like a circular buffer. Good for log streams.
- **Time-series collections** (5.0+) — optimized layout for timestamped measurement data.
- **Views** — virtual read-only "collections" defined by an aggregation pipeline.

### Database (the cabinet)

A group of collections. One MongoDB server can host many databases. Common names: `myapp`, `analytics`, `staging`, `test`.

### The full hierarchy

```
mongod (the server process)
  |
  +-- database: myapp
  |     |
  |     +-- collection: users
  |     |     +-- document: { _id: ..., name: "alice" }
  |     |     +-- document: { _id: ..., name: "bob" }
  |     |
  |     +-- collection: orders
  |           +-- document: { _id: ..., user_id: ..., amount: 9 }
  |
  +-- database: analytics
        |
        +-- collection: events
              +-- document: { _id: ..., type: "click", at: ... }
```

A **namespace** is a collection's full name: `<database>.<collection>`, e.g. `myapp.users`. You will see this in error messages and explain output.

## _id and ObjectId

Every document has a field called `_id` and that field must be **unique within its collection.** The `_id` is the primary key. There is no way to have a document without one.

If you do not provide an `_id` when you insert, MongoDB makes one for you, and the value it makes is an **ObjectId.**

### What's an ObjectId?

A 12-byte (24-hex-character) value that is **almost certainly globally unique** without needing a central authority. It is structured like this:

```
ObjectId("66af5a3c8d2f1c0001a3b2c4")
          \------/\---/\---/\------/
            |      |    |     |
            |      |    |     +-- 3-byte counter (incremented per process)
            |      |    +-- 2-byte process id
            |      +-- 3 bytes of random unique value (per process)
            +-- 4-byte timestamp (Unix seconds since epoch)
```

The first four bytes are the **timestamp the ObjectId was generated.** Two cool consequences:

- ObjectIds sort roughly by creation time. If you `find().sort({_id:1})` you get oldest-to-newest.
- You can extract the creation time from an ObjectId without storing it separately.

```javascript
> ObjectId("66af5a3c8d2f1c0001a3b2c4").getTimestamp()
ISODate("2024-08-04T12:34:36.000Z")
```

### Can I use my own _id?

Yes. If you have a natural unique identifier (an email address, a username, a UUID), you can pass it as `_id` directly:

```javascript
> db.users.insertOne({ _id: "alice@example.com", age: 30 })
```

The benefit: lookup by `_id` uses a special highly-optimized scan called **IDHACK** that is the fastest possible query. The cost: you cannot ever change `_id` after insert (you would have to delete and re-insert), and `_id` values are immutable for the life of the document.

## CRUD

CRUD = **Create, Read, Update, Delete.** Every database does these four things. Here is how MongoDB does them.

We are going to use a fake **mongosh** session. `mongosh` is the official MongoDB shell — it is a command-line program that connects to a MongoDB server and lets you type queries in JavaScript. We will see how to start it under "Hands-On."

### Create — insertOne, insertMany

Insert a single document:

```javascript
> db.users.insertOne({ name: "alice", age: 30 })
{
  acknowledged: true,
  insertedId: ObjectId("66af5a3c8d2f1c0001a3b2c4")
}
```

`db.users` is shell-speak for "the users collection in the current database." `insertOne(...)` adds one card to that drawer. The server prints back an acknowledgement and the `_id` it generated.

Insert many at once:

```javascript
> db.users.insertMany([
... { name: "bob", age: 25 },
... { name: "carol", age: 40 },
... { name: "dave", age: 35 }
... ])
{
  acknowledged: true,
  insertedIds: {
    '0': ObjectId("66af5a4d8d2f1c0001a3b2c5"),
    '1': ObjectId("66af5a4d8d2f1c0001a3b2c6"),
    '2': ObjectId("66af5a4d8d2f1c0001a3b2c7")
  }
}
```

By default `insertMany` is **ordered** — if one insert fails, the rest stop. Pass `{ ordered: false }` to keep going.

### Read — find, findOne

Find every document in the collection:

```javascript
> db.users.find()
[
  { _id: ObjectId("..."), name: "alice", age: 30 },
  { _id: ObjectId("..."), name: "bob",   age: 25 },
  { _id: ObjectId("..."), name: "carol", age: 40 },
  { _id: ObjectId("..."), name: "dave",  age: 35 }
]
```

Find with a filter — pass a JSON-shaped query document:

```javascript
> db.users.find({ age: 30 })
[ { _id: ObjectId("..."), name: "alice", age: 30 } ]
```

Find with a comparison operator — operators start with a `$`:

```javascript
> db.users.find({ age: { $gte: 30 } })
[
  { _id: ObjectId("..."), name: "alice", age: 30 },
  { _id: ObjectId("..."), name: "carol", age: 40 },
  { _id: ObjectId("..."), name: "dave",  age: 35 }
]
```

`$gte` means "greater than or equal." We will see the full operator menu in the next section.

Find one — returns the first matching document or `null`:

```javascript
> db.users.findOne({ name: "alice" })
{ _id: ObjectId("..."), name: "alice", age: 30 }
```

Sort, limit, skip — chain them like LEGO:

```javascript
> db.users.find().sort({ age: -1 }).limit(2)
[
  { _id: ObjectId("..."), name: "carol", age: 40 },
  { _id: ObjectId("..."), name: "dave",  age: 35 }
]
```

`{ age: -1 }` means descending. `{ age: 1 }` means ascending.

Project — return only certain fields:

```javascript
> db.users.find({}, { name: 1, _id: 0 })
[
  { name: "alice" },
  { name: "bob" },
  { name: "carol" },
  { name: "dave" }
]
```

The second argument is the **projection.** `1` means include, `0` means exclude. By default `_id` is always included, so if you do not want it you have to explicitly say `_id: 0`.

### Update — updateOne, updateMany

Update one matching document with `$set`:

```javascript
> db.users.updateOne(
... { name: "alice" },
... { $set: { age: 31, email: "alice@example.com" } }
... )
{
  acknowledged: true,
  insertedId: null,
  matchedCount: 1,
  modifiedCount: 1,
  upsertedCount: 0
}
```

`$set` sets one or more fields. If the field does not exist, it is created. If it exists, it is overwritten.

Other update operators you will use constantly:

- `$set: { x: 1 }` — set a field
- `$unset: { x: "" }` — remove a field entirely
- `$inc: { count: 1 }` — increment a number
- `$mul: { price: 1.1 }` — multiply a number
- `$rename: { old: "new" }` — rename a field
- `$push: { tags: "new-tag" }` — append to an array
- `$pull: { tags: "old-tag" }` — remove from an array
- `$addToSet: { tags: "tag" }` — append only if not already present
- `$pop: { tags: 1 }` — pop the last element (or `-1` for first)
- `$min: { lowest: 5 }` — set if 5 is lower than current value
- `$max: { highest: 100 }` — set if 100 is higher than current value
- `$currentDate: { updated: true }` — set to current server time

Update many:

```javascript
> db.users.updateMany(
... { age: { $lt: 30 } },
... { $set: { teen_or_twenties: true } }
... )
{ acknowledged: true, matchedCount: 1, modifiedCount: 1, upsertedCount: 0 }
```

Upsert — update if match found, otherwise insert:

```javascript
> db.users.updateOne(
... { name: "eve" },
... { $set: { age: 28 } },
... { upsert: true }
... )
{
  acknowledged: true,
  matchedCount: 0,
  modifiedCount: 0,
  upsertedId: ObjectId("66af5b...")
}
```

`upsert: true` says "if you don't find a card matching `{ name: "eve" }`, create one with the union of the filter and the update."

### Replace — be careful

There is also `replaceOne` which **replaces the entire document** except `_id`:

```javascript
> db.users.replaceOne({ name: "alice" }, { name: "alice", age: 32 })
```

This is **dangerous.** It blows away every other field on alice's card. People hit this all the time when they meant `$set`. **Use `$set` unless you really mean to replace.**

### Delete — deleteOne, deleteMany

```javascript
> db.users.deleteOne({ name: "bob" })
{ acknowledged: true, deletedCount: 1 }

> db.users.deleteMany({ age: { $lt: 18 } })
{ acknowledged: true, deletedCount: 0 }

> db.users.deleteMany({})       # delete EVERYTHING in the collection — be careful
{ acknowledged: true, deletedCount: 3 }
```

There is no built-in undo. There is no recycling bin. If you `deleteMany({})` your data is gone. Restore from backup. This is why we have backups.

## Query Operators

The query operators are the verbs of MongoDB's query language. They all start with `$`. There are a lot of them. Here are the ones you will use over and over.

### Comparison

| Operator   | Meaning                          | Example                           |
|------------|----------------------------------|-----------------------------------|
| `$eq`      | equal                            | `{ age: { $eq: 30 } }`            |
| `$ne`      | not equal                        | `{ age: { $ne: 30 } }`            |
| `$lt`      | less than                        | `{ age: { $lt: 30 } }`            |
| `$lte`     | less than or equal               | `{ age: { $lte: 30 } }`           |
| `$gt`      | greater than                     | `{ age: { $gt: 30 } }`            |
| `$gte`     | greater than or equal            | `{ age: { $gte: 30 } }`           |
| `$in`      | matches any of an array          | `{ age: { $in: [25, 30, 35] } }`  |
| `$nin`     | matches none of an array         | `{ age: { $nin: [25, 30] } }`     |

A bare value is shorthand for `$eq`. `{ age: 30 }` means the same as `{ age: { $eq: 30 } }`.

### Logical

| Operator | Meaning                      | Example                                                       |
|----------|------------------------------|---------------------------------------------------------------|
| `$and`   | all clauses must match       | `{ $and: [ { age: { $gt: 18 } }, { age: { $lt: 30 } } ] }`    |
| `$or`    | any clause must match        | `{ $or: [ { name: "alice" }, { name: "bob" } ] }`             |
| `$not`   | negate                       | `{ age: { $not: { $gt: 30 } } }`                              |
| `$nor`   | none must match              | `{ $nor: [ { name: "alice" }, { age: 25 } ] }`                |

The implicit-AND case is common: `{ a: 1, b: 2 }` already means "a equals 1 AND b equals 2." You only need `$and` explicitly when you have multiple conditions on the **same** field that JavaScript object syntax can't express, e.g. `{ $and: [ { age: { $gt: 18 } }, { age: { $lt: 30 } } ] }`.

### Element

| Operator   | Meaning                                  | Example                                |
|------------|------------------------------------------|----------------------------------------|
| `$exists`  | field is present (or absent)             | `{ email: { $exists: true } }`         |
| `$type`    | field has a specific BSON type           | `{ age: { $type: "int" } }`            |

### Evaluation

| Operator     | Meaning                                  | Example                                     |
|--------------|------------------------------------------|---------------------------------------------|
| `$regex`     | regex match against a string             | `{ name: { $regex: "^al", $options: "i" } }`|
| `$mod`       | modulo                                   | `{ age: { $mod: [2, 0] } }` (even)          |
| `$expr`      | use aggregation expressions in a query   | `{ $expr: { $gt: ["$spent", "$budget"] } }` |
| `$jsonSchema`| validate against JSON Schema             | `{ $jsonSchema: { ... } }`                  |
| `$where`     | run arbitrary JavaScript (avoid)         | `{ $where: "this.age > 18" }`               |

`$where` is a **trap.** It runs JavaScript on the server for every document. It is slow. It cannot use indexes. It is a security risk. Avoid it. Use `$expr` instead in modern code.

### Array

| Operator      | Meaning                                       | Example                                         |
|---------------|-----------------------------------------------|-------------------------------------------------|
| `$elemMatch`  | array element matches all of a sub-query      | `{ scores: { $elemMatch: { $gt: 80, $lt: 90 } } }` |
| `$all`        | array contains all listed elements            | `{ tags: { $all: ["red", "blue"] } }`           |
| `$size`       | array has exact length                        | `{ tags: { $size: 3 } }`                        |

`$elemMatch` is important. Without it, `{ scores: { $gt: 80, $lt: 90 } }` matches if **any** element is > 80 **and any** element is < 90 — they could be different elements. With `$elemMatch`, the same element has to satisfy both conditions.

### Geo

| Operator               | Meaning                                          |
|------------------------|--------------------------------------------------|
| `$near`                | sort by distance from a point                    |
| `$geoWithin`           | inside a polygon                                 |
| `$geoIntersects`       | overlaps with a shape                            |
| `$nearSphere`          | sort by spherical distance                       |

These require a 2dsphere index on the field.

### Putting it all together

```javascript
> db.users.find({
...   age: { $gte: 18, $lt: 65 },
...   country: { $in: ["US", "CA", "UK"] },
...   email: { $exists: true },
...   tags: { $all: ["premium"] },
...   $or: [
...     { last_login: { $gt: ISODate("2025-01-01") } },
...     { vip: true }
...   ]
... })
```

That's "active adults under 65, in the US/CA/UK, with an email, who are premium, and who either logged in this year or are VIPs."

## Aggregation Pipeline

Find queries are great for filter-and-fetch. But what if you want to **transform** the data — group it, count it, join it, reshape it? That is what the **aggregation pipeline** is for.

### The pipeline mental model

```
documents in --> [stage 1] --> [stage 2] --> [stage 3] --> documents out
```

You hand the pipeline a list of stages. Each stage takes documents from the previous stage and emits documents to the next stage. The output of the whole pipeline is the output of the last stage.

It is exactly like a Unix pipeline (`ls | grep | sort | head`) except each stage operates on JSON-shaped documents instead of lines of text.

### A worked example

You have a `users` collection. Each user has a `country` and a `state`. You want to count how many users are in each US state, sorted by count descending, top 5.

```javascript
> db.users.aggregate([
...   { $match: { country: "US" } },
...   { $group: { _id: "$state", n: { $sum: 1 } } },
...   { $sort: { n: -1 } },
...   { $limit: 5 }
... ])
[
  { _id: "CA", n: 1234 },
  { _id: "TX", n:  987 },
  { _id: "NY", n:  765 },
  { _id: "FL", n:  543 },
  { _id: "WA", n:  321 }
]
```

Walk through it:

1. `$match` — keep only US users. Like `find({ country: "US" })`.
2. `$group` — group by `state`, and inside each group sum 1 for each document. The output is one document per state with `_id: <state>` and `n: <count>`.
3. `$sort` — order by count descending.
4. `$limit` — keep the top 5.

### The big stages

| Stage          | What it does                                                                  |
|----------------|-------------------------------------------------------------------------------|
| `$match`       | filter (use the same operators as `find`)                                     |
| `$project`     | reshape — pick fields, rename, compute new fields                             |
| `$addFields`   | add fields without removing existing ones (newer alias: `$set`)               |
| `$unset`       | remove fields                                                                 |
| `$group`       | group by an expression and aggregate (`$sum`, `$avg`, `$max`, etc.)           |
| `$sort`        | sort                                                                          |
| `$limit`       | keep first N                                                                  |
| `$skip`        | skip first N                                                                  |
| `$unwind`      | turn an array field into one document per element                             |
| `$lookup`      | left-outer-join another collection                                            |
| `$facet`       | run multiple sub-pipelines in parallel and combine results                    |
| `$bucket`      | group into ranges (histogram)                                                 |
| `$bucketAuto`  | auto-bucket into N equal-count buckets                                        |
| `$graphLookup` | recursive joins, e.g. follow a parent pointer up a tree                       |
| `$count`       | replace stream with `{ _id: null, n: <count> }`                               |
| `$sample`      | random sample                                                                 |
| `$out`         | write the pipeline output to a new collection (overwrites)                   |
| `$merge`       | write/merge into a target collection (smarter than `$out`)                    |
| `$replaceRoot` | replace the document with a sub-document                                      |
| `$setWindowFields` | window functions (5.0+) — running totals, moving averages, ranks         |

### A more interesting example: $unwind + $group

You have orders, each with an `items` array. You want total revenue per product across all orders.

```javascript
> db.orders.aggregate([
...   { $unwind: "$items" },
...   { $group: {
...       _id: "$items.sku",
...       revenue: { $sum: { $multiply: ["$items.qty", "$items.price"] } }
...   }},
...   { $sort: { revenue: -1 } }
... ])
```

`$unwind` flattens — if an order has 3 items, it becomes 3 separate documents in the pipeline, each with `items` being one of the 3 items. Then `$group` groups by SKU and sums `qty * price` per group.

### $lookup — the JOIN, sort of

```javascript
> db.orders.aggregate([
...   { $lookup: {
...       from: "users",
...       localField: "user_id",
...       foreignField: "_id",
...       as: "user"
...   }},
...   { $unwind: "$user" }
... ])
```

For each order, look up matching user(s) in the `users` collection where `orders.user_id` equals `users._id`, and stick the matches in a new array field `user` on the order document. Then `$unwind` flattens the array since we know there is exactly one match.

`$lookup` is a **left-outer-join.** It can be slow for big data sets — it does not have the optimizer that a SQL database has for joins. We will return to this in **Common Confusions.**

### $facet — multiple pipelines, one query

```javascript
> db.users.aggregate([
...   { $facet: {
...       by_country: [
...         { $group: { _id: "$country", n: { $sum: 1 } } }
...       ],
...       by_age_bucket: [
...         { $bucket: {
...             groupBy: "$age",
...             boundaries: [0, 18, 30, 50, 65, 200],
...             default: "unknown"
...         }}
...       ],
...       total: [ { $count: "n" } ]
...   }}
... ])
```

You get one document back with three fields, each holding the result of one sub-pipeline. Useful for dashboards.

### Pipeline optimization

The order of stages matters. **Put `$match` and `$project` early** so later stages have less work to do. The MongoDB optimizer can sometimes reorder for you, but not always. Rule of thumb: filter first, then transform.

```
WRONG:                                       RIGHT:
docs --> $sort --> $match --> $limit         docs --> $match --> $sort --> $limit
       (sorts everything)                          (filters first)
```

### Pipeline visual

```
[input collection]
     |
     v
+----------+
|  $match  |    drop documents that don't match
+----------+
     |
     v
+----------+
| $project |    reshape — pick the fields you want
+----------+
     |
     v
+----------+
|  $group  |    bucket and aggregate
+----------+
     |
     v
+----------+
|  $sort   |    sort the groups
+----------+
     |
     v
+----------+
|  $limit  |    keep the top N
+----------+
     |
     v
[result documents]
```

## Indexes

Without indexes, MongoDB looks at every single document in a collection to answer a query. That is a **collection scan** (often called **COLLSCAN**). Fine on 100 documents. Catastrophic on 100 million.

Indexes are extra data structures that MongoDB maintains so it can find documents quickly. They are like the index in the back of a textbook — they tell the database "for the value `30` of field `age`, the matching cards are at these positions."

### Single-field index

```javascript
> db.users.createIndex({ email: 1 })
```

`1` means ascending. `-1` is descending. For a single-field index it does not matter; for a compound index it matters a lot.

### Compound index

```javascript
> db.users.createIndex({ country: 1, age: -1 })
```

A compound index works for queries that filter on the **prefix** of the index. So this index helps:

- `find({ country: "US" })` — yes
- `find({ country: "US", age: { $lt: 30 } })` — yes
- `find({ age: { $lt: 30 } })` — **no, because the prefix is `country`**

This is the **ESR rule** (Equality, Sort, Range): for best results, put equality fields first, then sort fields, then range fields.

### Multikey index (for arrays)

If you index a field that holds an array, MongoDB makes one index entry **per array element.** That is a **multikey index.** It is automatic — you do not opt in.

```javascript
> db.posts.createIndex({ tags: 1 })
```

If a post has `tags: ["red","blue","green"]`, three index entries point to that document. Now `find({ tags: "blue" })` is fast.

**Gotcha:** you cannot have a compound index where **two** fields are arrays (parallel arrays). MongoDB refuses with `cannot index parallel arrays`.

### Text index

For full-text search on string fields:

```javascript
> db.posts.createIndex({ body: "text", title: "text" })
> db.posts.find({ $text: { $search: "mongodb tutorial" } })
```

Limited compared to dedicated search engines like Elasticsearch or **Atlas Search** (which is Lucene-based).

### Hashed index

```javascript
> db.users.createIndex({ user_id: "hashed" })
```

Hashes the field value before indexing. Trades range-query performance (lost) for even distribution (good for sharding by hash).

### Geospatial index

```javascript
> db.places.createIndex({ location: "2dsphere" })
> db.places.find({
...   location: {
...     $near: {
...       $geometry: { type: "Point", coordinates: [-122.6, 45.5] },
...       $maxDistance: 5000
...     }
...   }
... })
```

`2dsphere` understands the curve of the Earth. `2d` is the older flat-plane variant.

### Partial index

```javascript
> db.users.createIndex(
...   { email: 1 },
...   { partialFilterExpression: { active: true } }
... )
```

Only indexes documents matching the filter. Saves space if you only ever query active users.

### Sparse index

```javascript
> db.users.createIndex({ phone: 1 }, { sparse: true })
```

Skips documents that don't have the field at all. Old name for the partial-index idea — partial indexes are more general.

### Unique index

```javascript
> db.users.createIndex({ email: 1 }, { unique: true })
```

Refuses inserts that would create a duplicate `email`. The error you'll see is the famous **E11000 duplicate key error**.

### TTL (Time To Live)

```javascript
> db.sessions.createIndex({ created_at: 1 }, { expireAfterSeconds: 3600 })
```

A background process deletes documents whose indexed date is older than the threshold. Great for sessions, caches, expiring tokens. Approximate timing — runs every 60 seconds.

### Wildcard index

```javascript
> db.products.createIndex({ "specs.$**": 1 })
```

Indexes every field under `specs`. Useful when subfield names are dynamic (variable product attributes).

### Hidden index

```javascript
> db.users.createIndex({ email: 1 }, { hidden: true })
```

Index exists but query planner ignores it. Lets you "soft-disable" an index to test performance impact before dropping for real (4.4+).

### Listing and dropping

```javascript
> db.users.getIndexes()
[
  { v: 2, key: { _id: 1 }, name: "_id_" },
  { v: 2, key: { email: 1 }, name: "email_1", unique: true }
]

> db.users.dropIndex({ email: 1 })
{ nIndexesWas: 2, ok: 1 }
```

The `_id` index is automatic and cannot be dropped.

### Index-type comparison

```
type            best for                         gotchas
---------       -----------------------------    ----------------------
single          one-field equality / range       basic
compound        multi-field, sort + filter       prefix rule (ESR)
multikey        array element matching           no parallel arrays
hashed          even sharding distribution       no range queries
text            full-text search                 limited; use Atlas Search
2dsphere        geo on Earth's surface           field must be GeoJSON
partial         queries always include filter    smaller, faster
sparse          field is often missing           replaced by partial
unique          enforce no duplicates            E11000 on collision
TTL             auto-expire old data             ~60s precision
wildcard        unknown subfield names           larger
hidden          soft-disabled                    still updated on writes
```

### explain()

How do you know if your index is being used? You ask:

```javascript
> db.users.explain("executionStats").find({ email: "alice@example.com" })
```

Look at the `winningPlan.stage`. You want to see `IXSCAN` (index scan) or `IDHACK` (`_id` lookup). You do **not** want to see `COLLSCAN` (collection scan) — that means a full table read.

Also look at:

- `executionTimeMillis` — how long the query took
- `totalDocsExamined` — how many documents the engine looked at
- `totalKeysExamined` — how many index entries it looked at
- `nReturned` — how many it returned

Ratio of `totalDocsExamined` to `nReturned` should be close to 1. If you're examining 1,000,000 documents to return 5, your index is wrong.

## Schema Validation

Even though MongoDB does not enforce a schema by default, you can opt in. You attach a JSON Schema validator to a collection.

```javascript
> db.createCollection("users", {
...   validator: {
...     $jsonSchema: {
...       bsonType: "object",
...       required: ["name", "email"],
...       properties: {
...         name:  { bsonType: "string", minLength: 1 },
...         email: { bsonType: "string", pattern: "^.+@.+$" },
...         age:   { bsonType: "int", minimum: 0, maximum: 150 }
...       }
...     }
...   },
...   validationLevel: "strict",
...   validationAction: "error"
... })
```

`validationLevel`:

- `strict` — all inserts and updates must validate.
- `moderate` — only existing valid documents must remain valid; legacy non-conforming docs are left alone.

`validationAction`:

- `error` — refuse the write.
- `warn` — log a warning, allow the write.

This gives you SQL-style schema enforcement when you want it, without giving up the document model.

## Transactions

Until 4.0 (2018), MongoDB had **per-document atomicity** only. A single update was atomic; a multi-document operation was not.

Since **4.0**, you can wrap multiple operations across multiple collections inside a transaction on a **replica set.** Since **4.2**, transactions also work on **sharded clusters.**

```javascript
> const session = db.getMongo().startSession()
> session.startTransaction()
> const users = session.getDatabase("myapp").users
> const orders = session.getDatabase("myapp").orders
> users.updateOne({ _id: 1 }, { $inc: { credit: -10 } })
> orders.insertOne({ user_id: 1, total: 10 })
> session.commitTransaction()
> session.endSession()
```

If anything inside the block throws, you `session.abortTransaction()` and nothing is applied.

### Caveats

- Default transaction time limit is **60 seconds.** Long transactions are an anti-pattern.
- You will sometimes see `TransientTransactionError` — the recommended response is to **retry the whole transaction** from the start.
- Transactions on sharded clusters cost more than single-shard ones; design so most transactions stay on one shard.
- For high-throughput write paths, consider **single-document atomicity with embedded data** instead of multi-document transactions.

## Replica Sets

A **replica set** is a group of mongod servers that all hold copies of the same data. It is how MongoDB does **high availability.**

### The cast

- **Primary** — the one server that accepts writes. There is exactly one at a time.
- **Secondary** — a server that replicates from the primary. Read-only by default.
- **Arbiter** — a tiny no-data participant that votes in elections. Used to break ties in even-numbered sets. **Arbiters store no data.**
- **Hidden node** — a secondary that is invisible to drivers (useful for analytics or backup).
- **Delayed node** — a secondary that lags behind on purpose (a poor-person's "oops button" against bad writes).

### The picture

```
                    +---------+
   write -->        | PRIMARY | --- replicates oplog --->  +-----------+
                    +---------+                            | SECONDARY |
                         |                                 +-----------+
                         |     replicates oplog --->       +-----------+
                         +---------------------------->    | SECONDARY |
                                                           +-----------+
                                                                |
                                                                v
                                                             [reads]
```

### The oplog

The **oplog** is a special capped collection on the primary that records every write operation. Secondaries tail the oplog and apply each operation locally. The oplog is the heart of replication.

```
     primary's oplog (capped collection)
     +----+----+----+----+----+----+----+
     | i1 | u1 | u2 | i2 | d1 | i3 | u3 |
     +----+----+----+----+----+----+----+
       |    |    |
       v    v    v
     [secondary applies in order]
```

If a secondary falls behind so far that the primary's oplog has rotated past where it left off, the secondary cannot catch up — that is **stale** state and you have to resync. The window of time the oplog covers is called the **oplog window.** Aim for at least 24 hours.

### Election

If the primary dies, the remaining members hold an **election** and one of them becomes the new primary. Elections take a few seconds. During the election, the cluster cannot accept writes.

### writeConcern

When you do a write, how many nodes must acknowledge before the driver returns?

- `w: 1` — primary only (default in many cases).
- `w: 2` — primary + 1 secondary.
- `w: "majority"` — a majority of voting members. **This is what you want for durability.**
- `w: 0` — fire-and-forget; the driver does not wait. Risky.

You can also add `j: true` to require the write be **journaled** (written to the journal file), and `wtimeout` to bound how long to wait.

```javascript
> db.users.insertOne({ name: "alice" }, { writeConcern: { w: "majority", j: true, wtimeout: 5000 } })
```

### readPreference

Where does a read go?

- `primary` (default) — primary only.
- `primaryPreferred` — primary if available, else secondary.
- `secondary` — secondary only.
- `secondaryPreferred` — secondary if available, else primary.
- `nearest` — whichever node has lowest network latency.

Reading from secondaries spreads load but you might see slightly stale data (replication lag).

### readConcern

How fresh must the read be?

- `local` — whatever this node currently has.
- `available` — fast, may return data that could later be rolled back (sharded clusters).
- `majority` — only data that has been replicated to a majority. Will not be rolled back.
- `linearizable` — strongest; reflects all writes acknowledged before the read started.
- `snapshot` — used inside multi-document transactions.

## Sharding

Replica sets give you **availability** (and read scaling). They do **not** give you write scaling — every write still goes to one primary.

For write scaling and very large data sets, you **shard.** Sharding splits one logical collection across many servers based on a **shard key.**

### The cast

```
                              +-----------+
                              |  CLIENT   |
                              +-----------+
                                    |
                                    v
                              +-----------+
                              |   mongos  |  <-- query router
                              +-----------+
                              /     |     \
                             /      |      \
                +----------+ +----------+ +----------+
                |  SHARD A | |  SHARD B | |  SHARD C |
                | (replica | | (replica | | (replica |
                |   set)   | |   set)   | |   set)   |
                +----------+ +----------+ +----------+
                       \         |         /
                        \        |        /
                         v       v       v
                          +--------------+
                          | CONFIG SERVER|
                          | (replica set)|
                          +--------------+
```

- **mongos** — the router. Clients connect here. mongos figures out which shard(s) to ask.
- **shard** — a replica set holding part of the collection.
- **config server** — a special replica set that holds metadata: which chunk lives on which shard. Must be a replica set since 3.4.

### Chunks

A sharded collection is divided into **chunks** based on shard-key ranges. Default chunk size is **128 MB** (was 64 MB in older versions). The **balancer** is a background process that moves chunks between shards to keep things even.

A **jumbo chunk** is one that is bigger than the chunk size and cannot be split (because all its documents share one shard-key value). Jumbo chunks are bad — they cause uneven distribution.

### Shard key

This is the most important decision in sharding. The shard key is the field (or compound of fields) that MongoDB uses to decide which shard a document belongs on.

Two flavors:

- **Ranged** — chunks are ranges (e.g. `_id` from `0` to `1000`, `1000` to `2000`, etc.). Good locality for range queries; risk of hot-spot if writes target a range.
- **Hashed** — the key is hashed first; documents with adjacent keys land on different shards. Even distribution; range queries are scatter-gather.

A bad shard key can ruin you. Look for:

- High cardinality (many distinct values).
- Even write distribution.
- Queries that include the shard key (so mongos can route, not broadcast).

Pre-4.4 you could not change the shard key. Since **4.4**, `refineCollectionShardKey` lets you add suffix fields. Since **5.0**, **resharding** lets you change the key entirely (online).

### enable + shard

```javascript
> sh.enableSharding("mydb")
> sh.shardCollection("mydb.events", { customer_id: "hashed" })
```

That's it — the collection is now sharded by `customer_id` with a hashed strategy.

### Zones

Sometimes you want certain data to live on certain shards (e.g. EU customers on EU shards for data residency). **Zones** let you tag shards and assign shard-key ranges to zones.

## Change Streams

A change stream is a subscription to insert/update/delete events on a collection, database, or whole cluster. Internally, it tails the oplog. From your code, it looks like a long-running cursor.

```javascript
> db.users.watch()
{ resumeToken: ..., operationType: "insert", fullDocument: { ... }, ... }
{ resumeToken: ..., operationType: "update", documentKey: ..., updateDescription: ..., ... }
```

### resume tokens

Every event has a **resume token.** If your code crashes, you can pass the last token back to `watch({ resumeAfter: token })` and pick up where you left off — as long as the oplog still has the entries.

### fullDocument: "updateLookup"

By default, an update event only tells you the change (the `$set`/`$unset`). If you want the post-update document, ask for it:

```javascript
> db.users.watch([], { fullDocument: "updateLookup" })
```

Since **6.0** you can also request `fullDocumentBeforeChange` — but this requires you enable change-stream pre/post images on the collection first.

### Use cases

- Sync to a search engine (push every change to Elasticsearch / Atlas Search).
- Sync to a cache.
- Audit log.
- Real-time UI updates.
- Triggers (Atlas has Triggers built on this).

## Atlas (managed) and the Stable API

### Atlas

**MongoDB Atlas** is MongoDB Inc.'s managed cloud service. It runs MongoDB on AWS, GCP, or Azure. It handles backups, monitoring, scaling, sharding, and upgrades for you. Free tier exists.

Atlas adds features that are not in the open-source server:

- **Atlas Search** — Lucene-based full-text search.
- **Atlas Vector Search** — kNN vector search for embeddings (RAG, semantic search).
- **Atlas Charts** — built-in charting on your data.
- **Atlas Triggers** — change-stream-driven serverless functions.
- **Atlas Functions** — server-side JS functions (think Lambda).
- **Atlas Data Lake** — query S3-stored data.
- **Atlas Online Archive** — auto-tier cold data to cheaper storage.

### Stable API

For a long time, MongoDB drivers spoke "whatever-this-server-supports." That made upgrades fragile. Since **5.0**, the **Stable API** (called Versioned API at first) lets you pin your client to a frozen subset of commands. Servers that support that version promise the API will keep working.

```javascript
// connection string side, or in driver options:
const client = new MongoClient(uri, { serverApi: { version: '1', strict: true } });
```

`strict: true` means: any command not in v1 will be rejected. Frees you to upgrade the server underneath.

## Common Errors

These are real error strings you will see, ordered by how often they bite. Memorize the canonical fix for each.

### MongoServerError: E11000 duplicate key error collection: myapp.users index: email_1 dup key: { email: "alice@example.com" }

A unique-index violation. Some other document already has that value.

Fix: pick a different value, or do an upsert/update if you meant to overwrite.

### MongoServerError: cannot index parallel arrays [tags] [scores]

You tried to create a compound index where two of the indexed fields are both arrays. MongoDB cannot do this — multikey indexing on two parallel arrays would explode the index size.

Fix: pick one to index, or restructure the data.

### MongoServerError: command not found

You typed a command name that does not exist on this server. Common cause: typo, or you used a name that was deprecated and removed in your version.

Fix: check spelling; check the server version; check the Stable API.

### MongoServerError: not authorized on myapp to execute command { find: "users", ... }

You are connected as a user without privileges for this action.

Fix: connect as an authorized user, or grant the appropriate role (`readWrite`, `dbAdmin`, etc.).

### MongoServerError: ConnectionPool failed: connection refused

Your driver could not connect to the host:port at all. Either nothing is listening, or a firewall is in the way.

Fix: confirm `mongod` is running (`sudo systemctl status mongod`), confirm the port is right (default 27017), confirm `bindIp` allows your IP.

### MongoServerError: server selection timeout

Driver waited the configured `serverSelectionTimeoutMS` (30 s default) trying to find a suitable server (primary, etc.) and gave up.

Fix: check connection string, replica set name (`replicaSet=...`), DNS for `mongodb+srv://`, network reachability of every member.

### MongoServerError: WriteConflict: write conflict during plan execution

Two operations tried to modify the same document at the same time. Usually appears inside transactions or under heavy concurrent updates.

Fix: retry. The driver retries automatically in many cases. Inside a transaction, abort and retry the whole transaction.

### MongoServerError: cursor not found, cursor id: 12345

Your client paused too long between batches and the server timed out the cursor (default 10 min on the server, configurable).

Fix: process batches faster, or use `noCursorTimeout: true` on long-running cursors (and remember to close them yourself).

### MongoServerError: TransientTransactionError

Something inside a transaction temporarily failed (often a `WriteConflict`).

Fix: catch the error and retry the whole transaction. The driver provides a helper (`session.withTransaction(...)`) that does this for you.

### MongoServerError: ExceededTimeLimit: operation exceeded time limit

You set a `maxTimeMS` on the operation and it ran too long.

Fix: optimize the query (add an index, reshape the pipeline) or raise the timeout if appropriate.

### MongoServerError: NoSuchTransaction

You called `commitTransaction` on a session whose transaction has already aborted, expired, or never started.

Fix: check your transaction lifecycle; do not commit twice; retry from the start of the transaction.

## Hands-On

Time to type real things. We'll assume you have `mongod` (the server) running locally on port 27017, the default. If you don't, the easiest path is `brew install mongodb-community` on macOS, or follow the official install for your distro, then `brew services start mongodb-community` (or `sudo systemctl start mongod`).

You will also need `mongosh` — the modern shell. It comes with MongoDB Community packages, or install standalone with `brew install mongosh`.

We'll go through 30+ commands, simplest to scariest.

### 1. Start mongosh

```
$ mongosh
Current Mongosh Log ID: 6620aa11...
Connecting to:          mongodb://127.0.0.1:27017/?directConnection=true
Using MongoDB:          7.0.5
Using Mongosh:          2.1.5

For mongosh info see: https://docs.mongodb.com/...

test>
```

`test>` is the prompt. The current database is `test`.

### 2. Connect to a remote/explicit URI

```
$ mongosh "mongodb://localhost:27017"
```

Or with `+srv` (DNS seedlist, used by Atlas):

```
$ mongosh "mongodb+srv://user:pass@cluster0.abcde.mongodb.net/myapp"
```

### 3. List databases

```
test> show dbs
admin    40 KiB
config  108 KiB
local   72.0 KiB
```

Three system databases. We have not made any of our own yet.

### 4. Switch to / create a database

```
test> use mydb
switched to db mydb
mydb>
```

The database does not exist yet. It will spring into being the first time you write to it.

### 5. List collections

```
mydb> show collections
```

(empty — we have not created any)

### 6. Insert one document

```
mydb> db.users.insertOne({ name: "alice", age: 30 })
{
  acknowledged: true,
  insertedId: ObjectId("66af5a3c8d2f1c0001a3b2c4")
}
```

The collection `users` was created automatically.

### 7. Insert many

```
mydb> db.users.insertMany([
... { name: "bob", age: 25 },
... { name: "carol", age: 40, country: "US" },
... { name: "dave", age: 35, country: "CA" }
... ])
{
  acknowledged: true,
  insertedIds: {
    '0': ObjectId("..."),
    '1': ObjectId("..."),
    '2': ObjectId("...")
  }
}
```

### 8. Find with a filter

```
mydb> db.users.find({ age: { $gte: 18 } })
[
  { _id: ObjectId("..."), name: "alice", age: 30 },
  { _id: ObjectId("..."), name: "bob",   age: 25 },
  { _id: ObjectId("..."), name: "carol", age: 40, country: "US" },
  { _id: ObjectId("..."), name: "dave",  age: 35, country: "CA" }
]
```

### 9. Sort and limit

```
mydb> db.users.find().sort({ age: -1 }).limit(10)
[
  { _id: ObjectId("..."), name: "carol", age: 40, country: "US" },
  { _id: ObjectId("..."), name: "dave",  age: 35, country: "CA" },
  { _id: ObjectId("..."), name: "alice", age: 30 },
  { _id: ObjectId("..."), name: "bob",   age: 25 }
]
```

### 10. Update one with $set

```
mydb> db.users.updateOne(
... { name: "alice" },
... { $set: { email: "alice@example.com" } }
... )
{
  acknowledged: true,
  insertedId: null,
  matchedCount: 1,
  modifiedCount: 1,
  upsertedCount: 0
}
```

### 11. Aggregation: count by country

```
mydb> db.users.aggregate([
... { $match: { country: { $exists: true } } },
... { $group: { _id: "$country", n: { $sum: 1 } } },
... { $sort: { n: -1 } }
... ])
[ { _id: "CA", n: 1 }, { _id: "US", n: 1 } ]
```

### 12. Create a unique index on email

```
mydb> db.users.createIndex({ email: 1 }, { unique: true })
email_1
```

### 13. Try to insert a duplicate email (will fail)

```
mydb> db.users.insertOne({ name: "alice2", email: "alice@example.com" })
MongoServerError: E11000 duplicate key error collection: mydb.users index: email_1 dup key: { email: "alice@example.com" }
```

That's the most famous error in MongoDB. Now you've seen it.

### 14. Create a geospatial index

```
mydb> db.places.insertOne({
...   name: "park",
...   location: { type: "Point", coordinates: [-122.6, 45.5] }
... })
mydb> db.places.createIndex({ location: "2dsphere" })
location_2dsphere
```

### 15. List indexes

```
mydb> db.users.getIndexes()
[
  { v: 2, key: { _id: 1 }, name: '_id_' },
  { v: 2, key: { email: 1 }, name: 'email_1', unique: true }
]
```

### 16. Explain a query

```
mydb> db.users.explain("executionStats").find({ email: "alice@example.com" })
{
  queryPlanner: { ... winningPlan: { stage: 'IXSCAN', ... } ... },
  executionStats: {
    nReturned: 1,
    executionTimeMillis: 0,
    totalKeysExamined: 1,
    totalDocsExamined: 1,
    ...
  },
  ...
}
```

Look for `IXSCAN`. Good. `nReturned == totalDocsExamined`. Good.

### 17. Drop an index

```
mydb> db.users.dropIndex({ email: 1 })
{ nIndexesWas: 2, ok: 1 }
```

### 18. Collection stats

```
mydb> db.users.stats()
{
  ns: 'mydb.users',
  count: 4,
  size: 200,
  avgObjSize: 50,
  storageSize: 36864,
  ...
}
```

### 19. Server status

```
mydb> db.serverStatus()
{
  host: '...',
  version: '7.0.5',
  connections: { current: 1, available: 838859 },
  uptime: 12345,
  ...
}
```

A wall of operational metrics. Useful for ops dashboards.

### 20. Ping

```
mydb> db.runCommand({ ping: 1 })
{ ok: 1 }
```

The basic "are you alive" health check.

### 21. Replica set status (only on replica sets)

```
mydb> rs.status()
{
  set: 'rs0',
  members: [
    { name: 'h1:27017', stateStr: 'PRIMARY', ... },
    { name: 'h2:27017', stateStr: 'SECONDARY', ... },
    { name: 'h3:27017', stateStr: 'SECONDARY', ... }
  ],
  ...
}
```

If you are not on a replica set, this errors.

### 22. Initialize a replica set

```
mydb> rs.initiate({
...   _id: "rs0",
...   members: [
...     { _id: 0, host: "h1:27017" },
...     { _id: 1, host: "h2:27017" },
...     { _id: 2, host: "h3:27017" }
...   ]
... })
{ ok: 1 }
```

### 23. Add a replica set member

```
mydb> rs.add("h4:27017")
{ ok: 1 }
```

### 24. Show replica set config

```
mydb> rs.config()
{ _id: "rs0", version: 2, members: [...], settings: {...} }
```

### 25. Sharding status (on a sharded cluster, connected via mongos)

```
mydb> sh.status()
--- Sharding Status ---
  sharding version: { ... }
  shards: [...]
  databases: [...]
```

### 26. Enable sharding for a database

```
mydb> sh.enableSharding("mydb")
{ ok: 1 }
```

### 27. Shard a collection

```
mydb> sh.shardCollection("mydb.events", { customer_id: "hashed" })
{ ok: 1 }
```

### 28. List databases (admin command)

```
mydb> db.adminCommand({ listDatabases: 1 })
{ databases: [...], totalSize: ..., ok: 1 }
```

### 29. Watch a collection (change stream)

```
mydb> db.users.watch()
[
  { _id: { _data: '...' },
    operationType: 'insert',
    fullDocument: { ... },
    ... },
  ...
]
```

Cursor sits open and emits one event per change. Press Ctrl-C to stop.

### 30. mongostat (from your shell, not mongosh)

```
$ mongostat --host localhost
insert query update delete getmore command dirty used flushes vsize  res qrw arw net_in net_out conn        time
    *0    *0     *0     *0       0     1|0  0.0% 0.0%       0 1.78G 102M 0|0 0|0   158b   45.0k    1 Apr 27 12:00:00.123
    *0    *0     *0     *0       0     1|0  0.0% 0.0%       0 1.78G 102M 0|0 0|0   158b   45.0k    1 Apr 27 12:00:01.123
```

One line per second. Insert/query/update/delete rates, dirty cache %, network in/out.

### 31. mongotop

```
$ mongotop --host localhost
ns                total      read      write   2025-04-27T12:00:01Z
mydb.users         0ms       0ms        0ms
admin.system.roles 0ms       0ms        0ms
```

Per-namespace time spent reading/writing.

### 32. mongodump (backup)

```
$ mongodump --db mydb --out /backup
2025-04-27T12:00:01.000-0700  writing mydb.users to /backup/mydb/users.bson
2025-04-27T12:00:01.001-0700  done dumping mydb.users (4 documents)
```

A binary BSON dump, one file per collection.

### 33. mongorestore

```
$ mongorestore --db mydb /backup/mydb
2025-04-27T12:01:00.000-0700  the --db and --collection flags are deprecated...
2025-04-27T12:01:00.001-0700  restoring mydb.users from /backup/mydb/users.bson
2025-04-27T12:01:00.002-0700  finished restoring mydb.users (4 documents, 0 failures)
```

### 34. mongoexport (JSON / CSV)

```
$ mongoexport --collection users --out users.json --db mydb
2025-04-27T12:02:00.000-0700  connected to: mongodb://localhost:27017
2025-04-27T12:02:00.001-0700  exported 4 records
```

Resulting `users.json` is one JSON document per line (NDJSON) by default.

### 35. mongoimport

```
$ mongoimport --collection users --file users.json --db mydb
2025-04-27T12:03:00.000-0700  connected to: mongodb://localhost:27017
2025-04-27T12:03:00.001-0700  4 document(s) imported successfully. 0 document(s) failed to import.
```

For arrays inside one big JSON file, add `--jsonArray`.

### 36. One-off shell command from outside

```
$ mongosh --eval "db.users.estimatedDocumentCount()" mydb
4
```

`estimatedDocumentCount()` reads from collection metadata — instant, but approximate during heavy churn. Use `countDocuments({...})` for an exact count using the query engine.

### 37. Server command-line options used at startup

```
mydb> db.serverCmdLineOpts()
{ argv: [ 'mongod', '--config', '/etc/mongod.conf' ],
  parsed: { config: '/etc/mongod.conf', net: { ... }, storage: { ... } },
  ok: 1 }
```

### 38. Default read/write concern

```
mydb> db.adminCommand({ getDefaultRWConcern: 1 })
{
  defaultReadConcern: { level: 'local' },
  defaultWriteConcern: { w: 'majority', wtimeout: 0 },
  ...
}
```

### 39. Create a user

```
mydb> use admin
switched to db admin
admin> db.createUser({
...   user: "app",
...   pwd: "X",
...   roles: [ { role: "readWrite", db: "mydb" } ]
... })
{ ok: 1 }
```

(Use a real password. Then enable auth in `mongod.conf` and restart.)

That is more than 30 — congratulations, you have used the breadth of mongosh.

## Common Confusions

These are real misunderstandings that bite beginners (and not-so-beginners) over and over.

### 1. "Schemaless" does not mean "no schema"

You will still have a schema in your code, in your tests, and in your head. MongoDB just does not enforce one **at the database level by default.** When you want enforcement, attach a `$jsonSchema` validator. Without one, you are responsible.

### 2. Upsert is not the same as "insert if missing"

`updateOne(filter, update, { upsert: true })` does this: if a document matches the filter, apply the update; otherwise create a new document that is the **union** of the filter and the operators in the update. People expect the new doc to be exactly the update arg — it is not, it is the merge.

### 3. `$set` updates fields. `replaceOne` blows away the document.

```javascript
// keeps every other field, sets only `age`:
> db.users.updateOne({ _id: 1 }, { $set: { age: 31 } })

// REPLACES the whole document with `{ age: 31 }`. Loses name, email, everything:
> db.users.replaceOne({ _id: 1 }, { age: 31 })
```

99% of the time you want `$set`.

### 4. Nested-field dot notation

To filter on a nested field, use a quoted dotted string:

```javascript
> db.users.find({ "address.city": "Portland" })
```

You **must** quote it. `address.city: "Portland"` (without quotes around the key) is a JavaScript syntax error.

### 5. Multikey indexes affect compound indexes

You cannot have a compound index where two indexed fields are both arrays in the same document. The error is `cannot index parallel arrays`. If both fields you want to index are usually arrays, either store one as an embedded object instead, or accept two separate single-field multikey indexes.

### 6. Order of `$match` in pipelines matters

Put `$match` as **early as possible** in the pipeline. The aggregator can sometimes push it earlier itself, but not always. A `$match` after a `$group` cannot use indexes — the data is no longer indexed. A `$match` before a `$group` can.

### 7. Projection `{_id: 0, name: 1}`

By default `_id` is **always included.** If you want it gone, you have to say `_id: 0` explicitly. You also cannot mix include and exclude in the same projection except for `_id`. `{name: 1, age: 0}` is illegal. `{_id: 0, name: 1, age: 1}` is fine.

### 8. ObjectId timestamp extraction

The first four bytes of an ObjectId are the creation Unix timestamp. You can use this in a pinch:

```javascript
> ObjectId("66af5a3c8d2f1c0001a3b2c4").getTimestamp()
ISODate("2024-08-04T12:34:36.000Z")
```

So if you need "documents created in the last hour," and you don't have a `createdAt` field, you can compute the bounding ObjectIds and `find({_id: {$gte: ObjectId.fromHex(<hex>)}})`. Cute, but most apps just store an explicit `createdAt`.

### 9. `readPreference: "primary"` vs `"secondary"`

`primary` (default) gives you the freshest possible data — every read goes to the one server that accepts writes. `secondary` reads from a replica, so it scales out reads at the cost of possible **replication lag** (milliseconds usually, but sometimes seconds). Pick `primary` when you cannot tolerate stale reads (e.g. read-your-write); pick `secondary` for analytics or background jobs.

### 10. writeConcern semantics

`w: 1` returns when the primary has the write. `w: "majority"` returns when a majority of the replica set has it — this is the only way to guarantee the write **survives a primary failover.** `w: 0` is unacknowledged — the driver returns immediately, never sees errors. Newer MongoDB versions default to `w: "majority"` for many operations.

### 11. Transactions are only on replica sets (and sharded clusters)

A standalone single-server deployment **cannot** run multi-document transactions. You need a replica set (even a single-node replica set works). On sharded clusters, transactions work since 4.2 but cost more.

### 12. `findOneAndUpdate` vs `updateOne` + `findOne`

`findOneAndUpdate` is **atomic** — the read and write happen as one operation, no other client can sneak in between. `updateOne` followed by `findOne` is two round trips and another client could change the document between them. Always prefer `findOneAndUpdate` (or `findOneAndReplace`, `findOneAndDelete`) when you need the value.

It also takes a `returnDocument` option: `"before"` (default) returns the document as it was before the update, `"after"` returns it after.

### 13. Arbiters do not store data

An arbiter is a vote-only replica set member. It exists to break ties so an even-numbered set can still elect a primary. An arbiter has no oplog, holds no documents, and **does not count toward `w: "majority"`** for data durability. Best practice in modern MongoDB is to **not** use arbiters — prefer an actual data-bearing third member.

### 14. Chunk migrations are visible

When the balancer moves a chunk from shard A to shard B, the migration is online. Reads and writes keep working. But the migration adds load to both shards, and you can see brief latency spikes. You can configure a **balancer window** so balancing only runs at off-peak times.

### 15. `$lookup` is a left-outer-join — and slow if you're not careful

`$lookup` does what a SQL left outer join does, conceptually. But MongoDB does not have a SQL-grade query optimizer with statistics-based join plans. For small lookups (a few thousand docs joined with an indexed foreign field), it is fine. For huge cross-products, it is brutal. Two rules:

- **Always** make sure `foreignField` is indexed.
- If you find yourself doing many `$lookup`s in one pipeline, consider whether the data should have been embedded in the first place, or whether you should denormalize.

### 16. `find({a: {b: 1}})` is not the same as `find({"a.b": 1})`

```javascript
// Matches docs where field `a` is the EXACT object {b:1}:
> db.coll.find({ a: { b: 1 } })

// Matches docs where the nested field a.b equals 1, regardless of other fields:
> db.coll.find({ "a.b": 1 })
```

The first is an equality match on the whole sub-document. The second drills into the path. Easy to confuse.

### 17. `$expr` for cross-field comparisons

A normal find query can't say "where field X is greater than field Y on the same document." For that you need `$expr`:

```javascript
> db.budgets.find({ $expr: { $gt: ["$spent", "$budget"] } })
```

`$expr` lets you use aggregation expression syntax inside a query.

## Vocabulary

| Term | Meaning |
|------|---------|
| MongoDB | A document-oriented NoSQL database, the subject of this sheet |
| mongod | The MongoDB server process |
| mongos | The query router that fronts a sharded cluster |
| mongoc | Older shorthand for the C driver (`libmongoc`) |
| mongosh | The modern interactive shell |
| mongo (legacy shell) | The pre-2021 shell, replaced by mongosh |
| MongoDB Compass | The official GUI client |
| MongoDB Atlas | The managed cloud service from MongoDB Inc. |
| MongoDB Enterprise | The paid on-prem edition with auth/audit/encryption add-ons |
| BSON | Binary JSON — MongoDB's on-disk and wire format |
| JSON | The text data format that BSON extends |
| document | One JSON-shaped record; the index card |
| collection | A group of documents; the drawer |
| database | A group of collections; the cabinet |
| _id | The required, unique primary key field on every document |
| ObjectId | A 12-byte unique value with embedded timestamp; the default `_id` |
| NumberLong | A 64-bit integer in BSON |
| NumberInt | A 32-bit integer in BSON |
| NumberDecimal (Decimal128) | A 128-bit high-precision decimal type, good for money |
| Date | A BSON date type, milliseconds since epoch |
| ISODate | The mongosh constructor for a Date in ISO-8601 form |
| BinData | A BSON binary blob |
| MD5 | A 128-bit hash, common for fingerprints |
| UUID | A 128-bit universally unique identifier; storable as BinData subtype 4 |
| MinKey | A BSON value that compares less than every other value |
| MaxKey | A BSON value that compares greater than every other value |
| regex | A regular expression, a BSON type for pattern matching |
| namespace | A collection's full name: `<database>.<collection>` |
| capped collection | Fixed-size FIFO collection; oldest rolls out when full |
| time-series collection | Specialized layout for timestamped measurement data (5.0+) |
| index | An auxiliary structure that speeds up queries on a field |
| single-field | An index on one field |
| compound | An index on multiple fields, in order |
| multikey | An index on a field that holds arrays — one entry per element |
| text | An index for full-text search on string fields |
| hashed | An index that hashes the field value before indexing |
| geospatial | An index on geographic coordinates |
| 2d | The older flat-plane geo index |
| 2dsphere | The Earth-curvature geo index |
| partial filter expression | The optional filter on a partial index |
| sparse | An index that skips documents missing the field |
| unique | An index that refuses duplicate values |
| TTL (expireAfterSeconds) | An index option that auto-deletes old documents |
| wildcard index | An index that covers any subfield under a path |
| hidden index | An index the planner ignores; for soft-disabling |
| IXSCAN | The explain-plan label for an index scan |
| COLLSCAN | The explain-plan label for a full-collection scan (bad on large data) |
| FETCH | The explain-plan stage that retrieves a document by its index entry |
| IDHACK | The optimized stage for `_id` lookups |
| COUNT_SCAN | The explain-plan stage for a covered count |
| query plan | The chosen strategy for executing a query |
| explain | The command that prints the query plan |
| queryPlanner | The explain mode that shows just the plan |
| executionStats | The explain mode that runs the query and reports timings |
| allPlansExecution | The explain mode that runs all candidate plans |
| aggregation pipeline | A chain of transformation stages applied to documents |
| $match | Filter stage |
| $group | Group-and-aggregate stage |
| $project | Field-shaping stage |
| $addFields | Add new fields without removing existing ones |
| $unset | Remove fields |
| $sort | Sort stage |
| $limit | Take first N |
| $skip | Skip first N |
| $count | Replace stream with a single count document |
| $facet | Run multiple sub-pipelines in parallel |
| $unwind | Flatten array elements into separate documents |
| $lookup | Left-outer-join another collection |
| $graphLookup | Recursive joins for tree/graph traversal |
| $bucket | Group documents into ranges (histogram) |
| $bucketAuto | Auto-bucket into N equal-count groups |
| $sample | Random sample stage |
| $merge | Write/merge results into a target collection |
| $out | Replace a target collection with pipeline output |
| $set (in pipeline) | Synonym for `$addFields` (4.2+) |
| $replaceRoot | Replace the document with a sub-document |
| $replaceWith | Shorter alias for `$replaceRoot` |
| $expr | Use aggregation expressions inside a query |
| $cond | The if/then/else aggregation expression |
| $switch | Multi-branch case expression |
| $arrayElemAt | Pick the Nth element of an array |
| $size (agg) | The aggregation form of array-size |
| $reduce | Fold a function over an array |
| $map | Apply a function to each element of an array |
| $filter | Keep array elements that satisfy a predicate |
| $zip | Combine multiple arrays element-wise |
| $cmp | Compare two values (returns -1/0/1) |
| $strLenBytes | Length of a string in bytes |
| $regexMatch | Boolean regex match in aggregations |
| $dateToString | Format a date |
| $dateFromString | Parse a date from a string |
| $year/$month/$dayOfWeek | Extract components from a date |
| $sum | Aggregate sum |
| $avg | Aggregate average |
| $min | Aggregate min |
| $max | Aggregate max |
| $first | First value in a group |
| $last | Last value in a group |
| $stdDevPop | Population standard deviation |
| $stdDevSamp | Sample standard deviation |
| $accumulator | User-defined accumulator with JS functions |
| $function | Inline server-side JavaScript function (use carefully) |
| accumulator stages | Stages that aggregate values into one (`$group`, `$bucket`) |
| window functions $setWindowFields | Running totals, ranks, moving averages (5.0+) |
| schema validator | A server-side rule to validate documents |
| $jsonSchema | Validate documents against JSON Schema |
| validationLevel (strict/moderate) | How aggressively the validator runs |
| validationAction (error/warn) | What happens on validation failure |
| retryable writes | Driver-level automatic retry of failed writes |
| retryable reads | Driver-level automatic retry of failed reads |
| MongoClient | The driver-level client object |
| connection string mongodb:// | The standard connection string scheme |
| mongodb+srv:// | Connection string with DNS seedlist (Atlas style) |
| readPreference primary | Reads always go to the primary |
| readPreference primaryPreferred | Reads prefer primary, fall back to secondary |
| readPreference secondary | Reads always go to a secondary |
| readPreference secondaryPreferred | Reads prefer secondary, fall back to primary |
| readPreference nearest | Reads go to the lowest-latency node |
| readConcern local | Read whatever this node has now |
| readConcern majority | Read only data committed by a majority |
| readConcern snapshot | Used inside transactions |
| readConcern linearizable | Strongest read; reflects all acknowledged writes |
| readConcern available | Sharded-cluster fast read; may roll back |
| writeConcern w:1 | Wait for primary only |
| writeConcern w:2 | Wait for primary plus one secondary |
| writeConcern w:majority | Wait for majority — durable across failover |
| writeConcern w:0 | Unacknowledged — fire and forget |
| j:true (journal) | Wait for the write to be journaled |
| wtimeout | Max time to wait for the write concern |
| replicaSet=name URI param | Tells the driver the replica-set name |
| replica set | A group of mongods replicating the same data |
| primary | The replica-set member that accepts writes |
| secondary | A replica-set member that replicates from the primary |
| arbiter | A vote-only replica-set member; stores no data |
| hidden node | A secondary invisible to clients |
| delayed node | A secondary that lags on purpose |
| priority | A replica-set member's election preference |
| vote | A replica-set member's voting rights |
| oplog | The capped collection of write operations on the primary |
| oplog window | How far back in time the oplog covers |
| replication lag | The delay between primary and secondary |
| election | The vote to choose a new primary |
| heartbeat | The periodic ping between replica-set members |
| settings (catchUpTimeoutMillis) | Replica-set tunable for catch-up after election |
| rs.initiate | Bootstrap a new replica set |
| rs.add | Add a member |
| rs.remove | Remove a member |
| rs.stepDown | Force the primary to step down |
| rs.freeze | Prevent a node from running for primary for N seconds |
| rs.printReplicationInfo | Print oplog status summary |
| change stream | A subscription to insert/update/delete events |
| db.collection.watch() | Open a change stream on a collection |
| resume token | A position marker for change-stream resumption |
| fullDocument before/after | Optional payloads showing pre/post state |
| sharded cluster | A cluster split across multiple shards |
| mongos | The query router |
| config server | The replica set holding cluster metadata |
| shard | A replica set holding part of a sharded collection |
| chunk | A range of shard-key values held by a shard |
| chunk size | Default 128 MB max per chunk |
| balancer | The background process that migrates chunks |
| balancer window | Optional time window for balancing |
| jumbo chunk | A chunk too big to split — bad |
| shard key | The field(s) MongoDB uses to route documents to shards |
| hashed shard key | Shard key applied through a hash function |
| ranged shard key | Shard key used in literal ranges |
| compound shard key | A shard key on multiple fields |
| zone | A tagged group of shards for data locality |
| zone key range | The shard-key range assigned to a zone |
| sh.addShardTag (legacy) | Older zone-tagging command |
| refineCollectionShardKey | Add suffix fields to an existing shard key (4.4+) |
| reshardCollection | Change the shard key online (5.0+) |
| Atlas | Managed MongoDB cloud service |
| MongoDB Atlas Search | Lucene-based search add-on |
| Atlas Vector Search | kNN vector search for embeddings |
| Atlas Triggers | Change-stream-driven serverless functions |
| Atlas Functions | Server-side JS functions |
| Atlas Charts | Built-in charting |
| Atlas Data Lake | Query data on cloud object storage |
| Atlas Online Archive | Auto-tier cold data |
| Stable API | Pinned subset of commands a driver speaks |
| apiVersion: '1' | The first stable API version |
| deprecated commands | Commands removed from the Stable API |
| GridFS | A spec for storing big binary blobs across multiple chunk docs |
| x.509 auth | Mutual-TLS certificate authentication |
| SCRAM-SHA-1 | Salted Challenge Response Auth, SHA-1 variant |
| SCRAM-SHA-256 | The default password auth in modern versions |
| LDAP enterprise | Enterprise feature for LDAP auth |
| Kerberos | Enterprise feature for Kerberos auth |
| Field-Level Encryption (CSFLE) | Client-side encryption of specific fields |
| Queryable Encryption | Encrypted fields you can still query equality on (6.0+) |
| encryption at rest | The data files on disk are encrypted |
| KMIP | A key management protocol used for at-rest encryption |
| AWS KMS | Amazon's key management service |
| Azure KV | Azure Key Vault |
| GCP KMS | Google Cloud KMS |
| local key | A local key file for encryption (testing) |
| TLS | Transport-layer encryption between client and server |
| mongoexport | Tool to export a collection to JSON or CSV |
| mongoimport | Tool to import JSON/CSV/TSV into a collection |
| mongodump | Tool to dump a database to BSON |
| mongorestore | Tool to restore a BSON dump |
| mongostat | Tool that shows real-time per-second server stats |
| mongotop | Tool that shows per-namespace read/write times |
| mongofiles | Tool for working with GridFS files |
| mongoreplay (legacy) | Old tool for capturing and replaying traffic |
| mongoperf | Disk I/O benchmark utility |
| mongotemplate (Spring) | The Spring Data MongoDB abstraction in Java |
| motor (Python async) | The official asyncio Python driver |
| pymongo | The official synchronous Python driver |
| mongoose (Node) | A popular ODM for Node.js with schemas |
| mgo / mongo-go-driver | The Go drivers (mgo legacy, mongo-go-driver official) |
| ReactiveMongo (Scala) | A reactive Scala driver |
| MongoDB Realm (Atlas Device SDK) | The mobile sync product |

## Try This

You learn this by typing things, not by reading. Try these in order.

1. **Get a server.** `brew install mongodb-community && brew services start mongodb-community` on macOS, or follow the official Linux install for your distro. Verify with `mongosh`.
2. **Make a database.** `use playground`. Insert ten documents about anything — pets, books, songs.
3. **Filter, sort, project.** Find all documents where some field is greater than something. Sort by another field descending. Project away the `_id`.
4. **Update with `$set` and `$inc`.** Pick one document and increment a count field. Run it three times. Verify the count went up by 3.
5. **Run a real aggregation.** `$group` your collection by some field and count how many documents land in each group. Sort descending. Limit 5.
6. **Add an index.** Pick a field you filter on a lot. `createIndex({field: 1})`. `explain("executionStats")` your filter before and after. See `IXSCAN` replace `COLLSCAN`.
7. **Crash the unique constraint.** Add a unique index on email. Try to insert two documents with the same email. Read the `E11000` error word for word.
8. **Spin up a replica set.** Use `mongodb-shell` or the Atlas free tier. `rs.status()`. Watch the primary/secondary roles. Kill the primary. Watch a secondary become primary.
9. **Open a change stream.** In one terminal, `db.users.watch()`. In another, insert a document. Watch the event arrive in the first terminal in real time.
10. **Read the explain output.** Run `explain("executionStats")` on a slow query. Read every field. Understand what `totalDocsExamined` means and why it might be way bigger than `nReturned`.

You will be a competent MongoDB user after step 10.

## Where to Go Next

If you found this useful, here are the natural next stops in this cheat-sheet collection:

- **databases/mongodb** — the dense reference page (commands, operators, options).
- **databases/postgresql** — the SQL counterpart you should know either way.
- **databases/redis** — for caching layered in front of MongoDB.
- **databases/sql** — the language to learn if you have to talk to non-Mongo databases.
- **ramp-up/postgres-eli5** — the SQL ELI5 sister sheet.

After that, real-world skill comes from:

1. **Building a small project** — a CRUD app on top of MongoDB. Pick a language driver and just go.
2. **Reading the official docs** — they are excellent. Especially the Aggregation and Indexing sections.
3. **MongoDB University** — free courses, taught by MongoDB Inc., with hands-on labs.
4. **Reading post-mortems and design docs** — search for "MongoDB at <company> scale" on engineering blogs to see how big companies actually run this thing.

## See Also

- [databases/mongodb](../databases/mongodb.md)
- [databases/postgresql](../databases/postgresql.md)
- [databases/mysql](../databases/mysql.md)
- [databases/sqlite](../databases/sqlite.md)
- [databases/redis](../databases/redis.md)
- [databases/sql](../databases/sql.md)
- [databases/time-series](../databases/time-series.md)
- [databases/graph-databases](../databases/graph-databases.md)
- [ramp-up/postgres-eli5](postgres-eli5.md)
- [ramp-up/redis-eli5](redis-eli5.md)
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md)
- [ramp-up/tcp-eli5](tcp-eli5.md)
- [ramp-up/docker-eli5](docker-eli5.md)

## Version Notes

MongoDB has shipped major versions roughly once a year. The big landmarks:

- **3.6 (2017)** — Causal consistency via client sessions; change streams introduced.
- **4.0 (2018)** — Multi-document transactions on replica sets. License changed from AGPL v3 to **SSPL** (Server Side Public License). The license change was controversial — AWS, IBM, and others responded by forking older versions or building their own compatible APIs (Amazon DocumentDB, etc.).
- **4.2 (2019)** — Distributed transactions across sharded clusters. Retryable writes generally available. Field-Level Redaction.
- **4.4 (2020)** — Hidden indexes; `refineCollectionShardKey`; compound hashed shard keys; mirrored reads; union with `$unionWith`.
- **5.0 (2021)** — Time-series collections; window functions (`$setWindowFields`); the Stable API; live resharding.
- **6.0 (2022)** — Queryable Encryption; `$lookup` with sharded targets; change-stream pre/post images; `clusterToTime` for point-in-time reads.
- **7.0 (2023)** — Atlas Vector Search; change-stream pre/post images on by default in some configs; sharding-management UX overhaul.
- **8.0 (2024)** — Performance focus; new shard management; query optimizations; better TLS.

License history matters: until 2018 MongoDB was AGPL v3. In late 2018 the company switched to SSPL, which is **not OSI-approved.** Cloud providers responded by either keeping older AGPL versions or building independent compatible products. If you ship MongoDB to customers as part of a product, talk to a lawyer.

## References

- **mongodb.com/docs** — the canonical reference, organized by feature area. Especially good: the Aggregation reference and the Indexing tutorial.
- **MongoDB: The Definitive Guide** by Bradshaw, Brazil, and Chodorow — the standard book on MongoDB. Third edition covers 4.0+.
- **MongoDB University** (university.mongodb.com) — free courses with hands-on labs. M001 (Basics), M121 (Aggregation), M201 (Performance), M310 (Security), M312 (Diagnostics).
- **Atlas docs** (mongodb.com/docs/atlas) — the managed-service-specific reference.
- **Designing Data-Intensive Applications** by Martin Kleppmann — the chapter on document-vs-relational data models is the best single explanation of why each side is what it is. Read it.
- **MongoDB in Action** by Kyle Banker — older, but the worked-example style is excellent for a first read-through.
- **The MongoDB Engineering Blog** — for performance internals, storage engine notes, and architectural deep-dives written by the people who built it.
