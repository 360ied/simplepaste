package util

import (
	"os"
	"strconv"
)

func EnvDefault(key, default_ string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	} else {
		return default_
	}
}

func EnvDefaultInt64(key string, default_ int64) int64 {
	if val, ok := os.LookupEnv(key); ok {
		ret, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			panic("error decoding environment variable " + key + ": " + err.Error())
		}
		return ret
	} else {
		return default_
	}
}
