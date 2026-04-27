# HashiCorp Vault — ELI5

> Vault is a bank vault for your secrets. Apps and people prove who they are at the door, Vault checks an access list, and hands them a short-lived withdrawal slip good for limited operations.

## Prerequisites

(none — start here)

This sheet is the very first stop for understanding HashiCorp Vault. You do not need to know what a "secret" is. You do not need to know what a "database password" is. You do not need to know what "encryption" is. By the end of this sheet you will know all of those things in plain English, and you will have typed real Vault commands into a real terminal and watched a real vault open and close.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## Plain English

### Imagine a real bank vault

Picture a real bank. The kind of bank with the giant round metal door, the door with the spinning wheel handle, the door that takes ten people pulling at the same time to swing it open. Inside that vault are safety deposit boxes. Inside the safety deposit boxes are people's important things. Wills. Jewelry. Old photographs. Stock certificates. Things that, if a stranger got hold of them, could ruin a life.

Now imagine that this bank does not just hold things. It hands them out. People walk in. People prove who they are by showing a driver's license, or by saying a password, or by putting their thumb on a fingerprint reader. Once the bank knows who they are, the bank looks them up on a list. The list says, "Alice can open box 17 and box 18. Alice cannot touch any other box. Bob can open box 23 only." If Alice asks for box 23, the bank says no. If Alice asks for box 17, the bank goes back, opens box 17, takes out what is inside, gives it to Alice, and writes down in a logbook that Alice took it.

That is Vault.

The bank vault is the program called Vault. The safety deposit boxes are called **secrets**. The drivers' licenses and passwords are called **auth methods**. The list of who can open what is called **policies**. The logbook is called the **audit log**. And the small piece of paper that says "yes, this person is allowed in for the next hour" is called a **token**.

That is the whole idea. Everything else in this sheet is detail.

### Why do we need a vault at all?

Programs need passwords. Lots of passwords. Your photo-sharing website needs the password to its database. Your email server needs the password to its mail server. Your video game server needs the key to talk to the credit-card company. Your weather app needs an API key to ask the weather service for forecasts. Every program is full of secrets.

For a long time, programmers wrote those secrets directly into their code. They typed `password = "hunter2"` into a file and called it done. This was a terrible idea. The secret got copied every time someone copied the code. The secret got into backups. The secret got pushed to GitHub. The secret got emailed around. By the time you tried to change the password, you had no idea how many copies were out there. If a single laptop got stolen, you had to change every password on every server, and you would never be sure you had got them all.

Then programmers tried environment variables. They took the password out of the code and put it in a thing called an environment variable, which is a little label your computer holds in memory while a program runs. This was slightly better. The secret was no longer in the code. But it was still on the disk somewhere. It was still in the launch script. It was still in the deployment configuration. It was still backed up. Anyone who could read the disk could read the password.

Then programmers tried encrypted files. They put the secret in a file but scrambled it with another password. This just moved the problem. Now you needed somewhere to store the password that scrambled the file. Where? In another file? Encrypted with another password? Where does it stop?

Vault is where it stops. Vault is the one place. Vault holds every secret. Programs do not store secrets. Programs go to Vault and ask for them, and Vault hands them out one at a time, and writes down every hand-out in a logbook, and the secret goes away when the program is done with it.

### A second picture: Vault as a hotel concierge

Imagine you are staying at a fancy hotel. There is a concierge at the desk. The concierge has a special drawer of room keys, restaurant reservations, parking tickets, and gift envelopes. Guests walk up to the concierge and ask for things. "I want to get into my room." "I want a table at the restaurant." "I want my car from the parking lot."

The concierge does not just hand things out to anybody. The concierge checks. "What is your room number? Show me your wristband. Yes, I see you are room 412. Here is a key card, but it only works on room 412, and only for the next thirty minutes. After that come back and I will print you a new one."

The concierge writes everything down. "At 9:42, room 412 asked for a key. At 10:15, room 412 returned the key." If anything goes missing, the hotel can look at the logbook and know exactly who had a key when.

That is Vault. The concierge is the Vault server. The wristband is your authentication. The temporary key card is a Vault **token**. The fact that the key card stops working in thirty minutes is called a **lease**. The logbook is the **audit log**.

### A third picture: Vault as the locksmith who never gives out the master key

Imagine a building with a hundred doors. Every door has a different lock. The owner of the building does not want to walk around with a hundred keys, and the owner does not trust anyone with the master key, because if you lose the master key the whole building has to be re-keyed.

So the owner hires a locksmith. The locksmith lives in a tiny shed outside the building. The locksmith owns the master key, but the master key never leaves the shed. When somebody needs to get into a door, they knock on the shed. They prove who they are. The locksmith takes the master key, walks over, opens the door, and walks back. The visitor never sees the master key. The visitor only sees the door open.

Vault is the locksmith. The master key never leaves Vault. Programs never get to hold the database password — they ask Vault for the database password, Vault hands them a copy that lasts one hour, and after that hour the copy stops working. Or, even better, Vault makes up a brand-new database username and password just for that program, just for that hour, and at the end of the hour Vault deletes the username from the database.

This is called **dynamic secrets**, and it is the most magical thing Vault does. We will come back to it.

### A fourth picture: Vault as a paranoid librarian

Imagine a librarian who is very, very paranoid. The library is locked up tight. When the librarian goes home for the night, the library locks itself, and every book inside disappears into a hidden vault that even the librarian cannot open alone. The librarian could not steal a book even if the librarian wanted to.

In the morning, five trusted citizens of the town arrive. Each of them carries a piece of a key. No single person has the whole key. They have to all stand at the door together and put their pieces together to open the vault. Once the vault is open, the books come back, and the library can run as normal.

This is called the **seal/unseal ceremony**, and it is one of the most distinctive features of Vault. When Vault starts up, it is **sealed**. Sealed means it cannot read its own data. The encryption key is not in memory. To wake Vault up, several humans have to come together with their key shares and unseal it. Until that happens, Vault refuses to do anything except tell you "I am sealed, please come back after the unsealing ceremony."

We will spend a long time on the seal/unseal ceremony later in the sheet. It is weird the first time you see it. Once you understand it, it is one of the most beautiful security ideas in the whole world of computers.

### Why so many pictures?

Just like the kernel, Vault is invisible. You cannot see Vault working. You only see the doors opening and the secrets coming out. Different pictures help different parts of your brain understand what Vault is doing.

The **bank vault** picture is best for understanding that Vault is the one place secrets live.

The **hotel concierge** picture is best for understanding tokens and leases.

The **locksmith** picture is best for understanding dynamic secrets.

The **paranoid librarian** picture is best for understanding the seal/unseal ceremony.

If a picture is not clicking, switch to another one.

## What Even Is Vault

Let's get more concrete. Vault is a program. It runs on a computer (or several computers — we'll get to that). It listens for connections from other programs. The other programs send requests. Vault sends back answers. That is all it does, mechanically.

But the requests it answers fall into a few categories.

### "Who are you?" requests

These are called **authentication** requests. A program or a person walks up and says "I am Alice, here is my password." Vault checks the password against its records. If the password is right, Vault gives back a piece of paper called a **token**. The token has an expiration date. The token says "the bearer of this paper is Alice, until 5pm." From now on, Alice does not have to keep typing the password. Alice just hands over the token.

Vault has many ways to ask "who are you?". A username and password is one. A cloud computer's identity is another (Vault can recognize that you are an AWS computer with a particular tag). A Kubernetes pod identity is another. A GitHub login is another. A certificate is another. We will list all of them in the **Auth Methods** section.

### "Give me a secret" requests

Once you have a token, you can ask for a secret. "Give me the database password." Vault checks the rules. The rules say which tokens can ask for which secrets. If your token is allowed, Vault hands the secret over. If not, Vault says "permission denied" and writes the failed attempt in the logbook.

### "Make me a new secret" requests

This is the locksmith trick. You ask Vault, "I need a database username and password to talk to the customer database." Vault, instead of handing you a password somebody typed in last week, walks over to the database, makes up a brand-new username, gives that username permission to read the customer table, and hands you the new credentials. Those credentials are good for one hour. At the end of that hour, Vault walks back to the database and deletes the username. The username never existed before you asked, and it will never exist again.

This is called a **dynamic secret**. It is the difference between Vault and a password manager. A password manager stores passwords. Vault stores passwords *and* makes brand-new ones on demand.

### "Encrypt this for me" requests

The last big category is encryption-as-a-service. You can ask Vault to scramble a piece of data without ever telling you the key. You hand Vault a credit card number. Vault scrambles it and hands back a meaningless string of characters. You store the string in your database. Later, you hand the string back to Vault, and Vault unscrambles it. You never had the key. Even if your database gets stolen, the strings are useless without Vault. This is called the **transit engine**, and we will come back to it.

### Where Vault came from

A company called HashiCorp wrote Vault, and first released it in 2015. HashiCorp also wrote Terraform (which builds infrastructure) and Consul (which keeps track of services) and Nomad (which runs containers).

In 2023, HashiCorp changed Vault's license from open-source (Mozilla Public License 2.0) to a "Business Source License" (BUSL). This made some people unhappy, because BUSL is not a true open-source license — you cannot use Vault to compete with HashiCorp commercially. So a group of people forked Vault, and called the fork **OpenBao**. OpenBao is now hosted by the Linux Foundation. The command-line tool for OpenBao is called `bao`. It works almost exactly like the `vault` command. Most things in this sheet apply to both Vault and OpenBao.

In 2024, IBM announced it was buying HashiCorp. This made some people even more nervous about whether Vault would stay friendly to small users. OpenBao continues to grow as the open-source path forward. If you want to be safe, you can use OpenBao and you will be fine. The concepts are the same. The commands are nearly identical.

## The Storage Backends

Vault has to put your secrets somewhere. Where it puts them is called the **storage backend**. Vault supports many storage backends. We will look at the important ones.

### Raft (integrated storage)

This is the recommended choice in 2025 and later. **Raft** is a way for several copies of Vault to agree on the same data. You run three or five copies of Vault on three or five computers. They talk to each other. They make sure they all have the same secrets, and they all agree on the latest version. If one computer dies, the others keep going. If two computers die (in a five-computer cluster), the other three keep going. Only if you lose more than half do you stop.

Raft is "integrated" because Vault holds the data itself. You do not need any other program. Just Vault. This is the easiest way to run Vault in production.

```
   raft cluster (3 nodes)
   +----------+    +----------+    +----------+
   | vault-1  |<-->| vault-2  |<-->| vault-3  |
   |  LEADER  |    | follower |    | follower |
   +----------+    +----------+    +----------+
        ^                ^                ^
        |                |                |
        +----------------+----------------+
                client requests
        (always go to the leader, which
         replicates to followers, which
         must majority-acknowledge before
         the write is considered durable)
```

The **leader** does the writing. The **followers** copy whatever the leader writes. If the leader dies, the followers vote and one of them becomes the new leader. Raft was invented to be easier to understand than the previous algorithm (Paxos). Vault has used Raft as a built-in option since version 1.4.

