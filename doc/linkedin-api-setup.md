# LinkedIn API Key Setup

This guide walks you through obtaining API credentials for LinkedIn to use with GoViral.

## 1. Create a LinkedIn Developer Application

1. Go to [linkedin.com/developers](https://www.linkedin.com/developers/) and sign in with your LinkedIn account.
2. Click **Create App**.
3. Fill in the required fields:
   - **App name** (e.g., "GoViral")
   - **LinkedIn Page** — associate it with a LinkedIn Company Page (create one if needed)
   - **App logo** — upload any image
4. Accept the legal terms and click **Create app**.

## 2. Request Product Permissions

Your app needs specific API products enabled to access the required endpoints.

1. In your app's settings, go to the **Products** tab.
2. Request access to:
   - **Share on LinkedIn** — required for posting content
   - **Sign In with LinkedIn using OpenID Connect** — required for authentication
3. Some products are approved instantly; others may take a few minutes.

## 3. Get Client ID and Client Secret

1. Go to the **Auth** tab of your app.
2. Copy the **Client ID** and **Client Secret** shown at the top of the page.

## 4. Generate an OAuth2 Access Token

LinkedIn requires a 3-legged OAuth 2.0 flow to generate access tokens.

### Step A: Add a Redirect URL

1. In the **Auth** tab, scroll to **OAuth 2.0 settings**.
2. Under **Authorized redirect URLs for your app**, add `http://localhost:8080/callback` and save.

### Step B: Get an Authorization Code

Open this URL in your browser (replace `YOUR_CLIENT_ID`):

```
https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:8080/callback&scope=openid%20profile%20w_member_social
```

LinkedIn will ask you to sign in and authorize your app. After you approve, your browser will redirect to something like:

```
http://localhost:8080/callback?code=AUTHORIZATION_CODE
```

The page won't load (nothing is listening on localhost:8080), but that's fine — copy the `code` value from the URL bar.

### Step C: Exchange the Code for an Access Token

Run this curl command (replace the placeholders):

```bash
curl -X POST https://www.linkedin.com/oauth/v2/accessToken \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=AUTHORIZATION_CODE&redirect_uri=http://localhost:8080/callback&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

The JSON response contains your `access_token`. Copy it.

> **Note:** The authorization code expires within minutes — run the curl command right after step B. Access tokens expire after ~60 days and will need to be regenerated.

## 5. Add Credentials to Config

Edit (or create) `~/.goviral/config.yaml` and fill in the `linkedin:` section:

```yaml
linkedin:
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
  access_token: "YOUR_ACCESS_TOKEN"
```

All three fields are required for LinkedIn integration.

See `config.yaml.example` in the project root for a full config template.
