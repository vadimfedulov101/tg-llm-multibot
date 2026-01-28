package conf

import (
	"encoding/json"
	"fmt"
	"os"

	"tg-handler/logging"
)

// Bot config
type BotConf struct {
	Main     MainSettings     `json:"bot_conf"`
	Optional OptionalSettings `json:"options"`
}

// Main settings for LLM
type MainSettings struct {
	Role         string `json:"role"`
	CandidateNum int    `json:"candidate_num"`
}

// Optional settings for LLM
type OptionalSettings struct {
	Temperature   float32 `json:"temperature,omitempty"`
	RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
	TopP          float32 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	Seed          int     `json:"seed,omitempty"`
}

// Loads settings or panics
func MustLoadBotConf(
	path string, logger *logging.Logger,
) *BotConf {
	var botConf BotConf

	// --- LOGGER ---
	const errMsg = "failed to load bot config"
	logger = logger.With(
		logging.ConfigType("bot"),
		logging.Path(path),
	)
	// --- LOGGER ---

	// Read JSON data from file
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Panic(errMsg,
			logging.Err(
				fmt.Errorf("%v: %v", errReadFailed, err),
			),
		)
	}

	// Decode JSON data to settings
	err = json.Unmarshal(data, &botConf)
	if err != nil {
		logger.Panic(errMsg,
			logging.Err(
				fmt.Errorf("%v: %v", errUnmarshalFailed, err),
			),
		)
	}

	// Validate candidate number or panic
	mustValidateCandidateNum(&botConf, logger)

	// Validate token limit
	setTokenLimit(&botConf, logger)

	return &botConf
}

// Validates candidate num or panics
func mustValidateCandidateNum(
	conf *BotConf, logger *logging.Logger,
) {
	const errMsg = "failed to load bot config"
	if conf.Main.CandidateNum < 0 {
		logger.Panic(errMsg, logging.Err(errNegCandidateNum))
	}
}

// TOKEN LIMIT LOGIC
func setTokenLimit(conf *BotConf, logger *logging.Logger) {
	// 0 means "use model default" (usually infinite or -1).
	// Force limit to fit in Telegram message (approx 4096 chars).
	if conf.Optional.NumPredict == 0 {
		conf.Optional.NumPredict = 600
		logger.Info(
			"defaulted num_predict to 600 (for Telegram)",
		)
	}

}
