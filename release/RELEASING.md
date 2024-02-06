# RELEASING - FOR DEVELOPERS

Before merging a release into the `main` branch, please verify the following:

1. All changes for the release are merged into the `develop` branch.
2. The release candidate has been verified by an authorized developer.
3. All unit tests pass.
4. The application can be compiled and run on all releasable targets.
5. `version/release.go` has been updated with the release candidate's version identifier.
6. The release candidate's latest commit, which should include the updated `version/release.go`, has been tagged with the version via `git tag`.
7. All changes for the release, from `develop`, are pushed into a `release-{version}` branch.