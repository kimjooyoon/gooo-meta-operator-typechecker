# Immutable release activation

Before publishing a tag, the repository owner enables release immutability
through GitHub's repository Administration REST API:

```text
PUT /repos/kimjooyoon/gooo-meta-operator-typechecker/immutable-releases
```

The activation response was followed by a repository settings read returning
`enabled=true`. The release workflow intentionally does not repeat this
admin-only read with `GITHUB_TOKEN`; it verifies the published release through
the release API and requires `.immutable=true`. A release is never replaced
or overwritten if its tag or release name already exists.
