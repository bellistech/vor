# npm (Node Package Manager)

> Package manager for JavaScript/Node.js — install dependencies, run scripts, and manage projects.

## Project Init

```bash
npm init                               # interactive project setup
npm init -y                            # accept all defaults
npm init @scope                        # scoped package
```

## Install

### Dependencies

```bash
npm install                            # install all from package.json
npm install express                    # add production dependency
npm install -D typescript              # add dev dependency (--save-dev)
npm install -g typescript              # install globally
npm install express@4.18.2             # exact version
npm install "express@^4.18.0"          # semver range
npm install express@latest             # latest version
npm install express@next               # next/pre-release tag
npm install ./local-package            # install local package
npm install git+https://github.com/user/repo.git  # from git
```

### Options

```bash
npm install --production               # skip devDependencies
npm install --ignore-scripts           # skip pre/post install scripts
npm install --legacy-peer-deps         # ignore peer dependency conflicts
npm install --force                    # force install (ignore cache)
npm ci                                 # clean install from lockfile (CI/CD)
npm install --package-lock-only        # update lockfile without installing
```

## Uninstall

```bash
npm uninstall express                  # remove and update package.json
npm uninstall -g typescript            # remove global package
npm uninstall -D jest                  # remove dev dependency
```

## Update

```bash
npm update                             # update all (within semver range)
npm update express                     # update specific package
npm outdated                           # show outdated packages
npm outdated -g                        # show outdated global packages

# update beyond semver range
npx npm-check-updates -u              # update package.json ranges
npm install                            # then install new versions
```

## Scripts

### Running

```bash
npm run build                          # run script from package.json
npm run test                           # run test script
npm test                               # shorthand for npm run test
npm start                              # shorthand for npm run start
npm run dev -- --port 3000             # pass args to script (after --)
npm run                                # list available scripts
```

### package.json Scripts

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "test": "vitest",
    "lint": "eslint src/",
    "format": "prettier --write src/",
    "pretest": "npm run lint",
    "postbuild": "echo 'Build complete'"
  }
}
```

## npx (Execute Packages)

```bash
npx create-react-app my-app            # run without installing globally
npx -y cowsay "hello"                  # auto-confirm install
npx --package=typescript tsc --init    # use specific package
npx -p node@18 node --version          # run with different node version
npx --yes degit user/repo my-project   # scaffold from template
```

## Workspaces (Monorepo)

### Setup

```json
{
  "name": "monorepo",
  "workspaces": [
    "packages/*",
    "apps/*"
  ]
}
```

### Commands

```bash
npm install                            # install all workspace deps
npm install lodash -w packages/utils   # add dep to specific workspace
npm run build --workspaces             # run in all workspaces
npm run build -w packages/core         # run in specific workspace
npm run test --workspaces --if-present # run only if script exists
npm ls --workspaces                    # list deps across workspaces
```

## Audit (Security)

```bash
npm audit                              # check for vulnerabilities
npm audit fix                          # auto-fix vulnerabilities
npm audit fix --force                  # fix with breaking changes
npm audit --json                       # JSON output
npm audit --omit=dev                   # skip devDependencies
npm audit signatures                   # verify package signatures
```

## Info and List

```bash
npm info express                       # package metadata
npm info express versions              # all available versions
npm view express dependencies          # show dependencies
npm ls                                 # dependency tree (local)
npm ls --all                           # full dependency tree
npm ls --depth=0                       # top-level only
npm ls express                         # find where express is used
npm explain express                    # explain why package is installed
npm ls -g --depth=0                    # list global packages
```

## Pack and Publish

```bash
npm pack                               # create tarball
npm pack --dry-run                     # show what would be packed
npm publish                            # publish to registry
npm publish --access public            # publish scoped package as public
npm publish --tag beta                 # publish with tag
npm unpublish package@1.0.0            # remove version (72h window)
npm deprecate package@"<2.0" "Use v2"  # deprecate versions
npm version patch                      # bump patch version (1.0.0 -> 1.0.1)
npm version minor                      # bump minor (1.0.0 -> 1.1.0)
npm version major                      # bump major (1.0.0 -> 2.0.0)
```

## Config

```bash
npm config list                        # show current config
npm config get registry                # show specific setting
npm config set registry https://registry.npmjs.org/
npm config set save-exact true         # pin exact versions on install
npm config delete registry             # remove setting
npm config edit                        # edit config file

# .npmrc (per-project or user config)
# ~/.npmrc or ./.npmrc
registry=https://registry.npmjs.org/
save-exact=true
engine-strict=true
@myorg:registry=https://npm.pkg.github.com/
```

## Cache

```bash
npm cache ls                           # list cached packages
npm cache clean --force                # clear cache
npm cache verify                       # verify cache integrity
```

## Tips

- `npm ci` is faster and more reliable than `npm install` in CI/CD. It deletes `node_modules` and installs from the lockfile exactly.
- `--save-exact` or `save-exact=true` in .npmrc prevents semver surprises by pinning exact versions.
- `npx` runs a package binary without permanent global install. Great for scaffolding tools and one-off commands.
- `npm ls --depth=0` shows only your direct dependencies, cutting through the noise.
- `npm explain <pkg>` shows the full dependency chain that caused a package to be installed.
- `pre` and `post` lifecycle scripts run automatically: `pretest` runs before `test`, `postbuild` runs after `build`.
- `npm audit fix --force` can introduce breaking changes. Always review the diff before committing.
- `package-lock.json` should be committed to version control. It ensures reproducible installs.
- Use `--ignore-scripts` when installing untrusted packages to prevent arbitrary code execution.
- `npm outdated` shows current, wanted (semver-compatible), and latest versions side by side.
