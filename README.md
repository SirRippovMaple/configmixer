# configmixer

`configmixer` generates a `.env` file by mixing together one "mixin" from each
of several disjoint categories (environment, tenant, OAuth app, OAuth
account, ...). Some mixins are only compatible with certain others — e.g. a
`prod` OAuth app only makes sense with a `prod` environment. configmixer
enforces those compatibility rules and merges the chosen mixins' key/value
pairs into a single `.env` file.

The set of categories isn't hardcoded: it's just whatever subdirectories
exist under your mixins directory, so the same binary works for any domain.

## Install

### Homebrew

```sh
brew tap SirRippovMaple/tap
brew install configmixer
```

### From source

```sh
go build -o configmixer .
```

## Mixin directory layout

One subdirectory per category, one YAML file per mixin. The directory name
is the category; the filename (minus extension) is the mixin's id.

```
mixins/
  environment/
    dev.yaml
    prod.yaml
  tenant/
    acme.yaml
    globex.yaml
  oauth-app/
    dev.yaml
    prod.yaml
  oauth-account/
    main.yaml
```

### Mixin file format

```yaml
tags:
  env: prod            # arbitrary key/value facts this mixin declares about itself

requires:               # (optional) this mixin is only valid if ALL of these hold
  - env == prod         #   against the tags of every OTHER selected mixin
excludes:               # (optional) this mixin is invalid if ANY of these hold
  - env == dev

env:                    # (optional) static key/value pairs to emit into the .env
  OAUTH_CLIENT_ID: "12345"
  OAUTH_CLIENT_SECRET: "sekrit"

script:                 # (optional) run a script to generate additional pairs
  path: ./scripts/fetch-secrets.sh
  format: json           # "dotenv" (default) | "json" | "yaml"
```

- **`tags`** — facts this mixin contributes to the pool of "all tags from the
  current selection". Any mixin's `requires`/`excludes` can reference any
  other selected mixin's tags, regardless of category.
- **`requires`** / **`excludes`** — boolean tag expressions:
  `key == value`, `key != value`, combinable with `&&`, `||`, and
  parentheses, e.g. `(env == dev || env == staging) && region == eu`.
  Evaluated against the tags of every *other* mixin in the final selection.
  A mixin with no `requires`/`excludes` is compatible with anything.
- **`env`** — static key/value pairs contributed to the final `.env`.
- **`script`** — an executable to run instead of (or in addition to) `env`.
  Its stdout is parsed according to `format` and merged in. JSON/YAML values
  are stringified (numbers, bools); nested objects/arrays are rejected since
  dotenv can't represent them. `path` is resolved relative to the mixin's own
  YAML file unless it's absolute. A key produced by both `env` and `script`,
  or by two different selected mixins, is a hard error — there's no implicit
  precedence.

## Usage

### `configmixer list`

Show what's available:

```sh
configmixer list --mixins-dir ./mixins
```

```
environment:
  dev  tags={env=dev}
  prod  tags={env=prod}
oauth-account:
  main  tags={oauth-account=main}
oauth-app:
  dev  tags={oauth-app=dev}
  prod  tags={oauth-app=prod}
tenant:
  acme  tags={tenant=acme}
  globex  tags={tenant=globex}
```

Pass a category name to filter: `configmixer list --mixins-dir ./mixins tenant`.

### `configmixer generate`

Select one mixin per category, either via flags (one `--select-<category>` flag
per category discovered in your mixins directory) or interactively for
whatever you don't specify:

```sh
configmixer generate \
  --mixins-dir ./mixins \
  --select-environment dev \
  --select-tenant acme \
  --select-oauth-app dev \
  --select-oauth-account main \
  --out .env
```

```
wrote .env
```

Omit any category flag to be prompted for it, restricted to choices that are
actually compatible with what you've already picked:

```sh
$ configmixer generate --mixins-dir ./mixins --select-environment dev
Select tenant:
  1) acme
> 1
Select oauth-app:
  1) dev
> 1
Select oauth-account:
  1) main
> 1
wrote .env
```

(Note `tenant globex` and `oauth-app prod` were already filtered out
because they `require env == prod`, which conflicts with `environment: dev`.)

An incompatible combination fails clearly instead of silently producing a
broken `.env`:

```sh
$ configmixer generate --mixins-dir ./mixins --select-environment dev --select-oauth-app prod ...
error: incompatible selection:
  - oauth-app/prod requires env == prod
```

Flags:

| Flag             | Default      | Description                                  |
|------------------|--------------|-----------------------------------------------|
| `--mixins-dir`   | `.configmixer` | Directory containing mixin categories       |
| `--out`          | `.env`       | Path to write the generated `.env` file       |
| `--select-<category>` | *(prompt)* | One flag per discovered category, e.g. `--select-environment`, `--select-tenant` |

The generated `.env` carries a marker comment (`# configmixer-selection: {...}`)
recording which mixin id was chosen per category. `dotenv` parsing ignores
`#`-prefixed lines, so this is invisible to anything else reading the file;
`configmixer regenerate` uses it to rebuild the file later.

### `configmixer regenerate`

Rebuild a `.env` file from the selection embedded in it by a previous
`generate`/`regenerate`, recomputing env vars (including script output) fresh.
**It never prompts** — if a category has no embedded or flag-provided
selection and more than one compatible mixin, it errors instead of asking:

```sh
$ configmixer regenerate --mixins-dir ./mixins --out .env
wrote .env
```

Pass `--select-<category>` flags to override part or all of the embedded selection,
as long as the result still passes the same compatibility checks `generate`
uses:

