# config-requirements-check

A small, generic CLI for catching **silent config-migration gaps**: cases
where a breaking change makes an old config combination invalid, but nothing
crashes or errors — the affected feature just quietly stops working.

Given two YAML files — a **requirements registry** you write once per
breaking change, and a **values file** describing one environment/instance —
it reports:

- which named config paths ("features") are enabled, disabled, or unset
- which requirements are satisfied, not applicable, or misconfigured
- which prerequisites live in other repos/systems and must be checked manually

It knows nothing about any specific product, Helm chart, or schema. It only
understands the generic "path/equals" shape below.

## Install

```
go install github.com/smalisetti/config-requirements-check@<tag>
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
```

Adding a new breaking change is just appending a new entry to
`requirements:` — no schema changes needed. `verify.type` is a named,
extensible field: only `manual` (print-as-checklist, never blocks) is
implemented; a later version could add a real automated checker for a
specific dependency type without touching the schema. Until such a checker
exists, treat every external dependency as **unverified**, not
confirmed-absent.

## Usage

```
config-requirements-check -values <path>
            [-requirements config-requirements.yaml]   # default shown
            [-features]     # print resolved feature-gate states
            [-check]        # validate conditions/requires, report violations
            [-deps]         # print external-dependency checklist
            [-feature <id>] # restrict output to a single requirement id
            [-format text|json]   # default text
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

External dependencies (verify manually):
  [consolidated-health-enabled-toggle] kafka-topic-consolidated-health-events: Kafka topic `consolidated-health-events` must exist and be writable by the consuming service. (owner: platform-kafka / kafka-topics repo)
  [consolidated-health-enabled-toggle] ingress-consolidated-health-grpc: Ingress/gateway rule exposing the ConsolidatedHealthService gRPC endpoint must be present. (owner: platform-ingress / ingress-config repo)
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
go install github.com/smalisetti/config-requirements-check@<tag-or-sha>
config-requirements-check -check -deps -format json \
  -requirements config-requirements.yaml \
  -values envs/<env>/values.yaml
```

Pin `<tag-or-sha>` to the version whose requirements you want enforced (e.g.
the version of the app/chart the target environment is upgrading to). Parse
the JSON output (`features`, `requirements`, `dependencies` keys) to drive a
CI check, a rollout gate, or a dashboard — or just rely on the exit code if
you only need pass/fail.
