// watcher package provides functionality to monitor file changes.
// It can be used to watch any file for events like creation, modification, deletion, and renaming.

package watcher

import (
	"os"
	"time"
	"inspector/mylogger"
	"github.com/fsnotify/fsnotify"
)

/* 
* Waits for the specified file to appear within the given timeout duration.
* The timeoutStr should be in a format recognized by time.ParseDuration (e.g., "10s", "1m").
* Returns an error if the file does not appear within the specified timeout.
*/
 func waitUntilFind(filename string, timeoutStr string) error {
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		mylogger.MainLogger.Errorf("Invalid timeout format: %s", timeoutStr)
		return err
	}

	start := time.Now()
	for {
		time.Sleep(1 * time.Second)
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				if time.Since(start) > timeout {
					return os.ErrNotExist
				}
				continue
			}
			return err
		}
		return nil
	}
}

/* 
* Attempts to restore monitoring of the specified file after it has been lost.
* It waits for the file to reappear within the given timeout. If the file is restored,
* it re-adds the file to the watcher. If the file is not restored within the timeout,
* it logs an error and stops the watcher.
*/
func restoreFile(filename string, timeoutStr string, watcher *fsnotify.Watcher) error {
	if err := waitUntilFind(filename, timeoutStr); err != nil {
		if err == os.ErrNotExist {
			mylogger.MainLogger.Errorf("File not restored within timeout, stopping watcher: %s", filename)
			return nil
		}
		return err
	}
	if err := watcher.Add(filename); err != nil {
		return err
	}
	mylogger.MainLogger.Infof("File restored: %s", filename)
	return nil
}

// handleEvent processes filesystem events and takes appropriate actions based on the event type.
// It logs creation, removal, and renaming events, and sends write events to the provided channel.
func handleEvent(event fsnotify.Event, filename string, watcher *fsnotify.Watcher, events chan<- string) error {
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// File content was modified; send this event to the channel.
		events <- "Write: " + event.Name
	case event.Op&fsnotify.Create == fsnotify.Create:
		// Log file creation.
		mylogger.MainLogger.Infof("Create: %s", event.Name)
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// Log file removal and attempt to restore the watcher.
		mylogger.MainLogger.Infof("Remove: %s", event.Name)
		return restoreFile(filename, "600s", watcher)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// Log file rename and attempt to restore the watcher.
		mylogger.MainLogger.Infof("Rename: %s", event.Name)
		return restoreFile(filename, "600s", watcher)
	}
	return nil
}

/* 
* WatchFile continuously monitors the specified file for changes.
* It sends only content-related events (write) to the provided channel.
* Other events such as create, remove, and rename are logged to the terminal.
*/
func WatchFile(filename string, events chan<- string) error {
	// Initially check if the file exists without any timeout.
	if err := waitUntilFind(filename, "0s"); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(filename); err != nil {
		return err
	}

	var lastEventTime time.Time
	var lastOp fsnotify.Op

	for {
		select {
		case event := <-watcher.Events:
			// Ignore chmod events as they do not affect the content.
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}
			// Prevent processing duplicate events within a short time frame.
			if time.Since(lastEventTime) < time.Second && event.Op == lastOp {
				continue
			}

			if err := handleEvent(event, filename, watcher, events); err != nil {
				return err
			}

			// Update the last event time and operation to avoid duplicates.
			lastEventTime = time.Now()
			lastOp = event.Op

		case err := <-watcher.Errors:
			mylogger.MainLogger.Errorf("Watcher error: %v", err)
			return err
		}
	}
}
