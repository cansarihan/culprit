# Security policy

Culprit reads a repository through `git log` and `git show`. It never checks
out, builds or runs the code it reads, and the panel only accepts commit hashes
it can validate, so a request cannot pass options through to git.

Report a suspected vulnerability privately through GitHub's security advisory
flow for this repository rather than a public issue. Include a minimal
reproduction and the affected version.
