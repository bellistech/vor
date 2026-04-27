# AWS CLI — ELI5

> The AWS CLI is a universal remote control for the entire AWS data center. Every button on the AWS web console has a matching `aws <service> <action>` command, and your terminal becomes the dashboard.

## Prerequisites

You should have a terminal you can type commands into (a "shell"). If you have never used a terminal, read **ramp-up/bash-eli5** first. Then come back here.

You should have an AWS account. Go to `https://aws.amazon.com/`, click "Create an AWS Account," and follow the steps. You will need a credit card. The "Free Tier" gives you a small amount of free usage every month for the first 12 months — enough to learn on without spending real money. **You do not need to pay anything to read this sheet.** You only pay if you actually create resources that cost money.

You should know how to copy and paste. That is it.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is aws-cli

### Imagine the AWS data center is a giant warehouse

Picture a really, really big warehouse. The kind of warehouse that is bigger than a city. Inside this warehouse there are millions of computers stacked on shelves. There are also robots that build new computers when you ask. There are conveyor belts that move data between computers. There are giant filing cabinets full of files. There are little post offices that pass messages between computers. There are security guards checking who is allowed in.

This warehouse is **AWS** — short for "Amazon Web Services." It is owned by a company called Amazon. AWS rents out tiny slices of this warehouse to anybody on the internet who has a credit card. You ask AWS for a computer, and AWS gives you one. You ask AWS for a place to store files, and AWS gives you one. You stop needing it, you tell AWS to take it back, and you stop paying.

The warehouse is real. There are buildings. There are racks of servers. There is electricity. There is air conditioning. There are people who walk around with hard hats. But you never go there. You never see it. **You talk to the warehouse over the internet.**

### How do you talk to the warehouse?

There are two main ways.

**Way 1: the web browser.** You go to `https://console.aws.amazon.com/` and log in. You see a website with thousands of buttons. Click "Start a Server," fill out a form, click "Launch." A real server somewhere in the warehouse turns on. You can see it on the website. This is called the **AWS Console**. It is good for poking around and learning. It is bad for doing the same thing fifty times in a row.

**Way 2: the command line.** You type a command into your terminal: `aws ec2 run-instances --image-id ami-12345 --instance-type t3.micro`. The command goes over the internet to AWS, the server turns on, you get back a little piece of text saying "okay, here is your server's ID." This is the **AWS CLI** — short for "Command Line Interface." It is good for doing the same thing fifty times in a row, for scripting, for automation, for not clicking around in a browser when you are concentrating in your terminal.

The CLI and the web console **do exactly the same thing.** They both talk to AWS. They are both controls for the warehouse. The only difference is one has buttons and one has words.

### So aws-cli is like a TV remote

Think of AWS like a giant TV with thousands of channels. Each channel is a service: there is the "computers" channel, the "files" channel, the "databases" channel, the "passwords" channel, the "robots" channel.

The web console is like the TV itself with a touchscreen. You poke buttons on the screen.

The CLI is like a **universal remote control.** You can change every channel from your couch. You don't have to walk over to the TV. You can also program the remote — make a single button that changes channels, turns the volume down, and starts recording, all in one press. That is **scripting.**

Every button on the TV has a matching button on the remote. Every action in the AWS Console has a matching `aws <service> <action>` command in the CLI.

### Why bother learning the CLI when the web console exists?

Six big reasons.

**1. Automation.** If you have to do the same thing every Monday morning — start ten servers, copy files between them, kick off a job — you can write a tiny script with the CLI and run the script. Saves hours.

**2. Speed.** Once you know the commands, typing `aws s3 ls` is faster than opening a browser, logging in, finding the right service, and clicking around.

**3. Reproducibility.** A web-console click is invisible. A CLI command is text. Text can be saved, shared, version-controlled, reviewed in a code review, replayed by a coworker. "How did you set that up?" is much easier to answer when the answer is a paste-able command.

**4. Scripting.** Bash, Python, Make, GitHub Actions — all of them can call `aws` and react to the output. You cannot script a web browser easily.

**5. Servers.** Most production servers don't have a screen. They are headless boxes in a data center. There is no browser. The only way to talk to AWS from a server is the CLI (or an SDK, which is the same idea but inside another programming language).

**6. Composition.** You can pipe `aws` output through `jq` and `grep` and `awk` and `xargs` and chain it with other commands. The web console is a dead end — once data is on a webpage, it's stuck on the webpage.

### What does the CLI actually do under the hood?

When you type `aws s3 ls`, here is what happens.

```
+--------+   1. you type        +-----------+
|  YOU   | -------------------> |  aws CLI  |
+--------+                      +-----------+
                                      |
                                      | 2. read credentials from ~/.aws/credentials
                                      | 3. read region from ~/.aws/config
                                      | 4. build an HTTPS request:
                                      |    GET https://s3.us-east-1.amazonaws.com/
                                      |    Authorization: AWS4-HMAC-SHA256 ...
                                      | 5. sign the request with SigV4
                                      v
                                +-----------+
                                |   AWS    |
                                | service  |  6. AWS verifies your signature
                                +-----------+  7. AWS returns XML or JSON
                                      |
                                      | 8. CLI parses the response
                                      v
                                +-----------+
                                | terminal | 9. you see the buckets printed
                                +-----------+
```

So the CLI is **just a fancy HTTP client.** It signs the request with your credentials so AWS knows it's really you, and it formats the response so you can read it. Underneath, every CLI command is a regular `https://` request to a regular web server. The signing is the only magic part. Everything else is plain old web.

The signing algorithm is called **SigV4** — Signature Version 4. It hashes your request together with your secret key. The secret key never leaves your computer. AWS can check the signature without ever seeing your secret. (This is the same trick HTTPS uses for websites, just with different math.)

### What can the CLI control?

Almost everything. AWS has more than two hundred services. The CLI has commands for virtually all of them. You will only use a handful regularly:

- **S3** — storing files (like Dropbox, but for programs).
- **EC2** — renting servers (virtual computers in the warehouse).
- **IAM** — managing who is allowed to do what.
- **Lambda** — running short snippets of code without renting a whole server.
- **CloudFormation** — describing your whole setup as a text file and saying "make me one of these."
- **DynamoDB** — a database where you store little JSON documents.
- **SQS** — a message queue where one program drops a message and another program picks it up.
- **Secrets Manager** — a vault for passwords and API keys.
- **CloudWatch** — logs and graphs of how your stuff is running.

The rest you can learn as you need them.

## v1 vs v2 (always use v2)

There are two major versions of the AWS CLI. **You only ever want v2.**

### What is v1?

Version 1 of the CLI came out in 2013. It was written in Python. To install it, you needed Python on your computer, which was annoying because every operating system shipped a slightly different Python and breakage was common.

### What is v2?

Version 2 became generally available in February 2020. It is shipped as a **single binary** — one file, no Python required, no dependencies. You drop it on your computer and it works. v2 also added:

- **Built-in SSO support.** v1 had no clue what SSO was; you had to use a third-party tool to log in via SSO. v2 supports `aws sso login` natively.
- **Auto-prompt mode.** Hit `Tab` after `aws ec2 ` and the CLI shows you all the actions you can take, like an interactive menu.
- **Faster startup.** v2 starts up roughly 5x faster than v1 because no Python interpreter has to load.
- **YAML output.** `--output yaml` and `--output yaml-stream` for human-friendlier reading.
- **Wizard mode.** `aws dynamodb wizard new-table` walks you through creating a table interactively.
- **IPv6 / dualstack endpoints** (since v2.13).
- **FIPS endpoints** configurable from the config file (since v2.15).
- **Refreshable SSO tokens** (since v2.17).

### What about v1?

**Version 1 is end-of-life.** Amazon stopped maintaining it. New AWS services do not get added to v1 anymore. Old v1 commands may stop working as APIs evolve. **Do not install v1.** Do not use the `awscli` PyPI package (that is v1). Do not follow tutorials that say `pip install awscli`.

If you already have v1 installed, uninstall it with `pip uninstall awscli` and install v2 instead.

### How do I tell which one I have?

```bash
$ aws --version
aws-cli/2.17.34 Python/3.11.9 Source/x86_64.darwin.23 prompt/off
```

If the first part says `aws-cli/1.x.y`, you have v1 and you need to upgrade. If it says `aws-cli/2.x.y`, you are good.

## Installation

This is where things go sideways. There are two PyPI packages and two installer paths and a Homebrew formula and an MSI installer, and they do not all install the same thing. Read carefully.

### The package name confusion

There are two Python packages on PyPI:

- `awscli` → installs **v1** (legacy, do not use)
- `awscli-v2` → unofficial third-party wrapper, also do not use

**The official way to install v2 is NOT pip.** Amazon does not publish v2 to PyPI. You install v2 from the binary installer, Homebrew, or `apt` / `dnf` / Chocolatey, depending on your OS.

### macOS — Homebrew (easiest)

```bash
$ brew install awscli
```

The Homebrew formula `awscli` installs **v2.** (Confusing, but true. Homebrew never carried v1, so on macOS the name `awscli` happens to mean v2.)

Verify:

```bash
$ aws --version
aws-cli/2.17.34 Python/3.11.9 Source/x86_64.darwin.23 prompt/off
```

### macOS — official installer

If you do not have Homebrew, download the `.pkg`:

```bash
$ curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o AWSCLIV2.pkg
$ sudo installer -pkg AWSCLIV2.pkg -target /
```

### Linux — official installer

```bash
$ curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o awscliv2.zip
$ unzip awscliv2.zip
$ sudo ./aws/install
```

For ARM64 (Graviton, Raspberry Pi):

```bash
$ curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o awscliv2.zip
```

### Linux — apt or dnf

The CLI is in many distro repos but the version is often old. Prefer the official installer above. If you must use the package manager:

```bash
$ sudo apt install awscli           # Debian/Ubuntu — may be v1, check version!
$ sudo dnf install awscli2          # Fedora — explicitly v2
```

Always check `aws --version` after installing.

### Windows — MSI installer

Download `https://awscli.amazonaws.com/AWSCLIV2.msi` and double-click. Or via Chocolatey:

```powershell
choco install awscli
```

Or via winget:

```powershell
winget install Amazon.AWSCLI
```

### Verify

```bash
$ aws --version
aws-cli/2.17.34 Python/3.11.9 Source/x86_64.darwin.23 prompt/off
$ which aws
/opt/homebrew/bin/aws
```

### Tab completion

Add this to your shell config (`~/.bashrc`, `~/.zshrc`):

```bash
complete -C '/usr/local/bin/aws_completer' aws
```

Now you can type `aws s` then `Tab` and the shell will offer `s3`, `sns`, `sqs`, etc.

### Auto-prompt mode

Set this once and the CLI will show you available subcommands and arguments interactively:

```bash
$ aws configure set cli_auto_prompt on-partial
```

`on` means always prompt, `on-partial` means only prompt when the command is incomplete, `off` means never. Most people like `on-partial`.

## Authentication

This is the most-confusing part of the CLI. Don't panic. We'll go slow.

### What does "authentication" mean here?

