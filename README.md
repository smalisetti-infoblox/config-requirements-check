# config-requirements-check

A small, generic CLI for catching **silent config-migration gaps**: cases
where a breaking change makes an old config combination invalid, but nothing
crashes or errors — the affected feature just quietly stops working.

Given two YAML files — a **requirements registry** you write once per
breaking change, and a **values file** describing one environment/instance —
it reports:

- which named config paths ("features") are enabled, disabled, or unset
- which requirements are satisfied, not applicable, or misconfigured
- which prerequisites live in other repos/systems, and — for each — whether
  any environment is known to have already set it up (something to copy
  when standing up a new one)

It knows nothing about any specific product, Helm chart, or schema. It only
understands the generic "path/equals" shape below.

## Install

```
go install github.com/smalisetti-infoblox/config-requirements-check@<tag>
```

## Quick start

```
config-requirements-check -init > config-requirements.yaml
```

Prints a generic, placeholder-filled starter file (with comments explaining
every field) to stdout — replace the placeholders with your own breaking
change, delete the `external_dependencies`/`references` blocks if you don't
need them, then validate your edits with `-lint` (see below) before using it
for real.

## The requirements schema

```yaml
requirements:
  - id: consolidated-health-requires-full-stack
    summary: >
      Consolidated Health checks require HMS and Redis to both be explicitly
      enabled. If consolidatedHealth is on but HMS or Redis are off, the
      health checks silently fail.

    # ALL entries must hold (AND) for this requirement to apply to a given
    # values file. Each condition specifies a path and exactly one operator
    # (equals, gte, gt, lte, lt, contains, between, not_equals).
    conditions:
      - path: consolidatedHealth.enabled
        equals: true

    # Optional: forbidden states — if the requirement applies AND any of these
    # conditions hold, it's reported as unmet with prefix "FORBIDDEN: <path>".
    unless:
      - path: maintenanceMode
        equals: true

    # Optional: conditionally skip this entire requirement if any condition holds.
    skip_if:
      - path: legacyHealthChecks
        equals: true

    # ALL entries must hold (AND) once conditions apply. Each unmet entry is
    # reported individually, by path. Uses same operators as conditions.
    # Here we express the full transitive dependency chain: consolidated health
    # requires both HMS and Redis (the dependencies of HMS).
    requires:
      - path: hms.enabled
        equals: true
      - path: redis.enabled
        equals: true
      - path: hms.version
        gte: "2.0.0"    # semver-aware (e.g., "2.0.0" >= "1.5.0")

    # Optional: error (default), warn, or info. Descriptive only — does not
    # change exit code behavior (exit 1 only on violated conditions/requires).
    severity: error

    remediation: >
      Enable the full Consolidated Health stack:
      1. Set consolidatedHealth.enabled: true (already set)
      2. Set hms.enabled: true (HMS service performs the checks)
      3. Set redis.enabled: true (Redis backend required by HMS)
      4. Upgrade HMS to version 2.0.0 or later

    # Optional: structured fix hints, gated by path/condition if needed.
    remediation_hints:
      - type: set_field
        path: consolidatedHealth.enabled
        value: true
        description: Enable consolidated health feature
        # Optional: hint only applies if this path/condition hold
        if_path: consolidatedHealth.disabled
        if_condition:
          path: consolidatedHealth.disabled
          equals: true

    # Prerequisites this tool cannot verify itself — owned by other
    # repos/systems. Always surfaced as a checklist when `conditions` hold;
    # never affects the exit code.
    external_dependencies:
      - id: kafka-topic-consolidated-health-events
        description: Kafka topic `consolidated-health-events` must exist.
        owner: platform-kafka / kafka-topics repo
        verify:
          type: manual   # only supported type today — extensible field
        # Optional: environments where this was already done, so whoever's
        # setting up a *new* environment has something to copy. This is NOT
        # verification that the dependency holds for the values file you're
        # currently checking — environments don't share infrastructure just
        # because they share a chart.
        known_implementations:
          - environment: us-dev-2
            url: https://github.com/org/kafka-topics-repo/pull/123

    # Optional: audit trail of where this requirement's condition actually
    # came from in a real environment (e.g. the PR that first turned on
    # redis.enabled=true there). Purely informational — printed alongside
    # every status (-check), never affects it.
    references:
      - label: "us-dev-5: Valkey enabled for consolidated health"
        url: https://github.com/org/deployment-configurations/pull/128418
```

## Condition Operators

Each condition specifies a `path` and exactly one operator from:

