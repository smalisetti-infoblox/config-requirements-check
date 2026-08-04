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

## The requirements schema

```yaml
requirements:
  - id: consolidated-health-enabled-toggle
    summary: >
      consolidatedHealth.enabled now explicitly gates consolidated health
      (previously inferred from redis.enabled).

    # ALL entries must hold (AND) for this requirement to apply to a given
    # values file. A single entry today; add more to gate on several flags
    # together.
    conditions:
      - path: redis.enabled
        equals: true

    # ALL entries must hold (AND) once conditions apply. Each unmet entry is
    # reported individually, by path.
    requires:
      - path: consolidatedHealth.enabled
        equals: true

    remediation: >
      Set consolidatedHealth.enabled: true (in addition to redis.enabled:
      true, if Redis is used for other purposes).

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

## Usage

```
config-requirements-check -values <path>
            [-requirements config-requirements.yaml]   # default shown
            [-features]     # print resolved feature-gate states
            [-check]        # validate conditions/requires, report violations
            [-deps]         # print external-dependency checklist
            [-feature <id>] # restrict output to a single requirement id
            [-format text|json]   # default text

config-requirements-check -lint [-requirements config-requirements.yaml] [-format text|json]
```

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

The process exits non-zero **only** when `-check` (or the no-flag default,
which includes `-check`) finds at least one requirement whose `conditions`
hold but whose `requires` doesn't. `-features` and `-deps` are purely
informational and never affect the exit code.

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
