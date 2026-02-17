# Stackposts - LinkedIn API Product Restriction Workaround

Source: https://doc.stackposts.com/docs/stackposts-v9/problems-solutions/linkedin-this-api-product-requires-that-it-be-the-only-product-on-the-application-for-legal-and-security-reasons-15917/

## The Problem

LinkedIn's Community Management API has a strict requirement:

> "This API product requires that it be the only product on the application for legal and security reasons."

If your developer app already has other LinkedIn products enabled (like "Share on LinkedIn" or "Sign In with LinkedIn"), you **cannot** add the Community Management API to it.

## Why This Happens

LinkedIn enforces this restriction at the application level. Apps with multiple LinkedIn products cannot simultaneously access the Community Management API. Other apps with this API were either:
- Created specifically for that single API, or
- Belong to LinkedIn's approved partner program

## The Solution: Create a Dedicated App

1. Visit the LinkedIn Developer Portal at https://www.linkedin.com/developers/apps
2. Create a **new** application with a descriptive name
3. **Do not add any other products** at this stage
4. Request access to the Community Management API
5. Provide a use case explanation (e.g., "Our platform enables verified business owners to manage LinkedIn Company Pages, publish content, and access analytics within LinkedIn's policy guidelines")

## Note on Product Compatibility

Marketing-tier APIs (Advertising, Lead Sync) also require a dedicated app and cannot be mixed with "Share on LinkedIn" or "Sign In with LinkedIn".

However, after the initial Community Management API Development Tier approval, if both apps are verified under the same company page, both products can eventually exist on the same developer application (see the migration guide FAQ).
