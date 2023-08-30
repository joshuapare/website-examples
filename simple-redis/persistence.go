package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"sync"
	"time"
)

type PersistenceLog struct {
	// The command used for the action
	Command string
	// The arguments used for the action
	Arguments []string
	// Timestamp of the operation
	Timestamp time.Time
}

type PersistenceEngine struct {
	// The path for a top level data directory
	DataDir string
	// The path to the append-only file directory
	AOFDir string
	// The path to the snapshot directory
	SnapshotDir string
	// Interval at which to run a snapshot
	SnapshotInterval time.Duration
	// The time since the last database snapshot
	LastSnapshotTime time.Time
	// The time since the last log to the AOF
	LastLogFlushTime time.Time
	// Channel accepting AOF log messages
	LogChannel chan PersistenceLog
	// Signal whether to stop ingesting logs temporarily (like during a snapshot)
	LogLock sync.Mutex
	// The time interval between log writes to flush to storage
	LogFlushInterval time.Duration
	// Pointer to the in-memory data store
	StoreRef *Store
	// Buffered writer to use for persisting logs to disk
	AOFBufferedWriter *bufio.Writer
	// Encoder for writing binary files to the log
	AOFEncoder *gob.Encoder
	// Underlying writer for persisting logs
	AOFFile *os.File
}

// NewPersistenceEngine returns a new persistence engine
func NewPersistenceEngine(store *Store) *PersistenceEngine {
	fmt.Println("Starting persistence engine")
	return &PersistenceEngine{
		DataDir:          "_data",
		AOFDir:           "aof",
		SnapshotDir:      "snapshot",
		LastLogFlushTime: time.Now(),
		SnapshotInterval: time.Minute * 1,
		LogFlushInterval: time.Second * 5,
		LogChannel:       make(chan PersistenceLog), // Give the log channel a buffer to make non-blocking
		StoreRef:         store,
	}
}

// Start starts the persistence engine
func (p *PersistenceEngine) Start() {
	p.ensureDirsExist()

	// Ingest logs from the channel on a background thread
	go func() {
		for {
			// Check if we need to take a snapshot
			if p.IsPendingSnapshot() {
				p.PerformSnapshot()
			}
			// Check for incoming messages
			toLog := <-p.LogChannel
			if err := p.ProcessLog(&toLog); err != nil {
				fmt.Println(err)
			}
		}
	}()
}

func makeDirsIfNotExists(dirs []string) error {
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				fmt.Println(err)
				return err
			}
		}
	}

	return nil
}

// ensureDirsExist ensures that the required directories exist on the filesystem
func (p *PersistenceEngine) ensureDirsExist() error {
	// Ensure our data directories exists
	dirs := []string{
		p.DataDir,
		fmt.Sprintf("%s/%s", p.DataDir, p.SnapshotDir),
		fmt.Sprintf("%s/%s", p.DataDir, p.AOFDir),
	}

	if err := makeDirsIfNotExists(dirs); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// Log sends a message through the log to the channel
func (p *PersistenceEngine) Log(log *PersistenceLog) error {
	p.LogChannel <- *log
	return nil
}

// Log to a file
func (p *PersistenceEngine) ProcessLog(log *PersistenceLog) error {
	if log.Timestamp == (time.Time{}) {
		log.Timestamp = time.Now()
	}

	// Log to our AOF
	p.AOFEncoder.Encode(log)

	// Check if we need to flush to disk
	flushAfterTimeIs := p.LastLogFlushTime.Add(p.LogFlushInterval)
	if flushAfterTimeIs.Before(time.Now()) {
		err := p.AOFBufferedWriter.Flush()
		if err != nil {
			fmt.Println("error flushing to disk:", err)
		}
		p.LastLogFlushTime = time.Now()
	}

	return nil
}

// rotateLogFile rotate the AOF log file to the newest snapshot time
func (p *PersistenceEngine) RotateLog() error {
	// Flush any buffers, if they exist, and close
	if p.AOFFile != nil && p.AOFBufferedWriter != nil && p.AOFEncoder != nil {
		// Flush and close
		if err := p.AOFBufferedWriter.Flush(); err != nil {
			fmt.Println("failed to flush AOF log", err)
		}
		if err := p.AOFFile.Close(); err != nil {
			fmt.Println("failed to close file", err)
		}
		p.AOFEncoder = nil
	}

	// Create the buffered writer and encoder
	newFileName := fmt.Sprintf("%s/%s/%d", p.DataDir, p.AOFDir, p.LastSnapshotTime.UnixMilli())
	file, err := os.Create(newFileName)
	if err != nil {
		return err
	}
	bufferedWriter := bufio.NewWriter(file)
	enc := gob.NewEncoder(bufferedWriter)

	// Make sure our engine is using the new writer
	p.AOFFile = file
	p.AOFBufferedWriter = bufferedWriter
	p.AOFEncoder = enc

	return nil
}

// IsPendingSnapshot returns back whether a snapshot is pending
func (p *PersistenceEngine) IsPendingSnapshot() bool {
	if p.LastSnapshotTime == (time.Time{}) {
		fmt.Println("startup snapshot is pending")
		return true
	}
	// Check if we need to take a database snapshot
	snapshotAfterTimeIs := p.LastSnapshotTime.Add(p.SnapshotInterval)
	isPending := snapshotAfterTimeIs.Before(time.Now())
	if isPending {
		fmt.Println("snapshot is pending")
	}
	return isPending
}

// PerformSnapshot Performs a snapshot to the disk
func (p *PersistenceEngine) PerformSnapshot() error {
	fmt.Println("performing snapshot")
	// Create a buffer to hold the gob-encoded data
	var gobBuffer bytes.Buffer

	newSnapshotTime := time.Now()

	// Create a new gob encoder and encode the map into the buffer
	enc := gob.NewEncoder(&gobBuffer)
	if err := enc.Encode(*p.StoreRef); err != nil {
		fmt.Println("error encoding store:", err)
	}

	// Create a file to write the compressed data
	snapshotName := fmt.Sprintf("%s/%s/%d.gob.gz", p.DataDir, p.SnapshotDir, newSnapshotTime.UnixMilli())
	file, err := os.Create(snapshotName)
	if err != nil {
		fmt.Println("error creating snapshot:", err)
	}
	defer file.Close()

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Write the gob-encoded data to the gzip writer
	if _, err := gzipWriter.Write(gobBuffer.Bytes()); err != nil {
		fmt.Println("error writing compressed data to snapshot:", err)
	}

	// Flush the gzip writer to ensure all data is written
	if err := gzipWriter.Flush(); err != nil {
		fmt.Println("error flushing snapshot to disk:", err)
	}

	fmt.Printf("successfully wrote snapshot %d to disk\n", newSnapshotTime.UnixMilli())

	// Successful snapshot! Rotate the AOF log and restart AOF ingestion from new starting point
	p.LastSnapshotTime = newSnapshotTime
	p.RotateLog()

	return nil
}
