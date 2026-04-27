# Grafana — ELI5

> Grafana is the dashboard wall in a control room. The graphs draw themselves. Grafana doesn't store any of the data — it just asks other systems for it and paints pictures.

## Prerequisites

Before reading this, it really helps to have skimmed:

- [ramp-up/prometheus-eli5](prometheus-eli5.md) — because Prometheus is the most common thing Grafana asks for data.
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md) — because we will mention "service running on a port" a lot, and that sheet sets up what that means.
- [ramp-up/docker-eli5](docker-eli5.md) — because almost everybody runs Grafana from a Docker image these days.
- [ramp-up/kubernetes-eli5](kubernetes-eli5.md) — because the production way to deploy Grafana is via the Helm chart on Kubernetes.

You do not need to know any specific query language. We will introduce them one at a time. You do not need to know JavaScript, even though Grafana is a JavaScript app inside the browser. You do not need to know Go, even though Grafana the server is written in Go. All you need is "I can open a web browser and I can run a docker command."

If a word feels unfamiliar, look it up in the **Vocabulary** at the bottom. Every important word in this sheet is in that table with a one-line plain-English definition.

If a code block has a `$` at the start of a line, that means "type the rest of this line into your terminal." You do not type the `$`. The lines without a `$` are what your computer prints back at you. We call that "output."

## What Even Is Grafana

### The control room picture

Imagine you walk into the control room of a power station. There is a wall. The wall is covered in screens. Some screens show line graphs that wiggle up and down. Some screens show big numbers that flash red when they get too high. Some screens show maps with little dots on cities. Some screens show tables with rows and rows of warnings. The screens never stop updating. People sit in chairs in front of the wall and watch.

Where does the data on those screens come from? It does not live in the screens. The screens are just glass. The data is being pulled out of sensors all over the power station — temperature sensors, voltage meters, vibration monitors, security cameras, network switches, lab thermometers, weather stations. Each sensor has its own way of being asked. The temperature sensor speaks one language. The vibration monitor speaks a different language. The network switch speaks a third language.

In the middle of the control room is a clever computer. The computer knows how to talk to every sensor in its own language. When a screen on the wall says "show me the temperature graph for the last hour," the clever computer goes and asks the temperature sensor, gets the numbers back, draws them as a line, and pushes the line out to the screen. When a different screen says "show me how many alarms went off today," the computer goes and asks the alarm system, gets the count back, and pushes a big number to that screen.

**Grafana is that clever computer.**

The wall full of screens is a **dashboard.** Each individual screen is a **panel.** The sensors are called **datasources.** The languages they each speak are query languages. Grafana speaks every one of them. You write the queries, Grafana runs them, the answers come back, Grafana draws the graphs.

The most important thing to remember is this: **Grafana does not store any data.** No sensors live inside Grafana. No history is kept inside Grafana. Grafana is just glass and pencils. The data lives somewhere else. Grafana asks, draws, repeats.

### Why Grafana exists

Before Grafana, every monitoring system had its own bad dashboard. Nagios had a sad green-and-red web page. Cacti drew bad-looking graphs. Each cloud had its own panel. If you wanted to see CPU from one source, errors from another, and trace data from a third, you had to flip between three browser tabs and squint.

Grafana said: pick the data wherever it lives. We will draw it nicely in one place.

Today Grafana is the standard way teams "look at" everything. Almost every team running Prometheus also runs Grafana. Almost every team running Loki also runs Grafana. Most teams running cloud monitoring also use Grafana to overlay it with their on-prem data. There are tens of thousands of dashboards published for free at `grafana.com/grafana/dashboards`, ready to import.

### The three flavours

Grafana comes in three flavours. They look almost identical and the dashboards work the same way. The differences are pricing and a few extra features.

- **Grafana OSS** — the free, open-source version. Runs on your own server. Most of the world uses this. AGPLv3 license.
- **Grafana Enterprise** — the OSS version plus paid features: SAML SSO, reporting, fine-grained RBAC, white-labelling, support contract.
- **Grafana Cloud** — the hosted version. You don't run a server, you log in to grafana.net. Includes hosted Prometheus (Mimir), hosted Loki, hosted Tempo. There is a free tier.

For this sheet we will assume the OSS version, but everything we say applies to Enterprise and Cloud too.

### The three things Grafana actually does

Boil Grafana down and it does three jobs.

#### Job 1: Ask the datasource

You point Grafana at a datasource by giving it a URL and some credentials. Grafana now knows how to ask Prometheus, or Loki, or whatever. Every panel says "I want data from datasource X using query Y." Grafana sends the query to the datasource, the datasource answers, and Grafana keeps the answer in memory just long enough to draw it.

#### Job 2: Draw the panel

Grafana takes the answer (which is usually a series of timestamps with values) and renders it. Time-series graphs, bar charts, gauges, big numbers, tables, geomaps, heatmaps, log streams, traces. The drawing happens in your browser, in JavaScript, using a render engine called the **panel plugin.** Different plugin = different shape of drawing.

#### Job 3: Tell you when something is wrong

You define an **alert rule** that says "if this query crosses this threshold, fire." Grafana evaluates the rule on a schedule (every minute, every five seconds, whatever you set). When the rule fires, a **notification** goes out to a **contact point** (Slack, PagerDuty, email). On the way to the contact point, the alert passes through a **notification policy** that decides who actually gets paged. There can also be a **mute timing** that says "don't page anybody during the maintenance window" and a **silence** that says "don't page about this specific thing for the next hour."

That's it. Ask. Draw. Alert. Three jobs.

### A picture of all of it

```
+---------------------------------------------------------+
|                       YOU                                |
|              (looking at the dashboard wall)             |
+--------------------------+-------------------------------+
                           |
                           | clicks, time-range picks, 
                           | dashboard-variable changes
                           v
+---------------------------------------------------------+
|                     GRAFANA                              |
|                                                          |
|  +-------------+   +------------+   +----------------+   |
|  | Dashboards  |   | Panels     |   | Alert Rules    |   |
|  +-------------+   +------------+   +----------------+   |
|                                                          |
|        |                |                  |             |
|        +----------------+------------------+             |
|                         |                                |
|                  query string                            |
+-------------------------+--------------------------------+
                          |
                          v
+---------------------------------------------------------+
|                    DATASOURCES                           |
|                                                          |
|  Prometheus  Loki  Tempo  Mimir  Postgres  MySQL         |
|  Elasticsearch  InfluxDB  Graphite  CloudWatch           |
|  Azure Monitor  Stackdriver  ClickHouse  Snowflake       |
+---------------------------------------------------------+
```

Notice that all the data lives down at the bottom, in the datasources. Grafana is just the middle box. You at the top tell Grafana what to show. Grafana tells the datasources what to fetch. Datasources answer. Grafana draws.

## Datasources

A datasource is a thing Grafana can ask for data. You configure it once with a URL and credentials and then any panel on any dashboard can pick it.

Each datasource speaks its own query language. Prometheus speaks **PromQL.** Loki speaks **LogQL.** Tempo speaks **TraceQL.** Postgres speaks **SQL.** Elasticsearch speaks **Lucene** or **KQL.** InfluxDB speaks **Flux** or **InfluxQL.** Graphite speaks Graphite functions. CloudWatch has its own metrics math. Azure Monitor has KQL too. Google Cloud Operations (formerly Stackdriver) has its own metric filter language. Each one is a slightly different dialect, but Grafana shows you a query editor that helps.

Let's go through the most common ones.

### Prometheus

Prometheus is a metrics database. It is the most common Grafana datasource. It scrapes metrics from your services every 15 seconds and keeps them as **time series.** A time series is a sequence of (timestamp, value) pairs with a name and some labels.

You ask Prometheus questions in PromQL:

```
rate(http_requests_total[5m])
```

That means: take the metric called `http_requests_total`, look at the last 5 minutes, and compute the per-second rate.

Grafana sends that string to Prometheus, Prometheus answers with a list of time series, Grafana draws each series as a line.

```
+------------------+              +-------------------+
| Grafana panel    |              | Prometheus        |
| query: rate(...) |  HTTP GET    | /api/v1/query     |
|                  | -----------> | _range             |
|                  |              |                   |
|                  | <----------- | answer: []series  |
|                  | JSON         |                   |
+------------------+              +-------------------+
```

You configure Prometheus as a datasource by giving Grafana the URL `http://prometheus:9090` (the default Prometheus port). You can also tell Grafana to send traces and exemplars along with the metrics so you can click from a graph spike right into a trace.

### Loki

Loki is the logs database. Made by the same company (Grafana Labs) as Grafana. Loki only indexes labels (like `app=nginx, env=prod`), not the log lines themselves. That makes it cheap. You ask it questions in LogQL:

```
{app="nginx"} |= "500" | json | line_format "{{.method}} {{.path}}"
```

Translation: "give me logs from the nginx app, only the lines containing the substring 500, parse them as JSON, then format each line as method-then-path."

LogQL also supports metric-style queries on top of logs:

```
sum by (status) (rate({app="nginx"}[1m]))
```

That counts how many lines per second nginx produced, grouped by HTTP status. So Loki is partly a logs system and partly a "logs that look like metrics" system.

