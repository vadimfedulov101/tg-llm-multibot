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

// Loads settings or panics
func MustLoadBotConf(
	path string,
	defaults *OptionalSettings,
	logger *logging.Logger,
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

	// Validate token limit
	botConf.Optional = *mergeOptions(&botConf.Optional, defaults)

	// Validate candidate number or panic
	mustValidateCandidateNum(&botConf, logger)

	return &botConf
}

// Helper to merge options (Bot overrides Default)
func mergeOptions(bot, def *OptionalSettings) *OptionalSettings {
	if bot.Temperature == 0 {
		bot.Temperature = def.Temperature
	}
	if bot.RepeatPenalty == 0 {
		bot.RepeatPenalty = def.RepeatPenalty
	}
	if bot.TopP == 0 {
		bot.TopP = def.TopP
	}
	if bot.TopK == 0 {
		bot.TopK = def.TopK
	}
	if bot.NumPredict == 0 {
		bot.NumPredict = def.NumPredict
	}
	if bot.Seed == 0 {
		bot.Seed = def.Seed
	}
	return bot
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
