# Elasticsearch (Distributed Search Engine)

Elasticsearch is a distributed RESTful search and analytics engine built on Apache Lucene, providing full-text search with custom analyzers, structured queries, aggregations, and index lifecycle management across horizontally scalable clusters of shards and replicas.

## Cluster Management

```bash
# Check cluster health
curl -s localhost:9200/_cluster/health?pretty

# Cluster stats
curl -s localhost:9200/_cluster/stats?pretty

# Node info
curl -s localhost:9200/_cat/nodes?v

# Shard allocation
curl -s localhost:9200/_cat/shards?v

# Pending tasks
curl -s localhost:9200/_cat/pending_tasks?v

# Allocation explain (debug unassigned shards)
curl -s -X POST localhost:9200/_cluster/allocation/explain?pretty

# Cluster settings
curl -s localhost:9200/_cluster/settings?include_defaults=true&pretty
```

## Index Management

```bash
# Create index with settings
curl -s -X PUT localhost:9200/products -H 'Content-Type: application/json' -d '{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "refresh_interval": "5s"
  }
}'

# List indices
curl -s localhost:9200/_cat/indices?v&s=index

# Get index settings
curl -s localhost:9200/products/_settings?pretty

# Get index mapping
curl -s localhost:9200/products/_mapping?pretty

# Delete index
curl -s -X DELETE localhost:9200/products

# Close/open index (save resources)
curl -s -X POST localhost:9200/products/_close
curl -s -X POST localhost:9200/products/_open

# Reindex
curl -s -X POST localhost:9200/_reindex -H 'Content-Type: application/json' -d '{
  "source": {"index": "products_v1"},
  "dest": {"index": "products_v2"}
}'

# Index aliases
curl -s -X POST localhost:9200/_aliases -H 'Content-Type: application/json' -d '{
  "actions": [
    {"remove": {"index": "products_v1", "alias": "products"}},
    {"add": {"index": "products_v2", "alias": "products"}}
  ]
}'
```

## Mappings

```bash
# Create explicit mapping
curl -s -X PUT localhost:9200/products -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "name":        {"type": "text", "analyzer": "standard"},
      "sku":         {"type": "keyword"},
      "description": {"type": "text", "analyzer": "english"},
      "price":       {"type": "float"},
      "category":    {"type": "keyword"},
      "tags":        {"type": "keyword"},
      "created_at":  {"type": "date", "format": "yyyy-MM-dd HH:mm:ss||epoch_millis"},
      "location":    {"type": "geo_point"},
      "metadata":    {"type": "object"},
      "variants":    {"type": "nested", "properties": {
        "color": {"type": "keyword"},
        "size":  {"type": "keyword"},
        "stock": {"type": "integer"}
      }}
    }
  }
}'

# Key field types:
# text     — full-text search (analyzed, tokenized)
# keyword  — exact match, sorting, aggregations (not analyzed)
# long/integer/float/double — numeric
# date     — date/time
# boolean  — true/false
# nested   — independent sub-documents (for arrays of objects)
# geo_point — latitude/longitude
```

## Custom Analyzers

```bash
# Index with custom analyzer
curl -s -X PUT localhost:9200/articles -H 'Content-Type: application/json' -d '{
  "settings": {
    "analysis": {
      "analyzer": {
        "custom_english": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "english_stop", "english_stemmer", "asciifolding"]
        },
        "autocomplete": {
          "type": "custom",
          "tokenizer": "autocomplete_tokenizer",
          "filter": ["lowercase"]
        }
      },
      "tokenizer": {
        "autocomplete_tokenizer": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20,
          "token_chars": ["letter", "digit"]
        }
      },
      "filter": {
        "english_stop": {"type": "stop", "stopwords": "_english_"},
        "english_stemmer": {"type": "stemmer", "language": "english"}
      }
    }
  }
}'

# Test analyzer
curl -s -X POST localhost:9200/articles/_analyze -H 'Content-Type: application/json' -d '{
  "analyzer": "custom_english",
  "text": "The quick brown foxes were jumping"
}'
```

## Search Queries

