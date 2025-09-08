package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// parseChatID normalizes various types (number or string) into int64
func ParseChatID(v interface{}) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case int:
		return int64(t), nil
	case int64:
		return t, nil
	case string:
		if t == "" {
			return 0, errors.New("empty chat id")
		}
		i, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			f, err2 := strconv.ParseFloat(t, 64)
			if err2 == nil {
				return int64(f), nil
			}
			return 0, err
		}
		return i, nil
	case nil:
		return 0, errors.New("nil chat id")
	default:
		b, _ := json.Marshal(v)
		var f float64
		if err := json.Unmarshal(b, &f); err == nil {
			return int64(f), nil
		}
		return 0, fmt.Errorf("unsupported chat id type %T", v)
	}
}

func ParseToInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		if t == "" {
			return 0
		}
		if i, err := strconv.ParseInt(t, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return int64(f)
		}
		return 0
	default:
		return 0
	}
}
