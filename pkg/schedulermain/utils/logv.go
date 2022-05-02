package utils

import "log"

var V int = 0

// LogV 根据日志详细级别打印日志
func LogV(level int, content string) {
	if level <= V {
		log.Println(content)
	}
}
