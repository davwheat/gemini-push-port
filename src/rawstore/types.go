package rawstore

import "time"

type XmlMessageWithTime struct {
	MessageTime time.Time
	Message     string
}

func (x XmlMessageWithTime) GetFilePath() string {
	return getFilePathForTime(x.MessageTime)
}

func getFilePathForTime(t time.Time) string {
	return t.Format("2006/01/02/15") + ".pport"
}
