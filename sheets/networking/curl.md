# curl (URL Transfer)

Command-line tool for transferring data with URLs — supports HTTP, HTTPS, FTP, SCP, SFTP, and dozens more protocols.

## GET Requests

### Basic GET
```bash
curl https://api.example.com/users
curl -s https://api.example.com/users           # silent (no progress bar)
curl -sS https://api.example.com/users          # silent but show errors
curl -o output.json https://api.example.com/users   # save to file
curl -O https://example.com/file.tar.gz         # save with remote filename
```

### Follow redirects
```bash
curl -L https://example.com                     # follow 3xx redirects
curl -L --max-redirs 5 https://example.com      # limit redirect depth
```

## POST Requests

### JSON POST
```bash
curl -X POST https://api.example.com/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice","email":"alice@example.com"}'
```

### Form data
```bash
curl -X POST https://example.com/login \
  -d 'username=alice&password=secret'

# URL-encode data automatically
curl --data-urlencode 'query=hello world' https://example.com/search
```

### Multipart / file upload
```bash
curl -X POST https://example.com/upload \
  -F 'file=@document.pdf' \
  -F 'description=Annual report'

curl -F 'images=@photo1.jpg' -F 'images=@photo2.jpg' https://example.com/gallery
```

### Read body from file
```bash
curl -X POST https://api.example.com/data \
  -H 'Content-Type: application/json' \
  -d @payload.json
```

## Other Methods

### PUT, PATCH, DELETE
```bash
curl -X PUT https://api.example.com/users/1 \
  -H 'Content-Type: application/json' \
  -d '{"name":"bob"}'

curl -X PATCH https://api.example.com/users/1 \
  -H 'Content-Type: application/json' \
  -d '{"email":"bob@new.com"}'

curl -X DELETE https://api.example.com/users/1
```

### HEAD (headers only)
```bash
curl -I https://example.com
curl --head https://example.com
```

## Headers

### Custom headers
```bash
curl -H 'Accept: application/json' https://api.example.com/data
curl -H 'X-API-Key: abc123' https://api.example.com/data
curl -H 'Accept: application/json' -H 'X-Request-ID: req-42' https://api.example.com
```

### Show response headers
```bash
curl -i https://example.com              # headers + body
curl -I https://example.com              # headers only
curl -v https://example.com              # full request/response trace
curl -D headers.txt https://example.com  # save headers to file
```

## Authentication

### Basic auth
```bash
curl -u alice:secret https://api.example.com/private
curl -u alice https://api.example.com/private    # prompt for password
```

### Bearer token
```bash
curl -H 'Authorization: Bearer eyJhbGc...' https://api.example.com/me
```

### Digest auth
```bash
curl --digest -u alice:secret https://api.example.com/private
```

## TLS / Certificates

### Certificate options
```bash
curl -k https://self-signed.example.com         # skip TLS verification (insecure)
curl --cacert /path/to/ca.pem https://example.com
curl --cert client.pem --key client-key.pem https://mtls.example.com
curl --tlsv1.2 https://example.com              # minimum TLS 1.2
```

## Cookies

### Send and receive cookies
```bash
curl -b 'session=abc123' https://example.com
curl -b cookies.txt https://example.com          # read from file
curl -c cookies.txt https://example.com/login    # save cookies to file
curl -b cookies.txt -c cookies.txt https://example.com  # read + update
```

## Proxy

### HTTP/SOCKS proxy
```bash
curl -x http://proxy.example.com:8080 https://example.com
curl -x socks5://127.0.0.1:1080 https://example.com
curl --proxy-user user:pass -x http://proxy:8080 https://example.com
curl --noproxy 'localhost,*.internal' https://internal.example.com
```

## Output and Timing

### Write-out variables
```bash
curl -s -o /dev/null -w '%{http_code}\n' https://example.com
curl -s -o /dev/null -w '%{time_total}\n' https://example.com

# Detailed timing
curl -s -o /dev/null -w 'dns: %{time_namelookup}s\nconnect: %{time_connect}s\ntls: %{time_appconnect}s\nttfb: %{time_starttransfer}s\ntotal: %{time_total}s\n' https://example.com
```

### Download with progress
```bash
curl -# -O https://example.com/large-file.iso   # progress bar
curl --limit-rate 1M -O https://example.com/file.iso  # bandwidth limit
curl -C - -O https://example.com/file.iso        # resume interrupted download
```

## Retry and Timeout

### Timeouts
```bash
curl --connect-timeout 5 https://example.com     # connection timeout
curl -m 30 https://example.com                   # max total time
curl --max-time 30 https://example.com           # same as -m
```

### Retry
```bash
curl --retry 3 https://example.com
curl --retry 3 --retry-delay 2 https://example.com
curl --retry 3 --retry-all-errors https://example.com  # retry on any error
```

## Tips

- `-s` (silent) + `-S` (show errors) is the standard combo for scripts
- `-f` (fail) makes curl return exit code 22 on HTTP errors (4xx/5xx) — use in CI
- `-w '%{http_code}'` is the reliable way to check status codes in scripts
- `--compressed` tells the server you accept gzip/brotli and auto-decompresses
- Use `@-` to read from stdin: `echo '{}' | curl -d @- https://api.example.com`
- `curl -v` dumps TLS handshake details — invaluable for cert debugging
- `-o /dev/null` discards the body when you only care about headers or timing
- Environment variable `CURL_CA_BUNDLE` sets the default CA cert path
- On macOS, system curl supports `--apple-ssl` backend; Homebrew curl uses OpenSSL

## References

- [curl Official Documentation](https://curl.se/docs/)
- [curl Man Page](https://man7.org/linux/man-pages/man1/curl.1.html)
- [curl — The curl Guide to HTTP](https://everything.curl.dev/)
- [curl GitHub Repository](https://github.com/curl/curl)
- [curl — Command Line Options Reference](https://curl.se/docs/manpage.html)
- [curl — HTTP Scripting Tutorial](https://curl.se/docs/httpscripting.html)
- [curl — SSL/TLS Certificate Handling](https://curl.se/docs/sslcerts.html)
- [curl — libcurl API Documentation](https://curl.se/libcurl/)
