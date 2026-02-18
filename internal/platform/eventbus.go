package platform

import (
	"context"
	"log/slog"
	"sync"
)

// Event represents an in-process event.
type Event struct {
	Type    string
	Payload any
}

// EventHandler processes an event.
type EventHandler func(ctx context.Context, event Event)

// EventBus is a simple in-process pub/sub bus using Go channels.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
	ch       chan Event
	logger   *slog.Logger
}

// NewEventBus creates an event bus with the given buffer size.
func NewEventBus(logger *slog.Logger, bufferSize int) *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
		ch:       make(chan Event, bufferSize),
		logger:   logger,
	}
}

// Subscribe registers a handler for an event type.
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish sends an event to the bus.
func (eb *EventBus) Publish(event Event) {
	select {
	case eb.ch <- event:
	default:
		eb.logger.Warn("event bus full, dropping event", "type", event.Type)
	}
}

// Start begins processing events. Call in a goroutine.
func (eb *EventBus) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eb.ch:
			eb.mu.RLock()
			handlers := eb.handlers[event.Type]
			eb.mu.RUnlock()

			for _, h := range handlers {
				go func(handler EventHandler) {
					defer func() {
						if r := recover(); r != nil {
							eb.logger.Error("event handler panic", "type", event.Type, "error", r)
						}
					}()
					handler(ctx, event)
				}(h)
			}
		}
	}
}
