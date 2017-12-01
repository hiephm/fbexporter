package util

import (
	"time"
	"strings"
)

func ToSqlTime(fbTime string) (string, error) {
	t, err := time.Parse("2006-01-02T15:04:05-0700", fbTime)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02 03:04:05"), nil
}

func EscapeString(input string) string {
	output := input
	output = strings.Replace(output, "\\", "\\\\'", -1)
	output = strings.Replace(output, "'", "\\'", -1)
	return output
}
