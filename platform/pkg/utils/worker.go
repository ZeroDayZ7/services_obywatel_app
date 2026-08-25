package utils

import (
	"runtime/debug"

	"github.com/zerodayz7/platform/pkg/shared"
)

// SafeGo uruchamia funkcję w goroutine z mechanizmem recover
//#region SafeGo
func SafeGo(log *shared.Logger, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from panic in goroutine", map[string]any{
					"panic": r,
					"stack": string(debug.Stack()),
				})
			}
		}()
		fn()
	}()
}
