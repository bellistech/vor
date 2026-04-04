# The Theory of XML — Grammar, Schemas, and Processing Models

> *XML is a meta-language for defining markup languages. It's context-free, well-formed by syntactic rules, and valid by schema constraints. Its processing models (DOM, SAX, StAX) trade memory for flexibility. XPath and XSLT form a Turing-complete transformation language over tree structures.*

---

## 1. XML Grammar — Well-Formedness

### Formal Productions (W3C XML 1.0, Fifth Edition)

```
document    ::= prolog element Misc*
prolog      ::= XMLDecl? Misc* (doctypedecl Misc*)?
element     ::= EmptyElemTag | STag content ETag
STag        ::= '<' Name (S Attribute)* S? '>'
ETag        ::= '</' Name S? '>'
EmptyElemTag::= '<' Name (S Attribute)* S? '/>'
content     ::= CharData? ((element | Reference | CDSect | PI | Comment) CharData?)*
Attribute   ::= Name Eq AttValue
```

### Well-Formedness Rules

| Rule | Description | Violation Example |
|:-----|:-----------|:------------------|
| Single root element | Exactly one top-level element | `<a/><b/>` — two roots |
| Proper nesting | Tags close in LIFO order | `<a><b></a></b>` |
| Attribute uniqueness | No duplicate attributes per element | `<x a="1" a="2"/>` |
| Case sensitivity | Tag names are case-sensitive | `<A></a>` — mismatched |
| Quoted attributes | Attribute values must be quoted | `<x a=1/>` — unquoted |

### Grammar Class

XML is **context-free** with additional well-formedness constraints (matching tag names) that make full validation context-sensitive. In practice, parsers use a stack to match open/close tags.

---

## 2. Namespaces — Avoiding Name Collisions

### The Problem

Two XML vocabularies might use the same element name:

```xml
<!-- HTML table -->
<table><tr><td>Data</td></tr></table>

<!-- Furniture table -->
<table><legs>4</legs><material>Oak</material></table>
```

### The Solution: Namespace URIs

```xml
<html:table xmlns:html="http://www.w3.org/1999/xhtml">
  <html:tr><html:td>Data</html:td></html:tr>
</html:table>

<furn:table xmlns:furn="http://example.com/furniture">
  <furn:legs>4</furn:legs>
</furn:table>
```

### Expanded Name

Every element has an **expanded name** = (namespace URI, local name):

$$\text{QName} = \text{prefix}:\text{local} \to (\text{namespace URI}, \text{local})$$

The prefix is just a shorthand — `html:table` and `xhtml:table` are the same element if bound to the same URI.

### Default Namespace

```xml
<table xmlns="http://www.w3.org/1999/xhtml">
  <tr><td>Data</td></tr>
</table>
```

Unprefixed elements inherit the default namespace. Attributes do **not** inherit it.

---

## 3. Schema Languages — Type Systems for XML

### Three Schema Languages

| Language | Complexity | Type System | Standard |
|:---------|:-----------|:------------|:---------|
| DTD | Low | Weak (no data types) | XML 1.0 spec |
| XML Schema (XSD) | High | Rich (45 built-in types) | W3C |
| RELAX NG | Medium | Elegant (pattern-based) | ISO/IEC 19757 |

### DTD Example

```xml
<!DOCTYPE person [
  <!ELEMENT person (name, age, email?)>
  <!ELEMENT name (#PCDATA)>
  <!ELEMENT age (#PCDATA)>
  <!ELEMENT email (#PCDATA)>
  <!ATTLIST person id ID #REQUIRED>
]>
```

Content model operators:
- `,` = sequence
- `|` = choice
- `?` = optional (0 or 1)
- `*` = zero or more
- `+` = one or more

### XML Schema Type Hierarchy

```
anyType
 ├── anySimpleType
 │    ├── string
 │    │    ├── normalizedString
 │    │    │    └── token
 │    │    │         ├── language, Name, NMTOKEN, ...
 │    │    │         └── NCName → ID, IDREF, ENTITY
 │    ├── decimal
 │    │    └── integer
 │    │         ├── long → int → short → byte
 │    │         └── nonNegativeInteger → positiveInteger
 │    ├── boolean, float, double
 │    ├── dateTime, date, time, duration
 │    ├── base64Binary, hexBinary
 │    └── anyURI
 └── anyComplexType (elements with children/attributes)
```

### Validation as Constraint Satisfaction

Schema validation checks:

$$\text{valid}(d, S) = \text{well\_formed}(d) \land \bigwedge_{e \in d} \text{type\_valid}(e, S(e))$$

Each element must satisfy its declared type's constraints (content model, attribute list, data type facets).

---

## 4. Processing Models

### DOM — Document Object Model

Build a complete in-memory tree:

$$\text{Memory} = O(n), \quad n = \text{document size}$$

Typical overhead: **3-10x** the raw XML size (node objects, pointers, metadata).

| Advantage | Disadvantage |
|:----------|:------------|
| Random access | High memory |
| Modification | Slow for large documents |
| XPath evaluation | Must load entire document |

