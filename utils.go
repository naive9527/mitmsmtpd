package main

import (
	"fmt"
	"hash/fnv"
	"os"
	"time"
)

func SaveMail(data []byte) error {
	timestamp := time.Now().Format("20060102_150405")
	subjectHash := hashSubject(data)
	filename := fmt.Sprintf("%s_%d.eml", timestamp, subjectHash)
	return os.WriteFile(filename, data, 0644)
}

func hashSubject(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}
