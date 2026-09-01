# Release policy

Releases are immutable observations of a successful GitHub Actions run on the merged implementation PR.

1. Create exactly one implementation PR from the implementation branch.
2. Require the Actions conformance and quality job to pass before merge.
3. Merge the PR without rewriting failed runs, tags, releases, or PR history.
4. Create an annotated tag whose message names the contract and denominator.
5. Create a release from that tag and attach the caller-owned conformance report as a release asset.

The release note must state the merge SHA, tag object and target, Actions run/job, asset ID/size/digest, the seven fixture outcomes, the exact metric vectors, and the local validation command counts. No external utility evidence is used as CI or maintainer self-test evidence.