In Grafana, Loki shows up in two main panel types: the **Logs** panel (raw lines) and the **Time series** panel (when you use the metric-style version of LogQL). And both can sit in the same dashboard.

### Tempo

Tempo is the traces database. Also from Grafana Labs. It stores **distributed traces**: sequences of spans showing how a request flowed through many services.

You ask Tempo questions in TraceQL:

```
{ duration > 500ms && service.name = "checkout" }
```

That returns all traces longer than 500 milliseconds where one of the services involved was called "checkout."

The big trick with Tempo is **trace ID linking.** You can configure your Loki datasource so that when a log line contains a trace ID, Grafana automatically renders the trace ID as a clickable link that pulls up the trace in Tempo. Logs to traces to metrics in two clicks. That is the whole point of the Grafana-Loki-Tempo set.

### Mimir

Mimir is the long-term metrics store. Also Grafana Labs. It is a Prometheus-compatible TSDB that scales horizontally. From Grafana's point of view it looks exactly like Prometheus — same PromQL, same `/api/v1/query`. The difference is that Mimir can hold years of data and tens of millions of active series.

> **Version note:** Mimir replaced an older project called **Cortex.** If you read old blog posts that say "Cortex," they are talking about what is now Mimir. Cortex itself still exists, but Grafana Labs's recommended scalable Prometheus is now Mimir.

### Postgres, MySQL, MS SQL Server

Grafana can query SQL databases directly. You give it a connection string and it lets you write SQL. The trick is that for time-series panels, your SQL has to return a column called `time` (or aliased to time) and one or more numeric value columns. Like:

```sql
SELECT
  time_bucket('1 minute', created_at) AS time,
  count(*) AS orders
FROM orders
WHERE $__timeFilter(created_at)
GROUP BY 1
ORDER BY 1
```

The `$__timeFilter()` macro is Grafana's way of injecting "WHERE time is between the dashboard's currently picked time range." That way the query auto-respects whatever the user has selected at the top of the dashboard.

Postgres works the same as MySQL works the same as MS SQL Server works the same as MariaDB. Different SQL dialects, same Grafana wrapper.

### Elasticsearch and OpenSearch

Elasticsearch (and its open-source fork OpenSearch) is for log search and full-text. You query in the Elasticsearch DSL (JSON) or in **Lucene** syntax (a one-line query language) or in **KQL** (Kibana Query Language, a saner one-line dialect). Grafana shows you a query builder where you pick the index, the time field, the metric (count, average, sum, percentile), and the filter.

This is mostly used for tail-of-logs panels and for ad-hoc searches in **Explore** mode.

### InfluxDB

InfluxDB is a time-series database from a different company (InfluxData). Speaks **InfluxQL** (SQL-like) or **Flux** (a functional pipeline language). Common in IoT and building monitoring. Grafana has full first-class support and a query builder.

### Graphite

Graphite is the original time-series database from way back in the late 2000s. Still alive at lots of older shops. Has its own function syntax: `aliasByNode(scaleToSeconds(myhost.cpu.user, 1), -1)`. Grafana has a tree-style query editor where you click metric names.

### CloudWatch

CloudWatch is AWS's monitoring service. Grafana queries it with the CloudWatch SDK. You pick a region, a namespace (like `AWS/EC2` or `AWS/RDS`), a metric, dimensions, statistic. Grafana also supports **CloudWatch Logs Insights** for log queries with its own query syntax (kinda SQL-like).

### Azure Monitor

Azure Monitor is Microsoft Azure's monitoring service. It speaks **KQL** (Kusto Query Language). Grafana has a built-in Azure Monitor datasource that handles the OAuth and lets you query Application Insights, Log Analytics, and the Metrics API.

### Google Cloud Monitoring (Stackdriver)

Used to be called Stackdriver, now called Google Cloud Operations Suite or just Cloud Monitoring. Grafana has a datasource. You authenticate with a service account JSON, pick a project, pick a metric type. Same idea, different cloud.

### A picture of the fan-out

```
                    +----------+
                    | GRAFANA  |
                    +----+-----+
                         |
   ----------------------+----------------------
   |     |      |     |      |       |       |
   v     v      v     v      v       v       v
[Prom][Loki][Tempo][Mimir][PG/MySQL][ES][CloudWatch]
  |     |      |     |      |       |       |
  v     v      v     v      v       v       v
metrics logs traces metrics rows  events  cloud
                            (any                metrics
                            shape)
```

Grafana is the trunk. Each datasource is a branch. Each panel is a leaf.

## Panels

A panel is one box on a dashboard. Each panel has one query (or several) and one renderer that turns query results into a picture. Grafana ships about a dozen built-in panel types and you can install more from the plugin catalog.

Here is a tour of the ones you will use 95% of the time.

### Time series

The default. A line graph with time on the X axis and one or more numeric values on the Y axis. Each series in the result becomes a line. Used for almost everything: CPU usage, request rate, latency, queue depth, sales-per-minute.

```
   ^
   |        /\        /\
   |       /  \      /  \      /\
   |      /    \    /    \    /  \____
   |  ___/      \__/      \__/
   +---------------------------------> time
```

You can stack lines, fill underneath them, draw them as bars, switch to logarithmic scale, set min/max, add thresholds (horizontal red/yellow lines), and a thousand other tweaks.

### Stat

A single big number, optionally with a sparkline (tiny line graph) underneath. Used for things like "current CPU %", "uptime", "users online right now."

```
   +---------------------+
   |                     |
   |       42.7%         |
   |    ___/\__/\_____   |
   |     CPU usage       |
   |                     |
   +---------------------+
```

### Gauge

A semicircle dial like a fuel gauge. Same data as Stat (a single number) but rendered with the dial graphic and a coloured zone (green / yellow / red).

```
        ___
       /   \
      / ↗   \
     |   75% |
     |       |
      \_____/
```

### Bar chart

X-axis is categorical (not time). Y-axis is numeric. Used for "top 10 endpoints by request count" or "errors per service."

### Bar gauge

A horizontal bar that fills up like a progress bar. Also has thresholds. Often used in lists: "DB1: ████░░░░ 40%, DB2: ██████░░ 65%, DB3: ███████░ 80%."

### Table

Rows and columns of numbers and strings. You can apply per-column formatting, value mappings (turn 0 into "OK" and 1 into "FAIL"), data links, and cell colouring.

### Heatmap

A grid where the X axis is time, the Y axis is buckets, and each cell is shaded by how many samples landed in that bucket. Used for **histograms over time** — like "show me the distribution of request durations every minute."

```
   bucket
   high  | . . . X X . .
         | . X X X X X .
         | . X X X X X .
         | X X X X X X X
   low   | X X X X X X X
         +-----------------> time
              dark = many samples
```

### Geomap

A world map with dots, lines, or shaded countries. Plug in latitude and longitude columns from your query and you get a map of where your traffic, customers, or sensors are.

### Logs

A scrolling stream of log lines, with filtering and de-duplication options. Pairs naturally with Loki, but works with Elasticsearch and CloudWatch Logs too.

### Trace

Renders a single distributed trace as a flame-graph-like waterfall, showing each span's start time, duration, and parent. Pairs with Tempo, Jaeger, and Zipkin.

### Status history

A horizontal timeline showing the state of something over time, coloured by category. Like "service A status: green green green YELLOW YELLOW RED RED green green." Useful for SLO panels.

### A panel rendering pipeline

When you save a dashboard with a panel, the lifecycle of one frame goes like this:

```
[browser opens dashboard]
        |
        v
[Grafana frontend loads dashboard JSON]
        |
        v
[for each panel: send the query to the datasource via /api/ds/query]
        |
        v
[datasource answers with rows of (time, value, labels)]
        |
        v
[transformations applied (rename, calc, join, etc.)]
        |
        v
[overrides applied (per-series colour, formatting)]
        |
        v
[panel plugin draws the picture]
        |
        v
[user sees the panel]
        |
        v
[wait refresh interval, do it all again]
```

Each step is configurable. Transformations live on the panel's "Transform data" tab. Overrides live on the panel's "Overrides" tab. Refresh interval is set on the dashboard.

## Dashboards

A dashboard is just a JSON file. The JSON contains:

- A title
- A list of panels (each with its query, its visualisation, its position on the grid)
- A list of dashboard variables
- A time range default (e.g. "now-6h")
- A refresh setting (e.g. "30s")
- A list of dashboard links and panel links
- Tags

You can edit dashboards in the Grafana UI by dragging panels around and clicking buttons. Or you can edit the JSON directly. Both end up in the same place.

### Variables

A variable is a named placeholder in a query that the user can change at the top of the dashboard.

For example, you might have a variable called `host` that holds the name of a server. Your panel queries reference it as `$host`. The dashboard renders a dropdown at the top with all the available hostnames. When the user picks a different host, every panel's query is re-run with the new value substituted in. The same dashboard now shows that host's data.

Types of variable:

