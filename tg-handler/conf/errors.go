package conf

import "errors"

// Bot config errors
var (
	ErrNoPrompt  = errors.New("[conf] no prompt provided")
	ErrWrongSNum = errors.New("[conf] wrong %%s num")
	ErrWrongDNum = errors.New("[conf] wrong %%d num")
)
