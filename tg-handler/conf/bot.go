package conf

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"tg-handler/logging"

	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

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
	initConfOptional *OptionalSettings,
	logger *logging.Logger,
) *BotConf {
	var botConf BotConf

	// --- LOGGER ---
	const errMsg = "failed to load bot config"
	logger = logger.With(
		logging.ConfigType("bot"),
		logging.ConfigPath(path),
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

	// Set options
	setOptions(
		&botConf.Optional, initConfOptional, logger,
	)

	// Validate candidate number or panic
	mustValidateCandidateNum(&botConf, logger)

	return &botConf
}

// Sets bot options from non-zero bot options with precedence,
// sets from init options as a fall back
func setOptions(
	bot, init *OptionalSettings,
	logger *logging.Logger,
) {
	setOption(
		bot.Temperature, init.Temperature,
		"temperature set", logging.Temperature, logger,
	)
	setOption(
		bot.RepeatPenalty, init.RepeatPenalty,
		"repeat penalty set", logging.RepeatPenalty, logger,
	)
	setOption(
		bot.TopP, init.TopP,
		"top P set", logging.TopP, logger,
	)
	setOption(
		bot.TopK, init.TopK,
		"top K set", logging.TopK, logger,
	)
	setOption(
		bot.NumPredict, init.NumPredict,
		"num predict set", logging.NumPredict, logger,
	)
	setOption(
		bot.Seed, init.Seed,
		"seed set", logging.Seed, logger,
	)
}

// Sets bot option from non-zero bot option with precedence,
// sets from init option as a fall back
func setOption[T Number](
	botOption, initOption T,
	msgStr string,
	newAttr func(p T) slog.Attr,
	logger *logging.Logger,
) {
	if botOption == 0 {
		// Set default option from init config (fallback)
		botOption = initOption
		logger = logger.With(
			logging.OptionSource("init config"),
		)
	} else {
		// Log custom option from bot config (already set)
		logger = logger.With(
			logging.OptionSource("bot config"),
		)
	}

	logger.Info(msgStr, newAttr(botOption))
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
