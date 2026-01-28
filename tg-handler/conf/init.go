package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"tg-handler/logging"
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
func MustLoadInitConf(
	path string,
	logger *logging.Logger,
) *InitConf {
	var initConf InitConf

	// --- LOGGER ---
	const errMsg = "failed to load init config"
	logger = logger.With(
		logging.ConfigType("init"),
		logging.Path(path),
	)
	// --- LOGGER ---

	// Read JSON data from file
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Panic(
			errMsg,
			logging.Err(
				fmt.Errorf("%w: %v", errReadFailed, err),
			),
		)
	}

	// Decode JSON data to InitConf
	err = json.Unmarshal(data, &initConf)
	if err != nil {
		logger.Panic(
			errMsg,
			logging.Err(
				fmt.Errorf("%w: %v", errUnmarshalFailed, err),
			),
		)
	}

	// Validate prompt templates or panic
	mustValidateTemplates(
		&initConf.BotSettings.PromptTemplates,
		logger,
	)

	return &initConf
}

// Validates prompt templates
func mustValidateTemplates(
	templates *PromptTemplates, logger *logging.Logger,
) {
	mustValidateResponseTemplate(templates.Response, logger)
	mustValidateSelectTemplate(templates.Select, logger)
	mustValidateTagsTemplate(templates.Tags, logger)
	mustValidateCarmaTemplate(templates.Carma, logger)
}

// Validates response template or panics
func mustValidateResponseTemplate(
	template string, logger *logging.Logger,
) {
	logger = logger.With(logging.TemplateType("response"))

	mustValidateNumOf(template, "%s", responseSNum, logger)
	mustValidateNumOf(template, "%d", responseDNum, logger)
}

// Validates select template or panics
func mustValidateSelectTemplate(
	template string, logger *logging.Logger,
) {
	logger = logger.With(logging.TemplateType("select"))

	mustValidateNumOf(template, "%s", selectSNum, logger)
	mustValidateNumOf(template, "%d", selectDNum, logger)
}

// Validates note template or panics
func mustValidateTagsTemplate(
	template string, logger *logging.Logger,
) {
	logger = logger.With(logging.TemplateType("tags"))

	mustValidateNumOf(template, "%s", tagsSNum, logger)
	mustValidateNumOf(template, "%d", tagsDNum, logger)
}

// Validates all templates or panics
func mustValidateCarmaTemplate(
	template string,
	logger *logging.Logger,
) {
	logger = logger.With(logging.TemplateType("carma"))

	mustValidateNumOf(template, "%s", carmaSNum, logger)
	mustValidateNumOf(template, "%d", carmaDNum, logger)
}

// Validates number of template placeholders or panic
func mustValidateNumOf(
	template string,
	placeholder string,
	placeholderNeed int,
	logger *logging.Logger,
) {
	// --- LOGGER ---
	const errMsg = "failed to validate placeholder number"
	logger = logger.With(
		logging.Placeholder(placeholder),
		logging.PlaceholderNeed(placeholderNeed),
	)
	// --- LOGGER ---

	// Handle empty template
	if template == "" {
		logger.Panic(errMsg, logging.Err(errEmptyTemplate))
	}

	// Check placeholder number or panic
	placeholderCount := strings.Count(template, placeholder)
	logger = logger.With(logging.PlaceholderCount(placeholderCount))
	if placeholderCount < placeholderNeed {
		logger.Panic(
			errMsg,
			logging.Err(
				fmt.Errorf(
					"%w: %v",
					errWrongPlaceholderNum, errPlaceholderOverflow,
				),
			),
		)
	}
	if placeholderCount > placeholderNeed {
		logger.Panic(
			errMsg,
			logging.Err(
				fmt.Errorf(
					"%w: %v",
					errWrongPlaceholderNum, errPlaceholderUnderflow,
				),
			),
		)
	}
}
