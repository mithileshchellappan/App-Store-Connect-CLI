# Publishing Process Simplification

## Goal

Collapse App Store publishing guidance to one canonical answer while keeping
older automation working during the migration window.

## Canonical command map

- `asc release run` - canonical App Store publish path
- `asc release stage` - canonical pre-submit preparation path
- `asc publish testflight` - canonical TestFlight publish path
- `asc submit preflight|status|cancel` - lower-level submission lifecycle tools
- `asc review ...` - raw review-submission resource management

## What changed

Two App Store-facing command paths were causing avoidable confusion:

- `asc publish appstore`
- `asc submit create`

Both still work as deprecated compatibility paths, but neither should be taught
as the primary answer to "how do I publish to the App Store?"

## Use-when guidance

### Use `asc release run` when

- you want to publish an App Store release
- you have a build ID already selected
- you want one deterministic command that ensures the version exists, applies
  metadata, attaches the build, validates readiness, and submits for review
- you want the command agents and humans should reach for first

### Use `asc release stage` when

- you want the same high-level preparation flow
- you are not ready to submit yet
- you need to stage metadata and build attachment before a later manual or
  automated approval step

### Use `asc publish testflight` when

- you are distributing to TestFlight
- you want an IPA-first high-level flow for beta delivery

### Use `asc submit ...` when

- you want preflight checks without running the full release pipeline
- you want submission status or cancellation commands
- you are debugging review state
- you are maintaining an older direct-submit script and have not migrated off
  `asc submit create` yet

### Use `asc review ...` when

- you need direct access to review-submission resources, items, attachments, or
  history
- you are doing advanced or API-shaped review workflow debugging

## Why `release run` is the best App Store command

- It matches the real user intent: publish an App Store release.
- It includes the surrounding steps users routinely forget when they jump
  straight to submission.
- It aligns the CLI with the documentation and with agent expectations.
- It preserves `submit` for lifecycle/tooling duties instead of overloading it
  as both a publish command and a submission-debug command.

## Migration policy

- Keep `asc publish appstore` runnable with a deprecation warning.
- Keep `asc submit create` runnable with a deprecation warning.
- Hide deprecated App Store entry points from primary discovery where practical.
- Prefer `asc release run` in help text, templates, migration hints, examples,
  and CI docs.
