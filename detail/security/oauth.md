# The Mathematics of OAuth — Protocol State Machines and Cryptographic Proof Channels

> *OAuth 2.0 is a distributed state machine where authorization decisions propagate through cryptographic proof channels. PKCE transforms a public client's vulnerability into a commitment scheme, and token lifecycle follows information-theoretic entropy bounds.*

---

## 1. Authorization Code Flow as State Machine (Automata Theory)

### Protocol States

The OAuth 2.0 authorization code flow is a deterministic finite automaton with states:

$$Q = \{S_0, S_1, S_2, S_3, S_4, S_5, S_{\text{err}}\}$$

| State | Description | Holder |
|:---|:---|:---|
| $S_0$ | Initial (unauthenticated) | Client |
| $S_1$ | Authorization request sent | Auth Server |
| $S_2$ | User authenticated, consent granted | Auth Server |
| $S_3$ | Authorization code issued | Client |
| $S_4$ | Token exchange complete | Client |
| $S_5$ | Resource access granted | Resource Server |
| $S_{\text{err}}$ | Error / denied | Any |

### Transition Function

$$\delta: Q \times \Sigma \rightarrow Q$$

$$\delta(S_0, \text{initiate}) = S_1$$
$$\delta(S_1, \text{authenticate}) = S_2$$
$$\delta(S_2, \text{consent}) = S_3$$
$$\delta(S_3, \text{exchange}) = S_4$$
$$\delta(S_4, \text{api\_call}) = S_5$$
$$\delta(S_i, \text{error}) = S_{\text{err}} \quad \forall i$$

The protocol is a linear chain with error edges from every state, making it a simple path graph with failure transitions.

---

## 2. PKCE as Commitment Scheme (Cryptographic Theory)

### Commitment Phase

PKCE (Proof Key for Code Exchange) implements a hash-based commitment scheme:

**Setup:** Client generates random verifier $v$:

$$v \xleftarrow{R} \{A\text{-}Z, a\text{-}z, 0\text{-}9, \text{-}, ., \_, \sim\}^{[43,128]}$$

**Commit:** Client computes challenge $c$:

$$c = \text{BASE64URL}(\text{SHA-256}(v))$$

The commitment has two properties:

**Hiding:** Given $c$, finding $v$ requires inverting SHA-256:

$$\Pr[\mathcal{A}(c) = v] \leq 2^{-256}$$

**Binding:** Finding $v' \neq v$ such that $\text{SHA-256}(v') = \text{SHA-256}(v)$:

$$\Pr[\text{collision}] \leq \frac{1}{2^{128}} \quad \text{(birthday bound)}$$

### Verification Phase

At token exchange, the auth server verifies:

$$\text{BASE64URL}(\text{SHA-256}(v_{\text{presented}})) \stackrel{?}{=} c_{\text{stored}}$$

### Security Reduction

An attacker intercepting the authorization code learns $(\text{code}, c)$ but not $v$. To forge the token request, they must find $v$ from $c$, which is the preimage resistance problem:

$$\text{Advantage}_{\text{attacker}} \leq \text{Adv}^{\text{preimage}}_{\text{SHA-256}}(t) \leq \frac{t}{2^{256}}$$

Where $t$ is the number of hash evaluations. For $t = 2^{80}$ (computationally feasible):

$$\text{Advantage} \leq 2^{80-256} = 2^{-176} \approx 0$$

---

## 3. Token Entropy and Lifetime (Information Theory)

### Access Token Entropy Requirements

For bearer tokens, the minimum entropy prevents brute-force guessing:

$$H(T) \geq 128 \text{ bits}$$

For a token of length $\ell$ characters from alphabet $|\Sigma|$:

$$H = \ell \cdot \log_2 |\Sigma|$$

| Token Type | Length | Alphabet Size | Entropy |
|:---|:---:|:---:|:---:|
| Opaque (hex) | 32 | 16 | 128 bits |
| Opaque (base64url) | 22 | 64 | 132 bits |
| Opaque (base64url) | 43 | 64 | 258 bits |
| UUID v4 | 36 | 16 | 122 bits |

### Brute-Force Resistance

Expected attempts to guess a valid token from a pool of $N$ valid tokens in a space of size $S$:

$$E[\text{attempts}] = \frac{S}{N} = \frac{2^{H}}{N}$$

For $H = 128$ bits and $N = 10^6$ active tokens:

$$E[\text{attempts}] = \frac{2^{128}}{10^6} \approx 3.4 \times 10^{32}$$

At $10^9$ attempts/second: $\approx 10^{16}$ years.

### Token Lifetime Optimization

The optimal access token lifetime balances security and usability:

$$L_{\text{optimal}} = \arg\min_L \left[ \alpha \cdot P(\text{theft}) \cdot L + \beta \cdot \frac{1}{L} \right]$$

Where:
- $\alpha \cdot P(\text{theft}) \cdot L$ = expected damage from stolen token (proportional to lifetime)
- $\beta / L$ = user friction cost (inversely proportional to lifetime)

Taking the derivative and setting to zero:

$$L_{\text{optimal}} = \sqrt{\frac{\beta}{\alpha \cdot P(\text{theft})}}$$

