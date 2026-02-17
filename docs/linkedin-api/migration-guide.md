# Migration Guide for Community Management API

Source: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/community-management-api-migration-guide?view=li-lms-2025-02

## Overview

LinkedIn Marketing Solutions released a new Community Management API product. This guide covers migration from the older Marketing Developer Platform (MDP) product.

## Features Supported

- **Page Access management**: Find org admins, access control details, brand profiles. Permission: `rw_organization_admin`
- **Content management**: Create, modify, search, read rich content (articles, images, videos, carousels, polls, comments, likes). Permissions: `w_organization_social`, `r_organization_social`, `w_member_social`
- **Content Analysis**: Analyze content social actions (comments, likes, reactions), follower/page/share stats. Permissions: `w_organization_social`, `r_organization_social`, `w_member_social`
- **At mentions member** (new): Typeahead search API to find org followers for @mentions

## Applying for Access

### Prerequisites
- Company Page verification (use same page if you have existing MDP app)
- Must be a registered legal entity (not individual developers)

### Provisioning Tiers

**Development Tier** (default):
- 500 API calls per app per 24 hrs
- 100 API calls per member per 24 hrs
- No BATCH_GET allowed
- No Webhooks for Social Actions
- Must build core use cases within 12 months

**Standard Tier** (full access):
- No API restrictions
- Requires screencast video demo of OAuth flow + all core functionality
- Must show post creation, engagement display, member data handling

## Onboarding for Existing MDP Developers

If you already have MDP access, you must still create a new app:

1. Note down the company page verified for your MDP App
2. Create a new developer application at developer.linkedin.com
3. Verify new app with the same company page
4. Request Community Management API Development tier
5. Get approved, then request Standard tier upgrade
6. Navigate to your original MDP app -> Products -> Request Community Management API
7. You should see Standard tier option (both apps share same company page)
8. Fill out Standard tier form with new app's client ID to skip most questions

The newly created app was for verification only and can be discarded after your main app has Community Management API access.

## New Permissions (replacing old MDP permissions)

### Organization Permissions

| Old Permission | New Permission | Affected APIs |
|---|---|---|
| `r_organization_social` | `r_organization_social_feed` | /reactions, /socialActions, /socialActions/comments, /socialActions/likes, /socialMetadata (GET, BATCH_GET, FINDER) |
| `w_organization_social` | `w_organization_social_feed` | /reactions, /socialActions/comments, /socialActions/likes, /socialMetadata (CREATE, DELETE, PARTIAL_UPDATE) |

### Member Permissions

| Old Permission | New Permission | Affected APIs |
|---|---|---|
| `w_member_social` | `w_member_social_feed` | /reactions, /socialActions/comments, /socialActions/likes, /socialMetadata (CREATE, DELETE, PARTIAL_UPDATE) |

**Deprecation**: Old permissions (`r_organization_social`, `w_organization_social`, `w_member_social`) stopped working after June 2023.

## Member Feed Management (r_member_social)

Apps with `r_member_social` access must be approved for Standard tier Community Management APIs to retain access. Failure to complete vetting by June 2023 resulted in removal of Member Feed Management API product.

**Current status**: `r_member_social` is a **closed** permission. LinkedIn is not accepting new access requests.
