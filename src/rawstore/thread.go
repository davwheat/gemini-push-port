package rawstore

import (
	"gemini-push-port/logging"
	"os"
	"path"
)

func Thread(rawMessageChan chan *XmlMessageWithTime) {
	workdir := os.Getenv("PUSH_PORT_DUMP_WORKDIR")
	if workdir == "" {
		panic("PUSH_PORT_DUMP_WORKDIR environment variable not set")
	}

	// ensure the directory exists
	err := os.MkdirAll(workdir, 0755)
	if err != nil {
		panic(err)
	}

	for {
		msg, ok := <-rawMessageChan
		if !ok {
			// Channel closed, exit the loop
			return
		}

		err := appendMessageToFile(workdir, msg)
		if err != nil {
			logging.Logger.ErrorE("failed to append message to file", err)
		}
	}
}

func appendMessageToFile(workdir string, msg *XmlMessageWithTime) error {
	filePath := path.Join(workdir, msg.GetFilePath())

	// ensure the directory exists
	err := os.MkdirAll(path.Dir(filePath), 0755)
	if err != nil {
		panic(err)
	}

	// append the message to the file
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logging.Logger.ErrorE("failed to close file", err)
		}
	}(f)
	_, err = f.WriteString(msg.Message + "\n")
	if err != nil {
		return err
	}

	return nil
}
