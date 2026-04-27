# OAuth 2.0 + OIDC — ELI5 (Valet Keys for the Internet)

> OAuth gives a stranger a special key that opens just one door of your house, and OIDC stapes a "here's who actually owns this house" note onto the key.

## Prerequisites

(none — but `cs ramp-up tls-eli5` helps; OAuth/OIDC ride on TLS)

If you have not done the TLS ramp-up yet, that is fine. You can read this whole sheet without it. The only thing you might want to know up front is that everything OAuth and OIDC do happens **inside an encrypted tunnel** (HTTPS). Without that tunnel, anyone listening on the wire could grab the secrets we are about to talk about and use them to pretend they are you. So the rule of thumb is: every URL in this sheet starts with `https://`. There is no `http://` version. If you ever see plain `http://` for an OAuth endpoint, somebody is doing something wrong.

If a word feels weird, look it up in the **Vocabulary** table at the bottom. Every weird word you see is in that table with a one-line plain-English definition. You will run into a lot of weird three-letter words like JWT, JWS, JWE, JWA, JWK, JWKS, RP, OP, IdP, PKCE, CIBA, JARM, FAPI. Do not panic. They are just labels people stuck on small ideas to talk about them faster. You do not have to memorize them. Read the sheet. Glance at the table when you need it. Eventually the words will stick because you will have used them.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. We call that "output."

## What Even Is OAuth?

### The valet at the fancy restaurant

Picture this. You drive up to a fancy restaurant. Out front there is a guy in a vest. He is the valet. The valet's job is to park your car in the parking lot behind the building. You don't want to walk back there in the rain. You don't want to spend ten minutes searching for a spot. So you let the valet do it for you.

But there is a problem. To park your car, the valet needs to be able to **drive your car.** That means he needs the key. And if you give him your full key ring, he gets:

- the key to your house
- the key to your office
- the key to your gym locker
- the key to your trunk (where you keep your laptop, your tax returns, the gift you bought your spouse, whatever)
- the key to your glove box (where you keep your registration, insurance card, garage opener, maybe a spare hundred-dollar bill)
- the key that starts the car

The valet only needs that last one. He needs to start the car, drive it twenty feet, park it, and turn it off. He does **not** need to break into your house. He does **not** need to open your trunk. He does **not** need to rummage through your glove box.

So the carmaker did something clever. They made a special second key. We call it the **valet key.** The valet key looks just like a normal key, but it can only do a few things:

- It can open the driver's door.
- It can start the engine.
- It cannot open the trunk.
- It cannot open the glove box.
- It only works for thirty minutes (some cars even have this rule).
- It cannot make new copies of itself.

You hand the valet the valet key, not your master key. He parks the car. When you come back, you tell him "give me my car," he hands you the valet key back, and that's it. If he was a bad guy, he could have stolen your car, but he couldn't have stolen the laptop in the trunk. He couldn't have driven over to your house and let himself in. The damage he could do was limited by the limits baked into the valet key.

**OAuth is the valet key model for the internet.**

When some random app says "I need to read your Google calendar so I can show your meetings on a fancy dashboard," you have a choice. The bad way is to type your Google username and password into that random app. Now they have your master key. They could read your email. They could send mail in your name. They could delete every file in your Drive. They could change your password and lock you out. They have the keys to the kingdom.

The good way is OAuth. The random app sends you over to Google. Google checks your password (only Google ever sees it). Google asks you, "Hey, do you actually want to give this app permission to read your calendar?" You say yes. Google hands the random app a valet key. The valet key only opens the calendar. It does not open your email. It does not open your Drive. It expires after an hour. The random app uses it to read your calendar, and that's it.

If the random app turns out to be evil, the worst they can do is read your calendar. They cannot read your mail. They cannot delete your files. They cannot send messages pretending to be you. The valet-key model contained the damage.

That is OAuth in one sentence: **OAuth lets a third-party app do specific things on your behalf without giving them your password.**

### Why is "without giving them your password" so important?

Old software used to do the bad way. You would install some app, and to use it with your Gmail, you would type in your Gmail password. The app would log in to your account directly using that password. This was called **password screen scraping** or **credential delegation.** It was awful for many reasons.

First, you had no idea what the app was actually doing. It had your full master key. It could be reading every email you ever sent.

Second, you could not take the key back. The only way to make the app stop was to change your password. But if you change your password, every device, every other app, every browser, all of them stop working. You have to log in everywhere again. So most people never bothered.

Third, if the app got hacked, the hackers walked away with your real password. Your real password might be the same one you used at your bank. Your real password might be the one you used everywhere. One company's bad security became your nightmare.

Fourth, you could not see who was logged in or revoke just one app. There was no list. The bank either trusted you completely or did not.

OAuth fixes all of this. The app never sees your password. The app only sees a valet key. You can revoke that one key any time you want, in one click, on a "connected apps" page. The app cannot use the key to log in as you everywhere — only on the specific resources Google said the key was good for. If the app gets hacked, the hackers walk away with a thirty-day token, not your real password.

That is the whole reason OAuth exists. **Apps need limited, revocable, scoped access to your stuff. OAuth is how you give it to them.**

### The four actors

Every OAuth dance has the same four actors. We are going to use them over and over for the rest of this sheet, so let's name them clearly. We will use the restaurant analogy and the OAuth jargon side by side.

**1. Resource Owner (you).**

That's you. The human who owns the stuff. In the valet analogy, you are the car owner. In the calendar example, you are the Google user whose calendar belongs to you. Nobody can give the random app permission to read your calendar except you. The OAuth specs call you the **Resource Owner** because you own the resource (the calendar, the photos, the files, whatever).

**2. Client (the third-party app).**

The Client is the app that wants to do something. In our example, it's that random calendar dashboard. In the valet analogy, the Client is the valet himself. Note: "Client" in OAuth-speak is **not the user**. The user is the Resource Owner. The Client is the **app**. A lot of newcomers get this wrong, because in normal English "client" usually means a customer. In OAuth land "Client" means software.

**3. Authorization Server (the IdP, Google in our example).**

The Authorization Server is the place that knows your password and decides who is allowed to do what. In our example, it is Google's login system. We sometimes call this the **Identity Provider** or **IdP** or **OP** (OpenID Provider) when OIDC is involved. In the valet analogy, the Authorization Server is the carmaker — the entity that designed the keys and decides which keys can do what.

The Authorization Server is the only place that ever sees your password. Not the Client. Not the Resource Server. Just the Authorization Server. That is the magic. Your password lives in one place and one place only.

**4. Resource Server (the API holding your data).**

The Resource Server is the actual data store. In our example, it is Google Calendar's API. In the valet analogy, the Resource Server is the parking lot — the place where the actual valet activity happens. The Resource Server does not know your password. The Resource Server only knows how to look at a valet key (an access token) and decide whether the bearer is allowed to do whatever they're trying to do.

Sometimes the Authorization Server and Resource Server are the same company (Google runs both). Sometimes they are different (Auth0 issues tokens, your-startup.com is the Resource Server). The OAuth design lets them be either.

### Walking through "Sign in with Google" at ELI5 level

Picture you are on a website called PomodoroFun.com, a little timer app for getting work done. PomodoroFun has a "Sign in with Google" button on its login page. You click it.

**Step 1.** PomodoroFun has zero information about you. PomodoroFun does not even know your name yet. PomodoroFun says, "Okay, I will redirect this user over to Google. Google can sort it out." Your browser jumps from `https://pomodorofun.com/login` to `https://accounts.google.com/o/oauth2/v2/auth?client_id=...&scope=openid+profile+email&...`. That URL has a bunch of stuff in it. We will pick it apart later. For now, just know your browser landed on Google.

**Step 2.** Google's page sees you. If you are already logged in to Google, it skips ahead. If not, it asks for your password. Important: this happens on `accounts.google.com`. PomodoroFun is not seeing this page. PomodoroFun never sees your password.

**Step 3.** Google says, "Hey, this app called PomodoroFun is asking to know your name and email. Are you okay with that?" You see a consent screen with the PomodoroFun logo and the words "wants access to your name, email address." You click **Allow.**

**Step 4.** Google generates a tiny piece of paper called an **authorization code.** Imagine it like a one-time-use ticket stub. Google sends your browser back to PomodoroFun with that ticket stub: `https://pomodorofun.com/callback?code=abc123&state=xyz...`. Your browser is now back on PomodoroFun.

**Step 5.** PomodoroFun's server takes the ticket stub (the code) and says to Google, in a private back-channel call (server to server, not through your browser), "Hey, here is the ticket stub I just got. Plus my own client secret. Trade me a real key." Google checks the ticket stub is valid, checks PomodoroFun's secret, and trades them an **access token** and an **ID token.**

**Step 6.** PomodoroFun looks at the ID token. The ID token is a sealed letter from Google that says, "This user's name is Jane, their email is jane@example.com, their unique Google user ID is 109283740." PomodoroFun reads the letter, says "okay, you are Jane!" and logs you in.

That is "Sign in with Google." Six steps. The user only saw two: clicking the button, and clicking Allow. Everything else happened behind the scenes.

```
   YOU                  POMODORO              GOOGLE
   |                       |                    |
   |---click "Sign in"---->|                    |
   |                       |---redirect---------|
   |<-----browser jump to Google----------------|
   |---enter password+consent ----------------->|
   |<-----redirect back with code---------------|
   |---deliver code to Pomodoro------>|         |
   |                       |---code+secret---->|
   |                       |<---tokens--------|
   |<-----you are logged in|                    |
```

That dance — that exact dance — is the **Authorization Code Flow.** It is the most common, most secure OAuth flow. We will draw it again with more detail in a moment.

### Why is it called "OAuth 2.0"?

Because there was a 1.0. OAuth 1.0 came out in 2007. It worked. It was painful. It made every Client developer write code that signed every request with HMAC-SHA1 and a complicated nonce-and-timestamp dance. People hated it. It was secure but it was not fun.

OAuth 2.0 (the one we use today) was published as **RFC 6749** in 2012. It threw out the request signing. It said, "Look, just use TLS. The whole protocol is going to ride inside an HTTPS tunnel. The bearer of a valid token is the legitimate user. Done." That made implementations way easier. It also made it possible for browsers and mobile apps to be Clients (1.0 was very server-to-server-y).

There is also a draft "**OAuth 2.1**" which is essentially "OAuth 2.0, minus the bad parts." The bad parts that 2.1 cuts include the implicit grant and the resource owner password credentials grant. We will get to why those are bad later. As of 2026, 2.1 is still a draft, but most modern OAuth servers already follow its rules anyway.

