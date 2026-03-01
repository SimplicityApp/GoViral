package prompts

import "fmt"

// Platform represents a social media platform.
type Platform string

const (
	PlatformX        Platform = "x"
	PlatformLinkedIn Platform = "linkedin"
)

// GeneratePrompt returns the system prompt for content generation.
func GeneratePrompt(platform Platform, isRepost bool) string {
	switch platform {
	case PlatformLinkedIn:
		if isRepost {
			return SystemPromptRepostLinkedIn
		}
		return SystemPromptRewriteLinkedIn
	default: // x
		if isRepost {
			return SystemPromptRepostX
		}
		return SystemPromptRewriteX
	}
}

// ImageDecisionPrompt returns the prompt for deciding whether to include an image.
func ImageDecisionPrompt(platform Platform) string {
	switch platform {
	case PlatformLinkedIn:
		return imageDecisionLinkedIn
	default:
		return imageDecisionX
	}
}

// ImageGenerationPrompt returns the prompt for generating a Gemini image prompt.
func ImageGenerationPrompt(platform Platform) string {
	switch platform {
	case PlatformLinkedIn:
		return imageGenerationLinkedIn
	default:
		return imageGenerationX
	}
}

// ClassifyPrompt returns the prompt for classifying posts as rewrite vs repost.
func ClassifyPrompt() string {
	return classifyPrompt
}

// CompetePrompt returns the system prompt for ranking generated content by viral potential.
func CompetePrompt() string {
	return competePrompt
}

// PersonaPrompt returns the platform-specific persona analysis prompt.
func PersonaPrompt(platform Platform) string {
	switch platform {
	case PlatformLinkedIn:
		return personaLinkedIn
	default:
		return personaX
	}
}

// ToneDirective returns the tone blending instructions for the given context.
func ToneDirective(platform Platform, isRepost bool) string {
	var naturalPct, personaPct, originalPct, platformPct int
	switch {
	case !isRepost && platform == PlatformX:
		naturalPct, personaPct, originalPct, platformPct = 30, 40, 10, 20
	case !isRepost && platform == PlatformLinkedIn:
		naturalPct, personaPct, originalPct, platformPct = 25, 35, 15, 25
	case isRepost && platform == PlatformX:
		naturalPct, personaPct, originalPct, platformPct = 25, 25, 30, 20
	case isRepost && platform == PlatformLinkedIn:
		naturalPct, personaPct, originalPct, platformPct = 20, 25, 30, 25
	}

	return fmt.Sprintf(`## Tone Blending

Blend these 4 tone sources at the following weights:

1. NATURAL HUMAN (%d%%): Sound like a real person, not a copywriter or AI. Specifically:
   - NO "Here's the thing...", "Let that sink in", "Read that again", "This.", "Period."
   - NO excessive emoji strings or motivational-poster language
   - NO corporate buzzwords ("leverage", "synergy", "thought leader", "game-changer")
   - NO "I'm not crying, you're crying" type cliches
   - Imperfect grammar is OK when it sounds natural
   - Use contractions ("don't" over "do not", "it's" over "it is")
   - Vary sentence length — mix short punchy with longer ones
   - Write like you're texting a smart friend, not drafting a press release

2. USER PERSONA (%d%%): Match the specific writing patterns, vocabulary, quirks, and voice from the persona profile provided. This is the user's authentic voice — lean into their distinctive habits.

3. ORIGINAL POST TONE (%d%%): Mirror the energy, register, and emotional pitch of the trending post being adapted. If it's sarcastic, lean sarcastic. If it's earnest, lean earnest. Don't flatten the original's vibe.

4. PLATFORM CONVENTIONS (%d%%): Follow the norms and expectations of the target platform in terms of length, formatting, and style.`,
		naturalPct, personaPct, originalPct, platformPct)
}
