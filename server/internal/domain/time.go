package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// UnixMillisTime оборачивает time.Time для сериализации и десериализации в виде миллисекунд эпохи Unix.
type UnixMillisTime struct {
	time.Time
}

// NewUnixMillisTime создаёт новый экземпляр UnixMillisTime, оборачивающий переданное время.
func NewUnixMillisTime(t time.Time) UnixMillisTime {
	return UnixMillisTime{Time: t}
}

// MarshalJSON сериализует время в формат Unix timestamp в миллисекундах.
func (t UnixMillisTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UnixMilli())
}

// UnmarshalJSON десериализует Unix timestamp в миллисекундах в time.Time.
func (t *UnixMillisTime) UnmarshalJSON(data []byte) error {
	var millis int64
	if err := json.Unmarshal(data, &millis); err != nil {
		return fmt.Errorf("failed to parse unix millis timestamp: %w", err)
	}
	t.Time = time.UnixMilli(millis)
	return nil
}
