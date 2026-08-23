package rabbitmq

const (
	// Topics (Routing Keys)
	TopicAuditLogCreated = "audit.log.created"
	TopicCitizenCreated  = "citizen.created"

	// Queues
	QueueAuditProcessor = "audit.queue.processor"
	QueueAuthCitizen    = "auth.queue.citizen_created"
)
