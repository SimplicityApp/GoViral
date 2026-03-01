export function Terms() {
  return (
    <div className="mx-auto max-w-2xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Terms of Service</h2>
      <div className="space-y-4 text-sm leading-relaxed text-[var(--color-text-secondary)]">
        <p className="text-xs text-[var(--color-text-secondary)]">Last updated: February 28, 2026</p>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">1. Acceptance of Terms</h3>
          <p>
            By accessing or using GoViral ("the Service"), you agree to be bound by these Terms of
            Service. If you do not agree to these terms, do not use the Service.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">2. Description of Service</h3>
          <p>
            GoViral is a developer content tool that helps users create, manage, and publish content
            to social media platforms including X (Twitter), LinkedIn, TikTok, and YouTube. The
            Service provides AI-powered content generation, trending topic discovery, and automated
            publishing capabilities.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">3. User Responsibilities</h3>
          <p>You agree to:</p>
          <ul className="mt-2 list-inside list-disc space-y-1">
            <li>Provide accurate information when using the Service</li>
            <li>Comply with all applicable laws and platform terms of service</li>
            <li>Not use the Service to create spam, misleading, or harmful content</li>
            <li>Maintain the security of your API keys and credentials</li>
            <li>Not attempt to reverse engineer or disrupt the Service</li>
          </ul>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">4. Third-Party Services</h3>
          <p>
            The Service integrates with third-party platforms and APIs. Your use of these
            integrations is subject to the respective third-party terms of service. We are not
            responsible for the availability, accuracy, or policies of third-party services.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">5. Content Ownership</h3>
          <p>
            You retain ownership of all content you create using the Service. By using the Service,
            you grant us a limited license to process your content solely for the purpose of
            providing the Service to you.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">6. Disclaimer of Warranties</h3>
          <p>
            The Service is provided "as is" without warranties of any kind, express or implied. We
            do not guarantee that generated content will achieve specific engagement results or
            comply with all platform guidelines.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">7. Limitation of Liability</h3>
          <p>
            To the fullest extent permitted by law, we shall not be liable for any indirect,
            incidental, special, or consequential damages arising from your use of the Service,
            including but not limited to account suspensions on third-party platforms.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">8. Changes to Terms</h3>
          <p>
            We reserve the right to modify these terms at any time. Continued use of the Service
            after changes constitutes acceptance of the updated terms.
          </p>
        </section>

        <section>
          <h3 className="mb-2 font-medium text-[var(--color-text)]">9. Contact</h3>
          <p>
            For questions about these Terms, please open an issue on our GitHub repository.
          </p>
        </section>
      </div>
    </div>
  )
}
