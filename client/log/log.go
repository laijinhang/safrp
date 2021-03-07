package log

import "github.com/sirupsen/logrus"

var log logrus.Logger

func GetLog() *logrus.Logger {
	return &log
}

func init() {
	log = *logrus.New()
	log.SetLevel(logrus.TraceLevel)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
	})
	log.SetReportCaller(true)
	log.Println(123)
	//log.SetLevel(logrus.PanicLevel)
}