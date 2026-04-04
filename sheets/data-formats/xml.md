# XML (Extensible Markup Language)

Structured markup format for documents and data exchange with schemas and namespaces.

## Elements

### Basic structure

```bash
# <?xml version="1.0" encoding="UTF-8"?>
# <root>
#   <user id="1">
#     <name>Alice</name>
#     <email>alice@example.com</email>
#   </user>
#   <user id="2">
#     <name>Bob</name>
#     <email>bob@example.com</email>
#   </user>
# </root>
```

### Self-closing elements

```bash
# <br/>
# <img src="photo.jpg" alt="Photo"/>
# <config key="timeout" value="30"/>
```

### Empty vs self-closing (equivalent)

```bash
# <notes></notes>
# <notes/>
```

## Attributes

### Elements vs attributes

```bash
# Attribute style:
# <server host="db.example.com" port="5432"/>
#
# Element style:
# <server>
#   <host>db.example.com</host>
#   <port>5432</port>
# </server>
```

### Multiple attributes

```bash
# <connection
#     host="db.example.com"
#     port="5432"
#     ssl="true"
#     timeout="30"/>
```

## Namespaces

### Default namespace

```bash
# <catalog xmlns="http://example.com/books">
#   <book>
#     <title>The Art of Unix</title>
#   </book>
# </catalog>
```

### Prefixed namespace

```bash
# <bk:catalog xmlns:bk="http://example.com/books"
#              xmlns:pub="http://example.com/publisher">
#   <bk:book>
#     <bk:title>The Art of Unix</bk:title>
#     <pub:publisher>Addison-Wesley</pub:publisher>
#   </bk:book>
# </bk:catalog>
```

### SOAP example

```bash
# <soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
#   <soap:Header/>
#   <soap:Body>
#     <GetUser xmlns="http://example.com/api">
#       <userId>42</userId>
#     </GetUser>
#   </soap:Body>
# </soap:Envelope>
```

## CDATA

### Raw text (no escaping needed)

```bash
# <script><![CDATA[
#   if (a < b && c > d) {
#     console.log("no need to escape < > & here");
#   }
# ]]></script>
```

### Without CDATA (must escape)

```bash
# <message>Use &lt;tag&gt; for markup and &amp; for ampersand</message>
```

### Entity references

```bash
# &lt;    <
# &gt;    >
# &amp;   &
# &apos;  '
# &quot;  "
```

## DTD (Document Type Definition)

### Inline DTD

```bash
# <!DOCTYPE catalog [
#   <!ELEMENT catalog (book+)>
#   <!ELEMENT book (title, author, year)>
#   <!ELEMENT title (#PCDATA)>
#   <!ELEMENT author (#PCDATA)>
#   <!ELEMENT year (#PCDATA)>
#   <!ATTLIST book id ID #REQUIRED>
# ]>
```

### External DTD

```bash
# <!DOCTYPE catalog SYSTEM "catalog.dtd">
```

## XPath Basics

### Selection expressions

```bash
# /root/user              # absolute path — all <user> under <root>
# //user                  # any <user> anywhere in the document
# //user[@id='1']         # <user> with id attribute = '1'
# //user/name             # <name> children of <user>
# //user[1]               # first <user> (1-indexed)
# //user[last()]          # last <user>
# //user[position()<=3]   # first 3 users
# //user/name/text()      # text content of <name>
# //@id                   # all id attributes
# //user[@active='true']  # filter by attribute value
# //book[price>30]        # numeric comparison
```

### Using XPath with xmllint

```bash
xmllint --xpath '//user/name/text()' data.xml
xmllint --xpath 'count(//user)' data.xml
xmllint --xpath '//user[@id="1"]/email/text()' data.xml
```

### Using XPath with xmlstarlet

```bash
xmlstarlet sel -t -v '//user/name' data.xml
xmlstarlet sel -t -m '//user' -v 'name' -n data.xml    # iterate
```

## Common Patterns

### Configuration file

```bash
# <?xml version="1.0" encoding="UTF-8"?>
# <configuration>
#   <appSettings>
#     <add key="DatabaseServer" value="db.example.com"/>
#     <add key="MaxConnections" value="100"/>
#   </appSettings>
#   <connectionStrings>
#     <add name="Default"
#          connectionString="Server=db;Database=mydb;User=app"/>
#   </connectionStrings>
# </configuration>
```

### RSS feed

```bash
# <?xml version="1.0" encoding="UTF-8"?>
# <rss version="2.0">
#   <channel>
#     <title>My Blog</title>
#     <link>https://example.com</link>
#     <item>
#       <title>First Post</title>
#       <link>https://example.com/first</link>
#       <description>Hello world</description>
#     </item>
#   </channel>
# </rss>
```

### Maven pom.xml

```bash
# <project xmlns="http://maven.apache.org/POM/4.0.0">
#   <modelVersion>4.0.0</modelVersion>
#   <groupId>com.example</groupId>
#   <artifactId>myapp</artifactId>
#   <version>1.0.0</version>
#   <dependencies>
#     <dependency>
#       <groupId>junit</groupId>
#       <artifactId>junit</artifactId>
#       <version>4.13.2</version>
#       <scope>test</scope>
#     </dependency>
#   </dependencies>
# </project>
```

## Command-Line Tools

### Validate XML

```bash
xmllint --noout data.xml                       # syntax check
xmllint --schema schema.xsd data.xml           # validate against XSD
xmllint --format data.xml                      # pretty print
```

### Transform with XSLT

```bash
xsltproc transform.xsl data.xml
```

### Query and edit with xmlstarlet

```bash
xmlstarlet sel -t -v '//name' data.xml         # select values
xmlstarlet ed -u '//port' -v '8080' data.xml   # edit value
xmlstarlet el data.xml                         # list element paths
```

## Tips

- XML is case-sensitive: `<User>` and `<user>` are different elements.
- Always declare encoding: `<?xml version="1.0" encoding="UTF-8"?>`.
- Use CDATA for content that contains `<`, `>`, or `&` to avoid escaping.
- Namespaces prevent element name collisions when combining XML from different sources.
- XPath is 1-indexed, not 0-indexed: `//user[1]` is the first element.
- `xmllint --format` pretty-prints XML. `xmllint --noout` validates without output.
- Prefer JSON for APIs and data exchange. XML remains dominant in enterprise, SOAP, and document-centric workflows.
- Comments use `<!-- text -->` and cannot be nested.

## See Also

- json, yaml, html, regex, xpath, sed

## References

- [W3C XML 1.0 Specification](https://www.w3.org/TR/xml/) -- Extensible Markup Language (Fifth Edition)
- [W3C XML Namespaces](https://www.w3.org/TR/xml-names/) -- namespace specification
- [W3C XPath 1.0](https://www.w3.org/TR/xpath-10/) -- XPath expression language for XML
- [W3C XSLT 1.0](https://www.w3.org/TR/xslt-10/) -- XML stylesheet transformations
- [W3C XML Schema (XSD)](https://www.w3.org/TR/xmlschema11-1/) -- schema definition language
- [RELAX NG Specification](https://relaxng.org/spec-20011203.html) -- compact alternative to XSD
- [man xmllint](https://linux.die.net/man/1/xmllint) -- libxml2 CLI for parsing, validating, and formatting XML
- [man xmlstarlet](https://xmlstar.sourceforge.net/doc/UG/xmlstarlet-ug.html) -- command-line XML toolkit
- [SAX (Simple API for XML)](http://www.saxproject.org/) -- event-driven XML parsing interface
- [libxml2 Documentation](https://gnome.pages.gitlab.gnome.org/libxml2/devhelp/) -- C XML parsing library
