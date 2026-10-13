# fetch-secrets: improvement plan

| | |
|---|---|
| Status | Proposed |
| Date | 2026-10-13 |
| Baseline | `v0.3.3`, plus two CI/docs-only commits on `main` |

## 1. Why this document exists

`fetch-secrets` has not had a functional change since `v0.3.2` (March 2024); later releases only bumped dependencies and CI. It is a start-up wrapper for many services, so small failures in it have an outsized effect.

A review of the code, the release pipeline and the behavior of every release line found problems in four areas. The binary was run, unmodified, against a local fake of STS, IAM and Secrets Manager, so the behaviors below were observed rather than inferred (see [Appendix B](#appendix-b-how-the-findings-were-reproduced)).

1. **Failure handling and diagnostics.** Failures are skipped silently, or reported in a form that hides the cause.
2. **Secret value handling.** Values that are not strings turn into garbage, and key names are not validated.
3. **Release and supply-chain hygiene.** Out-of-support toolchain, third-party actions pinned by tag, no CI gate, Dependabot not running.
4. **Tests and tooling.** No CI, the start-up flow is untested, `make lint` is broken.

This document turns the findings into a sequence of small PRs that can ship one at a time. Each section under [PR plan](#4-pr-plan) is the scope of one future PR. Tick the checkboxes as the work lands.

## 2. Principles

- **Small, ordered PRs.** One concern per PR. Behavior-neutral work (CI, tooling, tests, refactor) lands before any behavior change.
- **Pin behavior before changing it.** PR 3 adds characterization tests for today's behavior. Later PRs change those assertions explicitly, so every behavior change shows up as a test diff.
- **Do not surprise consumers.** Consumers pin a release URL and may depend on exit codes and the log format. Anything that changes a default ships behind a flag first and flips in a later release, with release notes.
- **Keep the design.** Replacing the process with `syscall.Exec` (no fork) is right: signals, the PID and the exit status reach the wrapped program unchanged.

### Not planned

- Changing which value wins when a name exists both in the container environment and in a fetched secret. Current behavior is left as it is.
- Alternative delivery mechanisms (files, tmpfs). Secrets stay in the process environment; PR 10 documents the trade-offs.

## 3. Findings index

| ID | Finding | Severity | Planned in |
|---|---|---|---|
| F1 | Since `v0.3.0`, a secret that cannot be loaded (not found, access denied, throttled, 5xx after retries, bad JSON) is logged as a warning and skipped, and the wrapped program starts without it. Earlier releases stop with a non-zero exit code in all of these cases | High | PR 6, PR 7 |
| F2 | Errors are unreadable. `%e` (a float verb) is applied to `error` values and prints a struct dump such as `&{%!e(string=Secrets Manager) …}`; the tag-listing error drops its cause entirely. `go vet` and staticcheck do not flag it | High | PR 4 |
| F3 | Non-string JSON values become garbage (`PORT=%!s(float64=8080)`, `DEBUG=%!s(bool=true)`, `EMPTY=%!s(<nil>)`), and integers above 2^53 lose precision | High | PR 5 |
| F4 | Panics, which exit with code 2 (the README documents 2 as "problem loading AWS config"): a secret that has `SecretBinary` and no `SecretString`, and running with no arguments | Medium | PR 4 |
| F5 | Exit codes do not match the README. A timeout in the STS or IAM call exits 3, not 4; a failed `exec` logs no cause | Medium | PR 4 |
| F6 | Logs go to stdout, mixed into the wrapped program's own output; the full argv is logged; the success path logs key names but not which secret IDs were loaded; the timeout is logged in nanoseconds | Low–Medium | PR 4 |
| F7 | Secret keys become environment variable names verbatim: empty names, names containing `=` or NUL, and names that influence the dynamic loader or language runtimes all pass through | Medium | PR 5 (names that can never work), PR 8 (sensitive names) |
| F8 | The role name is the second `/`-separated segment of the caller ARN, so an IAM user with a path (`user/team/alice`) resolves to the wrong role | Low | PR 8 |
| F9 | A plain-text (non-JSON) secret leaks its first offending character through the JSON parse error (`invalid character 'h' …`) | Low | PR 4 |
| F10 | Release workflow: toolchain pinned to a tarball URL (Go 1.25.4), third-party actions pinned by tag, no test/vet/vulncheck gate, `packages: write` unused, no checksums or provenance | Medium–High | PR 2 |
| F11 | Dependabot is not running: the file is `.github/.dependabot.yml` (GitHub reads `dependabot.yml`) and the `github-actions` entry uses `directory: "/.github"` (it should be `/`). No Dependabot commit exists in the history | Medium | PR 1 |
| F12 | `make lint` is broken (`.golangci.yml` is v1 format, the untagged image is v2); `make generate` installs `mockgen@latest` | Low | PR 1 |
| F13 | No CI. `main` and `getSecrets` have 0% coverage (43% for the package), and the AWS clients are package-level globals | Medium | PR 1, PR 3 |
| F14 | Consumers install with a hand-edited `ADD <url>` line: no checksum, no machine-readable version reference, so fixes reach them slowly and unevenly | Medium | PR 9 |
| F15 | Docs: the install example pins `v0.2.0`; the IAM example grants `sts:GetCallerIdentity`, which needs no permission; KMS permissions are not mentioned; the README says "default 1min" for a timeout that cannot be configured | Low | PR 10 (and each PR) |

## 4. PR plan

Sizes are rough: **S** is about half a day, **M** is one to two days.

### Phase 0: foundations (no behavior change)

#### PR 1: `chore: add CI and fix repo tooling`

**Fixes:** F11, F12, F13 (CI part) · **Depends on:** none · **Size:** S

**Goal.** Every later PR gets automatic checks, and dependency and Actions updates start arriving.

**Tasks**

- [ ] Rename `.github/.dependabot.yml` to `.github/dependabot.yml`. Set `directory: "/"` for the `github-actions` ecosystem, group the AWS SDK modules into one update group, and use a weekly schedule to keep the noise down.
- [ ] Add `.github/workflows/ci.yaml` (on `pull_request` and on push to `main`) running `go build ./...`, `go vet ./...`, `go test -race ./...` and `govulncheck ./...`.
- [ ] Migrate `.golangci.yml` to the golangci-lint v2 format (see its migration guide), run it in CI, and pin the version in both CI and `make lint` (no `latest`).
- [ ] In `make generate`, pin `mockgen` to the version in `go.mod`, and add a CI step that fails when `go generate ./...` changes files.
- [ ] Keep the `go` directive on a supported release and have CI use the newest patch of it.

**Acceptance criteria**

- CI is green on the PR itself.
- Insights → Dependency graph → Dependabot lists both ecosystems without errors, and the first update PRs arrive.
- `make lint` and `make test` pass on a clean checkout.

**Risk and rollout.** The first lint run can surface a backlog. Fix the trivial findings in the same PR and scope the rest with `--new-from-rev` rather than disabling linters.

#### PR 2: `ci: harden the release workflow`

**Fixes:** F10 · **Depends on:** PR 1 · **Size:** S–M

**Goal.** A release is built by a supported toolchain, only after the checks pass, from pinned inputs, and consumers can verify what they download.

**Why.** `v0.3.3` was built with Go 1.25.4. Go 1.25 is no longer supported upstream. Rebuilding the same source with the newest 1.25 patch removes 20 of the 29 standard-library advisories that `govulncheck -mode=binary` reports for the shipped binary, and the remaining 9 are fixed only in Go 1.26.9 and later. Binary mode lists linked symbols and does not prove reachability, and the tool only talks to AWS, so practical exposure is low, but the fix is cheap. Older release assets were built with Go 1.20.4 (`v0.1.4`) and Go 1.21.8 (`v0.3.2`).

**Tasks**

- [ ] Replace the `goversion` tarball URL with `actions/setup-go` (`go-version-file: go.mod`, `check-latest: true`). Move the `go` directive to a supported release. Build with `CGO_ENABLED=0 -trimpath -ldflags "-s -w"`.
- [ ] Pin every third-party action to a full commit SHA (`go-release-action`, `checkout`, `setup-go`) and let Dependabot bump them.
- [ ] Make the CI workflow a gate for the release job (`workflow_call` or `needs`).
- [ ] Least privilege: `contents: write` only on the job that uploads assets, and drop `packages: write`.
- [ ] Publish `SHA256SUMS` with every release and add a build-provenance attestation (`actions/attest-build-provenance`, which needs `id-token: write` and `attestations: write`).
- [ ] Evaluate replacing the third-party release action with `go build` plus `gh release upload`.

**Acceptance criteria**

- A pre-release produces the four binaries (linux and darwin, amd64 and arm64), `SHA256SUMS` and an attestation.
- The workflow fails when a test fails.
- `govulncheck -mode=binary` on an asset lists no advisory that the toolchain used has already fixed.

**Risk and rollout.** The release path can only be exercised by cutting a release. Use a pre-release tag first and delete it afterward.

#### PR 3: `refactor: make main testable`

**Fixes:** F13 · **Depends on:** PR 1 · **Size:** M · **No behavior change**

**Goal.** The start-up flow can be tested end to end, and today's behavior is pinned before it is changed.

**Tasks**

- [ ] Extract `run(ctx, args, environ, deps) int`. Inject the AWS clients (the interfaces already exist) and the exec function, and remove the package-level client globals. `main` only wires things together and exits.
- [ ] Add an in-process fake of STS, IAM and Secrets Manager (`httptest` plus `AWS_ENDPOINT_URL`) so tests drive the real SDK stack, retries included.
- [ ] Add characterization tests that pin today's behavior, each annotated with the PR that will change it (`// current behavior; changes in PR 4`): partial failure, no tags, STS and IAM errors, a timeout in each phase, non-JSON and binary secrets, non-string values, exec failure.
- [ ] Add one test per documented exit code.

**Acceptance criteria**

- `go test -race ./...` is green, and `run` and `getSecrets` are covered.
- No production behavior differs: the diff changes the structure of `main.go` only.

**Risk and rollout.** Low. Keep the PR mechanical and do not fix anything on the way.

### Phase 1: correctness and diagnostics

#### PR 4: `fix: readable errors, correct exit codes and logging`

**Fixes:** F2, F4, F5, F6, F9 · **Depends on:** PR 3 · **Size:** M

**Goal.** When something fails, the log says what and why, the exit code says which phase, and the program never panics.

**Tasks**

- [ ] Replace `%e` with `%w`. Wrap both the sentinel and the cause (`fmt.Errorf("%w: %w", errGetTags, err)`), and include the cause in the tag-listing error. Tests assert on the cause (access denied, not found, deadline) with `errors.As`.
- [ ] Remove the panics. Use `aws.ToString` for `Arn`, `Key` and `Value`, report a secret without `SecretString` (binary) with a clear error, and print a usage message when no command is given.
- [ ] Add named exit-code constants and a single exit path. Map deadline errors with `errors.Is(err, context.DeadlineExceeded)` so a timeout in any phase exits 4. Log the `exec` error. Add a top-level `recover` that logs and exits with a distinct, documented code, so a panic can never look like "AWS config".
- [ ] Logging: write to stderr; one INFO line per secret ID loaded, with key names at DEBUG; add `FS_LOG_LEVEL`; stop logging the full argv (log the resolved executable instead); log the timeout as a duration string.
- [ ] Do not echo JSON parser errors for secret payloads, since they can contain a character of the secret. Log the secret ID and a generic message.
- [ ] Update the README exit-code table, the log-destination note and the troubleshooting section.

**Acceptance criteria**

- Every exit-code path has a test.
- Piping the wrapped program's stdout yields only its own output.
- A binary secret and a missing command return a documented exit code instead of a stack trace.

**Risk and rollout.** This is the first PR with consumer-visible changes: log destination, message formats, and exit code 4 for STS and IAM timeouts. Release it as a minor version with release notes. Consumers should check their log-parsing rules and any alert keyed on exit codes, and confirm that the log pipeline maps the JSON `level` field instead of treating everything on stderr as an error.

#### PR 5: `fix: decode secret values correctly`

**Fixes:** F3, F7 (names that can never work) · **Depends on:** PR 3 · **Size:** S–M

**Goal.** A value reaches the program exactly as written in the secret, or start-up fails with a message that names the key.

**Tasks**

- [ ] Decode with `json.Decoder.UseNumber()`. Strings pass through unchanged, numbers and booleans become their JSON text, objects and arrays are an error that names the key (the README promises flat key/value), and `null` is handled as decided in [open question 2](#6-open-questions).
- [ ] Reject keys that can never be environment variable names (empty, or containing `=` or NUL), and reject values containing NUL with a clear error. Today that surfaces as an `exec` failure with no cause. Names such as `SECRET-1` stay allowed.
- [ ] Sort keys, and the secrets derived from tags, so the environment order is the same on every run.
- [ ] Tests: a type matrix, integers above 2^53, Unicode, multi-line values, NUL handling.

**Acceptance criteria**

- `{"PORT": 8080}` yields `PORT=8080`.
- `9007199254740993` is preserved.
- A nested value fails with an error that names the key.

**Risk and rollout.** A secret holding a number or boolean currently produces a garbage value that cannot work, so fixing it is safe. A secret with `null` or nested values will now warn or fail; call that out in the release notes.

### Phase 2: failure semantics

#### PR 6: `feat: opt-in strict mode and a configurable timeout`

**Fixes:** F1 (first half) · **Depends on:** PR 4, PR 5 · **Size:** S–M

**Goal.** A failed secret can stop start-up, and a skipped one is always visible.

**Tasks**

- [ ] Add `FS_STRICT` (name to be agreed). When true, failing to fetch or parse any discovered secret exits with the "secrets" exit code after logging which secret and why. The default stays lenient in this PR.
- [ ] Always log one summary line (`loaded N of M secrets, failed: [ids]`), at ERROR when anything failed, so a skipped secret is visible in lenient mode too.
- [ ] Warn when no `secrets_` tag is found (fatal under strict mode).
- [ ] Add `FS_TIMEOUT` (default 60s) and document it.
- [ ] Tests: strict and lenient for each failure kind (not found, access denied, 5xx after the SDK's retries, bad JSON, binary).

**Acceptance criteria**

- With `FS_STRICT=true`, no failure kind can result in a successful start.
- Without it, behavior matches today's plus the summary line.

**Risk and rollout.** Low (opt-in).

#### PR 7: `feat!: strict mode by default`

**Fixes:** F1 (second half) · **Depends on:** PR 6, plus one release cycle of adoption · **Size:** S (code), plus release notes

**Goal.** "A secret failed to load" is fatal unless the operator says otherwise.

**Tasks**

- [ ] Flip the default. Keep `FS_STRICT=false` as the escape hatch and document it.
- [ ] Release notes with a migration checklist: deploy the previous release with `FS_STRICT=true` in a non-production environment first, and look for `failed:` in the summary line.

**Acceptance criteria**

- Every failure kind exits non-zero by default, and the escape hatch restores the old behavior.

**Risk and rollout.** This restores the original contract, since releases before `v0.3.0` already stop when a secret cannot be fetched. It can still surface services that start today with a missing secret, which is why PR 6 ships first.

### Phase 3: hardening

#### PR 8: `feat: warn on sensitive variable names and parse the caller ARN properly`

**Fixes:** F7 (sensitive names), F8 · **Depends on:** PR 5 · **Size:** S

**Goal.** A secret cannot silently change how the wrapped program is loaded or where it sends its AWS traffic, and the role is derived reliably.

**Tasks**

- [ ] Add a built-in list of sensitive names and prefixes: dynamic-loader variables (`LD_*`, `DYLD_*`), shell start-up variables, runtime option variables (`NODE_OPTIONS`, `JAVA_TOOL_OPTIONS`, `PYTHONSTARTUP`), and SDK endpoint overrides (`AWS_ENDPOINT_URL*`). Phase A: log a WARN that names the key, never the value, with no behavior change.
- [ ] Phase B, in a separate release: opt-in enforcement (`FS_DENY_NAMES` and `FS_ALLOW_NAMES`, names to be agreed). Decide on any default only after seeing the warnings in the field.
- [ ] Parse the caller ARN with `arn.Parse` and require an `assumed-role/<name>/<session>` resource. Return a clear error for users, federated users and the root account, with table tests that use real ARN shapes.
- [ ] Document that tag write access decides which readable secret is loaded, and recommend limiting `iam:TagRole` on service roles.

**Acceptance criteria**

- Warnings fire for the listed names and only for them.
- The ARN table tests pass, and `user/team/alice` no longer resolves to a role named `team`.

**Risk and rollout.** Some consumers may legitimately provide names on the list (for example SDK settings), which is why this is warn-first.

### Phase 4: consumption

#### PR 9: `feat: make fetch-secrets easier to keep current`

**Fixes:** F14 · **Depends on:** PR 2 · **Size:** M

**Goal.** Consumers can verify what they install and receive updates automatically.

**Tasks**

- [ ] Publish a minimal container image that holds only the static binary (multi-arch, attested), so consumers can `COPY --from=<image>@sha256:…` and have Dependabot or Renovate propose bumps. Decide the registry and visibility (see [open question 3](#6-open-questions)).
- [ ] Document the checksum-pinned form `ADD --checksum=sha256:… --chmod=+x <url> /fetch-secrets` for consumers that keep downloading the binary, with a Renovate regex-manager example.
- [ ] Add `--version`, and log the version at start-up.

**Acceptance criteria**

- A consumer repository can be bumped by a bot PR.
- A wrong checksum fails the image build with `digest mismatch`.

**Risk and rollout.** Mostly registry and permission decisions rather than code.

#### PR 10: `docs: refresh the README and add a migration guide`

**Fixes:** F15 · **Depends on:** PR 4–8 (it describes their final behavior) · **Size:** S

**Tasks**

- [ ] README: current version and a checksum-pinned install in the examples, the updated exit-code table, logging and strict mode, `FS_TIMEOUT`, and troubleshooting organized by symptom.
- [ ] IAM example: drop `sts:GetCallerIdentity` (AWS documents that no permission is required) and note `kms:Decrypt` for secrets encrypted with a customer-managed key.
- [ ] State the delivery trade-off: secrets live in the process environment, so they are readable through `/proc/<pid>/environ` by the same user and root, and inherited by child processes.
- [ ] Add `docs/migrating.md` with the behavior differences between release lines ([Appendix A](#appendix-a-behavior-by-release-line)) and an upgrade checklist.
- [ ] Add `CHANGELOG.md` ([Keep a Changelog](https://keepachangelog.com)) and point release notes at it.

**Acceptance criteria**

- Examples are copy-pasteable and tested, and links resolve.

## 5. Sequencing and releases

| PR | Depends on |
|---|---|
| 1 | none |
| 2 | 1 |
| 3 | 1 |
| 4 | 3 |
| 5 | 3 |
| 6 | 4, 5 |
| 7 | 6, plus one release cycle |
| 8 | 5 |
| 9 | 2 |
| 10 | 4–8 |

Suggested releases (version numbers are a proposal):

| Release | Contains | Consumer impact |
|---|---|---|
| `v0.3.4` | PR 1–3 | None. Same behavior, rebuilt with a supported Go toolchain, with checksums |
| `v0.4.0` | PR 4, 5, 6, 8 (phase A) | stderr logging, new exit codes, correct value decoding, opt-in strict mode, warnings |
| `v0.5.0` | PR 7 | Strict mode by default |
| any | PR 9, 10 | None. Distribution and docs |

**Definition of done for every PR:** tests added or updated, README and CHANGELOG touched where behavior changes, CI green (build, vet, race tests, govulncheck, lint), and no new package-level state.

## 6. Open questions

1. **Strict-mode flag name and timeline.** `FS_STRICT`? How many releases between PR 6 and PR 7?
2. **`null` values.** Skip with a warning, or export an empty string?
3. **Registry for the image.** Where should it be published, and should it be public?
4. **Reliance on current behavior.** Does anything parse the stdout log format or alert on exit codes? Check before PR 4.
5. **Support policy for old release lines.** Releases up to `v0.1.x` use AWS SDK for Go v1, for which AWS ended support on 2025-07-31. Publish a deprecation notice and recommend upgrading, or backport fixes?
6. **Tracking.** Tick the checklists in this file as PRs merge, or move the items into issues?

## Appendix A: behavior by release line

Observed by running each release line's logic against the fake described in Appendix B. `v0.2.0` was read from source.

| Release line | SDK | Secret cannot be fetched | Tag listing fails | Non-JSON secret | Binary secret | Logs |
|---|---|---|---|---|---|---|
| `v0.1.2`–`v0.1.4` | AWS SDK for Go v1 (end of support 2025-07-31) | exit 1, with the AWS error | **ignored: continues with no secrets** | panic, exit 2 | panic, exit 2 | stderr; secret IDs printed to stdout |
| `v0.2.0` (source) | AWS SDK for Go v2 | exit 1, with the AWS error | exit 1 | panic, exit 2 | panic, exit 2 | stderr; logs secret IDs and the full argv |
| `v0.3.0`, `v0.3.1` | AWS SDK for Go v2 | n/a | n/a | n/a | n/a | **Do not use.** The tag prefix is `secret_`, so no `secrets_` tag matches (fixed in `v0.3.2`) |
| `v0.3.2`, `v0.3.3` | AWS SDK for Go v2 | **warning, skipped** | exit 3 | warning, skipped | panic, exit 2 | JSON on stdout |

## Appendix B: how the findings were reproduced

The compiled binary was run against a small local fake of the three AWS endpoints (STS `GetCallerIdentity`, IAM `ListRoleTags`, Secrets Manager `GetSecretValue`). `AWS_ENDPOINT_URL` pointed at the fake, the credentials were dummy values, and the environment was scrubbed. The scenario was selected by the fake access key ID. The scenarios were:

| Scenario | What it shows |
|---|---|
| Happy path | Baseline |
| One of two secrets not found | F1, per release line |
| Secret access denied | F2 (error text) |
| Transient 500 on one secret | F1: the SDK makes three attempts in about 2 seconds, then the secret is skipped |
| Tag listing denied, STS denied | F2, F5 |
| Role with no `secrets_` tags | Silent success |
| Plain-text secret | F9 |
| JSON with numbers, booleans, null, nested values and large integers | F3 |
| Binary secret, no arguments | F4 |
| NUL byte in a value | F5 (exec failure without a cause) |
| Keys that are empty, contain `=`, or are loader or runtime variables | F7 |
| Caller ARN of an IAM user with a path | F8 |
| 75-second delay in each phase against the 60-second deadline | F5 |

PR 3 should turn this fake into a Go test helper so the same scenarios run in CI.