When the CLI sends a request to AWS, AWS needs to know **who** is sending it and **what they are allowed to do.** The CLI proves who you are by **signing every request** with a secret only you and AWS know. AWS checks the signature; if it matches, the request is allowed. If not, the request is rejected with `SignatureDoesNotMatch`.

So "authenticating" the CLI means: telling the CLI what your secret is, so it can sign requests on your behalf.

### The two pieces of credentials

Every AWS credential set has two pieces:

- **Access Key ID** — like a username. Looks like `AKIAIOSFODNN7EXAMPLE`. Public-ish; can show up in logs.
- **Secret Access Key** — like a password. Looks like `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`. **Never paste this anywhere public.** Never check it into git.

Sometimes there is a third piece for short-term credentials:

- **Session Token** — only used when you are using temporary credentials (more on this later). Long string starting with `IQoJb3JpZ2luX2VjE...`.

### Way 1: aws configure (the easy starting point)

Run this once and answer the prompts:

```bash
$ aws configure
AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
Default region name [None]: us-east-1
Default output format [None]: json
```

This creates two files in your home directory:

```
~/.aws/credentials   ← contains your secret keys
~/.aws/config        ← contains your region and output format
```

Open them up. They are plain text:

```ini
# ~/.aws/credentials
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

```ini
# ~/.aws/config
[default]
region = us-east-1
output = json
```

`[default]` means "this is the default profile." We'll cover other profiles in the next section.

**Test it:**

```bash
$ aws sts get-caller-identity
{
    "UserId": "AIDAJDPLRKLG7UEXAMPLE",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/Alice"
}
```

`sts get-caller-identity` is the universal "who am I" command. If it works, your credentials are configured.

### Way 2: environment variables

If you don't want to write credentials to disk, or if you want to override them temporarily, use environment variables:

```bash
$ export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
$ export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
$ export AWS_REGION=us-east-1
$ aws sts get-caller-identity
```

Useful in CI, in scripts, in Docker containers. Environment variables **override** the files. If you have credentials in `~/.aws/credentials` AND in env vars, the env vars win.

### Way 3: IAM Identity Center (formerly SSO)

This is the modern, recommended way for human users. Instead of writing long-term keys to disk, you log in through your company's identity provider (Google, Okta, Azure AD, AWS-hosted), and the CLI receives a **short-term token** that expires after a few hours.

Setup:

```bash
$ aws configure sso
SSO session name (Recommended): my-company
SSO start URL: https://my-company.awsapps.com/start
SSO region: us-east-1
SSO registration scopes [sso:account:access]:
```

A browser window pops up. You log in with your normal company credentials. You pick which AWS account and role you want. The CLI writes a profile to `~/.aws/config`.

```ini
[profile my-dev]
sso_session = my-company
sso_account_id = 123456789012
sso_role_name = Developer
region = us-east-1

[sso-session my-company]
sso_start_url = https://my-company.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
```

To log in:

```bash
$ aws sso login --profile my-dev
```

Browser opens, you click "approve," done. The CLI now has a 1-12 hour token cached in `~/.aws/sso/cache/`. After that expires, run `aws sso login` again.

**Why is this better than long-term keys?** Because if your laptop gets stolen, the worst case is the thief has access for a few hours, not forever. Long-term keys live until you remember to revoke them, which most people forget.

### Way 4: instance profile (running on EC2)

When the CLI runs on an EC2 instance with an attached **IAM role**, the CLI doesn't need any credentials configured. The CLI calls a special URL only available from inside the instance — `http://169.254.169.254/latest/meta-data/iam/security-credentials/<role-name>` — and AWS hands back temporary credentials. These rotate automatically.

**You write zero credential code.** Just attach a role to the instance and `aws sts get-caller-identity` works.

This is the right answer for production servers. Long-term keys on a server are a bad idea: if the server is compromised, the keys are stolen and never expire.

### Way 5: container credentials

Same idea as instance profile, but for ECS / EKS / Fargate / App Runner. The container runtime sets the env var `AWS_CONTAINER_CREDENTIALS_RELATIVE_URI` and the CLI fetches creds from there.

### Way 6: AssumeRole (cross-account)

You have credentials for Account A, but you want to do something in Account B. You can **assume a role** in Account B that trusts Account A.

```bash
$ aws sts assume-role \
    --role-arn arn:aws:iam::222222222222:role/CrossAccountAdmin \
    --role-session-name my-session
{
    "Credentials": {
        "AccessKeyId": "ASIA...",
        "SecretAccessKey": "...",
        "SessionToken": "IQoJ...",
        "Expiration": "2026-04-27T18:00:00Z"
    }
}
```

You get back temporary credentials valid for up to 1 hour (default; configurable up to 12 hours via `--duration-seconds`). Set them as env vars or write them into a profile, and now you are operating in Account B.

A cleaner way: configure a profile with `role_arn` and `source_profile`, and the CLI handles the AssumeRole automatically.

```ini
[profile prod-admin]
role_arn = arn:aws:iam::222222222222:role/CrossAccountAdmin
source_profile = default
region = us-east-1
```

Now `aws --profile prod-admin sts get-caller-identity` will use the source profile to call AssumeRole and use the resulting credentials. The CLI caches the assumed-role credentials for the session.

### The credential provider chain

When you run `aws something`, the CLI looks for credentials in this order. It uses the first one it finds:

```
+---------------------------------------------------------+
|  CREDENTIAL PROVIDER CHAIN — order matters!             |
+---------------------------------------------------------+
| 1. CLI flags  (--profile, --region)                     |
| 2. Environment variables                                 |
|    AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY            |
|    AWS_SESSION_TOKEN (if temporary)                     |
|    AWS_PROFILE (selects which profile in files)         |
| 3. Shared credentials file ~/.aws/credentials           |
|    [default] section, or [profile xyz] if AWS_PROFILE   |
| 4. Shared config file ~/.aws/config                     |
|    SSO sessions, role chaining, credential_process      |
| 5. ECS/EKS container credentials                        |
|    AWS_CONTAINER_CREDENTIALS_RELATIVE_URI               |
|    AWS_CONTAINER_CREDENTIALS_FULL_URI                   |
| 6. EC2 instance metadata                                 |
|    http://169.254.169.254/latest/meta-data/...          |
+---------------------------------------------------------+
| First match wins. If nothing matches:                   |
|   "Unable to locate credentials"                        |
+---------------------------------------------------------+
```

The most common bug: you have `AWS_ACCESS_KEY_ID` set in your shell from a previous task, and you wonder why your `[default]` profile is being ignored. Run `env | grep AWS` to see what's set, and `unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN` to clear them.

## Profiles and Regions

### What is a profile?

A profile is a **named bundle of credentials and settings.** You can have multiple profiles for multiple AWS accounts and switch between them.

```ini
# ~/.aws/credentials
[default]
aws_access_key_id = AKIA...
aws_secret_access_key = ...

[work]
aws_access_key_id = AKIB...
aws_secret_access_key = ...

[clientA-prod]
aws_access_key_id = AKIC...
aws_secret_access_key = ...
```

### Selecting a profile

Three ways. CLI flag wins over env var, env var wins over default.

```bash
# CLI flag (highest priority)
$ aws --profile work s3 ls

# environment variable
$ export AWS_PROFILE=work
$ aws s3 ls

# default profile (when neither flag nor env var is set)
$ aws s3 ls
```

### Profile resolution decision tree

```
Did you pass --profile FOO?
  YES -> use profile FOO
  NO  -> Is AWS_PROFILE set?
           YES -> use that profile
           NO  -> Is AWS_DEFAULT_PROFILE set?
                    YES -> use that profile
                    NO  -> use [default]
                            (or fall through to env vars,
                             instance metadata, etc. as
                             per the credential chain)
```

`AWS_PROFILE` and `AWS_DEFAULT_PROFILE` mean the same thing in v2. v1 used `AWS_DEFAULT_PROFILE`; v2 added `AWS_PROFILE`. v2 honors both, with `AWS_PROFILE` winning if both are set.

### Listing profiles

```bash
$ aws configure list-profiles
default
work
clientA-prod
```

Show what the current profile is set to:

```bash
$ aws configure list
      Name                    Value             Type    Location
      ----                    -----             ----    --------
   profile                <not set>             None    None
access_key     ****************MPLE shared-credentials-file
secret_key     ****************KKEY shared-credentials-file
    region                us-east-1      config-file    ~/.aws/config
```

### Regions

AWS divides the world into **regions** — clusters of data centers in geographic areas. Each region is independent: an S3 bucket created in `us-east-1` does not exist in `eu-west-1`. You pay differently per region. Some services are only in some regions.

Common regions:

| Region | Where |
| --- | --- |
| `us-east-1` | N. Virginia (often the cheapest, where new services launch first) |
| `us-east-2` | Ohio |
| `us-west-2` | Oregon |
| `eu-west-1` | Ireland |
| `eu-central-1` | Frankfurt |
| `ap-southeast-2` | Sydney |
| `ap-northeast-1` | Tokyo |

The CLI picks the region in this order:

1. `--region us-east-1` flag.
2. `AWS_REGION` env var.
3. `AWS_DEFAULT_REGION` env var.
4. `region = us-east-1` in `~/.aws/config` for the active profile.
5. Instance metadata (when running on EC2).

If no region is set and the service is regional, you get `You must specify a region. You can also configure your region by running "aws configure".`

### Global vs regional services

Most services are **regional**. A few are **global** — they exist across all regions.

- Global: IAM, Route 53, CloudFront, AWS Organizations, S3 (bucket names are globally unique, but data lives in a region).
- Regional: EC2, S3 (data), Lambda, RDS, DynamoDB, SQS, almost everything else.

For global services, the region setting is mostly a hint; the API endpoint is the same everywhere (`iam.amazonaws.com`).

## Output Formats

The CLI can print results in five formats. Pick the one that fits your eyes or your script.

### --output json (default)

Machine-readable, what most scripts use.

```bash
$ aws s3api list-buckets --output json
{
    "Buckets": [
        {
            "Name": "my-bucket",
            "CreationDate": "2026-01-15T10:30:00.000Z"
        }
    ],
    "Owner": {
        "ID": "abc123..."
    }
}
```

### --output yaml

Same data, friendlier whitespace.

```bash
$ aws s3api list-buckets --output yaml
Buckets:
- CreationDate: '2026-01-15T10:30:00.000Z'
  Name: my-bucket
Owner:
  ID: abc123...
```

### --output yaml-stream

YAML one document per page; useful for streaming large results.

```bash
$ aws s3api list-objects-v2 --bucket B --output yaml-stream
---
- Key: file1.txt
  Size: 100
---
- Key: file2.txt
  Size: 200
```

### --output text

Tab-separated columns, no decoration. Easy to chain with `awk`, `cut`, `xargs`.

```bash
$ aws s3api list-buckets --output text
BUCKETS  2026-01-15T10:30:00.000Z   my-bucket
OWNER   abc123...
```

```bash
$ aws ec2 describe-instances --output text \
    --query 'Reservations[].Instances[].InstanceId'
i-0abc12345678
i-0def98765432
```

### --output table

Pretty boxes for human reading. Slow on huge result sets.