```sh
$ configmixer regenerate --mixins-dir ./mixins --out .env --select-environment prod
error: incompatible selection:
  - tenant/acme requires env == dev
$ configmixer regenerate --mixins-dir ./mixins --out .env --select-environment prod --select-tenant globex --select-oauth-app prod
wrote .env
```

The overridden selection becomes the new embedded selection — a later bare
`regenerate` reproduces the overridden result, not the original one.

Pass `--prompt-<category>` to force an interactive prompt for just that
category, bypassing both its embedded selection and any configured default,
while every other category still resolves automatically:

```sh
$ configmixer regenerate --mixins-dir ./mixins --out .env --prompt-oauth-account
Select oauth-account:
  1) main
  2) generated
> 2
wrote .env
```

It's an error to pass `--prompt-<category>` for a category with only one
compatible mixin, since there's nothing to choose between. It's also
mutually exclusive with `--select-<category>` and `--no-default-<category>`
for the same category.

Flags:

| Flag             | Default      | Description                                  |
|------------------|--------------|-----------------------------------------------|
| `--mixins-dir`   | `.configmixer` | Directory containing mixin categories       |
| `--out`          | `.env`       | Path to the `.env` file to read from and overwrite |
| `--select-<category>` | *(embedded)* | One flag per discovered category to override the embedded selection |
| `--no-default-<category>` | `false` | Disable the configured default mixin for that category |
| `--prompt-<category>` | `false` | Force an interactive prompt for that category |

## Worked example

This mirrors the scenario configmixer was built for: an environment, a
tenant (each tenant only valid in one environment), an OAuth app (dev app
only valid in dev, prod app only valid in prod), and an OAuth account
(valid everywhere).

**`mixins/environment/dev.yaml`**
```yaml
tags:
  env: dev
env:
  APP_ENV: "development"
  LOG_LEVEL: "debug"
```

**`mixins/environment/prod.yaml`**
```yaml
tags:
  env: prod
env:
  APP_ENV: "production"
  LOG_LEVEL: "info"
```

**`mixins/tenant/acme.yaml`** — only eligible for `dev`:
```yaml
tags:
  tenant: acme
requires:
  - env == dev
env:
  TENANT_NAME: "Acme Co"
  TENANT_ID: "acme"
```

**`mixins/tenant/globex.yaml`** — only eligible for `prod`:
```yaml
tags:
  tenant: globex
requires:
  - env == prod
env:
  TENANT_NAME: "Globex Corp"
  TENANT_ID: "globex"
```

**`mixins/oauth-app/dev.yaml`** — only valid on dev:
```yaml
tags:
  oauth-app: dev
requires:
  - env == dev
env:
  OAUTH_CLIENT_ID: "1111111111"
  OAUTH_CLIENT_SECRET: "dev-secret"
```

**`mixins/oauth-app/prod.yaml`** — only valid on prod:
```yaml
tags:
  oauth-app: prod
requires:
  - env == prod
env:
  OAUTH_CLIENT_ID: "2222222222"
  OAUTH_CLIENT_SECRET: "prod-secret"
```

**`mixins/oauth-account/main.yaml`** — valid anywhere (no `requires`):
```yaml
tags:
  oauth-account: main
env:
  OAUTH_ACCOUNT_ID: "999999"
  OAUTH_ACCESS_TOKEN: "shared-account-token"
```

**`mixins/oauth-account/generated.yaml`** — same category, but fetches
its values from a script instead of declaring them statically:
```yaml
tags:
  oauth-account: generated
script:
  path: ./scripts/fetch-oauth-account.sh
  format: json
```

**`mixins/scripts/fetch-oauth-account.sh`**:
```sh
#!/bin/sh
echo '{"OAUTH_ACCOUNT_ID": "888888", "OAUTH_ACCESS_TOKEN": "generated-at-runtime-token"}'
```

Generating with a valid combination:

```sh
$ configmixer generate --mixins-dir mixins \
    --select-environment dev --select-tenant acme --select-oauth-app dev --select-oauth-account main
wrote .env
$ cat .env
APP_ENV=development
LOG_LEVEL=debug
OAUTH_ACCESS_TOKEN=shared-account-token
OAUTH_ACCOUNT_ID=999999
OAUTH_CLIENT_ID=1111111111
OAUTH_CLIENT_SECRET=dev-secret
TENANT_ID=acme
TENANT_NAME="Acme Co"
```

Swapping in the script-backed account mixin instead:

```sh
$ configmixer generate --mixins-dir mixins \
    --select-environment dev --select-tenant acme --select-oauth-app dev --select-oauth-account generated
wrote .env
$ cat .env
APP_ENV=development
LOG_LEVEL=debug
OAUTH_ACCESS_TOKEN=generated-at-runtime-token
OAUTH_ACCOUNT_ID=888888
OAUTH_CLIENT_ID=1111111111
OAUTH_CLIENT_SECRET=dev-secret
TENANT_ID=acme
TENANT_NAME="Acme Co"
```

Mixing `dev` environment with `globex` tenant or the `prod` OAuth app
fails, since those declare `requires: env == prod`:

```sh
$ configmixer generate --mixins-dir mixins \
    --select-environment dev --select-tenant globex --select-oauth-app dev --select-oauth-account main
error: incompatible selection:
  - tenant/globex requires env == prod
```

An equivalent, runnable fixture (using the names `facebook-app` /
`facebook-account` for its categories) lives under
[`testdata/.configmixer`](testdata/.configmixer) in this repo, and is used by the
automated tests.

## Development

```sh
go build ./...
go test ./...
```
