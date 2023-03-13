package aponoapi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var nilTime = (time.Time{}).UnixNano()

type Instant struct {
	time.Time
}

func (i *Instant) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		i.Time = time.Time{}
		return
	}

	var (
		seconds int64
		nano    int64
	)
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return errors.New("illegal time format")
	}

	seconds, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return err
	}

	if len(parts) == 2 {
		nano, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return err
		}
	}

	i.Time = time.Unix(seconds, nano)
	return
}

func (i *Instant) MarshalJSON() ([]byte, error) {
	if i.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%d.%d\"", i.Time.Unix(), i.Time.Nanosecond())), nil
}

func (i *Instant) IsSet() bool {
	return i.UnixNano() != nilTime
}
