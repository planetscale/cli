package history

import (
	"time"

	"github.com/planetscale/cli/internal/connections"
)

type Capture struct {
	At   time.Time                  `json:"at"`
	List connections.ConnectionList `json:"capture"`
}

func NewCapture(list connections.ConnectionList) Capture {
	return Capture{
		At:   list.CapturedAt,
		List: list,
	}
}
