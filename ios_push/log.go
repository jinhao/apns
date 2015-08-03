package main

import (
	"fmt"
	log "github.com/cihub/seelog"
)

func log_open() {
	//logger, err := log.LoggerFromConfigAsBytes([]byte(testConfig))
	logger, err := log.LoggerFromConfigAsFile("mgoproxy.xml")

	if err != nil {
		fmt.Printf("log_open | open err:%-v", err)
	}

	loggerErr := log.ReplaceLogger(logger)

	if loggerErr != nil {
		fmt.Println(loggerErr)
	}

	/* Usage: */
	/*	log.Trace("Test message!")
		log.Info("Hello from Seelog!")
		log.Error("Error msg!")
		log.Warn("Warn msg!")
		log.Critical("Critical msg!") */
}
