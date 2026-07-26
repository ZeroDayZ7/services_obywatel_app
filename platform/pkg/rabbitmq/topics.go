package rabbitmq

const (
	// Topics
	TopicAuditLogCreated = "audit.log.created"
	TopicUserAuthSuccess = "auth.user.login.success"
	TopicUserAuthFailed  = "auth.user.login.failed"

	// Queues
	QueueAuditProcessor = "audit.queue.processor"
)