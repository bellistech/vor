# Fluent Bit (Log Processor)

Lightweight, high-performance log processor and forwarder with a plugin-based pipeline architecture for collecting, parsing, filtering, and routing logs from containers, files, and system sources.

## Pipeline Architecture

```bash
# Fluent Bit pipeline:
# INPUT → PARSER → FILTER → BUFFER → OUTPUT
#
# Input:  Collect logs (tail, systemd, tcp, http, forward)
# Parser: Structure raw text into fields (regex, json, logfmt)
# Filter: Modify, enrich, or drop records (grep, modify, lua, kubernetes)
# Buffer: Memory or filesystem buffering for backpressure
# Output: Send to destinations (elasticsearch, loki, s3, stdout)
```

## Configuration (fluent-bit.conf)

```ini
[SERVICE]
    Flush         5
    Daemon        Off
    Log_Level     info
    Parsers_File  parsers.conf
    HTTP_Server   On
    HTTP_Listen   0.0.0.0
    HTTP_Port     2020
    storage.path  /var/log/flb-storage/
    storage.sync  normal
    storage.checksum  off
    storage.max_chunks_up  128

[INPUT]
    Name          tail
    Path          /var/log/app/*.log
    Tag           app.*
    Parser        json
    DB            /var/log/flb_app.db
    Mem_Buf_Limit 10MB
    Skip_Long_Lines On
    Refresh_Interval 10
    Read_from_Head   On

[INPUT]
    Name          systemd
    Tag           systemd.*
    Systemd_Filter _SYSTEMD_UNIT=nginx.service
    Systemd_Filter _SYSTEMD_UNIT=sshd.service
    Read_From_Tail On

[INPUT]
    Name          tcp
    Listen        0.0.0.0
    Port          5170
    Tag           tcp.input
    Format        json

[FILTER]
    Name          grep
    Match         app.*
    Regex         log level=(error|warn)

[FILTER]
    Name          modify
    Match         *
    Add           hostname ${HOSTNAME}
    Add           environment production
    Rename        message log
    Remove        _p

[FILTER]
    Name          record_modifier
    Match         *
    Record        cluster us-east-1
    Remove_key    secret_field

[FILTER]
    Name          parser
    Match         app.nginx*
    Key_Name      log
    Parser        nginx
    Reserve_Data  On
    Preserve_Key  On

[OUTPUT]
    Name          es
    Match         app.*
    Host          elasticsearch
    Port          9200
    Index         app-logs
    Type          _doc
    Logstash_Format On
    Logstash_Prefix app
    Retry_Limit   5
    tls           On
    tls.verify    Off
    Suppress_Type_Name On

[OUTPUT]
    Name          loki
    Match         systemd.*
    Host          loki
    Port          3100
    Labels        job=systemd, source=fluentbit
    Auto_Kubernetes_Labels On

[OUTPUT]
    Name          stdout
    Match         *
    Format        json_lines
```

## Parsers (parsers.conf)

```ini
[PARSER]
    Name        json
    Format      json
    Time_Key    time
    Time_Format %Y-%m-%dT%H:%M:%S.%L%z

[PARSER]
    Name        nginx
    Format      regex
    Regex       ^(?<remote>[^ ]*) - (?<user>[^ ]*) \[(?<time>[^\]]*)\] "(?<method>\S+)(?: +(?<path>[^ ]*) +\S*)?" (?<code>[^ ]*) (?<size>[^ ]*)
    Time_Key    time
    Time_Format %d/%b/%Y:%H:%M:%S %z

[PARSER]
    Name        syslog-rfc3164
    Format      regex
    Regex       ^\<(?<pri>[0-9]+)\>(?<time>[^ ]* {1,2}[^ ]* [^ ]*) (?<host>[^ ]*) (?<ident>[a-zA-Z0-9_\/\.\-]*)(?:\[(?<pid>[0-9]+)\])?(?:[^\:]*\:)? *(?<message>.*)$
    Time_Key    time
    Time_Format %b %d %H:%M:%S

[PARSER]
    Name        logfmt
    Format      logfmt

[PARSER]
    Name        docker
    Format      json
    Time_Key    time
    Time_Format %Y-%m-%dT%H:%M:%S.%L%z

[PARSER]
    Name        go_multiline
    Format      regex
    Regex       ^(?<time>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) (?<message>.*)$
    Time_Key    time
    Time_Format %Y/%m/%d %H:%M:%S
```

## Multiline Parsing

