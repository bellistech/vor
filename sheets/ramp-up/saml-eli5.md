# SAML — ELI5 (Concert Tickets for Enterprise Login)

> SAML is how a company lets you log in to fifty different apps without typing your password fifty times: the company runs a box office that hands you a signed ticket, and every app is a venue gate that trusts the signature.

## Prerequisites

(none — but `cs ramp-up tls-eli5` helps; SAML rides on TLS)

You do not need to know what XML is. You do not need to know what a "browser redirect" is. You do not need to have ever logged in to anything fancier than a phone app. By the end of this sheet you will have decoded a real SAML ticket, you will know why `Audience restriction` is the most common error in the world, and you will know why every five years another team accidentally re-invents a way to forge tickets that the gate still accepts.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is SAML?

### The concert-ticket picture

Imagine you are going to a concert. There is a really popular band playing tonight. You want to get inside the venue. The venue is a giant building with a gate, and there is a security guard at the gate. You walk up. The guard says, "Show me your ticket."

You don't have a ticket yet. So the guard says, "Go to the box office across the street. Get a ticket. Come back."

You walk across the street to the box office. The box office is run by people who know who you are. They have your ID on file. They know whether you paid for your ticket. They know whether you are old enough to be at this concert. They are the **only people in the whole world** allowed to print real tickets for tonight's show.

You hand them your ID. They check it. They check that you paid. They check your age. Then they print you a ticket. The ticket says:

```
ADMIT ONE
Name: Jane Smith
Date: 2026-04-27
Show: Big Band Tonight
Venue: The Big Dome
Issued by: Big Dome Box Office
```

And here is the most important part: **the ticket has a special holographic stamp on it.** That stamp is from the box office. Only the box office has the machine that makes that stamp. You can't fake the stamp. Anyone who looks at the ticket can see the stamp and know it is real.

You take the ticket back across the street. You hand it to the guard at the gate. The guard does **not** call the box office on the phone. The guard does not need to. The guard just looks at the holographic stamp, sees it is real, reads your name off the ticket, and lets you in.

That is SAML.

### The three roles

SAML has three players. Always three. We are going to use the same three people for the rest of the sheet, so memorize them.

**You** are the user. You are the one trying to log in. You are the concert-goer. In SAML talk, you are sometimes called the **Subject** or the **Principal**.

**The box office** is the **Identity Provider**, or **IdP**. The IdP is the only thing in the whole system that knows who you are, knows your password, and is allowed to print tickets. There is usually one IdP per company. At a big company, the IdP might be Okta, or Azure AD (now called Entra ID), or AD FS, or Auth0, or Keycloak, or Google Workspace, or OneLogin, or Ping, or Shibboleth.

**The venue** is the **Service Provider**, or **SP**. The SP is the app you are trying to log in to. Salesforce is an SP. Slack is an SP. Workday is an SP. Tableau is an SP. Your custom internal expense-report tool is an SP. Each company will have dozens or hundreds of SPs. Every one of them needs a ticket from the IdP to let you in.

That is the whole cast of characters: you, the box office, the gate. Three players. Always the same three.

### What is the ticket actually called?

The ticket has a name. SAML calls it an **Assertion**. An Assertion is a chunk of XML that says "I, the box office, hereby assert that this user has logged in, that their email is jane@example.com, that they belong to the engineering group, and that this assertion is good for the next ten minutes." It is wrapped inside a bigger envelope called a **Response**, which has its own headers and its own signature.

Both the Assertion and the Response can be signed. Either, or both. Most SPs require **at least one** signature. Most secure setups sign **both**.

### Why bother? Why not just log in with a password?

You could log in with a password to every single app. That is what the world looked like in 1998. It was a disaster.

- People used the same password everywhere, so when one app got hacked, every app got hacked.
- People wrote passwords on sticky notes.
- IT had to reset passwords for fifty different apps when someone got fired.
- There was no single place to enforce MFA.
- There was no single place to log who logged in to what.
- When you joined a company, IT had to create accounts for you in every single app.

SAML fixes all of this. There is one place — the IdP — that knows your password. If IT disables you in the IdP, every app is locked instantly. If you are required to use MFA, the IdP can enforce it once and every app benefits. If a hacker steals your password, they only have to be detected in one place.

### Why is the ticket so weird?

The ticket is XML. It is huge. It has dozens of fields. It has nested tags. It has signatures that depend on whitespace. It has time windows. It has audience restrictions. It has subject confirmations.

SAML is from 2002 (1.1) and 2005 (2.0). That was the **golden age of XML**. Everything was XML. Java was XML-pilled. Web services were SOAP, which was XML. Configuration was XML. SAML is a child of that era. If we were inventing it today, it would be JSON and we would call it JWT, and we did, and that's OAuth/OIDC, and the next sheet is about that.

For now, hold your nose. SAML is XML. We're going to make peace with it.

## SAML 1.1 vs SAML 2.0

There were two big versions of SAML. You will only ever see SAML 2.0 in production today. But people still casually say "SAML" without saying which version, and once in a blue moon a really old vendor will hand you a SAML 1.1 metadata file. Here is the lay of the land.

### SAML 1.0 (November 2002)

The first version. It worked, but barely. Few products implemented it.

### SAML 1.1 (September 2003)