```bash
$ aws ec2 describe-instances --output table \
    --query 'Reservations[].Instances[].[InstanceId,State.Name,InstanceType]'
-----------------------------------------
|           DescribeInstances           |
+----------------+----------+-----------+
|  i-0abc12345   |  running | t3.micro  |
|  i-0def98765   |  stopped | t3.small  |
+----------------+----------+-----------+
```

### Setting a default

Per-command:

```bash
$ aws --output table s3api list-buckets
```

In your config:

```ini
[default]
output = yaml
```

Via env var:

```bash
$ export AWS_DEFAULT_OUTPUT=table
```

## JMESPath Queries

JMESPath is a query language for JSON. The `--query` flag lets you slice and filter the CLI's output **on the client side** before it's printed. It is one of the most powerful flags in the whole CLI.

The full spec is at `https://jmespath.org/`. Read it. Bookmark it.

### Basic shape

`--query 'EXPRESSION'`

```bash
$ aws s3api list-buckets --query 'Buckets[].Name'
[
    "my-bucket",
    "logs-bucket",
    "backups-bucket"
]
```

The `Buckets[]` says "for each item in Buckets, look at..." and `.Name` says "...its Name field."

### Common patterns

**All values from a field:**

```bash
$ aws ec2 describe-instances --query 'Reservations[].Instances[].InstanceId'
```

**Filter by a field value:**

```bash
$ aws ec2 describe-instances \
    --query 'Reservations[].Instances[?State.Name==`running`].InstanceId'
```

Note the **backticks** around `running`. JMESPath uses backticks for literal strings inside expressions.

**Pull multiple fields into a list:**

```bash
$ aws ec2 describe-instances \
    --query 'Reservations[].Instances[].[InstanceId,InstanceType,State.Name]' \
    --output table
```

**Find a tag value:**

```bash
$ aws ec2 describe-instances \
    --query 'Reservations[].Instances[].[InstanceId,Tags[?Key==`Name`].Value|[0]]' \
    --output table
```

The `|[0]` flattens "the first element of the matched values."

**Count things:**

```bash
$ aws s3api list-buckets --query 'length(Buckets)'
12
```

**Sort:**

```bash
$ aws s3api list-buckets --query 'sort_by(Buckets, &CreationDate)[].Name'
```

**Slice (first 5):**

```bash
$ aws s3api list-buckets --query 'Buckets[0:5].Name'
```

### --query vs --filter

- `--query` is **client-side.** The CLI fetches everything and then filters locally. Works for any service.
- `--filters` is **server-side** (and only available on certain commands like `ec2 describe-instances`). The server filters before returning, which is faster and cheaper on big result sets.

Use `--filters` when available, fall back to `--query`.

```bash
# server-side filter, then client-side query
$ aws ec2 describe-instances \
    --filters Name=tag:Env,Values=prod \
    --query 'Reservations[].Instances[].InstanceId' \
    --output text
```

### When JMESPath is too clunky, pipe to jq

`jq` is a separate CLI tool that is more flexible. Install with `brew install jq` or `apt install jq`. Use it when the JMESPath gets unreadable.

```bash
$ aws ec2 describe-instances --output json \
  | jq '.Reservations[].Instances[] | select(.State.Name=="running") | .InstanceId'
```

`jq` and JMESPath solve the same problem. JMESPath is built into the CLI; `jq` is more powerful but external.

## Pagination

AWS APIs return results in **pages** — usually 100 or 1000 items at a time. The CLI auto-paginates by default: it walks through every page and gives you the combined result. For most commands this is what you want.

### Auto-paginate (default)

```bash
$ aws s3api list-objects-v2 --bucket huge-bucket
# might take a while if the bucket has 1,000,000 objects,
# but you get all of them
```

### --max-items

Limit how many you pull total.

```bash
$ aws s3api list-objects-v2 --bucket huge-bucket --max-items 50
```

The CLI also returns a `NextToken` when there are more results. Pass it back in `--starting-token` to keep going.

### --starting-token

```bash
$ aws s3api list-objects-v2 --bucket huge-bucket --max-items 50
{
  "Contents": [...50 items...],
  "NextToken": "ZmlsZTUxLnR4dA=="
}
$ aws s3api list-objects-v2 --bucket huge-bucket --max-items 50 \
    --starting-token "ZmlsZTUxLnR4dA=="
```

### --no-paginate

Stop after the first server-side page. Faster when you only need a peek.

```bash
$ aws s3api list-objects-v2 --bucket huge-bucket --no-paginate
```

### --page-size

How many items the server returns per page (CLI still aggregates them all unless you use `--no-paginate` or `--max-items`).

```bash
$ aws s3api list-objects-v2 --bucket huge-bucket --page-size 100
```

### The pager

Long output goes through `less` by default in v2. Set `AWS_PAGER` to control it:

```bash
$ export AWS_PAGER=""              # no pager — print everything to stdout
$ export AWS_PAGER="less -R"       # default-ish, with color codes
$ export AWS_PAGER="cat"           # straight to stdout
```

Or use the flag:

```bash
$ aws --no-cli-pager s3api list-objects-v2 --bucket huge-bucket
```

In scripts, **always** set `AWS_PAGER=""` or use `--no-cli-pager` so the command doesn't get stuck waiting for a non-existent terminal.

## Wait Commands

Many actions are **asynchronous.** You ask AWS to start an instance, and AWS says "okay, starting" and returns immediately, even though the instance isn't running yet. You then poll until it's ready.

The CLI ships **waiters** for common transitions. They poll the API for you, with sane backoff, until the resource reaches the right state, or until they give up.

```bash
$ aws ec2 start-instances --instance-ids i-0abc
$ aws ec2 wait instance-running --instance-ids i-0abc
# (returns silently when the instance is running, or errors if it never starts)
```

Other useful waiters:

```bash
$ aws ec2 wait instance-stopped --instance-ids i-0abc
$ aws ec2 wait instance-terminated --instance-ids i-0abc
$ aws cloudformation wait stack-create-complete --stack-name my-stack
$ aws cloudformation wait stack-update-complete --stack-name my-stack
$ aws s3api wait bucket-exists --bucket my-bucket
$ aws s3api wait object-exists --bucket my-bucket --key path/to/file
$ aws rds wait db-instance-available --db-instance-identifier my-db
$ aws ecs wait services-stable --cluster c --services s
```

To list available waiters for a service:

```bash
$ aws ec2 wait help
```

Waiters time out after a built-in number of attempts (varies by command, typically 40 attempts at 15s intervals = 10 minutes). For long-running things like `cloudformation wait`, this is usually plenty.

## Service Quick-Tour

A whirlwind through the services you'll touch most often. Each has its own deep-dive sheet (see `cloud/aws-cli` for more).

### S3 — Simple Storage Service (file storage)

Buckets hold objects. Buckets have globally-unique names; objects have keys (paths within a bucket). Pay per GB stored, per request, per byte transferred.

```bash
$ aws s3 ls                                      # list all your buckets
$ aws s3 ls s3://my-bucket/                      # list top-level keys
$ aws s3 cp file.txt s3://my-bucket/path/        # upload
$ aws s3 cp s3://my-bucket/path/file.txt .       # download
$ aws s3 sync ./local s3://my-bucket/prefix/     # smart upload (only changed files)
$ aws s3 rm s3://my-bucket/path/file.txt         # delete
```

### EC2 — Elastic Compute Cloud (rented servers)

Servers are called **instances**. They run from images called **AMIs**. They live in **subnets** inside **VPCs**. They have **security groups** as firewalls.

```bash
$ aws ec2 describe-instances                     # list your instances
$ aws ec2 start-instances --instance-ids i-X     # turn one on
$ aws ec2 stop-instances --instance-ids i-X      # turn one off
$ aws ec2 terminate-instances --instance-ids i-X # destroy permanently
$ aws ec2 run-instances --image-id ami-X --instance-type t3.micro --key-name K
```

### IAM — Identity and Access Management

Who can do what. Users, groups, roles, policies. Everything in AWS goes through IAM checks.

```bash
$ aws iam list-users
$ aws iam create-user --user-name alice
$ aws iam attach-user-policy --user-name alice \
    --policy-arn arn:aws:iam::aws:policy/ReadOnlyAccess
$ aws iam list-roles
```

### Lambda — serverless functions

Run code without renting a whole server. Pay per millisecond. Triggered by HTTP, events, schedules, queues.

```bash
$ aws lambda list-functions
$ aws lambda invoke --function-name my-fn --payload '{"k":"v"}' /tmp/out.json
$ cat /tmp/out.json
$ aws lambda update-function-code --function-name my-fn --zip-file fileb://function.zip
```

### CloudFormation — infrastructure as code (IaC)

Describe a whole stack of AWS resources in a YAML or JSON file; CloudFormation creates them.

```bash
$ aws cloudformation deploy --template-file t.yaml --stack-name my-stack \
    --parameter-overrides Env=prod
$ aws cloudformation describe-stacks --stack-name my-stack
$ aws cloudformation describe-stack-events --stack-name my-stack
$ aws cloudformation delete-stack --stack-name my-stack
```

### ECS — Elastic Container Service

Run Docker containers without managing Kubernetes.

```bash
$ aws ecs list-clusters
$ aws ecs describe-services --cluster my-cluster --services my-svc
$ aws ecs update-service --cluster my-cluster --service my-svc --desired-count 5
```

### EKS — Elastic Kubernetes Service

Managed Kubernetes. The big sibling of ECS.

```bash
$ aws eks update-kubeconfig --name my-cluster --region us-east-1
# now kubectl works
$ kubectl get nodes
```

### RDS — Relational Database Service

Managed PostgreSQL, MySQL, MariaDB, SQL Server, Oracle. Aurora is AWS's own engine.

```bash
$ aws rds describe-db-instances
$ aws rds create-db-snapshot --db-instance-identifier my-db --db-snapshot-identifier my-snap
```

### DynamoDB — NoSQL key-value/document database

Schemaless, serverless, scales horizontally without you doing anything.

```bash
$ aws dynamodb list-tables
$ aws dynamodb scan --table-name my-table --max-items 10
$ aws dynamodb get-item --table-name my-table --key '{"id":{"S":"123"}}'
$ aws dynamodb put-item --table-name my-table --item '{"id":{"S":"123"},"name":{"S":"Alice"}}'
```

### SQS — Simple Queue Service

A message queue. Producer drops a message; consumer picks it up later.

```bash
$ aws sqs list-queues
$ aws sqs send-message --queue-url X --message-body 'hello'
$ aws sqs receive-message --queue-url X --max-number-of-messages 10
$ aws sqs delete-message --queue-url X --receipt-handle <handle>
```

### SNS — Simple Notification Service

Pub/sub topics. Publish a message, every subscriber gets a copy (email, SMS, HTTPS, SQS, Lambda).

```bash
$ aws sns list-topics
$ aws sns publish --topic-arn arn:aws:sns:us-east-1:123:my-topic --message 'alert'
```

### KMS — Key Management Service

Encryption keys. AWS holds the master key; you ask KMS to encrypt and decrypt small payloads.

