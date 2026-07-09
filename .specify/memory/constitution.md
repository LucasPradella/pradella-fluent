<!--
Sync Impact Report
==================
Version change: (template) → 1.0.0
Modified principles: n/a (initial ratification — all placeholders filled)
Added sections:
  - Core Principles (I. Spec-Driven Development, II. Test-First Quality,
    III. Simplicity & YAGNI, IV. Code Review & Quality Gates,
    V. Observability & Versioning)
  - Additional Constraints
  - Development Workflow
  - Governance
Removed sections: none
Templates requiring updates:
  - .specify/templates/plan-template.md ✅ compatible (Constitution Check gate
    is populated per-feature from this file; no static changes required)
  - .specify/templates/spec-template.md ✅ compatible
  - .specify/templates/tasks-template.md ✅ compatible
  - .specify/templates/checklist-template.md ✅ compatible
Follow-up TODOs:
  - TODO(PROJECT_NAME): directory is named "my-project"; replace with the real
    product name when it is decided.
  - TODO(TECH_STACK): Additional Constraints defers stack-specific rules until
    the first feature plan fixes the language/framework choices.
-->

# My Project Constitution

<!-- TODO(PROJECT_NAME): replace "My Project" with the real product name when decided -->

## Core Principles

### I. Spec-Driven Development

Every feature MUST begin as a written specification in `specs/<###-feature-name>/spec.md`
before any implementation code is written. Specifications MUST describe user value,
prioritized user stories, and testable acceptance scenarios — not implementation details.
Plans and tasks MUST be derived from the approved spec, and code changes that alter
behavior MUST be reflected back into the spec of the feature they belong to.

**Rationale**: The spec is the single source of truth that keeps intent, implementation,
and verification aligned; skipping it produces untestable, drifting features.

### II. Test-First Quality

Behavior defined in acceptance scenarios MUST be verifiable by automated tests. When a
feature spec requests tests, they MUST be written before or alongside the implementation
and MUST fail before the implementation makes them pass. Bug fixes MUST include a
regression test that reproduces the defect. No task is complete while its tests fail.

**Rationale**: Tests written against the spec — not against the code — are the only
scalable proof that the system does what was agreed.

### III. Simplicity & YAGNI

Start with the simplest structure that satisfies the current spec: a single project
unless the spec demonstrably requires more. Speculative abstractions, unused
configuration options, and layers "for later" MUST NOT be introduced. Any added
complexity (extra projects, patterns, indirection) MUST be justified in the plan's
Complexity Tracking table with the simpler alternative that was rejected and why.

**Rationale**: Complexity is the dominant long-term cost; it must be paid for by a
present, documented need rather than a predicted one.

### IV. Code Review & Quality Gates

All changes MUST pass the project's quality gates before merge: linting/formatting,
the full test suite, and a review against this constitution. Reviewers MUST verify
that the change traces to a spec task and that no principle is silently violated.
Violations found in review MUST either be fixed or explicitly justified in
Complexity Tracking — never waved through.

**Rationale**: Gates enforced at merge time are cheaper than audits after the fact
and keep the constitution operative instead of decorative.

### V. Observability & Versioning

Code MUST fail loudly and diagnosably: errors surfaced with actionable messages and
context, never silently swallowed. Services and libraries MUST use structured,
greppable logging for significant events. Public contracts (APIs, CLIs, schemas)
MUST be versioned; breaking changes require a MAJOR version increment and a
documented migration path.

**Rationale**: Systems are debugged and upgraded far more often than they are
written; observability and explicit versioning are what make both survivable.

## Additional Constraints

- TODO(TECH_STACK): The language, framework, and storage choices are fixed by the
  first approved feature plan; once fixed, deviations require an amendment here.
- Dependencies MUST be added deliberately: prefer the standard library, then
  well-maintained widely-used packages; every new dependency is named in the plan.
- Secrets and credentials MUST NOT be committed to the repository; configuration
  is supplied via environment or ignored local files.

## Development Workflow

- Features follow the spec-kit lifecycle: `/speckit-specify` → `/speckit-plan` →
  `/speckit-tasks` → `/speckit-implement`, with `/speckit-clarify` and
  `/speckit-analyze` used when ambiguity or inconsistency is detected.
- Each feature lives on its own `###-feature-name` branch and directory under
  `specs/`; work lands via reviewed merges, never direct pushes to the default branch.
- The plan's Constitution Check gate MUST pass before Phase 0 research and be
  re-checked after Phase 1 design; unresolved violations block implementation.

## Governance

This constitution supersedes all other development practices in this repository.
Amendments are made by editing this file with: (1) the change and its rationale,
(2) a semantic version bump, and (3) propagation of any impacts to the templates
under `.specify/templates/`.

Versioning policy: MAJOR for removing or redefining a principle in a backward-
incompatible way; MINOR for adding a principle or materially expanding guidance;
PATCH for clarifications and wording fixes.

Compliance review: every plan MUST pass the Constitution Check gate, and every
code review MUST confirm the change complies or records a justified exception in
Complexity Tracking. Use `CLAUDE.md` (or equivalent agent guidance files) for
runtime development guidance that does not belong in this constitution.

**Version**: 1.0.0 | **Ratified**: 2026-07-04 | **Last Amended**: 2026-07-04
