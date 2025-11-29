package conf

// Bot config.
// It serves as part of api.Settings, so the load function and config are
// suited for: {"bot_conf": ..., "options": {}}
type BotConf struct {
	SystemPrompt string            `json:"system_prompt"`
	CmdPrompts   map[string]string `json:"cmd_prompts"`
	CandidateNum int               `json:"candidate_num"`
}

// Formats system prompt with chat title or panics
func MustValidateSystemPrompt(systemPrompt string) {
	mustValidateNumOfS(systemPrompt, 1, "bot")
}
