package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type UnixMillisTime struct {
	time.Time
}

func NewUnixMillisTime(t time.Time) UnixMillisTime {
	return UnixMillisTime{Time: t}
}

func (t UnixMillisTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UnixMilli())
}

func (t *UnixMillisTime) UnmarshalJSON(data []byte) error {
	var millis int64
	if err := json.Unmarshal(data, &millis); err != nil {
		return fmt.Errorf("failed to parse unix millis timestamp: %w", err)
	}
	t.Time = time.UnixMilli(millis)
	return nil
}
