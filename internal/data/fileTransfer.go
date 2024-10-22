package data

import (
	"time"
)

type MetaData struct {
	Modified time.Time `json:"modified"` // nullable
	Accessed time.Time `json:"accessed"` // nullable
}

type File struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	Size        int64       `json:"size"`
	FileType    string    `json:"fileType"`
	Sha256      string    `json:"sha256"`   // nullable
	Preview     string    `json:"preview"`  // nullable
	Metadata    *MetaData `json:"metadata"` // nullable
	Done        bool      `json:"-"`
	Token       string    `json:"-"`
	Destination string    `json:"-"`
}

type PreparePayload struct {
	Info  *PeerInfo        `json:"info"`
	Files map[string]*File `json:"files"`
}

type SessionInfo struct {
	SessionID string            `json:"sessionId"`
	Files     map[string]string `json:"files"`
}
