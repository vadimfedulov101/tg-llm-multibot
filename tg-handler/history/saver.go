package history

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"

	"tg-handler/logging"
)

// Saver errors
var (
	errWriteFailed   = errors.New("failed to write file")
	errMarshalFailed = errors.New("failed to marshal file")
)

// Saves history on update signal
func (history *History) Saver(
	ctx context.Context,
	path string,
	updateCh <-chan any,
	logger *logging.Logger,
) {
	// Save history on signal until channel CLOSED or context DONE
	// Before exit make try to save the changes
	defer logger.Info("saver shut down gracefully")
	defer history.Save(path, logger)
	for {
		select {
		case _, ok := <-updateCh:
			if !ok {
				logger.Error("history update channel was closed")
				return
			}
			history.Save(path, logger)
		case <-ctx.Done():
			logger.Error("saver received shutdown signal")
			return
		}
	}
}

// Saves history
func (h *History) Save(
	path string, logger *logging.Logger,
) {
	// Set error message
	const errMsg = "failed to save history"

	// Ensure secure access
	h.lock()
	defer h.unlock()

	// Convert to Proto struct
	protoRoot := h.toProto()

	// Marshal to binary
	data, err := proto.Marshal(protoRoot)
	if err != nil {
		logger.Error(
			errMsg, logging.Err(
				fmt.Errorf("%w: %v", errMarshalFailed, err),
			),
		)
		return
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		logger.Error(
			errMsg, logging.Err(
				fmt.Errorf("%w: %v", errWriteFailed, err),
			),
		)
		return
	}

	logger.Info("history written")
}