- **`equals: <value>`** — exact equality (default fallback; also handles `null`/unset)
- **`gte: <value>`** — greater-than-or-equal (numeric or semver-aware if both are version-like)
- **`gt: <value>`** — greater-than (numeric or semver)
- **`lte: <value>`** — less-than-or-equal (numeric or semver)
- **`lt: <value>`** — less-than (numeric or semver)
- **`contains: <value>`** — array membership (checks if looked-up value is an array containing this element)
- **`between: {min: <a>, max: <b>}`** — range validation (`min <= value <= max`, numeric)
- **`not_equals: <value>`** — negated equality

Numeric/semver comparisons coerce types intelligently: `"1.2.3"` (string),
`1.2.3` (float), and `true` (boolean, as 1.0) can all be compared. Semver
comparison (for `gte`/`gt`/`lte`/`lt` only) activates when **both** operands
parse as semantic versions (format: `major[.minor[.patch]][-prerelease]`).

## Adding Requirements

Adding a new breaking change is just appending a new entry to
`requirements:` — no schema changes needed. `verify.type` is a named,
extensible field: only `manual` (print-as-checklist, never blocks) is
implemented; a later version could add a real automated checker for a
specific dependency type without touching the schema. Until such a checker
exists, treat every external dependency as **unverified for the environment
you're checking** — `known_implementations` documents other environments
that solved it, as a template to copy, not as proof it's done here. Always
add the specific environment name alongside the link: a fix landing in one
env's config, or one env's kafka-topic list, doesn't mean another env has
it — only a genuinely shared default (e.g. a chart's base `values.yaml`
with no per-env override) covers more than one environment, and even then
it's worth double-checking nothing overrides it downstream.

You can also gate requirements on environment name via `-environment <name>`:
each external dependency can list environments in `skip_in_environments`
(a `[]string` field) to hide it from the checklist in those specific environments.

## Usage

```
config-requirements-check -values <path>
            [-requirements config-requirements.yaml]   # default shown
            [-features]     # print resolved feature-gate states
            [-check]        # validate conditions/requires, report violations
            [-deps]         # print external-dependency checklist
            [-feature <id>] # restrict output to a single requirement id
            [-environment <name>]  # skip dependencies marked for this environment
            [-format text|json]   # default text

config-requirements-check -values-dir <path>
            [-requirements config-requirements.yaml]
            [-check]        # report files with violations
            [-format text|json]

config-requirements-check -lint [-requirements config-requirements.yaml] [-format text|json]

config-requirements-check -init
```

`-init` prints a starter `config-requirements.yaml` to stdout and exits — no
`-values` or `-requirements` needed (see Quick start above).

`-values-dir <path>` recursively walks a directory for all `values.yaml` files
(each found file is checked independently against the same requirements) —
mutually exclusive with `-values`. Useful for batch-checking a monorepo across
environments. Note: `-environment` is ignored in batch mode; environment-scoped
dependency filtering only applies to single-file mode.

`-lint` validates the requirements registry's own schema/structure — no
`-values` needed. Two layers of protection:

- **Strict YAML decoding** (always on, not just under `-lint`): an
  unrecognized field anywhere in `config-requirements.yaml` — e.g. a typo
  like `conditons:` instead of `conditions:` — is a hard parse error. Without
  this, a typo'd field silently decodes to an empty/zero value and the
  requirement just quietly does the wrong thing, which is exactly the kind
  of gap this tool exists to catch, except in its own config.
- **Structural checks** (`-lint` only): empty `conditions`/`requires`
  (a requirement that can never apply, or never fail), empty required
  string fields (`id`, `description`, path), duplicate requirement or
  dependency ids, and unrecognized `verify.type` values (only `"manual"` is
  understood today — anything else is almost certainly a typo).

```
$ config-requirements-check -lint -requirements config-requirements.yaml
OK: no issues found
```

With no output-selecting flag (`-features`/`-check`/`-deps`), all three run
and print together — a full report in one pass.

Exit codes:
- **0**: no violations found, or `-features` / `-deps` / `-lint` / `-init` ran successfully
- **1**: `-check` (or the no-flag default, which includes `-check`) found at least one
  requirement whose `conditions` hold but whose `requires` doesn't; or `-lint` found a
  structural problem in the requirements file
- **2**: usage error — bad flags, missing `-values` (or neither `-values` nor `-values-dir`),
  unreadable/unparsable file, or invalid `-format` value

`-features` and `-deps` are purely informational and never affect the exit code (only
`-check`/`-lint` failures cause non-zero exits).