### Consul

**Consul** is another HashiCorp product. Before Raft was integrated, Consul was the recommended way to run a clustered Vault. You would run Consul on several computers, and Vault would store its data in Consul. Consul itself uses Raft internally — so this was Raft, just with an extra program in the way.

Consul is still supported, and lots of older Vault clusters use it. If you are starting fresh in 2025, use integrated Raft instead. One fewer program to babysit.

### File backend

The **file backend** stores Vault's data in a directory on disk. It is dead simple. It is also single-node — you cannot have two copies of Vault sharing one directory. So this is only useful for tiny deployments, demos, or development. Do not run a real production system on a file backend.

```
storage "file" {
  path = "/opt/vault/data"
}
```

### In-memory (dev mode)

The **in-memory backend** stores Vault's data in RAM. Nothing is written to disk. When Vault stops, every secret is gone. This sounds useless, but it is the foundation of **dev mode**, which is how you learn Vault. In dev mode, Vault starts unsealed, with one root token printed to your terminal, and it forgets everything when you Ctrl-C.

```
$ vault server -dev
==> Vault server configuration:
             Api Address: http://127.0.0.1:8200
                     Cgo: disabled
         Cluster Address: https://127.0.0.1:8201
   ...
You may need to set the following environment variables:
    $ export VAULT_ADDR='http://127.0.0.1:8200'
The unseal key and root token are reproduced below in case you
want to seal/unseal the Vault or re-authenticate.
Unseal Key: 5VLE+...
Root Token: hvs.CAESI...
```

This is the easiest way to play with Vault. We will use it in the **Hands-On** section.

### Cloud object storage

You can also store Vault data in:

- **AWS S3** (`storage "s3"`). High-availability is not built in — combine with a separate locking mechanism if you need HA.
- **Azure Blob Storage** (`storage "azure"`).
- **Google Cloud Storage / GCS** (`storage "gcs"`).

These are useful when you cannot run a real server cluster but you trust a cloud provider. They are slower than Raft. Use Raft if you can.

There are even more backends — DynamoDB, PostgreSQL, MySQL, etcd, ZooKeeper. They mostly exist for historical reasons. The HashiCorp recommendation, written down in big letters, is: use integrated Raft.

## Seal/Unseal Ceremony

This is the part of Vault that surprises everyone the first time. We will go slow.

### Why is Vault sealed at all?

When Vault starts up, it has its data on disk (or in Raft, or in Consul, or wherever). The data is encrypted. Vault cannot read its own encrypted data unless somebody hands Vault the key.

Why? Because the operator does not want the key to live on disk next to the data. If a thief steals the server, the thief gets the encrypted data and the key in the same heist. That is no better than not encrypting at all.

So Vault keeps the key in memory only. When Vault is running, the key is in RAM. When Vault is restarted, the key is gone, and Vault has to be told the key again before it can do anything. Until then, Vault is **sealed**. A sealed Vault returns 503 Service Unavailable to almost every request, except for a couple of "tell me about your seal status" requests.

Telling Vault the key is called **unsealing**.

### Shamir's secret sharing — the splitting trick

If only one person knew the unseal key, that person could be coerced. They could be threatened, they could be bribed, they could just lose the key. So Vault uses a math trick called **Shamir's secret sharing**, named after a mathematician called Adi Shamir.

Shamir's trick lets you take one secret (the unseal key) and split it into N pieces. Each piece is meaningless on its own. You also pick a number K, called the **threshold**. As long as K of the N pieces are put back together, the original secret can be reconstructed. With fewer than K pieces, you have nothing.

A common setup is N=5 and K=3. You hand out five pieces (key shares) to five trusted operators. To unseal the vault, any three of them have to come together. Two cannot. Four can. If one operator quits, four still have keys, so unsealing still works. If two operators quit, three still have keys, so unsealing still works. If three quit, you are stuck — but you knew that going in.

```
 unseal ceremony (Shamir, 3-of-5)

 +---------+   +---------+   +---------+   +---------+   +---------+
 | share 1 |   | share 2 |   | share 3 |   | share 4 |   | share 5 |
 | Alice   |   | Bob     |   | Carol   |   | Dan     |   | Eve     |
 +----+----+   +----+----+   +----+----+   +----+----+   +----+----+
      |             |             |
      v             v             v
   +-----------------------------------+
   |  vault operator unseal (3 times)  |
   +-----------------------------------+
                     |
                     v
              [reconstructed master key]
                     |
                     v
              [decrypt the unseal key]
                     |
                     v
              [decrypt the encryption key]
                     |
                     v
              [vault is open!]
```

There are actually two layers of keys in here. The thing the operators reconstruct is called the **master key** (older docs) or the **unseal key**. That key is then used to decrypt the actual **encryption key**, which is the key Vault uses to read and write its data. Why two layers? Because if you ever want to change the unseal-share scheme (say, go from 3-of-5 to 4-of-7), you only have to re-encrypt the unseal key, not all of Vault's data. This is called **rekeying**.

### Auto-unseal

Asking three humans to unseal the vault every time the server reboots is great in theory. In practice, in production, with thousands of restarts, it gets old quickly. So Vault supports **auto-unseal**.

With auto-unseal, instead of human shares, Vault asks a cloud Key Management Service (KMS) to do the unsealing. The cloud KMS holds a key that Vault never sees. Vault hands the encrypted key to the cloud KMS. The cloud KMS decrypts it. Vault uses the result. Restart Vault, and Vault asks again, and unsealing happens automatically.

The cloud KMS becomes the new "vault for your vault." This is fine, as long as you trust the cloud provider with that key.

Supported auto-unseal services:

- **AWS KMS** (most common)
- **Google Cloud KMS**
- **Azure Key Vault**
- **OCI Vault** (Oracle Cloud)
- **Alibaba KMS**
- **HashiCorp's own Transit engine** — yes, you can use one Vault to auto-unseal another Vault. This is called **transit auto-unseal**, and it lets you keep auto-unseal entirely on your own infrastructure.

When you use auto-unseal, you do not have unseal shares anymore. Instead, you have **recovery keys**. Recovery keys cannot unseal Vault on a normal day (the cloud KMS does that). But if a few specific operations need extra approval — like generating a new root token, or rekeying — Vault asks for the recovery threshold of recovery keys. Same Shamir trick, different name.

### What seal looks like in practice

```
$ vault status
Key                     Value
---                     -----
Seal Type               shamir
Initialized             true
Sealed                  true
Total Shares            5
Threshold               3
Unseal Progress         0/3
Unseal Nonce            n/a
Version                 1.16.0
Storage Type            raft
HA Enabled              true
```

`Sealed: true` means Vault cannot do anything yet. `Unseal Progress 0/3` means we have given it zero of the three needed shares.

```
$ vault operator unseal
Unseal Key (will be hidden):
Key                Value
---                -----
Seal Type          shamir
Initialized        true
Sealed             true
Total Shares       5
Threshold          3
Unseal Progress    1/3       <- one share down, two to go
```

Three operators take turns running `vault operator unseal`. After the third one, `Sealed` flips to `false`, and Vault is open for business.

### Sealing on purpose

You can also seal Vault on purpose. If you suspect a break-in, you can run `vault operator seal` and Vault will immediately drop its keys from RAM and refuse all further requests until the unseal ceremony happens again. This is the panic button.

## Auth Methods

Now that Vault is unsealed, programs and people can ask "who are you?" questions. The answer to that question depends on which **auth method** you are using. Each auth method is a different way to prove identity.

You enable auth methods at paths. The default path matches the method name. So `userpass` lives at `auth/userpass/`, and `kubernetes` lives at `auth/kubernetes/`. You can change paths if you want two different LDAP servers, for example.

### token

The simplest auth method. You already have a token, and you use it to do things. Every Vault request carries a token in the `X-Vault-Token` header. The token method is always enabled. It is the foundation of everything else.

### userpass

A username and a password, like a website login. Easy to understand, easy to set up, but not as secure as the more advanced methods. Good for humans logging in by hand.

```
$ vault auth enable userpass
$ vault write auth/userpass/users/alice password=hunter2 policies=devs
$ vault login -method=userpass username=alice
Password (will be hidden):
Key                    Value
---                    -----
token                  hvs.CAESIN5...
token_accessor         9hX...
token_duration         768h
token_renewable        true
token_policies         ["default" "devs"]
identity_policies      []
policies               ["default" "devs"]
token_meta_username    alice
```

### AppRole

The classic auth method for **machines** (programs, not people). AppRole gives an application two pieces of information:

- **role_id** — like a username. Stored with the application.
- **secret_id** — like a password. Often delivered to the application at boot time and used once.

The application sends both pieces back to Vault, gets a token, and from then on uses the token to read secrets. AppRole is a building block: you can pair it with response wrapping and "secret_id is one-time-use" to get a pretty secure delivery story even with no fancy cloud identity.

A common pattern: a deployment system pre-creates a `role_id`, then asks Vault for a fresh `secret_id` wrapped in a one-time wrapping token, and hands the wrapping token to the application. The application unwraps it once and gets the `secret_id`. If anybody else tries to unwrap the same wrapping token, Vault knows somebody intercepted the message and refuses.

### Kubernetes

If your programs run in Kubernetes, every pod has a service account, and every service account comes with a JWT (a signed identity token). Vault's Kubernetes auth method takes the JWT, asks the Kubernetes API "is this a real JWT, and is this pod really who it says it is?", and if Kubernetes says yes, Vault hands the pod a Vault token tied to the role you assigned to that service account. No usernames, no passwords, no shared secrets.

This is the gold-standard way to run Vault in a Kubernetes cluster. We will see it in **Vault Agent** below.

### AWS IAM

If your program runs on an EC2 instance or as an IAM-authenticated workload, Vault can verify the signed AWS request and hand back a token. Same idea as Kubernetes — your cloud already proves identity, Vault trusts that proof. Two flavors: `iam` (signed sts:GetCallerIdentity request) and `ec2` (instance identity document). `iam` is more flexible.

### GCP

The Google Cloud equivalent. Vault verifies signed JWTs from Google's metadata server.

### Azure

The Azure equivalent. Vault verifies managed identity tokens from Azure's metadata service.

### JWT/OIDC

Vault accepts a JWT signed by an outside identity provider. OIDC (OpenID Connect) is the standardized version of JWT login for human browsers — Vault redirects your browser to your company's SSO provider (Okta, Auth0, Google Workspace, GitHub, Keycloak), the SSO provider authenticates you, and gives Vault a JWT it can verify. This was added in Vault 1.4 and is now the standard way to log humans in.

### GitHub

Authenticate by handing Vault a GitHub personal access token. Vault asks GitHub "is this token valid, and what teams is the user in?" and maps GitHub teams to Vault policies. Useful for small teams running internal tools, less useful for production secrets — GitHub goes down, your auth goes down.

### LDAP

The classic enterprise directory protocol. Vault accepts a username and password, asks the LDAP server (Active Directory, FreeIPA, OpenLDAP) to verify, and looks up the user's group memberships to map to Vault policies. Anyone who has used a corporate workstation has used LDAP without knowing it.

