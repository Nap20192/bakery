package enum

type ExchangeName string

const (
	ExchangeOrderEvents ExchangeName = "bakery.order-events"
)

func (e ExchangeName) String() string { return string(e) }

type ExchangeKind string

const (
	ExchangeKindFanout ExchangeKind = "fanout"
)

func (e ExchangeKind) String() string { return string(e) }

type QueueName string

const (
	// QueueBotOrderEvents is the bot's durable queue bound to the order-events
	// exchange; the bot consumes order notifications from it.
	QueueBotOrderEvents QueueName = "bakery.bot.order-events"
)

func (q QueueName) String() string { return string(q) }
