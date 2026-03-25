# zip (cross-platform archive)

Create and extract zip archives compatible with all operating systems.

## Create Archives

### Basic Creation

```bash
# Zip files
zip archive.zip file1.txt file2.txt

# Zip a directory recursively
zip -r project.zip project/

# Zip with compression level (0=store, 9=best)
zip -9 -r archive.zip /var/data/

# Zip silently
zip -q -r archive.zip project/
```

### Exclude Patterns

```bash
# Exclude files
zip -r project.zip project/ -x "*.git*" "*.DS_Store"

# Exclude directories
zip -r project.zip project/ -x "project/node_modules/*" "project/.git/*"

# Exclude from a file
zip -r project.zip project/ -x@exclude.lst
```

## Extract Archives

### Unzip

```bash
# Extract to current directory
unzip archive.zip

# Extract to a specific directory
unzip archive.zip -d /opt/

# Extract specific files
unzip archive.zip "path/to/file.txt"

# Extract matching pattern
unzip archive.zip "*.conf"

# Overwrite without prompting
unzip -o archive.zip

# Never overwrite (skip existing)
unzip -n archive.zip

# Quietly
unzip -q archive.zip
```

## List Contents

### View Without Extracting

```bash
# List files
unzip -l archive.zip

# List with more detail
unzip -v archive.zip

# Just filenames
zipinfo -1 archive.zip
```

## Password Protection

### Encrypted Zip

```bash
# Create password-protected zip
zip -e -r secure.zip /var/secrets/

# Create with password on command line (visible in history)
zip -P mysecretpass -r secure.zip /var/secrets/

# Extract password-protected zip
unzip -P mysecretpass secure.zip
```

## Update and Freshen

### Modify Existing Archives

```bash
# Add or update files in existing zip
zip -u archive.zip newfile.txt

# Freshen only (update existing files, don't add new)
zip -f archive.zip

# Delete a file from the archive
zip -d archive.zip "unwanted/file.txt"
```

## Split Archives

### Multi-Part Zip

```bash
# Create split archive (100MB parts)
zip -r -s 100m large_backup.zip /var/data/

# Result: large_backup.z01, large_backup.z02, ..., large_backup.zip

# Merge split archive before extracting
zip -s 0 large_backup.zip --out merged.zip
unzip merged.zip
```

## Symlinks and Permissions

### Preserve Metadata

```bash
# Store symlinks as symlinks (not the targets)
zip -r --symlinks archive.zip project/

# Note: zip does not preserve Unix permissions by default
# Use tar for permission-sensitive backups
```

## Tips

- `zip -r` is required for directories. Without `-r`, only the empty directory entry is added.
- zip's encryption (`-e`) uses the legacy ZipCrypto algorithm which is considered weak. For real security, use `7z` with AES-256 or `gpg`.
- `unzip -l` is the quickest way to inspect a zip without extracting.
- Use `zip -d` to remove files from an archive without recreating it.
- zip preserves directory structure by default. Use `unzip -j` to extract flat (ignore directory paths).
- `-x` exclude patterns must come after the source path in the command.
- zip is the best choice when archives need to be opened on Windows systems without additional software.

## References

- [Info-ZIP Home Page](https://infozip.sourceforge.net/)
- [zip(1) Man Page](https://man7.org/linux/man-pages/man1/zip.1.html)
- [unzip(1) Man Page](https://man7.org/linux/man-pages/man1/unzip.1.html)
- [ZIP File Format Specification (APPNOTE)](https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT)
- [libzip Library](https://libzip.org/)
- [Python zipfile Module](https://docs.python.org/3/library/zipfile.html)
