# Fastlane Resolution Center API Research

Research into how fastlane accesses the App Store Connect Resolution Center — the communication channel where Apple's App Review team sends rejection reasons and developers can reply.

## Summary

The Resolution Center is **not exposed in Apple's official public App Store Connect REST API**. Fastlane accesses it through undocumented internal endpoints that require Apple ID cookie-based authentication (JWT API key tokens do not work).

## API Layers

### 1. Legacy "Tunes" API (offline since July 2022)

**Base URL:** `appstoreconnect.apple.com/WebObjects/iTunesConnect.woa/ra/`

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Get threads | GET | `ra/apps/{app_id}/platforms/{platform}/resolutionCenter?v=latest` |
| Reply to thread | POST | `ra/apps/{app_id}/platforms/{platform}/resolutionCenter` |

Supported both reading and replying. Apple retired these endpoints around July 2022.

### 2. Current "iris/v1" API (undocumented, reverse-engineered)

**Base URL:** `appstoreconnect.apple.com/iris/v1/`

Implemented in [fastlane PR #20726](https://github.com/fastlane/fastlane/pull/20726), released in fastlane 2.211.0 (November 2022).

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List threads | GET | `/iris/v1/resolutionCenterThreads` |
| Get messages | GET | `/iris/v1/resolutionCenterThreads/{thread_id}/resolutionCenterMessages` |
| Get rejections | GET | `/iris/v1/reviewRejections` |

**Data format:** JSON:API (same structure as the official ASC API — `type`, `id`, `attributes`, `relationships`).

**Authentication:** Apple ID cookie-based web session only. JWT API key tokens do **not** work.

### 3. Official App Store Connect REST API — No Coverage

The official API at `api.appstoreconnect.apple.com` has zero Resolution Center endpoints. It supports:
- Submitting for review (`POST /v1/reviewSubmissions`)
- Managing review details/attachments (`/v1/appStoreReviewDetails`)
- Reading/replying to customer reviews (`/v1/customerReviews`, `/v1/customerReviewResponses`)

But **not** reading rejection reasons or replying to App Review.

## Fastlane Source Files

| File | Purpose |
|------|---------|
| `spaceship/lib/spaceship/connect_api/tunes/tunes.rb` | API client (`get_resolution_center_threads`, `get_resolution_center_messages`, `get_review_rejection`) |
| `spaceship/lib/spaceship/connect_api/models/resolution_center_thread.rb` | Thread model |
| `spaceship/lib/spaceship/connect_api/models/resolution_center_message.rb` | Message model |
| `spaceship/lib/spaceship/connect_api/models/review_rejection.rb` | Rejection model |
| `spaceship/lib/spaceship/connect_api/models/actor.rb` | Actor model (message sender) |

## Data Models

### ResolutionCenterThread

- `state` — thread state
- `thread_type` — one of: `REJECTION_BINARY`, `REJECTION_METADATA`, `REJECTION_REVIEW_SUBMISSION`, `APP_MESSAGE_ARC`, `APP_MESSAGE_ARB`, `APP_MESSAGE_COMM`
- `can_developer_add_node` — whether replies are allowed
- `created_date`, `last_message_response_date`
- relationships: `resolution_center_messages`, `app_store_version`

### ResolutionCenterMessage

- `message_body` — the message text
- `created_date`
- relationships: `rejections` (linked ReviewRejection objects), `from_actor` (Actor)

### ReviewRejection

- `reasons` — array of rejection reason strings/codes

### Actor

- `actor_type`, `user_first_name`, `user_last_name`, `user_email`, `api_key_id`

## Key Constraints

| Constraint | Detail |
|------------|--------|
| No public API | Apple has never exposed Resolution Center in the official REST API |
| Apple ID auth only | iris/v1 endpoints require cookie-based session auth, not JWT |
| Undocumented API | Reverse-engineered from web UI network traffic; can break at any time |
| 2FA required | Apple ID auth requires two-factor authentication |
| Read-only (current) | The iris/v1 implementation in fastlane is read-only; reply capability only existed in the legacy API |
| 4,000 char limit | Replies to App Review are capped at 4,000 characters |

## Implications for ASC CLI

1. **Cannot use JWT API key auth** — Resolution Center requires Apple ID session cookies, which is a fundamentally different auth model than what the ASC CLI uses.
2. **Undocumented and fragile** — Any implementation would depend on reverse-engineered endpoints that Apple can change without notice.
3. **Read-only** — Even fastlane only supports reading threads/messages in the current implementation, not replying.
4. **Not feasible for CI/CD** — The 2FA + cookie session requirement makes automation difficult.