Run `config-requirements-check -h` for the full flag reference with examples. See
`EXAMPLES.md` for detailed flag-by-flag walkthroughs and complex use cases.

## Worked example

`examples/config-requirements.yaml` and `examples/values.yaml` model the
`consolidatedHealth.enabled` migration described above, in a compliant state:

```
$ config-requirements-check -values examples/values.yaml -requirements examples/config-requirements.yaml
Features:
  consolidatedHealth.enabled=true (enabled)
  redis.enabled=true (enabled)

Requirements:
  [satisfied]      consolidated-health-enabled-toggle

External dependencies (not verified against this values file — set up per environment):
  [consolidated-health-enabled-toggle] kafka-topic-consolidated-health-events: Kafka topic `consolidated-health-events` must exist and be writable by the consuming service. (owner: platform-kafka / kafka-topics repo)
                    known implementation in us-dev-2: https://github.com/org/kafka-topics-repo/pull/123
  [consolidated-health-enabled-toggle] ingress-consolidated-health-grpc: Ingress/gateway rule exposing the ConsolidatedHealthService gRPC endpoint must be present. (owner: platform-ingress / ingress-config repo)
                    no known implementations on record — verify manually
$ echo $?
0
```

Now simulate the gap this tool exists to catch — an environment with
`redis.enabled: true` that never picked up the new explicit toggle:

```
$ cat <<EOF > /tmp/gap-values.yaml
redis:
  enabled: true
EOF
$ config-requirements-check -values /tmp/gap-values.yaml -requirements examples/config-requirements.yaml
Features:
  consolidatedHealth.enabled: unset
  redis.enabled=true (enabled)

Requirements:
  [MISCONFIGURED]  consolidated-health-enabled-toggle
                   consolidatedHealth.enabled now explicitly gates consolidated health (previously inferred from redis.enabled). ...
                   unmet: [consolidatedHealth.enabled]
                   fix: Set consolidatedHealth.enabled: true ...
...
$ echo $?
1
```

## Using it from another repo's CI

```
go install github.com/smalisetti-infoblox/config-requirements-check@<tag-or-sha>
config-requirements-check -check -deps -format json \
  -requirements config-requirements.yaml \
  -values envs/<env>/values.yaml
```

Pin `<tag-or-sha>` to the version whose requirements you want enforced (e.g.
the version of the app/chart the target environment is upgrading to). Parse
the JSON output (`features`, `requirements`, `dependencies` keys) to drive a
CI check, a rollout gate, or a dashboard — or just rely on the exit code if
you only need pass/fail.

## Multi-app requirements in deployment-configurations

For deployment repositories managing multiple applications, keep per-app
requirements files organized in a `config-requirements/` directory:

```
deployment-configurations/
├── config-requirements/
│   ├── ddi.dns.dtc.yaml          # Requirements for ddi.dns.dtc
│   ├── ddi.msad.collector.yaml   # Requirements for ddi.msad.collector
│   └── ddi.cloud.proxy.yaml      # Requirements for ddi.cloud.proxy
└── .github/workflows/
    └── validate-config-requirements.yaml  # CI workflow
```

The CI workflow can auto-detect which app a values file belongs to (based on
directory structure) and validate it against the correct requirements file:

```yaml
# values/envs/prod/ddi-dns-dtc/values.yaml
#   → validated against config-requirements/ddi.dns.dtc.yaml

# values/envs/staging/ddi-msad-collector/values.yaml
#   → validated against config-requirements/ddi.msad.collector.yaml
```

**Benefits:**
- Requirements co-located with deployment values (single source of truth)
- Easy to review requirement changes in the same PR as value changes
- Version-pinned (requirements locked at specific commit, no external sync needed)
- Scalable (add new apps just by adding new requirement files)
- Each app team can maintain their requirements independently

**Example CI workflow pattern:**
```bash
# Detect app from values file path
APP=$(echo "$VALUES_FILE" | grep -oE 'ddi[^/]+' | head -1 | tr '-' '.')

# Find and validate against app-specific requirements
REQUIREMENTS_FILE="config-requirements/$APP.yaml"
if [ -f "$REQUIREMENTS_FILE" ]; then
  config-requirements-check -check -deps \
    -requirements "$REQUIREMENTS_FILE" \
    -values "$VALUES_FILE"
fi
```

See [deployment-configurations](https://github.com/Infoblox-CTO/deployment-configurations)
for a working example with ddi.dns.dtc consolidated health requirements.
