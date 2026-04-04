# SBOM (Software Bill of Materials)

An SBOM is a machine-readable inventory of all software components, libraries, and dependencies in an application, enabling vulnerability tracking, license compliance, and supply chain transparency across SPDX and CycloneDX standard formats.

## Generation Tools

### Syft (Anchore)

```bash
# Install syft
brew install syft

# Generate SBOM for container image
syft myregistry.io/myapp:v1.0 -o syft-json > sbom.syft.json
syft myregistry.io/myapp:v1.0 -o cyclonedx-json > sbom.cdx.json
syft myregistry.io/myapp:v1.0 -o spdx-json > sbom.spdx.json

# Generate SBOM for local directory
syft dir:. -o cyclonedx-json > sbom.cdx.json

# Generate SBOM for archive
syft docker-archive:myimage.tar -o spdx-json > sbom.spdx.json

# Generate SBOM with all packages
syft myapp:latest -o syft-json --catalogers all

# Generate SBOM for specific ecosystems
syft dir:. -o cyclonedx-json --select-catalogers go,npm
```

### Trivy SBOM Generation

```bash
# CycloneDX format
trivy image --format cyclonedx --output sbom.cdx.json myapp:latest

# SPDX format
trivy image --format spdx-json --output sbom.spdx.json myapp:latest

# Filesystem SBOM
trivy fs --format cyclonedx --output sbom.cdx.json .

# List all packages
trivy image --format cyclonedx --list-all-pkgs myapp:latest
```

### cdxgen (CycloneDX Generator)

```bash
# Install cdxgen
npm install -g @cyclonedx/cdxgen

# Generate CycloneDX SBOM for project
cdxgen -o sbom.cdx.json .

# Specify project type
cdxgen -t go -o sbom.cdx.json .
cdxgen -t python -o sbom.cdx.json .
cdxgen -t java -o sbom.cdx.json .

# Generate with evidence (occurrences)
cdxgen -o sbom.cdx.json --evidence .

# Deep mode (reachability analysis)
cdxgen -o sbom.cdx.json --deep .

# Generate for container image
cdxgen -t docker -o sbom.cdx.json myapp:latest
```

## SPDX Format

### SPDX Document Structure

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "myapp-v1.0",
  "documentNamespace": "https://example.com/myapp-v1.0",
  "creationInfo": {
    "created": "2025-01-15T10:00:00Z",
    "creators": ["Tool: syft-v1.0.0"],
    "licenseListVersion": "3.22"
  },
  "packages": [
    {
      "SPDXID": "SPDXRef-Package-openssl",
      "name": "openssl",
      "versionInfo": "3.1.4",
      "supplier": "Organization: OpenSSL Project",
      "downloadLocation": "https://www.openssl.org/source/",
      "licenseConcluded": "Apache-2.0",
      "licenseDeclared": "Apache-2.0",
      "checksums": [
        { "algorithm": "SHA256", "checksumValue": "abc123..." }
      ],
      "externalRefs": [
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:apk/alpine/openssl@3.1.4"
        }
      ]
    }
  ],
  "relationships": [
    {
      "spdxElementId": "SPDXRef-DOCUMENT",
      "relationshipType": "DESCRIBES",
      "relatedSpdxElement": "SPDXRef-Package-openssl"
    }
  ]
}
```

### SPDX CLI Tools

```bash
# Validate SPDX document
pip install spdx-tools
pyspdxtools_parser sbom.spdx.json

# Convert between SPDX formats
pyspdxtools_converter -i sbom.spdx.json -o sbom.spdx.rdf

# SPDX tag-value format
syft myapp:latest -o spdx > sbom.spdx.tv
```

## CycloneDX Format

### CycloneDX Document Structure

```json
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.6",
  "serialNumber": "urn:uuid:3e671687-395b-41f5-a30f-a58921a69b79",
  "version": 1,
  "metadata": {
    "timestamp": "2025-01-15T10:00:00Z",
    "tools": { "components": [{ "name": "syft", "version": "1.0.0" }] },
    "component": { "name": "myapp", "version": "1.0", "type": "application" }
  },
  "components": [
    {
      "type": "library",
      "name": "openssl",
      "version": "3.1.4",
      "purl": "pkg:apk/alpine/openssl@3.1.4",
      "licenses": [{ "license": { "id": "Apache-2.0" } }],
      "hashes": [{ "alg": "SHA-256", "content": "abc123..." }]
    }
  ],
  "dependencies": [
    {
      "ref": "pkg:apk/alpine/openssl@3.1.4",
      "dependsOn": ["pkg:apk/alpine/libcrypto3@3.1.4"]
    }
  ]
}
```

### CycloneDX CLI

```bash
# Install CycloneDX CLI
npm install -g @cyclonedx/cyclonedx-cli

# Validate CycloneDX SBOM
cyclonedx validate --input-file sbom.cdx.json --input-format json

# Convert between formats
cyclonedx convert --input-file sbom.cdx.json --output-file sbom.cdx.xml

# Merge multiple SBOMs
cyclonedx merge --input-files sbom1.cdx.json sbom2.cdx.json --output-file merged.cdx.json

