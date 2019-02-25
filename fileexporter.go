package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
)

type FileExporter struct {
	dir     string
	counter uint64
}

func NewFileExporter(dir string) *FileExporter {
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0770)
	} else if err != nil {
		panic(err)
	} else if !stat.IsDir() {
		panic("specified directory is a file or something: " + dir)
	}

	return &FileExporter{
		dir: dir,
	}
}

func (e *FileExporter) Export(in []datadogSpan) {
	for _, datadogSpan := range in {
		e.exportDatadogSpan(datadogSpan)
	}
}

func (e *FileExporter) Flush() {

}

func (e *FileExporter) exportDatadogSpan(in datadogSpan) {
	counter := atomic.AddUint64(&e.counter, 1)

	name := path.Join(e.dir, fmt.Sprintf("%04d-%s", counter, in.Name))

	b, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(name, b, 0660)
}