- **Query variable** — populated by running a query against a datasource (e.g. `label_values(node_uname_info, instance)` to get a list of all Prometheus instances).
- **Custom variable** — a fixed list of values you type in.
- **Interval variable** — a list of time intervals (`1m, 5m, 1h, 1d`) so the user can pick the granularity.
- **Datasource variable** — lets the user switch which datasource the dashboard queries (great when you have one Grafana with multiple Prometheus instances per environment).
- **Ad-hoc filter** — automatically becomes label-filter clauses on every PromQL query. The user types `env=prod` at the top and every panel's query gets filtered.

### Repeating panels and rows

If a variable has multiple values selected, you can tell a panel to **repeat** itself — once per variable value. So if `host` is selected as `web1, web2, web3`, the dashboard renders the panel three times: once for web1, once for web2, once for web3. The panel is templated; the user picks how many copies they want.

You can also repeat **rows.** A row is a horizontal divider with multiple panels under it. Repeating a row repeats the whole row of panels per variable value.

### Dashboard links and panel links

A dashboard link is a button at the top of the dashboard that jumps to another dashboard, optionally passing variables. A panel link is a clickable hotspot inside a panel that jumps to another dashboard or to an external URL, optionally passing the value the user clicked. Together these form your **drilldown** structure: click a panel value, land on the dashboard for that value.

### Variable templating in pictures

```
USER picks $host = "web2"   in the dropdown
                |
                v
GRAFANA substitutes $host into every panel's query:
                |
                v
   panel A:  cpu_usage{host="$host"}    ->   cpu_usage{host="web2"}
   panel B:  mem_usage{host="$host"}    ->   mem_usage{host="web2"}
   panel C:  disk_usage{host="$host"}   ->   disk_usage{host="web2"}
                |
                v
each panel re-renders with web2's data
```

That is templating. One dashboard becomes many dashboards by varying the variables.

### The JSON model

Click the dashboard's settings gear, then "JSON Model." You will see something like:

```json
{
  "title": "Production Health",
  "uid": "abc123def",
  "schemaVersion": 38,
  "panels": [
    {
      "id": 1,
      "type": "timeseries",
      "title": "HTTP rate",
      "targets": [
        { "datasource": "Prometheus", "expr": "rate(http_requests_total[5m])" }
      ],
      "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 }
    }
  ],
  "templating": {
    "list": [
      {
        "name": "host",
        "type": "query",
        "datasource": "Prometheus",
        "query": "label_values(up, instance)"
      }
    ]
  },
  "time": { "from": "now-6h", "to": "now" },
  "refresh": "30s"
}
```

This is the source of truth for the dashboard. You can store it in Git. You can write tools that generate it. You can hand-edit it.

> **Version note:** Grafana 11 introduced **schema v2** and a new dashboard engine called **Scenes.** Older dashboards still load via the legacy schema; new features go in v2 first. If you write dashboards as code, target whichever schema your Grafana version supports.

### Provisioning

If you store dashboards in Git and want them to land in Grafana automatically, use **provisioning.** You drop YAML files in `/etc/grafana/provisioning/` and Grafana picks them up at startup.

`provisioning/dashboards/main.yaml`:

```yaml
apiVersion: 1
providers:
  - name: 'default'
    folder: 'Production'
    type: file
    options:
      path: /var/lib/grafana/dashboards
```

`provisioning/datasources/main.yaml`:

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    url: http://prometheus:9090
    access: proxy
    isDefault: true
```

The dashboards in `/var/lib/grafana/dashboards/` (each one a JSON file) appear in Grafana under the "Production" folder. You never had to click in the UI. The dashboards are version-controlled and reproducible.

## Folders and RBAC

A **folder** is a way to group dashboards. Like Drive folders. Folders are flat — there are no sub-folders in OSS Grafana (Grafana 10 introduced nested folders behind a feature flag, generally available in 11).

Permissions attach to folders. You grant a team or a user "viewer", "editor", or "admin" on a folder, and they get that level on every dashboard inside it. This is the basis of **RBAC** (role-based access control).

Roles in Grafana:

- **Server Admin** — controls the whole Grafana server. Manages organizations, users, plugins. Can do anything.
- **Org Admin** — controls one organization (Grafana supports multiple isolated orgs per server). Manages folders, datasources, users in that org.
- **Editor** — can create, edit, and delete dashboards in folders they have access to.
- **Viewer** — can only view dashboards. Cannot save or edit.

> **Enterprise version note:** Grafana Enterprise adds **fine-grained permissions** so you can grant individual permissions like "can edit alerts" or "can view this specific datasource" without giving the whole role.

```
                  +-----------+
                  |  SERVER   |
                  +-----+-----+
                        |
            +-----------+-----------+
            |                       |
       +----+----+             +----+----+
       |  ORG 1  |             |  ORG 2  |
       +----+----+             +----+----+
            |                       |
   +--------+-------+        +------+-------+
   | Folder: prod   |        | Folder: dev  |
   +--------+-------+        +------+-------+
            |                       |
   +--------+--------+               |
   |   Dashboards    |       (etc.)  |
   +-----------------+
```

That is the RBAC scope tree. Server > Org > Folder > Dashboard. Permissions can attach at any level and inherit downward.

## Alerting

Grafana has alerting built in. It will evaluate a query on a schedule, and if the result crosses a threshold, it routes a notification.

### Unified alerting

Since Grafana 8, all alerting goes through one system called **unified alerting.** This system uses Prometheus's Alertmanager model under the hood, even when you alert on non-Prometheus datasources. Old Grafana had a "legacy alerting" engine where each panel had its own alert; that is gone in OSS by default starting Grafana 9, and you can migrate the old rules in.

### Alert rule

An alert rule has these parts:

- **Query** — what to evaluate. Can target any datasource.
- **Expression** — optional math/logic on top of the query (e.g. "sum these series, then check > 100").
- **Condition** — when does it fire? Usually "is the latest value > X" or "is the average over 5m > X."
- **Evaluation interval** — how often Grafana runs the query (every 1m by default).
- **For** — how long the condition must be true before the alert actually fires (e.g. "fire only if the condition holds for 5 minutes").
- **Labels** — like Prometheus, key/value pairs attached to the alert. Used for routing.
- **Annotations** — human-readable description and summary, including templated values.

### Alert states

```
        +-------+
        | Normal|
        +---+---+
            |
            | condition becomes true
            v
        +-------+
        |Pending|     waiting out the "for" duration
        +---+---+
            |
            | "for" duration elapses
            v
        +--------+
        |Alerting|    fires; notifications sent
        +---+----+
            |
            | condition becomes false
            v
        +-------+
        | Normal|
        +-------+

   Side states:
        +-------+    no data returned by query
        | NoData|
        +-------+
        +-------+    query errored
        | Error |
        +-------+
```

### Contact points

A **contact point** is "where do alerts go?" Examples: a Slack webhook, a PagerDuty integration key, an email server, a Microsoft Teams webhook, a Webhook URL, OpsGenie, VictorOps, Discord, Telegram. You can have many contact points.

### Notification policy

A **notification policy** is the routing tree. It says "alerts with label `team=payments` go to the payments-pager contact point. Alerts with label `severity=info` go to email only. Everything else goes to the on-call channel." It looks like a tree:

```
                +-----------------+
                |   Root policy   |
                | -> default-team |
                +--------+--------+
                         |
        +----------------+------------------+
        |                                   |
+-------+-------+                  +--------+-------+
| match team=   |                  | match severity= |
| payments      |                  | info            |
| -> pay-pager  |                  | -> info-email   |
+---------------+                  +-----------------+
```

### Mute timings

A **mute timing** is a calendar window during which alerts are suppressed. You define the window once (like "Saturdays from 02:00 to 06:00 for maintenance") and attach it to a notification policy. During that window, alerts that would have routed there don't notify. They are still firing internally, but no page goes out.

### Silences

A **silence** is a one-off "don't notify about this alert from this label set, until this date." Like "silence everything matching `app=db1` until tomorrow noon because we know it's broken." Silences are usually clicked through the UI when something breaks.

### Unified alerting topology

```
+----------------------------------------------------------+
|                       GRAFANA                             |
|                                                           |
|  +-------------+    +-------------+    +-------------+    |
|  | Alert Rules |--->|  Evaluator  |--->| Internal    |    |
|  +-------------+    +-------------+    | Alertmanager|    |
|                                        +------+------+    |
+----------------------------------------------+-----------+
                                                |
                            +-------------------+----+
                            | Notification policies   |
                            +-------------------+----+
                                                |
                  +---------+---------+---------+---------+
                  |         |         |         |         |
                  v         v         v         v         v
                Slack   PagerDuty  Email  MS Teams  Webhook
