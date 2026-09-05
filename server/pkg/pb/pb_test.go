package dbsentinelv1_test

import (
	"testing"
	"time"

	dbsentinelv1 "DBSentinel/pkg/pb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNodeStatusSerialization(t *testing.T) {
	original := &dbsentinelv1.NodeStatus{
		NodeId:        "node-master-1",
		Role:          dbsentinelv1.NodeStatus_NODE_ROLE_MASTER,
		LagSeconds:    durationpb.New(0 * time.Second),
		UptimeSeconds: timestamppb.New(time.Now()),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal NodeStatus: %v", err)
	}

	restored := &dbsentinelv1.NodeStatus{}
	if err := proto.Unmarshal(data, restored); err != nil {
		t.Fatalf("failed to unmarshal NodeStatus: %v", err)
	}

	if restored.NodeId != original.NodeId || restored.Role != original.Role {
		t.Errorf("mismatch after serialization: got %v, want %v", restored, original)
	}
}
