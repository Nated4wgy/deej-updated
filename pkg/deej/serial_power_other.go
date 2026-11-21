//go:build !windows

package deej

// Non-Windows clients retain the existing serial reconnect/watchdog logic.
// Windows-only power subscriptions are intentionally not compiled here.
func subscribeSystemResume(func()) (func(), error) {
	return func() {}, nil
}