```bash
$ aws kms list-keys
$ aws kms list-aliases
$ aws kms encrypt --key-id alias/myapp --plaintext fileb://secret.bin \
    --output text --query CiphertextBlob | base64 -d > encrypted.bin
$ aws kms decrypt --ciphertext-blob fileb://encrypted.bin \
    --output text --query Plaintext | base64 -d
```

### Route 53 — DNS

Authoritative DNS. Buy domains here too.

```bash
$ aws route53 list-hosted-zones
$ aws route53 list-resource-record-sets --hosted-zone-id Z123
```

### Secrets Manager — passwords and API keys

Encrypted vault for app secrets. Auto-rotation supported.

```bash
$ aws secretsmanager list-secrets
$ aws secretsmanager get-secret-value --secret-id my-secret --query SecretString --output text
```

### Systems Manager Parameter Store — config and small secrets

Cheaper than Secrets Manager for plain config; supports `SecureString` for encrypted values.

```bash
$ aws ssm get-parameter --name /myapp/db/password --with-decryption
$ aws ssm put-parameter --name /myapp/db/password --type SecureString --value 'X' --overwrite
```

### CloudWatch — logs and metrics

Where every service writes its operational data.

```bash
$ aws logs tail /aws/lambda/my-fn --follow --since 1h
$ aws cloudwatch get-metric-statistics --namespace AWS/EC2 --metric-name CPUUtilization \
    --dimensions Name=InstanceId,Value=i-X \
    --start-time 2026-04-26T00:00:00 --end-time 2026-04-27T00:00:00 \
    --period 300 --statistics Average
```

### Session Manager — shell into an EC2 without SSH

Bypass SSH entirely. Requires the SSM agent on the instance and proper IAM.

```bash
$ aws ssm start-session --target i-0abc12345
# you're now inside the instance, no port 22 open
```

## S3 Power Cookbook

S3 has more knobs than any other AWS service. Here are the ones worth knowing.

### Copy and sync

```bash
# single file up
$ aws s3 cp file.txt s3://bucket/key

# single file down
$ aws s3 cp s3://bucket/key local.txt

# whole directory up
$ aws s3 cp ./dir s3://bucket/prefix --recursive

# whole directory up, but only changed files (faster, idempotent)
$ aws s3 sync ./dir s3://bucket/prefix

# sync, deleting files in S3 that no longer exist locally
$ aws s3 sync ./dir s3://bucket/prefix --delete

# exclude a pattern
$ aws s3 sync ./dir s3://bucket/prefix --exclude '*.tmp' --exclude '*.log'

# include only a pattern
$ aws s3 sync ./dir s3://bucket/prefix --exclude '*' --include '*.json'
```

### Presigned URLs

A presigned URL is a regular HTTPS URL that grants temporary download (or upload) permission. Anyone with the URL can fetch the object, no AWS login needed, until it expires.

```bash
$ aws s3 presign s3://bucket/key --expires-in 3600
https://bucket.s3.amazonaws.com/key?X-Amz-Algorithm=AWS4-HMAC-SHA256&...&X-Amz-Expires=3600&...
```

Default expiry is 1 hour. Max is 7 days (604800 seconds). Useful for sharing a download link without granting full bucket access.

### Server-side encryption

```bash
# upload with SSE-S3 (AES256 with AWS-managed key)
$ aws s3 cp file.txt s3://bucket/key --sse AES256

# upload with SSE-KMS (your KMS key)
$ aws s3 cp file.txt s3://bucket/key --sse aws:kms --sse-kms-key-id alias/my-key

# enforce default encryption on the bucket
$ aws s3api put-bucket-encryption --bucket bucket \
    --server-side-encryption-configuration \
    '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
```

### Versioning

Every overwrite or delete keeps the old version, recoverable.

```bash
$ aws s3api put-bucket-versioning --bucket bucket \
    --versioning-configuration Status=Enabled

$ aws s3api list-object-versions --bucket bucket --prefix path/
```

### Lifecycle policies

Move objects to cheaper storage automatically.

```bash
$ cat lifecycle.json
{
  "Rules": [
    {
      "ID": "archive-after-90-days",
      "Status": "Enabled",
      "Filter": {"Prefix": "logs/"},
      "Transitions": [
        {"Days": 30, "StorageClass": "STANDARD_IA"},
        {"Days": 90, "StorageClass": "GLACIER"}
      ],
      "Expiration": {"Days": 365}
    }
  ]
}
$ aws s3api put-bucket-lifecycle-configuration --bucket bucket \
    --lifecycle-configuration file://lifecycle.json
```

### Intelligent Tiering

S3 watches access patterns and moves objects to cheaper tiers automatically when they're cold.

```bash
$ aws s3 cp big.bin s3://bucket/key --storage-class INTELLIGENT_TIERING
```

### Replication

Copy every new object in bucket A to bucket B (same region or different).

```bash
$ aws s3api put-bucket-replication --bucket source-bucket \
    --replication-configuration file://replication.json
```

### Multi-part upload

For files bigger than 5 GB, S3 requires multi-part upload. The `aws s3 cp` command does this automatically. The flow:

```
+-------------+  1. CreateMultipartUpload   +--------+
| your laptop |---------------------------->|   S3   |
+-------------+ <----------------------------+--------+
                  uploadId=AAA

         (split file into 5 MB chunks)

  2. UploadPart 1 ----> S3   returns ETag1
  3. UploadPart 2 ----> S3   returns ETag2
  4. UploadPart 3 ----> S3   returns ETag3
              ...

  5. CompleteMultipartUpload(uploadId=AAA, parts=[(1,ETag1),(2,ETag2),(3,ETag3)])
                           ----> S3   returns the final object's ETag
```

Manually:

```bash
$ aws s3api create-multipart-upload --bucket B --key K
$ aws s3api upload-part --bucket B --key K --part-number 1 \
    --upload-id <id> --body part1.bin
$ aws s3api complete-multipart-upload --bucket B --key K --upload-id <id> \
    --multipart-upload file://parts.json
```

### Listing object metadata

```bash
$ aws s3api head-object --bucket B --key K
{
    "AcceptRanges": "bytes",
    "LastModified": "2026-04-27T10:00:00+00:00",
    "ContentLength": 1234,
    "ETag": "\"abc123\"",
    "ContentType": "text/plain",
    "ServerSideEncryption": "AES256"
}
```

## IAM Best Practices

IAM is the most-misunderstood service. A bad IAM policy is the most common AWS security incident. Take your time here.

### Least privilege

Every user, role, and policy should grant **the minimum** needed to do its job. Not "Admin" on everything. Not "S3 Full Access." Specifically the actions on the specific resources.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject"],
      "Resource": "arn:aws:s3:::my-app-data/*"
    }
  ]
}
```

That allows `GetObject` and `PutObject` on `my-app-data` only. Nothing else.

### Roles, not users

Long-term IAM users are 2010 thinking. Modern AWS:

- **Humans** authenticate via IAM Identity Center (SSO).
- **Workloads** (EC2, ECS, Lambda) use **IAM roles** with **short-term temporary credentials** automatically rotated.
- **CI** uses **OIDC federation** — your CI provider (GitHub Actions, GitLab) presents a JWT, AWS verifies it, hands back temporary credentials. No long-term keys in CI secrets.

### Role chaining

Role A can assume Role B can assume Role C. Each AssumeRole call is capped at 1 hour by default (and **the chained call is hard-capped at 1 hour** even if the role's `MaxSessionDuration` is higher — only the **first** AssumeRole in a chain can use the higher cap, up to 12 hours).

```
+---------+  AssumeRole(B)  +--------+  AssumeRole(C)  +--------+
| user A  |---------------->|  Role B |---------------->|  Role C |
+---------+   max 12h       +--------+    max 1h       +--------+
                            (if A is a user)         (because chained)
```

### Condition keys

Restrict policies further with conditions. Common ones:

- `aws:SourceIp` — only from this IP range.
- `aws:MultiFactorAuthPresent` — only if MFA is on.
- `aws:CurrentTime` — only during business hours.
- `aws:RequestedRegion` — only in this region.
- `aws:PrincipalTag/Department` — ABAC.

```json
{
  "Effect": "Allow",
  "Action": "s3:*",
  "Resource": "arn:aws:s3:::my-bucket/*",
  "Condition": {
    "StringEquals": {"aws:PrincipalTag/Department": "Engineering"}
  }
}
```

### ABAC — attribute-based access control

Tag your principals (users, roles) with attributes. Tag your resources with attributes. Write policies that say "the user's `Department` tag must equal the resource's `Department` tag." One policy scales to thousands of users without rewriting.

### Permissions boundaries

A "ceiling" on what a user/role **could ever** do, separate from what their policy grants. Useful for delegating "junior admins can create roles, but never roles more powerful than this boundary."

### SCPs (Service Control Policies)

Top-level guardrails applied at the AWS Organizations level. SCPs cannot grant; they can only deny. Common SCPs:

- "Deny `iam:*` outside the management account."
- "Deny anything outside `us-east-1` and `eu-west-1`."
- "Deny disabling CloudTrail."

### Trust policies vs identity-based vs resource-based

- **Identity-based policy**: attached to a user/group/role; says what *they* can do.
- **Resource-based policy**: attached to a resource (bucket, queue, KMS key); says who can do what *to it*.
- **Trust policy**: a special resource-based policy attached to an IAM role; says who is *allowed to assume* this role.

For an action to be allowed, **both** the identity policy *and* (if applicable) the resource policy must allow it.

### Policy simulator

Test before you ship.

```bash
$ aws iam simulate-principal-policy \
    --policy-source-arn arn:aws:iam::123:user/Alice \
    --action-names s3:GetObject \
    --resource-arns arn:aws:s3:::my-bucket/path/file.txt