```bash
# Match query (full-text, analyzed)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {"match": {"name": "wireless headphones"}}
}'

# Bool query (compound)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {
    "bool": {
      "must":     [{"match": {"name": "headphones"}}],
      "filter":   [{"range": {"price": {"gte": 50, "lte": 200}}}],
      "should":   [{"term": {"category": "premium"}}],
      "must_not": [{"term": {"status": "discontinued"}}]
    }
  }
}'

# Term query (exact match on keyword fields)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {"term": {"sku": {"value": "SKU-12345"}}}
}'

# Range query
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {"range": {"created_at": {"gte": "2024-01-01", "lt": "2025-01-01"}}}
}'

# Nested query
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {
    "nested": {
      "path": "variants",
      "query": {
        "bool": {
          "must": [
            {"term": {"variants.color": "blue"}},
            {"range": {"variants.stock": {"gt": 0}}}
          ]
        }
      }
    }
  }
}'

# Multi-match (search across multiple fields)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "query": {
    "multi_match": {
      "query": "noise cancelling",
      "fields": ["name^3", "description", "tags^2"],
      "type": "best_fields"
    }
  }
}'
```

## Aggregations

```bash
# Terms aggregation (top categories)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "size": 0,
  "aggs": {
    "by_category": {"terms": {"field": "category", "size": 20}},
    "price_stats": {"stats": {"field": "price"}},
    "price_ranges": {
      "histogram": {"field": "price", "interval": 50}
    }
  }
}'

# Date histogram
curl -s -X POST localhost:9200/orders/_search -H 'Content-Type: application/json' -d '{
  "size": 0,
  "aggs": {
    "orders_over_time": {
      "date_histogram": {
        "field": "created_at",
        "calendar_interval": "month"
      },
      "aggs": {
        "total_revenue": {"sum": {"field": "total"}}
      }
    }
  }
}'

# Nested aggregation within filter
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "size": 0,
  "aggs": {
    "premium_products": {
      "filter": {"term": {"category": "premium"}},
      "aggs": {
        "avg_price": {"avg": {"field": "price"}}
      }
    }
  }
}'
```

## Index Lifecycle Management (ILM)

```bash
# Create ILM policy
curl -s -X PUT localhost:9200/_ilm/policy/logs_policy -H 'Content-Type: application/json' -d '{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {"max_age": "7d", "max_primary_shard_size": "50gb"},
          "set_priority": {"priority": 100}
        }
      },
      "warm": {
        "min_age": "30d",
        "actions": {
          "shrink": {"number_of_shards": 1},
          "forcemerge": {"max_num_segments": 1},
          "set_priority": {"priority": 50}
        }
      },
      "cold": {
        "min_age": "90d",
        "actions": {
          "freeze": {},
          "set_priority": {"priority": 0}
        }
      },
      "delete": {
        "min_age": "365d",
        "actions": {"delete": {}}
      }
    }
  }
}'

# Apply ILM policy to index template
curl -s -X PUT localhost:9200/_index_template/logs_template -H 'Content-Type: application/json' -d '{
  "index_patterns": ["logs-*"],
  "template": {
    "settings": {
      "index.lifecycle.name": "logs_policy",
      "index.lifecycle.rollover_alias": "logs"
    }
  }
}'
```

## Painless Scripting

```bash
# Update by script
curl -s -X POST localhost:9200/products/_update/1 -H 'Content-Type: application/json' -d '{
  "script": {
    "source": "ctx._source.price *= params.factor",
    "params": {"factor": 1.1}
  }
}'

# Script in search (runtime field)
curl -s -X POST localhost:9200/products/_search -H 'Content-Type: application/json' -d '{
  "runtime_mappings": {
    "price_with_tax": {
      "type": "double",
      "script": "emit(doc['\''price'\''].value * 1.2)"
    }
  },
  "fields": ["price_with_tax"]
}'
```

## Tips

- Use `keyword` for exact matches, filtering, and aggregations; use `text` only for full-text search
- Always set explicit mappings in production; dynamic mapping can create suboptimal field types
- Put filters in the `filter` context of bool queries; they skip scoring and are cached
- Shard count is fixed at index creation; plan ahead or use rollover aliases for growing data
- Avoid deep pagination with `from`+`size` beyond 10,000; use `search_after` or scroll API instead
- ILM automates hot/warm/cold/delete transitions; essential for time-series and log data
- Force-merge read-only indices to one segment for optimal search performance
- Monitor `_cat/shards` for unassigned shards; common causes are disk watermarks and allocation rules
- Use index aliases for zero-downtime reindexing and schema migrations
- Nested fields maintain document boundaries in arrays but add query complexity; flatten when possible
- Set `refresh_interval` to 30s or higher for write-heavy workloads; the default 1s impacts indexing throughput

## See Also

- dynamodb, neo4j, mongodb, redis

## References

- [Elasticsearch Reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Elasticsearch Query DSL](https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl.html)
- [Elasticsearch Aggregations](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-aggregations.html)
- [Index Lifecycle Management](https://www.elastic.co/guide/en/elasticsearch/reference/current/index-lifecycle-management.html)
- [Elasticsearch Mapping Types](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-types.html)
