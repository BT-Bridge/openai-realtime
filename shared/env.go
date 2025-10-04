package shared

import (
	"fmt"
	"os"
	"strconv"
)

func getenv_(key string, required bool, defaultValue []string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		if required {
			return "", fmt.Errorf("environment variable %s is required", key)
		}
		if len(defaultValue) > 0 {
			value = defaultValue[0]
		}
	}
	return value, nil
}

type EnvGetter[T any] func(key string, required bool, defaultValue ...string) (T, error)

func MustGetenv[T any](getter EnvGetter[T], key string, required bool, defaultValue ...string) T {
	value, err := Getenv(getter, key, required, defaultValue...)
	if err != nil {
		panic(err)
	}
	return value
}

func Getenv[T any](getter EnvGetter[T], key string, required bool, defaultValue ...string) (T, error) {
	value, err := getter(key, required, defaultValue...)
	if err != nil {
		return value, fmt.Errorf("failed to get env: %w", err)
	}
	return value, nil
}

func getenv[T any](key string, required bool, defaultValue []string, parse func(string) (T, error)) (T, error) {
	strValue, err := getenv_(key, required, defaultValue)
	if err != nil {
		return *new(T), err
	}
	return parse(strValue)
}

func GetenvString(key string, required bool, defaultValue ...string) (string, error) {
	return getenv_(key, required, defaultValue)
}

func GetenvBool(key string, required bool, defaultValue ...string) (bool, error) {
	return getenv(key, required, defaultValue, func(s string) (bool, error) {
		switch s {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("invalid boolean value: %s", s)
		}
	})
}

func GetenvInt(key string, required bool, defaultValue ...string) (int, error) {
	return getenv(key, required, defaultValue, func(s string) (int, error) {
		return strconv.Atoi(s)
	})
}

func GetenvInt32(key string, required bool, defaultValue ...string) (int32, error) {
	return getenv(key, required, defaultValue, func(s string) (int32, error) {
		v, err := strconv.ParseInt(s, 10, 32)
		return int32(v), err
	})
}

func GetenvInt64(key string, required bool, defaultValue ...string) (int64, error) {
	return getenv(key, required, defaultValue, func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
}

func GetenvUint(key string, required bool, defaultValue ...string) (uint, error) {
	return getenv(key, required, defaultValue, func(s string) (uint, error) {
		v, err := strconv.ParseUint(s, 10, 0)
		return uint(v), err
	})
}

func GetenvUint32(key string, required bool, defaultValue ...string) (uint32, error) {
	return getenv(key, required, defaultValue, func(s string) (uint32, error) {
		v, err := strconv.ParseUint(s, 10, 32)
		return uint32(v), err
	})
}

func GetenvUint64(key string, required bool, defaultValue ...string) (uint64, error) {
	return getenv(key, required, defaultValue, func(s string) (uint64, error) {
		return strconv.ParseUint(s, 10, 64)
	})
}

func GetenvFloat32(key string, required bool, defaultValue ...string) (float32, error) {
	return getenv(key, required, defaultValue, func(s string) (float32, error) {
		v, err := strconv.ParseFloat(s, 32)
		return float32(v), err
	})
}

func GetenvFloat64(key string, required bool, defaultValue ...string) (float64, error) {
	return getenv(key, required, defaultValue, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	})
}
