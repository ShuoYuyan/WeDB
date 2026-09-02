# Contributing to WeDB

Thank you for your interest in contributing to WeDB! This project is
licensed under **GNU Affero General Public License v3 (AGPL-3.0)**.
Contributions are made under the same license.

This project uses the **Developer Certificate of Origin (DCO)** — the
same mechanism used by the Linux kernel, Kubernetes, Docker, and many
other major open-source projects. It is lightweight: no CLA to sign,
just a `Signed-off-by` line in your commits.

## Why DCO instead of CLA?

* **No paperwork**: Contributors don't sign anything — just `git commit -s`.
* **No friction**: Same process as Linux kernel / Kubernetes.
* **Legal clarity**: DCO explicitly states you have the right to submit
  the contribution under the project's license.

## How to Sign Your Work

Use `git commit -s` to automatically add a `Signed-off-by` line:

```bash
git commit -s -m "feat(storage): add MVCC snapshot isolation"
```

This appends:
```
Signed-off-by: Your Name <your.email@example.com>
```

By adding this line, you certify the following (from
[developercertificate.org](https://developercertificate.org/)):

```
Developer Certificate of Origin 1.1

By signing the contribution, the contributor certifies that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/)
specification. Example:

```
feat(storage): add MVCC snapshot isolation for read transactions

Implemented per-statement snapshot for REPEATABLE READ level. The
snapshot captures the current committed transaction ID and reuses
it for subsequent reads within the same transaction.

Signed-off-by: Your Name <your.email@example.com>
```

Common prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`,
`perf:`, `build:`, `ci:`.

## Reporting Issues

* **Bugs**: Use the GitHub issue tracker. Include reproduction steps,
  expected vs actual behavior, and your environment.
* **Security issues**: Please email the maintainer directly (do not open
  a public issue).
* **Feature requests**: Open a GitHub issue with the `enhancement` label.
  Describe the use case, not just the solution.

## Code Style

* Go: `gofmt` + `go vet`
* C: K&R style, 4-space indent
* Pascal: standard Delphi formatting
* PowerShell: consistent with existing scripts (see `e2e.ps1`)

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit with DCO sign-off (`git commit -s -m "feat: ..."`)
4. Push to your fork (`git push origin feature/my-feature`)
5. Open a Pull Request against `main`
6. Ensure CI passes (when available)
7. Wait for review

## Important: License Reminder

WeDB is licensed under **AGPL-3.0**. By contributing, you agree that
your contributions will be licensed under the same terms. The DCO
sign-off confirms you have the right to make the contribution under
these terms.

If you or your employer require a proprietary use license (without
AGPL obligations), please contact the maintainer for a commercial
license arrangement.
