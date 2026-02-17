# X (Twitter) API Key Setup

This guide walks you through obtaining API credentials for X (formerly Twitter) to use with GoViral.

## 1. Create a Developer Account

1. Go to [developer.x.com](https://developer.x.com) and sign in with your X account.
2. If you don't have a developer account, click **Sign up for Free Account** (or apply for Basic/Pro if you need higher rate limits).
3. Complete the application by describing your use case and accepting the developer agreement.

## 2. Create a Project and App

1. In the Developer Portal, navigate to **Projects & Apps** in the sidebar.
2. Click **+ Add Project**, give it a name (e.g., "GoViral"), and select your use case.
3. Inside your new project, click **+ Add App** and name it (e.g., "goviral-cli").

## 3. Copy Your App Credentials

After creating the app, the Developer Portal shows you three credentials:

| Portal label | Config field |
|---|---|
| **Consumer Key** | `api_key` |
| **Consumer Secret** (Secret Key) | `api_secret` |
| **Bearer Token** | `bearer_token` |

Copy all three immediately — they are only shown once. If you lose them, you can regenerate them from the **Keys and tokens** tab of your app.

> The **Bearer Token** is required for GoViral to read posts and trending content via the X API v2. The Consumer Key/Secret are needed for future write access.

## 4. Get OAuth 2.0 Client Credentials (for Write Access)

When you request write access for your app, the Developer Portal provides OAuth 2.0 credentials:

| Portal label | Config field |
|---|---|
| **Client ID** | `client_id` |
| **Client Secret** | `client_secret` |

These are used for the OAuth 2.0 authorization flow (PKCE / Confidential Client) to obtain user-scoped access tokens for posting content.

> **Note:** These are **not** the same as the OAuth 1.0a Access Token/Secret below. Client ID/Secret initiate an auth flow; Access Token/Secret are direct user credentials.

## 5. (Optional) Generate OAuth 1.0a Access Token and Secret

These are an alternative way to authenticate for write access. They are **not** shown during app creation — you must generate them separately.

1. Go to your app's **Keys and tokens** tab.
2. Scroll to **Authentication Tokens**.
3. Click **Generate** under **Access Token and Secret**.
4. Make sure the permissions are set to **Read and Write** if you want to post content.
5. Copy both the **Access Token** and **Access Token Secret**.

## 6. Find Your Username

Your username is your X handle without the `@` symbol. For example, if your profile URL is `https://x.com/johndoe`, your username is `johndoe`.

## 7. Add Credentials to Config

Edit (or create) `~/.goviral/config.yaml` and fill in the `x:` section:

```yaml
x:
  api_key: "YOUR_CONSUMER_KEY"
  api_secret: "YOUR_CONSUMER_SECRET"
  bearer_token: "YOUR_BEARER_TOKEN"
  access_token: "YOUR_ACCESS_TOKEN"
  access_token_secret: "YOUR_ACCESS_TOKEN_SECRET"
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
  username: "YOUR_USERNAME"
```

At minimum, `bearer_token` and `username` are required for read operations. For write access, provide either OAuth 2.0 credentials (`client_id`, `client_secret`) or OAuth 1.0a credentials (`access_token`, `access_token_secret`).

See `config.yaml.example` in the project root for a full config template.
