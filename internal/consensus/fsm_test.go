package consensus

import (
	"encoding/json"
	"testing"

	"distributed-cache-service/internal/core/service"
	"distributed-cache-service/internal/store"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

func TestFSM_Apply(t *testing.T) {
	memStore := store.New()
	fsm := NewFSM(memStore)

	// Test Set Command
	cmdSet := service.Command{
		Op:    service.SetOp,
		Key:   "key1",
		Value: "val1",
		TTL:   0,
	}
	data, _ := json.Marshal(cmdSet)
	logEntry := &raft.Log{Data: data}

	fsm.Apply(logEntry)

	val, found := memStore.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "val1", val)

	// Test Delete Command
	cmdDel := service.Command{
		Op:  service.DeleteOp,
		Key: "key1",
	}
	dataDel, _ := json.Marshal(cmdDel)
	logEntryDel := &raft.Log{Data: dataDel}

	fsm.Apply(logEntryDel)

	_, found = memStore.Get("key1")
	assert.False(t, found)
}
