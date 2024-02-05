package turniere

import "time"

type Turnament struct {
	Link                  string
	Title                 string
	Series                []string
	Location              string
	TurnamentDate         *time.Time
	RegistrationStartDate *time.Time
	Changed               time.Time
}