```

Same picture, every alert.

### Recording rules

A **recording rule** runs a query on a schedule and writes the result back into the datasource as a new metric. So "the rate over 5 minutes of HTTP requests" can become a stored metric called `instance:http_requests:rate5m`. Dashboards then query that pre-computed metric instead of recomputing rate every time. Cheaper and faster. Recording rules are part of the same alerting subsystem.

## Explore Mode

**Explore** is a single-panel ad-hoc query screen. Open `/explore` in the Grafana URL. Pick a datasource. Type a query. You get the result. No dashboard required.

This is where you go when you do not yet know what dashboard to build. You poke the data, find the right query, then click "Add to dashboard" and pick which dashboard it should land in.

> **Version note:** Grafana 9 added "Explore to Dashboard" which moves the query you just wrote into a new panel on a target dashboard with one click. Saves a lot of copy-paste.

You can also run **split view** in Explore: two queries side by side, one against logs, one against metrics, with synchronized time. That is the easiest way to see "the spike in errors at 14:32 in the logs lined up with the latency rise in the metrics."

## Plugins

Grafana is extensible via plugins. There are three kinds:

- **Panel plugins** — new visualisations (e.g. pie chart, status panel, table heatmap).
- **Datasource plugins** — connectors to new data backends (e.g. ClickHouse, Snowflake, Splunk).
- **App plugins** — bundled experiences that combine panels, datasources, and dashboards into one product (e.g. the Kubernetes Monitoring app, the Synthetic Monitoring app).

Plugins are listed in the Grafana Plugin Catalog at `grafana.com/plugins`. You install them with the CLI:

```
$ grafana-cli plugins install grafana-piechart-panel
```

Then restart Grafana. The new panel type or datasource appears.

### Plugin signatures

Grafana checks every plugin for a signature from Grafana Labs or a verified publisher. If the signature is missing or invalid, Grafana refuses to load it by default. You can override that with:

```
[plugins]
allow_loading_unsigned_plugins = my-custom-panel
```

in `grafana.ini` — but only do that for your own internal plugins, never random downloads from the internet.

## Provisioning

We touched on this above but here is the full picture. Provisioning is how you express Grafana's configuration as YAML files that ship in your config-management system. Grafana reads them at startup.

The directories live under `/etc/grafana/provisioning/` (configurable via `[paths]` in `grafana.ini`):

- `datasources/` — `*.yaml` files defining datasources.
- `dashboards/` — `*.yaml` files describing where dashboard JSON files live.
- `alerting/` — `*.yaml` files defining alert rules, contact points, notification policies, mute timings.
- `plugins/` — `*.yaml` files declaring plugin app configurations.

A provisioned datasource is read-only in the UI — Grafana shows a banner saying "this datasource was provisioned." That is intentional. You only change provisioned things by editing the YAML and re-deploying.

You can also use **dashboard-as-code** tools to generate the JSON:

- **Grafonnet** — Jsonnet library for writing dashboards.
- **Grizzly** (`gr`) — CLI that pushes dashboards, datasources, alerts to a Grafana instance from local files.
- **Tanka** — Jsonnet config tool used by Grafana Labs themselves.
- **terraform-provider-grafana** — Terraform resources for everything Grafana exposes via API.

These tools all end up calling the same `/api/dashboards/db` endpoint underneath. Grafonnet generates JSON, Grizzly pushes JSON, Terraform pushes JSON. The endpoint is the same.

## SAML, OAuth, LDAP

Grafana can authenticate users from several places. By default it uses its own internal user database (`grafana.db`, a SQLite file). For production you usually plug in an existing identity provider.

### SAML SSO (Enterprise)

SAML is the SSO protocol big enterprises use (Okta, Azure AD, Auth0). Grafana Enterprise speaks SAML. You upload the IdP's metadata XML, configure the assertion mappings (which SAML attribute is the email, which one is the role), and now users log in via your existing SSO. **OSS does not include SAML — that's Enterprise only.**

### Generic OAuth

Grafana speaks OAuth2 / OIDC out of the box for OSS. You configure `[auth.generic_oauth]` with the auth URL, token URL, client ID, client secret, scopes, and role-mapping rules. Works with GitHub, GitLab, Google, Keycloak, Auth0, Okta (OAuth flavour), Azure AD (OAuth flavour). User clicks "Sign in with X," gets bounced to the IdP, comes back with a token, gets logged in.

You can also map IdP groups to Grafana orgs and roles via `role_attribute_path` (a JMESPath expression on the OIDC claim).

### LDAP

Grafana speaks LDAP / Active Directory. Configure `ldap.toml` with your LDAP server's host, port, bind DN, bind password, and the search filter. Map LDAP groups to Grafana orgs and roles. Users log in with their corporate username and password, Grafana asks LDAP, LDAP says yes/no.

LDAP is older and clunkier than OAuth/SAML. Use it if your shop is heavily AD-bound.

### Team Sync (Enterprise)

Team Sync syncs IdP groups to Grafana teams automatically. No more manual team membership.

### Anonymous access and public dashboards

You can let unauthenticated visitors view a dashboard with `[auth.anonymous] enabled = true`. Or you can mark individual dashboards as **public** (Grafana 10+) which gives them a shareable URL without needing a Grafana login. Use sparingly — both are easy ways to leak data.

## Image Renderer

By default Grafana cannot render PNGs of dashboards on its own — that needs a headless browser. The **Image Renderer** is a separate service (a Node.js + Chromium process) that Grafana talks to over HTTP. You install it as a plugin or run it as a sidecar:

```
$ docker run -d --name=renderer --net=host grafana/grafana-image-renderer:latest
```

Then point Grafana at it via the `GF_RENDERING_SERVER_URL` env var. Now Grafana can:

- Generate panel PNGs on demand (the "share -> direct image" link).
- Send PNG attachments in alert notifications (so the Slack alert includes a graph).
- Generate scheduled PDF reports (Enterprise — the **Reporting** feature).

## Reporting (Grafana Enterprise)

Reporting is Enterprise. It schedules a dashboard render to PDF on a cron and emails it to a list. Useful for execs who want a daily or weekly summary in their inbox without logging in. Behind the scenes it uses the Image Renderer.

## Grafana OnCall

Grafana OnCall is a dedicated incident response tool. Free tier exists. It handles on-call schedules, escalation paths, paging, and post-incident retros. It plugs into the same Grafana you already run. Think PagerDuty but Grafana-native and open-source-friendly.

OnCall is separate from the alerting system: alerts in Grafana fire to a contact point, the contact point is OnCall, OnCall pages the right human according to schedules. Grafana sees "send a notification." OnCall handles the human side.

## Cloud vs On-Prem

### On-Prem

You run Grafana yourself. Common deployments:

- One container with `grafana/grafana:latest` and a SQLite database (small/dev).
- One pod via Helm chart with an external Postgres or MySQL for the Grafana metadata DB (medium).
- Stateful set behind an ingress with persistent storage and high availability (large; HA needs an external DB).

You install plugins yourself, you upgrade the binary yourself, you keep certs renewed yourself.

### Grafana Cloud

You don't run anything. You log in to grafana.net, get a Grafana URL like `<your-stack>.grafana.net`. Grafana Cloud bundles hosted Prometheus (Mimir), hosted Loki, hosted Tempo, and OnCall. There is a free tier with 10k metrics, 50 GB of logs, etc. You point your apps at the Cloud endpoints with API keys. Grafana Cloud is ideal for small teams who don't want to run a TSDB.

There is also **Grafana Cloud Pro** and **Grafana Cloud Advanced** with bigger quotas and more features.

## Common Errors

These are exact strings you will see in logs or in the UI. Fixes alongside.

```
failed to fetch frontend assets
```
Grafana cannot reach `grafana-storage` for its CSS/JS bundle. Usually caused by a wrong `[paths] static_root_path` in `grafana.ini`, or a corrupted install. Fix: re-install Grafana, or restore the `public/` directory.

```
query returned no data points
```
Your query is valid PromQL/LogQL/etc but the datasource has nothing to return for the given time range. Either no data exists (check the metric name with `up` or `{__name__=~".+"}`), or the time range is wrong. Check the dashboard time picker.

```
datasource not found
```
A panel references a datasource UID that doesn't exist (anymore). Probably the dashboard was exported from one Grafana and imported into another with a different datasource name. Open the panel, pick the correct datasource from the dropdown, save.

```
data source proxy error
```
Grafana proxies the request to the datasource on the backend. Something between Grafana and the datasource is broken. Common causes: wrong URL in the datasource config, datasource is down, firewall is blocking, TLS cert is expired. Check `journalctl -u grafana-server` for the underlying error.

```
failed to load dashboard JSON
```
The dashboard JSON is malformed or references a schema version Grafana doesn't recognise. Open it in a JSON validator. Check `schemaVersion` is supported by your Grafana version.

```
A query is required to use this datasource
```
You picked a datasource on a panel but didn't actually write a query. Add a query expression in the query editor.

```
Permission denied: User does not have access
```
RBAC is denying the user. Either they aren't in a team with access to the folder, or their role is too low. Check the folder's "Permissions" tab.

```
License has expired
```
Grafana Enterprise license is past its end date. Renew with Grafana Labs and replace the `license.jwt` file in the data directory.

```
Failed to send notification: dial tcp ... connect: connection refused
```
Grafana tried to send to a contact point (Slack, PagerDuty, SMTP) and the network refused. Check the contact point URL is reachable from the Grafana host. `curl <url>` from the same machine.

```
Timed out waiting for response from data source
```
The datasource took too long to answer. Either the query is too heavy (reduce time range, add `topk()`), or the datasource is overloaded, or there's a network slowdown. Increase the per-datasource timeout in the datasource config.

## Hands-On

These are real commands. Type them. Watch them work.

```
$ docker run -d -p 3000:3000 --name=grafana grafana/grafana
```
Pull and start a Grafana container. Open `http://localhost:3000` in your browser. Default login is `admin` / `admin`. You'll be prompted to change the password on first login.