```ini
[MULTILINE_PARSER]
    Name          java_stacktrace
    Type          regex
    Flush_Timeout 1000
    Rule          "start_state"  "/^(\d{4}-\d{2}-\d{2})/"  "cont"
    Rule          "cont"         "/^\s+(at|Caused by|\.{3})/"  "cont"

[MULTILINE_PARSER]
    Name          go_panic
    Type          regex
    Flush_Timeout 1000
    Rule          "start_state"  "/^(goroutine |panic:|runtime error:)/"  "cont"
    Rule          "cont"         "/^\s+/"  "cont"

[INPUT]
    Name              tail
    Path              /var/log/app/java.log
    Tag               java.*
    multiline.parser  java_stacktrace
    Read_from_Head    On
```

## Kubernetes Filter

```ini
[INPUT]
    Name              tail
    Path              /var/log/containers/*.log
    Tag               kube.*
    Parser            docker
    DB                /var/log/flb_kube.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On

[FILTER]
    Name              kubernetes
    Match             kube.*
    Kube_URL          https://kubernetes.default.svc:443
    Kube_CA_File      /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    Kube_Token_File   /var/run/secrets/kubernetes.io/serviceaccount/token
    Merge_Log         On
    Merge_Log_Key     log_processed
    K8S-Logging.Parser On
    K8S-Logging.Exclude On
    Labels            On
    Annotations       Off
    Buffer_Size       0
```

## Lua Filter

```ini
[FILTER]
    Name          lua
    Match         *
    script        /etc/fluent-bit/scripts/filter.lua
    call          process

# filter.lua:
# function process(tag, timestamp, record)
#     if record["status"] ~= nil then
#         record["status"] = tonumber(record["status"])
#     end
#     if record["password"] ~= nil then
#         record["password"] = "***REDACTED***"
#     end
#     return 1, timestamp, record    -- 1=keep, 0=drop, -1=keep+modify timestamp
# end
```

## Buffering & Backpressure

```ini
# Filesystem buffering (survives restarts)
[SERVICE]
    storage.path           /var/log/flb-storage/
    storage.sync           normal
    storage.max_chunks_up  128      # max chunks in memory

[INPUT]
    Name            tail
    Path            /var/log/*.log
    Tag             logs.*
    storage.type    filesystem       # enable disk buffering for this input
    Mem_Buf_Limit   50MB             # memory limit before backpressure

# When Mem_Buf_Limit is reached:
# - Input pauses (backpressure)
# - Data is stored to filesystem
# - Resumes when output catches up
```

## Health & Metrics

```bash
# Built-in HTTP monitoring
curl http://localhost:2020/api/v1/health
curl http://localhost:2020/api/v1/metrics
curl http://localhost:2020/api/v1/metrics/prometheus
curl http://localhost:2020/api/v1/storage

# Hot reload (SIGHUP)
kill -HUP $(pidof fluent-bit)

# Validate config
fluent-bit -c fluent-bit.conf --dry-run
```

## Tips

- Use filesystem buffering (`storage.type filesystem`) for production to survive restarts and output outages
- Set `Mem_Buf_Limit` on every tail input to prevent unbounded memory growth during output failures
- Use the `DB` option with tail inputs to track file read positions across restarts
- Enable `Skip_Long_Lines` to prevent a single huge log line from blocking the entire pipeline
- Use the Kubernetes filter's `Merge_Log` option to parse JSON inside container log wrappers
- Use `grep` filter to drop noisy logs early in the pipeline before they hit the output
- Use Lua filters for complex transformations like PII redaction or field type conversion
- Set `Retry_Limit` on outputs rather than using infinite retries, which can hide persistent failures
- Monitor `/api/v1/metrics/prometheus` with Prometheus to track pipeline health and throughput
- Use multiline parsers for Java stack traces and Go panics to keep related log lines together
- Prefer `logfmt` parser when your apps log in key=value format for zero-regex parsing
- Set `Refresh_Interval` to control how often Fluent Bit checks for new files matching the tail path glob

## See Also

rsyslog, logrotate, opentelemetry, elasticsearch, grafana

## References

- [Fluent Bit Documentation](https://docs.fluentbit.io/)
- [Fluent Bit Input Plugins](https://docs.fluentbit.io/manual/pipeline/inputs)
- [Fluent Bit Filter Plugins](https://docs.fluentbit.io/manual/pipeline/filters)
- [Fluent Bit Output Plugins](https://docs.fluentbit.io/manual/pipeline/outputs)
- [Fluent Bit Kubernetes Deployment](https://docs.fluentbit.io/manual/installation/kubernetes)
