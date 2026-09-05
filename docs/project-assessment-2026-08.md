---
Status: HISTORICAL
DateCreated: 2026-08-26
DateLastReviewed: 2026-08-29
Supersedes: nothing — this is the record that opened the project
SupersededBy: >
  §5's open questions are settled in "[`docs/decision-log.md`](decision-log.md)";
  §6's roadmap is superseded by "[`docs/design.md`](design.md)" §9 and
  "[`docs/roadmap.md`](roadmap.md)"
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/design.md`](design.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
---

# hasp — State of the Project

**Date:** 2026-08-26
**Branch:** `master` @ `d67288f`
**Assessed by:** code read + live execution of every implemented command

---

> ## ⚠ This document is historical, and describes a codebase that is no longer live
>
> **It assesses the Python implementation of hasp**, which was not carried forward. hasp was
> re-implemented in Go, in this repository, per
> [`T27`](tech-decision-log.md#t27) — which cites this document as its own evidence and records
> why the repair roadmap in §6 below was not taken. In short: `D12` removed persisted state
> entirely, which dissolved Phase 1's plan to decide the store and Phase 2's goal to finish
> `sync key` — the two phases §6 identifies as the bulk of the work and the MVP.
>
> **Nothing below is a live plan.** §6's Phase 0–5 roadmap describes repairs to files that do not
> exist in this repository. It is preserved because the *findings* it records are the evidence
> the whole design rests on — the split-brain TOML/SQLite state, the `# TODO: Left off here`
> marker, the zero-test `ContextBorg` singleton, the `libmagic` and `ssh-keygen` dependencies.
> Every one of those is cited by a decision that exists because of it.
>
> **§5's five open design questions are all settled**, and this is where they landed:
>
> | §5 question | Settled by |
> | --- | --- |
> | 1. TOML or SQLite? | [`D12`](decision-log.md#d12) — **neither.** hasp persists nothing and derives everything, so there is no store to choose |
> | 2. Verb×resource matrix or `factory/`? | The matrix survives — [`D10`](decision-log.md#d10) classifies the nouns, [`T3`](tech-decision-log.md#t3) registers commands from a table. `factory/` is not carried forward |
> | 3. Is the Borg pattern earning its keep? | **No.** [`tdd.md` §12](tdd.md) makes the key directory an explicitly injected value, never a global — named there as the discipline that structurally rules out this document's own "why it had zero tests" finding |
> | 4. What is an "alias", exactly? | [`D5`](decision-log.md#d5) — an additional *name* for a key, materialized as a link, sharing one flat namespace with names |
> | 5. Groups vs. the README's profiles | [`D1`](decision-log.md#d1) — one concept, hierarchically named. "Key group" and "profile group" are retired; [`D9`](decision-log.md#d9) reinstates "host group" with a container meaning |

---

## 1. Executive summary

`hasp` is a Click-based CLI for managing SSH identity (keys now; hosts and config later).
Real work stopped in **September 2020**; 2021–2023 commits are Snyk dependency bumps only.
The three commits from **2026-08-26** are your dependency/packaging refresh — they did not
touch program logic.

The project is **not abandoned mid-refactor by accident — it is abandoned mid-refactor
deliberately**, and the refactor is the thing that has to be finished before anything else
matters. In Sept 2020 you began migrating state storage from a **TOML file** to a
**SQLite/SQLAlchemy database** (`6bbea38 WIP: Switching state mgmt over to sqlite db`) and
stopped partway. The result is a codebase with two competing state stores wired into
different commands.

Current runtime reality:

| Command | Status |
| --- | --- |
| `hasp --help`, `--version` | Works |
| `hasp list key` | Works — reads **TOML** state |
| `hasp show key -n NAME` | Works — reads **TOML** state |
| `hasp new key -n NAME` | Works — writes **TOML** state (with a type bug, §3.3) |
| `hasp sync key` | Runs, but **imports nothing** — the key-scanning body is commented out; only group rows are written to **SQLite** |
| `hasp edit key`, `find key` | `CLINotImplementedError` stub |
| all `host` / `config` subcommands | `CLINotImplementedError` stub |
| **Any second invocation of any command** | **Crashes** — see §3.1 |

Roughly **30–40% of the intended MVP exists**, and it is currently unusable as a daily tool.
The good news: the blockers are small and well-localised. A focused day gets you back to a
working single-store tool; the interesting design work survives intact.

---

## 2. What is actually here

### 2.1 Layout

```
src/hasp/
├── cli/
│   ├── __init__.py        logging config (YAML + env-var shims), rich console, option_ partial,
│   │                      CLINotImplementedError
│   ├── entry.py           root Click group, engine/session creation, ContextBorg init  ← has debug cruft
│   ├── resource/          per-noun implementations: key.py (real), host.py (stubs), config.py (stubs)
│   └── verb/              per-verb composition: new, edit, find, list_, show, sync
├── factory/               Abstract-Factory scaffolding — DEAD CODE, imported by nothing
├── resources/             config.toml, state.toml, log_conf.yaml (packaged data)
├── hasp.py                domain logic: sync, import, present, update
├── io.py                  Borg-based Context, TOML read/write, key-group parsing
└── orm.py                 SQLAlchemy 2.0 declarative models
```

The **verb × resource matrix** is the good architectural idea in this codebase and it is
fully wired. `cli/verb/<verb>.py` imports the same-named function from each of the three
`cli/resource/*.py` modules and registers them as subcommands, so `hasp <verb> <noun>` is
uniform and adding a noun is mechanical. Keep this.

### 2.2 The data model (`orm.py`)

Already modernised to SQLAlchemy 2.0 `DeclarativeBase` / `Mapped[...]` style:

- `SshKey` — PK `name`; `type`, `comment`, `bits`, `md5`, `format`
- `SshKeyGroup` — PK `name`; many-to-many with `SshKey` via `keys__key_groups`
- `SshKeyAlias` — PK `name`; FK to `SshKey` (models the symlink-alias concept)
- `SshHost`, `SshHostGroup`, `SshConfig`, `SshConfigGroup` — **commented out**, sketched only

Verified: `Base.metadata.create_all()` produces valid DDL for all four tables.

### 2.3 What the last WIP commit (`36523f8`) was doing

That commit is a snapshot of un-committed 2020-era work, captured this month. It contained
three separate threads:

1. **Py3.9+ typing modernisation** — `List`/`Optional`/`Union` → `list`/`| None`. Done.
2. **The `factory/` package** — a full Abstract-Factory skeleton (`AbstractManager` +
   `Create/Read/Update/Delete` metas, with concrete `KeyManager`/`HostManager`/`ConfigManager`).
   Every method body is `...`. **Nothing imports it.** This was an alternative dispatch design
   you were exploring alongside the verb×resource matrix — the two overlap, and you never
   picked one.
3. **Live debugging scaffolding in `entry.py`** — hard-coded DB path, `echo=True`, and a
   hard-coded `SshKey(name="foo", ...)` insert. This is what breaks the CLI today.

`hasp.py:224` still carries the literal marker `# TODO: Left off here`, immediately before the
key-group association logic in `update_key()`. That is the exact point where work stopped.

### 2.4 Tests

`tests/features/` holds three Gherkin `.feature` files. `keys.feature` is genuinely valuable —
it enumerates eleven concrete scenarios you wanted (import, generate, search by hash, dedupe
duplicates into symlinks, regenerate missing pub-keys, single-pane report, assign to host,
assign to group, rename, replace). `hosts.feature` and `config.feature` are empty templates.

**There are zero step definitions and zero Python test files.** `pytest` collects 0 items and
exits 0 — a green result that means nothing. `pytest-bdd`, `pytest-cov`, `coverage` and
`hypothesis` are all installed and unused.

---

## 3. Confirmed defects

Each of these was reproduced by running the CLI, not inferred from reading.

### 3.1 BLOCKER — debug scaffolding makes every command single-use

`src/hasp/cli/entry.py:45-66`, in the root callback that runs before *every* subcommand:

```python
db_path = "/tmp/quicktest.db"        # hard-coded, not XDG, wiped on reboot
engine = create_engine(..., echo=True)   # dumps all SQL to the console
...
foo = SshKey(name="foo", type="", comment="", bits="", md5="", format="")
session.add(foo); session.commit()
print(f"Created foo: {foo} {foo!r}")
```

Reproduced: first run of any command succeeds; the second run dies with
`IntegrityError: UNIQUE constraint failed: ssh_keys.name`, because `name` is the primary key
and `foo` is re-inserted unconditionally. `echo=True` also buries all real output under
hundreds of lines of SQL. This is three lines of deletion and one config decision.

### 3.2 BLOCKER — split-brain state: TOML and SQLite are both live

- `list key` / `show key` → `present_key*()` → **TOML** (`c.state.toml_doc["ssh_key"]`)
- `new key` → `write_ssh_key()` → **TOML**
- `sync key` → `init_state_groups()` / `update_key()` → **SQLite** (`c.session`)

So `sync` can never see a key that `new` created, and `list` can never see anything `sync`
imported. The migration from `6bbea38` was never completed. `hasp.py` still carries the
pre-migration `update_key()` preserved as a ~45-line commented block (lines 232-277) for
reference.

### 3.3 BUG — `new key` records the wrong key type

`cli/resource/key.py:36,75`. The command maps the user-facing choice to an ssh-keygen
algorithm (`ec` → `ed25519`) for the `ssh-keygen` invocation, but then persists the *choice*:

```python
type_: str = key_type.lower()          # "ec"
type_map = {"rsa": "rsa", "ec": "ed25519"}
...
new_key_md = {"type": type_}           # writes "ec", not "ed25519"
```

Reproduced: `hasp new key -n newtest -y` wrote `type = "ec"` to state, whereas
`read_real_key()` (used by `sync`) reports the same key as `"ed25519"`. Records created by
`new` and by `sync` are therefore not comparable. `bits` has the same class of problem —
`new` writes the string `"4096"`, `read_real_key` returns the int `4096`, and the ORM declares
`bits: Mapped[int]`.

### 3.4 BUG — `sync key` never imports keys

`hasp.py:59-73`. The entire init/prune block that walks `~/.ssh` and calls
`init_state_entries()` / `prune_state_entries()` is commented out. Only
`init_state_groups()` and `prune_state_groups()` run — and `prune_state_groups()` is `pass`.

Reproduced against a scratch `~/.ssh` and a scratch config: `sync key` logged
`Found host member in config file that has no counterpart: "personal/id_ec_rocket"` for every
single group member, because the keys table is never populated. The `--all`, `--name`,
`--prune`, `--init`, `--both` options are all parsed and all ignored.

### 3.5 BUG — the packaged `config.toml` is from a different project

`src/hasp/resources/config.toml` contains `[defaults] project_name / python_version /
workspace` and `[hasp.project|venv|links]` — leftovers from a project-scaffolding tool, not
from `hasp`. It has **no `[key_groups]` section**, which `ContextBorg.consume_groups()`
requires unconditionally.

Reproduced: pointing `hasp` at its own bundled template dies with
`NonExistentKey: 'Key "key_groups" does not exist.'`

`src/hasp/resources/state.toml` is likewise stale — it uses a `[keyx.*]` table name where all
code expects `[ssh_key.*]`.

There is also **no bootstrap path at all**: with no `~/.config/hasp/config.toml` and no
`~/.local/share/hasp/state.toml` present (which is the case on this machine right now), every
command dies with a raw `FileNotFoundError` traceback. There is no `hasp init`.

### 3.6 Untracked working files at repo root

- `bk` — a backup of your real 2020 state file: 13 SSH keys with md5 fingerprints, comments
  (`jesse@thor`, `rocket rsa`, `vcs-admin`), and two alias entries. This is the **only
  surviving example of a fully-populated real-world state file** and is what I used to
  exercise `list`/`show`. Worth preserving as a test fixture — after review, since it is a
  fingerprint inventory of your actual keys.
- `qtest.py` — a 9-line scratch script that prints `CreateTable(...)` DDL for the ORM models.
  Useful as a throwaway; not worth committing as-is.

---

## 4. Toolchain and hygiene

**Good:** `uv` migration is clean and the lockfile resolves. `uv build` produces a valid wheel;
package data (`config.toml`, `log_conf.yaml`, `state.toml`) and the `__init__.py`-less
`cli/resource/` directory are both included correctly. `libmagic` is present and
`python-magic` works. The entry point `hasp = hasp.cli.entry:main` is registered.

**Needs attention:**

| Item | Finding |
| --- | --- |
| `ruff check` | **59 errors** (39 auto-fixable): 19 unsorted-imports, 16 unused-import, 5 deprecated `typing.Text`, 5 deprecated-import, 3 unused-variable, plus `RUF013` implicit-optional and `PLW1510` `subprocess.run` without `check=` |
| `ruff format` | **10 of 21 files** would be reformatted |
| `ty check` | **12 diagnostics** — mostly `x: T = None` dataclass fields in `io.py` that should be `T \| None = None`, plus 4 genuine unresolved-attribute errors in `io.py:113-116` (`get_key_list` sets `c.toml_file` / `c.toml_doc`, attributes that do not exist on `Context`; that function is dead code from the pre-`State`/`Config` era) |
| `pyproject.toml` | **No `[tool.ruff]` section at all** — ruff runs at its default 88-col while `.editorconfig` and `setup.cfg` both say 120. No `[tool.pytest.ini_options]` either |
| `setup.cfg` | Obsolete: `[bdist_wheel]`, `[aliases] test = pytest`, `[flake8]`, two `[pylama:*]` sections. Its `[tool:pytest]` block is currently the pytest rootdir config and contains `collect_ignore`, which pytest rejects as an unknown option |
| `requirements.txt` / `requirements-dev.txt` | 16 KB and 49 KB, tracked, pinned to 2023, kept only to feed Snyk. Superseded by `uv.lock` |
| `.python-version` | Absent (one exists in `.bkup` pinning 3.8.5). `requires-python = ">=3.14"` with no pin means no reproducible interpreter |
| `.vscode/settings.json` | Points at a `black` provider, `pylint`, and a `~/.cache/pypoetry/virtualenvs/hasp-Tp-ZPRU2-py3.11` path that no longer exists |
| CI | None. No `.github/`, no `.pre-commit-config.yaml` — though `pre-commit` **is** a declared dev dependency |
| Stale branches | Five `snyk-fix-*` branches on origin, all merged |
| README | References `./vendor/advanced-ssh-config`, which does not exist in the repo |

---

## 5. Design decisions that are still open

These are the questions the 2020 you left unanswered. They should be settled before writing
new code, because each one changes what "finished" means.

1. **TOML or SQLite?** You started moving to SQLite and stopped. SQLite buys relational
   groups/aliases and real queries for `find key --md5`. TOML buys a human-editable,
   diffable, git-committable state file — which fits a personal SSH-config tool, and fits
   your stated goal that "SSH config file may be edited by hand safely." A defensible third
   answer: **TOML is the state, SQLite is a derived cache** rebuilt by `sync`. Pick one and
   delete the other path.

2. **Verb×resource matrix or `factory/`?** Both are dispatch mechanisms for the same
   noun×verb grid. The matrix works today; `factory/` is 300 lines of unreferenced
   abstraction. Deleting `factory/` is the low-risk call — it is in git history if you want it.

3. **Is the Borg pattern earning its keep?** `ContextBorg` is a process-wide mutable
   singleton reached by constructing `ContextBorg()` inside every function. It makes the
   code untestable without global setup, which is part of why there are no tests. Click's
   own `ctx.obj` already does this job with explicit passing.

4. **What is an "alias", exactly?** `SshKeyAlias.name` is a primary key, so alias names are
   globally unique across all keys. Fine if aliases model `~/.ssh` symlinks (they do today —
   `init_state_entries` creates one per symlink). Worth confirming before building on it.

5. **Groups vs. the README's "profiles and profile groups."** The README promises profiles
   ("personal" / "work", multiple profiles per group). The code implements `key_groups` with
   one level of nesting flattened to `"work.foobarco"` strings. These may or may not be the
   same concept. Reconcile the vocabulary.

---

## 6. Roadmap

Ordered so that each phase leaves the tool in a better working state than it found it.

### Phase 0 — Make it run twice (½ day)

Non-negotiable prerequisite for everything else.

- [ ] Strip the debug scaffolding from `entry.py`: delete the `foo` insert, drop `echo=True`
      (or bind it to `--verbose`), move `db_path` off `/tmp/quicktest.db` to an XDG path
      derived from `--state-file`.
- [ ] Wire `--verbose` to something. It is currently accepted and discarded.
- [ ] Replace the bundled `resources/config.toml` and `resources/state.toml` with real
      `hasp` templates containing `[key_groups]` and `[ssh_key]`.
- [ ] Add `hasp init` to write those templates to the XDG paths, and make missing-file
      handling a clean error instead of a `FileNotFoundError` traceback.
- [ ] `ruff check --fix` + `ruff format`; add `[tool.ruff]` with `line-length = 120`.
- [ ] Delete `setup.cfg`; move the pytest config into `[tool.pytest.ini_options]`.
- [ ] Delete `requirements.txt`, `requirements-dev.txt`, `.bkup/`; add `.python-version`.

**Exit criterion:** `hasp list key` run twice in a row succeeds twice, on a clean machine
with no pre-existing config.

### Phase 1 — Decide the store, then converge on it (1–2 days)

- [ ] Answer §5.1 in writing at the top of this file's successor (an ADR under `docs/`).
- [ ] Rewrite `new key` / `list key` / `show key` / `sync key` to all use the chosen store.
- [ ] Delete the losing path, including the commented-out `update_key()` at `hasp.py:232-277`
      and the dead `get_key_list()` in `io.py`.
- [ ] Delete `factory/` (§5.2) unless you are choosing it over the verb matrix.

**Exit criterion:** a key created by `new` is visible to `list` and is preserved by `sync`.

### Phase 2 — Finish `sync key` (1–2 days) — *this is the MVP*

- [ ] Uncomment and repair the init/prune block in `sync_keys()`; honour `--all`, `--name`,
      `--init`, `--prune`, `--both`.
- [ ] Implement `prune_state_groups()`.
- [ ] Finish the key↔group association at the `# TODO: Left off here` marker.
- [ ] Fix the `new key` type/bits mismatch (§3.3) — the single source of truth for a key's
      metadata should be `read_real_key()`, called after generation, not the CLI's own guess.

**Exit criterion:** point `hasp` at a real `~/.ssh`, run `sync key`, and get an accurate
inventory including symlink aliases and group membership.

### Phase 3 — Tests (1–2 days)

- [ ] Write `pytest-bdd` step definitions for the scenarios already in `keys.feature`.
- [ ] Add a `tmp_path`-based fixture that builds a throwaway `~/.ssh` with generated keys;
      sanitise `bk` into a committed state fixture.
- [ ] Add Click `CliRunner` smoke tests for every registered command — including the
      "run it twice" case from Phase 0.
- [ ] Add `.pre-commit-config.yaml` (`pre-commit` is already a dependency) and a GitHub
      Actions workflow running `ruff`, `ty`, and `pytest`.

**Exit criterion:** `pytest` collects a non-zero number of tests and CI is green.

### Phase 4 — Complete the key noun (2–3 days)

- [ ] `find key --md5` with the normalisation the README promises (`de:ad:be:ef`, `deadbeef`,
      `DEADBEEF` all match).
- [ ] `edit key` — rename, re-comment, reassign groups.
- [ ] The maintenance scenarios `keys.feature` already specifies: detect duplicate keys and
      offer to convert them to symlinks; regenerate missing `.pub` files.
- [ ] Replace the hand-rolled `print(f'\t{k:>8}: "{v}"')` in `present_key` with a `rich`
      table — `rich` is a dependency and the console is already configured.

### Phase 5 — The `host` and `config` nouns (open-ended)

This is where the original ambition lives, and the hardest requirement in the README is
**"SSH config file may be edited by hand safely (won't lose edits)"** — a round-tripping,
comment-preserving `ssh_config` parser. Nothing in the current dependency set does that;
`tomlkit` round-trips TOML, not `ssh_config`. Investigate before committing:

- [ ] Survey `storm`, `advanced-ssh-config`, `paramiko.SSHConfig`, `sshconf` for a
      round-tripping parser worth adopting or vendoring. This is a genuine build-vs-buy
      decision and deserves its own spike.
- [ ] Uncomment and finish the `SshHost` / `SshConfig` ORM models.
- [ ] Implement rolling backups on write (`config.feature`'s one scenario).

### Deferred

GPG and Git identity management (README "Future Ideas"). Do not touch until Phase 4 lands.

---

## 7. Recommended first move

Phase 0 is mechanical, has no open design questions, and converts the repo from "crashes on
second use" to "demonstrably works." Do it in one sitting, commit it, and you will have a
tool you can actually run while you think about §5.1 — which is the only decision on this
list that genuinely requires thought.
