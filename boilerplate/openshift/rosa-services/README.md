# ROSA services ownership aliases

This convention copies the shared [OWNERS_ALIASES](../../../OWNERS_ALIASES) file
to the root of a consuming repository. It does not create or replace `OWNERS`;
each repository maintains its own `OWNERS` files and can reference the shared
aliases in them.

To subscribe, add the following line to `boilerplate/update.cfg`:

```text
openshift/rosa-services
```

Run `make boilerplate-update` and commit the resulting changes. Subsequent
changes to alias membership take effect in a consuming repository only after
that repository updates and commits its boilerplate files.

Do not subscribe alongside another convention that also writes the root
`OWNERS_ALIASES`, unless both conventions intentionally use the same source.