### Cert (mTLS)

Authenticate by presenting a TLS client certificate. Vault checks the certificate against a trusted CA list and lets you in if the cert matches. Great for service-to-service auth in environments where everyone already has certificates.

### Okta

Direct integration with Okta — Vault accepts a username and password and forwards the auth attempt (including push MFA) to Okta. Slightly tighter than the OIDC method.

### RADIUS

A very old protocol mostly used for VPN and Wi-Fi auth. Vault can use a RADIUS server as the authoritative identity source. Rarely used today outside specific telecom or military environments.

### Choosing an auth method

| Caller          | Recommended method      |
| --------------- | ----------------------- |
| Human, browser  | OIDC (via Okta/Google)  |
| Human, terminal | userpass or OIDC        |
| Pod in K8s      | Kubernetes              |
| EC2 instance    | AWS IAM                 |
| Azure VM        | Azure                   |
| Other server    | AppRole + response wrap |
| Service mesh    | Cert (mTLS)             |

## Identity

Vault has a concept on top of auth methods called **identity**. Identity lets you say "Alice the person is the same Alice whether she logged in via userpass on her laptop or via OIDC on her browser." Without identity, Vault would see two completely separate logins. With identity, both logins are aliases of one shared **entity**, and policy decisions can be made about the entity.

The pieces:

- **Entity** — a real-world person or service. Has a unique ID inside Vault.
- **Alias** — one specific way that entity logs in. Alice has a userpass alias and an OIDC alias.
- **Group** — a collection of entities. Maps to one or more policies. Can also map to "external groups" pulled from LDAP/OIDC.

You almost never create entities by hand. Vault creates an entity automatically the first time someone logs in via any auth method. You then go and merge the aliases together so Vault knows "this OIDC alias and this userpass alias are the same human."

```
                             +------------+
                             |  Group:    |
                             |  devs      |
                             +-----+------+
                                   |
                  +----------------+---------------+
                  |                                |
            +-----+------+                  +------+-----+
            | Entity:    |                  | Entity:    |
            | alice      |                  | bob        |
            +-+--------+-+                  +-+--------+-+
              |        |                      |        |
       +------+--+  +--+------+        +------+--+  +--+------+
       | alias:  |  | alias:  |        | alias:  |  | alias:  |
       | userpass|  | oidc    |        | userpass|  | github  |
       | "alice" |  | (sub..) |        | "bob"   |  | "bob42" |
       +---------+  +---------+        +---------+  +---------+
```

## Secrets Engines

If auth methods are the front door, **secrets engines** are the rooms inside. Each secrets engine offers a different kind of secret. You enable secrets engines at paths, just like auth methods.

### KV v1 vs v2

The most common secrets engine. KV stores arbitrary key-value pairs at arbitrary paths. There are two versions and you should always use v2 unless you have a reason not to.

- **KV v1** — store one value per path. Writing replaces. Deletion is permanent. No history.
- **KV v2** — versioned. Every write creates a new version. Reads default to the latest. Old versions can be retrieved or rolled back. Deletion soft-deletes (you can undelete). Destroy is permanent. Includes per-path metadata.

KV v2 is also slightly different in how you talk to it. The path you use as a human (`secret/myapp`) is rewritten internally to `secret/data/myapp`. The `vault kv` subcommand hides this for you. If you go to the HTTP API directly, you have to know about the `data` and `metadata` prefixes.

```
$ vault secrets enable -path=secret kv-v2
Success! Enabled the kv-v2 secrets engine at: secret/

$ vault kv put secret/myapp foo=bar baz=qux
============= Secret Path =============
secret/data/myapp

======= Metadata =======
Key                Value
---                -----
created_time       2026-04-27T12:00:00.123Z
custom_metadata    <nil>
deletion_time      n/a
destroyed          false
version            1

$ vault kv get secret/myapp
==== Secret Path ====
secret/data/myapp

======= Metadata =======
Key                Value
---                -----
created_time       2026-04-27T12:00:00.123Z
deletion_time      n/a
destroyed          false
version            1

==== Data ====
Key    Value
---    -----
baz    qux
foo    bar
```

### Database

Generates dynamic credentials for databases. Postgres, MySQL, Mongo, Cassandra, MSSQL, Oracle, Redis, Snowflake, Elasticsearch, and a long tail of others. You configure Vault with a single admin connection, and Vault uses that admin connection to make up new short-lived users on demand.

### AWS

Generates dynamic IAM credentials. You ask Vault, Vault asks AWS to create a new IAM user (or assume a role), Vault hands you the access key, and at the end of the lease the user is deleted (or the assumed role expires).

### GCP

Same idea for GCP service accounts.

### Azure

Same idea for Azure service principals.

### PKI/CA

Vault becomes a certificate authority. You ask for a certificate, Vault issues one, signed by a CA Vault holds. We will spend a section on this.

### SSH

