export function Privacy() {
  return (
    <div className="mx-auto max-w-2xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Privacy Policy</h2>
      <div className="space-y-4 text-sm leading-relaxed text-[var(--color-text-secondary)]">
        <p className="text-xs text-[var(--color-text-secondary)]">Last updated: February 28, 2026</p>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">1. Information We Collect</h3>
          <p>GoViral collects and processes the following types of information:</p>
          <ul className="mt-2 list-inside list-disc space-y-1">
            <li>
              <strong>API Keys and Credentials</strong> — stored locally in your configuration file
              (~/.goviral/config.yaml) and never transmitted to our servers
            </li>
            <li>
              <strong>Content Data</strong> — posts, drafts, and generated content stored in your
              local SQLite database
            </li>
            <li>
              <strong>Platform Data</strong> — trending posts and user profile information fetched
              from connected platforms
            </li>
          </ul>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">2. How We Use Your Information</h3>
          <p>Your information is used to:</p>
          <ul className="mt-2 list-inside list-disc space-y-1">
            <li>Generate and publish content on your behalf</li>
            <li>Analyze trending topics and build persona profiles</li>
            <li>Store your content history locally for future reference</li>
          </ul>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">3. Data Storage</h3>
          <p>
            All data is stored locally on your machine. Configuration, credentials, and content
            history remain in your local filesystem and database. We do not operate cloud servers
            that store your personal data.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">4. Third-Party Services</h3>
          <p>
            The Service communicates with third-party APIs on your behalf, including:
          </p>
          <ul className="mt-2 list-inside list-disc space-y-1">
            <li>Anthropic (Claude API) — for AI content generation</li>
            <li>Google (Gemini API) — for AI content generation</li>
            <li>X (Twitter) API — for posting and fetching content</li>
            <li>LinkedIn API — for posting and fetching content</li>
            <li>TikTok API — for video publishing</li>
            <li>YouTube API — for video publishing</li>
          </ul>
          <p className="mt-2">
            Each third-party service has its own privacy policy. We recommend reviewing their
            policies to understand how they handle your data.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">5. Data Sharing</h3>
          <p>
            We do not sell, trade, or share your personal information with third parties. Data is
            only sent to third-party APIs when you explicitly initiate an action (e.g., publishing
            a post or fetching trending topics).
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">6. Data Retention</h3>
          <p>
            Your data remains on your local machine for as long as you choose to keep it. You can
            delete your data at any time by removing the GoViral configuration directory
            (~/.goviral/) and the local database.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">7. Security</h3>
          <p>
            API keys and credentials are stored in your local configuration file. We recommend
            securing your machine and configuration files with appropriate filesystem permissions.
            The Service does not transmit your credentials to any server other than the intended
            API endpoints.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">8. Changes to This Policy</h3>
          <p>
            We may update this Privacy Policy from time to time. Changes will be reflected on this
            page with an updated revision date.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">9. Contact</h3>
          <p>
            For questions about this Privacy Policy, please open an issue on our GitHub repository.
          </p>
        </section>
      </div>
    </div>
  )
}
