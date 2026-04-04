# Neo4j (Graph Database)

Neo4j is a native graph database using the property graph model where nodes and relationships have labels and properties, queried with the Cypher declarative language, supporting ACID transactions, graph algorithms, full-text search, and the Bolt binary protocol.

## Installation and Connection

```bash
# Docker
docker run -d --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/password123 \
  -v neo4j-data:/data \
  neo4j:5

# Connect via cypher-shell
cypher-shell -u neo4j -p password123
cypher-shell -a bolt://localhost:7687 -u neo4j -p password123

# Neo4j browser: http://localhost:7474

# Check status
neo4j status
neo4j console                          # run in foreground
```

## Cypher Basics

### CREATE (Nodes and Relationships)

```cypher
// Create a node with label and properties
CREATE (p:Person {name: 'Alice', age: 30, email: 'alice@example.com'})
RETURN p;

// Create multiple nodes
CREATE (a:Person {name: 'Alice'}),
       (b:Person {name: 'Bob'}),
       (c:Company {name: 'Acme Corp'});

// Create a relationship
MATCH (a:Person {name: 'Alice'}), (b:Person {name: 'Bob'})
CREATE (a)-[:KNOWS {since: 2020}]->(b);

// Create node and relationship together
CREATE (a:Person {name: 'Carol'})-[:WORKS_AT {role: 'Engineer'}]->(c:Company {name: 'TechCo'});
```

### MATCH (Read Queries)

```cypher
// Find all people
MATCH (p:Person) RETURN p;

// Find by property
MATCH (p:Person {name: 'Alice'}) RETURN p;

// Find with WHERE clause
MATCH (p:Person) WHERE p.age > 25 AND p.age < 40 RETURN p.name, p.age;

// Follow relationships
MATCH (a:Person)-[:KNOWS]->(b:Person) RETURN a.name, b.name;

// Variable-length paths (1 to 3 hops)
MATCH (a:Person {name: 'Alice'})-[:KNOWS*1..3]->(b:Person)
RETURN DISTINCT b.name;

// Shortest path
MATCH p = shortestPath(
  (a:Person {name: 'Alice'})-[:KNOWS*..10]-(b:Person {name: 'Zara'})
)
RETURN p, length(p);

// All shortest paths
MATCH p = allShortestPaths(
  (a:Person {name: 'Alice'})-[:KNOWS*..10]-(b:Person {name: 'Zara'})
)
RETURN p;

// Pattern matching with direction
MATCH (p:Person)-[r:WORKS_AT]->(c:Company)
RETURN p.name, r.role, c.name;

// Optional match (left outer join equivalent)
MATCH (p:Person)
OPTIONAL MATCH (p)-[:LIVES_IN]->(city:City)
RETURN p.name, city.name;
```

### MERGE (Upsert)

```cypher
// Create if not exists, match if exists
MERGE (p:Person {name: 'Dave'})
ON CREATE SET p.created = datetime()
ON MATCH SET p.lastSeen = datetime()
RETURN p;

// Merge relationship
MATCH (a:Person {name: 'Alice'}), (b:Person {name: 'Bob'})
MERGE (a)-[:KNOWS]->(b);
```

### SET, REMOVE, DELETE

```cypher
// Update properties
MATCH (p:Person {name: 'Alice'})
SET p.age = 31, p.verified = true;

// Add a label
MATCH (p:Person {name: 'Alice'})
SET p:Employee;

// Remove property
MATCH (p:Person {name: 'Alice'})
REMOVE p.email;

// Remove label
MATCH (p:Person {name: 'Alice'})
REMOVE p:Employee;

// Delete node (must have no relationships)
MATCH (p:Person {name: 'Dave'})
DELETE p;

// Delete node and all relationships (DETACH)
MATCH (p:Person {name: 'Dave'})
DETACH DELETE p;

// Delete all data
MATCH (n) DETACH DELETE n;
```

### Aggregation and Ordering

```cypher
// Count
MATCH (p:Person) RETURN count(p);

// Group by with aggregation
MATCH (p:Person)-[:WORKS_AT]->(c:Company)
RETURN c.name, count(p) AS employees, avg(p.age) AS avg_age
ORDER BY employees DESC;

// Collect into list
MATCH (p:Person)-[:KNOWS]->(friend:Person)
RETURN p.name, collect(friend.name) AS friends;

// DISTINCT
MATCH (p:Person)-[:KNOWS]->(friend)
RETURN DISTINCT friend.name;

// LIMIT and SKIP
MATCH (p:Person) RETURN p.name ORDER BY p.name SKIP 10 LIMIT 20;

// UNION
MATCH (p:Person) RETURN p.name AS name
UNION
MATCH (c:Company) RETURN c.name AS name;
```

## Indexes and Constraints

```cypher
// Create index
CREATE INDEX person_name FOR (p:Person) ON (p.name);

// Composite index
CREATE INDEX person_name_age FOR (p:Person) ON (p.name, p.age);

// Full-text index
CREATE FULLTEXT INDEX person_search FOR (p:Person) ON EACH [p.name, p.bio];

// Unique constraint (also creates index)
CREATE CONSTRAINT person_email_unique FOR (p:Person) REQUIRE p.email IS UNIQUE;

// Node key constraint (composite unique + not null)
CREATE CONSTRAINT person_key FOR (p:Person) REQUIRE (p.name, p.email) IS NODE KEY;

// Existence constraint
CREATE CONSTRAINT person_name_exists FOR (p:Person) REQUIRE p.name IS NOT NULL;

// List all indexes and constraints
SHOW INDEXES;
SHOW CONSTRAINTS;

// Drop
DROP INDEX person_name;
DROP CONSTRAINT person_email_unique;
```

