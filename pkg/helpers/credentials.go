package helpers

import (
	"errors"
	"strings"
)

func FormatConnectionString(connectionParams []string) (string, error) {
	if len(connectionParams) == 0 {
		return "", errors.New("must contain non zero amount of Connection parameters")
	}

	var sb strings.Builder
	for _, param := range connectionParams {
		sb.WriteString(param + ";")
	}

	return sb.String(), nil
}
