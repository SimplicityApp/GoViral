# Community Management API - Overview

Source: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/community-management-overview?view=li-lms-2025-11

The Community Management API enables developers to manage LinkedIn company pages on behalf of clients, access account details (admins, roles, follower details), and analytics including comments, reactions, and page activity.

## Approved Use Cases

### Page Management
Use Organizations and Brand APIs to create, update, and delete organization and brand information. Create and manage company posts, comments, and reactions while monitoring member engagement.

### Page Analytics
- **Follower Statistics**: Lifetime and time-bound follower stats for organizations
- **Page Statistics**: Lifetime and time-bound view/click stats for org/brand pages
- **Share Statistics**: Lifetime and time-bound share stats with aggregate data
- **Social Metadata**: Reactions, comments on shares/posts (organic and sponsored)
- **Video Analytics**: Watch time, video views, video viewers

### Member Analytics
- **Follower Statistics**: Lifetime and time-bound follower stats for a member
- **Post Statistics**: Lifetime and time-bound post stats for specific posts or members
- **Video Analytics**: Watch time, video views, video viewers for member-owned videos

### Profile Management
Create and manage posts, comments, and reactions on behalf of individual profiles.

### Employee Advocacy
Enable brands to leverage employee networks for content resharing via:
- Organization Social Actions Notifications
- People Typeahead for @mentions

## Program Tiers

- **Development Tier**: Initial approval with limited API call volume
- **Standard Tier**: Full access, requires upgrade from Development Tier with screencast demo

## Member Post Access (r_member_social)

**IMPORTANT**: `r_member_social` is a **closed** permission. LinkedIn is **not accepting access requests** at this time due to resource constraints.

This means reading a member's own posts via the API is currently not available to new applicants.

## FAQ Highlights

**Q: How to get Community Management API if I already have Advertising API access?**
1. Create a new developer app with the same company page as your Advertising API app
2. Apply for Community Management API Development Tier access
3. Upon approval, complete Standard Tier access request
4. Once approved, request Community Management API Standard Tier on your existing Ads API app

**Q: Can the Community Management API be added to an existing app?**
Yes, follow the steps above. Both apps must be verified with the same company page.

**Q: Why is Community Management API grayed out?**
Only request Development Tier access with **new** developer applications that don't have access to other API products.

## Rate Limits (Development Tier)

| Scope | Limit |
|---|---|
| Per App | 500 requests / 24 hrs |
| Per Member | 100 requests / 24 hrs |
| BATCH_GET | Not allowed |
| Webhooks | Disabled |
