package history

import (
	"time"

	"tg-handler/carma"
	"tg-handler/history/pb"
	"tg-handler/tags"
)

// --- ADAPTERS ---

// Convert Go internal -> Proto
func (h *History) toProto() *pb.RootHistory {
	root := &pb.RootHistory{
		SharedQueues: make(map[int64]*pb.ChatQueue),
		Bots:         make(map[string]*pb.BotData),
	}

	// Snapshot Shared Queues
	for cid, scq := range h.SharedChatQueues {
		root.SharedQueues[cid] = chatQueueToProto(scq.ChatQueue)
	}

	// Snapshot Bots
	for name, botData := range h.Bots.History {
		pbBot := &pb.BotData{
			Chats:    make(map[int64]*pb.ChatHistory),
			Contacts: make(map[string]*pb.BotContact),
		}

		// Contacts
		for user, c := range botData.Contacts.Contacts {
			pbBot.Contacts[user] = &pb.BotContact{
				Carma: int32(c.Carma),
				Tags:  c.Tags.Serialize(),
			}
		}

		// Chat Histories
		for cid, ch := range botData.History.History {
			pbChat := &pb.ChatHistory{
				ReplyChains: replyChainsToProto(ch.ReplyChains.ReplyChains),
			}

			// KEY LOGIC: If shared, do not save local_queue
			if !ch.ChatQueue.IsShared {
				pbChat.LocalQueue = chatQueueToProto(ch.ChatQueue.ChatQueue)
			}

			pbBot.Chats[cid] = pbChat
		}
		root.Bots[name] = pbBot
	}

	return root
}

// Convert Proto -> Go internal
func fromProto(p *pb.RootHistory, cids []int64) *History {
	h := NewHistory(cids) // Helper to init empty maps

	// Load Shared Queues
	// Overwrite empty ones created by NewHistory or fill new
	for cid, pQueue := range p.SharedQueues {
		// Check if this CID is allowed
		if _, isAllowed := h.SharedChatQueues[cid]; !isAllowed {
			// Skip loading history for chats removed from config
			continue
		}

		scq := NewSafeChatQueue(true)
		scq.ChatQueue = protoToChatQueue(pQueue)
		h.SharedChatQueues[cid] = scq
	}

	// Load Bots
	for name, pBot := range p.Bots {
		botData := NewBotData()

		// Contacts
		for user, pCont := range pBot.Contacts {
			botData.Contacts.Contacts[user] = BotContact{
				Carma: carma.Carma(pCont.Carma),
				Tags:  tags.DeserializeTags(pCont.Tags),
			}
		}

		// Histories
		for cid, pChat := range pBot.Chats {
			// Restore Reply Chains
			replyChains := NewSafeReplyChains()
			replyChains.ReplyChains = protoToReplyChains(pChat.ReplyChains)

			// Restore Chat Queue
			var scq *SafeChatQueue

			if pChat.LocalQueue != nil {
				// Case A: It was saved as local
				scq = NewSafeChatQueue(false)
				scq.ChatQueue = protoToChatQueue(pChat.LocalQueue)
			} else {
				// Case B: It is shared, link to the SharedChatQueues
				if shared, exists := h.SharedChatQueues[cid]; exists {
					scq = shared
				} else {
					// Fallback if shared queue missing (shouldn't happen)
					scq = NewSafeChatQueue(true)
				}
			}

			botData.History.History[cid] = &ChatHistory{
				ChatQueue:   scq,
				ReplyChains: replyChains,
			}
		}
		h.Bots.History[name] = botData
	}

	return h
}

// --- HELPERS ---

func chatQueueToProto(cq ChatQueue) *pb.ChatQueue {
	pq := &pb.ChatQueue{Messages: make([]*pb.MessageEntry, len(cq))}
	for i, m := range cq {
		pq.Messages[i] = &pb.MessageEntry{
			Line:      m.Line,
			Timestamp: m.Timestamp.Unix(),
		}
	}
	return pq
}

func protoToChatQueue(pq *pb.ChatQueue) ChatQueue {
	if pq == nil {
		return make(ChatQueue, 0)
	}
	cq := make(ChatQueue, len(pq.Messages))
	for i, m := range pq.Messages {
		cq[i] = MessageEntry{
			Line:      m.Line,
			Timestamp: time.Unix(m.Timestamp, 0),
		}
	}
	return cq
}

func replyChainsToProto(rc ReplyChains) *pb.ReplyChains {
	prc := &pb.ReplyChains{Chains: make(map[string]*pb.MessageEntry)}
	for k, v := range rc {
		prc.Chains[k] = &pb.MessageEntry{
			Line:      v.Line,
			Timestamp: v.Timestamp.Unix(),
		}
	}
	return prc
}

func protoToReplyChains(prc *pb.ReplyChains) ReplyChains {
	rc := make(ReplyChains)
	if prc == nil {
		return rc
	}
	for k, v := range prc.Chains {
		rc[k] = MessageEntry{
			Line:      v.Line,
			Timestamp: time.Unix(v.Timestamp, 0),
		}
	}
	return rc
}
