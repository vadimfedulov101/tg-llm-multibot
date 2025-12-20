package memory

import (
	"strings"

	"tg-handler/conf"
	"tg-handler/history"
)

// messaging.MessageInfo abstraction
type LineChain interface {
	lineProvider
	prevLineProvider
}

type lineProvider interface {
	Line() string
}

type prevLineProvider interface {
	PrevLine() string
}

// Types are in `memory/types.go`
type Memory = struct {
	ChatQueueLines  ChatQueueLines           // Last message lines
	ReplyChainLines ReplyChainLines          // Previous reply lines
	BotContacts     *history.SafeBotContacts // User carmas/personas
	Limits          *conf.MemoryLimits       // Limits metadata
}

// Constructs memory from safe chat history and safe bot contacts
// with memory limits.
func New(
	sch *history.SafeChatHistory,
	sbc *history.SafeBotContacts,
	lc LineChain,
	lims *conf.MemoryLimits,
) *Memory {
	// Get chat queue and reply chains
	chatQueue, replyChains := sch.Unpack()

	// Get memory limits
	var (
		chatQueueLim  = lims.ChatQueue
		replyChainLim = lims.ReplyChain
	)

	// Return memory via pointer
	return &Memory{
		ChatQueueLines: chatQueue.GetLines(chatQueueLim),
		ReplyChainLines: replyChains.GetLines(
			lc.PrevLine(), lc.Line(), replyChainLim,
		),
		BotContacts: sbc,
		Limits:      lims,
	}
}

// Memory
type (
	ChatQueueLines  []string
	ReplyChainLines []string
)

// Chat queue lines as string
func (cq ChatQueueLines) String() string {
	var sb strings.Builder
	for _, line := range cq {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Reply chain lines as string
func (rc ReplyChainLines) String() string {
	var sb strings.Builder
	for _, line := range rc {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}