### SAX — Simple API for XML

Event-driven, forward-only:

```
startDocument()
startElement("person", attrs)
  startElement("name", attrs)
    characters("Alice")
  endElement("name")
  startElement("age", attrs)
    characters("30")
  endElement("age")
endElement("person")
endDocument()
```

$$\text{Memory} = O(d), \quad d = \text{max nesting depth}$$

| Advantage | Disadvantage |
|:----------|:------------|
| Constant memory | Forward-only |
| Fast | No random access |
| Streaming | Complex state management |

### StAX — Streaming API for XML

Pull-based (caller controls iteration), vs SAX push-based:

```java
while (reader.hasNext()) {
    int event = reader.next();
    if (event == START_ELEMENT && reader.getLocalName().equals("name")) {
        String name = reader.getElementText();
    }
}
```

---

## 5. XPath — Tree Query Language

### Data Model

XPath views an XML document as a tree of 7 node types:

| Node Type | Example | XPath Test |
|:----------|:--------|:-----------|
| Root | Document itself | `/` |
| Element | `<person>` | `person` |
| Attribute | `id="42"` | `@id` |
| Text | `"Alice"` | `text()` |
| Comment | `<!-- note -->` | `comment()` |
| Processing Instruction | `<?xml-stylesheet?>` | `processing-instruction()` |
| Namespace | `xmlns:x="..."` | `namespace::x` |

### Axes — Directions of Traversal

| Axis | Selects |
|:-----|:--------|
| `child` | Direct children (default) |
| `descendant` | All descendants |
| `parent` | Direct parent |
| `ancestor` | All ancestors |
| `following-sibling` | Later siblings |
| `preceding-sibling` | Earlier siblings |
| `self` | Current node |
| `attribute` | Attributes of current node |

### XPath Expressions — Examples

| Expression | Meaning |
|:-----------|:--------|
| `/bookstore/book` | All `book` children of root `bookstore` |
| `//book[@lang='en']` | All `book` elements anywhere with `lang="en"` |
| `//book[price > 30]` | Books with price > 30 |
| `//book[1]` | First book (1-indexed) |
| `//book[last()]` | Last book |
| `count(//book)` | Number of books |
| `//author/text()` | Text content of all author elements |

### XPath Complexity

For a document of size $n$ and XPath expression of size $q$:

$$\text{Evaluation time} = O(n \times q) \quad \text{(most practical expressions)}$$

Worst case with complex predicates: $O(n^2)$ or higher.

---

## 6. XSLT — Transformation Language

### Template-Based Transformation

XSLT transforms XML documents by matching templates against XPath patterns:

```xml
<xsl:template match="person">
  <div class="person">
    <h2><xsl:value-of select="name"/></h2>
    <p>Age: <xsl:value-of select="age"/></p>
  </div>
</xsl:template>
```

### XSLT is Turing Complete

XSLT 1.0 with recursion is **Turing complete** — it can compute any computable function:

```xml
<!-- Factorial via recursion -->
<xsl:template name="factorial">
  <xsl:param name="n"/>
  <xsl:choose>
    <xsl:when test="$n = 0">1</xsl:when>
    <xsl:otherwise>
      <xsl:variable name="sub">
        <xsl:call-template name="factorial">
          <xsl:with-param name="n" select="$n - 1"/>
        </xsl:call-template>
      </xsl:variable>
      <xsl:value-of select="$n * $sub"/>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>
```

---

## 7. Security Concerns

### XML External Entity (XXE) Attack

```xml
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<data>&xxe;</data>
```

The parser reads `/etc/passwd` and includes its content. **Defense:** Disable external entity processing.

### Billion Laughs (Entity Expansion)

```xml
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!-- ... -->
  <!ENTITY lol9 "&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;">
]>
<lolz>&lol9;</lolz>
```

Expansion: $10^9$ = 1 billion "lol" strings from ~1KB of input. **Defense:** Limit entity expansion depth/count.

---

## 8. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Grammar | Context-free with context-sensitive well-formedness |
| Namespaces | URI-based, prefix is just shorthand |
| Schema languages | DTD (simple), XSD (rich), RELAX NG (elegant) |
| DOM | In-memory tree, $O(n)$ memory, random access |
| SAX | Event-driven, $O(d)$ memory, forward-only |
| XPath | Tree query language, 13 axes, 7 node types |
| XSLT | Turing-complete transformation language |
| Security | XXE, billion laughs, SSRF via entities |

---

*XML is not a data format — it's a framework for creating data formats. Its verbosity is the cost of self-description. Its complexity enables schema validation, namespace composition, and Turing-complete transformation. When you need these features (document markup, enterprise integration, protocol specifications), nothing else provides them. When you don't, use JSON.*

## Prerequisites

- Tree-structured document models (DOM, SAX, StAX)
- Namespaces and schema validation (DTD, XSD, RELAX NG)
- XPath expressions and XSLT transformations
- Character encoding and entity references
