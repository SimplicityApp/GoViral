# LinkedIn Posts API

Source: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api?view=li-lms-2025-11

The Posts API facilitates the creation and retrieval of organic and sponsored posts.
It replaces the legacy ugcPosts API.

## Required Headers

All APIs require:
- `Linkedin-Version: {YYYYMM}`
- `X-Restli-Protocol-Version: 2.0.0`

## Permissions

| Permission | Description |
|---|---|
| `w_organization_social` | Post, comment, and like posts on behalf of an organization. Restricted to orgs where the authenticated member has ADMINISTRATOR, DIRECT_SPONSORED_CONTENT_POSTER, or CONTENT_ADMIN roles. |
| `r_organization_social` | Retrieve organizations' posts, comments, and likes. Same role restrictions as above. |
| `w_member_social` | Post, comment, and like posts on behalf of an authenticated member. |
| `r_member_social` | Retrieve posts, comments, and likes on behalf of an authenticated member. **Restricted** — available to **approved users only**. |

## Content Types

| Content Type | Organic | Sponsored |
|---|---|---|
| Text only | Yes | Yes |
| Images | Yes | Yes |
| Videos | Yes | Yes |
| Documents | Yes | Yes |
| Article | Yes | Yes |
| Carousels | No | Yes |
| MultiImage | Yes | No |
| Poll | Yes | No |
| Celebration | Yes | No |

## Post Schema (Key Fields)

| Field | Format | Description | Required |
|---|---|---|---|
| author | Person or Organization URN | Author of the post | create-only required |
| commentary | little text | User generated commentary | required |
| content | content type | Posted content (media, article, poll, etc.) | optional |
| visibility | MemberNetworkVisibility | CONNECTIONS, PUBLIC, LOGGED_IN, CONTAINER | required |
| distribution | distribution type | Feed distribution settings | create-only required |
| lifecycleState | Enum | DRAFT, PUBLISHED, PUBLISH_REQUESTED, PUBLISH_FAILED | required |
| id | ugcPostUrn or shareUrn | Unique ID | read-only |
| createdAt | Time (ms since epoch) | Creation time | read-only |
| lastModifiedAt | Time (ms since epoch) | Last modified time | read-only |

## Create a Post

### Text-Only Post

```http
POST https://api.linkedin.com/rest/posts
```

```json
{
  "author": "urn:li:organization:5515715",
  "commentary": "Sample text Post",
  "visibility": "PUBLIC",
  "distribution": {
    "feedDistribution": "MAIN_FEED",
    "targetEntities": [],
    "thirdPartyDistributionChannels": []
  },
  "lifecycleState": "PUBLISHED",
  "isReshareDisabledByAuthor": false
}
```

Response: `201` with `x-restli-id` header containing the Post ID (e.g., `urn:li:share:6844785523593134080`).

### Post with Media (Image/Video/Document)

Same as above, but include a `content.media` object:

```json
"content": {
  "media": {
    "title": "title of the video",
    "id": "urn:li:video:C5F10AQGKQg_6y2a4sQ"
  }
}
```

Media must first be uploaded via Images API, Videos API, or Documents API to obtain the URN.

## Get Posts by URN

```http
GET https://api.linkedin.com/rest/posts/{encoded ugcPostUrn|shareUrn}
```

Optional parameter: `viewContext` (`READER` default, or `AUTHOR` for latest version including drafts).

URNs in URL params must be URL encoded (e.g., `urn:li:ugcPost:12345` -> `urn%3Ali%3AugcPost%3A12345`).

## Batch Get Posts

```http
GET https://api.linkedin.com/rest/posts?ids=List({encoded ugcPostUrn},{encoded ugcPostUrn})
```

Requires header: `X-RestLi-Method: BATCH_GET`

## Find Posts by Authors

```http
GET https://api.linkedin.com/rest/posts?author={encoded PersonURN|OrganizationURN}&q=author&count=10&sortBy=LAST_MODIFIED
```

Requires header: `X-RestLi-Method: FINDER`

**Important**: To retrieve posts authored by a person, `r_member_social` permission is required.
To retrieve posts authored by an organization, `r_organization_social` permission is required.

Parameters:
- `author` (required): Person or Organization URN
- `viewContext` (optional): READER or AUTHOR (default: READER)
- `start` (optional): Pagination start index (default: 0)
- `count` (optional): Items per page, max 100 (default: 10)
- `sortBy` (optional): LAST_MODIFIED or CREATED (default: LAST_MODIFIED)

## Update Posts

```http
POST https://api.linkedin.com/rest/posts/{encoded ugcPostUrn|shareUrn}
```

Requires header: `X-RestLi-Method: PARTIAL_UPDATE`

Updatable fields: `commentary`, `contentCallToActionLabel`, `contentLandingPage`, `lifecycleState`, `adContext`.

## Delete Posts

```http
DELETE https://api.linkedin.com/rest/posts/{encoded ugcPostUrn|shareUrn}
```

Deletions are idempotent. Returns `204`.

## Mentions and Hashtags

Use markdown-like syntax in commentary:
- Organization mention: `@[Devtestco](urn:li:organization:2414183)`
- Hashtag: `#coding`

Text must match the entity's actual name (case-sensitive) for link conversion.
