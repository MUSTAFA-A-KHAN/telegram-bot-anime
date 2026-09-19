# AI Integration Proposals for Telegram Bots

This document outlines potential AI integrations across the different bot modules to improve moderation, increase game engagement, and provide valuable content summarization. These features can be powered by the existing LLM infrastructure (`api.llm7.io`) and speech/vision APIs already present in the application.

## 1. Better Moderation (ModBot)

Currently, moderation relies heavily on exact keyword matching, regex, and static rules. AI can introduce nuanced, context-aware moderation.

*   **Toxicity & Sentiment Analysis:** Move beyond simple bad-word filters. An LLM can analyze the *intent* of a message to detect passive-aggressive behavior, bullying, harassment, and hate speech, even if users try to bypass filters with symbols (e.g., "h@te") or spacing.
*   **Context-Aware Spam Detection:** Spammers constantly change their scripts to evade regex filters (e.g., crypto scams, phishing). The AI can analyze the semantics of a message to identify spam or promotional content, even if it's uniquely worded.
*   **"Why was I tagged?" (Mod Context Summary):** When a user tags an admin (e.g., `@admin`), the bot can use an LLM to summarize the last 20-50 messages. This gives the admin immediate context (e.g., "User A and User B are arguing about politics") without them having to scroll up and read the entire conversation.
*   **Semantic Rule Enforcement:** Groups have rules like "no political discussions" or "stay on topic." An LLM can be prompted with the group's rules and evaluate if a user's message subtly violates them, issuing soft warnings before an admin needs to step in.
*   **Vision-Based NSFW Media Filtering:** Extend the existing Vision API capabilities (`WriteImage`) to automatically scan images, memes, and stickers sent in the group for NSFW, violent, or inappropriate content, instantly deleting them.

## 2. More Engaging Games (AnimeBot, GeographyBot, WordleBot, ScramyBot)

AI can transform the static game experiences (JSON-based trivia) into dynamic, endlessly replayable, and interactive events.

*   **Dynamic, Contextual Hints (Anime & Geography):** Instead of relying on a static list of hints, the AI can dynamically generate clues based on how much time has passed or how many wrong guesses have been made. The hints can get progressively easier or more specific (e.g., generating a short rhyming riddle about a country).
*   **Forgiving "AI Judge" for Answers:** Currently, users must guess the exact string (or a very close match) to win. An AI semantic matcher could evaluate guesses and award points for conceptually correct answers (e.g., accepting "US" or "United States" for "USA", or accepting localized anime names that aren't in the exact database).
*   **Endless AI-Generated Puzzles (Scramy & WordGrid):** Instead of a fixed word list, users could request custom puzzle themes (e.g., `/scramy theme Harry Potter`). The AI would instantly generate a list of relevant words on the fly and feed them into the game engine.
*   **Interactive Character Roleplay Game Mode:** A completely new game mode where the AI takes on the persona of a famous historical figure, celebrity, or anime character. Users can ask the bot questions (e.g., "What is your favorite weapon?"), the bot responds entirely in character, and users compete to guess who the bot is pretending to be.
*   **Adaptive Difficulty:** If the bot detects that a chat hasn't solved a Geography or Anime puzzle in the last 15 minutes, the AI can intervene, playfully mock the chat for being slow, and provide a massive "giveaway" clue to reset the game state and keep the chat engaged.

## 3. Content Summarization (Main Chat, InstagramBot, TranslatorBot)

In busy Telegram groups, users often lose track of conversations or don't have time to consume long-form media. AI summarization can act as the ultimate group assistant.

*   **`/tldr` Catch-up Summaries:** A highly requested feature for busy groups. If a user wakes up to 300 unread messages, they can type `/tldr`. The bot fetches the recent chat history and provides a concise, bulleted summary (e.g., "- Alice and Bob discussed the new movie. - Charlie shared a link about Go programming. - The group agreed to play a game later.").
*   **Instagram Reel & Video Summaries (InstagramBot):** When a user drops an Instagram Reel or TikTok link in the chat, not everyone can watch it (e.g., they are at work). The bot could extract the audio/transcript and provide a quick 1-2 sentence summary of what happens in the video (e.g., "Summary: A 30-second recipe for making easy homemade pizza.").
*   **Long-Form Article Summarization:** If a user sends a long URL (like a news article, blog post, or Wikipedia page), the bot can automatically scrape the text (using a web scraper) and drop a 3-bullet-point summary in the chat so people can discuss it without having to click away from Telegram.
*   **Voice Note "Too Long; Didn't Listen":** When a user sends a 3-minute voice note, the bot currently transcribes it. It could be enhanced to *also* append a one-sentence summary of the voice note.

## Recommended Next Steps

1.  **Prioritize:** Select one or two of these features that provide the highest immediate value. (e.g., The `/tldr` chat summary or the AI judge for games).
2.  **Implementation:** Create a specialized LLM client function in `translator.go` (similar to `GetHintForGeography`) specifically tuned with a system prompt for the chosen feature.
3.  **State Management:** For features like `/tldr` or ModBot context, implement a rolling ring-buffer in memory (e.g., storing the last 100 messages per chat ID) so the data is instantly available to feed to the LLM without excessive database reads.
