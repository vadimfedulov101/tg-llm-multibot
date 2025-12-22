package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Initialization config errors
var (
	ErrIConfReadFailed = errors.New(
		"[conf] read init config failed",
	)
	ErrIConfUnmarshalFailed = errors.New(
		"[conf] unmarshal init config failed",
	)
	ErrIConfEmptyTemplate = errors.New(
		"[conf] empty template",
	)
	ErrIConfWrongPlaceholderNum = errors.New(
		"[conf] wrong placeholder number",
	)
)

// Initialization config
type InitConf struct {
	Paths           Paths           `json:"paths"`
	CleanerSettings CleanerSettings `json:"cleaner_settings"`
	BotSettings     BotSettings     `json:"bot_settings"`
}

// Paths
type Paths struct {
	History     string `json:"history"`
	BotsConfDir string `json:"bots_conf_dir"`
}

// Cleaner settings
type CleanerSettings struct {
	MessageTTL      Duration `json:"msg_ttl"`
	CleanupInterval Duration `json:"cleanup_interval"`
}

// Bot settings
type BotSettings struct {
	PromptTemplates PromptTemplates `json:"prompt_templates"`
	AllowedChats    AllowedChats    `json:"allowed_chats"`
	MemoryLimits    MemoryLimits    `json:"memory_limits"`
}

// Allowed chats
type AllowedChats struct {
	Usernames []string `json:"usernames"`
	IDs       []int64  `json:"ids"`
}

// Prompt templates
type PromptTemplates struct {
	Response string `json:"response"`
	Select   string `json:"select"`
	Note     string `json:"note"`
	Carma    string `json:"carma"`
}

// Memory limits
type MemoryLimits struct {
	ChatQueue  int `json:"chat_queue"`
	ReplyChain int `json:"reply_chain"`
	Note       int `json:"note"`
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// Loads init config or panics
func MustLoadInitConf(confPath string) *InitConf {
	var initConf InitConf

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Panicf("%v (%s): %v", ErrIConfReadFailed, confPath, err)
	}

	// Decode JSON data to InitConf
	err = json.Unmarshal(data, &initConf)
	if err != nil {
		log.Panicf("%v (%s): %v", ErrIConfUnmarshalFailed, confPath, err)
	}

	// Validate templates or panic
	mustValidateTemplates(&initConf.PromptTemplates)

	return &initConf
}

// Validates prompts
func mustValidateTemplates(templates *PromptTemplates) {
	mustValidateResponseTemplate(templates.Response)
	mustValidateSelectTemplate(templates.Select)
	mustValidateNoteTemplate(templates.Note)
	mustValidateCarmaTemplate(templates.Carma)
}

// Validates response template or panics
func mustValidateResponseTemplate(template string) {
	const tType = "response template"
	mustValidateNumOf(template, "%s", 5, tType)
}

// Validates select template or panics
func mustValidateSelectTemplate(template string) {
	const tType = "select template"
	mustValidateNumOf(template, "%s", 5, tType)
	mustValidateNumOf(template, "%d", 1, tType)
}

// Validates note template or panics
func mustValidateNoteTemplate(template string) {
	const tType = "note template"
	mustValidateNumOf(template, "%s", 10, tType)
	mustValidateNumOf(template, "%d", 2, tType)
}

// Validates all templates or panics
func mustValidateCarmaTemplate(template string) {
	const tType = "carma template"
	mustValidateNumOf(template, "%s", 8, tType)
	mustValidateNumOf(template, "%d", 1, tType)
}

// Validates that 'template' of 'tType'
// contains 's' exactly 'n' times or panics.
func mustValidateNumOf(
	template string, s string, n int, tType string,
) {
	var err error

	// Handle empty template
	if template == "" {
		log.Panicf("%v", ErrIConfEmptyTemplate)
	}

	// Count s in template
	num := strings.Count(template, s)
	// Detect errors
	if num < n {
		err = fmt.Errorf("less than %d %s in %s", n, s, tType)
	}
	if num > n {
		err = fmt.Errorf("more than %d %s in %s", n, s, tType)
	}
	// Panic on error
	if err != nil {
		log.Panicf(
			"%v: %v: \"%s\"",
			ErrIConfWrongPlaceholderNum, err, template,
		)
	}
}
