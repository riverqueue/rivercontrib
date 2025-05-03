# rivercontrib development

## Run tests

Run tests:

    make test

## Run lint

Run the linter and try to autofix:

    make lint

## Releasing a new version

1. Fetch changes to the repo and any new tags. Export `VERSION` by incrementing the last tag. Execute `update-mod-version` to add it the project's `go.mod` files:

    ```shell
    git checkout master && git pull --rebase
    export VERSION=v0.x.y
    make update-mod-version
    git checkout -b $USER-$VERSION
    ```

2. Prepare a PR with the changes, updating `CHANGELOG.md` with any necessary additions at the same time. Include **`[skip ci]`** in the commit description so that CI doesn't run and pollute the Go Module cache by trying to fetch a version that's not available yet (and it would fail anyway). Have it reviewed and merged.

3. Upon merge, pull down the changes, tag each module with the new version, and push the new tags:

    ```shell
    git checkout master && git pull --rebase
    git tag datadogriver/$VERSION -m "release datadogriver/$VERSION"
    git tag nilerror/$VERSION -m "release nilerror/$VERSION"
    git tag otelriver/$VERSION -m "release otelriver/$VERSION"
    git tag $VERSION
    ```

4. Push new tags to GitHub:

    ```shell
    git push --tags
    ```

5. Cut a new GitHub release by visiting [new release](https://github.com/riverqueue/rivercontrib/releases/new), selecting the new tag, and copying in the version's `CHANGELOG.md` content as the release body.