```
$ docker logs -f grafana
```
Watch Grafana's startup output. You should see `HTTP Server Listen, address=[::]:3000`.

```
$ docker exec -it grafana grafana-cli plugins ls
```
List installed plugins inside the container.

```
$ docker exec -it grafana grafana-cli plugins install grafana-piechart-panel
$ docker restart grafana
```
Install the pie chart panel plugin and restart so it loads.

```
$ docker exec -it grafana grafana-cli plugins update-all
$ docker restart grafana
```
Update every installed plugin to the latest version.

```
$ docker exec -it grafana grafana-cli admin reset-admin-password newpassword
```
Reset the admin password from the CLI without logging in. Useful when you forget it.

```
$ grafana-server --config=/etc/grafana/grafana.ini --homepath=/usr/share/grafana
```
Run Grafana as a binary directly (not in a container). Pass an explicit config file.

```
$ systemctl status grafana-server
$ systemctl start grafana-server
$ systemctl enable grafana-server
$ journalctl -u grafana-server -f
```
On a Debian/RHEL host where Grafana is installed as a deb/rpm, manage the systemd service and tail its logs.

```
$ kubectl port-forward svc/grafana 3000:80 -n monitoring
```
Forward a Grafana service running in Kubernetes to your localhost. Open `http://localhost:3000`.

```
$ kubectl logs -n monitoring grafana-0 | grep -i error
```
Search Grafana pod logs for errors.

```
$ kubectl rollout restart deployment grafana -n monitoring
```
Restart Grafana in Kubernetes to pick up new provisioning configmaps.

```
$ helm repo add grafana https://grafana.github.io/helm-charts
$ helm repo update
$ helm install grafana grafana/grafana -n monitoring --create-namespace
```
Install Grafana via the official Helm chart.

```
$ helm upgrade --reuse-values grafana grafana/grafana -n monitoring
```
Upgrade an existing Helm-managed Grafana to the latest chart version, keeping current values.

```
$ kubectl create configmap grafana-dashboards --from-file=./dashboards/ -n monitoring
```
Build a configmap from a folder of dashboard JSON files. The Grafana sidecar can pick those up via the `grafana_dashboard` label on the configmap.

```
$ curl -u admin:admin http://localhost:3000/api/health
```
Check Grafana's health endpoint. Returns `{"database":"ok","version":"..."}` when healthy.

```
$ curl -u admin:admin http://localhost:3000/api/datasources
```
List all configured datasources.

```
$ cat > datasource.json <<EOF
{
  "name": "Prometheus",
  "type": "prometheus",
  "url": "http://prometheus:9090",
  "access": "proxy",
  "isDefault": true
}
EOF
$ curl -u admin:admin -X POST -H "Content-Type: application/json" -d @datasource.json http://localhost:3000/api/datasources
```
Create a Prometheus datasource via the API.

```
$ curl -u admin:admin -X POST http://localhost:3000/api/dashboards/db -H "Content-Type: application/json" -d @dashboard.json
```
Upload a dashboard from a JSON file.

```
$ curl -u admin:admin http://localhost:3000/api/dashboards/uid/abc123def | jq . > dashboard.json
```
Export a dashboard's full JSON model. Now you can store it in Git.

```
$ curl -u admin:admin http://localhost:3000/api/folders
```
List all folders in the org.

```
$ curl -u admin:admin -X POST http://localhost:3000/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{"name":"alice","email":"alice@example.com","login":"alice","password":"hunter2"}'
```
Create a user via the admin API.

```
$ jsonnet -J vendor dashboards/myapp.jsonnet > dashboards/myapp.json
```
Generate a dashboard JSON file from Grafonnet. (Grafonnet is the Jsonnet library; you import it from `vendor/`.)

```
$ gr push dashboards/
$ gr pull dashboards/
$ gr diff dashboards/
```
Use Grizzly to push, pull, and diff dashboards as code against a live Grafana.

```
$ grafana-toolkit plugin:test
```
Run plugin tests for a custom plugin you're developing. (`grafana-toolkit` is legacy and being replaced by `@grafana/create-plugin`, but you'll still see it in older repos.)

```
$ openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
$ cp cert.pem /etc/grafana/grafana.crt
$ cp key.pem /etc/grafana/grafana.key
$ chown grafana:grafana /etc/grafana/grafana.{crt,key}
$ systemctl restart grafana-server
```
Renew or rotate Grafana's TLS cert. (Set `protocol = https` and the cert paths in `grafana.ini` first.)

```
$ curl https://grafana.com/api/dashboards/13639/revisions/latest/download \
  > cadvisor.json
$ curl -u admin:admin -X POST http://localhost:3000/api/dashboards/import \
  -H "Content-Type: application/json" \
  -d "{\"dashboard\":$(cat cadvisor.json),\"folderId\":0,\"overwrite\":true,\"inputs\":[{\"name\":\"DS_PROMETHEUS\",\"type\":\"datasource\",\"pluginId\":\"prometheus\",\"value\":\"Prometheus\"}]}"
```
Import a community dashboard (here, ID 13639 is the cAdvisor dashboard) directly via the API.

```
$ terraform init
$ cat > main.tf <<'EOF'
terraform {
  required_providers {
    grafana = { source = "grafana/grafana" }
  }
}
provider "grafana" {
  url  = "http://localhost:3000"
  auth = "admin:admin"
}
resource "grafana_folder" "prod" {
  title = "Production"
}
resource "grafana_dashboard" "homepage" {
  config_json = file("dashboards/homepage.json")
  folder      = grafana_folder.prod.id
}
EOF
$ terraform plan
$ terraform apply
```
Manage Grafana folders and dashboards declaratively with the `terraform-provider-grafana`.

```
# In Prometheus alertmanager.yml
$ cat <<EOF
receivers:
  - name: grafana
    webhook_configs:
      - url: http://grafana:3000/api/alertmanager/grafana/api/v1/alerts
        send_resolved: true
EOF
```
Relay Prometheus Alertmanager alerts into Grafana's internal alertmanager so they show up alongside Grafana-native alerts.

```
$ curl -u admin:admin "http://localhost:3000/api/v1/provisioning/alert-rules"
```
List provisioned alert rules via the API.

