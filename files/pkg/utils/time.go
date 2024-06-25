package utils

import (
	"strings"
	"time"
)

func TodayBegin() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func TodayEnd() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond)
}

func CurrentDateTime() string {
	return time.Now().Format(time.RFC3339)
}

var zeroTimeStr = `":"` + time.Time{}.Format(time.RFC3339) + `"`

func ClearZeroTime(data []byte) []byte {
	dataStr := string(data)
	for {
		index := strings.Index(dataStr, zeroTimeStr)
		if index == -1 {
			break
		}
		startIndex := strings.LastIndex(dataStr[:index], `"`)
		endIndex := index + len(zeroTimeStr)
		if dataStr[startIndex-1] == ',' {
			startIndex--
		} else if dataStr[endIndex] == ',' {
			endIndex++
		}
		dataStr = dataStr[:startIndex] + dataStr[endIndex:]
	}
	return []byte(dataStr)
}