# Diff two SBOMs
cyclonedx diff sbom-v1.cdx.json sbom-v2.cdx.json
```

## NTIA Minimum Elements

### Required Fields Checklist

```bash
# NTIA minimum elements for an SBOM:
# 1. Supplier Name          - who supplies the component
# 2. Component Name         - name of the software component
# 3. Version                - version identifier
# 4. Unique Identifier      - PURL, CPE, or SPDXID
# 5. Dependency Relationship - how components relate
# 6. Author of SBOM Data    - who created the SBOM
# 7. Timestamp              - when the SBOM was created

# Validate NTIA minimum elements with ntia-conformance-checker
pip install ntia-conformance-checker
ntia-checker --file sbom.spdx.json

# Check with sbomqs (SBOM quality score)
go install github.com/interlynk-io/sbomqs@latest
sbomqs score sbom.cdx.json

# Quality score breakdown
sbomqs score sbom.cdx.json --detailed
```

## VEX (Vulnerability Exploitability eXchange)

### OpenVEX Documents

```bash
# Install vexctl
go install github.com/openvex/vexctl@latest

# Create a VEX document
vexctl create \
  --product "pkg:oci/myapp@sha256:abc123" \
  --vuln CVE-2024-1234 \
  --status not_affected \
  --justification "vulnerable_code_not_present" \
  > myapp.vex.json

# Add another statement
vexctl add \
  --in-file myapp.vex.json \
  --product "pkg:oci/myapp@sha256:abc123" \
  --vuln CVE-2024-5678 \
  --status affected \
  --action-statement "Upgrade to v2.0" \
  > myapp-updated.vex.json

# Apply VEX to scan results (grype)
grype myapp:latest --vex myapp.vex.json

# Apply VEX to trivy results
trivy image --vex myapp.vex.json myapp:latest

# Merge VEX documents
vexctl merge file1.vex.json file2.vex.json > merged.vex.json
```

### VEX Status Values

```bash
# VEX status options:
# not_affected          - vulnerability does not affect this product
# affected              - vulnerability affects this product
# fixed                 - vulnerability was present but has been fixed
# under_investigation   - impact is being assessed

# Justifications for not_affected:
# component_not_present
# vulnerable_code_not_present
# vulnerable_code_cannot_be_controlled_by_adversary
# vulnerable_code_not_in_execute_path
# inline_mitigations_already_exist
```

## Dependency Graph Analysis

### Visualize Dependencies

```bash
# Generate dependency tree with syft
syft dir:. -o syft-json | jq '.artifactRelationships'

# Visualize with graphviz (from CycloneDX)
cyclonedx convert -i sbom.cdx.json -o sbom.dot --output-format dot
dot -Tpng sbom.dot -o dependency-graph.png

# Count direct vs transitive dependencies
cat sbom.cdx.json | jq '[.components[]] | length'
cat sbom.cdx.json | jq '[.dependencies[] | .dependsOn[]?] | length'

# Find components without version info
cat sbom.cdx.json | jq '[.components[] | select(.version == null or .version == "")] | length'
```

## CI/CD Integration

### Pipeline SBOM Workflow

```yaml
# GitHub Actions: generate, scan, attest
- name: Generate SBOM
  run: syft myapp:${{ github.sha }} -o cyclonedx-json > sbom.cdx.json

- name: Scan SBOM for vulnerabilities
  run: grype sbom:sbom.cdx.json --fail-on high

- name: Attest SBOM
  run: |
    cosign attest \
      --predicate sbom.cdx.json \
      --type cyclonedx \
      myregistry.io/myapp@${{ steps.build.outputs.digest }}

- name: Upload SBOM artifact
  uses: actions/upload-artifact@v4
  with:
    name: sbom
    path: sbom.cdx.json
```

## Tips

- Use CycloneDX 1.6+ for VEX integration and dependency graph support in a single document
- Generate SBOMs at build time in CI rather than scanning deployed images for most accurate results
- Include both direct and transitive dependencies; use `--catalogers all` with syft for completeness
- Validate SBOMs against NTIA minimum elements before publishing to catch missing metadata
- Use Package URL (PURL) as the unique identifier for cross-tool interoperability
- Attach SBOMs to container images with cosign and sign them for tamper evidence
- Use VEX documents to suppress false positives rather than ignoring CVEs in scanner config
- Store SBOMs alongside releases in your artifact repository for auditing and compliance
- Diff SBOMs between versions to detect unexpected dependency changes before release
- SPDX is ISO/IEC 5962:2021; CycloneDX is ECMA-424 -- both are accepted by US federal mandates
- Run `sbomqs score` to measure SBOM quality and identify missing fields

## See Also

- cosign, grype, trivy, vulnerability-scanning, container-security, compliance

## References

- [SPDX Specification](https://spdx.github.io/spdx-spec/v2.3/)
- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
- [NTIA Minimum Elements](https://www.ntia.gov/report/2021/minimum-elements-software-bill-materials-sbom)
- [OpenVEX Specification](https://github.com/openvex/spec)
- [Syft Documentation](https://github.com/anchore/syft)
- [EO 14028 - Improving the Nation's Cybersecurity](https://www.whitehouse.gov/briefing-room/presidential-actions/2021/05/12/executive-order-on-improving-the-nations-cybersecurity/)