```
$ curl -u admin:admin -X POST -H "Content-Type: application/json" \
  http://localhost:3000/api/dashboards/db -d '{
    "dashboard": null,
    "folderId": 0,
    "overwrite": false
  }'
```
Sanity-test the dashboard import endpoint. (This will fail because `dashboard` is null — but it should fail with a useful error code, proving the API is alive and you're authenticated.)

```
$ curl -u admin:admin "http://localhost:3000/api/search?folderIds=0&query=&type=dash-db" | jq '.[] | .title'
```
List every dashboard's title in the org.

```
$ curl -u admin:admin "http://localhost:3000/api/ds/query" \
  -H "Content-Type: application/json" \
  -d '{
    "queries":[{
      "refId":"A",
      "datasource":{"uid":"prometheus"},
      "expr":"up"
    }],
    "from":"now-1h",
    "to":"now"
  }'
```
Run an ad-hoc query through Grafana's data-source query API. The same endpoint Explore uses.

```
$ docker run --rm -v $(pwd):/work grafana/grafonnet-builder \
  -j main.jsonnet > dashboard.json
```
Generate a dashboard from Grafonnet inside a container, no local toolchain.

```
$ grizzly apply -t Dashboard -f dashboards/
```
Apply only Dashboard-type resources from a folder via Grizzly. Filters out other resource types.

That is well over thirty hands-on commands. Even better, every one of them runs against the same Grafana you `docker run`'d in the first command. Try them.

## Common Confusions

These are the things that trip everybody up at first. Each is a question you will eventually ask out loud, with the answer.

### 1. Grafana doesn't store metrics

It is the dashboard. The data lives in Prometheus, Loki, Tempo, etc. If your Prometheus dies, Grafana shows blank panels. If Grafana dies, Prometheus keeps storing and you can rebuild Grafana from the JSON. Repeat to yourself: Grafana is the glass, the datasources are the water.

### 2. Unified vs legacy alerting

In Grafana 8, all alerting moved to a single engine called **unified alerting** built around Prometheus's Alertmanager. The old engine, where each panel had its own alert tab, is now called **legacy alerting** and was removed by default in Grafana 9 OSS. If you still see "alert" tabs on individual panels in your screenshots, you're looking at legacy and should migrate.

### 3. How dashboard variables actually re-template queries

Variables are pure string substitution. When the user picks `host=web2`, Grafana literally replaces `$host` with `web2` in every query and re-runs them. There is no fancy server-side templating. Knowing this means you can debug by inspecting the rendered query in the panel inspector ("Query inspector" -> "Query"). Whatever string Grafana sent is whatever the datasource saw.

### 4. Stat vs Gauge

Both render a single number. **Stat** is rectangular, optionally with a sparkline trend. **Gauge** is a semicircle dial with min/max ranges. Use Stat for things where the absolute number matters (throughput, count). Use Gauge for things where the percentage of a known range matters (CPU 0-100, disk 0-100). Neither is "better" — pick whichever reads faster.

### 5. What is the JSON model — and why edit dashboard YAML?

The dashboard itself is JSON. The provisioning **wrapper** that tells Grafana where to find the JSON is YAML. People conflate the two. You write your dashboard in JSON (or a tool that emits JSON like Grafonnet/Grizzly/Terraform). You write your provisioning manifest in YAML, and the YAML points at the JSON. They live in different folders for different reasons.

### 6. PromQL vs LogQL vs TraceQL

All three are query languages from Grafana Labs. PromQL is for Prometheus metrics: time series math. LogQL is for Loki logs: stream-of-strings, with optional metric extraction. TraceQL is for Tempo traces: filter spans by attributes. They share **selector syntax** (`{label="value"}`) on purpose so they feel like one family. But the underlying data shapes are completely different.

### 7. Tempo trace ID linking from logs

If your logs include a `trace_id=abc123` field and your Loki datasource is configured with a "Derived Fields" rule pointing at Tempo, then in any Logs panel that trace ID becomes a clickable link that opens Tempo and shows the trace. This is the "logs to traces in two clicks" magic. You configure it once in the Loki datasource UI.

### 8. Grafana Mimir replaces Cortex

Cortex was the old scalable Prometheus from Grafana Labs. Around 2022, Grafana Labs forked Cortex into **Mimir** and started developing Mimir as the new flagship. Cortex still exists (in maintenance) but Mimir is what new deployments should use. Same Prometheus API on the front, much better internals.

### 9. Alerting from Grafana vs Prometheus's own Alertmanager

You can fire alerts from Prometheus directly using its native Alertmanager. Or you can fire them from Grafana. Or both. Grafana's unified alerting is built on the same Alertmanager codebase, so it understands the same rules format. People typically pick Grafana when they want all alerts (across many datasources) in one place, and Prometheus-native when they want alerting to keep working even if Grafana is down.

### 10. Server Admin role vs Org Admin role

A Grafana server can host many **organizations** (orgs). Each org is isolated: its own dashboards, datasources, users. **Server Admin** is the god-mode for the whole server — manages orgs themselves, can log into any org. **Org Admin** is the highest role inside one org. Most teams have only one org and only one server admin.

### 11. What is a viewer/editor/admin in an Org?

These are the three default roles inside an organization.

- **Viewer** — read-only.
- **Editor** — can create/edit/delete dashboards.
- **Admin** — Editor plus can manage datasources, plugins, alert rules.

You assign roles per-user-per-org or per-team-per-folder.

### 12. Dashboard provisioning vs JSON-API

Provisioning means "Grafana reads YAML on startup and writes the dashboards into its own DB." JSON-API means "you POST dashboards over HTTP at runtime." Provisioned dashboards are read-only in the UI; API-pushed ones are editable. Pick provisioning for stable infrastructure dashboards. Pick API-push for dashboards you generate from CI on every deploy.

### 13. Refresh interval is per-dashboard, not per-panel

A common gotcha: every panel respects the dashboard-wide refresh setting at the top right. You can't make panel A refresh every 5s and panel B every 1m on the same dashboard out of the box. (You can with hacks, but the simple model is: one refresh interval rules them all.)

### 14. Anonymous access vs Public dashboards

Both let unauthenticated visitors see a dashboard. **Anonymous access** makes the entire Grafana server visible to anyone (with a configurable role); not granular. **Public dashboards** (Grafana 10+) is per-dashboard: you mark one specific dashboard as public, get a shareable URL, and only that dashboard is visible. Public dashboards are the safer pick.

### 15. Why panels say "no data" but Explore shows data

Usually because of variable interpolation. The dashboard variables substituted into the query produce something that returns nothing. Open the panel, click "Query inspector," look at the literal query that was sent. Paste it into Explore. If Explore shows data, your variable values are wrong. If Explore also shows nothing, your query is.

### 16. Datasources are scoped per-org

If you create a Prometheus datasource in org A, org B does not see it. People sometimes set up a datasource and wonder why it's missing in the other org. Each org has its own.

### 17. Time range vs query range

The time range at the top right of the dashboard sets the X axis for every panel. Most queries (PromQL `[5m]`, SQL `$__timeFilter`) respect that range automatically. But some don't: a hard-coded time literal in your SQL won't move when the picker moves. Always use the macros.

## Vocabulary

| Word | Plain English |
|------|---------------|
| Grafana | The app you're learning. The dashboard server. |
| Grafana Labs | The company that builds Grafana. |
| Grafana OSS | The open-source free version. AGPLv3. |
| Grafana Enterprise | The paid version with SSO, reporting, RBAC, support. |
| Grafana Cloud | The hosted SaaS version. |
| Datasource | A backend Grafana queries for data. |
| Panel | One visualisation box on a dashboard. |
| Row | A horizontal divider with panels under it. |
| Dashboard | A page of panels, with variables and a time range. |
| Folder | A container of dashboards. |
| Organization (org) | An isolated tenant inside a Grafana server. |
| User | A login. |
| Team | A named group of users you can grant permissions to. |
| Viewer | Lowest role. Read-only. |
| Editor | Middle role. Can edit dashboards. |
| Admin | Org-level highest role. Can manage datasources, alerts. |
| Server Admin | Server-level highest role. Manages orgs and global config. |
| Org Admin | Highest role inside one org. |
| Role-based access control (RBAC) | Permissions tied to roles you assign. |
| Fine-grained permissions | Enterprise feature: individual permissions per user. |
| Query | The string sent to a datasource to get data. |
| Expression | Math/logic applied to one or more queries' results. |
| Transformation | Per-panel data reshaping (rename, join, calculate). |
| Override | Per-series visual customisation on a panel. |
| Threshold | A numeric line on a panel; values past it are coloured. |
| Mappings | Replace numeric values with strings (0 -> "OK"). |
| Value mapping | Same as mappings; just verbose form. |
| Color scheme | How series are coloured. |
| Legend | The list of series at the bottom or side of a panel. |
| Tooltip | The hover-popup showing exact values. |
| Axes | X and Y axis settings on a panel. |
| Time-shift | Render the query as it was N minutes ago for comparison. |
| Time-window | The duration over which a query aggregates. |
| Time picker | The widget at the top of a dashboard that sets the range. |
| Refresh interval | How often the dashboard re-runs all queries. |
| Variables | Dashboard-level placeholders that templates queries. |
| Query variable | A variable populated by a datasource query. |
| Custom variable | A variable with a hand-typed list of values. |
| Interval variable | A variable holding time durations. |
| Datasource variable | A variable for picking which datasource to query. |
| Ad-hoc filter | A variable that becomes label-filter clauses everywhere. |
| Repeating panel | A panel that renders once per variable value. |
| Repeating row | A row that renders once per variable value. |
| Dashboard link | Top-of-dashboard button to another dashboard. |
| Panel link | In-panel clickable that jumps elsewhere. |
| Drilldown | Clicking from a high-level view to a lower-level one. |
| JSON model | The dashboard's full source JSON. |
| Provisioning | Configuring Grafana from YAML files at startup. |
| `datasources.yaml` | Provisioning file for datasources. |
| `dashboards.yaml` | Provisioning file for dashboard providers. |
| `alerting.yaml` | Provisioning file for alert rules and routes. |
| Unified alerting | The single alerting engine since Grafana 8. |
| Legacy alerting | The old per-panel alerting engine, removed in 9 OSS. |
| Alert rule | The "if X then fire" definition. |
| Contact point | Destination of a notification (Slack, email, etc.). |
| Notification policy | Tree that routes alerts to contact points. |
| Mute timing | Calendar window during which alerts don't notify. |
| Silence | One-off "shut up about this" rule with an expiry. |
| Recording rule | Pre-computed metric stored back into the datasource. |
| Alert state | Normal / Pending / Alerting / NoData / Error. |
| Alertmanager | The Prometheus-native router; built into Grafana too. |
| Notification channel | The legacy name for contact point. |
| Webhook | A URL Grafana POSTs to on alert; you write the consumer. |
| Slack | Chat platform; common contact point. |
| PagerDuty | On-call paging platform; common contact point. |
| OpsGenie | Atlassian's paging platform. |
| VictorOps | Splunk's paging platform (now SignalFx OnCall). |
| MS Teams | Microsoft chat; common contact point. |
| Email | SMTP; basic contact point. |
| Telegram | Messenger; supported as a contact point. |
| Discord | Chat; supported as a contact point. |
| Plugin | Extension you install into Grafana. |
| Panel plugin | New visualisation type. |
| Datasource plugin | New backend connector. |
| App plugin | Bundle of panels + datasources + dashboards. |
| Plugin signature | Crypto signature proving who built the plugin. |
| Signed plugin | Plugin with a valid Grafana Labs signature. |
| `allow_loading_unsigned_plugins` | Config to bypass signature checks. |
| Image renderer | Sidecar service that turns dashboards into PNG/PDF. |
| Reporting | Enterprise feature: scheduled PDF dashboards by email. |
| Grafana OnCall | On-call schedules and paging. |
| Grafana Pyroscope | Continuous profiling backend. |
| Profiling | Recording where a program spends its CPU and memory. |
| Continuous profiling | Profiling enabled in production all the time. |
| Frame graph | A visualisation of profiles. |
| Flame graph | The same thing, more common name. |
| Exemplar | A trace ID attached to a metric data point. |
| Trace | The full record of one request's path through services. |
| Span | One leg of a trace; a single service's portion of work. |
| OTel | OpenTelemetry, the standard. |
| OpenTelemetry | The vendor-neutral observability standard. |
| OpenTelemetry Collector | The OTel router/proxy. |
| OTLP | OpenTelemetry Protocol. |
| OTLP/HTTP | OTLP over HTTP+JSON or HTTP+protobuf. |
| OTLP/gRPC | OTLP over gRPC. |
| Loki | Grafana Labs' log database. |
| LogQL | Loki's query language. |
| Tempo | Grafana Labs' trace database. |
| TraceQL | Tempo's query language. |
| Prometheus | The standard open-source TSDB. |
| PromQL | Prometheus's query language. |
| Mimir | Grafana Labs' scalable Prometheus-compatible TSDB. |
| Grafana Mimir | Same as Mimir. |
| Cortex | The legacy scalable Prometheus, now Mimir. |
| VictoriaMetrics | An alternative TSDB Grafana can query. |
| InfluxDB | Time-series database from InfluxData. |
| Flux | InfluxDB's newer functional query language. |
| InfluxQL | InfluxDB's older SQL-like query language. |
| Elasticsearch | Search engine; common log backend. |
| OpenSearch | The Apache 2.0 fork of Elasticsearch. |
| KQL | Kibana Query Language (or Kusto in Azure context). |
| CloudWatch | AWS's monitoring service. |
| AWS X-Ray | AWS's tracing service. |
| GCP Operations | Google's monitoring suite (was Stackdriver). |
| Azure Monitor | Microsoft Azure's monitoring service. |
| Application Insights | Azure's APM service. |
| BigQuery | Google's data warehouse; can be a Grafana datasource. |
| Snowflake | Cloud data warehouse; Grafana datasource via plugin. |
| Postgres | Open-source SQL database. |
| MySQL | Open-source SQL database. |
| Microsoft SQL Server | Microsoft's SQL database. |
| MariaDB | MySQL fork. |
| ClickHouse | Open-source analytics SQL DB; great for logs. |
| Jaeger | Open-source tracing backend. |
| Zipkin | Older open-source tracing backend. |
| OpenSearch trace | OpenSearch's trace store. |
| Grizzly (gr) | CLI for dashboards-as-code. |
| Grafonnet | Jsonnet library for dashboards-as-code. |
| Tanka | Grafana Labs' Jsonnet config tool. |
| Kustomize | Kubernetes YAML overlay tool, used for Grafana manifests. |
| Helm chart | Templated Kubernetes manifests; the Grafana chart. |
| `terraform-provider-grafana` | Terraform provider for Grafana resources. |
| Dashboard-as-code | Storing dashboards in Git, generating JSON. |
| Alert-as-code | Same idea, for alert rules. |
| Kiosk mode | Fullscreen mode hiding the Grafana chrome. |
| `?kiosk` | URL query param that activates kiosk mode. |
| Embed iframe | `<iframe>` of a Grafana panel for another web page. |
| Anonymous access | Letting unauthenticated visitors see Grafana. |
| Public dashboards | Per-dashboard public URL (Grafana 10+). |
| Dashboard snapshots | A frozen copy of a dashboard, shareable. |
| `/api/dashboards` | API endpoint for dashboard CRUD. |
| `/api/datasources` | API endpoint for datasource CRUD. |
| `/api/folders` | API endpoint for folder CRUD. |
| `/api/teams` | API endpoint for teams. |
| `/api/users` | API endpoint for users (org-scoped). |
| `/api/admin` | Admin-only API endpoints. |
| `/api/health` | Liveness probe endpoint. |
| `/api/ds/query` | Generic datasource query endpoint. |
| `/api/alertmanager/grafana` | The Grafana-internal Alertmanager API. |
| `/api/v1/provisioning/alert-rules` | API for alert rules as code. |
| `ldap.toml` | LDAP config file. |
| `generic_oauth` | The generic OAuth section of `grafana.ini`. |
| `oauth2-proxy` | An external auth proxy you can sit in front of Grafana. |
| SAML SSO | Enterprise SSO via SAML protocol. |
| Team Sync | Enterprise feature: sync IdP groups to Grafana teams. |

That's well over 120 entries. Bookmark this section.

## Try This

Concrete things to do, in order, to make Grafana stick. Each one takes ten minutes or less.

### 1. Run Grafana locally

```
$ docker run -d -p 3000:3000 --name=grafana grafana/grafana
```

Open `http://localhost:3000`. Log in with admin/admin. Set a new password.

### 2. Add the play.grafana.org Prometheus

Go to Connections > Data sources > Add data source > Prometheus. Set URL to `https://prometheus.demo.do.prometheus.io`. Save & test. Now you have a working datasource without running Prometheus yourself.

### 3. Build a panel from scratch

Click Dashboards > New > New dashboard > Add visualization. Pick the Prometheus datasource. Type query: `up`. You should see a binary "up/down" graph for the demo Prometheus's targets.

### 4. Add a variable

Click the dashboard settings gear > Variables > New variable. Name: `instance`. Type: Query. Query: `label_values(up, instance)`. Save and refresh. Now there's a dropdown at the top with each instance.

Reference it in your panel query: `up{instance="$instance"}`. Now changing the dropdown filters the panel.

### 5. Import a community dashboard

Dashboards > New > Import. Paste ID `1860` (the famous Node Exporter Full dashboard). Pick your Prometheus datasource. Done. You have an enterprise-grade dashboard in 30 seconds.

### 6. Set up an alert

Open the panel from step 3. Click the panel title > Edit > Alert tab. Click "Create alert rule from this panel." Set condition to `IS BELOW 1`. Save. Now if any target goes down, the rule will fire.

Configure a contact point: Alerting > Contact points > New > Type: Webhook > URL: `https://webhook.site/<your-id>` (grab one from webhook.site). Save.

Edit the default notification policy to point at your webhook contact point.

When a target goes down, you should see a webhook hit at webhook.site.

### 7. Provision a datasource

Stop the container. Recreate it with a provisioning mount:

```
$ mkdir -p ./grafana/provisioning/datasources
$ cat > ./grafana/provisioning/datasources/prometheus.yaml <<EOF
apiVersion: 1
datasources:
  - name: Prometheus-provisioned
    type: prometheus
    access: proxy
    url: https://prometheus.demo.do.prometheus.io
    isDefault: false
EOF
$ docker run -d -p 3000:3000 \
  -v $(pwd)/grafana/provisioning:/etc/grafana/provisioning \
  --name=grafana grafana/grafana
```

Log in. Look at Data sources. You should see "Prometheus-provisioned" with a banner saying it's provisioned (read-only).

### 8. Export a dashboard as JSON

Click the dashboard share button > Export > Save to file. Open the file. Notice the structure: `panels`, `templating`, `time`, `refresh`. Edit the title field. Re-import. See the change.

### 9. Use Explore split view

Click Explore in the left nav. Click "Split" at the top right. Now you have two query panes side by side. In one, query Prometheus. In the other, switch to Loki (or another metric). Notice the synchronized time range.

### 10. Read the logs of your Grafana

```
$ docker logs grafana | tail -50
```

See the startup messages, the auth events, the datasource connections. This is invaluable when something goes wrong.

## Where to Go Next

Once Grafana clicks, the next stops are:

- **PromQL deep dive** — every dashboard ends up using rate, sum, by, histogram_quantile. Learn them.
- **Alerting fluency** — rule structure, notification policies, mute timings. The most common production-time work.
- **Dashboards as code** — pick one of Grafonnet, Grizzly, or terraform-provider-grafana and standardise on it. Once your dashboards are in Git, you can review them like any other code.
- **OpenTelemetry** — if you don't yet have traces, OTel is the single best investment. Then plug Tempo in.
- **Loki for logs** — if you're still on Elasticsearch and bleeding money, Loki is dramatically cheaper.
- **Mimir for scale** — when one Prometheus is no longer enough, Mimir is the Grafana-native scale-out.
- **OnCall** — the layer above alerting. Handles the human side: schedules, escalations, post-mortems.

## See Also

- [monitoring/grafana](../monitoring/grafana.md) — the technical cheatsheet, dense.
- [monitoring/grafana-tempo](../monitoring/grafana-tempo.md) — Tempo specifics.
- [monitoring/prometheus](../monitoring/prometheus.md) — the most common Grafana datasource.
- [monitoring/opentelemetry](../monitoring/opentelemetry.md) — instrumentation that feeds Grafana.
- [monitoring/snmp](../monitoring/snmp.md) — for network-device metrics.
- [monitoring/sflow](../monitoring/sflow.md) — for network flow data.
- [monitoring/netflow-ipfix](../monitoring/netflow-ipfix.md) — alternative network flow data.
- [monitoring/model-driven-telemetry](../monitoring/model-driven-telemetry.md) — modern device telemetry.
- [monitoring/ip-sla](../monitoring/ip-sla.md) — synthetic measurements as datasource.
- [ramp-up/prometheus-eli5](prometheus-eli5.md) — the Grafana-without-Prometheus is rare.
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md) — the foundation everything sits on.
- [ramp-up/kubernetes-eli5](kubernetes-eli5.md) — the most common deployment target.
- [ramp-up/docker-eli5](docker-eli5.md) — the most common quick-start.