```

### Last-accessed info

See which permissions are unused.

```bash
$ aws iam generate-service-last-accessed-details --arn arn:aws:iam::123:user/Alice
$ aws iam get-service-last-accessed-details --job-id <returned-id>
```

Use this to ratchet down policies — if a service hasn't been called in 90 days, you probably don't need that permission.

### MFA on the root user

The root user is the email-and-password owner of the account. Lock it in a vault. Enable MFA. Never use root for daily work.

## Common Errors

Verbatim error messages and what to do. Memorize these — you will see them again.

### `An error occurred (UnauthorizedOperation) when calling the X operation`

EC2-flavored "you can't do that." The IAM policy attached to your principal does not allow the action. Check the policy. Often appears as `UnauthorizedOperation: You are not authorized to perform this operation. Encoded authorization failure message: <long blob>`. Decode the blob with:

```bash
$ aws sts decode-authorization-message --encoded-message <blob>
```

That tells you exactly which action and resource were denied.

### `An error occurred (AccessDenied) when calling the X operation`

The S3-flavored cousin. Same root cause: missing IAM permissions, or a bucket policy that denies you, or a KMS key policy that denies you, or an SCP that denies you. Read the message carefully — it sometimes says "explicit deny" which means a `Deny` statement is winning.

### `Could not connect to the endpoint URL: "https://service.us-east-1.amazonaws.com/"`

Network issue or wrong endpoint. Check:

- DNS resolves the endpoint (try `dig service.us-east-1.amazonaws.com`).
- No proxy is blocking egress.
- Region is correct (some services don't exist in every region).
- VPC endpoint, if you set one, points at the right service.

### `The config profile (X) could not be found`

You ran `aws --profile X` and there is no `[profile X]` block in `~/.aws/config` and no `[X]` block in `~/.aws/credentials`. Run `aws configure list-profiles` to see what's there. Spelling matters.

### `Unable to locate credentials. You can configure credentials by running "aws configure".`

The credential provider chain found nothing. No env vars, no files, no instance metadata. Run `aws configure` if it's a laptop, or attach a role if it's an EC2.

### `A client error (ExpiredToken) occurred when calling the X operation: The provided token has expired`

Short-term credentials timed out. If you used `aws sso login`, run it again. If you used `aws sts assume-role`, fetch a fresh set. If you're in CI, your role-assumption probably needs to be re-run before the long-running step.

### `The security token included in the request is expired`

Same as above. Different wording, same cause.

### `An error occurred (RequestExpired) when calling the X operation`

Different from token expiry — this means your **clock is wrong**. SigV4 signs the request timestamp; AWS rejects requests more than 5-15 minutes off the wall clock. Run NTP on your machine.

### `An error occurred (EntityAlreadyExists) when calling the CreateUser operation`

You tried to create a user/role/policy with a name that's already taken. Pick a different name or check whether you already created it.

### `An error occurred (Throttling) when calling the X operation: Rate exceeded`

You're hitting AWS too hard, too fast. Each service has a throttle. The CLI retries automatically with exponential backoff (legacy mode does 4 retries, standard mode does 3, adaptive mode adapts). You can tune:

```bash
$ export AWS_RETRY_MODE=adaptive
$ export AWS_MAX_ATTEMPTS=10
```

If you keep hitting limits, slow down or request a quota increase.

### `SignatureDoesNotMatch`

The CLI's computed signature doesn't match what AWS expected. Causes:

- Clock skew (run NTP).
- Secret key with trailing whitespace or wrong characters (re-paste).
- A proxy is rewriting headers (turn it off).

### `ValidationException`

The request body is malformed. The error includes which field is wrong. Read it. Often: missing required field, value too long, invalid enum.

### `ResourceNotFoundException`

You asked about something that doesn't exist. Check the name. Check the region (you might be looking in `us-west-2` for something in `us-east-1`).

### `LimitExceeded`

You've hit an account quota — too many EC2 instances, too many security groups, too many secrets. Open a support ticket via the Service Quotas console to raise the limit.

### `InvalidClientTokenId`

The Access Key ID does not exist. Check `~/.aws/credentials` for typos. Maybe you deleted the key and forgot to update.

### `An error occurred (OptInRequired)`

The region you're using requires opt-in (some new regions do, like `me-south-1`, `ap-east-1`). Enable the region in the AWS Console under Account → Regions.

## Hands-On

Type these in. Watch what happens. Read the output. The numbers are guideposts, not steps; pick what you want to learn.

### 1. Version

```bash
$ aws --version
aws-cli/2.17.34 Python/3.11.9 Source/x86_64.darwin.23 prompt/off
```

### 2. Configure default profile

```bash
$ aws configure
AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
Default region name [None]: us-east-1
Default output format [None]: json
```

### 3. Show what's configured

```bash
$ aws configure list
      Name                    Value             Type    Location
      ----                    -----             ----    --------
   profile                <not set>             None    None
access_key     ****************MPLE shared-credentials-file
secret_key     ****************KKEY shared-credentials-file
    region                us-east-1      config-file    ~/.aws/config
```

### 4. List all profiles

```bash
$ aws configure list-profiles
default
work
```

### 5. SSO login

```bash
$ aws configure sso
SSO session name (Recommended): my-org
SSO start URL: https://my-org.awsapps.com/start
SSO region: us-east-1
SSO registration scopes [sso:account:access]:
Attempting to automatically open the SSO authorization page in your default browser.
```

### 6. Trigger SSO login for a specific profile

```bash
$ aws sso login --profile my-dev
Attempting to automatically open the SSO authorization page in your default browser.
Successfully logged into Start URL: https://my-org.awsapps.com/start
```

### 7. Who am I (the universal sanity check)

```bash
$ aws sts get-caller-identity
{
    "UserId": "AIDAJDPLRKLG7UEXAMPLE",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/Alice"
}
```

### 8. AssumeRole into another account

```bash
$ aws sts assume-role \
    --role-arn arn:aws:iam::222222222222:role/CrossAccountAdmin \
    --role-session-name my-session
{
    "Credentials": {
        "AccessKeyId": "ASIAxxxxx",
        "SecretAccessKey": "xxx",
        "SessionToken": "IQoJb3JpZ2luX2VjE...",
        "Expiration": "2026-04-27T19:00:00Z"
    },
    "AssumedRoleUser": {
        "AssumedRoleId": "AROAEXAMPLEID:my-session",
        "Arn": "arn:aws:sts::222222222222:assumed-role/CrossAccountAdmin/my-session"
    }
}
```

### 9. List S3 buckets

```bash
$ aws s3 ls
2026-01-15 10:30:00 my-app-data
2026-02-01 14:22:00 my-app-logs
2026-03-12 09:18:00 my-app-backups
```

### 10. List objects in a path

```bash
$ aws s3 ls s3://my-app-data/uploads/
2026-04-26 10:00:00      12345 photo.jpg
2026-04-26 10:01:00      67890 video.mp4
```

### 11. Upload a file

```bash
$ aws s3 cp file.txt s3://my-app-data/
upload: ./file.txt to s3://my-app-data/file.txt
```

### 12. Sync a directory (with delete and exclude)

```bash
$ aws s3 sync ./local s3://my-app-data/prefix --delete --exclude '*.tmp'
upload: local/a.txt to s3://my-app-data/prefix/a.txt
upload: local/b.txt to s3://my-app-data/prefix/b.txt
delete: s3://my-app-data/prefix/old.txt
```

### 13. Generate a presigned URL

```bash
$ aws s3 presign s3://my-app-data/file.txt --expires-in 3600
https://my-app-data.s3.amazonaws.com/file.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...&X-Amz-Date=20260427T120000Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=...
```

### 14. List bucket names only

```bash
$ aws s3api list-buckets --query 'Buckets[].Name' --output text
my-app-data    my-app-logs    my-app-backups
```

### 15. Enable versioning on a bucket

```bash
$ aws s3api put-bucket-versioning --bucket my-app-data \
    --versioning-configuration Status=Enabled
# (no output on success)
```

### 16. Pretty-print EC2 instances

```bash
$ aws ec2 describe-instances \
    --query 'Reservations[].Instances[].[InstanceId,State.Name,Tags[?Key==`Name`].Value|[0]]' \
    --output table
-------------------------------------------------------------------
|                       DescribeInstances                         |
+----------------+----------+-------------------------------------+
|  i-0abc123456  |  running | web-server-prod                     |
|  i-0def987654  |  stopped | batch-worker-staging                |
|  i-0ghi456789  |  running | database-replica                    |
+----------------+----------+-------------------------------------+
```

### 17. Start instances

```bash
$ aws ec2 start-instances --instance-ids i-0abc123456 i-0def987654
{
    "StartingInstances": [
        {"InstanceId": "i-0abc123456", "CurrentState": {"Code": 0, "Name": "pending"}, "PreviousState": {"Code": 80, "Name": "stopped"}},
        {"InstanceId": "i-0def987654", "CurrentState": {"Code": 0, "Name": "pending"}, "PreviousState": {"Code": 80, "Name": "stopped"}}
    ]
}
```

### 18. Stop an instance

```bash
$ aws ec2 stop-instances --instance-ids i-0abc123456
{
    "StoppingInstances": [
        {"InstanceId": "i-0abc123456", "CurrentState": {"Code": 64, "Name": "stopping"}, "PreviousState": {"Code": 16, "Name": "running"}}
    ]
}
```

### 19. Terminate an instance (permanent)

```bash
$ aws ec2 terminate-instances --instance-ids i-0abc123456
{
    "TerminatingInstances": [
        {"InstanceId": "i-0abc123456", "CurrentState": {"Code": 32, "Name": "shutting-down"}, "PreviousState": {"Code": 16, "Name": "running"}}
    ]
}
```

### 20. Wait until running

```bash
$ aws ec2 wait instance-running --instance-ids i-0abc123456
# (returns silently; exits 0 when running, non-zero if it never starts)
```

### 21. Filter by tag

```bash
$ aws ec2 describe-instances --filters Name=tag:Env,Values=prod \
    --query 'Reservations[].Instances[].InstanceId' --output text
i-0abc123456    i-0jkl098765    i-0mno543210
```

### 22. Describe security groups

```bash
$ aws ec2 describe-security-groups --query 'SecurityGroups[].[GroupName,GroupId]' --output table
```

### 23. List VPCs

```bash
$ aws ec2 describe-vpcs --query 'Vpcs[].[VpcId,CidrBlock,IsDefault]' --output table
```

### 24. List subnets

```bash
$ aws ec2 describe-subnets --query 'Subnets[].[SubnetId,VpcId,AvailabilityZone,CidrBlock]' --output table
```

### 25. List your AMIs

```bash
$ aws ec2 describe-images --owners self \
    --query 'Images[].[ImageId,Name,CreationDate]' --output table
```

### 26. Launch an EC2 instance

```bash
$ aws ec2 run-instances \
    --image-id ami-0abcdef1234567890 \
    --instance-type t3.micro \
    --key-name my-keypair \
    --subnet-id subnet-0abc123 \
    --security-group-ids sg-0xyz789
{
    "Instances": [
        {"InstanceId": "i-0newinstance", "InstanceType": "t3.micro", "State": {"Code": 0, "Name": "pending"}, ...}
    ]
}
```

### 27. List IAM users

```bash
$ aws iam list-users --query 'Users[].[UserName,CreateDate]' --output table
```

### 28. Create an IAM user

```bash
$ aws iam create-user --user-name alice
{
    "User": {
        "UserName": "alice",
        "UserId": "AIDAEXAMPLEUSERID",
        "Arn": "arn:aws:iam::123456789012:user/alice",
        "CreateDate": "2026-04-27T12:00:00+00:00"
    }
}
```

### 29. Create an access key for that user

```bash
$ aws iam create-access-key --user-name alice
{
    "AccessKey": {
        "UserName": "alice",
        "AccessKeyId": "AKIAEXAMPLEALICEKEY",
        "SecretAccessKey": "EXAMPLEALICESECRETKEY",
        "Status": "Active",
        "CreateDate": "2026-04-27T12:01:00+00:00"
    }
}
```

The secret is shown ONCE. Save it now or rotate it.

### 30. Attach a managed policy

```bash
$ aws iam attach-user-policy --user-name alice \
    --policy-arn arn:aws:iam::aws:policy/ReadOnlyAccess
