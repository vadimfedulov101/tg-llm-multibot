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

// Placeholder numbers for templates
const (
	responseSNum = 4
	responseDNum = 0

	selectSNum = 3
	selectDNum = 1

	tagsSNum = 7
	tagsDNum = 1

	carmaSNum = 6
	carmaDNum = 0
)

// Initialization config errors
var (
	errIReadFailed          = errors.New("[conf] read init config failed")
	errIUnmarshalFailed     = errors.New("[conf] unmarshal init config failed")
	errIEmptyTemplate       = errors.New("[conf] empty template")
	errIWrongPlaceholderNum = errors.New("[conf] wrong placeholder number")
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
	Tags     string `json:"tags"`
	Carma    string `json:"carma"`
}

// Memory limits
type MemoryLimits struct {
	ChatQueue  int `json:"chat_queue"`
	ReplyChain int `json:"reply_chain"`
	Tags       int `json:"tags"`
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
		log.Panicf(
			"%v (%s): %v", errIReadFailed, confPath, err,
		)
	}

	// Decode JSON data to InitConf
	err = json.Unmarshal(data, &initConf)
	if err != nil {
		log.Panicf(
			"%v (%s): %v", errIUnmarshalFailed, confPath, err,
		)
	}

	// Validate prompt templates or panic
	mustValidateTemplates(&initConf.BotSettings.PromptTemplates)

	return &initConf
}

// Validates prompt templates
func mustValidateTemplates(templates *PromptTemplates) {
	mustValidateResponseTemplate(templates.Response)
	mustValidateSelectTemplate(templates.Select)
	mustValidateTagsTemplate(templates.Tags)
	mustValidateCarmaTemplate(templates.Carma)
}

// Validates response template or panics
func mustValidateResponseTemplate(template string) {
	const tType = "response"
	mustValidateNumOf(template, "%s", responseSNum, tType)
	mustValidateNumOf(template, "%d", responseDNum, tType)
}

// Validates select template or panics
func mustValidateSelectTemplate(template string) {
	const tType = "select"
	mustValidateNumOf(template, "%s", selectSNum, tType)
	mustValidateNumOf(template, "%d", selectDNum, tType)
}

// Validates note template or panics
func mustValidateTagsTemplate(template string) {
	const tType = "tags"
	mustValidateNumOf(template, "%s", tagsSNum, tType)
	mustValidateNumOf(template, "%d", tagsDNum, tType)
}

// Validates all templates or panics
func mustValidateCarmaTemplate(template string) {
	const tType = "carma"
	mustValidateNumOf(template, "%s", carmaSNum, tType)
	mustValidateNumOf(template, "%d", carmaDNum, tType)
}

// Validates number of template placeholders or panic
func mustValidateNumOf(
	template string, placeholder string, n int, tType string,
) {
	// Handle empty template
	if template == "" {
		log.Panicf("%v", errIEmptyTemplate)
	}

	// Set placeholder error
	var err error = fmt.Errorf(
		"%w in %s template", errIWrongPlaceholderNum, tType,
	)

	// Check placeholder number or panic
	num := strings.Count(template, placeholder)
	if num < n {
		log.Fatalf("%v: less than %d %s", err, n, placeholder)
	}
	if num > n {
		log.Fatalf("%v: more than %d %s", err, n, placeholder)
	}
}
