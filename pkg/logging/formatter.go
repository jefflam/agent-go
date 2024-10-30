package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type ColoredJSONFormatter struct {
	// Include timestamp in the output
	TimestampFormat string
	// Customize field sorting
	SortingFunc func([]string) []string
	// Disable colors when not in terminal
	DisableColors bool
}

func NewColoredJSONFormatter() *ColoredJSONFormatter {
	return &ColoredJSONFormatter{
		TimestampFormat: time.RFC3339,
		SortingFunc:     defaultFieldSorting,
	}
}

func (f *ColoredJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields)
	for k, v := range entry.Data {
		data[k] = v
	}

	// Add standard fields
	data["level"] = entry.Level.String()
	data["msg"] = entry.Message
	data["time"] = entry.Time.Format(f.TimestampFormat)

	// Get field keys for sorting
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	if f.SortingFunc != nil {
		keys = f.SortingFunc(keys)
	} else {
		sort.Strings(keys)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Format with colors based on level
	levelColor := getLevelColor(entry.Level)
	fieldColor := color.New(color.FgCyan)
	valueColor := color.New(color.FgWhite)
	timeColor := color.New(color.FgYellow)

	// Start with timestamp
	timeStr := timeColor.Sprintf("%s", data["time"])
	b.WriteString(fmt.Sprintf("%s ", timeStr))

	// Add level with color
	levelStr := levelColor.Sprintf("%-7s", strings.ToUpper(data["level"].(string)))
	b.WriteString(fmt.Sprintf("%s ", levelStr))

	// Add message with level color
	if msg, ok := data["msg"].(string); ok {
		b.WriteString(levelColor.Sprintf("%s", msg))
	}
	b.WriteString(" ")

	// Add remaining fields
	for _, k := range keys {
		if k != "time" && k != "level" && k != "msg" {
			v := data[k]
			// Format value based on type
			var valueStr string
			switch v := v.(type) {
			case string:
				valueStr = fmt.Sprintf("%q", v)
			case error:
				valueStr = fmt.Sprintf("%q", v.Error())
			default:
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					valueStr = fmt.Sprintf("%v", v)
				} else {
					valueStr = string(jsonBytes)
				}
			}

			// Highlight important fields
			if isImportantField(k) {
				fieldColor = color.New(color.FgGreen)
			} else {
				fieldColor = color.New(color.FgCyan)
			}

			b.WriteString(fieldColor.Sprintf("%s=", k))
			b.WriteString(valueColor.Sprint(valueStr))
			b.WriteString(" ")
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func getLevelColor(level logrus.Level) *color.Color {
	switch level {
	case logrus.DebugLevel:
		return color.New(color.FgBlue)
	case logrus.InfoLevel:
		return color.New(color.FgGreen)
	case logrus.WarnLevel:
		return color.New(color.FgYellow)
	case logrus.ErrorLevel:
		return color.New(color.FgRed)
	case logrus.FatalLevel, logrus.PanicLevel:
		return color.New(color.FgRed, color.Bold)
	default:
		return color.New(color.FgWhite)
	}
}

func isImportantField(field string) bool {
	important := map[string]bool{
		"tweet_id":        true,
		"conversation_id": true,
		"author_id":       true,
		"error":           true,
	}
	return important[field]
}

func defaultFieldSorting(keys []string) []string {
	priorityFields := map[string]int{
		"time":            1,
		"level":           2,
		"msg":             3,
		"tweet_id":        4,
		"conversation_id": 5,
		"author_id":       6,
		"error":           7,
	}

	sort.Slice(keys, func(i, j int) bool {
		iPriority := priorityFields[keys[i]]
		jPriority := priorityFields[keys[j]]
		if iPriority != 0 && jPriority != 0 {
			return iPriority < jPriority
		}
		if iPriority != 0 {
			return true
		}
		if jPriority != 0 {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}
