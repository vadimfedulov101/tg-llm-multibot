package logging

import (
	"log/slog"
	"time"
)

// --- ERROR ---

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

// --- HISTORY ---

func BotName(name string) slog.Attr {
	return slog.String("bot_name", name)
}

func ChatID(id int64) slog.Attr {
	return slog.Int64("chat_id", id)
}

func UserName(name string) slog.Attr {
	return slog.String("user_name", name)
}

func LastLine(line string) slog.Attr {
	return slog.String("last_line", line)
}

func PrevLine(line string) slog.Attr {
	return slog.String("prev_line", line)
}

// --- METHODS ---

func ChatQueueLen(n int) slog.Attr {
	return slog.Int("chat_queue_len", n)
}

func ReplyChainLen(n int) slog.Attr {
	return slog.Int("reply_chain_len", n)
}

// --- SECRET / MODEL  ---

func ApiKey(key string) slog.Attr {
	return slog.String("api_key", key)
}

func EnvVar(s string) slog.Attr {
	return slog.String("env_var", s)
}

func Path(path string) slog.Attr {
	return slog.String("path", path)
}

// --- CONFIG ---

func ConfigType(t string) slog.Attr {
	return slog.String("config_type", t)
}

func TemplateType(t string) slog.Attr {
	return slog.String("template_type", t)
}

func Placeholder(p string) slog.Attr {
	return slog.String("placeholder", p)
}

func PlaceholderNeed(pn int) slog.Attr {
	return slog.Int("placeholder_need", pn)
}

func PlaceholderCount(pc int) slog.Attr {
	return slog.Int("placeholder_count", pc)
}

// --- MODEL ---

func Iter(n int) slog.Attr {
	return slog.Int("iter", n)
}

func Duration(d time.Duration) slog.Attr {
	return slog.String("duration", d.String())
}

func Candidate(s string) slog.Attr {
	return slog.String("candidate", s)
}

func Tags(s string) slog.Attr {
	return slog.String("tags", s)
}

func CarmaUpdate(s string) slog.Attr {
	return slog.String("carma_update", s)
}

// --- OLLAMA ---

func RawResponse(s string) slog.Attr {
	return slog.String("raw_response", s)
}

// --- MESSAGING ---

func Signal(s string) slog.Attr {
	return slog.String("signal", s)
}