# (no output on success)
```

### 31. Account summary

```bash
$ aws iam get-account-summary
{
    "SummaryMap": {
        "Users": 7,
        "Groups": 3,
        "Roles": 28,
        "Policies": 14,
        "MFADevices": 5,
        ...
    }
}
```

### 32. List Lambda functions

```bash
$ aws lambda list-functions --query 'Functions[].[FunctionName,Runtime,LastModified]' --output table
```

### 33. Invoke a Lambda

```bash
$ aws lambda invoke --function-name my-fn --payload '{"name":"world"}' /tmp/out.json
{
    "StatusCode": 200,
    "ExecutedVersion": "$LATEST"
}
$ cat /tmp/out.json
{"message":"hello, world"}
```

### 34. Update Lambda code from a zip

```bash
$ aws lambda update-function-code --function-name my-fn --zip-file fileb://function.zip
```

(Note `fileb://` for binary; `file://` for text.)

### 35. Deploy a CloudFormation stack

```bash
$ aws cloudformation deploy --template-file template.yaml --stack-name my-stack \
    --parameter-overrides Env=prod Owner=alice
```

### 36. Watch CloudFormation events

```bash
$ aws cloudformation describe-stack-events --stack-name my-stack \
    --query 'StackEvents[].[Timestamp,ResourceStatus,ResourceType,LogicalResourceId,ResourceStatusReason]' \
    --output table --max-items 20
```

### 37. ECS list clusters

```bash
$ aws ecs list-clusters
{
    "clusterArns": [
        "arn:aws:ecs:us-east-1:123456789012:cluster/prod",
        "arn:aws:ecs:us-east-1:123456789012:cluster/staging"
    ]
}
```

### 38. ECS describe a service

```bash
$ aws ecs describe-services --cluster prod --services web
```

### 39. EKS update kubeconfig

```bash
$ aws eks update-kubeconfig --name my-cluster --region us-east-1
Updated context arn:aws:eks:us-east-1:123456789012:cluster/my-cluster in /Users/alice/.kube/config
$ kubectl get nodes
NAME                              STATUS   ROLES    AGE   VERSION
ip-10-0-1-23.ec2.internal         Ready    <none>   3d    v1.28.5-eks-...
ip-10-0-2-45.ec2.internal         Ready    <none>   3d    v1.28.5-eks-...
```

### 40. RDS describe DBs

```bash
$ aws rds describe-db-instances --query 'DBInstances[].[DBInstanceIdentifier,Engine,DBInstanceStatus]' --output table
```

### 41. DynamoDB list tables

```bash
$ aws dynamodb list-tables
{
    "TableNames": [
        "users",
        "orders",
        "products"
    ]
}
```

### 42. DynamoDB scan (sample 10)

```bash
$ aws dynamodb scan --table-name users --max-items 10
{
    "Items": [
        {"id":{"S":"u-001"},"name":{"S":"Alice"}},
        {"id":{"S":"u-002"},"name":{"S":"Bob"}},
        ...
    ],
    "Count": 10,
    "ScannedCount": 10
}
```

### 43. SQS list queues

```bash
$ aws sqs list-queues
{
    "QueueUrls": [
        "https://sqs.us-east-1.amazonaws.com/123456789012/work-queue"
    ]
}
```

### 44. SQS receive messages

```bash
$ aws sqs receive-message \
    --queue-url https://sqs.us-east-1.amazonaws.com/123456789012/work-queue \
    --max-number-of-messages 10
{
    "Messages": [
        {
            "MessageId": "abc-123",
            "ReceiptHandle": "AQEBxxxx...",
            "Body": "{\"task\":\"resize-image\"}",
            "MD5OfBody": "..."
        }
    ]
}
```

### 45. SNS list topics

```bash
$ aws sns list-topics
```

### 46. KMS list keys

```bash
$ aws kms list-keys
$ aws kms list-aliases --query 'Aliases[].[AliasName,TargetKeyId]' --output table
```

### 47. Encrypt with KMS

```bash
$ aws kms encrypt --key-id alias/myapp --plaintext fileb://secret.bin \
    --output text --query CiphertextBlob | base64 -d > encrypted.bin
$ ls -la encrypted.bin
-rw-r--r--  1 alice  staff  256 Apr 27 12:34 encrypted.bin
```

### 48. Decrypt with KMS

```bash
$ aws kms decrypt --ciphertext-blob fileb://encrypted.bin \
    --output text --query Plaintext | base64 -d
hunter2
```

### 49. SSM get parameter

```bash
$ aws ssm get-parameter --name /myapp/db/password --with-decryption \
    --query 'Parameter.Value' --output text
super-secret-password
```

### 50. SSM put parameter (SecureString)

```bash
$ aws ssm put-parameter --name /myapp/db/password --type SecureString \
    --value 'new-password' --overwrite
{
    "Version": 2,
    "Tier": "Standard"
}
```

### 51. Secrets Manager get secret

```bash
$ aws secretsmanager get-secret-value --secret-id my-secret \
    --query SecretString --output text
{"username":"alice","password":"hunter2"}
```

### 52. Session Manager — shell into an EC2

```bash
$ aws ssm start-session --target i-0abc123456
Starting session with SessionId: alice-abc123
sh-4.2$ whoami
ssm-user
sh-4.2$ exit
Exiting session with sessionId: alice-abc123.
```

### 53. Tail Lambda logs

```bash
$ aws logs tail /aws/lambda/my-fn --follow --since 1h
2026-04-27T12:00:00 START RequestId: 123-abc Version: $LATEST
2026-04-27T12:00:00 hello from my-fn
2026-04-27T12:00:01 END RequestId: 123-abc
2026-04-27T12:00:01 REPORT RequestId: 123-abc Duration: 23.45 ms ...
```

### 54. CloudWatch metric

```bash
$ aws cloudwatch get-metric-statistics \
    --namespace AWS/EC2 \
    --metric-name CPUUtilization \
    --dimensions Name=InstanceId,Value=i-0abc123456 \
    --start-time 2026-04-26T00:00:00 \
    --end-time 2026-04-27T00:00:00 \
    --period 300 \
    --statistics Average \
    --query 'Datapoints[].[Timestamp,Average]' \
    --output table
```

## Common Confusions

The traps people fall into. Memorize these.

### v1 vs v2

Only use v2. v1 is end-of-life. Homebrew's `awscli` formula installs v2. PyPI's `awscli` package installs v1 — avoid.

### Credential provider chain order

CLI flag → env vars → credentials file → config file → ECS metadata → EC2 instance metadata. **First match wins.** A leftover `AWS_ACCESS_KEY_ID` in your shell will silently override your `[default]` profile and you will spend an hour debugging.

### `--profile` vs `AWS_PROFILE`

`--profile` (CLI flag) wins over `AWS_PROFILE` (env var). `AWS_PROFILE` wins over `AWS_DEFAULT_PROFILE` (older name). All of those override the `[default]` profile in your config files.

### Long-term IAM keys vs short-term STS

Long-term keys start with `AKIA`. Short-term STS credentials start with `ASIA` and require an `AWS_SESSION_TOKEN`. If you have `ASIA...` and no session token, every request will fail with `InvalidClientTokenId` or `SignatureDoesNotMatch`.

### AssumeRole duration cap

The first AssumeRole in a chain can run up to `MaxSessionDuration` of the role (1-12 hours). **Chained AssumeRole calls are hard-capped at 1 hour** regardless. Plan accordingly.

### Regional vs global services

IAM, Route 53, CloudFront, AWS Organizations — global; the region in your config doesn't matter for these. S3 bucket names — global namespace, but data lives in a specific region. Almost everything else — regional.

### `--query` vs `--filter`

`--query` is **client-side**, JMESPath, works on every command. `--filters` is **server-side**, only on certain commands (mostly EC2), faster. Use `--filters` when available; combine both when useful.

### Pagination is automatic

By default the CLI walks every page and returns all items. For huge result sets this can be slow. Use `--max-items 50` for a peek or `--no-paginate` to stop after the first server page.

### Throttling and retries

When you hit a rate limit, the CLI retries with **exponential backoff** automatically. Three retry modes: `legacy` (the old default), `standard` (v2 default), `adaptive` (rate-aware client-side throttle). Set with `AWS_RETRY_MODE=adaptive`.

### Signature errors

`SignatureDoesNotMatch` → bad secret key, bad clock, or proxy mangling headers. `RequestExpired` → clock more than 5-15 minutes off. Run NTP.

### SSO vs IAM users

SSO (IAM Identity Center) gives you short-term tokens after a browser login. IAM users have long-term keys. SSO is recommended for humans; IAM users are mostly legacy.

### Instance profile vs `ec2-instance-role`

When you "attach a role to an EC2," AWS creates an *instance profile* that wraps the role. The role has the policy; the instance profile is just the wrapper that allows EC2 to use it. You name them the same usually, so the distinction rarely matters — but if you ever see a "role exists but instance profile doesn't" error, that's why.

### Which env var sets the region

`AWS_REGION` (preferred) and `AWS_DEFAULT_REGION` (older name) both work. The CLI honors either, with `AWS_REGION` winning if both are set.

### SAM CLI vs `cloudformation deploy`

Both deploy CloudFormation. **AWS SAM** (Serverless Application Model) extends CloudFormation with shortcuts for serverless apps (Lambda, API Gateway, DynamoDB). The SAM CLI also runs Lambdas locally and packages assets. For pure CloudFormation, use `aws cloudformation deploy`. For serverless, `sam deploy` is friendlier.

### `s3` vs `s3api`

`aws s3 ...` is the high-level command (cp, sync, ls, mb, rb). `aws s3api ...` is the low-level command that maps 1:1 to S3 REST operations (head-object, list-buckets, put-bucket-policy). `s3` is for moving data; `s3api` is for managing bucket configuration.

### `file://` vs `fileb://`

`file://` reads a file as text (UTF-8). `fileb://` reads as raw bytes. For zip files, binaries, encrypted blobs, use `fileb://`. For JSON, YAML, plain text, use `file://`. Wrong one gives `could not decode UTF-8` errors.

### `--cli-binary-format`

The CLI v2 defaults to `base64` for binary blob inputs/outputs. v1 defaulted to `raw-in-base64-out`. If a v1 script is failing, set:

```bash
$ aws configure set cli_binary_format raw-in-base64-out
```

## Vocabulary

Plain-English definitions. If a word in this sheet (or in any AWS documentation) confused you, look here.