## References

- `grafana.com/docs` — the official documentation. The canonical source.
- *The Definitive Guide to Grafana* by Sven Brandt — book-length walk through every feature.
- `grafana.com/blog` — Grafana Labs's engineering blog.
- `grafana.com/grafana/dashboards` — the community catalog of 6000+ pre-built dashboards.
- `play.grafana.org` — a public sandbox where you can click around without setup.
- KubeCon talks — annual presentations by Grafana Labs covering new features and large-scale practices.
- `github.com/grafana/grafana` — the source code. Surprisingly readable for a Go + TypeScript app of its size.
- `github.com/grafana/grafonnet` — the Jsonnet library.
- `github.com/grafana/grizzly` — the dashboards-as-code CLI.
- `github.com/grafana/terraform-provider-grafana` — the Terraform provider.

## Version Notes

- **Grafana 8** (June 2021) — Unified alerting reaches GA. Major UI refresh.
- **Grafana 9** (June 2022) — Explore-to-Dashboard. Public-preview of public dashboards. Legacy alerting deprecated.
- **Grafana 10** (June 2023) — Navigation sidebar redesign. Nested folders behind a flag. Public dashboards GA. Scenes-based dashboards in preview.
- **Grafana 11** (May 2024) — Schema v2. Scenes-based dashboards GA. Subfolders GA. Bigger improvements to alerting performance.
- **Grafana 12** (mid-2025 onward) — More schema-v2-only dashboards; lots of work on dashboard performance and hibernation; deeper Tempo integration.
- **Grafana 13** (early-2026 onward) — The most recent stable line; default Prometheus datasource picks up exemplar UI improvements; native support for OTel traces from Loki via derived fields is now bundled.

