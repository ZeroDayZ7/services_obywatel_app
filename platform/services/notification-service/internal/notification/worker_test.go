package notification

import (
	"testing"

	"github.com/zerodayz7/platform/pkg/shared"
)

func TestNotificationWorkerStart_RespectDisabledFlag(t *testing.T) {
	worker := &NotificationWorker{
		redis:   nil,
		enabled: false,
		logger:  shared.InitLogger("development", false),
	}

	if worker.isEnabled() {
		t.Fatal("worker powinien być wyłączony przy enabled=false")
	}

	if !worker.shouldSkipRedis() {
		t.Fatal("worker powinien pomijać Redis, gdy jest wyłączony")
	}
}
