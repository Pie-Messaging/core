package routing

import (
	"database/sql/driver"
	"encoding/json"
)

type Addr []string

func (a *Addr) Scan(value any) error {
	b := value.([]byte)
	err := json.Unmarshal(b, a)
	if err != nil {
		return err
	}
	return nil
}

func (a Addr) Value() (driver.Value, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return b, nil
}
