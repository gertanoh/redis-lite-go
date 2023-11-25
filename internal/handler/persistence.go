package handler

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ger/redis-lite-go/internal/resp"
)

// Implement persistence for the redis lite server
// using Append only file.
// All SET commands are written in the file, which sync is forced every 1s.
// At startup, the file is read and applied to the in-memory data structure

type Aof struct {
	file *os.File
	mu   sync.Mutex
}

func NewAof() (*Aof, error) {

	// aof file is located in /var/data/redis-lite/database.aof
	dbDir := "data/redis-lite"
	dbFile := "database.aof"

	err := os.MkdirAll(dbDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(dbDir, dbFile), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	aof := &Aof{
		file: f,
	}

	// Replay commands and apply to database
	aof.Read()
	// Go routine to fsync every 1 s
	go func() {
		for {
			aof.mu.Lock()
			aof.file.Sync()
			aof.mu.Unlock()

			time.Sleep(time.Second)
		}
	}()

	return aof, nil
}

func (a *Aof) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}

func (a *Aof) Read() {
	a.mu.Lock()
	defer a.mu.Unlock()

	respReader := resp.NewRespReader(a.file)
	for {
		cmd, err := respReader.Read()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			return
		}
		request, params := resp.ParseRequest(&cmd)
		updateInMemoryStore(request, params)
	}

}

func (a *Aof) Write(p *resp.Payload) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, err := a.file.Write(p.Write())
	if err != nil {
		return err
	}
	return nil
}