If you are reading something written before mid-2021, mentally tag it as legacy alerting. If you are reading something before Grafana 9, mentally tag it as legacy navigation. The fundamentals — dashboards, panels, datasources, queries — have been stable since the beginning. Only the UI and the alert engine have shifted significantly.

## A Day in the Life of a Grafana Query

To make all this concrete, let's walk through what happens when a single user opens a dashboard. We will follow one panel from click to picture.

### Step 1: User opens the URL

The user types `https://grafana.example.com/d/abc123/production-health` and hits enter. The browser sends a request to the Grafana server. Grafana's frontend (a JavaScript Single Page App) loads.

### Step 2: Grafana loads the dashboard

The browser asks Grafana's API: "give me the JSON for dashboard `abc123`." Grafana looks up the dashboard in its metadata DB (the SQLite or Postgres that holds Grafana's own state). It returns the JSON model.

### Step 3: Frontend parses the JSON

The Grafana SPA reads the JSON. It sees a list of panels, a list of variables, a time range, a refresh setting. It renders the layout: each panel gets a placeholder rectangle.

### Step 4: Variables resolve

For each variable of type "query", the frontend sends the variable's query to the chosen datasource via `/api/ds/query`. The answer becomes the dropdown's options. The default selection is computed.

### Step 5: For each panel, send the query

For each panel, the frontend takes the panel's query (with variables substituted), the current time range, and any refresh-time-based macros. It POSTs to `/api/ds/query`. The Grafana backend receives the request, looks up the datasource, makes the actual outbound HTTP call to Prometheus or Loki or whatever.

```
Browser ----HTTP POST /api/ds/query--->  Grafana backend
                                              |
                                              v
                                         Datasource 
                                         (Prometheus etc.)
                                              |
                                              v
                                         Grafana backend
                                              |
Browser <----HTTP 200 (DataFrame)-------- 
```

### Step 6: Frontend draws

The browser receives a `DataFrame` (Grafana's internal data shape: columns for time, value, labels, etc.). The panel plugin reads the DataFrame and renders. Time series becomes lines. Stat becomes a number. Heatmap becomes a grid.

### Step 7: Repeat

After the configured refresh interval (default 5s, often set to 30s or 1m or off), the frontend re-runs every panel's query. The whole cycle repeats. Forever, as long as the tab is open.

### Step 8: User clicks something

The user hovers a line. Grafana's frontend calls the panel plugin's tooltip handler with the X coordinate. The plugin returns the value at that X. The browser draws the tooltip.

The user changes a variable value. Step 5 onward repeats with the new substitution.

The user changes the time range. Step 5 onward repeats with the new range.

This is the whole dance. There is nothing magical underneath. It is HTTP requests and JavaScript rendering and JSON. That is genuinely all of it.

## A Day in the Life of an Alert

Let's walk through one alert fire-to-page.

### Step 1: Alert rule is evaluated

Every minute (or whatever the evaluation interval is), Grafana's internal **scheduler** wakes up the alert rule. It runs the query against the datasource. It applies any expression math. It checks the condition.

### Step 2: Condition becomes true

The query returns a value of `1.5`. The condition says "fire if > 1.0." The rule moves from `Normal` to `Pending`.

### Step 3: "For" duration

The rule has `for: 5m` set. So Grafana waits five minutes, re-evaluating each interval. If the condition stays true the whole time, the rule moves to `Alerting`. If it becomes false, the rule drops back to `Normal` without firing.

### Step 4: Internal Alertmanager

Grafana sends the firing alert to its own internal Alertmanager. The Alertmanager receives `{ labels: {team: "payments", severity: "critical"}, annotations: {...} }`.

### Step 5: Notification policy routes

The Alertmanager applies the notification policy tree. The first matcher that fits wins. `team=payments` matches the "payments" branch. The branch points at the `payments-pager` contact point.

### Step 6: Mute timings

If the current time falls inside any mute timing attached to the matched policy, the notification is suppressed. (The alert is still firing internally; just no page goes out.) Otherwise we continue.

### Step 7: Group, throttle, dedupe

The Alertmanager groups alerts with the same label set, throttles repeats within a `group_interval`, and deduplicates identical pages. Standard Alertmanager behavior.

### Step 8: Contact point delivers

The contact point ("payments-pager", a PagerDuty integration) sends an HTTPS POST to PagerDuty's API with the routing key. PagerDuty acknowledges. PagerDuty's own escalation logic takes over from there.

### Step 9: Resolution

The condition becomes false. Grafana sends a "resolved" event through the same path. PagerDuty marks the incident resolved. Slack or email gets a "RESOLVED" message.

That is the whole life cycle. It is not magic; it is just a scheduled query and a routing tree.

## Architecture Recap (one big picture)

```
+-------------------------------------------------------------------+
|                          BROWSER                                   |
|     React SPA: dashboards, panels, explore, alerting UI            |
+--------+----------------------------------------------------+------+
         |                                                    |
         | /api/ds/query                                      | /api/dashboards/db
         | /api/datasources                                   | /api/folders
         |                                                    |
         v                                                    v
+-------------------------------------------------------------------+
|                       GRAFANA SERVER (Go)                          |
|                                                                    |
|  +----------------+  +----------------+  +-------------------+     |
|  | Auth           |  | Datasource     |  | Dashboard         |     |
|  | (basic, OAuth, |  | proxy          |  | service           |     |
|  |  SAML, LDAP)   |  +-------+--------+  +---------+---------+     |
|  +----------------+          |                     |               |
|                              |                     |               |
|  +----------------+          |                     |               |
|  | Alerting       |          |                     |               |
|  | scheduler +    |          |                     |               |
|  | Alertmanager   |          |                     |               |
|  +----------------+          |                     |               |
|                              |                     |               |
+------------------------------+---------------------+---------------+
                               |                     |
                               v                     v
                  +----------------------+   +-----------------+
                  | DATASOURCES          |   | METADATA DB     |
                  |                      |   | (SQLite,        |
                  | Prometheus, Loki,    |   |  Postgres,      |
                  | Tempo, Mimir, SQL,   |   |  MySQL)         |
                  | Elasticsearch, ES,   |   |                 |
                  | InfluxDB, Cloud, etc.|   | Users, orgs,    |
                  +----------------------+   | dashboards,     |
                                              | alert rules    |
                                              +-----------------+
```

Three boxes: browser, server, datasources (and metadata DB). That is the whole architecture. Every feature lives in one of these boxes. Every API call goes between them. If you can hold that diagram in your head, you can debug almost any Grafana problem by asking "which box is misbehaving?"

## Final Words

If you remember three things from this sheet, remember:

1. **Grafana is glass, not water.** Data lives in the datasources. Grafana asks, draws, repeats.
2. **Variables are string substitution.** When you understand that, every "why is my panel empty" has a clear debug path: the rendered query in the inspector tells you exactly what was sent.
3. **Provision everything you can.** Dashboards in Git, datasources in YAML, alert rules in YAML. The UI is for poking and prototyping. The source of truth is files.

That is enough to be productive. Everything else in this sheet is depth on top of those three ideas.

Welcome to the dashboard wall. The graphs are waiting to draw themselves.