| Term | Meaning |
| --- | --- |
| `aws` | The CLI binary you type. |
| `awscli v1` | Old version (Python-based). End-of-life. Don't use. |
| `awscli v2` | Current version. Single binary. Always use this. |
| `aws-shell` | An older interactive REPL for the CLI. Mostly replaced by v2's auto-prompt. |
| `aws-vault` | Third-party tool that stores AWS keys in your OS keychain instead of `~/.aws/credentials`. Popular. |
| `aws-okta` | Older third-party tool for SSO; mostly superseded by `aws sso login`. |
| IAM | Identity and Access Management. The "who can do what" service. |
| IAM user | A long-term identity with name + password + access keys. Mostly legacy now. |
| IAM group | A bag of users that share policies. |
| IAM role | An identity that's *assumed*; produces short-term credentials. The modern way. |
| IAM policy | A JSON document of allow/deny statements. |
| Managed policy | A reusable policy you can attach to many users/roles. AWS-managed or customer-managed. |
| Inline policy | A policy embedded inside one user/role only. |
| Identity-based policy | Attached to a user/role; says what *they* can do. |
| Resource-based policy | Attached to a resource; says who can do what *to it*. |
| Trust policy | Special resource-based policy on a role; says who is allowed to assume it. |
| Permissions boundary | A ceiling on what a user/role *could ever* do. |
| SCP | Service Control Policy — top-level deny-only guardrail at the Organizations level. |
| AWS Organizations | A way to group many AWS accounts under one billing umbrella. |
| OU | Organizational Unit — a folder of accounts inside an Organization. |
| Account | An AWS account. The blast-radius unit. Has its own bill and its own resources. |
| Account ID | 12-digit number identifying an AWS account. Public-ish (shows up in ARNs). |
| ARN | Amazon Resource Name. Looks like `arn:aws:s3:::bucket/key`. The unique ID for any AWS thing. |
| Root user | The email-and-password owner of an AWS account. Use only for billing. Lock with MFA. |
| MFA | Multi-Factor Authentication. A second factor (phone app, hardware key) on top of password. |
| AWS IAM Identity Center | Formerly AWS SSO. Federated login that hands out short-term creds to humans. |
| Permission set | An IAM Identity Center bundle of policies for a role across accounts. |
| OIDC federation | OpenID Connect federation. CI presents a JWT, AWS exchanges for creds. |
| SAML federation | SAML 2.0 federation. Older enterprise SSO. |
| Web identity federation | Federation via Google/Facebook/etc. Mostly used by mobile apps. |
| AssumeRole | The STS API to swap your identity for a role's credentials. |
| AssumeRoleWithSAML | AssumeRole using a SAML assertion. |
| AssumeRoleWithWebIdentity | AssumeRole using a JWT (e.g., from GitHub Actions OIDC). |
| GetSessionToken | Older STS API; swaps long-term creds for short-term ones. |
| STS | Security Token Service. The service that hands out short-term credentials. |
| Session token | The third credential piece needed when using temporary STS credentials. |
| Signed request | An HTTP request with an `Authorization` header containing a SigV4 signature. |
| SigV4 | AWS Signature Version 4. The hash that proves it's really you. |
| `X-Amz-Security-Token` | HTTP header that carries the session token on signed requests. |
| Access advisor | The IAM feature that shows which services a principal hasn't used in N days. |
| Last-accessed-info | Same idea: timestamps of last use per service. |
| Profile | A named bundle of credentials and settings in `~/.aws/config` and `~/.aws/credentials`. |
| Named profile | Any profile other than `default`. |
| Default profile | The `[default]` block, used when no `--profile` and no `AWS_PROFILE`. |
| Region | A geographic AWS area. Resources are scoped per region (mostly). |
| Partition | A walled-off corner of AWS. `aws` (commercial), `aws-us-gov` (US government), `aws-cn` (China). |
| Endpoint | The HTTPS URL the CLI talks to, e.g., `s3.us-east-1.amazonaws.com`. |
| Regional endpoint | Endpoint for a specific region. |
| FIPS endpoint | Endpoint that uses FIPS 140-2 validated cryptography. Required for some federal workloads. |
| Dualstack | Endpoint that supports both IPv4 and IPv6. |
| IPv6 endpoint | Endpoint reachable over IPv6 only. |
| S3 | Simple Storage Service. Object storage. |
| Bucket | A top-level S3 container with a globally-unique name. |
| Object key | The "filename" inside a bucket. May contain slashes. |
| Prefix | A leading portion of object keys, used to filter. Like a folder, but not really. |
| Presigned URL | A short-lived HTTPS URL that grants access to one S3 object. |
| Multi-part upload | Splitting a big file into chunks and uploading them in parallel. |
| Intelligent Tiering | S3 storage class that auto-tiers between hot and cold based on access. |
| IA | Infrequent Access. A cheaper, slower S3 tier. |
| Glacier | Cold-storage S3 tier. Hours to retrieve. Very cheap to store. |
| Glacier Deep Archive | Even colder than Glacier. ~12 hours to retrieve. Even cheaper. |
| Lifecycle policy | Bucket rule to transition or delete objects after N days. |
| Replication | Auto-copy of new objects from one bucket to another. |
| MRAP | Multi-Region Access Point. One name resolves to the closest S3 region. |
| VPC | Virtual Private Cloud. A private network inside AWS. |
| Subnet | A slice of a VPC's IP range, scoped to one Availability Zone. |
| Public subnet | Subnet with a route to the internet via Internet Gateway. |
| Private subnet | Subnet with no direct internet route. |
| Internet Gateway | The thing that lets a VPC talk to the internet. |
| NAT Gateway | Lets private-subnet instances reach the internet outbound only. |
| VPC endpoint | A private path from your VPC to AWS services without going over the internet. |
| Gateway endpoint | A free type of VPC endpoint, only for S3 and DynamoDB. |
| Interface endpoint | A paid type of VPC endpoint, for most other services. Backed by ENIs. |
| Transit Gateway | A hub that connects many VPCs and on-prem networks. |
| Route table | A VPC's list of "for this CIDR, send traffic out via this gateway." |
| Security group | A stateful firewall around an instance/ENI. |
| NACL | Network Access Control List. Stateless subnet-level firewall. |
| EC2 | Elastic Compute Cloud. Rented servers. |
| EBS | Elastic Block Store. Persistent disks for EC2. |
| EFS | Elastic File System. Managed NFS. |
| FSx | Managed file systems for Lustre / Windows / NetApp / OpenZFS. |
| AMI | Amazon Machine Image. The OS+software template you boot an instance from. |
| Instance type | The size: `t3.micro`, `m5.large`, etc. |
| t3 / m5 / c5 / r5 / x2 / p4 / inf / trn | Instance families: burstable, general, compute, memory, hi-mem, GPU, ML inference, ML training. |
| Spot | Cheap, interruptible instances. Up to 90% off. |
| Reserved Instance | Pre-paid instance discount. Locked for 1 or 3 years. |
| Savings Plan | Newer, more flexible commitment-based discount. |
| Auto Scaling | Group that adds/removes instances based on rules. |
| ECS | Elastic Container Service. AWS's own container orchestrator. |
| ECR | Elastic Container Registry. Docker image registry. |
| EKS | Elastic Kubernetes Service. Managed Kubernetes. |
| Fargate | Serverless containers. You don't manage the EC2. |
| App Runner | Even more serverless: deploy from a Git repo or image, AWS handles the rest. |
| Lightsail | Simplified VPS. Like DigitalOcean droplets but on AWS. |
| Elastic Beanstalk | Older "deploy your code, we run it" PaaS. |
| Lambda | Serverless functions. Pay per millisecond. |
| Step Functions | State machine orchestrator over Lambdas and other services. |
| EventBridge | Event bus. Schedules and routes events between AWS services. |
| SQS | Simple Queue Service. FIFO/standard message queues. |
| SNS | Simple Notification Service. Pub/sub. |
| Kinesis | Streaming data ingestion (like Kafka). |
| MSK | Managed Streaming for Kafka. Actual Apache Kafka, managed. |
| RDS | Relational Database Service. Managed Postgres/MySQL/etc. |
| Aurora | AWS's own MySQL/Postgres-compatible engine. Faster, more expensive. |
| DynamoDB | NoSQL key-value/document database. Serverless. |
| ElastiCache | Managed Redis or Memcached. |
| Redshift | Data warehouse. Columnar Postgres-flavored. |
| Athena | Query S3 with SQL via Presto. Pay per query. |
| Glue | ETL service. Crawls data and runs transforms. |
| EMR | Elastic MapReduce. Hadoop/Spark clusters. |
| S3 Select | Run SQL on a single S3 object. |
| OpenSearch | Managed Elasticsearch fork. |
| KMS | Key Management Service. Encryption keys. |
| CMK | Customer Master Key (older term for KMS keys). |
| AWS-managed key | KMS key managed by AWS. You can't see its policy. |
| Customer-managed key | KMS key you own. You control the policy. |
| KMS alias | A friendly name like `alias/myapp` for a KMS key. |
| GenerateDataKey | KMS API that returns plaintext + encrypted versions of a fresh data key. |
| Encryption context | Extra "associated data" on KMS calls; used in audit and conditions. |
| Key policy | The resource-based policy on a KMS key. |
| Grant | A short-term permission on a KMS key, separate from the key policy. |
| KMS Multi-Region Keys | KMS keys that exist with the same key material across multiple regions. |
| Secrets Manager | Vault for app secrets. Auto-rotation supported. |
| Parameter Store (SSM) | Cheaper config + small secrets store. Part of Systems Manager. |
| Systems Manager | Umbrella service for fleet management: Run Command, Patch Manager, Session Manager, etc. |
| Run Command | SSM feature: execute a script across many instances. |
| Session Manager | SSM feature: shell into an EC2 without SSH. |
| Patch Manager | SSM feature: apply OS patches on a schedule. |
| State Manager | SSM feature: enforce desired-state config. |
| CloudFormation | AWS's native IaC. YAML/JSON templates. |
| CDK | Cloud Development Kit. Write CloudFormation in real programming languages. |
| AWS CDK v2 | Current major version of CDK. |
| Terraform | Third-party IaC by HashiCorp. Multi-cloud. |
| Pulumi | Third-party IaC. Real programming languages. |
| SAM | Serverless Application Model. CloudFormation extension for serverless. |
| Serverless Framework | Third-party IaC focused on Lambda + API Gateway. |
| AWS SDK | Library that wraps AWS APIs in your programming language. CLI is itself an SDK user. |
| boto3 | Python SDK. The most-popular AWS SDK. |
| botocore | The core Python library that boto3 and the AWS CLI v1 share. |
| AWS SDK for JavaScript v3 | Modular JS/TS SDK. The current one. |
| AWS SDK for Go v2 | Current Go SDK. |
| AWS SDK for Java 2.x | Current Java SDK. |
| AWS SDK for .NET | Current .NET SDK. |
| AWS Tools for PowerShell | Cmdlets for Windows admins. |
| jq | Third-party command-line JSON processor. Pairs well with `aws --output json`. |
| JMESPath | The query language used by `--query`. |
| `--query` | CLI flag for client-side JMESPath filtering. |
| `--filters` | CLI flag for server-side filtering (only on certain commands). |
| `--output` | CLI flag to pick output format. |
| `--output table` | Pretty boxes. Slow on huge results. |
| `--output yaml` | YAML formatting. |
| `--output text` | Tab-separated. Easy to pipe. |
| `--output json` | Default. Machine-readable. |
| `--no-cli-pager` | Stop the CLI from invoking `less` on long output. |
| `AWS_PAGER` | Env var to control or disable the pager. |
| Retry mode | How aggressively the CLI retries throttled requests. `legacy` / `standard` / `adaptive`. |
| `max_attempts` | Maximum retries per request. |
| `AWS_SDK_LOAD_CONFIG` | Old SDK env var to enable shared-config loading. v2 SDKs always load it. |
| `AWS_PROFILE` | Active profile name. |
| `AWS_DEFAULT_PROFILE` | Older alias for `AWS_PROFILE`. |
| `AWS_REGION` | Active region. |
| `AWS_DEFAULT_REGION` | Older alias for `AWS_REGION`. |
| `AWS_DEFAULT_OUTPUT` | Default output format. |
| `AWS_ENDPOINT_URL` | Override the service endpoint (LocalStack, etc.). |
| `AWS_USE_FIPS_ENDPOINT` | Enable FIPS endpoints. |
| `AWS_USE_DUALSTACK_ENDPOINT` | Enable IPv4+IPv6 endpoints. |
| `AWS_CA_BUNDLE` | Path to a custom CA cert bundle (for corporate proxies). |
| `AWS_CONFIG_FILE` | Override the path to `~/.aws/config`. |
| `AWS_SHARED_CREDENTIALS_FILE` | Override the path to `~/.aws/credentials`. |
| `AWS_SDK_LOG_LEVEL` | Verbosity for SDK debug logs. |
| `AWS_RETRY_MODE` | `legacy` / `standard` / `adaptive`. |