## What Even Is OIDC?

### OAuth is about permissions, OIDC is about identity

Here is the crucial distinction that blows people's minds the first time they hear it:

**OAuth tells the Client what the user is ALLOWED TO DO. OAuth does not tell the Client WHO THE USER IS.**

That sounds wrong, doesn't it? "But I just used Sign in with Google! I'm pretty sure Google told the app who I was!" Right — but it told them as a side effect, using OIDC. Pure OAuth, by itself, does not have a built-in concept of "who the user is."

Think back to the valet key analogy. The valet key lets you start the car. The valet key does **not** tell anyone whose car it is. If somebody hands you a valet key, all you know is "this key starts a car somewhere." You do not know whose car. The carmaker would have to give you a separate piece of paper that says "this key belongs to Jane Doe's 2024 Toyota."

In OAuth, the access token is the valet key. The Resource Server can use it to know "this person is allowed to read calendar X." But the Client itself, when it gets the token, does not necessarily know **who** is the person on the other end of the token. The token is opaque. It is just a string of characters.

For ten years (2007-2014ish), this caused a giant mess. Every site that wanted "Sign in with Twitter" or "Sign in with Facebook" had to do something hacky: get a token, then use that token to call a "/me" or "/profile" endpoint at the API to find out who you were. Every IdP had a different "/me" endpoint with different fields. There was no standard. Identity was bolted on after the fact.

Eventually, the OpenID Foundation said "this is silly, let's make a standard." They invented **OpenID Connect**, usually called **OIDC**. OIDC published its core spec in 2014. It took the OAuth flows and added two simple things on top:

1. A **special scope** called `openid`. If you ask for the `openid` scope when you start the OAuth dance, the server knows "this Client wants identity info, not just permissions."

2. A **new token** called the **ID Token**. The ID Token is a signed JSON Web Token (JWT) that the IdP gives back to the Client at the same time as the access token. The ID Token is a proof-of-identity letter. It says "I, Google, swear that the user who just authenticated is Jane Doe with sub=109283740 and email=jane@example.com, and here is my signature so you know it's really me saying it."

So OIDC is just **OAuth 2.0 with an identity layer added on top.** Same redirects. Same dance. Same actors. Just one extra scope (`openid`) and one extra token (the ID Token).

### The one-line summary you can quote at parties

> OAuth = "this app can do X on your behalf." OIDC = "this app knows you are Y."

If you remember nothing else, remember that line.

### What if I just want OAuth without OIDC?

You can. Pure OAuth is for things like "let this third-party photo editor read photos from my Dropbox." The app does not need to know your name. It just needs the photos. So it asks for the `photos.read` scope, gets a token, and reads photos. No identity. No `openid` scope. No ID Token.

But the moment your goal is "log this user into my app," you almost certainly want OIDC. You want to know the user's stable ID so you can save records under that ID. You want to know their email so you can email them. OIDC is the right tool.

### What if I just want OIDC without OAuth?

Cute idea, but no. OIDC **rides on** OAuth. Every OIDC flow is also an OAuth flow. There is no OIDC dance that is not also an OAuth dance. If you do OIDC, you are doing OAuth too. The ID Token comes along with an access token (or sometimes you can ask for ID Token only, but that is uncommon).

### The UserInfo endpoint

OIDC also defines a standard `userinfo_endpoint`. After you get an access token, you can call the UserInfo endpoint to get more details about the user. The basic ID Token usually has a small set of claims (sub, email, name, picture). UserInfo can give you more (phone number, address, custom fields the IdP knows about).

UserInfo is optional. If your ID Token already has everything you need, skip it. If you want richer profile data, hit it. The endpoint is at `https://idp.example.com/userinfo` and you call it with `Authorization: Bearer <access_token>`.

### A note on terminology

In OIDC speak, the Client is called the **Relying Party** or **RP** (the party that "relies" on the IdP for identity). The Authorization Server is called the **OpenID Provider** or **OP**. These are the same actors as in OAuth, just renamed.

```
+--------------------+
|   You (User)       |     <- "Resource Owner" in OAuth
|                    |     <- "End-User" in OIDC
+--------------------+
          |
          v
+--------------------+
|   Browser          |
+--------------------+
          |
          v
+--------------------+
|   PomodoroFun      |     <- "Client" in OAuth
|   (Web App)        |     <- "Relying Party" / "RP" in OIDC
+--------------------+
          |
          v
+--------------------+
|   Google IdP       |     <- "Authorization Server" in OAuth
|                    |     <- "OpenID Provider" / "OP" in OIDC
+--------------------+
          |
          v
+--------------------+
|   Calendar API     |     <- "Resource Server" in OAuth
|                    |     <- (no special OIDC name)
+--------------------+
```

## Authorization Code Flow (The Big One)

Let's draw it again, this time with more boxes and arrows. This is the dance you will see most. Almost every "Sign in with X" button in the world uses this flow.

### The five-step dance with PKCE

Modern OAuth always uses PKCE (pronounced "pixie"). PKCE stands for **Proof Key for Code Exchange.** We will explain what PKCE actually does in the next section. For now, just know it is an extra little dance step that makes the whole thing safer.

Here is the full picture:

```
USER             CLIENT                AUTH SERVER (IdP)            RESOURCE SERVER
 |                  |                          |                           |
 |---click login--->|                          |                           |
 |                  |                          |                           |
 |                  |--generate code_verifier  |                           |
 |                  |  (random 43-128 chars)   |                           |
 |                  |--code_challenge =        |                           |
 |                  |  SHA256(code_verifier)   |                           |
 |                  |                          |                           |
 |<----redirect to /authorize?...code_challenge=X--&state=Y-----|          |
 |                                             |                           |
 |---enter password + consent ---------------->|                           |
 |                                             |                           |
 |<--redirect back with ?code=Z&state=Y--------|                           |
 |                                             |                           |
 |---deliver code to Client--->|               |                           |
 |                  |                          |                           |
 |                  |--POST /token             |                           |
 |                  |  code=Z                  |                           |
 |                  |  code_verifier=...       |                           |
 |                  |  client_id=...           |                           |
 |                  |  client_secret=...       |                           |
 |                  |--------------------------->|                         |
 |                  |                          |                           |
 |                  |<--access_token, id_token, refresh_token----|         |
 |                  |                          |                           |
 |                  |--GET /api/data           |                           |
 |                  |  Authorization: Bearer access_token                  |
 |                  |---------------------------------------->             |
 |                  |                          |                           |
 |                  |<--{"calendar": [...]}-----|                          |
 |                  |                          |                           |
 |<--show data to user-----------|             |                           |
```

Five steps:

**Step 1 — Authorization Request.** The Client builds a URL pointing at the Authorization Server's `/authorize` endpoint and redirects the user there. The URL includes:
- `response_type=code` — "I want an authorization code."
- `client_id=abc123` — "I am Client abc123."
- `redirect_uri=https://pomodorofun.com/callback` — "After you're done, send the user back here."
- `scope=openid profile email calendar.read` — "Here are the permissions I want."
- `state=xyz` — "Echo this back to me later so I can match the response to my request."
- `code_challenge=hash` — "I have a secret. Here is its hash. Hold onto this."
- `code_challenge_method=S256` — "The hash is SHA-256."

**Step 2 — User Authenticates and Consents.** The user sees the IdP's login page. They type their password. They see a consent screen ("PomodoroFun wants access to your calendar"). They click Allow.

**Step 3 — Authorization Response.** The IdP generates a short-lived `code` and redirects the user back to the Client with `?code=Z&state=xyz`. The code is opaque — it is just a random-looking string. It is good for about thirty seconds. It can only be used once.

**Step 4 — Token Request.** The Client (running on its server) takes the code and POSTs it back to the IdP's `/token` endpoint. The Client includes:
- `grant_type=authorization_code`
- `code=Z` — the code it just got
- `redirect_uri=https://pomodorofun.com/callback` — same as before (must match)
- `client_id=abc123`
- `client_secret=secretvalue` — only confidential clients have this
- `code_verifier=originalSecret` — the original secret behind the hash

The IdP checks: "Is this code valid? Is it within the time window? Is the SHA-256 of code_verifier equal to the code_challenge I saw earlier? Does redirect_uri match what you sent the first time? Is your client_secret correct?" If everything checks out, the IdP returns:
- `access_token` — the valet key
- `id_token` — the proof-of-identity letter (only if `openid` scope was requested)
- `refresh_token` — a longer-lived key for getting more access tokens
- `token_type=Bearer`
- `expires_in=3600` — the access token is good for 1 hour

