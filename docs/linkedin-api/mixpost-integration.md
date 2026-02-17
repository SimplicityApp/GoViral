# Mixpost - LinkedIn Community Management API Integration

Source: https://docs.mixpost.app/services/social/linked-in/community-management-api/

Mixpost is a self-hosted social media management platform (Buffer alternative) that integrates with LinkedIn via the Community Management API.

## Key Requirement

The Community Management API is only available to **registered legal entities** (LLC, Corporations, 501(c), etc.), not individual developers.

## Setup Process

### Step 1: Create a LinkedIn Developer App
- Go to LinkedIn Developer Dashboard
- Create a new application with company details, privacy policy, and branding

### Step 2: Request Community Management API Access
LinkedIn requires:
- Business details (organization name, website, country, address)
- Use case descriptions: "Page management", "Page analytics", "Profile management"
- Detailed explanation of planned usage (scheduling, engagement, reporting)
- Demonstration video with audio explanation in English showing complete OAuth flow and all use cases

### Step 3: Configure Credentials
- Copy Client ID and Primary Client Secret from LinkedIn Auth tab
- Input into Mixpost's third-party services configuration

### Step 4: Set Up Redirect URIs
Two redirect URIs in LinkedIn's dashboard:
- `https://example.com/<MIXPOST_CORE_PATH>/callback/linkedin`
- `https://example.com/<MIXPOST_CORE_PATH>/callback/linkedin_page`

## Important Notes
- LinkedIn approval can take several days
- Approval is mandatory before scheduling functionality becomes available
- The Community Management API must be the **only product** on the LinkedIn developer application