## ASCII Diagrams

### Credential provider chain

```
                +--------------------------------------------+
   aws CLI ---> | 1. CLI flags  --profile  --region          |
                +--------------------------------------------+
                | 2. Env vars   AWS_ACCESS_KEY_ID, etc.      |
                +--------------------------------------------+
                | 3. ~/.aws/credentials  (active profile)    |
                +--------------------------------------------+
                | 4. ~/.aws/config       (SSO, role chain)   |
                +--------------------------------------------+
                | 5. Container metadata  (ECS/EKS/Fargate)   |
                +--------------------------------------------+
                | 6. Instance metadata   (EC2 IMDS v2)       |
                +--------------------------------------------+
                            |
                            v
                first match wins. otherwise:
                  "Unable to locate credentials"
```

### SigV4 request flow

```
         (your laptop)                              (AWS)
   +-------------------+                     +-----------------+
   |  CLI builds:      |                     |  service        |
   |   - method/path   |                     |  endpoint       |
   |   - headers       |    HTTPS request    |                 |
   |   - body          | ------------------> |                 |
   |   - timestamp     |    Authorization:   |  1. parse       |
   |   - region/svc    |     AWS4-HMAC-SHA256|  2. fetch your  |
   |                   |     Credential=...  |     secret      |
   |  hash with secret |     Signature=...   |  3. recompute   |
   |  -> signature     |                     |     signature   |
   +-------------------+                     |  4. compare     |
                                             |  5. if match,   |
                                             |     execute     |
                                             |  6. return XML  |
                                             |     or JSON     |
                                             +-----------------+
                                                     |
   +-------------------+                              v
   |  CLI parses,      | <-------------------- response
   |  applies --query, |
   |  prints           |
   +-------------------+
```

### Profile resolution decision tree

```
                +----------------------+
                |  --profile X passed? |
                +----------+-----------+
                           |
              YES   <------+------>   NO
               |                       |
               v                       v
            use X         +-------------------------+
                          |  AWS_PROFILE set?       |
                          +------------+------------+
                                       |
                          YES   <------+------>   NO
                           |                       |
                           v                       v
                       use $AWS_PROFILE  +-------------------------+
                                         |  AWS_DEFAULT_PROFILE?   |
                                         +------------+------------+
                                                      |
                                          YES  <------+------>  NO
                                           |                     |
                                           v                     v
                                  use $AWS_DEFAULT_PROFILE   use [default]
```

### S3 multi-part upload

```
   +-----------+
   | big.tar   |   3.5 GB
   +-----------+
        |
   split into 5 MB parts
        |
        v
   +---+---+---+---+---+---+ ... +---+
   | 1 | 2 | 3 | 4 | 5 | 6 | ... |700|
   +---+---+---+---+---+---+ ... +---+
        |
        | 1. CreateMultipartUpload(bucket, key) -> uploadId
        v
   +-------+
   |  S3   |
   +-------+
        ^
        | 2. UploadPart(uploadId, n=1..700, body=part_n) -> ETag_n
        |    (parts uploaded in parallel)
        |
        | 3. CompleteMultipartUpload(uploadId, [(1,ETag1)..(700,ETag700)])
        |    -> final ETag
        v
   +-------+
   |  S3   |  one stitched-together object
   +-------+
```

### AssumeRole chain

```
   +---------+
   | user A  |   long-term keys (AKIA...)
   +----+----+
        |
        | sts:AssumeRole(arn:.../RoleB, duration=12h)
        v
   +---------+
   |  RoleB  |   short-term creds (ASIA..., 12h max)
   +----+----+
        |
        | sts:AssumeRole(arn:.../RoleC, duration=1h max)
        v          (chained AssumeRole is hard-capped at 1h)
   +---------+
   |  RoleC  |   short-term creds (ASIA..., 1h)
   +---------+
```

## Version Notes

- **v1** — Python-based, available 2013-2024, **end-of-life**. Don't use.
- **v2 GA — Feb 2020.** Single binary, native SSO, auto-prompt, faster startup, YAML output.
- **v2.13+** — IPv6 / dualstack endpoint support via `AWS_USE_DUALSTACK_ENDPOINT=true`.
- **v2.15+** — FIPS endpoints configurable from `~/.aws/config` via `use_fips_endpoint = true`.
- **v2.17+** — Refreshable SSO tokens; long-running sessions don't force a re-login on every short expiry.
- **v2.x ongoing** — New service support is added monthly. Check the v2 release notes (`https://github.com/aws/aws-cli/blob/v2/CHANGELOG.rst`) for what's new.

When upgrading, run `aws --version` after to confirm. Pinning v2.x.y in CI is fine; AWS doesn't break v2 within minor versions.

## Try This

A series of warm-up tasks. Type each one. Read the output. If you don't understand it, scroll back up and re-read the relevant section.

1. `aws --version` — confirm you have v2.
2. `aws configure` — set up a profile, even if you make up the keys for now (they won't work, but the file gets created).
3. Open `~/.aws/credentials` in an editor. Look at the format. Add a fake `[work]` profile. Save.
4. `aws configure list-profiles` — your fake profile should appear.
5. `aws configure list` — show what's currently active.
6. Set `export AWS_REGION=us-west-2` in your shell. Run `aws configure list` again. Watch which line changed.
7. `aws sts get-caller-identity` — if your real keys are good, this prints your identity. If not, you get `InvalidClientTokenId`.
8. `aws s3 ls` — list buckets. Empty if you have none.
9. Make a bucket: `aws s3 mb s3://my-test-bucket-$RANDOM-$(date +%s)`. The `$RANDOM` and date make it globally unique.
10. Upload a file: `echo hello > a.txt && aws s3 cp a.txt s3://your-bucket/`.
11. Download it back: `aws s3 cp s3://your-bucket/a.txt b.txt && cat b.txt`.
12. Generate a presigned URL: `aws s3 presign s3://your-bucket/a.txt --expires-in 60`. Open it in a browser within 60 seconds.
13. Try `aws ec2 describe-instances`. If you have none, you get `Reservations: []`. That's fine.
14. Try `aws ec2 describe-instances --output table` to see the same data prettier.
15. Try `aws ec2 describe-instances --query 'Reservations[].Instances[].[InstanceId,State.Name]' --output text` to slice it.
16. `aws iam list-users --query 'Users[].UserName' --output text` — list IAM user names.
17. Set `AWS_PAGER=""` in your shell. Re-run a command. Notice no `less`.
18. Try `aws sso login --profile X` if you have an SSO setup. Otherwise skip.
19. Tear down: `aws s3 rm s3://your-bucket/a.txt && aws s3 rb s3://your-bucket`. Always clean up to avoid surprise charges.
20. Read your shell history with `history | grep aws` and notice how readable a chain of `aws` commands is.

## Where to Go Next

You now know enough CLI to do real work. To go deeper:

- **Read `cloud/aws-cli`** in this same `cs` tool for a denser reference.
- **Read `cloud/vpc`** to understand the networking under EC2.
- **Read `cloud/cloud-dns`** for Route 53 deep dive.
- **Read `ramp-up/terraform-eli5`** to start defining infrastructure as code.
- **Read `ramp-up/kubernetes-eli5`** if you'll be using EKS.
- **Read `ramp-up/docker-eli5`** if you'll push images to ECR.
- **Bookmark `https://docs.aws.amazon.com/cli/`.** The reference for every subcommand and flag is there.
- **Subscribe to the AWS What's New feed** at `https://aws.amazon.com/new/` to keep up with new services.
- **Watch a re:Invent talk per week.** Search YouTube for `AWS re:Invent <topic>`. Free, expert-quality.

A learning ladder, in rough order:

1. `aws sts get-caller-identity` and `aws s3 ls` — be sure your auth works.
2. `aws s3 sync` — manage real files.
3. `aws ec2 describe-instances` and `aws ec2 run-instances` — manage real servers.
4. `aws iam` family — read your own permissions.
5. `aws cloudformation deploy` — describe a stack as YAML.
6. `aws lambda invoke` — your first serverless call.
7. `aws ssm start-session` — never SSH again.
8. `aws logs tail --follow` — watch a service in real time.
9. `aws sso configure` and `aws sso login` — the modern human auth.
10. Write a bash script that strings several commands together. Save it. Use it next week.

## See Also

- `cloud/aws-cli` — denser reference for daily lookup.
- `cloud/gcloud` — the equivalent for Google Cloud.
- `cloud/azure-cli` — the equivalent for Microsoft Azure.
- `cloud/cloud-dns` — DNS management across clouds; Route 53 sits here.
- `cloud/vpc` — virtual networking on AWS.
- `ramp-up/kubernetes-eli5` — Kubernetes from zero, useful if you'll be using EKS.
- `ramp-up/terraform-eli5` — IaC across many clouds.
- `ramp-up/docker-eli5` — containers, images, registries.
- `ramp-up/linux-kernel-eli5` — what's actually running on your EC2 instances.
- `ramp-up/bash-eli5` — the shell you're typing `aws` into.

## References

- `https://docs.aws.amazon.com/cli/latest/userguide/` — official AWS CLI v2 user guide.
- `https://docs.aws.amazon.com/cli/latest/reference/` — every subcommand, every flag.
- `https://github.com/aws/aws-cli/blob/v2/CHANGELOG.rst` — v2 release notes.
- `https://jmespath.org/` — JMESPath specification, with a live tester.
- `https://jmespath.org/tutorial.html` — JMESPath tutorial.
- "AWS CLI in Action" by Andreas Wittig — book-length deep dive.
- "AWS Cookbook" by John Culkin and Mike Zazon — recipe-style task guide.
- AWS re:Invent talks (YouTube, free) — search "AWS re:Invent <service>" for any service.
- `https://aws.amazon.com/new/` — What's New feed; subscribe via RSS to keep up.
- `https://aws.amazon.com/security/security-bulletins/` — security advisories worth watching.
- `https://docs.aws.amazon.com/IAM/latest/UserGuide/` — IAM user guide; the deep IAM material lives here.
- `https://docs.aws.amazon.com/general/latest/gr/sigv4-signed-request-examples.html` — SigV4 request examples.
- `https://aws.amazon.com/architecture/well-architected/` — AWS Well-Architected Framework, the canonical "is my setup good" checklist.
