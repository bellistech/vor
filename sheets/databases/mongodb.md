# MongoDB (Document Database)

Document-oriented NoSQL database storing flexible JSON-like documents (BSON), with rich query language, aggregation pipeline, secondary indexes, replica sets for HA, and sharding for horizontal scaling.

## Connection

```bash
# mongosh (modern shell)
mongosh
mongosh "mongodb://localhost:27017"
mongosh "mongodb://user:pass@host:27017/mydb?authSource=admin"
mongosh "mongodb+srv://cluster.example.net/mydb"

# Connection with TLS
mongosh --tls --tlsCertificateKeyFile /path/cert.pem \
  "mongodb://host:27017/mydb"
```

## CRUD Operations

### Insert

```bash
# Insert one
db.users.insertOne({ name: "Alice", age: 30, tags: ["admin", "dev"] })

# Insert many
db.users.insertMany([
  { name: "Bob", age: 25, tags: ["dev"] },
  { name: "Carol", age: 35, tags: ["ops"] }
])
```

### Find (Read)

```bash
# Find all
db.users.find()

# Find with filter
db.users.find({ age: { $gt: 25 } })

# Find one
db.users.findOne({ name: "Alice" })

# Projection (select fields)
db.users.find({ age: { $gt: 25 } }, { name: 1, age: 1, _id: 0 })

# Query operators
db.users.find({ age: { $gte: 25, $lte: 35 } })    # range
db.users.find({ tags: { $in: ["admin", "ops"] } })  # in array
db.users.find({ name: { $regex: /^A/i } })          # regex
db.users.find({ $or: [{ age: 25 }, { age: 35 }] })  # logical OR
db.users.find({ tags: { $size: 2 } })                # array size
db.users.find({ "address.city": "NYC" })             # nested field

# Sort, limit, skip
db.users.find().sort({ age: -1 }).limit(10).skip(20)

# Count
db.users.countDocuments({ age: { $gt: 25 } })
```

### Update

```bash
# Update one
db.users.updateOne(
  { name: "Alice" },
  { $set: { age: 31 }, $push: { tags: "lead" } }
)

# Update many
db.users.updateMany(
  { age: { $lt: 30 } },
  { $inc: { age: 1 } }
)

# Replace entire document
db.users.replaceOne(
  { name: "Bob" },
  { name: "Bob", age: 26, tags: ["dev", "senior"] }
)

# Upsert (insert if not found)
db.users.updateOne(
  { name: "Dave" },
  { $set: { age: 28 } },
  { upsert: true }
)

# Array operators
db.users.updateOne({ name: "Alice" }, { $push: { tags: "manager" } })
db.users.updateOne({ name: "Alice" }, { $pull: { tags: "dev" } })
db.users.updateOne({ name: "Alice" }, { $addToSet: { tags: "admin" } })
```

### Delete

```bash
db.users.deleteOne({ name: "Bob" })
db.users.deleteMany({ age: { $lt: 25 } })
db.users.drop()                              # drop entire collection
```

## Indexes

```bash
# Create indexes
db.users.createIndex({ name: 1 })                    # ascending
db.users.createIndex({ age: -1 })                    # descending
db.users.createIndex({ name: 1, age: -1 })           # compound
db.users.createIndex({ name: 1 }, { unique: true })  # unique
db.users.createIndex({ email: 1 }, { sparse: true }) # skip nulls

# Text index
db.articles.createIndex({ content: "text", title: "text" })
db.articles.find({ $text: { $search: "mongodb tutorial" } })

# Geospatial index
db.places.createIndex({ location: "2dsphere" })
db.places.find({
  location: {
    $near: {
      $geometry: { type: "Point", coordinates: [-73.97, 40.77] },
      $maxDistance: 1000    # meters
    }
  }
})

# TTL index (auto-expire documents)
db.sessions.createIndex({ createdAt: 1 }, { expireAfterSeconds: 3600 })

# List and drop indexes
db.users.getIndexes()
db.users.dropIndex("name_1")

# Explain query plan
db.users.find({ name: "Alice" }).explain("executionStats")
```

## Aggregation Pipeline