Vault can issue short-lived SSH certificates (using OpenSSH's certificate format), or one-time SSH passwords (OTPs). Your developers no longer carry permanent SSH keys around. They run `vault ssh user@host`, get a one-time credential, and use it for that one connection.

### Transit

Encryption-as-a-service. Vault holds the keys, you hand it data to encrypt or decrypt, and you never see the keys. We will spend a section on this too.

### Transform (Enterprise)

Format-preserving encryption (FPE), masking, tokenization. The encrypted credit card number still looks like a credit card number. The masked SSN still looks like an SSN. Useful when downstream systems can only handle data that looks like the original. Enterprise only.

### KMIP (Enterprise)

KMIP is a standard protocol for talking to enterprise key managers. Vault speaks KMIP, so storage arrays and database engines that already speak KMIP can use Vault as their key manager. Enterprise only.

### Cubbyhole

Cubbyhole is the personal locker. Every token gets its own private cubbyhole. The cubbyhole goes away when the token goes away. Cubbyhole is used internally for response wrapping (more on that later) and is occasionally useful for handing one secret from one process to another.

## Static vs Dynamic Secrets

This is one of the most important distinctions in the entire Vault world.

### Static secret

A static secret is one you typed in. You put your AWS key into KV. You put your database password into KV. Vault stores it. Programs ask for it. Vault gives it back. The same value every time, until you change it. The value is the same to every caller.

Static secrets are fine for things you cannot dynamically generate (third-party API keys, license keys, the fixed credentials your auditor gave you). Static secrets do not give you the locksmith trick.

### Dynamic secret

A dynamic secret is one Vault makes up at request time. You ask for a database credential, Vault calls the database admin API, makes up a brand-new user with a random password, gives that user the right SQL grants, and hands you the username/password. At the end of the lease, Vault deletes the user.

Every caller gets a different username. Every caller can be tracked separately in the database log. If a credential leaks, you do not have to change the master password for every program in the company — you revoke the lease, the user vanishes, and nobody else is affected.

Dynamic secrets are the reason Vault is more than a password manager. If you only ever use Vault for static secrets, you are getting a fraction of the value.

## Lease Management

Every secret Vault hands out comes with a **lease**. The lease has:

- A **lease_id** — a unique string identifying this particular hand-out.
- A **lease_duration** — how long it is good for.
- A **renewable** flag — whether you can extend it.

Tokens have leases. Database credentials have leases. AWS credentials have leases. PKI certificates have a kind of pseudo-lease (they have an expiration date but Vault does not strictly track it as a lease unless you ask). KV secrets typically do not have leases — they are static and last until you delete them.

```
$ vault read database/creds/app-role
Key                Value
---                -----
lease_id           database/creds/app-role/H8r2N0yXwY...
lease_duration     1h
lease_renewable    true
password           A1Q3qH-4fkXd...
username           v-userpass-app-role-AbCdEf-1735325000
```

That `lease_id` is your handle on the credential. With it you can:

- **Look it up:** `vault lease lookup <lease_id>` — see when it was issued, when it expires.
- **Renew it:** `vault lease renew <lease_id>` — extend the lease (up to the max_ttl).
- **Revoke it:** `vault lease revoke <lease_id>` — kill it now. Database user vanishes immediately.

You can also revoke an entire prefix. `vault lease revoke -prefix database/creds/app-role` revokes every lease ever issued by that role. Useful when you change the application and need to rotate everything at once.

Two important TTLs apply to every lease:

- **default_lease_ttl** — what you get if you do not ask for a specific TTL.
- **max_lease_ttl** — the absolute ceiling. No matter how many times you renew, the lease cannot live past this.

A lease can be renewed up to `max_lease_ttl` from when it was originally issued. Once you hit the ceiling, renewal fails and the credential is gone.

## Policies

A policy in Vault is a list of rules. Each rule says "for this path, you can do these things." Policies are written in HashiCorp Configuration Language (HCL), which looks like a slightly nicer JSON.

### Capabilities

There are eight capabilities you can grant on a path:

- **read** — read the secret at this path.
- **list** — see what paths exist under this prefix.
- **create** — create a new secret here (where it didn't exist before).
- **update** — change an existing secret here.
- **delete** — soft-delete (in KV v2) or permanent-delete (KV v1).
- **sudo** — perform a privileged operation. A few endpoints require sudo.
- **deny** — explicitly forbid. Deny beats every other capability.
- **root** — root token capability. Can do anything. Does not actually appear in policies; only the root token has it.

### A simple policy

```hcl
# devs.hcl — devs can read and update their own app's secrets

path "secret/data/myapp/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "secret/metadata/myapp/*" {
  capabilities = ["read", "list"]
}

# devs can lease database credentials but not configure the engine
path "database/creds/app-role" {
  capabilities = ["read"]
}
```

### Denying

Deny is non-negotiable. If a token has both `read` and `deny` on a path, it is denied. Use deny when you want to make absolutely sure a sub-path is off-limits even though a parent path is allowed.

```hcl
path "secret/data/myapp/*" {
  capabilities = ["read", "list"]
}

path "secret/data/myapp/admin/*" {
  capabilities = ["deny"]
}
```

### Templating

Policies can include templates so each entity gets paths scoped to themselves.

```hcl
path "secret/data/users/{{identity.entity.name}}/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

When Alice (entity name "alice") logs in and reads `secret/data/users/alice/notes`, the policy substitutes `alice` and allows. When Alice tries to read `secret/data/users/bob/notes`, the substitution does not match `bob` and Alice is denied.

### Beyond ACL: Sentinel (Enterprise)

In open-source Vault, policies are pure ACLs. In Vault Enterprise, there are also **Sentinel policies** (called RGP for Role Governing Policy and EGP for Endpoint Governing Policy). Sentinel can express conditions like "only allow this between 9am and 5pm" or "only if MFA was used in the last 5 minutes." OpenBao does not include Sentinel.

## The PKI Engine

This is one of the most powerful and least-used parts of Vault. The PKI engine turns Vault into your private certificate authority.

### What is a CA?

A **certificate authority** signs certificates. A certificate is a piece of paper that says "this server is really `app.internal`, signed by us, the CA." Browsers and other programs trust certificates if they trust the CA that signed them. In the public web, a small set of CAs (Let's Encrypt, DigiCert, Sectigo) sign certificates that all browsers trust by default.

For internal services, you do not want to use a public CA — your services are not on the internet, and many public CAs cost money. You want your own private CA. You give all your servers a copy of your CA's public key, and from then on every signed certificate is trusted inside your network.

### Building a CA in Vault

```
$ vault secrets enable pki
Success! Enabled the pki secrets engine at: pki/

$ vault secrets tune -max-lease-ttl=87600h pki
Success! Tuned the secrets engine at: pki/

$ vault write pki/root/generate/internal \
    common_name="Internal CA" \
    ttl=87600h
Key              Value
---              -----
certificate      -----BEGIN CERTIFICATE-----...
expiration       1893456000
issuing_ca       -----BEGIN CERTIFICATE-----...
serial_number    7a:b1:...
```

That `87600h` is ten years. Your CA cert is now good for ten years. You probably want to copy it out, give it to all your servers as a trusted root, and never look at it again.

### Roles

Vault never lets anyone ask for any certificate they want. Instead you create **roles** that say "this role is allowed to issue certificates with these constraints."

```
$ vault write pki/roles/server-role \
    allowed_domains="internal" \
    allow_subdomains=true \
    max_ttl=72h
```

Now anyone with permission to use `pki/issue/server-role` can issue a certificate, but the common name must end in `.internal`, and the certificate cannot live longer than 72 hours. The 72-hour ceiling is great — your servers automate certificate renewal, and the certificates rotate so often that a stolen cert is useless within days.

### Issuing a certificate

```
$ vault write pki/issue/server-role \
    common_name=app.internal \
    alt_names=app2.internal \
    ttl=72h
Key                  Value
---                  -----
certificate          -----BEGIN CERTIFICATE-----...
issuing_ca           -----BEGIN CERTIFICATE-----...
private_key          -----BEGIN RSA PRIVATE KEY-----...
private_key_type     rsa
serial_number        4c:1d:...
```

Hand the certificate and private key to your application. The app uses them for TLS until the 72 hours are up, then asks again.

### Intermediates

In a real world, you do not want your root CA to sign every server certificate directly. You want an **intermediate CA** that signs server certificates day-to-day, while the root CA stays offline. Vault can be the intermediate, with the root living in some other secure place (or another Vault). The signing flow is:

1. New Vault generates an intermediate CSR.
2. CSR is sent to the root CA.
3. Root CA signs the CSR and sends back the intermediate certificate.
4. Vault imports the signed intermediate.

After this, Vault issues server certs using the intermediate, and clients trusting the root automatically trust everything Vault issues.

### CRL and OCSP

When a certificate is bad — server compromised, key leaked — you need to tell the world. Vault publishes a **certificate revocation list** (CRL) at `pki/crl`. It can also speak **OCSP**, a protocol where clients ask "is this specific cert revoked?" instead of downloading the whole list. PKI is huge. The Vault PKI engine handles all the standard pieces.

Newer versions of the PKI engine added **EAB** (External Account Binding) for ACME-style integrations, **issuer rotation**, and **cluster_aia_path** for AIA (Authority Information Access) URLs. The engine has been one of the most heavily developed parts of Vault from 1.11 through 1.16.

## The Database Engine

If you use Vault for one thing, use it for this. The database engine is the locksmith trick.

### Configuring a database connection

You start by giving Vault a single admin connection. The admin user must be able to create and drop other users.

```
$ vault secrets enable database
Success! Enabled the database secrets engine at: database/

$ vault write database/config/postgres-prod \
    plugin_name=postgresql-database-plugin \
    connection_url="postgresql://{{username}}:{{password}}@db.internal:5432/postgres?sslmode=require" \
    allowed_roles="app-role,readonly-role" \
    username="vault-admin" \
    password="kept-very-secret"
```

The `{{username}}` and `{{password}}` placeholders are filled in by Vault on every connection — Vault is the only thing that ever sees the admin credentials.

### Creating a role

A role describes the SQL Vault should run when somebody asks for credentials.

```
$ vault write database/roles/app-role \
    db_name=postgres-prod \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
                         GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    revocation_statements="REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM \"{{name}}\"; \
                           DROP ROLE IF EXISTS \"{{name}}\";" \
    default_ttl="1h" \
    max_ttl="24h"
```

Now any caller with permission to read `database/creds/app-role` will get a freshly created Postgres user with a random password and a one-hour TTL. At the end of the TTL Vault drops the role.

### Reading credentials

```
$ vault read database/creds/app-role
Key                Value
---                -----
lease_id           database/creds/app-role/V0bJ4m8...
lease_duration     1h
lease_renewable    true
password           A1Q3qH-4fkXd-r6gN
username           v-userpass-app-role-AbCdEf-1735329600
```

The application takes the username and password and connects. One hour later, either it renews the lease (and gets to keep using the same credential a bit longer) or the credential expires and the app fetches a new one. This is what people mean when they say Vault gives you "credentials per-instance, per-hour."

### Static roles

You can also have **static roles** — Vault rotates a fixed user's password on a schedule. Useful for legacy systems where you cannot create users on demand.

```
$ vault write database/static-roles/legacy-role \
    db_name=postgres-prod \
    username=legacy_app \
    rotation_period=24h
```

The `legacy_app` user is real and pre-existing. Vault rotates its password every 24 hours, and applications that need it can read `database/static-creds/legacy-role` to get the current password.

### Supported databases

A partial list: PostgreSQL, MySQL, MariaDB, MongoDB, MSSQL, Oracle, Cassandra, Couchbase, Elasticsearch, HanaDB, InfluxDB, Snowflake, Redis (Enterprise plugin), Redshift. Each has its own plugin (`postgresql-database-plugin`, `mysql-database-plugin`, etc.).

## The Transit Engine

The transit engine is encryption-as-a-service. You hand Vault data, Vault hands back ciphertext, you store the ciphertext anywhere. To get the data back, you hand the ciphertext to Vault, and Vault hands the data back. You never had the key.

### Why is this useful?

Imagine your application stores customer credit card numbers. You could store them in plaintext, but if your database leaks, every card is exposed. You could encrypt them with a key in your application — but then every copy of the application has the key, and an attacker who steals one server gets the key. With transit, the application has no key. The application calls Vault, gets ciphertext, stores ciphertext. An attacker with the database has nothing useful.

### Encrypting

```
$ vault secrets enable transit
$ vault write -f transit/keys/myapp
$ vault write transit/encrypt/myapp plaintext=$(echo -n 'hello world' | base64)
Key            Value
---            -----
ciphertext     vault:v1:abc123def456...
key_version    1
```

`vault:v1:abc123...` is the ciphertext. `v1` is the key version — Vault can rotate the key, and every ciphertext remembers which version it was encrypted with so old data can still be decrypted.

### Decrypting

```
$ vault write transit/decrypt/myapp ciphertext=vault:v1:abc123def456...
Key          Value
---          -----
plaintext    aGVsbG8gd29ybGQ=    # base64 for "hello world"
```

### Rotating

```
$ vault write -f transit/keys/myapp/rotate
$ vault read transit/keys/myapp
Key                       Value
---                       -----
keys                      map[1:1735320000 2:1735406400]
latest_version            2
min_decryption_version    1
```

New writes go through key version 2. Old ciphertext (`vault:v1:...`) still decrypts because version 1 is still in the map. To upgrade old ciphertext, you ask Vault to rewrap it:

```
$ vault write transit/rewrap/myapp ciphertext=vault:v1:abc123...
Key            Value
---            -----
ciphertext     vault:v2:...
```

Now you have the same plaintext but encrypted with version 2. Vault never decrypted it for you — rewrap is one operation, the plaintext stays inside Vault.

### Other transit operations

- **`transit/sign/myapp`** — sign a message with a transit key (asymmetric keys only).
- **`transit/verify/myapp`** — verify a signature.
- **`transit/hmac/myapp`** — HMAC a message.
- **`transit/datakey/plaintext/myapp`** — get a fresh data encryption key (DEK) and its wrapped form. Useful for envelope encryption: encrypt big things with the DEK locally, store the wrapped DEK with the data.
- **`transit/random`** — get cryptographically strong random bytes from Vault's RNG.

### Convergent encryption

By default, encrypting the same plaintext twice gives different ciphertext (good — protects against an attacker spotting that two records have the same value). But sometimes you need the opposite — searching an encrypted column requires the same plaintext to encrypt to the same ciphertext. **Convergent encryption** mode does that, at a cost: identical inputs produce identical outputs, so an attacker can spot patterns. Use convergent only when you have to.

### BYOK

Vault 1.10 added **bring-your-own-key** (BYOK) for transit, letting you import an externally-generated key instead of having Vault generate one for you. Useful for compliance regimes that require key generation in HSMs.

## Performance Standby vs Performance Replication

This part is Enterprise-only, but worth knowing about.

### Performance standby (a.k.a. read-only standby)

In an integrated Raft cluster, only the leader handles writes. By default, only the leader handles reads too — the followers are pure backups. With **performance standby**, followers handle reads, while writes still go to the leader. This scales reads without doing anything special on the client side.

### Performance replication (Enterprise)

In a multi-region setup, you may have one Vault cluster per region. **Performance replication** lets the secondary region serve reads locally and forward writes to the primary. Each region has near-zero read latency. Replication is asynchronous — writes take a moment to reach all secondaries.

```
                     +----------------+
                     |   PRIMARY      |
                     |   us-east      |
                     |   reads+writes |
                     +-------+--------+
                             |
              +--------------+---------------+
              |                              |
     +--------v--------+            +--------v--------+
     |   SECONDARY     |            |   SECONDARY     |
     |   us-west       |            |   eu-west       |
     |   reads only    |            |   reads only    |
     |   forward writes|            |   forward writes|
     +-----------------+            +-----------------+
```

## Disaster Recovery (DR Replication)

**DR replication** is different from performance replication. A DR secondary is a hot standby that does not serve any traffic at all — it just keeps a synced copy of the primary's state. If the primary cluster fails, you promote the DR secondary, and it takes over.

A common production setup uses both: performance replication for read scale-out, DR replication for "the whole region went down, fail over to the other region."

## Audit Devices

Vault writes a logbook. Every request, every response, every authentication, every secret read. The logbook is called the **audit log**, and it is written by **audit devices**.

### Enabling

Audit is opt-in. By default, no audit log is written. As soon as your Vault is more than a toy, enable at least one audit device.

```
$ vault audit enable file file_path=/var/log/vault/audit.log
Success! Enabled the file audit device at: file/
```

### Devices

- **file** — append every entry to a file as JSON-per-line. Good with `logrotate`.
- **syslog** — send entries to syslog (rsyslog, journald). Useful when you have a central log pipeline.
- **socket** — send entries to a TCP/UDP socket. Useful when you have a custom log shipper.

### What is in the log?

Every request, every response. The fields are HMAC'd by default — Vault hashes paths, parameters, and tokens with an internal HMAC key so you can correlate entries without leaking the actual values. If a security incident requires you to see the real value at a particular timestamp, you can ask Vault to compute the HMAC of a candidate value and compare it to the log.

If audit fails (disk full, syslog not responding), Vault refuses to respond to the in-flight request. This is intentional. A Vault that cannot log access cannot serve access. If you make a critical Vault depend on audit, plan accordingly — at least two audit devices, ideally backed by independent disks.

## Vault Agent

Calling Vault from every program in your fleet is a hassle. Every program needs to know its credentials, manage its token lifecycle, watch for token expiration, refetch secrets on rotation. **Vault Agent** is a sidecar that does all that for you.

You run the agent on the same machine (or pod, or container) as your application. The agent does three things:

1. **Auto-auth** — log in to Vault using one of the auth methods, and keep the resulting token alive (renewing automatically).
2. **Sink** — write the current token (or a wrapped form of it) to a file or a Unix socket so the application can pick it up.
3. **Template** — render templated configuration files using Vault data, and re-render them when the data changes.

```
        +--------------------+
        |   your application  |
        +-------^------------+
                |
                | reads /etc/myapp/conf.json
                |
        +-------+------------+
        |    Vault Agent     |   (sidecar)
        |  - auto-auths      |
        |  - keeps token live |
        |  - renders config   |
        |  - caches secrets   |
        +-------^------------+
                |
                | TLS to Vault server
                |
        +-------+------------+
        |   Vault server     |
        +--------------------+
```

### A simple agent config

```hcl
# agent.hcl

pid_file = "/var/run/vault-agent.pid"

vault {
  address = "https://vault.internal:8200"
}

auto_auth {
  method "approle" {
    config = {
      role_id_file_path   = "/etc/vault/role_id"
      secret_id_file_path = "/etc/vault/secret_id"
      remove_secret_id_file_after_reading = false
    }
  }

  sink "file" {
    config = {
      path = "/run/vault/token"
    }
  }
}

template {
  source      = "/etc/vault/db.tmpl"
  destination = "/etc/myapp/db.json"
  command     = "systemctl reload myapp"
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address     = "127.0.0.1:8100"
  tls_disable = true
}
```

The `template` block reads a template file, renders it with Vault data, and runs `systemctl reload myapp` whenever the rendered file changes. The `cache` block lets the application talk to the agent on `127.0.0.1:8100` and have requests proxied (and cached) to Vault.

### Templates

A template uses Go's `text/template` syntax with helpers from the `consul-template` library:

```
{
  "host": "db.internal",
  "username": "{{ with secret "database/creds/app-role" }}{{ .Data.username }}{{ end }}",
  "password": "{{ with secret "database/creds/app-role" }}{{ .Data.password }}{{ end }}"
}
```

Every time the lease nears expiration, the agent fetches a new credential, re-renders the file, and runs the reload command. Your application does not have to know Vault exists — it just keeps reading `/etc/myapp/db.json`.

### Vault Agent Injector (Kubernetes)

In Kubernetes, you do not even configure the agent yourself. The **agent injector** is a Kubernetes admission webhook that watches for pods with annotations like:

```
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "myapp"
  vault.hashicorp.com/agent-inject-secret-db.json: "database/creds/app-role"
  vault.hashicorp.com/agent-inject-template-db.json: |
    {
      "username": "{{ with secret "database/creds/app-role" }}{{ .Data.username }}{{ end }}",
      "password": "{{ with secret "database/creds/app-role" }}{{ .Data.password }}{{ end }}"
    }
```

When such a pod is scheduled, the injector adds a Vault Agent sidecar that handles auto-auth (using Kubernetes auth method) and renders the template into a shared volume. Your container reads `/vault/secrets/db.json`. Done.

### Persistent caching

Vault Agent can persist its cache to disk so a pod restart does not require a brand-new login (Vault 1.7+). Useful for slow-start services.

## The CLI vs API vs SDK

Vault has three faces.

### CLI

The `vault` (or `bao`) command-line tool. Wraps the HTTP API in friendly subcommands. Good for humans, for shell scripts, and for trying things out.

```
$ vault read database/creds/app-role
$ vault write secret/myapp foo=bar
$ vault list secret/
```

### HTTP API

Everything Vault does, it does over HTTP. The CLI is just a wrapper. You can hit Vault from `curl`, from your application's HTTP library, from anything.

```
$ curl --header "X-Vault-Token: $VAULT_TOKEN" \
       --request GET \
       https://vault:8200/v1/secret/data/myapp | jq
{
  "request_id": "...",
  "lease_id": "",
  "renewable": false,
  "lease_duration": 0,
  "data": {
    "data": {
      "foo": "bar"
    },
    "metadata": { ... }
  }
}
```

### SDK

Vault has official Go, Ruby, Python (`hvac`), .NET, JavaScript, and other SDKs. They wrap the HTTP API in idiomatic functions. If you are writing a long-lived application, prefer an SDK over hand-rolling HTTP.

## Sealed-Vault Recovery

This is the section nobody reads until they need it. Bookmark it.

### Vault won't unseal

If `vault operator unseal` keeps spinning at "unseal progress 0/3" and never accepts your shares, check:

1. Are you using the right shares? Vault stores share fingerprints; Wrong shares produce a generic error.
2. Is the storage backend reachable? An unsealing attempt that cannot reach Raft or Consul will fail in confusing ways.
3. Is the cluster healthy? In a Raft cluster, you must unseal each node. The leader needs its quorum of unsealed peers.

### Lost the unseal shares

If you lost more than `(N - K)` shares (lost more than `N - K = 5 - 3 = 2` of a 3-of-5 setup), you can no longer reach the threshold. There is no backdoor. Your data is gone. This is by design. Auto-unseal recovery keys are slightly better — the cloud KMS still holds the actual unseal key, so you only need recovery keys for special operations like generating a new root token.

### Lost the root token

If you have lost the root token but Vault is unsealed and otherwise healthy, you can run `vault operator generate-root` to make a new one. This requires the threshold of unseal (or recovery) shares. It is a deliberate ceremony; you cannot just type a command and get a new root.

### Vault keeps sealing on restart

Three causes:

1. **Out of memory.** The OS is killing Vault before it can unseal. Check `dmesg`.
2. **Wrong storage path.** Vault is seeing an empty backend, declares itself uninitialized, and refuses to start. Check that the storage path on disk is the same one it was last time.
3. **Auto-unseal failure.** The cloud KMS is not reachable, or its credentials have expired. Check the seal status output for the seal-type-specific error.

### Forensic recovery

If a bad operator changed something they should not have, the audit log is your friend. Run `vault read sys/internal/counters/activity` for usage trends. Run `vault list sys/leases/lookup` to see active leases. Replay the audit log to figure out what happened.

## Common Errors

Verbatim error messages and what they mean.

### `Error checking seal status: ... 503 Service Unavailable`

Vault is sealed. Run `vault status` to confirm. Then run `vault operator unseal` until it is open.

### `permission denied`

Your token does not have a policy that allows the operation you tried. Check `vault token lookup` to see your policies. Read your policies with `vault policy read <name>` to find the gap.

### `missing client token`

You did not pass `VAULT_TOKEN` (or `-token`) to your command, and you are not logged in. Run `vault login` or `export VAULT_TOKEN=...`.

### `* permission denied (post error)`

Same as plain `permission denied` but the request was a POST/PUT (write). Same fix.

### `Error initializing client: invalid character looking for beginning of value`

`VAULT_ADDR` is wrong, or it is pointing somewhere that is not a Vault server. Vault expected a JSON response but got HTML or nothing. Double-check the URL, scheme, and port.

### `Error making API request: 503 server error`

Vault is sealed, in maintenance, or the upstream service Vault talks to is unreachable. `vault status` first.

### `Vault is sealed`

Self-explanatory. Unseal it.

### `URL: PUT https://vault:8200/v1/...`
### `Code: 400. Errors:`
### `secret was not found`

You asked for a path that does not exist. Common with KV v2 — remember the path you write is `secret/myapp`, not `secret/data/myapp` (the CLI inserts `data/` for you, the API does not).

### `backend version mismatch`

You are talking to a KV v1 path with KV v2 commands (or vice versa). `vault secrets list -detailed` shows the version. Use the right command.

### `* route entry not found for path`

The secrets engine is not enabled at that path. Run `vault secrets list` to see what is mounted where.

### `license expired`

Vault Enterprise license has run out. Vault enters a degraded state. Renew the license.

### `Error: namespace does not exist`

Vault Enterprise has multi-tenant namespaces. You are pointing at one that has not been created. `vault namespace list` to see the existing ones.

### `This combined login token has multiple identities`

You are trying to merge entities in a way that does not make sense, typically because two aliases that should belong to the same entity are mapped to different entities and the system is confused. Use `vault list identity/entity/id` and `vault read identity/entity/id/<id>` to untangle.

### Other classics

- `failed to renew the lease: lease is not renewable` — the role's `renewable` flag is false. You must fetch a new credential each TTL.
- `failed to renew lease: lease expired` — too much time passed; lease is gone. Fetch a new one.
- `cluster is not active` — you are talking to a Raft follower that is not configured for performance standby. Hit the leader instead.

## Hands-On

You will need a terminal. We will install Vault, run it in dev mode, and walk through every category of operation. Each command's output will be roughly what you see; tokens and timestamps will differ.

### Install Vault (or OpenBao)

On macOS with Homebrew:

```
$ brew tap hashicorp/tap
$ brew install hashicorp/tap/vault
$ vault version
Vault v1.16.2
```

Or with OpenBao:

```
$ brew install openbao
$ bao version
OpenBao v2.0.2
```

### 1. Start a dev server

```
$ vault server -dev
==> Vault server configuration:
             Api Address: http://127.0.0.1:8200
                     Cgo: disabled
         Cluster Address: https://127.0.0.1:8201
              Listener 1: tcp (addr: "127.0.0.1:8200", cluster address: "127.0.0.1:8201", tls: "disabled")
               Log Level: info
                   Mlock: supported: false, enabled: false
           Recovery Mode: false
                 Storage: inmem
                 Version: Vault v1.16.2

==> Vault server started! Log data will stream in below:

WARNING! dev mode is enabled! In this mode, Vault runs entirely in-memory
and starts unsealed with a single unseal key. The root token is already
authenticated to the CLI, so you can immediately begin using Vault.

You may need to set the following environment variables:

    $ export VAULT_ADDR='http://127.0.0.1:8200'

The unseal key and root token are reproduced below in case you
want to seal/unseal the Vault or re-authenticate.

Unseal Key: 5VLE+JyOjXa...
Root Token: hvs.CAESINWh9...
```

In a second terminal:

```
$ export VAULT_ADDR='http://127.0.0.1:8200'
$ export VAULT_TOKEN='hvs.CAESINWh9...'  # paste from above
```

### 2. `vault status`

```
$ vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.16.2
Cluster Name    vault-cluster-...
Cluster ID      ...
HA Enabled      false
```

### 3. `vault operator init` (production-style)

In dev mode, Vault is already initialized. To see what initialization looks like for real, you would run something like:

```
$ vault operator init -key-shares=5 -key-threshold=3
Unseal Key 1: ...
Unseal Key 2: ...
Unseal Key 3: ...
Unseal Key 4: ...
Unseal Key 5: ...

Initial Root Token: hvs.AAAA...

Vault initialized with 5 key shares and a key threshold of 3. Please securely
distribute the key shares printed above. When the Vault is re-sealed,
restarted, or stopped, you must supply at least 3 of these keys to unseal it
before it can start servicing requests.
```

### 4. `vault operator unseal`

```
$ vault operator unseal
Unseal Key (will be hidden):
Key                Value
---                -----
Sealed             true
Unseal Progress    1/3
```

Repeat two more times.

### 5. `vault operator unseal -migrate`

Used when you change seal types (Shamir to auto-unseal, or vice versa). You unseal with the old shares while pointing at the new seal config; Vault re-encrypts the unseal key with the new mechanism.

### 6. Enable userpass and log in

```
$ vault auth enable userpass
Success! Enabled userpass auth method at: userpass/

$ vault write auth/userpass/users/alice password=hunter2 policies=devs
Success! Data written to: auth/userpass/users/alice

$ vault login -method=userpass username=alice
Password (will be hidden):
Key                    Value
---                    -----
token                  hvs.CAESI...
token_accessor         ...
token_duration         768h
token_renewable        true
token_policies         ["default" "devs"]
```

### 7. `vault token create`

```
$ vault token create -policy=devs -ttl=1h
Key                  Value
---                  -----
token                hvs.CAESI...
token_accessor       ...
token_duration       1h
token_renewable      true
token_policies       ["default" "devs"]
```

### 8. `vault token revoke`

```
$ vault token revoke hvs.CAESI...
Success! Revoked token (if it existed)
```

### 9. `vault token lookup`

```
$ vault token lookup
Key                 Value
---                 -----
accessor            ...
creation_time       1735329600
display_name        userpass-alice
entity_id           ...
expire_time         2026-05-29T12:00:00Z
explicit_max_ttl    0s
id                  hvs.CAESI...
policies            [default devs]
ttl                 766h12m45s
```

### 10. `vault auth list`

```
$ vault auth list
Path         Type        Accessor                  Description
----         ----        --------                  -----------
token/       token       auth_token_xxxxxxxx       token based credentials
userpass/    userpass    auth_userpass_xxxxxxxx    n/a
```

### 11. `vault auth enable` and `vault auth disable`

```
$ vault auth enable approle
Success! Enabled approle auth method at: approle/

$ vault auth disable approle
Success! Disabled the auth method (if it existed) at: approle/
```

### 12. Write a policy

```
$ cat > devs.hcl <<'EOF'
path "secret/data/myapp/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF

$ vault policy write devs ./devs.hcl
Success! Uploaded policy: devs
```

### 13. List and read policies

```
$ vault policy list
default
devs
root

$ vault policy read devs
path "secret/data/myapp/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

### 14. Enable the KV v2 secrets engine

```
$ vault secrets enable -path=secret kv-v2
Success! Enabled the kv-v2 secrets engine at: secret/
```

(In dev mode, this is enabled for you.)

### 15. Put and get a secret

```
$ vault kv put secret/myapp foo=bar baz=qux
============= Secret Path =============
secret/data/myapp

======= Metadata =======
Key                Value
---                -----
created_time       2026-04-27T12:00:00.123Z
custom_metadata    <nil>
deletion_time      n/a
destroyed          false
version            1

$ vault kv get secret/myapp
==== Secret Path ====
secret/data/myapp

======= Metadata =======
Key                Value
---                -----
created_time       2026-04-27T12:00:00.123Z
deletion_time      n/a
destroyed          false
version            1

==== Data ====
Key    Value
---    -----
baz    qux
foo    bar
```

### 16. JSON output

```
$ vault kv get -format=json secret/myapp
{
  "request_id": "...",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "data": {
      "baz": "qux",
      "foo": "bar"
    },
    "metadata": { ... }
  }
}
```

### 17. KV metadata and rollback

```
$ vault kv put secret/myapp foo=baz
$ vault kv put secret/myapp foo=qux
$ vault kv metadata get secret/myapp
========== Metadata ==========
Key                Value
---                -----
cas_required       false
created_time       2026-04-27T12:00:00Z
current_version    3
delete_version_after    0s
max_versions       0
oldest_version     0
updated_time       2026-04-27T12:01:00Z

====== Version 1 ======
   ...
====== Version 2 ======
   ...
====== Version 3 ======
   ...

$ vault kv rollback -version=1 secret/myapp
Key                Value
---                -----
created_time       2026-04-27T12:02:00Z
version            4
```

### 18. Secrets list

```
$ vault secrets list
Path          Type         Accessor              Description
----          ----         --------              -----------
cubbyhole/    cubbyhole    cubbyhole_xxxxxxxx    per-token private secret storage
identity/     identity     identity_xxxxxxxx     identity store
secret/       kv           kv_xxxxxxxx           key/value secret storage
sys/          system       system_xxxxxxxx       system endpoints used for control, policy and debugging
```

### 19. Enable the database engine

```
$ vault secrets enable database
Success! Enabled the database secrets engine at: database/
```

### 20. Configure a Postgres connection

```
$ vault write database/config/postgres-prod \
    plugin_name=postgresql-database-plugin \
    connection_url="postgresql://{{username}}:{{password}}@db.internal:5432/postgres?sslmode=require" \
    allowed_roles="app-role" \
    username="vault-admin" \
    password="kept-very-secret"
```

### 21. Define a database role

```
$ vault write database/roles/app-role \
    db_name=postgres-prod \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    default_ttl=1h \
    max_ttl=24h
```

### 22. Read dynamic credentials

```
$ vault read database/creds/app-role
Key                Value
---                -----
lease_id           database/creds/app-role/V0bJ4m8...
lease_duration     1h
lease_renewable    true
password           A1Q3qH-4fkXd-r6gN
username           v-userpass-app-role-AbCdEf-1735325000
```

### 23. Enable PKI

```
$ vault secrets enable pki
$ vault secrets tune -max-lease-ttl=87600h pki
$ vault write pki/root/generate/internal common_name="Internal CA" ttl=87600h
```

### 24. Define a PKI role and issue a cert

```
$ vault write pki/roles/server-role allowed_domains=internal allow_subdomains=true max_ttl=72h
$ vault write pki/issue/server-role common_name=app.internal alt_names=app2.internal ttl=72h
```

### 25. Enable transit and encrypt

```
$ vault secrets enable transit
$ vault write -f transit/keys/myapp
$ vault write transit/encrypt/myapp plaintext=$(echo -n 'hello' | base64)
Key            Value
---            -----
ciphertext     vault:v1:abc123...
key_version    1
```

### 26. Decrypt

```
$ vault write transit/decrypt/myapp ciphertext=vault:v1:abc123...
Key          Value
---          -----
plaintext    aGVsbG8=
```

### 27. Enable an audit device

```
$ vault audit enable file file_path=/var/log/vault/audit.log
Success! Enabled the file audit device at: file/

$ vault audit list
Path     Type    Description
----     ----    -----------
file/    file    n/a
```

### 28. Lease lookup, renew, revoke

```
$ vault lease lookup database/creds/app-role/abc123
$ vault lease renew database/creds/app-role/abc123
$ vault lease revoke database/creds/app-role/abc123
```

### 29. Namespaces (Enterprise)

```
$ vault namespace list
Keys
----
finance/
engineering/
```

### 30. Vault Agent

```
$ vault agent -config=agent.hcl
==> Vault Agent started! Log data will stream in below:
2026-04-27T12:30:00.000Z [INFO]  agent.auth.handler: starting auth handler
2026-04-27T12:30:00.001Z [INFO]  agent.auth.handler: authenticating
...
```

### 31. Print and use the current token

```
$ VAULT_TOKEN=$(vault print token) vault read sys/health
Key                            Value
---                            -----
initialized                    true
sealed                         false
standby                        false
performance_standby            false
replication_performance_mode   disabled
replication_dr_mode            disabled
server_time_utc                1735329600
version                        1.16.2
cluster_name                   vault-cluster-...
cluster_id                     ...
```

### 32. Manually seal Vault

```
$ vault operator seal
Success! Vault is sealed.
```

### 33. Take a Raft snapshot

```
$ vault operator raft snapshot save backup-2026-04-27.snap
$ vault operator raft snapshot restore backup-2026-04-27.snap   # to restore
```

## Common Confusions

Pairs of things that look the same but are not, and questions everyone gets wrong the first time.

### KV v1 vs v2 (versioning, soft-delete)

KV v1 paths look like `secret/myapp`. KV v2 paths look the same to a human but the API path is `secret/data/myapp`. The CLI hides this difference, but if you copy a path from a UI into a `curl` command, you have to know. KV v2 keeps every version, supports soft-delete and undelete, and has metadata. KV v1 has none of that. If you are still on KV v1 in 2026, migrate.

### Root token never expires (rotate carefully)

The root token has no TTL. It can do anything. You should generate it during initialization, use it to do bootstrap setup (enable auth, create the first admin policy, set up audit), then revoke it. If you need root again later, run `vault operator generate-root` and produce a fresh one. Do not leave a long-lived root token sitting on a laptop somewhere.

### Seal vs initialize

A Vault is **initialized** the first time it has ever been told to set itself up — you give it a number of key shares, it generates an unseal key, splits it, and starts running. After initialization, the Vault is **sealed** by default. Initialization happens once, ever, in the lifetime of a cluster. Sealing happens whenever the process restarts.

### What does Shamir's secret sharing actually mean?

It is a math trick. You pick a secret S. You pick a polynomial whose constant term is S, of degree (K-1). You evaluate the polynomial at N different points and hand each point out as a share. With any K shares you can interpolate the polynomial and recover S. With fewer than K shares you have no information at all about S — every value of S is equally consistent with the shares you have. The math is in a paper from 1979 by Adi Shamir.

### Auto-unseal does not require quorum but needs cloud KMS

If you use Shamir, you need K humans every time. If you use auto-unseal, you need the cloud KMS. The cloud KMS becomes a single point of failure. Plan for the case where the cloud KMS is unreachable — Vault will not unseal, and your apps cannot get secrets.

### AppRole role_id+secret_id are LIKE a username+password (not a "token")

People sometimes think AppRole gives you the token directly. It does not. role_id+secret_id is a credential you use to log in. You log in by passing them to `auth/approle/login`, and you get back a token. The token is what you use after that. AppRole credentials are long-lived; the token they produce is short-lived.

### The difference between policies and roles

A **policy** is what permissions a token has (what paths and capabilities). A **role** is the configuration of a particular auth method or secrets engine — which policies a logged-in user gets, what kind of dynamic credential to issue, what TTLs apply. Roles often *reference* policies. Policies do not reference roles. People mix the words up; pay attention.

### Orphan tokens vs renewable

A **renewable** token can have its TTL extended (up to its max TTL). A **non-renewable** token expires at TTL with no extension. An **orphan** token is one without a parent — its lifecycle is independent. Normally, when you create a child token from a parent, revoking the parent revokes the child too (the whole tree). Orphan tokens break that link. Most service tokens for long-running daemons are orphan and renewable.

### Lease vs token TTL

Tokens have a TTL. Leases (on dynamic secrets) have a TTL. They are tracked separately. Your token might expire while your database credential is still valid; the database credential might expire while your token is still valid. If your token expires you cannot ask Vault for anything new, but the credentials you already have keep working until their leases run out.

### What does response wrapping do?

Sometimes you want to hand a secret to a system that does not have its own auth, and you do not want the secret on disk in transit. Vault gives you a **wrapping token**: a one-time token that, when "unwrapped," reveals the secret. The wrapping token is small, single-use, and time-limited. You hand the wrapping token to the system. The system unwraps it, gets the secret, and the wrapping token is now invalid. If somebody else intercepted the wrapping token, the legitimate recipient's unwrap fails (because the token was already used) and you know there was an interception.

### One-time secret_id flow

A common AppRole pattern: the secret_id is one-time-use. The deployment system asks Vault for a one-time secret_id wrapped in a wrapping token. The deployment system passes the wrapping token to the application. The application unwraps once to get the secret_id, then logs in with role_id and secret_id, then has a token. The secret_id is now exhausted. Future logins for that role need fresh secret_ids.

### How does Vault Agent's sink keep tokens fresh?

Auto-auth logs in once and gets a token. The agent's renewer keeps that token alive by calling `auth/token/renew-self` periodically. The sink writes the current token to disk (or a socket) so applications can read it. If the renewer ever fails to renew (token expired, network broken), the agent goes back to auto-auth and logs in fresh. Applications keep reading the sink and pick up the new token automatically.

### Namespace inheritance (Enterprise)

In Vault Enterprise, **namespaces** are nested mini-Vaults. You can create `engineering/` and inside it `engineering/devops/` and inside that `engineering/devops/secrets/`. Auth, policies, and secrets engines are scoped to the namespace. A request for `engineering/devops/secret/myapp` is an entirely different secret from `secret/myapp` at the root. Tokens issued in a namespace can only do things in that namespace and its children, never higher up.

### The difference between identity entity and identity alias

An **entity** is the real-world person or service — Alice the human. An **alias** is one specific way that entity logs in — Alice's userpass account, Alice's OIDC account, Alice's GitHub account. Aliases belong to entities. When you write a policy that refers to `identity.entity.name`, that is the entity name (`alice`), not the alias name (which might be `alice@example.com` for OIDC).

### Why your AppRole logged in but the policies don't apply

If you create an AppRole role and forget to set `token_policies`, your tokens come back with only the `default` policy. The login itself works (Vault has no reason to reject it), but the resulting token cannot do much. Always set `token_policies` when you write the role:

```
$ vault write auth/approle/role/myapp \
    token_policies="devs" \
    secret_id_ttl=10m \
    token_ttl=1h \
    token_max_ttl=24h
```

If you forget, the symptom is "I can log in but every read returns permission denied." Run `vault token lookup` on your token; if it shows `policies: [default]`, the role is misconfigured.

### Cubbyhole vs KV

People sometimes use cubbyhole as a cheap KV. It works, but cubbyhole is **per-token**: when the token goes away, the cubbyhole goes away. Cubbyhole is mostly used internally for response wrapping. For "I want to store some data in Vault," use KV.

### Mount tuning

Every secrets engine and auth method has tunable parameters: default TTLs, max TTLs, audit-log filtering. You set them with `vault secrets tune` (or `vault auth tune`). Forgetting to tune is the cause of most "why is my token only good for 32 days when I asked for a year" complaints — the global default TTL is 32 days unless you tune the mount.

### Vault Agent caching vs templating

Caching is "act as a proxy and remember responses." Templating is "render a config file when secrets change." They are independent. You can use caching without templating (for client SDKs that don't want to manage tokens). You can use templating without caching (for static config files). Most agents do both.

### Read vs list

`read` lets you see the value at a path. `list` lets you see what paths exist under a prefix without seeing values. They are different capabilities; granting one does not grant the other. `vault list secret/` calls the API with a `LIST` method, not a `GET`.

## Vocabulary

| Term | Plain English |
| ---- | ------------- |
| Vault | The HashiCorp program that holds secrets and hands them out. |
| HashiCorp Vault | The full official name. |
| OpenBao | The open-source fork of Vault, hosted by the Linux Foundation. |
| `bao` | The OpenBao command-line tool, drop-in for `vault`. |
| vault server | A running Vault process. |
| dev mode | An ephemeral in-memory Vault for development; auto-unsealed, single root token. |
| init | The one-time ceremony to set up a brand-new Vault and produce unseal shares. |
| unseal | Tell Vault enough key material so it can decrypt its own data. |
| seal | Wipe Vault's keys from memory so it stops serving requests. |
| Shamir's secret sharing | Math trick to split a secret into N parts where any K can reconstruct it. |
| key share | One of the N pieces handed out to operators. |
| key threshold | The number K of shares required to unseal. |
| recovery key | Like an unseal share but used only for special operations under auto-unseal. |
| recovery threshold | How many recovery keys you need to perform special ops. |
| root token | The all-powerful token created at init. Should be revoked after bootstrap. |
| regenerate root | The ceremony to produce a new root token after the old one was revoked. |
| GenerateRoot | The API name for the regenerate-root ceremony. |
| auto-unseal | Have a cloud KMS unseal Vault automatically on startup. |
| AWS KMS unseal | Auto-unseal using AWS Key Management Service. |
| GCP KMS unseal | Auto-unseal using Google Cloud KMS. |
| Azure Key Vault unseal | Auto-unseal using Azure Key Vault. |
| transit unseal | Auto-unseal using another Vault's transit engine. |
| OCI KMS unseal | Auto-unseal using Oracle Cloud Vault/KMS. |
| raft | A consensus protocol used by Vault's integrated storage. |
| integrated storage | Vault's built-in Raft-backed storage (no Consul required). |
| Consul backend | Storing Vault's data in HashiCorp Consul. |
| file backend | Storing Vault's data in a directory on disk (single-node only). |
| in-memory backend | Storing Vault's data in RAM (dev mode). |
| S3 backend | Storing Vault's data in AWS S3. |
| Azure Blob backend | Storing Vault's data in Azure Blob Storage. |
| GCS backend | Storing Vault's data in Google Cloud Storage. |
| performance replication | Read-only secondary clusters that forward writes (Enterprise). |
| DR replication | A hot-standby secondary cluster for disaster recovery (Enterprise). |
| secondary | A non-primary cluster in a replicated setup. |
| primary | The cluster that owns all writes in a replicated setup. |
| paths-from-replica | A pattern where some paths are local to a secondary and never replicated. |
| license | Enterprise-only entitlement that unlocks namespaces, replication, etc. |
| namespace | A nested mini-Vault inside an Enterprise Vault. |
| Enterprise vs OSS | Vault Enterprise (paid) vs Vault Open Source (now BUSL); OpenBao is the OSS path. |
| auth method | A way for a caller to prove who they are (userpass, AppRole, etc). |
| secrets engine | A backend that produces or stores secrets (KV, database, PKI, transit). |
| secrets engine path | The mount point under which the engine answers requests. |
| mount | The act of enabling a secrets engine or auth method at a path. |
| accessor | A handle for an auth-method mount or a token, used in policies and revocation. |
| role | A configuration object that ties together an auth method or engine with policies and TTLs. |
| role_id | The "username" part of an AppRole credential. |
| secret_id | The "password" part of an AppRole credential. |
| response wrapping | Returning a wrapping token instead of the actual response payload. |
| wrap-ttl | How long a wrapping token is valid before it expires. |
| unwrap | Trade a wrapping token for the actual response inside it. |
| token | The credential a caller carries after authentication. |
| token TTL | How long a token is valid from now. |
| token max-TTL | The absolute ceiling on a token's lifespan, even with renewals. |
| token use-limit | Maximum number of uses a token has before it self-destructs. |
| renewable | Whether a token or lease can have its TTL extended. |
| periodic | A token type that, instead of an absolute max TTL, can be renewed forever every period. |
| batch token | A lightweight, non-persistent token (Vault 1.0+). |
| service token | The classic, persistent token type. |
| orphan token | A token with no parent, whose lifecycle is independent. |
| token role | A pre-defined template for creating tokens with specific policies and TTLs. |
| identity entity | A real-world subject (person or service) inside Vault. |
| identity alias | One specific login pathway tied to an entity. |
| identity group | A collection of entities; can map to policies. |
| group alias | The name an external system uses for a group, mapped to a Vault group. |
| external group | A group whose members come from an external auth method like LDAP/OIDC. |
| internal group | A group whose members are explicitly listed in Vault. |
| policy | A document granting capabilities on paths. |
| HCL | HashiCorp Configuration Language; the format of policy files. |
| capabilities | The actions a policy grants on a path: read, list, create, update, delete, sudo, deny. |
| path | A URL-style location inside Vault, like `secret/data/myapp`. |
| sudo capability | Required to call certain privileged endpoints. |
| deny capability | Forbids the action, beats every other capability on the path. |
| ACL | Access Control List; the open-source policy system. |
| EGP | Endpoint Governing Policy; a Sentinel policy on an endpoint (Enterprise). |
| RGP | Role Governing Policy; a Sentinel policy on a role (Enterprise). |
| Sentinel | HashiCorp's policy-as-code framework (Enterprise). |
| control group | An Enterprise feature that requires multiple approvals before a request runs. |
| KV v1 | The original key-value engine — single value per path, no history. |
| KV v2 | Versioned key-value engine — multiple versions, soft-delete, metadata. |
| kv put | Write a secret. |
| kv get | Read a secret. |
| kv list | List paths under a prefix. |
| kv metadata | Read a KV v2 path's metadata (versions, max_versions, etc). |
| kv rollback | Make an old version of a KV v2 secret the current one again. |
| kv destroy | Permanently destroy specific KV v2 versions. |
| kv undelete | Reverse a soft-delete in KV v2. |
| database secrets engine | Generates database credentials on demand. |
| dynamic credentials | Credentials Vault makes up at request time. |
| static roles | Vault rotates a fixed user's password on a schedule. |
| rotation_period | How often a static role's password is rotated. |
| max_lease_ttl | Maximum lease lifetime for a mount. |
| default_lease_ttl | Default lease lifetime for a mount. |
| PKI | Public Key Infrastructure; Vault's certificate authority engine. |
| root CA | The top of a certificate chain. |
| intermediate CA | A CA signed by another CA, used for day-to-day issuance. |
| issuer | A specific CA certificate used to sign new certs (PKI engine). |
| CRL distribution | The HTTP location where a Certificate Revocation List is published. |
| OCSP | Online Certificate Status Protocol; ask a server "is this cert revoked?". |
| AIA URLs | Authority Information Access; URLs in a cert pointing to its issuer's cert. |
| transit | Encryption-as-a-service engine. |
| encryption-as-a-service | A service that encrypts and decrypts on your behalf without exposing keys. |
| transit/keys/X | Path where a named encryption key X lives. |
| derived keys | Keys that are deterministically derived from a context, used for per-row encryption. |
| convergent encryption | Same plaintext always produces same ciphertext (transit feature, used carefully). |
| transit/datakey | Get a fresh data encryption key (DEK) for envelope encryption. |
| transit/random | Get cryptographically strong random bytes from Vault. |
| transit/hmac | Compute an HMAC of input using a transit key. |
| transit/sign | Sign a message using an asymmetric transit key. |
| transit/verify | Verify a signature against a transit key. |
| transit/rewrap | Re-encrypt ciphertext under the latest version of its key. |
| KMIP | Key Management Interoperability Protocol; Vault Enterprise speaks it. |
| Transform | Enterprise engine for FPE, masking, tokenization. |
| FPE | Format-Preserving Encryption — output looks like a real input format. |
| masking | Replacing parts of data with placeholders (Transform engine). |
| tokenization | Replacing data with random opaque tokens that can be reversed (Transform engine). |
| audit device | A backend that writes audit log entries (file, syslog, socket). |
| file audit | Audit device that appends JSON lines to a file. |
| syslog audit | Audit device that sends entries to syslog. |
| socket audit | Audit device that sends entries to a TCP/UDP socket. |
| hmac | Hashed Message Authentication Code; how the audit log obfuscates sensitive fields. |
| audit log obfuscation | The technique of HMAC'ing tokens and paths in the audit log. |
| lease | A handle on a dynamic secret with TTL and revocation rules. |
| lease_id | The unique identifier of a lease. |
| lease_duration | How long a lease is valid. |
| renewal | Extending the TTL of a lease (or token). |
| revocation | Killing a lease (or token) immediately. |
| lease lookup | Asking Vault about an existing lease. |
| sys/leases endpoints | The API endpoints under sys/leases for lease admin. |
| Vault Agent | A sidecar process that auto-auths, renders templates, and caches. |
| agent config | The HCL file describing how Vault Agent should run. |
| auto_auth | Vault Agent feature that logs in and keeps a token alive. |
| template | A file Vault Agent renders with secret values. |
| sink | A file or socket where Vault Agent writes the current token. |
| listener | A network endpoint Vault Agent or Vault server listens on. |
| cache | Vault Agent's in-memory store of recent responses. |
| persistent caching | Saving the agent's cache to disk to survive restarts. |
| response wrapping in agent | The agent can wrap responses before handing them to clients. |
| vault-k8s | The Kubernetes integration for Vault. |
| agent injector | A Kubernetes admission webhook that adds Vault Agent sidecars to pods. |
| secrets-injection annotations | Pod annotations that tell the injector which secrets to fetch. |
| CSI driver | Container Storage Interface driver that mounts secrets as volumes. |
| ESO | External Secrets Operator; a Kubernetes operator that pulls Vault secrets into Kubernetes Secrets. |
| BUSL | Business Source License — Vault's license since 2023. |
| MPL 2.0 | Mozilla Public License — Vault's old license, still on OpenBao. |
| HSM | Hardware Security Module; a tamper-resistant device that holds keys. |
| BYOK | Bring Your Own Key — import an externally-generated key. |
| envelope encryption | Encrypt data with a per-record DEK, encrypt the DEK with a master key. |
| EAB | External Account Binding — ACME pre-auth (PKI engine, Vault 1.14+). |
| ACME | Automatic Certificate Management Environment — Let's Encrypt's protocol. |
| service mesh | A network of sidecars (Istio, Consul, Linkerd) that handles service-to-service traffic. |
| transit BYOK | Importing externally-generated keys into the transit engine (Vault 1.10+). |
| Sentinel RGP | Role Governing Policy in Sentinel. |
| Sentinel EGP | Endpoint Governing Policy in Sentinel. |
| autopilot | Raft autopilot manages cluster health and dead-node cleanup (Vault 1.5+). |
| entropy augmentation | Mixing entropy from an HSM into Vault's RNG (Vault 1.7+). |
| license auto-rotation | Vault Enterprise can auto-fetch new licenses from HashiCorp (Vault 1.13+). |

## Try This

Five tiny exercises. Type each one. Make sure you understand the output before moving on.

### 1. Start dev mode and check status

```
$ vault server -dev &
$ export VAULT_ADDR='http://127.0.0.1:8200'
$ vault status
```

Look at the `Sealed` field. It should say `false`.

### 2. Write and read a secret

```
$ vault kv put secret/hello message="hello world"
$ vault kv get secret/hello
```

Now write a second version:

```
$ vault kv put secret/hello message="goodbye"
$ vault kv get -version=1 secret/hello
$ vault kv get secret/hello
```

Notice that the latest version is "goodbye" and version 1 still says "hello world."

### 3. Create a policy and a token

```
$ cat > readonly.hcl <<'EOF'
path "secret/data/hello" {
  capabilities = ["read"]
}
EOF
$ vault policy write readonly ./readonly.hcl
$ vault token create -policy=readonly
```

Use the new token:

```
$ VAULT_TOKEN=<the-new-token> vault kv get secret/hello
```

It works. Now try to write:

```
$ VAULT_TOKEN=<the-new-token> vault kv put secret/hello message=fail
```

You should get `permission denied`. The readonly policy did exactly what it said.

### 4. Encrypt and decrypt

```
$ vault secrets enable transit
$ vault write -f transit/keys/myapp
$ CIPHER=$(vault write -field=ciphertext transit/encrypt/myapp plaintext=$(echo -n "private message" | base64))
$ echo "$CIPHER"
vault:v1:abcdef...
$ vault write -field=plaintext transit/decrypt/myapp ciphertext="$CIPHER" | base64 -d
private message
```

You handed Vault data, you got back a token, you handed the token back, and Vault gave you the original data. You never saw the key.

### 5. Watch the audit log

```
$ mkdir -p /tmp/vault-audit
$ vault audit enable file file_path=/tmp/vault-audit/audit.log
$ vault kv get secret/hello
$ tail -1 /tmp/vault-audit/audit.log | python3 -m json.tool
```

Read the JSON entry. Notice that the path is HMAC'd. Notice the timestamp, the request and response objects, the source. This is the logbook.

## Where to Go Next

Once this sheet feels comfortable, here are good next stops:

1. **`security/vault`** — the operator-level cheat sheet for Vault. Heavier on flags, configuration, and ops gotchas.
2. **`secrets/sops`** — a different way to manage secrets, file-based, designed to live in Git. Useful in places where running a full Vault is overkill.
3. **`security/age`** — a modern file encryption tool that pairs well with sops.
4. **`secrets/gopass`** — a per-user password manager that uses GPG and Git.
5. **`security/pki`** — the underlying world of certificate authorities.
6. **`security/cryptography`** — the math behind it all.
7. **`security/tls`** — what TLS is and how Vault uses it.
8. **`ramp-up/tls-eli5`** — a plain-English ramp on TLS for context.
9. **`ramp-up/oauth-oidc-eli5`** — a plain-English ramp on OAuth/OIDC, the foundation of OIDC auth into Vault.
10. **`ramp-up/linux-kernel-eli5`** — the sister sheet that explains what a kernel is. If anything in this sheet about "process" or "memory" felt fuzzy, that sheet helps.
11. **`ramp-up/kubernetes-eli5`** — Vault's Kubernetes integration makes more sense once Kubernetes is comfortable.

Beyond cheatsheets, the real next steps are:

- **Run a real cluster.** Spin up three Vault servers on three Linux machines (or three Docker containers) with integrated Raft. Initialize, unseal, get a feel for how the leader behaves. Crash the leader, watch a follower take over.
- **Use it from a real app.** Pick one application you maintain. Stop hardcoding its database password. Make it fetch credentials from Vault, via Vault Agent or via an SDK. Then make it use dynamic credentials. Watch the credential rotate every hour.
- **Read an audit log.** Tail the audit log of a busy Vault for ten minutes. You will be surprised at how much noise there is, and how clearly you can see who is doing what.
- **Try OpenBao.** Install `bao` alongside `vault`. Confirm that almost everything works the same. Decide which one your future will be on.

## Version Notes

Vault evolved a lot. A short timeline of features that matter:

- **1.0 (2018)** — batch tokens, performance standby (Enterprise).
- **1.4 (2020)** — integrated Raft storage, OIDC auth method.
- **1.5 (2020)** — Raft autopilot for cluster maintenance.
- **1.7 (2021)** — entropy augmentation, persistent caching in agent.
- **1.10 (2022)** — transit BYOK, managed keys for PKI.
- **1.11 (2022)** — KV v2 metadata patch, PKI multi-issuer.
- **1.13 (2023)** — license auto-rotation, transit cache.
- **1.14 (2023)** — PKI EAB (ACME pre-auth), event notifications.
- **1.15 (2023)** — license shifted to BUSL; **OpenBao** fork starts.
- **1.16 (2024)** — PKI ACME GA, more event notifications.
- **2024** — IBM announces acquisition of HashiCorp.

If you are running anything older than 1.10, plan an upgrade. If you are picking a starting point in 2025 or later, either the latest Vault on BUSL, or OpenBao on MPL 2.0. Both work.

## See Also

- `security/vault` — operator-focused cheatsheet for Vault.
- `secrets/sops` — file-based secret encryption with cloud KMS keys.
- `security/age` — modern file encryption tool.
- `secrets/gopass` — Git-based password store.
- `security/pki` — certificate authorities and X.509.
- `security/cryptography` — symmetric, asymmetric, hashes, signatures.
- `security/tls` — TLS fundamentals.
- `ramp-up/tls-eli5` — plain-English TLS.
- `ramp-up/oauth-oidc-eli5` — plain-English OAuth/OIDC.
- `ramp-up/linux-kernel-eli5` — plain-English kernel.
- `ramp-up/kubernetes-eli5` — plain-English Kubernetes.

## References

- HashiCorp Vault docs — https://developer.hashicorp.com/vault/docs
- HashiCorp Learn tutorials — https://developer.hashicorp.com/vault/tutorials
- "Running HashiCorp Vault in Production" by Andrey Belov — practical operations book.
- "Vault: Up & Running" (when published, O'Reilly) — successor to the unofficial guides.
- OpenBao project — https://openbao.org
- OpenBao on GitHub — https://github.com/openbao/openbao
- `bao` CLI reference — `bao --help` and the OpenBao docs.
- `vault.hcl` reference — https://developer.hashicorp.com/vault/docs/configuration
- Raft consensus paper ("In Search of an Understandable Consensus Algorithm" by Ongaro & Ousterhout) — https://raft.github.io/raft.pdf
- Shamir's secret sharing original paper (1979) — Adi Shamir, "How to share a secret."
- Vault HTTP API reference — https://developer.hashicorp.com/vault/api-docs
- Vault Agent docs — https://developer.hashicorp.com/vault/docs/agent
- Vault on Kubernetes — https://developer.hashicorp.com/vault/docs/platform/k8s
- Linux Foundation announcement of OpenBao — https://www.linuxfoundation.org/press/openbao-joins-the-linux-foundation