**Step 5 — Use the Tokens.** The Client uses the access token to call APIs. It puts the token in the `Authorization` header: `Authorization: Bearer <access_token>`. The Resource Server checks the token (either by validating its signature locally if it's a JWT, or by calling the IdP's `/introspect` endpoint if it's opaque) and serves the data.

When the Client wants to use the ID token, it parses it as a JWT, verifies the signature against the IdP's public keys (from `jwks_uri`), and reads the claims to know who the user is.

### Why two channels?

You will notice that the Client and IdP communicate in two completely different ways during this dance:

**Front channel** (steps 1, 2, 3): the user's browser is the messenger. The Client redirects the browser to the IdP, the IdP redirects the browser back. All of this is visible to the user. URLs are visible in the browser's address bar. Anyone over the user's shoulder could see them. Anyone with browser history access could see them.

**Back channel** (step 4): the Client's server talks directly to the IdP's server. No browser. No user. Server to server. This is private. Nothing is visible to anyone except the two parties.

That is why the dance has two parts: the browser delivers a code (which is fine to be visible because it's useless without a secret), and the server trades that code for a token (in private, where the actual valuable token never crosses the user's machine in a way an attacker could grab).

Old OAuth (the implicit flow, which is now deprecated) tried to skip the back channel by handing the access token straight back through the browser. That meant the access token ended up in browser history, in browser referrer headers leaking to other sites, in browser extensions' grasp, in any malicious plugin's reach. It was a bad idea. PKCE plus authorization code is the modern replacement.

### The role of `client_secret`

A confidential Client (a server-side app like a real web backend) has a `client_secret`. The IdP issued it when the Client registered. It is essentially a password for the Client itself. The Client uses it in step 4 to prove "yes, I am the same Client that started this whole dance."

A public Client (a mobile app, a single-page app, a CLI) **cannot have a client_secret.** Why? Because the Client's code runs on the user's device. The user can decompile it. The user can extract the secret. So a "secret" embedded in a public Client is not really secret. Anyone can read it.

So how does a public Client prove it is itself in step 4? **PKCE.** That is what PKCE is for. PKCE replaces the client_secret for public Clients.

## PKCE (Proof Key for Code Exchange)

### The problem PKCE solves

Imagine you are a public Client — say, a mobile app on someone's phone. You start the dance. The user goes to the IdP, logs in, consents. The IdP redirects back to your app's `redirect_uri`. The redirect URI for a mobile app is usually a **custom URL scheme** like `myapp://callback`. The OS sees that scheme and launches your app, passing the URL with the code in the query string.

But mobile operating systems are messy places. Multiple apps can register the same URL scheme. A malicious app could register `myapp://` too. When the OS sees `myapp://callback?code=Z`, it might launch the malicious app instead of yours. Now the malicious app has your authorization code.

Without PKCE, the malicious app could now go to the IdP's `/token` endpoint, present the code, and exchange it for an access token. Game over. The attacker has tokens for the user's account.

PKCE prevents this.

### The PKCE dance

Before redirecting the user, the Client generates a random secret called the **code_verifier.** It is a random string between 43 and 128 characters long. Generate it freshly every time you start a dance. Treat it like a password — keep it in memory, do not log it, do not write it to disk.

The Client then computes the SHA-256 hash of the code_verifier and base64url-encodes it. That hash is called the **code_challenge.** The code_challenge is what the Client sends to the IdP at the start of the dance.

```
code_verifier (43-128 chars random):
"M25iVXpKU3puUjFaYWg3T1NDTDQtcW1ROUY5YXlwalNoc0hhakxifmZHag"

code_challenge = base64url(SHA-256(code_verifier)):
"qjrzSW9gMiUgpUvqgEPE4_-LSdh7yZsOWbWlo7Vn5_E"
```

The IdP stores the code_challenge alongside the authorization code it issues.

When the malicious app intercepts the code and tries to redeem it, it has to call the `/token` endpoint with the `code_verifier`. But the malicious app does not know the code_verifier — only the legitimate Client knows it. The malicious app cannot send the right code_verifier. The IdP runs SHA-256 on whatever it sends, sees the hash does not match the stored code_challenge, and refuses to issue tokens.

The legitimate Client, by contrast, has the original code_verifier in its memory. It sends `code_verifier=M25iVXpKU3puUjFaYWg3T1NDTDQtcW1ROUY5YXlwalNoc0hhakxifmZHag` and the IdP checks SHA-256 of that against the stored code_challenge, sees they match, and issues tokens.

```
PKCE protects against a stolen authorization code:

1. Client generates code_verifier (random secret).
2. Client sends SHA-256(code_verifier) = code_challenge with /authorize.
3. IdP stores code_challenge, returns code.
4. Attacker steals code (somehow). Attacker calls /token with stolen code.
5. Attacker MUST also send code_verifier. Attacker does not know it.
6. IdP refuses to issue tokens.
7. Legitimate Client calls /token with code + correct code_verifier. Tokens issued.
```

### "S256" vs "plain"

The PKCE spec defines two methods:
- **S256** — code_challenge = base64url(SHA-256(code_verifier)). This is what you should use. Always.
- **plain** — code_challenge = code_verifier. This is only allowed for backward compatibility. Do not use it.

If your library has an option, set it to S256. If you ever see a server accepting `plain`, that is a configuration smell.

### Should confidential clients also use PKCE?

Yes. The OAuth 2.1 draft says all clients should use PKCE, even confidential ones. The argument is: PKCE is essentially free, and it provides defense-in-depth against various weird attacks (like session fixation). There is no reason not to use it. Just turn it on for everybody.

Modern IdPs like Auth0, Okta, Google, Microsoft Azure AD, Keycloak all support PKCE for any Client type.

## Other OAuth Flows

The Authorization Code Flow with PKCE is the workhorse. But OAuth defines a few others, for specific situations.

### Implicit Flow (deprecated)

The Implicit Flow used to be the way single-page apps did OAuth. It worked like this: the IdP returned the access token directly in the redirect URL, in the fragment (the `#` part of the URL): `https://pomodorofun.com/callback#access_token=ABCD&token_type=Bearer&expires_in=3600`.

This was bad because:
- Tokens leaked into browser history.
- Tokens leaked via the Referer header to any analytics scripts.
- Tokens were exposed to any browser extension.
- No refresh tokens were available.
- No PKCE.

OAuth 2.1 deletes the Implicit Flow. Modern advice: SPAs should use Authorization Code with PKCE instead. Yes, an SPA can do that, even though it is "public." The PKCE protocol takes care of the security; the SPA can call `/token` directly from JavaScript, holding the code_verifier in memory.

### Resource Owner Password Credentials (deprecated)

This flow lets the Client take the user's username and password directly and POST them to the IdP's `/token` endpoint. The IdP returns tokens.

This is **not OAuth** in the spirit of OAuth. The whole point of OAuth was to avoid handing your password to third-party apps. The Password Credentials grant tosses that out the window. The Client sees the password.

It existed as a "migration helper" for legacy apps that used to do password screen-scraping. The idea was, before you fully convert your app to use Authorization Code, you can use Password Credentials to get a token using the password your app already has, then later refactor to redirects.

In practice, people abused it. It is now deprecated. OAuth 2.1 deletes it. Do not use it. If you find yourself wanting to use it, you almost certainly want a different flow.

### Client Credentials (machine-to-machine)

Sometimes a Client wants tokens for itself, not for a user. Like a backend service that calls another backend service. There is no human in the loop. There is no consent screen.

For this, OAuth defines the **Client Credentials Flow.** The Client POSTs its `client_id` and `client_secret` to `/token` directly:

```
POST /token HTTP/1.1
Host: idp.example.com
Authorization: Basic <base64(client_id:client_secret)>

grant_type=client_credentials&scope=read:reports
```

The IdP returns an access token good for the Client's identity. No user is involved. No ID Token is issued (there is no user to identify). It is purely "service A wants to call service B and needs to prove it is service A."

This is the right flow for cron jobs, microservice-to-microservice calls, scheduled tasks, server-side automation. Do not use it when there is a real user involved.

### Device Code Flow (CLI, Smart TV, IoT)

What about devices that have no keyboard? Like a smart TV. Or a CLI that wants to log in but cannot pop a browser. Or an IoT device with no screen.

The **Device Code Flow** (RFC 8628) handles this. It works like this:

1. The device asks the IdP, "I want to start a flow." The IdP returns a `user_code` (something short and friendly, like "WDJB-MJHT") and a `device_code` (a long opaque string just for the device) and a verification URL.
2. The device shows the user the URL and the user_code: "Go to https://example.com/device on your phone or laptop, enter the code WDJB-MJHT."
3. The user opens that URL on a different device (their phone). They log in normally. They see the consent screen. They click Allow.
4. Meanwhile, the device polls the IdP's `/token` endpoint every few seconds: "Has the user completed the flow yet?"
5. Once the user finishes, the next poll returns access and ID tokens.

This is how `gh auth login` works in the GitHub CLI. This is how `aws sso login` works. This is how Netflix on a smart TV works (kinda).

```
DEVICE                          IDP                   USER (on phone/laptop)
  |                              |                            |
  |---POST /device/code--------->|                            |
  |<--user_code WDJB-MJHT--------|                            |
  |   device_code abcdef...      |                            |
  |   verification_uri https://example.com/device             |
  |                              |                            |
  |---show user the code---------|                            |
  |                              |                            |
  |                              |<--user opens URL, enters code, logs in
  |                              |   approves on consent screen
  |                              |                            |
  |---poll /token--------------->|                            |
  |<--still pending--------------|                            |
  |---poll /token--------------->|                            |
  |<--access_token + id_token----|                            |
```

### Refresh Token Flow

Access tokens are short-lived (often 1 hour). When they expire, you need a new one. You **could** make the user log in again every hour. That would be terrible UX.

Instead, when the IdP first issued tokens, it included a **refresh token**. The refresh token is long-lived (sometimes weeks, sometimes months, sometimes "until revoked"). Whenever the access token expires, the Client POSTs the refresh token to `/token` and gets a brand new access token (and often a new refresh token too).

```
POST /token HTTP/1.1
Host: idp.example.com

grant_type=refresh_token
refresh_token=tGzv3JOkF0XG5Qx2TlKWIA
client_id=abc123
client_secret=secretvalue
```

Refresh tokens are powerful. If an attacker steals one, they can use it to mint access tokens for as long as it remains valid. So refresh tokens deserve serious protection. We will talk about refresh token rotation and theft detection later.

### Token Exchange (RFC 8693, federation)

Sometimes you have a token from one IdP and you want a token from another IdP. Like, your app is logged in via Google, but you want to call an internal service that only trusts your-company.com tokens. The internal service does not trust Google directly.

**Token Exchange** lets you trade one token for another. You send your Google access token to your-company.com's `/token` endpoint with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` and a few other params. Your-company.com validates the Google token, decides whether to trust it, and issues a new your-company.com token in return.

This is most common in enterprise federation. SaaS sign-in-with-something flows. Cross-tenant access. AWS STS GetSessionToken-style scenarios.

## Tokens — Three Kinds

You have already seen all three of these mentioned. Let's lay them out clearly.

### Access Token (the valet key)

The access token is the actual permission grant. It is what you put in the `Authorization` header when calling APIs. It is short-lived — usually 5 to 60 minutes. It says, "the bearer of this token is allowed to do X, Y, and Z, until time T."

Access tokens are usually opaque (just a random string) or structured (a JWT). If they are JWTs, the Resource Server can verify them locally without calling the IdP. If they are opaque, the Resource Server has to call the IdP's `/introspect` endpoint to check.

```
Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IjEifQ.eyJzdWIiOiIxMjMiLCJhdWQiOiJhcGkiLCJleHAiOjE3MDAwMDAwMDB9.signature...
```

### Refresh Token (the long-lived ticket)

The refresh token is how the Client gets new access tokens without involving the user. It is long-lived. It is sensitive. It should never be sent to a Resource Server — only ever to the IdP's `/token` endpoint.

Refresh tokens are usually opaque. They are not meant to be parsed or read; they are just opaque blobs the IdP looks up.

```
{
  "access_token": "eyJ...",
  "expires_in": 3600,
  "refresh_token": "tGzv3JOkF0XG5Qx2TlKWIA",
  "token_type": "Bearer"
}
```

### ID Token (the proof-of-identity letter)

The ID Token is OIDC's gift to identity. It is **always a JWT.** The Client receives it, parses it, verifies the signature, and reads the claims. This is how the Client knows who just logged in.

The ID Token has fields like:
- `iss` — the issuer (which IdP)
- `sub` — the subject (a stable user ID)
- `aud` — the audience (which Client this token is for)
- `exp` — when it expires
- `iat` — when it was issued
- `nonce` — what the Client sent to start the dance (echoed back to prove freshness)
- `email`, `name`, `picture` — typical profile fields
- `auth_time` — when the user actually authenticated
- `acr`, `amr` — authentication context (e.g., "MFA was used")

The ID Token is **not** for calling APIs. Do not put it in the `Authorization` header. It is **only** for the Client to know who the user is.

```
ID Token = "Letter from the IdP to the Client about the user."
Access Token = "Stamped permission slip the Client shows to APIs."
Refresh Token = "Ticket the Client trades for new permission slips."
```

```
+-----------+      +-----------+      +-----------+
| ID Token  |      | Access    |      | Refresh   |
| (who?)    |      | Token     |      | Token     |
|           |      | (what?)   |      | (renew?)  |
+-----------+      +-----------+      +-----------+
| Always    |      | Sometimes |      | Always    |
| a JWT     |      | a JWT,    |      | opaque    |
|           |      | sometimes |      |           |
| Read by   |      | opaque    |      | Sent only |
| Client    |      |           |      | to /token |
| only      |      | Read by   |      | endpoint  |
|           |      | API       |      |           |
| Short     |      | Short     |      | Long      |
| (~1hr)    |      | (~1hr)    |      | (~30 days)|
+-----------+      +-----------+      +-----------+
```

## JWT Anatomy

JWTs are everywhere in OAuth/OIDC. ID Tokens are JWTs. Many access tokens are JWTs. Let's pick one apart.

### The three parts

A JWT looks like this:
```
eyJhbGciOiJSUzI1NiIsImtpZCI6IjEifQ.eyJzdWIiOiIxMjMiLCJleHAiOjE3MDAwMDAwMDB9.signature_bytes_base64url
```

Three sections separated by dots: **header**, **payload**, **signature.**

Each section is **base64url-encoded.** Base64url is just regular base64 with `+` replaced by `-` and `/` replaced by `_` and no padding `=`. That makes it safe to put in URLs without escaping.

### The header

Base64url-decode the first part. You get JSON like:
```json
{
  "alg": "RS256",
  "kid": "1",
  "typ": "JWT"
}
```

- `alg` — the signature algorithm. Common: `RS256` (RSA + SHA-256), `ES256` (ECDSA + SHA-256), `HS256` (HMAC + SHA-256).
- `kid` — the key ID, a hint about which signing key was used. Look this up in JWKS to verify the signature.
- `typ` — usually `JWT`. Sometimes `at+jwt` for access tokens (RFC 9068).

### The payload (claims)

Base64url-decode the second part. JSON:
```json
{
  "iss": "https://accounts.google.com",
  "sub": "109283740",
  "aud": "abc123.apps.googleusercontent.com",
  "exp": 1700001000,
  "iat": 1700000000,
  "nonce": "n-0S6_WzA2Mj",
  "email": "jane@example.com",
  "name": "Jane Doe"
}
```

These fields are called **claims.** Some have standard meanings:

- `iss` (issuer) — who issued this token. Compare it against an allowlist.
- `sub` (subject) — who the token is about. The user's stable ID.
- `aud` (audience) — who this token is for. **Always validate this.**
- `exp` (expiration time) — Unix timestamp when the token expires. Reject after.
- `iat` (issued at) — Unix timestamp when the token was issued.
- `nbf` (not before) — Unix timestamp before which the token is invalid.
- `jti` (JWT ID) — a unique ID for this specific token. Useful for replay prevention.
- `scope` — for access tokens, the permissions granted.

OIDC adds:
- `nonce` — must match the nonce the Client sent in the auth request.
- `auth_time` — when the user actually authenticated.
- `acr`, `amr` — authentication strength markers.

```
JWT structure:

  HEADER          .   PAYLOAD             .   SIGNATURE
  (algorithm,         (claims about           (cryptographic
   key id)             user, expiry,           proof header
                       audience, etc)           + payload
                                                weren't tampered)

  base64url-encoded each, dot-separated
```

### The signature

Base64url-decode the third part. You get raw bytes — they are not JSON. They are the output of:
```
sign(secret_or_private_key, base64url(header) + "." + base64url(payload))
```

For `RS256`, sign with RSA private key. Verify with the matching RSA public key.
For `HS256`, "sign" with an HMAC over a shared secret. Verify with the same shared secret.

This is why the signature ties the header and payload together. If anyone changes a single character of the header or payload, the signature stops verifying. The token becomes invalid.

### Why "RS256, not HS256" for OAuth?

`HS256` uses a shared secret. The same secret signs and verifies. If the IdP and Client share the secret, both can sign. That means a malicious Client could forge tokens.

`RS256` uses asymmetric keys. The IdP has the private key and signs. The Client (or any Resource Server) has the public key and verifies. The Client cannot forge. Only the IdP can sign.

So for OAuth/OIDC, you almost always want **RS256 or ES256** (asymmetric). Only use HS256 if you know what you are doing and you have a single trusted Client.

There is also a famous attack called **alg-confusion**: the IdP advertises `RS256` in its metadata, but a sloppy verifier accepts `HS256`. A malicious actor sets `alg: HS256` in the header, then signs the JWT using **the public key as if it were an HMAC secret.** Some libraries check the public key with HMAC and accept it. Disaster.

The fix: pin the algorithm. Tell your library "verify only with RS256." Reject anything else.

### The "alg: none" attack

There is also a notorious "alg: none" attack. The original JWT spec allows `alg: none` for unsigned tokens. Some libraries respect that. So a malicious actor sends a token with `{ "alg": "none" }` in the header and an empty signature. The library reads "alg: none" and says "okay, no signature to check" and accepts the token.

Anybody could forge a token. This was a real vulnerability in many libraries circa 2015. Most libraries now reject `alg: none` by default. Your library should.

### One-liner to decode a JWT

```bash
echo "eyJhbGc..." | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

Or with `jq`:
```bash
JWT="eyJhbGc..."
jq -R 'split(".") | .[1] | @base64d | fromjson' <<< "$JWT"
```

Decoding just inspects. It does not verify. Anyone can decode a JWT and read the payload. The point of the signature is that nobody else can produce a valid JWT with their own payload.

## Scopes

Scopes are the granular permission list. They are how the Client says "I want to do X" and the user says "okay, I allow X."

### Standard OIDC scopes

- `openid` — required to get an ID Token. Without it, the IdP treats the request as pure OAuth (no identity).
- `profile` — gives access to standard profile claims: name, family_name, given_name, picture, locale, etc.
- `email` — gives access to email and email_verified claims.
- `phone` — gives access to phone_number and phone_number_verified.
- `address` — gives access to a structured address claim.
- `offline_access` — asks for a refresh token. Without this, the IdP might not issue one. Different IdPs have different rules.

### API-specific scopes

Each API defines its own scopes. Common conventions:
- `calendar.read`, `calendar.write` — Google-style "noun.verb"
- `read:messages`, `write:messages` — Auth0-style "verb:noun"
- `https://www.googleapis.com/auth/calendar.readonly` — full URL scope (Google does this for some APIs)
- `api://my-api/Read.All` — Microsoft-style with full URI

### How to combine

Just space-separate them in the auth request:
```
scope=openid profile email calendar.read drive.file
```

The user will see all of them on the consent screen.

### Down-scoping

You can ask for a subset of what was originally granted. Like, the user granted `calendar.read calendar.write drive.file`. You only need calendar at the moment. You can request a fresh token with `scope=calendar.read` only. The IdP returns a token scoped just to that. This limits blast radius if that specific token gets stolen.

### Up-scoping

You cannot up-scope. If the user only granted `calendar.read`, you cannot ask for a token with `calendar.write` without going back through user consent. The IdP will say "no, the user didn't agree to that."

## Discovery (.well-known/openid-configuration)

How does a Client know where the IdP's `/authorize` endpoint is? Where the `/token` endpoint is? What scopes are supported? What signing algorithms?

It looks at the discovery document. Every OIDC-compliant IdP publishes a JSON document at:

```
https://idp.example.com/.well-known/openid-configuration
```

This is **RFC 8414** for plain OAuth and an OIDC spec for OIDC. It says, "If you support discovery, host metadata at this well-known URL."

A typical discovery doc:
```json
{
  "issuer": "https://accounts.google.com",
  "authorization_endpoint": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_endpoint": "https://oauth2.googleapis.com/token",
  "userinfo_endpoint": "https://openidconnect.googleapis.com/v1/userinfo",
  "revocation_endpoint": "https://oauth2.googleapis.com/revoke",
  "jwks_uri": "https://www.googleapis.com/oauth2/v3/certs",
  "response_types_supported": ["code", "token", "id_token", ...],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid", "email", "profile"],
  "token_endpoint_auth_methods_supported": ["client_secret_post", "client_secret_basic"],
  "claims_supported": ["aud", "email", "email_verified", "exp", "family_name", ...],
  "code_challenge_methods_supported": ["plain", "S256"],
  "grant_types_supported": ["authorization_code", "refresh_token", ...]
}
```

A Client library can hit this URL once at startup and configure itself. No need to hardcode endpoints. If the IdP changes URLs, the Client picks it up automatically.

There is a similar OAuth-only well-known: `/.well-known/oauth-authorization-server` (RFC 8414). Most modern IdPs publish both.

## JWKS (JSON Web Key Set)

The discovery doc has a `jwks_uri` field. That URL points at a list of public signing keys. This is how the Client knows what public key to use to verify a JWT.

```
GET https://www.googleapis.com/oauth2/v3/certs

{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "1",
      "n": "vbcFrj193Gm6zeo5...big_modulus...",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "2",
      "n": "different_modulus...",
      "e": "AQAB"
    }
  ]
}
```

Each key has a `kid` (key ID). When the IdP signs a JWT, it sets the `kid` in the JWT's header. The Client looks at the `kid`, finds the matching key in JWKS, uses its `n` (modulus) and `e` (exponent) to construct the public key, and verifies the signature.

### Key rotation

IdPs rotate signing keys periodically. They might have two keys live at the same time: the new one (used for signing new tokens) and the old one (still in JWKS so Clients can verify older still-valid tokens). After all old tokens have expired, the old key is removed from JWKS.

```
TIME 0:  JWKS = [k1]               IdP signs with k1
TIME 30: JWKS = [k1, k2]           IdP starts signing with k2
                                    Old k1 tokens still verify
TIME 60: JWKS = [k2]               All k1 tokens have expired,
                                    k1 removed from JWKS
                                    IdP signs only with k2
TIME 90: JWKS = [k2, k3]           Rotation begins again
```

A Client should fetch JWKS once at startup, then refresh occasionally (every few hours). When verifying a JWT, look up the `kid` in the cached JWKS. If the `kid` is not in the cache, refresh JWKS once before failing — the IdP might have just rotated.

### Why ASCII-armored JSON instead of PEM?

Because JWKS is designed to be machine-readable and HTTP-friendly. PEM has line wrapping, header lines, base64 encoding. JWKS is plain JSON with structured fields. Easier to parse from any language. Easier to add metadata. Easier to support multiple key types in the same document (RSA, EC, OKP for EdDSA).

## Audience Validation

Every JWT has an `aud` claim. It says, "this token is for client_id abc123" or "this token is for api://my-api". When you receive a JWT, you must check the `aud` is what you expect.

### Why?

Imagine you run two services: PomodoroFun and FocusMaster. Both have their own client_ids. Both accept JWT access tokens.

A user logs into PomodoroFun and gets an access token. The access token's `aud` is `pomodorofun-client-id`.

Now imagine PomodoroFun has a security bug. The user's PomodoroFun token leaks. An attacker grabs it and tries to use it against FocusMaster's API.

If FocusMaster doesn't check `aud`, FocusMaster sees a valid signature, valid expiration, and accepts the token. The attacker now has access to the user's FocusMaster data, even though the user never granted access to FocusMaster.

If FocusMaster does check `aud`, FocusMaster sees `aud=pomodorofun-client-id`, says "this token is not for me," and rejects it. Attack stopped.

This is the **confused deputy attack.** Always check `aud`.

### What to check `aud` against

For a Client receiving an ID Token, check `aud == your_client_id`.

For a Resource Server receiving an access token, check `aud == your_api_identifier`. Sometimes it is a URL like `https://api.example.com`. Sometimes it is a name. Whatever the IdP configures.

If `aud` is an array (it can be), check that **your identifier is one of the entries.** Some IdPs put multiple audiences in.

There is also `azp` (authorized party). When `aud` is an array, `azp` says which one is the "primary" Client. You can check `azp` too if you care about that distinction.

## Refresh Token Best Practices

Refresh tokens live a long time. They are powerful. Treat them with care.

### Sliding window

Many IdPs use a "sliding window" model: every time you use a refresh token, the new refresh token's validity resets. So if your refresh token is good for 30 days, and you use it on day 25, the new refresh token is good for 30 more days from then. Active users effectively never get logged out.

### Rotation

A common pattern: every time the Client redeems a refresh token, the IdP issues a brand-new refresh token and invalidates the old one. The Client must store the new one.

```
TIME 0:  Client gets refresh_token_A
TIME 60: Client redeems A, gets back refresh_token_B + new access token
         A is now invalid.
TIME 120: Client redeems B, gets back refresh_token_C + new access token
         B is now invalid.
```

This way, if a refresh token is ever stolen, it can only be used once. After that one use, it is dead.

### Reuse detection

Combined with rotation: if the IdP ever sees an "old, already-rotated-out" refresh token come in for redemption, that is a red flag. It means **either** the legitimate Client is using a stale token (network glitch, restart) **or** an attacker is using a stolen token. Many IdPs treat this as a security event: invalidate the entire chain (the current token, the original token, all descendants), force the user to log in again.

```
Refresh token rotation with reuse detection:

TIME 0:  Client A gets RT1
TIME 30: Attacker steals RT1
TIME 60: Client A uses RT1, gets RT2, RT1 invalidated.
TIME 90: Attacker uses RT1. IdP sees: "RT1 was already redeemed!"
         IdP invalidates RT2 too. User forced to log in again.
         Attacker locked out. Legitimate user notices something happened
         and can investigate.
```

This is great defense-in-depth. It does not perfectly prevent theft, but it limits the damage and gives the legitimate user a chance to notice.

### Storage

Refresh tokens should not live in browser localStorage. Local storage is readable by any JavaScript on the page. An XSS attack could exfiltrate it.

For SPAs, the modern recommendation is:
- Don't issue refresh tokens to SPAs at all (use silent re-auth via the IdP cookie).
- Or use the **BFF (Backend-for-Frontend)** pattern, where a server-side component handles refresh tokens and the SPA only ever sees access tokens or just session cookies.

For mobile apps, store refresh tokens in the OS secure storage (iOS Keychain, Android Keystore).

For server-side apps, store them in a database, encrypted at rest, with a way to revoke them.

## Security Pitfalls

OAuth has a lot of corners where you can go wrong. Here are the big ones.

### CSRF on the redirect (state parameter required)

When the IdP redirects the user back to the Client with a code, the Client must know "this code came from a flow I started." Without that check, an attacker could trick the user's browser into delivering an attacker's code, and the Client would link it to the user's session — letting the attacker hijack the account.

The fix: send a `state` parameter in the auth request. It is a random opaque value the Client generates. Echo it back unchanged in the redirect. The Client then checks the state in the redirect matches the state it generated. Mismatch = abort.

### Open redirect

If the Client allows arbitrary `redirect_uri` values to flow through, an attacker can craft a URL like `?redirect_uri=https://evil.com/steal-tokens` and steal codes.

The fix: the IdP must validate `redirect_uri` against an allowlist. The Client registers permitted redirect URIs ahead of time. Anything else is rejected. **Exact matching, including query string and fragment.**

### Mix-up attack

Some OAuth deployments support multiple IdPs. The Client lets the user pick one. The attacker uses one IdP to get a code, then tricks the Client into believing it came from a different IdP, where the attacker is registered.

The fix: include the IdP's `iss` in the response (or use the `iss` parameter, RFC 9207). Validate `iss` matches the one you started the flow with.

### HS256/RS256 confusion

Already discussed. Pin the algorithm. Reject HS256 unless you really mean it.

### "alg: none"

Already discussed. Reject any JWT with `alg: none`. Most modern libraries do.

### Public client without PKCE

If a public Client (mobile, SPA) does Authorization Code without PKCE, a malicious app can intercept the code and exchange it for a token. PKCE prevents this. **Always use PKCE.**

### Storing tokens in localStorage

Already discussed. localStorage is not secure for XSS. Use httpOnly cookies, secure storage APIs, or the BFF pattern.

### Long-lived access tokens

Access tokens are bearer tokens — anyone holding the string can use it. The longer they live, the more damage a stolen one can do. Keep them short. 5 minutes to 1 hour is typical. Lean on refresh tokens for the long-lived part.

### Implicit flow

Already discussed. Just don't.

### Resource owner password credentials

Already discussed. Just don't.

### Insufficient session binding

If the IdP uses cookies (session cookies) on the IdP domain, those cookies tie a session to a browser. An access token issued in one browser should not just float over to another browser. Things like **DPoP** (RFC 9449) and **mTLS-bound tokens** (RFC 8705) bind tokens to a specific cryptographic key, making them useless if leaked.

### Insufficient logout

When a user logs out of the Client, do you also invalidate tokens at the IdP? Do you call `revocation_endpoint`? Do you redirect them to `end_session_endpoint`? OIDC defines RP-initiated logout, front-channel logout, and back-channel logout to handle this, but most apps do them poorly.

## Common Errors

OAuth defines a small set of standard error codes that every implementation returns. Memorize them and you can debug 90% of issues by reading the error.

### `invalid_client`

The Client failed authentication. Either the `client_id` is wrong, the `client_secret` is wrong, or the auth method (Basic auth header vs body params) doesn't match what the IdP expects.

Debug: print the exact Client ID you're using. Print the auth method. Compare to the IdP's expected config. Common mistake: trailing whitespace in client_secret. Or you registered the Client as `client_secret_post` and you're using Basic auth (which is `client_secret_basic`).

### `invalid_grant`

The grant — code, refresh token, password, whatever — is invalid. Could mean:
- The authorization code expired (codes only live ~30 seconds).
- The authorization code was already used (codes are one-time-use).
- The `redirect_uri` you sent at /token doesn't match the one you sent at /authorize.
- The `code_verifier` doesn't match the `code_challenge`.
- The refresh token has been revoked.
- The user revoked consent on the IdP side.

Debug: walk through the dance manually with `curl`. Check timestamps. Check that you're sending the same redirect_uri at both endpoints.

### `invalid_scope`

The Client asked for a scope it isn't allowed to ask for, or a scope that doesn't exist.

Debug: check the IdP's `scopes_supported` in the discovery doc. Check the Client's registered allowed scopes.

### `invalid_token`

The Resource Server rejected an access token. Could mean:
- The token is expired.
- The signature didn't verify.
- The `aud` doesn't match the Resource Server's expected audience.
- The token was revoked.
- The Resource Server can't find the matching public key (kid not in JWKS).

Debug: decode the token. Check `exp`, `aud`, `iss`. Try fetching JWKS manually. Try introspecting the token via `/introspect`.

### `unauthorized_client`

The Client is not allowed to use this grant type. Like, it tried Client Credentials but is configured as a public Client (which doesn't get Client Credentials).

Debug: check the Client's allowed grant types in the IdP admin console.

### `access_denied`

The user said no on the consent screen. Or some policy on the IdP side denied the flow.

Debug: nothing the Client can do programmatically. The user must say yes.

### `server_error`

Something is wrong on the IdP. Misconfiguration. Network. Database. Whatever.

Debug: check the IdP's logs. Open a support ticket.

### Less common errors

- `unsupported_response_type` — you asked for a response_type the IdP doesn't support. Check `response_types_supported` in discovery.
- `unsupported_grant_type` — you used a grant_type the IdP doesn't support.
- `temporarily_unavailable` — IdP is overloaded; back off and retry.
- `interaction_required` — you tried `prompt=none` (silent flow) but the user actually needs to interact (re-login, MFA, consent).
- `login_required` — `prompt=none` flow, but the user has no active session at the IdP.
- `consent_required` — `prompt=none` flow, but the user hasn't consented to these scopes yet.

## Hands-On

Open a terminal. Type the lines that start with `$`. Watch the magic.

```bash
# 1. Fetch Google's OIDC discovery document.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .

# 2. Just look at the issuer.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .issuer

# 3. List supported scopes.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .scopes_supported

# 4. List supported grant types.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .grant_types_supported

# 5. List supported signing algorithms for ID tokens.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .id_token_signing_alg_values_supported

# 6. Find the JWKS URI from discovery.
JWKS_URI=$(curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq -r .jwks_uri)
echo "JWKS at: $JWKS_URI"

# 7. Fetch JWKS and look at the first key.
curl -s "$JWKS_URI" | jq .keys[0]

# 8. Count Google's signing keys.
curl -s "$JWKS_URI" | jq '.keys | length'

# 9. List the kids (key IDs) Google currently advertises.
curl -s "$JWKS_URI" | jq '.keys[].kid'

# 10. Decode an example JWT (just looks at payload, no verification).
JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkphbmUgRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.kbysX5YV9xrEWv3J9fsIEgTQuoIyP6FQQ_-IJgtkhtw"
echo "$JWT" | cut -d. -f2 | base64 -d 2>/dev/null
echo  # newline because base64 doesn't add one

# 11. Same thing with jq pretty-printing.
echo "$JWT" | cut -d. -f2 | base64 -d 2>/dev/null | jq .

# 12. Decode the header instead.
echo "$JWT" | cut -d. -f1 | base64 -d 2>/dev/null | jq .

# 13. Pure-jq one-liner (no cut+base64 piping).
jq -R 'split(".") | .[1] | @base64d | fromjson' <<< "$JWT"

# 14. Decode header with the same trick.
jq -R 'split(".") | .[0] | @base64d | fromjson' <<< "$JWT"

# 15. Pretty-print all three pieces of a JWT in sequence.
for i in 0 1; do
  echo "=== Section $i ==="
  jq -R "split(\".\") | .[${i}] | @base64d | fromjson" <<< "$JWT"
done
echo "=== Section 2 (signature, raw bytes, not decodable as text) ==="
echo "$JWT" | cut -d. -f3

# 16. Verify a JWT with smallstep's `step` CLI (install it first).
# brew install step
# step jwt verify --jwks "$JWKS_URI" "$JWT"

# 17. Microsoft's discovery doc (Azure AD).
curl -s "https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration" | jq .

# 18. Look at what's in their well-known — they have lots of "supported" arrays.
curl -s "https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration" | jq 'to_entries | map(select(.key | endswith("_supported")))'

# 19. GitHub's OAuth (no OIDC, no .well-known) — check via the docs URL.
# https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps

# 20. Auth0 tenant discovery (replace YOUR_TENANT).
# curl -s "https://YOUR_TENANT.auth0.com/.well-known/openid-configuration" | jq .

# 21. Okta org discovery.
# curl -s "https://YOUR_ORG.okta.com/.well-known/openid-configuration" | jq .

# 22. Keycloak realm discovery.
# curl -s "http://localhost:8080/realms/myrealm/.well-known/openid-configuration" | jq .

# 23. OAuth 2.0 only well-known (RFC 8414).
curl -s "https://accounts.google.com/.well-known/oauth-authorization-server" | jq . 2>/dev/null || echo "Google does not publish this; only the OIDC well-known."

# 24. Inspect a sample mock-oauth2-server discovery (run the server first).
# docker run -p 8080:8080 ghcr.io/navikt/mock-oauth2-server:2.1.0
# curl -s "http://localhost:8080/default/.well-known/openid-configuration" | jq .

# 25. Build an authorization URL by hand (don't actually visit, just look).
CLIENT_ID="my-client-id"
REDIRECT_URI="https://my-app.example.com/callback"
SCOPE="openid profile email"
STATE=$(openssl rand -hex 16)
echo "https://accounts.google.com/o/oauth2/v2/auth?response_type=code&client_id=${CLIENT_ID}&redirect_uri=${REDIRECT_URI}&scope=${SCOPE}&state=${STATE}"

# 26. Generate a PKCE code_verifier (random 43-128 chars).
CODE_VERIFIER=$(openssl rand -base64 64 | tr -d '=+/' | cut -c1-64)
echo "code_verifier: $CODE_VERIFIER"

# 27. Compute the PKCE code_challenge using S256.
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -binary -sha256 | openssl base64 | tr '+/' '-_' | tr -d '=')
echo "code_challenge: $CODE_CHALLENGE"

# 28. Use those in an authorization URL.
echo "https://idp.example.com/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=$REDIRECT_URI&scope=$SCOPE&state=$STATE&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256"

# 29. Client Credentials grant via curl.
# curl -X POST "https://idp.example.com/token" \
#   -H "Content-Type: application/x-www-form-urlencoded" \
#   -d "grant_type=client_credentials" \
#   -d "client_id=my-client-id" \
#   -d "client_secret=my-secret" \
#   -d "scope=read:reports"

# 30. Refresh token grant via curl.
# curl -X POST "https://idp.example.com/token" \
#   -H "Content-Type: application/x-www-form-urlencoded" \
#   -d "grant_type=refresh_token" \
#   -d "refresh_token=tGzv3JOkF0XG5Qx2TlKWIA" \
#   -d "client_id=my-client-id" \
#   -d "client_secret=my-secret"

# 31. Token introspection (RFC 7662).
# curl -X POST "https://idp.example.com/introspect" \
#   -u "my-client-id:my-secret" \
#   -d "token=eyJhbGciOiJSUzI1NiIs..."

# 32. Token revocation (RFC 7009).
# curl -X POST "https://idp.example.com/revoke" \
#   -u "my-client-id:my-secret" \
#   -d "token=tGzv3JOkF0XG5Qx2TlKWIA" \
#   -d "token_type_hint=refresh_token"

# 33. Call UserInfo with an access token.
# curl -H "Authorization: Bearer eyJhbGc..." "https://idp.example.com/userinfo"

# 34. Device Code grant — start.
# curl -X POST "https://idp.example.com/device/code" \
#   -d "client_id=my-client-id" \
#   -d "scope=openid profile"

# 35. Device Code grant — poll.
# curl -X POST "https://idp.example.com/token" \
#   -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
#   -d "device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS" \
#   -d "client_id=my-client-id"

# 36. Run a tiny local OIDC test using mock-oauth2-server.
docker run -d -p 8080:8080 --name mock-oidc ghcr.io/navikt/mock-oauth2-server:2.1.0
sleep 2
curl -s "http://localhost:8080/default/.well-known/openid-configuration" | jq .
docker stop mock-oidc && docker rm mock-oidc

# 37. Kubernetes-style service account token (also a JWT).
# kubectl create token my-service-account | head -c 100
# echo

# 38. AWS Cognito initiate auth (USER_PASSWORD flow, simplified example).
# aws cognito-idp initiate-auth \
#   --auth-flow USER_PASSWORD_AUTH \
#   --client-id YOUR_CLIENT_ID \
#   --auth-parameters USERNAME=jane@example.com,PASSWORD='hunter2'

# 39. AWS Cognito refresh tokens.
# aws cognito-idp initiate-auth \
#   --auth-flow REFRESH_TOKEN_AUTH \
#   --client-id YOUR_CLIENT_ID \
#   --auth-parameters REFRESH_TOKEN=eyJjdHkiOiJKV1Q...

# 40. Use Python authlib to do an authorization code dance.
# pip install authlib requests
cat <<'PY'
from authlib.integrations.requests_client import OAuth2Session
client = OAuth2Session(
    client_id="my-client-id",
    client_secret="my-secret",
    scope="openid profile email",
    redirect_uri="https://my-app/callback",
    code_challenge_method="S256",
)
url, state = client.create_authorization_url(
    "https://accounts.google.com/o/oauth2/v2/auth"
)
print("Visit:", url)
# After redirect with ?code=Z, exchange:
# tokens = client.fetch_token(
#     "https://oauth2.googleapis.com/token",
#     code="Z",
# )
# print(tokens)
PY

# 41. Decode and validate a token using Python (PyJWT).
# pip install pyjwt cryptography requests
cat <<'PY'
import jwt, requests
JWKS = requests.get("https://www.googleapis.com/oauth2/v3/certs").json()
def verify(token, audience):
    headers = jwt.get_unverified_header(token)
    kid = headers["kid"]
    key_data = next(k for k in JWKS["keys"] if k["kid"] == kid)
    public_key = jwt.algorithms.RSAAlgorithm.from_jwk(key_data)
    return jwt.decode(
        token, public_key,
        algorithms=["RS256"],
        audience=audience,
        issuer="https://accounts.google.com",
    )
# verify("eyJ...", audience="my-client-id.apps.googleusercontent.com")
PY

# 42. Watch a JWT expire in real time.
NOW=$(date +%s)
EXP=$((NOW + 30))
echo "Token expires at unix $EXP, in 30s."

# 43. Compare two issuers — sanity check you're talking to the right IdP.
ISS1="https://accounts.google.com"
ISS2=$(curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq -r .issuer)
[ "$ISS1" = "$ISS2" ] && echo "Issuer matches." || echo "MISMATCH"

# 44. Pretty-print the full Google discovery doc with sorted keys.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq -S .

# 45. List the auth methods Google supports at the token endpoint.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .token_endpoint_auth_methods_supported

# 46. List which response_types Google supports.
curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .response_types_supported
```

Lots of those last lines are commented out because they require credentials. The point is to give you the shape of the commands. Once you have a real Client and IdP, fill in your values and they will work.

## Common Confusions

**1. OAuth vs OIDC.**
OAuth is for authorization (permissions). OIDC is for authentication (identity). OIDC is built on top of OAuth. If you "Sign in with Google," you used both: OAuth gave you tokens, OIDC told you who the user was.

**2. Authentication vs Authorization.**
Authentication = "who are you?" Authorization = "what are you allowed to do?" Auth-N is identity. Auth-Z is permissions. OAuth is auth-Z. OIDC adds auth-N.

**3. Access Token vs ID Token.**
Access Token: "the bearer can do X." For APIs. ID Token: "the user is Y." For the Client. Never put an ID Token in `Authorization: Bearer`. Never use an access token to figure out who the user is — its `sub` might be the Client ID, not the user.

**4. Authorization Code vs Access Token.**
The code is a short-lived disposable ticket good only for one trade. The access token is the actual valet key. The code is what comes back through the browser; the access token never crosses the browser in modern flows.

**5. client_id vs client_secret.**
client_id is public. It is okay to have it in JavaScript. client_secret is private. It must never be in JavaScript or any code that runs on the user's machine. Public Clients (mobile, SPA) have a client_id but no secret — they use PKCE instead.

**6. Implicit Flow vs Authorization Code Flow.**
Implicit returned tokens directly through the browser. Deprecated, insecure. Authorization Code returns a code through the browser, traded for tokens server-side or via PKCE. Always use Authorization Code now, even for SPAs.

**7. Refresh Token vs Access Token.**
Access Token: short-lived, used to call APIs. Refresh Token: long-lived, used only to call /token. Different lifetimes, different audiences, different storage rules.

**8. JWT vs JWS vs JWE vs JWA vs JWK vs JWKS.**
JWA = list of algorithms. JWK = a single key, in JSON form. JWKS = a set of JWKs. JWS = a signed thing (header.payload.signature). JWE = an encrypted thing (different format). JWT = a JWS or a JWE that contains JSON claims. Most JWTs are JWSes.

**9. Scope vs Audience.**
Scope = "permissions list." Audience = "who is the token for." A token can have many scopes but typically one or a few audiences.

**10. Public Client vs Confidential Client.**
Confidential = runs on a server you control, can keep a secret. Public = runs on a user's device (mobile, SPA, CLI) and cannot keep a secret. Public Clients use PKCE. Confidential Clients use client_secret (and ideally PKCE too).

**11. Redirect URI vs Logout URI.**
Redirect URI is where the IdP sends the user after a successful auth. Logout URI is where the IdP sends the user after sign-out (post_logout_redirect_uri). They have to be registered separately.

**12. Front-channel vs Back-channel.**
Front-channel = via the user's browser (visible URLs). Back-channel = direct server-to-server (private). Modern flows mix both deliberately.

**13. iss vs aud vs sub.**
iss = issuer (which IdP). aud = audience (which app/API). sub = subject (which user). Always validate iss and aud. Use sub as your stable user identifier.

**14. Bearer Token vs Sender-Constrained Token.**
Bearer = "anybody who has this string can use it." Sender-Constrained = "only the holder of this private key can use it." DPoP and mTLS-bound tokens are sender-constrained. Better security, more complexity.

**15. SAML vs OIDC.**
Both are federation. SAML uses XML, predates OAuth, common in enterprise. OIDC uses JSON/JWT, simpler, mobile-friendly, ascendant. Both deliver "this user is X." If you're starting fresh, OIDC. If you're integrating with enterprise SSO, you might still need SAML.

**16. OAuth 1.0 vs OAuth 2.0.**
OAuth 1.0 (2007) used HMAC request signing. Painful. Secure even on plain HTTP. OAuth 2.0 (2012) requires TLS, drops signing, simpler. Different protocols. OAuth 1.0 is dead.

**17. nonce vs state.**
state = CSRF protection on the redirect (Client checks the redirect matches its own request). nonce = ID Token replay protection (Client checks the nonce in the ID Token matches the one it sent at /authorize). Both are random Client-generated values. Both must be checked.

## Vocabulary

| Term | Meaning |
|---|---|
| OAuth | Open Authorization — protocol for delegated access. |
| OAuth 2.0 | RFC 6749, the protocol everyone uses today. Published 2012. |
| OAuth 2.1 | A draft that consolidates 2.0 + best practices, removes deprecated grants. |
| OIDC | OpenID Connect — identity layer on top of OAuth 2.0. Published 2014. |
| OpenID 2.0 | Legacy protocol, replaced by OIDC. Use OIDC. |
| JWT | JSON Web Token — header.payload.signature, base64url-encoded JSON. |
| JWS | JSON Web Signature — the "signed" form of a JWT. |
| JWE | JSON Web Encryption — the "encrypted" form of a JWT. Uncommon in OAuth. |
| JWA | JSON Web Algorithms — RFC 7518, the algorithm catalog. |
| JWK | JSON Web Key — one cryptographic key in JSON form. |
| JWKS | JSON Web Key Set — a list of JWKs, served at jwks_uri. |
| JTI | JWT ID — unique identifier for a specific token (replay prevention). |
| JTI replay | Storing seen jti values to reject duplicate tokens. |
| claim | A name/value pair inside a JWT payload. |
| ID Token | OIDC's identity proof, always a JWT. Tells Client who the user is. |
| access token | The valet key. Short-lived. Used to call APIs. |
| refresh token | Long-lived, used to mint new access tokens. |
| opaque token | A token that's just an ID — has to be looked up at the IdP to validate. |
| structured token | A self-contained token, usually a JWT, validated locally. |
| bearer token | "Whoever holds this string is the legitimate user." Easy, fragile. |
| MAC token | Older, deprecated alternative to bearer. Required HMAC signing per request. |
| DPoP | Demonstrating Proof of Possession (RFC 9449) — token bound to client's key. |
| mTLS-bound token | Token bound to a TLS client certificate (RFC 8705). |
| token introspection | RFC 7662 — endpoint where you POST a token to get back its metadata. |
| token revocation | RFC 7009 — endpoint where you POST a token to invalidate it. |
| authorization endpoint | /authorize — where users get redirected to log in and consent. |
| token endpoint | /token — where Clients trade codes/refresh-tokens for access tokens. |
| userinfo endpoint | /userinfo — OIDC endpoint for getting user profile data. |
| jwks_uri | URL where the IdP publishes its public signing keys. |
| registration endpoint | /register — RFC 7591, where Clients can dynamically register. |
| introspection endpoint | /introspect — RFC 7662 token-validity endpoint. |
| revocation endpoint | /revoke — RFC 7009 token-revocation endpoint. |
| end_session_endpoint | OIDC RP-initiated logout endpoint. |
| discovery | The pattern of fetching metadata from a well-known URL. |
| .well-known | The IETF-blessed prefix for metadata URLs. |
| .well-known/openid-configuration | OIDC discovery doc URL suffix. |
| .well-known/oauth-authorization-server | OAuth 2.0 server metadata URL (RFC 8414). |
| authorization code | Short-lived ticket exchanged for tokens at /token. |
| authorization request | The browser redirect to /authorize. |
| authorization response | The redirect back from /authorize with code or tokens. |
| token request | POST to /token. |
| token response | JSON returned from /token. |
| redirect_uri | Where the IdP sends the user after auth. Must be allowlisted. |
| response_type | What the Client wants back. `code` for Auth Code Flow. |
| response_mode | How the response is delivered: query, fragment, form_post. |
| scope | Granular permission name. Space-separated list in requests. |
| state | Client-generated random; CSRF protection. |
| nonce | Client-generated random; replay protection in ID tokens. |
| audience (aud) | Which Client/API the token is intended for. |
| issuer (iss) | Which IdP issued the token. |
| subject (sub) | Stable user ID inside the IdP. |
| client_id | Public Client identifier. |
| client_secret | Confidential Client password. Never in browser code. |
| code_verifier | PKCE secret the Client generates and keeps in memory. |
| code_challenge | base64url(SHA-256(code_verifier)) — sent to IdP at /authorize. |
| S256 | The recommended PKCE method. |
| plain | The "no hashing" PKCE method; do not use. |
| PKCE | Proof Key for Code Exchange, RFC 7636. |
| RFC 7636 | The PKCE spec. |
| public client | Client that can't keep a secret (mobile, SPA, CLI). Uses PKCE. |
| confidential client | Server-side Client with a secret. |
| native app | Mobile or desktop app (RFC 8252). |
| SPA | Single-Page App. Always public, always uses PKCE. |
| server-side app | Web app where the Client logic runs on a server. |
| client credentials | M2M grant: Client gets tokens for itself, no user. |
| password grant | Resource Owner Password Credentials. Deprecated. |
| device code | RFC 8628 grant for devices with no keyboard. |
| RFC 8628 | The Device Authorization Grant spec. |
| token exchange | RFC 8693 — trade one token for another (federation). |
| RFC 8693 | The Token Exchange spec. |
| refresh token rotation | Each refresh issues a new refresh token, old one invalidated. |
| refresh token reuse detection | If an old refresh token is reused, invalidate the chain. |
| sender-constrained tokens | Tokens bound to a key/cert, not just bearer. |
| mTLS client certificates | Mutual TLS auth for Client to IdP/RS. |
| DPoP | RFC 9449 — proof-of-possession via Client keypair. |
| CIBA | Client-Initiated Backchannel Authentication. |
| RAR | Rich Authorization Requests, RFC 9396. |
| JAR | JWT-Secured Authorization Request, RFC 9101. |
| JARM | JWT-Secured Authorization Response Mode. |
| PAR | Pushed Authorization Request, RFC 9126. |
| FAPI | Financial-grade API — high-security profile of OAuth/OIDC. |
| OAuth Mix-Up Attack | Attack across multiple IdPs; mitigated by `iss` parameter. |
| IdP | Identity Provider — the place that knows the user's password. |
| identity provider | Same as IdP. |
| federation | Trusting an external IdP for identity. |
| SCIM | RFC 7642-7644 — System for Cross-domain Identity Management; user provisioning. |
| social login | Sign-in via consumer IdPs (Google, Facebook, GitHub). |
| MFA | Multi-Factor Authentication. |
| step-up auth | Going back to the IdP for stronger auth (e.g., MFA) before a sensitive op. |
| ACR claim | Authentication Context Class Reference (claim in ID token). |
| AMR claim | Authentication Methods References (e.g., ["pwd","otp"]). |
| login_hint | A hint about which user (often email) so the IdP can pre-fill. |
| prompt | "none", "login", "consent", or "select_account". Controls IdP behavior. |
| max_age | Maximum age of the user's authentication. |
| ui_locales | Preferred languages for the IdP's UI. |
| claims parameter | JSON in /authorize asking for specific claims. |
| request object | An entire /authorize request packed into a signed JWT. |
| response_mode query | Code/token in URL query string. |
| response_mode fragment | Code/token in URL fragment (used by Implicit; not by Code). |
| response_mode form_post | IdP POSTs an HTML form to redirect_uri instead of redirect. |
| Front-Channel Logout | OIDC logout via iframes. |
| Back-Channel Logout | OIDC logout via direct IdP-to-Client HTTP POST. |
| Single Logout | Logging out of all federated apps simultaneously. |
| RP | Relying Party — OIDC name for the Client. |
| OP | OpenID Provider — OIDC name for the Authorization Server / IdP. |
| end session | OIDC logout flow. |
| post_logout_redirect_uri | Where to redirect the user after logout. |
| id_token_hint | Pass an ID token to the end_session_endpoint to identify the session. |
| RFC 7591 | Dynamic Client Registration spec. |
| RFC 7592 | Dynamic Client Registration management. |
| RFC 6749 | Core OAuth 2.0 spec. |
| RFC 6750 | Bearer token usage in HTTP. |
| RFC 7009 | OAuth 2.0 Token Revocation. |
| RFC 7515 | JSON Web Signature (JWS). |
| RFC 7516 | JSON Web Encryption (JWE). |
| RFC 7517 | JSON Web Key (JWK). |
| RFC 7518 | JSON Web Algorithms (JWA). |
| RFC 7519 | JSON Web Token (JWT). |
| RFC 7662 | OAuth 2.0 Token Introspection. |
| RFC 8252 | OAuth for Native Apps best practices. |
| RFC 8414 | Authorization Server Metadata. |
| RFC 8628 | Device Authorization Grant. |
| RFC 8693 | Token Exchange. |
| RFC 8705 | OAuth 2.0 Mutual-TLS Client Authentication. |
| RFC 9068 | JWT Profile for OAuth Access Tokens. |
| RFC 9101 | JWT-Secured Authorization Request (JAR). |
| RFC 9126 | Pushed Authorization Requests (PAR). |
| RFC 9207 | OAuth 2.0 Authorization Server Issuer Identification. |
| RFC 9396 | Rich Authorization Requests (RAR). |
| RFC 9449 | DPoP — Demonstrating Proof of Possession. |
| OIDC Core 1.0 | The main OIDC spec. |
| OIDC Discovery 1.0 | The discovery spec. |
| OIDC DCR 1.0 | OIDC's take on Dynamic Client Registration. |
| OIDC Session Management 1.0 | Single-page session checking. |
| OIDC Front-Channel Logout 1.0 | Browser-based logout via iframe. |
| OIDC Back-Channel Logout 1.0 | Server-to-server logout. |
| FAPI 1.0 | Financial-grade API profile, original. |
| FAPI 2.0 | Financial-grade API profile, second generation. |
| token binding | Older proposal (TLS Token Binding); largely abandoned. |
| Bearer Auth header | `Authorization: Bearer <token>` syntax. |
| Basic Auth | `Authorization: Basic base64(user:pass)` — used for client_secret_basic. |
| client_secret_basic | Sending client_id+client_secret via Basic Auth header. |
| client_secret_post | Sending client_id+client_secret in POST body. |
| client_secret_jwt | Authenticating Client via HS256 JWT signed with client_secret. |
| private_key_jwt | Authenticating Client via RS256/ES256 JWT signed with private key. |
| tls_client_auth | mTLS-based Client authentication (RFC 8705). |
| consent screen | The IdP page where the user grants/denies scopes. |
| identity broker | A middleman IdP that federates to other IdPs. |
| OP-initiated logout | Logout triggered by the OP, propagated to RPs. |
| RP-initiated logout | Logout triggered by the Client. |
| Provider Configuration Information | The discovery JSON object. |
| Authentication Context | Environmental info about how the user authenticated (ACR/AMR). |
| ID Token at_hash | Hash of access token in the ID token, ties them together. |
| ID Token c_hash | Hash of the authorization code, used in hybrid flow. |
| hybrid flow | OIDC flow that returns ID token directly + code for token exchange. |
| OAuth 2.0 for Browser-Based Apps | BCP for SPAs (current draft). |
| OAuth 2.0 for First-Party Native Apps | BCP for native (current draft). |
| OAuth 2.0 Threat Model | RFC 6819, classic threat model document. |
| OAuth 2.0 Best Current Practice | Draft document with modern recommendations. |
| WebFinger | Protocol for discovering an OP given an email-style identifier. |
| Issuer Discovery | Looking up an OP from a user identifier (rare). |
| user-agent | The browser; OAuth uses this term often. |
| consent | The user's "yes" to the requested scopes. |
| concession | (Not OAuth) — but often confused with consent; ignore in this domain. |
| BFF | Backend-for-Frontend pattern: SPA talks to its own backend, which talks to IdP. |
| Resource Indicator | RFC 8707 `resource` parameter — request token for a specific resource. |
| Step-up Authentication | RFC 9470 — handle "we need stronger auth for this op." |
| OAuth Step-up Challenges | The standard way to bounce a request back for stronger auth. |

## Try This

**Experiment 1: Look at the discovery doc for a real IdP.**
Run `curl -s "https://accounts.google.com/.well-known/openid-configuration" | jq .` and look at every field. Find `authorization_endpoint`. Find `jwks_uri`. Find `scopes_supported`. Compare it to Microsoft's `https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration`. What's different? What's the same?

**Experiment 2: Decode a real JWT.**
If you have a real JWT (from your browser's developer tools after logging into a site with "Sign in with Google", or from any service you use), paste it into the decoder commands above. Look at `iss`, `aud`, `sub`, `exp`. Convert `exp` to a human-readable date with `date -r 1700000000` (macOS) or `date -d @1700000000` (Linux).

**Experiment 3: Run a local mock OIDC server.**
Use the `mock-oauth2-server` Docker image. Start it. Hit its `/.well-known/openid-configuration`. Use it to do a fake "Sign in with mock" flow with a tiny test app.

**Experiment 4: Generate PKCE values.**
Run the `code_verifier` / `code_challenge` commands above multiple times. Watch them change. Try to spot the pattern (there isn't one — they're random).

**Experiment 5: Watch JWKS rotation in real time.**
Save Google's JWKS today: `curl -s "$JWKS_URI" | jq '.keys[].kid' > kids-today.txt`. In a week or two, run it again and diff. You will see new kids appear and old ones disappear. That is rotation in action.

**Experiment 6: Build an authorization URL manually.**
Pick an IdP you have an account with (Google, GitHub, Auth0, etc). Register a test Client. Build the URL by hand using the Client ID. Visit it in a browser. See the consent screen. Approve. See the redirect with the code in it. Try to exchange the code with `curl`. Watch what works and what fails.

**Experiment 7: Trigger every error code.**
Use `curl` against the `/token` endpoint. Try wrong client_id (`invalid_client`), wrong code (`invalid_grant`), wrong scope (`invalid_scope`), expired refresh token (`invalid_grant`). Read the error responses.

**Experiment 8: Inspect a token's signature without verifying.**
Take a JWT. Slice off the third (signature) part. Try to decode it as base64url. You'll get raw bytes. Convert to hex with `xxd`. Notice it's deterministic-looking but unguessable — that's a good signature.

**Experiment 9: Try `alg: none` defense.**
Take a JWT. Decode the header. Replace `"alg": "RS256"` with `"alg": "none"`. Re-encode. Append a dot and an empty third part. Try to use it. A modern library should reject it. If it doesn't, file a bug.

**Experiment 10: Compare response_modes.**
For `response_type=code`, try `response_mode=query` (default) vs `response_mode=fragment` vs `response_mode=form_post`. Notice how the IdP delivers the code in different places: URL query, URL fragment, or HTTP form POST.

## Where to Go Next

- `cs auth oauth` — dense reference for OAuth flows, all RFCs, all error codes.
- `cs auth oidc` — dense reference for OIDC, ID Token claims, discovery, session management.
- `cs auth saml` — alternative federation protocol; XML-heavy, common in enterprise SSO.
- `cs detail auth/oauth` — formal token math, JOSE crypto deep dive.
- `cs ramp-up tls-eli5` — the encryption channel everything OAuth rides on.
- `cs ramp-up saml-eli5` — sibling sheet covering the XML-based federation world.

## See Also

- `auth/oauth`
- `auth/oidc`
- `auth/saml`
- `auth/ldap`
- `auth/kerberos`
- `security/tls`
- `security/pki`
- `security/cryptography`
- `ramp-up/tls-eli5`
- `ramp-up/saml-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 6749 — The OAuth 2.0 Authorization Framework
- RFC 6750 — OAuth 2.0 Bearer Token Usage
- RFC 6819 — OAuth 2.0 Threat Model and Security Considerations
- RFC 7009 — OAuth 2.0 Token Revocation
- RFC 7515 — JSON Web Signature (JWS)
- RFC 7516 — JSON Web Encryption (JWE)
- RFC 7517 — JSON Web Key (JWK)
- RFC 7518 — JSON Web Algorithms (JWA)
- RFC 7519 — JSON Web Token (JWT)
- RFC 7591 — OAuth 2.0 Dynamic Client Registration Protocol
- RFC 7592 — OAuth 2.0 Dynamic Client Registration Management Protocol
- RFC 7636 — Proof Key for Code Exchange (PKCE)
- RFC 7662 — OAuth 2.0 Token Introspection
- RFC 8252 — OAuth 2.0 for Native Apps (BCP)
- RFC 8414 — OAuth 2.0 Authorization Server Metadata
- RFC 8628 — OAuth 2.0 Device Authorization Grant
- RFC 8693 — OAuth 2.0 Token Exchange
- RFC 8705 — OAuth 2.0 Mutual-TLS Client Authentication
- RFC 9068 — JWT Profile for OAuth 2.0 Access Tokens
- RFC 9101 — JWT-Secured Authorization Request (JAR)
- RFC 9126 — OAuth 2.0 Pushed Authorization Requests (PAR)
- RFC 9207 — OAuth 2.0 Authorization Server Issuer Identification
- RFC 9396 — OAuth 2.0 Rich Authorization Requests (RAR)
- RFC 9449 — OAuth 2.0 Demonstrating Proof of Possession (DPoP)
- RFC 9470 — OAuth 2.0 Step-up Authentication Challenge Protocol
- RFC 9700 — OAuth 2.0 Security Best Current Practice
- OpenID Connect Core 1.0
- OpenID Connect Discovery 1.0
- OpenID Connect Dynamic Client Registration 1.0
- OpenID Connect Session Management 1.0
- OpenID Connect Front-Channel Logout 1.0
- OpenID Connect Back-Channel Logout 1.0
- OpenID Connect RP-Initiated Logout 1.0
- FAPI 1.0 Baseline / Advanced
- FAPI 2.0 Security Profile
- "OAuth 2 in Action" — Justin Richer & Antonio Sanso (Manning, 2017)
- "Identity and Data Security for Web Development" — Jonathan LeBlanc (O'Reilly)
- IETF OAuth Working Group: https://datatracker.ietf.org/wg/oauth/about/
- OpenID Foundation: https://openid.net/
