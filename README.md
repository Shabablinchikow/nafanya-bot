# nafanya-bot

Щитпост бот для Telegram на основе ChatGPT

## Deployment

Push to `master` builds the image and deploys it to a Kubernetes cluster with Helm
(`.github/workflows/build-docker.yml`). Cluster, registry and application configuration come
from repository secrets and variables; nothing environment-specific lives in this repo.

## Per-group ignore settings

Users can opt out of Nafanya interactions without being administrators:

- `/ignoreme` — ignore your messages in the current group.
- `/unignoreme` — opt back in, even while ignored.

Group administrators can manage other accounts (including bots):

- `/ignoreuser <numeric Telegram user ID>`
- `/unignoreuser <numeric Telegram user ID>`
- Alternatively, reply to a user's message with `/ignoreuser` or `/unignoreuser`.
- `/ignoredusers` — show the current group's list. These commands are also
  documented in `/chatConfigure`.

Commands can be addressed explicitly, e.g. `/ignoreme@NafanyaTheKitchenBot`.
Self-service commands never affect anyone except their sender; commands sent
anonymously or as a channel cannot change ignore settings. If Telegram does not
forward another bot's messages, an admin can add that bot by numeric ID instead.
No tokens or credentials are needed in this list.

Ignoring is per group/supergroup, not global, and does not change private-chat
behaviour. Ignored senders cannot trigger ordinary commands, AI replies, random
interruptions or link previews. Ignore-management commands remain available with
the permissions above. Other people can still interact with Nafanya when replying
to an ignored user's message; this feature is not a content-erasure mechanism.

### Deployment

Build and deploy the updated application normally. Startup's existing GORM
migration creates an `ignored_users` table (with the configured `DB_PREFIX`),
initially empty. The database account needs permission to create this table.
Existing chat settings are unchanged. Entries persist across restarts; adding and
removing users takes effect on subsequent messages without a restart. Requests
already in flight are not cancelled. Database lookup failures silently suppress
normal group-message processing rather than risk replying to an opted-out user.
Rolling back to an older version stops enforcing the list but leaves its data.

Run checks with `go test ./...` and `go vet ./...`.

To exercise migrations, persistence and concurrent opt-outs against a disposable
PostgreSQL database, set `NAFANYA_TEST_POSTGRES_DSN` when running `go test ./...`.
The integration test creates and cleans up uniquely prefixed tables; use a UTF-8
connection/database. It is skipped when the variable is unset.