Patched up some bugs in 1.0. It was used by some early federation projects (notably the Liberty Alliance and InCommon's first iteration). It is **not** wire-compatible with SAML 2.0. Different XML namespaces. Different element names. Different bindings. If you point a SAML 2.0 SP at a SAML 1.1 IdP, nothing works.

### SAML 2.0 (March 2005, OASIS Standard)

The version we use. It came out of merging SAML 1.1 with the Liberty Alliance ID-FF (Identity Federation Framework) and the Shibboleth project. OASIS ratified it as a standard in March 2005, and it has not changed in twenty-one years. There have been **errata** (bug fixes to the spec text) but no SAML 2.1 or SAML 3.0. It froze.

Everything in this sheet is SAML 2.0 unless we explicitly say otherwise. If somebody hands you metadata and you are not sure which version it is, look at the XML namespaces:

```xml
<!-- SAML 2.0 -->
xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"

<!-- SAML 1.1 -->
xmlns:saml="urn:oasis:names:tc:SAML:1.0:assertion"
xmlns:samlp="urn:oasis:names:tc:SAML:1.0:protocol"
```

If you see `2.0`, you are home. If you see `1.0` or `1.1`, you are in a museum.

> **Version note.** SAML 2.0 has been the standard since March 2005. The XML Signature spec it leans on dates back to 2002. The XML Canonicalization spec dates back to 2001. SAML is older than your favorite junior engineer.

## SP-Initiated vs IdP-Initiated Flows

There are two ways the dance starts. Both end the same way (you logged in to the SP), but they begin in different places.

### SP-initiated flow

You go to the app first. You type `https://salesforce.example.com` into your browser. Salesforce sees you are not logged in. Salesforce says, "Hold on, let me send you to the box office." Your browser is redirected to the IdP. You log in at the IdP. The IdP sends you back to Salesforce with a fresh ticket. Salesforce reads the ticket and lets you in.

This is **the most common flow**. It is what people mean when they say "SAML SSO."

### IdP-initiated flow

You go to the IdP first. The IdP shows you a portal — a page with a grid of icons, one icon per app. You see icons for Salesforce, Slack, Workday, GitHub, your expense tool, your wiki, etc. You are already logged in to the IdP because you logged in to see the portal. You click the Slack icon. The IdP packages up a ticket for Slack and **pushes you over to Slack** with the ticket already in your hand.

This is convenient for users. It is **more dangerous than SP-init.** The SP did not ask for the ticket; the ticket showed up out of nowhere. So:

- The SP cannot include an `InResponseTo` value (because there was no request to respond to).
- The SP cannot tie the response back to a specific RelayState that it issued.
- An attacker who steals an IdP-init response from one user's traffic can sometimes replay it to a different user's session more easily than with SP-init.

A common rule of thumb: **prefer SP-init. Allow IdP-init only when you must, and lock it down extra hard.**

## The SAML Dance (SP-Initiated, HTTP-POST Binding)

We are going to walk through the most common flow step by step. SP-initiated. HTTP-POST binding (we'll explain bindings in a minute, but POST is the common one). All seven steps.

### Step 1: User → SP: "I want in."

You type `https://app.example.com/dashboard` in your browser. Or you click a bookmark. Either way, your browser sends a regular HTTP GET to the SP.

```
GET /dashboard HTTP/1.1
Host: app.example.com
Cookie: (no session cookie yet)
```

The SP sees that you have no session cookie. You are not logged in. The SP knows you need to be authenticated.

### Step 2: SP → User: "Here's a SAML AuthnRequest. Take it to the IdP."

The SP generates a SAML **AuthnRequest** — an XML document that says, in essence, "I am `app.example.com`, please authenticate this user, send the answer to my ACS URL, and here is a unique ID so I can match the response when it comes back."

```xml
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_8a1b9c7e3f4d2"
    Version="2.0"
    IssueInstant="2026-04-27T15:34:21Z"
    Destination="https://idp.example.com/saml/sso"
    AssertionConsumerServiceURL="https://app.example.com/saml/acs"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
  <saml:Issuer>https://app.example.com/saml/metadata</saml:Issuer>
  <samlp:NameIDPolicy
      Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
      AllowCreate="true"/>
</samlp:AuthnRequest>
```

The SP now needs to deliver this XML to the IdP. The SP does not call the IdP itself; the SP **uses your browser as the courier**.

The SP returns an HTML page. The page has a form in it. The form's action is the IdP's SSO URL. The form has a hidden input named `SAMLRequest` whose value is the base64-encoded XML. The page has a tiny JavaScript snippet that auto-submits the form the instant it loads.

```html
<html><body onload="document.forms[0].submit()">
  <form method="POST" action="https://idp.example.com/saml/sso">
    <input type="hidden" name="SAMLRequest" value="PHNhbWxwOkF1dG..."/>
    <input type="hidden" name="RelayState" value="/dashboard"/>
  </form>
</body></html>
```

You never see this page. It flashes for less than one frame.

### Step 3: User → IdP (via redirect): AuthnRequest payload

Your browser auto-submits the form. The browser sends a POST to the IdP:

```
POST /saml/sso HTTP/1.1
Host: idp.example.com
Content-Type: application/x-www-form-urlencoded

SAMLRequest=PHNhbWxwOkF1dG...&RelayState=%2Fdashboard
```

The IdP receives the POST. It decodes the base64. It parses the XML. It checks: is this `Issuer` a known SP? Is the `Destination` me? Does the `AssertionConsumerServiceURL` match what I have in metadata for that SP? If anything fails, the IdP rejects the request and the dance dies right here.

### Step 4: IdP authenticates the user

The IdP shows you a login page. You type your username and password. Maybe the IdP also asks for an MFA code, or sends you a push notification to your phone, or asks you to touch a YubiKey. The IdP does whatever it needs to do to be sure you are you.

This step is **completely internal to the IdP**. The SP has no idea what is happening. The SP just waits for the browser to come back.

### Step 5: IdP → User: "Here's a signed SAML Response. Take it to the SP."

Once the IdP is happy, it builds a SAML **Response** that wraps an **Assertion**. The Response says "here is my answer to your AuthnRequest" and the Assertion says "this user is jane@example.com, she is in the engineering group, this assertion was issued at 15:34:25, and it expires at 15:39:25."

The IdP signs the Response (and/or the Assertion) with its private key. It encrypts the Assertion if encryption is configured. It base64-encodes the whole thing.

The IdP returns an HTML page to your browser. Just like step 2, this page has an auto-submitting form. The form's action is the SP's ACS URL.

```html
<html><body onload="document.forms[0].submit()">
  <form method="POST" action="https://app.example.com/saml/acs">
    <input type="hidden" name="SAMLResponse" value="PHNhbWxwOlJlc3..."/>
    <input type="hidden" name="RelayState" value="/dashboard"/>
  </form>
</body></html>
```

### Step 6: User → SP (via auto-submitting form POST): Response payload

Your browser auto-submits. The SP receives the POST.

```
POST /saml/acs HTTP/1.1
Host: app.example.com
Content-Type: application/x-www-form-urlencoded

SAMLResponse=PHNhbWxwOlJlc3...&RelayState=%2Fdashboard
```

### Step 7: SP validates signature, extracts identity, creates session

The SP decodes the base64. The SP parses the XML. The SP checks **a long list of things** before it trusts anything. We'll list them in order because every single bug in SAML history is about one of these checks being skipped, weakened, or done in the wrong order.

1. **Decrypt the Assertion** if it is encrypted. Use the SP's private key.
2. **Verify the signature** on the Response or the Assertion (or both). Use the IdP's public certificate from the IdP metadata.
3. **Check `Destination`** — it must be the SP's ACS URL.
4. **Check `Issuer`** — it must be the IdP we expect.
5. **Check `InResponseTo`** — it must match the `ID` of the AuthnRequest we sent.
6. **Check `AudienceRestriction`** — the assertion's `<Audience>` must include our entityID.
7. **Check `NotBefore` and `NotOnOrAfter`** — current time must be in that window (with small clock skew tolerance).
8. **Check `SubjectConfirmation`** — the `Recipient` must match our ACS URL, the `NotOnOrAfter` must not have passed.
9. **Check that the `ID` is fresh** — has it been used before? (Replay protection.)
10. **Pull out the NameID and attributes.** Map them to a local user. Create a session cookie. Redirect the user to wherever they originally wanted to go (using `RelayState`).

If you skip any of these checks, you have a security bug. SAML history is the history of teams skipping one of these checks.

### ASCII diagram of the SP-init dance

```
+----+        +-----------+        +----------+
|User|        |    SP     |        |   IdP    |
|    |        |app.exa... |        |idp.ex... |
+----+        +-----------+        +----------+
   |                |                    |
   | (1) GET /dash  |                    |
   |--------------->|                    |
   |                |                    |
   |  (2) HTML form w/ SAMLRequest       |
   |<---------------|                    |
   |                |                    |
   | (3) POST SAMLRequest (auto-submit)  |
   |------------------------------------>|
   |                |                    |
   |  (4) Login page (user types pw/MFA) |
   |<------------------------------------|
   |                |                    |
   | (4b) POST credentials               |
   |------------------------------------>|
   |                |                    |
   |  (5) HTML form w/ signed SAMLResp   |
   |<------------------------------------|
   |                |                    |
   | (6) POST SAMLResponse (auto-submit) |
   |--------------->|                    |
   |                |                    |
   |   (SP validates signature, etc.)    |
   |                |                    |
   |  (7) 302 to /dashboard + cookie     |
   |<---------------|                    |
   |                |                    |
   | (8) GET /dashboard w/ cookie        |
   |--------------->|                    |
   |                |                    |
   |  Application page                   |
   |<---------------|                    |
   |                |                    |
```

### ASCII diagram of the IdP-init dance

```
+----+        +-----------+        +----------+
|User|        |    SP     |        |   IdP    |
+----+        +-----------+        +----------+
   |                |                    |
   | (1) GET /portal (already logged in) |
   |------------------------------------>|
   |                |                    |
   |  (2) Portal page w/ app icons       |
   |<------------------------------------|
   |                |                    |
   | (3) Click "Slack" icon              |
   |------------------------------------>|
   |                |                    |
   |  (4) HTML form w/ unsolicited       |
   |      signed SAMLResponse            |
   |<------------------------------------|
   |                |                    |
   | (5) POST SAMLResponse to ACS        |
   |--------------->|                    |
   |                |                    |
   |   (SP validates — note: no          |
   |    InResponseTo to match!)          |
   |                |                    |
   |  (6) 302 to /home + cookie          |
   |<---------------|                    |
   |                |                    |
```

## SAML Bindings

A **binding** is "how the XML gets from one party to another." Same XML, different transport. SAML defines four bindings.

### HTTP-POST binding

The most common. The XML is base64-encoded and stuffed into a hidden form field. An HTML page with auto-submit JavaScript moves the form via the user's browser. We just walked through this above.

Pros: large payloads work fine (forms can be huge), signatures are easy.
Cons: requires JavaScript or a click-to-submit fallback.

### HTTP-Redirect binding

The XML is **DEFLATE-compressed** (raw deflate, not gzip), then base64-encoded, then URL-encoded, then put on the URL as the `SAMLRequest` query parameter. The user is redirected via a 302.

Pros: no form needed, plain redirect, works without JS.
Cons: URL length limits (most servers cap at 8 KB). Used mostly for AuthnRequests because they are small. Not used for Responses because Responses are big and signed.

```
GET /saml/sso?SAMLRequest=fVLLT...&RelayState=%2Fdashboard&SigAlg=...&Signature=... HTTP/1.1
Host: idp.example.com
```

### HTTP-Artifact binding

The IdP sends the user's browser to the SP carrying not a full assertion, but a tiny **artifact** ID — a short reference number. The SP then opens a **direct backend SOAP connection to the IdP** and trades the artifact for the real assertion.

Pros: the assertion never touches the user's browser, which is useful when you have privacy concerns or don't want big payloads in the URL.
Cons: requires the SP to make an outbound call to the IdP. Lots of SaaS SPs cannot do this. Rarely used outside government/edu.

```
+----+    +----+    +----+
|User|    | SP |    |IdP |
+----+    +----+    +----+
  |         |          |
  |  artifact via redirect
  |-------->|          |
  |         |  SOAP: "I have artifact X, send me the assertion"
  |         |--------->|
  |         |  SOAP: "Here is the assertion"
  |         |<---------|
```

### SOAP binding

Pure backend HTTP POST with a SOAP envelope. No browser involvement. Used for:
- Artifact resolution (above).
- Single Logout (sometimes).
- Attribute queries (rare).

You will almost never write SOAP-binding code by hand. If your library needs it, the library handles it.

## NameID

The `<NameID>` is the field in the assertion that **identifies the user**. It is the answer to "who is this?"

NameID has a **format**, which says how to interpret the value. There are six standard formats:

### emailAddress

`urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`

The value is an email address: `jane@example.com`. This is the most common in real-world deployments. Easy to read, easy to map to a local user account by email.

```xml
<saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
    jane@example.com
</saml:NameID>
```

### persistent

`urn:oasis:names:tc:SAML:2.0:nameid-format:persistent`

An opaque, **pairwise** identifier. The IdP gives a different opaque string to each SP for the same user. SP-A sees `aXq71!@xYz`, SP-B sees `kLm99##pQr`, and neither can correlate the two. This is privacy-preserving — useful when you don't want SPs to be able to link the same person across services.

```xml
<saml:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">
    aXq71xYz3pNm0kLp9wRs
</saml:NameID>
```

### transient

`urn:oasis:names:tc:SAML:2.0:nameid-format:transient`

A one-time-use random ID. New value every login. The SP cannot remember the user across sessions by NameID alone (it can use other attributes if it must). Used for fully anonymous access where the SP doesn't need a stable identity at all.

### unspecified

`urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified`

"Here is a string, the format is whatever the IdP and SP agreed on out of band." The most chaotic option. Used when neither side wanted to commit to a format. Often the source of integration bugs.

### X509SubjectName

`urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName`

The value is the X.509 subject DN of the user, e.g. `CN=Jane Smith, OU=Eng, O=Example`. Used in setups where the user has a client certificate.

### kerberos

`urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos`

The value is a Kerberos principal, e.g. `jane@EXAMPLE.COM`. Used in tight integrations with Active Directory.

### Choosing a format

If you are integrating with a SaaS SP, **check what format the SP expects.** Most SaaS SPs want `emailAddress`. Some want `persistent`. Salesforce wants the Federation ID to match an internal field. If you send the wrong format, you get the dreaded `NameID format unspecified` error. Check the SP's docs.

## Attribute Statements

NameID alone is rarely enough. The SP probably also wants to know the user's display name, their email (separately from NameID, which might be a persistent ID), their groups, their job title, their department, and so on.

These extra fields ride in an `<AttributeStatement>` block inside the assertion.

```xml
<saml:AttributeStatement>
  <saml:Attribute Name="emailAddress"
                  NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
    <saml:AttributeValue>jane@example.com</saml:AttributeValue>
  </saml:Attribute>
  <saml:Attribute Name="displayName">
    <saml:AttributeValue>Jane Smith</saml:AttributeValue>
  </saml:Attribute>
  <saml:Attribute Name="groups">
    <saml:AttributeValue>engineering</saml:AttributeValue>
    <saml:AttributeValue>oncall-platform</saml:AttributeValue>
    <saml:AttributeValue>github-admin</saml:AttributeValue>
  </saml:Attribute>
  <saml:Attribute Name="department">
    <saml:AttributeValue>Platform</saml:AttributeValue>
  </saml:Attribute>
  <saml:Attribute Name="employeeId">
    <saml:AttributeValue>E-12345</saml:AttributeValue>
  </saml:Attribute>
</saml:AttributeStatement>
```

Each `<Attribute>` has a name. Each can have one or more `<AttributeValue>` children. Multi-valued is normal (it is how groups are usually expressed).

### Attribute name conventions

There is no global naming scheme. Each SP picks names. SaaS SPs publish a list of "attribute names we want" in their docs. Common conventions:

- **basic**: `Name="email"`. Plain string.
- **URI**: `Name="urn:oid:0.9.2342.19200300.100.1.3"` (the LDAP-style OID for `mail`). Verbose, used by older federations like Shibboleth/InCommon.
- **Microsoft style**: `Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"`. AD FS / Azure AD use these long URLs.

When you wire up a new SP, **expect to spend a day on attribute mapping** even when everything else works. It is the single most fiddly part of any integration.

## Signing and Encryption

### What gets signed

You can sign:

- The whole `<Response>` element (covers the assertion plus everything else).
- The `<Assertion>` element on its own.
- Both.

You should pick at least one. **Most secure setups sign both.**

The signature is an enveloped XML Digital Signature (XMLDSig). The `<Signature>` element lives **inside** the element it signs, and the signed reference uses an enveloped-signature transform to logically remove itself before computing the digest.

```xml
<samlp:Response ID="_resp_123" ...>
  <saml:Issuer>https://idp.example.com/...</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod
          Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod
          Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_resp_123">
        <ds:Transforms>
          <ds:Transform
              Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform
              Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        </ds:Transforms>
        <ds:DigestMethod
            Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>iN8qf2...</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>3kR9...</ds:SignatureValue>
    <ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIDXTC...</ds:X509Certificate></ds:X509Data></ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>...</samlp:Status>
  <saml:Assertion>...</saml:Assertion>
</samlp:Response>
```

### What gets encrypted

You can encrypt the `<Assertion>` so that **even the user's browser cannot read it** in transit. The Response wraps an `<EncryptedAssertion>` instead of a plain `<Assertion>`. The SP decrypts it using its private key on arrival.

You can also encrypt individual `<NameID>` or `<Attribute>` values using `<EncryptedID>` or `<EncryptedAttribute>`. Rare in practice.

```xml
<samlp:Response ...>
  <saml:Issuer>...</saml:Issuer>
  <ds:Signature>...</ds:Signature>
  <samlp:Status>...</samlp:Status>
  <saml:EncryptedAssertion>
    <xenc:EncryptedData
        xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"
        Type="http://www.w3.org/2001/04/xmlenc#Element">
      <xenc:EncryptionMethod
          Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
      <ds:KeyInfo><xenc:EncryptedKey>...</xenc:EncryptedKey></ds:KeyInfo>
      <xenc:CipherData><xenc:CipherValue>AB12...</xenc:CipherValue></xenc:CipherData>
    </xenc:EncryptedData>
  </saml:EncryptedAssertion>
</samlp:Response>
```

The encryption is hybrid: a fresh symmetric key is generated, the assertion is encrypted with that symmetric key (AES-128-CBC, AES-256-GCM), and that symmetric key is itself encrypted with the SP's RSA public key (RSA-OAEP) and embedded in the message.

### Signed and encrypted: most secure

Best practice: **sign the Response, encrypt the Assertion**. That way:

- The signature proves the IdP issued the message and nothing was tampered with in transit.
- The encryption protects the assertion contents (NameID, attributes, groups) from anything that touches the response between IdP and SP — including the user's own browser, browser extensions, browser history, accidental logging.

### ASCII anatomy of a signed-and-encrypted Response

```
+============= samlp:Response (signed) =============+
| Issuer:    https://idp.example.com/...            |
|                                                   |
|  +---- ds:Signature (over the whole Response) --+ |
|  | SignedInfo                                   | |
|  |   CanonicalizationMethod: exc-c14n           | |
|  |   SignatureMethod: rsa-sha256                | |
|  |   Reference URI="#_resp_123"                 | |
|  |     Transforms: enveloped-signature, c14n    | |
|  |     DigestMethod: sha256                     | |
|  |     DigestValue: iN8qf2...                   | |
|  | SignatureValue: 3kR9...                      | |
|  | KeyInfo: X509Certificate                     | |
|  +----------------------------------------------+ |
|                                                   |
|  Status: Success                                  |
|                                                   |
|  +======= saml:EncryptedAssertion =========+      |
|  | EncryptedData                           |      |
|  |   EncryptionMethod: aes256-gcm          |      |
|  |   EncryptedKey                          |      |
|  |     EncryptionMethod: rsa-oaep          |      |
|  |     CipherValue: (sym key, RSA-encrypted)|     |
|  |   CipherValue: (assertion, AES-encrypted)|     |
|  +-----------------------------------------+      |
|     (when decrypted, contains:)                   |
|     +------- saml:Assertion (signed) -----+       |
|     | Issuer / Subject / NameID           |       |
|     | Conditions (NotBefore, NotOnOrAfter)|       |
|     | AuthnStatement (AuthnContext)       |       |
|     | AttributeStatement (email, groups)  |       |
|     | (own ds:Signature)                  |       |
|     +-------------------------------------+       |
+===================================================+
```

## XML Canonicalization (c14n)

Here is a thing that drove every SAML implementer mad in 2005 and still drives every junior implementer mad today.

### The problem

Two XML documents can **look different** but **mean the same thing**:

```xml
<!-- Document A -->
<root xmlns:a="http://a" xmlns:b="http://b"><a:foo>x</a:foo></root>

<!-- Document B -->
<root xmlns:b="http://b"  xmlns:a="http://a">
  <a:foo>x</a:foo>
</root>
```

Same meaning. Different bytes. Different SHA-256 hashes. So an XML signature that hashes the bytes would say "tampered!" even though nothing meaningful changed.

To fix this, the W3C invented **Canonical XML** (c14n, where 14 is the count of letters between the c and the n). Canonical XML is a **rule book for converting any XML into a single, unambiguous byte form.** If you canonicalize Document A and Document B above, you get the same bytes. Then you hash those bytes. Then you sign that hash. Then verification works.

### The flavors

- **Canonical XML 1.0** (W3C, 2001). The first version. Has trouble with documents that get embedded in other documents because it includes ancestor namespaces.
- **Canonical XML 1.1** (W3C, 2008). Patches some `xml:base` and `xml:id` issues.
- **Exclusive XML Canonicalization 1.0** (W3C, 2002). Solves the embedding problem. **Drops** ancestor namespaces that aren't used inside the signed subtree. SAML uses this. Algorithm URI: `http://www.w3.org/2001/10/xml-exc-c14n#`.

### Why this matters

Every SAML library has to canonicalize the same way the IdP did. If the IdP uses exclusive c14n and the SP uses inclusive c14n, signatures that should verify will fail. If your library has a canonicalization bug, signatures that **shouldn't** verify will succeed — that's how some XSW attacks (next section) get their foothold.

### What this looks like when it goes wrong

```
ERROR: Signature verification failed
  expected digest: iN8qf2...
  actual digest:   bX1tQa...
```

The bytes the SP canonicalized do not hash to the value embedded in the signature. That can mean:
- The XML was modified in transit (real tampering).
- The SP used a different c14n algorithm than the IdP.
- A whitespace-handling bug in one library or the other.
- The SP re-parsed the XML and the parser silently normalized something (e.g. attribute order, default namespaces).

The fix is rarely "edit the XML by hand." The fix is "stop re-parsing the XML between signature verification and use." See the XSW section.

## Metadata

Metadata is a chunk of XML that **describes one party**. There is IdP metadata and SP metadata. Each side publishes its metadata so the other side knows:

- What is your entityID?
- What endpoints do you have? (SSO, SLO, ACS, ArtifactResolution.)
- What bindings do you support? (HTTP-POST, HTTP-Redirect, HTTP-Artifact.)
- What public certificates do you use for signing and encryption?
- What NameID formats do you support?
- What attributes do you require or release?

A **federation** is just two parties exchanging metadata files. That's it. No magic protocol. Either you point your IdP at a URL and it pulls the SP's metadata.xml, or you copy-paste the file. Done. The federation is established.

### IdP metadata sample

```xml
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://idp.example.com/metadata">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCC...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://idp.example.com/saml/slo"/>
    <NameIDFormat>
        urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    </NameIDFormat>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://idp.example.com/saml/sso"/>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://idp.example.com/saml/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>
```

### SP metadata sample

```xml
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://app.example.com/saml/metadata">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
                   AuthnRequestsSigned="true"
                   WantAssertionsSigned="true">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCC...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <KeyDescriptor use="encryption">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCC...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://app.example.com/saml/slo"/>
    <NameIDFormat>
        urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    </NameIDFormat>
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://app.example.com/saml/acs"
        index="0" isDefault="true"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### ASCII diagram of federation metadata exchange

```
+-------------+                      +-------------+
|     IdP     |                      |     SP      |
| idp.example |                      | app.example |
+-------------+                      +-------------+
       |                                    |
       |  GET /sp/metadata                  |
       |----------------------------------->|
       |  <EntityDescriptor                 |
       |    entityID="..." ACS="...">       |
       |<-----------------------------------|
       |                                    |
       |  GET /idp/metadata                 |
       |<-----------------------------------|
       |  <EntityDescriptor                 |
       |    entityID="..." SSO="...">       |
       |----------------------------------->|
       |                                    |
   admin imports the file or sets a URL fetch
   on each side. From now on they trust each other.
```

Some big federations (like InCommon, eduGAIN, the UK Access Federation) publish a single **aggregate metadata file** with hundreds of EntityDescriptors. Members fetch the aggregate on a schedule, validate its signature, and pre-populate their trust stores from it.

## Single Logout (SLO)

SLO is the "log me out of EVERYTHING" feature. It is, in theory, lovely. In practice, it is broken on most deployments.

### How it should work (SP-initiated SLO)

1. User is logged in to SP-A, SP-B, SP-C, all via the same IdP session.
2. User clicks "Log out" on SP-A.
3. SP-A sends a `<LogoutRequest>` to the IdP.
4. The IdP gets the LogoutRequest.
5. The IdP sends a `<LogoutRequest>` to SP-B and SP-C in turn (front-channel via the user's browser, or back-channel via SOAP).
6. SP-B and SP-C destroy their sessions and respond with `<LogoutResponse>`.
7. The IdP destroys its session.
8. The IdP sends a final `<LogoutResponse>` back to SP-A.
9. SP-A redirects the user to a "logged out" page.

### Why it usually breaks

- One SP is down — the IdP times out and the chain stops.
- One SP's session ID doesn't match what the IdP thinks it is — that SP gets a request and silently fails it.
- A pop-up blocker eats one of the redirects mid-chain.
- Front-channel SLO requires the browser to be running the whole time; if the user closes the tab in step 5, B and C never log out.
- Back-channel SLO requires the IdP to know SP-B and SP-C's logout endpoints **and** to have valid certs to talk SOAP.

### What people do instead

Most enterprises just shrug and say "session timeout will catch it." They configure the IdP session for, say, 8 hours. They configure each SP for a similar window. When the user walks away from the laptop and comes back tomorrow, every session is dead. Good enough for most threat models.

If you actually care about SLO working, audit the chain. Test each SP individually. Plan for failures.

## SAML Attacks (XSW1-XSW8)

This section is the reason every senior security engineer has a story about SAML. It is the most famous attack family on the protocol.

### The setup

XML Signature Wrapping (XSW) is a class of attacks where the attacker takes a **legitimately signed** SAML response and **rearranges the XML** so that:

- The signature still verifies (the bytes the signature covers are unchanged).
- But the SP, when it later parses the XML to extract the user's identity, **looks at a different part of the XML** that the attacker controls.

If the SP's "verify" step and "use" step look at different XML, the attacker can effectively impersonate any user. They sign in as a low-privilege account, capture the response, modify the XML to insert a high-privilege account in a place the SP will read but the signature does not cover, and the SP says "yep, signature is valid, let me log you in as the admin."

### XSW1 — Wrap the original Response inside a forged Response

The legitimate signed Response is moved to be a child of a **new** outer Response. The new outer Response carries an attacker-chosen Assertion. The signature still references the **original** Response (now nested), and verification finds that signed object and reports success. The SP's identity-extraction code then walks the document and reads the **outer** Assertion.

```
   BEFORE (legitimate):
   Response (signed)
     Signature ---references---> Response
     Assertion(jane, role=user)

   AFTER (attacker-modified):
   Response   <-- outer, NOT signed, NOT referenced
     Signature ---references---> (still points to nested Response)
     Assertion(admin, role=admin)   <-- attacker's payload
     Response   <-- nested, the original signed object
       Assertion(jane, role=user)
```

If the SP's verifier finds the inner signature, validates it, and reports success, but the SP's user lookup code reads the first/outermost Assertion, the attacker becomes admin.

### XSW2-XSW8 — Variations on the theme

Each variant changes where the wrapped element goes and how the IDs and references line up. The full taxonomy from Somorovsky et al. (USENIX Security 2012, "On Breaking SAML: Be Whoever You Want to Be"):

- **XSW1**: Inject an Assertion as a sibling of the signed Response; place the originally-signed content as a sibling of the Signature. Many parsers pick up the injected Assertion when traversing children.
- **XSW2**: Same idea, with the injected Assertion placed differently relative to the Signature element.
- **XSW3**: Inject an Assertion at the top of the document; the legitimate signed Assertion is moved into a child of the injected one.
- **XSW4**: Like XSW3 but the legitimate Assertion is wrapped inside a different child structure.
- **XSW5**: The attacker's Assertion has the same `ID` as the signed Assertion; reference resolution can become ambiguous.
- **XSW6, XSW7, XSW8**: Various combinations of wrapping the assertion vs the response, with `Object` elements and `Extensions` elements abused as carrier slots.

### Mitigations

The single rule that defeats every XSW variant is:

> **Parse the XML once. Verify the signature. Then use ONLY the exact byte range that the signature covered. Do not re-parse, do not re-traverse, do not query by ID a second time.**

Concretely:

1. After signature verification, locate the element the signature actually referenced (by URI fragment) and **bind your application to that exact node**.
2. Reject any document with **more than one** Assertion.
3. Reject any document with duplicate `ID` attributes.
4. Reject any signature that references something outside the signed subtree's scope.
5. Use a vetted library (`OpenSAML`, `xmlsec1`, `python3-saml` with `defusedxml`, modern Java OpenSAML 4.x) that has been audited for XSW.
6. Validate the schema after canonicalization, before signature verification.
7. Disable XML external entity (XXE) processing in the parser. Use `defusedxml` in Python.

### ASCII diagram of an XSW1 attack

```
LEGITIMATE (what the IdP issued):

  <Response ID="_R1">
    <Signature>
      <Reference URI="#_R1">
        <DigestValue>HASH_OF_RESPONSE_1</DigestValue>
      </Reference>
      <SignatureValue>VALID_SIG</SignatureValue>
    </Signature>
    <Assertion>
      <Subject><NameID>jane@example.com</NameID></Subject>
      <Attribute Name="role"><Value>user</Value></Attribute>
    </Assertion>
  </Response>

ATTACK (what the attacker submits to the SP):

  <Response ID="_R_FAKE">                   <-- NEW outer wrapper
    <Assertion>                              <-- attacker payload
      <Subject><NameID>admin@example.com</NameID></Subject>
      <Attribute Name="role"><Value>admin</Value></Attribute>
    </Assertion>
    <Response ID="_R1">                      <-- original, still signed
      <Signature>
        <Reference URI="#_R1">
          <DigestValue>HASH_OF_RESPONSE_1</DigestValue>
        </Reference>
        <SignatureValue>VALID_SIG</SignatureValue>
      </Signature>
      <Assertion>
        <Subject><NameID>jane@example.com</NameID></Subject>
        <Attribute Name="role"><Value>user</Value></Attribute>
      </Assertion>
    </Response>
  </Response>

WHAT A NAIVE SP DOES:

  1. find Signature element     -> finds it
  2. resolve URI #_R1           -> finds the inner Response
  3. canonicalize and hash      -> matches HASH_OF_RESPONSE_1
  4. verify SignatureValue      -> OK, return SUCCESS
  5. find first Assertion       -> finds the OUTER Assertion (admin!)
  6. extract NameID             -> admin@example.com
  7. extract role attribute     -> admin

  Attacker is now logged in as admin@example.com.
```

The fix: in step 5, the SP must look only inside the element that step 3-4 actually verified.

## Library Bugs

The XSW papers from 2012 lit a fire under the SAML world. Then we found out every major library had bugs of its own. A non-exhaustive hall of fame:

- **ruby-saml CVE-2017-11428** — XML namespace injection. The library used different namespace handling between signature verification and identity extraction, allowing an attacker to substitute a chosen NameID. Affected GitLab, Zendesk, OmniAuth, etc.
- **ruby-saml CVE-2024-45409** — DOCTYPE handling allowed signature verification to be bypassed. Patched September 2024.
- **python-saml / python3-saml** — multiple historical CVEs in similar areas (signature verification and parser handling).
- **Spring Security SAML / Spring SAML Extension** — various CVEs over the years; some 2018-2020 bugs were in encrypted assertion handling.
- **.NET WIF / Microsoft.IdentityModel** — multiple CVEs in token validation, including signature wrapping variants and replay-prevention errors.
- **Microsoft AD FS** — past issues with replay caches and metadata trust.
- **SimpleSAMLphp** — multiple CVEs affecting both IdP and SP modes; mostly fixed promptly.
- **DocuSign, Salesforce, Okta, OneLogin** — every SaaS vendor with a SAML integration has had at least one disclosed bug.
- **OpenSAML** — Shibboleth project's library, the gold standard, has had a small number of bugs over a much longer time horizon.

### What this means for you

Use a maintained library. **Never roll your own SAML.** Read the changelog. Subscribe to the security advisories. When a CVE drops, patch within hours, not weeks — public exploits appear quickly because every SaaS vendor and every Fortune 500 has the same code.

If you absolutely must touch SAML internals, the rule is: **let the library do signature verification, and never hand-roll XML traversal afterward.** Use the library's "give me the validated assertion as a typed object" API. Do not string-match into the raw XML.

## Common Errors

These are the errors you will read in logs while debugging SAML. We list the literal text the SP usually emits, the cause, and the fix.

### `Signature validation failed`

**Cause:** the SP cannot verify the signature.

**Why:**
- Wrong certificate — the SP has an old/rotated cert from the IdP. Pull the latest IdP metadata and re-import.
- Wrong canonicalization — IdP and SP disagree on c14n method. Usually a library version bug.
- Modified XML — the bytes were tampered with in transit (or by a proxy that re-pretty-printed XML).
- Clock skew (sometimes shows up as signature failure if the cert's NotBefore is in the future).

**Fix:** re-fetch IdP metadata, verify cert thumbprint matches, ensure no proxy is re-formatting XML, check `xmlsec1 --verify`.

### `Audience restriction validation failed` / `Audience mismatch`

**Cause:** the assertion's `<Audience>` element does not list the SP's entityID.

**Why:** the IdP is configured with a different SP entityID than what the SP is using.

**Fix:** in the IdP, set the audience for this SP to match exactly what the SP advertises in its metadata `entityID` attribute. Watch for trailing slashes and `http` vs `https`.

### `SubjectConfirmation failed` / `Recipient mismatch` / `InResponseTo missing or unexpected`

**Cause:** the `<SubjectConfirmationData>` block's `Recipient` attribute does not match the SP's ACS URL, or the `InResponseTo` does not match an outstanding AuthnRequest.

**Why:**
- IdP has the wrong ACS URL configured.
- SP isn't tracking outstanding AuthnRequest IDs.
- IdP-init flow but the SP requires `InResponseTo` (which can never be present in IdP-init).

**Fix:** confirm `Recipient` exactly equals the SP's ACS URL; confirm SP's outstanding-request store is working; if doing IdP-init, allow responses without `InResponseTo`.

### `NameID format unspecified` / `Cannot map subject to user`

**Cause:** the IdP didn't include the NameID format, or the format doesn't match what the SP expects.

**Fix:** explicitly set `<NameIDPolicy Format="..."/>` on the AuthnRequest, configure the IdP to release NameID in the agreed format, confirm the SP's mapping logic.

### `Required attribute X missing`

**Cause:** the SP wants an attribute (e.g. `email`, `groups`) that the IdP didn't release.

**Fix:** add a claims rule / attribute release policy at the IdP for this SP that includes the missing attribute. Confirm the attribute name matches **exactly** what the SP expects (case-sensitive, full URI in some setups).

### `Replayed assertion` / `Assertion ID already used`

**Cause:** the same assertion `ID` arrived twice within the SP's replay window.

**Why:**
- A user double-clicked the auto-submit form.
- A debugger replayed an old captured response.
- A real attacker replayed a stolen assertion.

**Fix:** the SP **should** reject the second use; that is correct behavior. If a user complains they keep getting this error on legitimate logins, look for double-submit bugs in the auto-submit JavaScript.

### `Clock skew too large` / `NotBefore in the future` / `NotOnOrAfter has passed`

**Cause:** the SP's clock and the IdP's clock disagree by more than the SP's tolerance window (often 5 minutes).

**Fix:** run NTP. Both sides. Always. If the assertion's lifetime is shorter than network round-trip latency in some weird deployment, ask the IdP to widen the window slightly, but **don't** widen the SP's tolerance to hours. Five minutes is plenty.

### `Invalid Destination`

**Cause:** the AuthnRequest's `Destination` attribute doesn't match the IdP's SSO URL, or the Response's `Destination` doesn't match the SP's ACS URL.

**Fix:** sync the URL on both sides. Watch for `http` vs `https` and trailing slashes.

### `XML parse error` / `Schema validation failed`

**Cause:** the XML is malformed or missing required elements.

**Fix:** capture the raw response (decoded base64) and run it through `xmllint --noout --schema saml-schema-protocol-2.0.xsd`. The error will be specific.

### `Unable to decrypt EncryptedAssertion`

**Cause:** SP doesn't have the private key matching the certificate the IdP used to encrypt.

**Fix:** rotate the SP's encryption keypair and re-publish metadata so the IdP picks up the new public key.

## Hands-On

This section contains real commands you can run. Some need fixtures (a real SAML response, a real cert). Where you need a fixture, we'll show you how to capture or fake one. **Do every command in this list.** That is how the SAML XML stops looking like wallpaper and starts looking like a structure you can read at a glance.

```bash
# 1) Decode a base64'd SAML POST body and pretty-print it.
$ SAML_RESPONSE_BASE64='PHNhbWxwOlJlc3BvbnNlIHht...'
$ echo "$SAML_RESPONSE_BASE64" | base64 -d | xmllint --format -

# 2) Decode a base64'd, deflated SAML query-string param (HTTP-Redirect binding).
#    The HTTP-Redirect binding compresses with raw deflate before base64.
$ SAML_REQ='fVLLT...'   # value of ?SAMLRequest=
$ python3 -c "import sys, base64, zlib; \
    print(zlib.decompress(base64.b64decode(sys.argv[1]), -15).decode())" \
    "$SAML_REQ" | xmllint --format -

# 3) Extract the NameID from a SAML response on stdin.
$ cat saml-response.xml | xmllint --xpath \
    "//*[local-name()='Subject']/*[local-name()='NameID']/text()" -

# 4) Extract every attribute name and value from an assertion.
$ xmllint --xpath \
    "//*[local-name()='Attribute']" saml-response.xml

# 5) Find the entityID of a metadata file.
$ xmllint --xpath \
    "string(//*[local-name()='EntityDescriptor']/@entityID)" \
    idp-metadata.xml

# 6) Find every SingleSignOnService endpoint in an IdP metadata file.
$ xmllint --xpath \
    "//*[local-name()='SingleSignOnService']/@Location" \
    idp-metadata.xml

# 7) Pull the X.509 cert out of metadata and write it to a PEM file.
$ xmllint --xpath \
    "string(//*[local-name()='X509Certificate'])" \
    idp-metadata.xml | \
  awk 'BEGIN{print "-----BEGIN CERTIFICATE-----"} \
       {gsub(/[ \t\n\r]/,""); for(i=1;i<=length($0);i+=64) \
        print substr($0,i,64)} \
       END{print "-----END CERTIFICATE-----"}' > idp-cert.pem

# 8) Inspect the certificate you just extracted.
$ openssl x509 -in idp-cert.pem -text -noout

# 9) Just the cert subject and dates.
$ openssl x509 -in idp-cert.pem -noout -subject -issuer -dates

# 10) Verify a signed SAML response with xmlsec1.
#     (Requires xmlsec1 — `apt-get install xmlsec1` or `brew install xmlsec1`.)
$ xmlsec1 --verify --pubkey-cert-pem idp-cert.pem signed-response.xml

# 11) Verify and only print success/failure (suitable for scripting).
$ xmlsec1 --verify --pubkey-cert-pem idp-cert.pem \
    --enabled-key-data x509 signed-response.xml >/dev/null 2>&1 \
    && echo OK || echo FAIL

# 12) Sign a SAML AuthnRequest as if you were the SP.
$ xmlsec1 --sign --privkey-pem sp-private.pem \
    --output signed-authnrequest.xml unsigned-authnrequest.xml

# 13) Fetch IdP metadata over HTTPS.
$ curl -fsSL -o idp-metadata.xml \
    https://idp.example.com/saml/metadata

# 14) Fetch an SP's metadata.
$ curl -fsSL https://app.example.com/saml/metadata | xmllint --format -

# 15) Validate an XML document against a schema.
$ xmllint --noout --schema saml-schema-protocol-2.0.xsd response.xml

# 16) Pretty-print a metadata file with indentation.
$ xmllint --format idp-metadata.xml > idp-metadata.pretty.xml

# 17) Generate a fresh keypair for SP signing.
$ openssl req -newkey rsa:2048 -nodes -keyout sp-sign.key \
    -x509 -days 365 -out sp-sign.crt \
    -subj "/CN=app.example.com SAML SP signing"

# 18) Generate a fresh keypair for SP encryption.
$ openssl req -newkey rsa:2048 -nodes -keyout sp-enc.key \
    -x509 -days 365 -out sp-enc.crt \
    -subj "/CN=app.example.com SAML SP encryption"

# 19) Show the SHA-256 fingerprint of a cert (paste into IdP UI for trust pinning).
$ openssl x509 -in sp-sign.crt -noout -fingerprint -sha256

# 20) Java keystore: import the IdP cert into a truststore for a Java SP.
$ keytool -import -trustcacerts -file idp-cert.pem \
    -alias idp-saml -keystore truststore.jks \
    -storepass changeit -noprompt

# 21) Java keystore: list trusted certs.
$ keytool -list -keystore truststore.jks -storepass changeit

# 22) Capture a SAML POST in mitmproxy (interactive).
$ mitmproxy --listen-port 8080 -p 8080 \
    --set save_stream_file=saml-traffic.flows
#  then in mitmproxy: save the flow body via 'b' to a file, then base64 -d.

# 23) Capture HTTPS traffic with tcpdump (packets are encrypted, but useful for
#     timing/connection diagnosis).
$ sudo tcpdump -i any -n 'tcp port 443 and host idp.example.com'

# 24) Curl an SP's ACS endpoint with a captured (or test) response and watch the redirect.
$ curl -i -X POST \
    -d "SAMLResponse=$(cat response-b64.txt)" \
    -d "RelayState=/dashboard" \
    https://app.example.com/saml/acs

# 25) Decode and inspect with python3-saml's xmlsec helpers.
$ python3 - <<'PY'
import base64, sys
from lxml import etree
b64 = open('response.b64').read().strip()
xml = base64.b64decode(b64)
tree = etree.fromstring(xml)
print(etree.tostring(tree, pretty_print=True).decode())
PY

# 26) Get all attribute names from an assertion in one line of Python.
$ python3 - <<'PY'
from lxml import etree
NS = {'saml':'urn:oasis:names:tc:SAML:2.0:assertion'}
t = etree.parse('assertion.xml')
for a in t.findall('.//saml:Attribute', NS):
    name = a.get('Name')
    vals = [v.text for v in a.findall('saml:AttributeValue', NS)]
    print(f'{name}: {vals}')
PY

# 27) Generate an SP metadata file with python3-saml.
$ python3 - <<'PY'
from onelogin.saml2.settings import OneLogin_Saml2_Settings
settings = OneLogin_Saml2_Settings(custom_base_path='./settings', sp_validation_only=True)
print(settings.get_sp_metadata().decode())
PY

# 28) saml-tool (Node.js CLI) — pretty-print a SAML response.
$ npx saml-tool decode --base64 "$SAML_RESPONSE_BASE64"

# 29) saml-tool — verify a signature.
$ npx saml-tool verify --xml signed-response.xml \
    --cert idp-cert.pem

# 30) Use Keycloak's admin CLI to fetch a client's SAML descriptor.
$ kcadm.sh get clients/<client-uuid>/installation/providers/saml-idp-descriptor \
    -r my-realm

# 31) Okta CLI — list SAML apps.
$ okta apps list --type SAML

# 32) Auth0 CLI — list SAML applications.
$ auth0 apps list --reveal-secrets | grep -i saml

# 33) Test that a metadata URL is reachable and returns valid XML.
$ curl -fsSL https://idp.example.com/saml/metadata | \
    xmllint --noout - && echo "metadata OK"

# 34) Diff two metadata files (e.g. before/after a cert rotation).
$ diff <(xmllint --format old-metadata.xml) \
       <(xmllint --format new-metadata.xml) | less

# 35) Quickly check what NameID formats an IdP advertises.
$ xmllint --xpath \
    "//*[local-name()='NameIDFormat']/text()" idp-metadata.xml

# 36) See whether the SP requires signed AuthnRequests.
$ xmllint --xpath \
    "string(//*[local-name()='SPSSODescriptor']/@AuthnRequestsSigned)" \
    sp-metadata.xml

# 37) See whether the SP wants encrypted assertions.
$ xmllint --xpath \
    "string(//*[local-name()='SPSSODescriptor']/@WantAssertionsSigned)" \
    sp-metadata.xml

# 38) Print the full Conditions block of an assertion (audience, time window).
$ xmllint --xpath \
    "//*[local-name()='Conditions']" assertion.xml

# 39) Print the AuthnContext (Password, MFA, etc.).
$ xmllint --xpath \
    "//*[local-name()='AuthnContextClassRef']/text()" assertion.xml

# 40) Use openssl to confirm a private key matches a cert (modulus check).
$ openssl rsa -in sp-sign.key -modulus -noout | openssl md5
$ openssl x509 -in sp-sign.crt -modulus -noout | openssl md5
#  the two md5s must match — if they don't, the keypair is broken.

# 41) Convert a PFX/PKCS12 (Windows-style) cert+key bundle to PEM.
$ openssl pkcs12 -in saml.pfx -out saml.pem -nodes
```

That's 41 commands. Run them. Burn through them. By the end you will read SAML XML the way you read English.

## Common Confusions

These are the mistakes everybody makes the first month they deal with SAML. Memorize the difference.

### SP vs IdP

The SP is the application. The IdP is the login page. **The IdP is the trusted authority; the SP relies on it.** If somebody says "log in to the IdP," they mean go to Okta/Entra/Keycloak, not to your app.

### entityID vs ACS URL

`entityID` is a **name** for the SP. It is unique. It is often a URL but does not have to resolve to anything real. The ACS URL is **where the SAML Response is delivered**. Two different things. The entityID lives in the assertion's `<Audience>`. The ACS URL lives in the message's `Destination` and the SubjectConfirmationData's `Recipient`.

### Response vs Assertion

The Response is the **outer envelope**. The Assertion is the **payload inside it**. Either or both can be signed. Either can be encrypted (well, only the Assertion can be encrypted in standard usage). Saying "verify the SAML signature" is ambiguous — verify the Response signature, the Assertion signature, or both.

### NameID vs email attribute

The NameID is the **primary identifier** of the user. The email attribute is **additional metadata**. They might happen to have the same value (when the NameID format is `emailAddress`). They might be different (NameID is a persistent opaque ID, email is `jane@example.com`). The SP must **decide which one to use as the join key** to its local user table.

### HTTP-POST vs HTTP-Redirect binding

HTTP-POST: form, large payload OK, used for both directions.
HTTP-Redirect: query string, small payload only, deflated and signed in the URL, mostly used for AuthnRequests.
**Don't** put a SAML Response on HTTP-Redirect. It will be too big for browser URL limits and the deflate-then-sign-then-base64 is brittle.

### SAML 1.1 vs SAML 2.0

Different XML namespaces, different element names, **not interoperable**. Ninety-nine percent of the time you only care about 2.0. If a vendor offers you "SAML 1.1 SSO" in 2026, ask why and assume the answer is "we haven't touched our auth code in fifteen years."

### Signed vs encrypted

Signed: integrity. The receiver knows the message wasn't modified and the IdP issued it.
Encrypted: confidentiality. The receiver is the only one who can read the contents.
You can have either, both, or neither. **Signed is mandatory in practice.** Encrypted is optional but good.

### IdP-init vs SP-init

SP-init: user starts at the app, gets bounced to the IdP, comes back. The SP issued an AuthnRequest, so it can match `InResponseTo`.
IdP-init: user starts at the IdP portal, clicks an app, lands at the SP with an unsolicited Response. The SP **never sent a request**, so there is no `InResponseTo`. Allows convenience, weakens replay protection.

### XSW1 vs XSW2 vs ... XSW8

All in the XML Signature Wrapping family. They differ in **where** the attacker injects the forged element. The fix is the same for all of them: **bind to the verified subtree, do not re-traverse.** You do not need to memorize the exact taxonomy unless you are writing a SAML library.

### SLO vs session timeout

Single Logout actively notifies every SP that the user has logged out. Session timeout is a passive expiration — the session simply ages out. SLO is fragile in practice; session timeout is reliable. Most enterprises rely on session timeout and treat SLO as best-effort.

### Metadata trust vs cert pinning

Metadata trust: I trust whatever cert is in the metadata file (or URL) for this entityID.
Cert pinning: I have the cert on disk, regardless of what metadata says.
Pinning is more secure but breaks every time the IdP rotates. Most enterprises trust the metadata URL over HTTPS and re-fetch on a schedule.

### `Audience` vs `Destination`

Both look like SP URLs. Different jobs.
`Destination` (on the Response): "this message is being **delivered** to this URL." Must equal the SP's ACS.
`Audience` (in the assertion's `<AudienceRestriction>`): "this assertion is **intended for** this party." Must equal the SP's entityID.

### Assertion ID vs Response ID

Both are random strings used as anchors. The Assertion ID is what an Assertion-level signature references; the Response ID is what a Response-level signature references. They are different elements. Replay protection should track **Assertion IDs** (because the assertion is the thing whose reuse matters).

### IdP cert rotation vs SP cert rotation

When the IdP rotates its signing cert, the SP needs to load the new cert before the old one expires. When the SP rotates its encryption cert, the IdP needs to load the new cert. Both sides usually publish two certs in metadata during rotation windows.

### `WantAuthnRequestsSigned` vs `AuthnRequestsSigned`

Both look like booleans about signing AuthnRequests. They live on **opposite sides**.
`WantAuthnRequestsSigned` lives on the **IdPSSODescriptor** — "I, the IdP, want SPs to sign their AuthnRequests."
`AuthnRequestsSigned` lives on the **SPSSODescriptor** — "I, the SP, sign my AuthnRequests."
If only one side asserts it, the other side may still ignore the requirement. Confirm both.

### `WantAssertionsSigned` vs response-level signing

`WantAssertionsSigned` on the SPSSODescriptor means "I want the **Assertion** itself signed, not just the outer Response." Many SPs accept either; some accept only one or the other. Always try to sign **both** if your IdP supports it.

### SAML SSO vs WS-Federation

Both are XML-based SSO protocols, often deployed by the same IdPs (especially AD FS). They are **not** the same. WS-Federation has different message formats and different bindings. If a vendor offers "WS-Fed SSO" they mean a different protocol; do not pass them SAML metadata.

### Custom NameID format URN vs the standard ones

Some vendors define their own NameID format URNs. As long as both sides agree, that's fine. **Do not** invent your own without coordinating with the other party — they will reject the assertion.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **SAML** | Security Assertion Markup Language. The protocol for federated login using signed XML tickets. |
| **SAML 2.0** | The current version (March 2005). What you actually use. |
| **SAML 1.1** | The legacy version (2003). Different namespaces, not wire-compatible with 2.0. |
| **SP** | Service Provider. The app the user wants to log in to. |
| **IdP** | Identity Provider. The login authority. The "box office." |
| **Identity Provider** | Same as IdP. |
| **Service Provider** | Same as SP. |
| **AuthnRequest** | The SP's "please authenticate this user" message to the IdP. |
| **AuthnResponse** | Often a sloppy synonym for "Response." |
| **Response** | The outer SAML message that wraps an Assertion and carries status. |
| **Assertion** | The signed inner payload that says "I claim this user is X with these attributes." |
| **Subject** | The element in the assertion that identifies who the assertion is about. |
| **NameID** | The string identifier of the subject (email, persistent ID, etc.). |
| **NameID format: emailAddress** | NameID is interpreted as an email. Most common. |
| **NameID format: persistent** | Opaque pairwise ID, different per SP. Privacy-preserving. |
| **NameID format: transient** | One-time random ID. New each session. |
| **NameID format: unspecified** | Format is whatever the parties agreed offline. |
| **NameID format: X509SubjectName** | NameID is an X.509 DN. |
| **NameID format: kerberos** | NameID is a Kerberos principal. |
| **AttributeStatement** | Block in the assertion holding extra user attributes. |
| **AttributeValue** | One value for one attribute (multi-valued attrs allowed). |
| **AuthnContext** | Description of how the user authenticated (password, MFA, smartcard...). |
| **AuthnContextClassRef** | The URI naming the auth context class. |
| **Password (AuthnContext)** | "User typed a password." |
| **PasswordProtectedTransport** | "Password over TLS." |
| **MFA (AuthnContext)** | Multi-factor login. |
| **Smartcard (AuthnContext)** | Smartcard-based login. |
| **X509 (AuthnContext)** | Client X.509 cert. |
| **SubjectConfirmation** | "How you can prove you are the subject of this assertion." |
| **SubjectConfirmationData** | The fields backing that — Recipient, NotOnOrAfter, InResponseTo. |
| **Conditions** | Block constraining when and where the assertion is valid. |
| **AudienceRestriction** | "This assertion is only for these audiences (entityIDs)." |
| **OneTimeUse** | Marks the assertion as single-use; SPs should reject replays. |
| **ProxyRestriction** | Limits how many proxies in a chain can re-issue derived assertions. |
| **NotBefore** | Earliest time the assertion is valid. |
| **NotOnOrAfter** | Latest time (exclusive) the assertion is valid. |
| **Issuer** | The entityID of the party that issued the message (the IdP for a Response). |
| **Audience** | The intended recipient entityID. |
| **InResponseTo** | The Response's reference back to the AuthnRequest's ID. |
| **RelayState** | Opaque value the SP attaches; the IdP echoes it back. Used for "where was the user going?" |
| **ID** | Unique XML attribute used by signatures and replay caches. |
| **IssueInstant** | Timestamp when the message was created. |
| **Destination** | Endpoint URL the message is being delivered to. |
| **Recipient** | The ACS URL inside SubjectConfirmationData. |
| **ProtocolBinding** | Which binding the response should use (HTTP-POST, etc.). |
| **AssertionConsumerServiceURL** | The SP endpoint that receives Responses. ACS. |
| **ACS** | Short for AssertionConsumerService. Where Responses get POSTed. |
| **entityID** | The unique name of an SP or IdP. Usually looks like a URL. |
| **Metadata** | XML describing one party's endpoints, certs, supported bindings. |
| **EntityDescriptor** | The root element of a metadata file. |
| **IDPSSODescriptor** | Metadata block describing IdP-side endpoints. |
| **SPSSODescriptor** | Metadata block describing SP-side endpoints. |
| **SingleSignOnService** | The IdP endpoint that accepts AuthnRequests. |
| **SingleLogoutService** | Endpoint for SLO messages. |
| **ArtifactResolutionService** | SOAP endpoint used for HTTP-Artifact binding. |
| **KeyDescriptor** | Metadata block holding a public cert (signing/encryption use). |
| **HTTP-POST binding** | Auto-submit form. Most common. |
| **HTTP-Redirect binding** | Deflated, signed query string. Small payloads only. |
| **HTTP-Artifact binding** | Tiny artifact ID via redirect, full assertion via SOAP. |
| **SOAP binding** | Backend SOAP for SLO and Artifact resolution. |
| **signature** | Cryptographic proof of integrity and origin. |
| **XML Signature** | The W3C standard for signing XML. SAML uses it. |
| **XMLDSig** | Common abbreviation for XML Signature. |
| **c14n** | Canonicalization. The rule book for converting XML to a canonical byte form. |
| **exclusive c14n** | The variant SAML uses. Drops unused ancestor namespaces. |
| **Canonical XML** | The W3C c14n spec (1.0 and 1.1). |
| **RSA-SHA256** | The signature algorithm in modern SAML. |
| **ECDSA-SHA256** | An ECC alternative. Less common in SAML. |
| **EncryptedAssertion** | An assertion encrypted with the SP's public key. |
| **EncryptedID** | An encrypted NameID. |
| **encryption** | Confidentiality protection so only the receiver can read. |
| **AES-128-CBC** | Symmetric cipher historically used for assertion encryption. |
| **AES-256-GCM** | Modern AEAD cipher; preferred. |
| **RSA-OAEP** | The padding mode used to wrap the symmetric key. |
| **federation** | A trust relationship between IdPs and SPs, established by exchanging metadata. |
| **federation metadata** | Aggregate metadata file for a federation (e.g. InCommon's). |
| **IDP metadata** | The metadata XML for one IdP. |
| **SP metadata** | The metadata XML for one SP. |
| **eduGAIN** | Inter-federation service connecting research/education federations globally. |
| **InCommon** | The US higher-education identity federation. |
| **Shibboleth** | A reference IdP/SP implementation popular in academia. |
| **OneLogin** | A commercial IdP vendor. |
| **Okta** | A widely used commercial IdP. |
| **Auth0** | Commercial IdP, owned by Okta. |
| **Keycloak** | Open-source IdP from Red Hat. |
| **AD FS** | Active Directory Federation Services — Microsoft's on-prem IdP. |
| **ADFS** | Common spelling of AD FS. |
| **Azure AD / Entra ID** | Microsoft's cloud IdP, formerly Azure Active Directory. |
| **SimpleSAMLphp** | Popular PHP SAML implementation. |
| **mod_auth_mellon** | Apache module for SAML SP behavior. |
| **mod_auth_saml** | Older Apache module for SAML. |
| **ITfoxtec.Identity.Saml2** | A .NET SAML library. |
| **OpenSAML** | Shibboleth's reference Java library. |
| **ruby-saml** | Ruby gem implementing SAML, widely used. |
| **python-saml** | Older Python SAML library (OneLogin). |
| **python3-saml** | Python 3 successor. |
| **xmlsec** | A C library for XML signature/encryption. |
| **xmlsec1** | The CLI tool from xmlsec. |
| **lxml** | Python XML library that wraps libxml2. |
| **defusedxml** | Python library hardening lxml/xml.etree against XXE and entity attacks. |
| **XSW** | XML Signature Wrapping. The attack family. |
| **XSW1-XSW8** | The eight canonical variants from Somorovsky et al. (2012). |
| **signature wrapping** | Generic name for the XSW family. |
| **replay attack** | Re-submitting a captured assertion to log in again. |
| **audience confusion** | Submitting an assertion intended for SP-A to SP-B. |
| **NameID round-tripping** | Pattern of using the same NameID across federations; can leak identity. |
| **Single Logout** | The "log me out everywhere" feature. |
| **SLO** | Short for Single Logout. |
| **LogoutRequest** | The SLO version of an AuthnRequest. |
| **LogoutResponse** | The SLO version of a Response. |
| **Just-In-Time provisioning** | The SP creates the local user account on first SAML login. |
| **JIT** | Short for Just-In-Time provisioning. |
| **SAML profiles** | Defined combinations of bindings + flows. |
| **Web SSO profile** | The SP-init or IdP-init flow we walked through. |
| **ECP profile** | Enhanced Client/Proxy. SOAP-based, no browser. Used by non-browser clients. |
| **HoK** | Holder-of-Key SubjectConfirmation. The user must present a key (e.g., a TLS client cert) to prove they are the subject. |

## Try This

Pick at least three. Working through these is how SAML moves from "alphabet soup" to "yeah, I get it."

### 1. Decode a real assertion

If your company uses SAML, install a browser extension like **SAML Tracer** (Firefox) or **SAML-tracer** (Chrome). Log in to one of your SaaS apps. Watch the trace. Find the `SAMLResponse` POST. Copy the base64. Paste it into your terminal and decode it with `base64 -d | xmllint --format -`. Read the assertion. Find your NameID. Find the audience. Find the time window. Compute how long the assertion was valid for.

### 2. Generate a self-signed SP keypair

Run command 17 from the Hands-On section. Look at the cert. Run command 19 and read the SHA-256 fingerprint. You just made a real production-style SAML SP cert.

### 3. Diff IdP metadata before and after a cert rotation

Save the current IdP metadata file. Wait for or trigger a cert rotation (in a test IdP). Save the new metadata. Run command 34. Look at the diff. Confirm the new `<X509Certificate>` value differs and the `<KeyDescriptor>` may include both old and new certs during the overlap period.

### 4. Build a fake XSW1 attack payload

Take a known-valid signed Response (in a test environment, never against production). Wrap it inside a forged outer Response with a different NameID. Submit it to a deliberately-vulnerable test SP. Watch it succeed (or fail, if the SP's library is patched). Then patch the SP and watch the attack fail. This is the fastest way to internalize why XSW matters.

### 5. Read your IdP's claims rules

Find the attribute release / claims rules for one SaaS app at your IdP. Look at the literal mapping — `mail` → `email`, `memberOf` → `groups`, etc. Decide whether each claim is truly necessary; remove the ones that aren't.

### 6. Validate a metadata file against the schema

Download the official `saml-schema-metadata-2.0.xsd` from the OASIS site. Run command 15-style validation against your IdP's metadata. Watch for warnings. Most production metadata is technically schema-noncompliant in subtle ways (extra extension blocks, slightly off attribute orders). It still works, but you'll see why some validators throw errors.

### 7. Set up a Keycloak test IdP locally

Run `docker run -p 8080:8080 -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin quay.io/keycloak/keycloak:latest start-dev`. Log in to the admin console. Add a SAML client. Download the IdP metadata. Use it to set up a test SP. You now have a controlled SAML lab on your laptop.

### 8. Write a Python script that prints everything in an assertion

Use commands 25 and 26 as a starting point. Extend the script to print the NameID, every attribute, the AuthnContext, the issuer, the audience, NotBefore, NotOnOrAfter, and the SubjectConfirmation Recipient. Run it on a real assertion. You will not need to run `xmllint` for this purpose ever again.

### 9. Capture and decode a SAML Redirect-binding AuthnRequest

Pick a SaaS app that uses HTTP-Redirect binding for AuthnRequests (most do for the SP→IdP direction). Trigger a login. In SAML Tracer, find the GET to the IdP. Note the `SAMLRequest` query param. Decode it (base64-decode, then raw-deflate-decompress). See the AuthnRequest XML. Notice how much smaller it is than the Response.

### 10. Audit a metadata file for missing encryption support

Check whether your SPs publish encryption KeyDescriptors. Many don't. If they don't, the IdP cannot encrypt assertions to them. Decide whether you should add encryption support — and if so, generate a fresh keypair (command 18) and add it to the metadata.

### 11. Trace the full round-trip with timestamps

Use SAML Tracer to capture a full SP-init flow. Note the wall-clock timestamps at each step: GET to SP, redirect to IdP, POST of credentials, redirect back with Response, redirect to home page. Compare with the assertion's `IssueInstant`, `NotBefore`, and `NotOnOrAfter`. You will find that the assertion's lifetime window is usually only 5-10 minutes — far less than the user's actual session, which is held by an SP cookie set after step 7.

### 12. Read your IdP's SAML log

Most IdPs log every assertion they issue. Find the log. Find your own login. Read the JSON entry. Note which fields are logged and which are not (most IdPs do **not** log the SignatureValue but do log the assertion ID, the audience, the NameID, the timestamps, and any error). Set up an alert on `Signature validation failed` events — that error in the SP's logs is sometimes the only signal that someone is fuzzing your SP.

### 13. Force an assertion replay and watch it fail

In a test environment, capture a valid Response. Re-submit it to the SP a second time. The SP **must** reject it as a replay (most do — by tracking assertion IDs in a short-lived cache). If your SP accepts the replay, that is a finding to report immediately.

### 14. Compare HTTP-POST vs HTTP-Artifact for one SP

If your IdP and SP both support HTTP-Artifact binding, configure both flows side by side. Run a login on each. Note that the HTTP-Artifact flow has the SP making an outbound SOAP call to the IdP — visible in your egress firewall logs. HTTP-POST has no such outbound call. Decide which fits your network posture better.

## Where to Go Next

- `cs security saml` — dense reference for SAML implementation specifics.
- `cs auth saml` — auth-side reference, complementary view.
- `cs auth oauth` — OAuth 2.0, the modern alternative for new applications.
- `cs auth oidc` — OpenID Connect, SAML's spiritual successor for web SSO.
- `cs auth ldap` — the underlying directory protocol most IdPs talk to.
- `cs auth kerberos` — what AD FS and Azure AD secretly speak under the hood.
- `cs security pki` — public-key infrastructure that backs SAML signing/encryption certs.
- `cs security tls` — the transport SAML rides on.
- `cs security cryptography` — the underlying primitives.
- `cs ramp-up tls-eli5` — sibling sheet covering TLS in plain English.
- `cs ramp-up oauth-oidc-eli5` — sibling sheet covering the modern alternative to SAML.
- `cs ramp-up linux-kernel-eli5` — the original ELI5 in this series.

## See Also

- `security/saml`
- `auth/saml`
- `security/oauth`
- `auth/oidc`
- `auth/ldap`
- `auth/kerberos`
- `security/pki`
- `security/tls`
- `security/cryptography`
- `ramp-up/tls-eli5`
- `ramp-up/oauth-oidc-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- OASIS SAML 2.0 Core (`saml-core-2.0-os.pdf`) — the assertion and protocol element schema. <https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf>
- OASIS SAML 2.0 Bindings (`saml-bindings-2.0-os.pdf`) — defines HTTP-POST, HTTP-Redirect, HTTP-Artifact, SOAP. <https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf>
- OASIS SAML 2.0 Profiles (`saml-profiles-2.0-os.pdf`) — Web SSO, ECP, SLO, etc. <https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf>
- OASIS SAML 2.0 Metadata (`saml-metadata-2.0-os.pdf`) — EntityDescriptor and friends. <https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf>
- OASIS SAML 2.0 Conformance — interoperability requirements. <https://docs.oasis-open.org/security/saml/v2.0/saml-conformance-2.0-os.pdf>
- OASIS SAML 2.0 Errata — accumulated bug fixes to the spec. <https://docs.oasis-open.org/security/saml/v2.0/>
- W3C XML Signature Syntax and Processing — <https://www.w3.org/TR/xmldsig-core/>
- W3C XML Encryption Syntax and Processing — <https://www.w3.org/TR/xmlenc-core1/>
- W3C Canonical XML 1.0 — <https://www.w3.org/TR/xml-c14n>
- W3C Canonical XML 1.1 — <https://www.w3.org/TR/xml-c14n11/>
- W3C Exclusive XML Canonicalization — <https://www.w3.org/TR/xml-exc-c14n/>
- "Mastering SAML" community docs — <https://developers.onelogin.com/saml>
- xmlsec.org — the C library and CLI used by half the SAML world. <https://www.aleksey.com/xmlsec/>
- Shibboleth wiki — extensive deployment notes. <https://shibboleth.atlassian.net/wiki/spaces/IDP4/>
- man `xmlsec1`, `xmllint` — local references on any Linux/macOS system.
- OWASP SAML Security Cheat Sheet — <https://cheatsheetseries.owasp.org/cheatsheets/SAML_Security_Cheat_Sheet.html>
- "On Breaking SAML: Be Whoever You Want to Be" — Somorovsky, Mayer, Schwenk, Kampmann, Jensen. USENIX Security 2012. The XSW paper. <https://www.usenix.org/system/files/conference/usenixsecurity12/sec12-final91-8-23-12.pdf>
- "Multiple Bypasses of SAML Conditions Validation in Ruby-SAML" — CVE-2024-45409 advisory. <https://github.com/SAML-Toolkits/ruby-saml/security/advisories/GHSA-jw9c-mfg7-9rx2>
- InCommon Federation operations docs — <https://incommon.org/federation/>
- Shibboleth IdP and SP documentation — <https://shibboleth.atlassian.net/wiki/spaces/SHIB/>
- Microsoft AD FS / Entra ID SAML reference — <https://learn.microsoft.com/en-us/azure/active-directory/develop/single-sign-on-saml-protocol>
- Okta's developer SAML guide — <https://developer.okta.com/docs/concepts/saml/>
- Keycloak SAML SSO guide — <https://www.keycloak.org/docs/latest/server_admin/#_saml>
