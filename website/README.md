# Mintlify Website

This directory contains the recovered Mintlify source for `https://docs.asccli.sh`.

The content was restored from the live site's markdown exports on 2026-03-27 and
placed in a dedicated `website/` monorepo path so it can be managed alongside the
CLI without colliding with the repository's existing `docs/` directory.

## Local preview

```bash
cd website
npx mint dev
```

## Validation

```bash
cd website
npx mint validate
npx mint broken-links
```