## APOC Procedures

```cypher
// Install APOC plugin (add to neo4j.conf or Docker env)
// NEO4J_PLUGINS='["apoc"]'

// Load JSON from URL
CALL apoc.load.json('https://api.example.com/data')
YIELD value
MERGE (p:Person {id: value.id}) SET p.name = value.name;

// Load CSV
LOAD CSV WITH HEADERS FROM 'file:///import/people.csv' AS row
CREATE (p:Person {name: row.name, age: toInteger(row.age)});

// Batch operations (periodic iterate)
CALL apoc.periodic.iterate(
  'MATCH (p:Person) WHERE p.needsUpdate = true RETURN p',
  'SET p.updated = datetime(), p.needsUpdate = false',
  {batchSize: 1000}
);

// Export to JSON
CALL apoc.export.json.all('/tmp/export.json', {});

// Generate UUID
RETURN apoc.create.uuid() AS uuid;

// Path expansion
MATCH (start:Person {name: 'Alice'})
CALL apoc.path.expandConfig(start, {
  relationshipFilter: 'KNOWS>',
  minLevel: 1,
  maxLevel: 3,
  uniqueness: 'NODE_GLOBAL'
})
YIELD path
RETURN path;
```

## Graph Algorithms (GDS)

```cypher
// Install Graph Data Science plugin
// NEO4J_PLUGINS='["graph-data-science"]'

// Create a projected graph
CALL gds.graph.project('social', 'Person', 'KNOWS');

// PageRank
CALL gds.pageRank.stream('social')
YIELD nodeId, score
RETURN gds.util.asNode(nodeId).name AS name, score
ORDER BY score DESC LIMIT 10;

// Community detection (Louvain)
CALL gds.louvain.stream('social')
YIELD nodeId, communityId
RETURN gds.util.asNode(nodeId).name AS name, communityId
ORDER BY communityId;

// Shortest path (Dijkstra)
MATCH (source:Person {name: 'Alice'}), (target:Person {name: 'Zara'})
CALL gds.shortestPath.dijkstra.stream('social', {
  sourceNode: source,
  targetNode: target
})
YIELD path
RETURN path;

// Betweenness centrality
CALL gds.betweenness.stream('social')
YIELD nodeId, score
RETURN gds.util.asNode(nodeId).name, score
ORDER BY score DESC LIMIT 10;

// Drop projected graph
CALL gds.graph.drop('social');
```

## neo4j-admin

```bash
# Dump database
neo4j-admin database dump neo4j --to-path=/backups/

# Load database
neo4j-admin database load neo4j --from-path=/backups/neo4j.dump

# Check consistency
neo4j-admin database check neo4j

# Import CSV (bulk import for initial load)
neo4j-admin database import full neo4j \
  --nodes=Person=/import/persons.csv \
  --relationships=KNOWS=/import/knows.csv

# Set initial password
neo4j-admin dbms set-initial-password newpassword

# Memory recommendations
neo4j-admin server memory-recommendation
```

## Configuration (neo4j.conf)

```ini
# Memory
server.memory.heap.initial_size=2g
server.memory.heap.max_size=4g
server.memory.pagecache.size=4g

# Network
server.bolt.listen_address=0.0.0.0:7687
server.http.listen_address=0.0.0.0:7474
server.https.listen_address=0.0.0.0:7473

# Enable APOC and GDS
dbms.security.procedures.unrestricted=apoc.*,gds.*
dbms.security.procedures.allowlist=apoc.*,gds.*
```

## Tips

- Use MERGE instead of CREATE when you need idempotent operations to avoid duplicate nodes
- Always create indexes on properties used in MATCH/WHERE clauses; unindexed lookups scan the entire store
- Use parameterized queries (`$param`) instead of string interpolation to leverage query plan caching
- DETACH DELETE is required for nodes with relationships; plain DELETE will fail
- Variable-length path queries (`*1..N`) can be expensive; always set an upper bound
- Use EXPLAIN/PROFILE before queries to check the execution plan and identify full scans
- The GDS library requires creating projected in-memory graphs before running algorithms
- APOC's `periodic.iterate` handles large batch operations without running out of memory
- Use `UNWIND` to process lists efficiently instead of multiple MATCH statements
- Bolt protocol (port 7687) is the production protocol; HTTP API is for browser and debugging only
- LOAD CSV is transactional per row by default; wrap in periodic commit for large imports

## See Also

- elasticsearch, dynamodb, redis, postgresql

## References

- [Neo4j Documentation](https://neo4j.com/docs/)
- [Cypher Manual](https://neo4j.com/docs/cypher-manual/current/)
- [APOC Documentation](https://neo4j.com/labs/apoc/)
- [Graph Data Science Library](https://neo4j.com/docs/graph-data-science/current/)
- [Neo4j Operations Manual](https://neo4j.com/docs/operations-manual/current/)
