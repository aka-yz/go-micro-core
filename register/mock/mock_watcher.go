package mock

import (
	"errors"
	registry "github.com/aka-yz/go-micro-core/register"
)

type mockWatcher struct {
	exit chan bool
	opts registry.WatchOptions
}

func (m *mockWatcher) Next() (*registry.Result, error) {
	// not implement so we just block until exit
	select {
	case <-m.exit:
		return nil, errors.New("watcher stopped")
	}
}

func (m *mockWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}
