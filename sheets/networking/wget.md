# wget (Web Downloader)

Non-interactive network downloader — excels at recursive downloads, mirroring, and resumable transfers.

## Basic Downloads

### Download files
```bash
wget https://example.com/file.tar.gz                     # save with original name
wget -O output.tar.gz https://example.com/file.tar.gz    # save with custom name
wget -O - https://example.com/data.json                  # output to stdout
wget -q https://example.com/file.tar.gz                  # quiet mode
```

### Multiple files
```bash
wget https://example.com/file1.tar.gz https://example.com/file2.tar.gz
wget -i urls.txt                                          # download URLs from file
```

## Resume and Retry

### Continue interrupted downloads
```bash
wget -c https://example.com/large-file.iso                # resume partial download
```

### Retry on failure
```bash
wget --tries=5 https://example.com/file.tar.gz            # retry 5 times
wget --retry-connrefused https://example.com/file.tar.gz  # retry on connection refused
wget --waitretry=10 https://example.com/file.tar.gz       # wait between retries
wget -t 0 https://example.com/file.tar.gz                 # retry forever
```

## Recursive Downloads

### Download a directory listing
```bash
wget -r https://example.com/pub/                          # recursive
wget -r -l 2 https://example.com/docs/                    # depth limit of 2
wget -r -np https://example.com/docs/                     # no parent (stay in dir)
wget -r -np -nH https://example.com/docs/                 # no host directory prefix
wget -r -np -nH --cut-dirs=1 https://example.com/pub/data/  # strip path segments
```

### Filter by file type
```bash
wget -r -A '*.pdf' https://example.com/reports/           # accept only PDFs
wget -r -R '*.jpg,*.png' https://example.com/docs/        # reject images
wget -r --accept-regex='2024.*\.csv' https://example.com/data/
```

## Mirroring

### Mirror a website
```bash
wget --mirror https://example.com
# Equivalent to: wget -r -N -l inf --no-remove-listing

wget --mirror --convert-links --page-requisites \
  --no-parent https://example.com/docs/
# -k (--convert-links): adjust links for local viewing
# -p (--page-requisites): get CSS, JS, images
```

### Offline website copy
```bash
wget --mirror -k -p -E https://example.com
# -E: save .html extension for text/html content-type
```

## Authentication

### HTTP auth
```bash
wget --user=alice --password=secret https://example.com/private/
wget --ask-password --user=alice https://example.com/private/
```

### FTP
```bash
wget --ftp-user=alice --ftp-password=secret ftp://ftp.example.com/file.tar.gz
```

### Cookies
```bash
wget --load-cookies cookies.txt https://example.com/protected/
wget --save-cookies cookies.txt --keep-session-cookies https://example.com/login
```

## Proxy

### HTTP proxy
```bash
wget -e http_proxy=http://proxy:8080 https://example.com/file.tar.gz
wget --no-proxy https://example.com/file.tar.gz   # bypass proxy
```

### Environment variables
```bash
export http_proxy=http://proxy:8080
export https_proxy=http://proxy:8080
export no_proxy=localhost,*.internal
wget https://example.com/file.tar.gz
```

## Bandwidth and Rate Control

### Limit download speed
```bash
wget --limit-rate=500k https://example.com/file.iso      # 500 KB/s
wget --limit-rate=2m https://example.com/file.iso         # 2 MB/s
```

### Wait between requests
```bash
wget -r --wait=2 https://example.com/                     # 2 seconds between requests
wget -r --random-wait https://example.com/                # random 0.5x-1.5x of --wait
```

## Output Control

### Directory and naming
```bash
wget -P /tmp/downloads/ https://example.com/file.tar.gz   # save to directory
wget -N https://example.com/file.tar.gz                   # timestamping (skip if not newer)
wget --content-disposition https://example.com/download    # use server-suggested filename
```

### Logging
```bash
wget -o wget.log https://example.com/file.tar.gz          # log to file
wget -a wget.log https://example.com/file.tar.gz          # append to log
wget -nv https://example.com/file.tar.gz                  # less verbose, still shows progress
```

## TLS / Certificates

### Certificate options
```bash
wget --no-check-certificate https://self-signed.example.com  # skip TLS verification
wget --ca-certificate=/path/to/ca.pem https://example.com
wget --certificate=client.pem --private-key=key.pem https://mtls.example.com
```

## Tips

- `wget -c` is the simplest way to resume a download — just re-run with `-c`
- `--mirror` + `-k` + `-p` is the classic recipe for offline website copies
- `-np` (no parent) prevents crawling above the target directory — almost always what you want for recursive
- `--random-wait` is polite to servers when spidering; without it, rapid requests may get you blocked
- `wget` creates a directory tree matching the remote structure; use `--cut-dirs` and `-nH` to flatten it
- Use `-N` (timestamping) for incremental mirrors — only downloads changed files
- `wget` respects `robots.txt` by default; use `-e robots=off` to ignore (be courteous)
- `--content-disposition` handles servers that set `Content-Disposition: attachment; filename=...`
- For single API calls, `curl` is usually better; `wget` shines for bulk and recursive downloads
