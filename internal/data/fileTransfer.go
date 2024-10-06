package data

import (
	"time"
)

type MetaData struct {
	Modified time.Time `json:"modified"` // nullable
	Accessed time.Time `json:"accessed"` // nullable
}

type File struct {
	Id       string    `json:"id"`
	FileName string    `json:"fileName"`
	Size     int       `json:"size"`
	FileType string    `json:"fileType"`
	Sha256   string    `json:"sha256"`   // nullable
	Preview  string    `json:"preview"`  // nullable
	Metadata *MetaData `json:"metadata"` // nullable
}

type PreparePayload struct {
	Info  *PeerInfo        `json:"info"`
	Files map[string]*File `json:"files"`
}

type Session struct {
	SessionId string           `json:"sessionId"`
	Files     map[string]string `json:"files"`
}
