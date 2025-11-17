package port

// Publisher defines the interface for publishing parsed messages
type Publisher interface {
	// Publish publishes a parsed message
	Publish(message interface{}) error
}

