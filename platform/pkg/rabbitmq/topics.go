// platform/pkg/rabbitmq/topics.go
package rabbitmq

const (
	// Topics (Routing Keys)
	TopicAuditLogCreated = "audit.log.created"
	TopicCitizenCreated  = "citizen.created"

	// Queues
	QueueAuditProcessor = "audit.queue.processor"
	QueueAuthCitizen    = "auth.queue.citizen_created"
	// Queue used by notification-service to receive citizen created events
	QueueNotificationCitizen = "notification.queue.citizen_created"
)
