package rawstore

import (
	"fmt"
	"gemini-push-port/logging"
	"os"
	"path/filepath"
	"time"
)

func CleanUpLocalFilesJob() {
	logging.Logger.Infof("Starting local file cleanup job...")

	workDir := os.Getenv("PUSH_PORT_DUMP_WORKDIR")
	if workDir == "" {
		panic("PUSH_PORT_DUMP_WORKDIR environment variable not set")
	}

	nowTime := time.Now().UTC()
	cleanupCutoff := nowTime.Add(-48 * time.Hour)

	err := recursiveDeletionWalk(workDir, cleanupCutoff)
	if err != nil {
		logging.Logger.ErrorE("failed to clean up local files", err)
	} else {
		logging.Logger.Infof("local file cleanup job completed successfully")
	}
}

func recursiveDeletionWalk(dir string, cutoff time.Time) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of %s: %v", dir, err)
	}

	// delete files older than 48 hours by their file path, looking for files that match the pattern {workdir}/YYYY/MM/DD/HH.pport
	return filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// parse the file path to get the time
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			logging.Logger.Warnf("failed to get relative path for %s: %v", path, err)
			return nil
		}

		var year, month, day, hour int
		n, err := fmt.Sscanf(relPath, "%d/%d/%d/%d.pport", &year, &month, &day, &hour)
		if err != nil || n != 4 {
			// not a file we care about
			return nil
		}

		fileTime := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.UTC)
		if fileTime.Before(cutoff) {
			logging.Logger.Infof("deleting file %s, >48h old", path)
			// delete the file
			err := os.Remove(path)
			if err != nil {
				logging.Logger.Warnf("failed to delete file %s: %v", path, err)
			}
		}

		return nil
	})
}