Industry practice: $L_{\text{access}} = 300\text{s to } 900\text{s}$, $L_{\text{refresh}} = 86400\text{s to } 2592000\text{s}$.

---

## 4. Refresh Token Rotation (Markov Chains)

### Token State Transitions

Refresh token rotation creates a Markov chain of token states:

$$T_0 \xrightarrow{\text{use}} T_1 \xrightarrow{\text{use}} T_2 \xrightarrow{\text{use}} \cdots$$

Each $T_i$ is valid only once. The state transition:

$$P(T_{i+1} | T_i) = 1 \quad \text{(deterministic rotation)}$$
$$P(T_i \text{ valid after use}) = 0 \quad \text{(immediate invalidation)}$$

### Theft Detection

If token $T_i$ is stolen and both attacker and legitimate user attempt to use it:

$$P(\text{detection}) = 1 - P(\text{attacker uses before user})$$

With rotation, the first use of $T_i$ invalidates it. The second use triggers the theft detection:

$$\text{detection event}: \text{use}(T_i) \text{ when } T_i \text{ already consumed}$$

Response: invalidate entire token family $\{T_0, T_1, \ldots, T_n\}$.

### Token Family Tree

$$\text{family}(T_0) = \{T_i : T_i = \text{rotate}^i(T_0)\}$$

Any reuse of a consumed token in the family triggers revocation of all descendants.

---

## 5. Scope as Access Control Matrix (Access Control Theory)

### Scope Algebra

OAuth scopes form a set algebra over resource-action pairs:

$$\text{scope} \subseteq \mathcal{R} \times \mathcal{A}$$

Where $\mathcal{R}$ = resources and $\mathcal{A}$ = actions. Common scope patterns:

$$\text{api.read} = \{(r, \text{read}) : r \in \text{API}\}$$
$$\text{api.write} = \{(r, \text{write}) : r \in \text{API}\}$$
$$\text{api.admin} = \{(r, a) : r \in \text{API}, a \in \mathcal{A}\}$$

### Scope Reduction

Token exchange can only maintain or reduce scope (monotonic restriction):

$$\text{scope}(T_{\text{access}}) \subseteq \text{scope}(T_{\text{authorization}}) \subseteq \text{scope}(\text{consent})$$

This forms a chain:

$$S_{\text{consented}} \supseteq S_{\text{requested}} \supseteq S_{\text{granted}} \supseteq S_{\text{refreshed}}$$

---

## 6. CSRF Protection via State Parameter (Probability Theory)

### State Parameter Entropy

The `state` parameter prevents CSRF by binding the authorization request to the session:

$$\text{state} = \text{HMAC-SHA256}(\text{session\_id}, \text{random\_nonce})$$

An attacker must guess the state value:

$$P(\text{CSRF success}) = \frac{1}{2^{|\text{state}|}}$$

For 256-bit state:

$$P(\text{CSRF}) = 2^{-256} \approx 0$$

### OIDC Nonce

The `nonce` parameter binds the ID token to the authorization request:

$$\text{nonce} \xleftarrow{R} \{0,1\}^{256}$$

Verification: $\text{nonce}_{\text{id\_token}} \stackrel{?}{=} \text{nonce}_{\text{stored}}$

This prevents ID token replay across sessions.

---

## 7. Token Introspection Latency (Queueing Theory)

### Introspection vs Local Validation

| Method | Latency | Freshness | Model |
|:---|:---|:---|:---|
| Introspection (remote) | $\mu^{-1} + d$ | Real-time | M/M/1 queue |
| JWT validation (local) | $O(1)$ | Stale by $\leq L$ | None |
| Hybrid (cache + introspect) | $O(1)$ avg | Stale by $\leq C$ | TTL cache |

Where $\mu^{-1}$ is server processing time and $d$ is network latency.

### Introspection Request Rate

For $N$ resource servers each handling $\lambda$ requests/second:

$$\Lambda_{\text{introspection}} = N \cdot \lambda \cdot (1 - p_{\text{cache}})$$

With cache hit rate $p_{\text{cache}}$ and token TTL $C$:

$$p_{\text{cache}} = 1 - \frac{1}{\lambda \cdot C}$$

For $\lambda = 1000$ req/s and $C = 60$s:

$$p_{\text{cache}} = 1 - \frac{1}{60000} \approx 99.998\%$$

---

## Prerequisites

finite-automata, cryptographic-hash-functions, information-theory, probability, markov-chains

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| PKCE challenge generation | $O(1)$ — single SHA-256 | $O(1)$ — 32 bytes |
| Token generation (opaque) | $O(1)$ — CSPRNG | $O(1)$ — token length |
| JWT validation (local) | $O(n)$ — n = claims count | $O(1)$ — key cached |
| Token introspection (remote) | $O(1) + \text{RTT}$ | $O(1)$ |
| Scope intersection | $O(\min(\|S_1\|, \|S_2\|))$ | $O(\|S_1 \cap S_2\|)$ |
| Refresh token rotation | $O(1)$ — generate + invalidate | $O(k)$ — k = family size |
