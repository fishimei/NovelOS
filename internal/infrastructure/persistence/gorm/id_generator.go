package gormstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type IDGenerator struct{}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (g *IDGenerator) New(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		fallback := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
		if prefix == "" {
			return fallback
		}
		return prefix + "_" + fallback
	}
	suffix := fmt.Sprintf("%x%s", time.Now().UTC().UnixMilli(), hex.EncodeToString(buf))
	if prefix == "" {
		return suffix
	}
	return prefix + "_" + suffix
}
