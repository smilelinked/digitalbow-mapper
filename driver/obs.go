package driver

import (
	"os"
)

// 获取环境变量信息
func GetEnvDefault(key, defVal string) string {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	return val
}

var Obs = map[string]string{
	"AK":   "NCZPQASHJNW2URNGB9SI",
	"SK":   "lXLZ9J1yUJYMrUBYZX2oAmzc3uvbSEIOSckpEsvN",
	"URI":  "obs.cn-east-3.myhuaweicloud.com",
	"NAME": "smilelink",
}

func GetObs() map[string]string {
	Obs["NAME"] = GetEnvDefault("BUCKET", "smilelink")
	return Obs
}
