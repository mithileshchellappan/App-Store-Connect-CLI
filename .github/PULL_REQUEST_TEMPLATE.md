## Summary

- 

## Validation

- [ ] `make format`
- [ ] `make check-docs`
- [ ] `make lint`
- [ ] `ASC_BYPASS_KEYCHAIN=1 make test`

If this PR only updates `docs/wall-of-apps.json`, use `make check-wall-of-apps` instead of the full checklist above.

## Wall of Apps (only if this PR adds/updates a Wall app)

- [ ] I used `asc apps wall submit --app "1234567890" --confirm` (or made the equivalent single-file update manually)
- [ ] This PR only updates `docs/wall-of-apps.json`
- [ ] I ran `make check-wall-of-apps`

Entry template:

```json
{
  "app": "Your App Name",
  "link": "https://apps.apple.com/app/id1234567890",
  "creator": "your-github-handle",
  "platform": ["iOS"]
}
```

Common Apple labels: `iOS`, `macOS`, `watchOS`, `tvOS`, `visionOS`.
