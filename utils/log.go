package utils

import (
	"fmt"
	"log"
)

func LogDebug(message string) {
	logGen("DEBUG", message)
}

func LogInfo(message string) {
	logGen("INFO", message)
}

func LogWarn(message string) {
	logGen("WARN", message)
}

func LogError(message string) {
	logGen("ERROR", message)
}

func logGen(prefix, message string) {
	log.SetPrefix(fmt.Sprintf("[%s] ", prefix))
	log.Println(message)
}
