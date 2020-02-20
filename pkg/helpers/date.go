package helpers

import "time"

const ISOLayout = "2006-01-02"

func ParseISODate(dateStr string) (time.Time, error) {
	date, err := time.Parse(ISOLayout, dateStr)

	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}
