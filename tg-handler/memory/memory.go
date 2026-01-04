package memory

import (
	"fmt"
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

type Memory struct {
	ChatQueueLines  ChatQueueLines           // Last messages
	ReplyChainLines ReplyChainLines          // Previous messages
	BotContacts     *history.SafeBotContacts // Users known
	Limits          *conf.MemoryLimits       // Limits as metadata
}

// Constructs memory from chat history and limits,
// also keeping safe bot contacts for reading and modifying.
func New(
	ch *history.ChatHistory,
	sbc *history.SafeBotContacts,
	lc LineChain,
	lims *conf.MemoryLimits,
) *Memory {
	// Get memory data
	var (
		chatQueue   = ch.ChatQueue
		replyChains = ch.ReplyChains
	)

	// Get memory limits
	var (
		chatQueueLim  = lims.ChatQueue
		replyChainLim = lims.ReplyChain
	)

	return &Memory{
		BotContacts:    sbc,
		ChatQueueLines: chatQueue.GetLines(chatQueueLim),
		ReplyChainLines: replyChains.GetLines(
			lc.PrevLine(), lc.Line(), replyChainLim,
		),
		Limits: lims,
	}
}

func (m *Memory) String() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.BotContacts, m.ChatQueueLines, m.ReplyChainLines,
	)
}

// Memory types
type (
	ChatQueueLines  []string
	ReplyChainLines []string
)

func (cqls ChatQueueLines) String() string {
	var sb strings.Builder

	// Describe and present chat queue
	sb.WriteString("Chat Queue (last messages):\n")
	sb.WriteString(strings.Join(cqls, "\n"))

	return sb.String()
}

func (rcls ReplyChainLines) String() string {
	var sb strings.Builder

	// Describe and present reply chain
	sb.WriteString("Reply Chain (previous messages):\n")
	sb.WriteString(strings.Join(rcls, "\n"))

	return sb.String()
}