```bash
db.orders.aggregate([
  { $match: { status: "completed" } },
  { $group: {
      _id: "$customer_id",
      totalSpent: { $sum: "$amount" },
      orderCount: { $sum: 1 },
      avgOrder: { $avg: "$amount" }
  }},
  { $sort: { totalSpent: -1 } },
  { $limit: 10 },
  { $lookup: {
      from: "customers",
      localField: "_id",
      foreignField: "_id",
      as: "customer"
  }},
  { $unwind: "$customer" },
  { $project: {
      customerName: "$customer.name",
      totalSpent: 1,
      orderCount: 1,
      avgOrder: { $round: ["$avgOrder", 2] }
  }}
])

# Aggregation stages reference:
# $match    — filter documents
# $group    — aggregate by key ($sum, $avg, $min, $max, $push)
# $sort     — sort results
# $limit    — limit results
# $skip     — skip results
# $project  — reshape / include / exclude fields
# $lookup   — left outer join
# $unwind   — flatten arrays
# $addFields — add computed fields
# $bucket   — group into ranges
# $facet    — multiple pipelines in parallel
# $out      — write results to collection
# $merge    — merge results into collection
```

## Replica Sets

```bash
# Initialize replica set
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "mongo1:27017", priority: 2 },
    { _id: 1, host: "mongo2:27017", priority: 1 },
    { _id: 2, host: "mongo3:27017", priority: 1 }
  ]
})

# Check status
rs.status()
rs.isMaster()

# Add / remove members
rs.add("mongo4:27017")
rs.remove("mongo4:27017")

# Read from secondaries
db.users.find().readPref("secondaryPreferred")
# Modes: primary, primaryPreferred, secondary, secondaryPreferred, nearest
```

## Sharding

```bash
# Enable sharding on database
sh.enableSharding("mydb")

# Shard a collection
sh.shardCollection("mydb.orders", { customer_id: "hashed" })  # hashed
sh.shardCollection("mydb.logs", { timestamp: 1 })              # ranged

# Check sharding status
sh.status()
db.orders.getShardDistribution()

# Balancer
sh.startBalancer()
sh.stopBalancer()
sh.isBalancerRunning()
```

## Administration

```bash
# Database operations
show dbs
use mydb
db.stats()
db.getCollectionNames()
db.users.stats()

# User management
db.createUser({
  user: "appuser",
  pwd: "secret",
  roles: [{ role: "readWrite", db: "mydb" }]
})

# Backup and restore
mongodump --uri="mongodb://localhost:27017" --out=/backup/
mongorestore --uri="mongodb://localhost:27017" /backup/

# Export/import JSON
mongoexport --db=mydb --collection=users --out=users.json
mongoimport --db=mydb --collection=users --file=users.json
```

## Tips

- Always create indexes for fields used in queries, sorts, and joins; use `explain()` to verify index usage
- Use compound indexes that match your most common query patterns, with equality fields first, then sort, then range
- Choose hashed shard keys for even distribution; ranged shard keys for range queries on the shard key
- Use the aggregation pipeline instead of MapReduce; it is faster and supports more operations
- Set TTL indexes on session and log collections to auto-expire old documents without manual cleanup
- Use `$lookup` sparingly; if you join frequently, consider embedding the data or denormalizing
- Use `updateMany` with `$inc` for counters rather than read-modify-write to avoid race conditions
- Enable the profiler (`db.setProfilingLevel(1, { slowms: 100 })`) to find slow queries in production
- Use read preferences (`secondaryPreferred`) to offload read traffic from the primary in replica sets
- Limit document size to well under the 16 MB BSON limit; large documents hurt cache efficiency
- Use `$project` early in aggregation pipelines to reduce the data flowing through subsequent stages

## See Also

redis, postgresql, cassandra, clickhouse, elasticsearch

## References

- [MongoDB Documentation](https://www.mongodb.com/docs/manual/)
- [MongoDB Aggregation Pipeline](https://www.mongodb.com/docs/manual/core/aggregation-pipeline/)
- [MongoDB Indexes](https://www.mongodb.com/docs/manual/indexes/)
- [MongoDB Sharding](https://www.mongodb.com/docs/manual/sharding/)
- [mongosh Reference](https://www.mongodb.com/docs/mongodb-shell/)
