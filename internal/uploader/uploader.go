package uploader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/atomic-7/gocalsend/internal/data"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"github.com/atomic-7/gocalsend/internal/server"
	"time"
)

type Uploader struct {
	node   *data.PeerInfo
	client *http.Client
	sessMan *server.SessionManager
}

func CreateUploader(node *data.PeerInfo) *Uploader {
	log.Println("Creating client")
	return &Uploader{
		node: node,
		client: &http.Client{},
		sessMan: server.NewSessionManager("/home/atomc/Downloads/gocalsend"),
	}
}

func (cl *Uploader) UploadFiles(peer *data.PeerInfo, files []string) error {

	sessionID, err := cl.prepareUpload(peer, files)
	if err != nil {
		log.Fatal("Failed uploading: ", err)
	}

	sess := cl.sessMan.Sessions[sessionID]
	for _, file := range sess.Files {
		log.Printf("Uploading %s", file.FileName)
		err = cl.singleUpload(peer, sess.SessionID, file)
		if err != nil {
			log.Printf("Failed to upload %s: %v", file.FileName, err)
		}
		cl.sessMan.FinishFile(sessionID, file.ID)
	}
	return nil
}

func (cl *Uploader) genID(file string) string {
	return "ID-" + file
}

func (cl *Uploader) prepareUpload(peer *data.PeerInfo, filePaths []string) (string, error) {

	// TODO: Implement pins

	idmap := make(map[string]*data.File, len(filePaths))
	for _, path := range filePaths {
		info, err := os.Stat(path)
		if err != nil {
			log.Printf("Failed to stat %s: %v", path, err)
			return "", err
		}
		fileName := info.Name()
		fileID := cl.genID(fileName)
		idmap[fileID] = &data.File{
			ID:          cl.genID(fileName),
			FileName:    fileName,
			Size:        info.Size(),
			Destination: path,
			Metadata: &data.MetaData{
				Modified: info.ModTime(),
				Accessed: time.Now(),
			},
		}

	}
	payload := data.PreparePayload{
		Info:  cl.node,
		Files: idmap,
	}
	url := fmt.Sprintf("http://%s:%d/api/localsend/v2/prepare-upload", peer.IP, peer.Port)
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		log.Fatal("Failed to marshal prep-upload data: ", err)
	}
	resp, err := cl.client.Post(url, "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		log.Printf("Error sending prepare-upload payload: %v\n", err)
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 400:
			return "", errors.New("Invalid body")
		case 401:
			return "", errors.New("Invalid pin")
		case 403:
			return "", errors.New("Rejected")
		case 409:
			return "", errors.New("Blocked by another session")
		case 429:
			return "", errors.New("Too many requests")
		case 500:
			return "", errors.New("Server error")
		default:
			return "", errors.New("Somthing is not good. And it's you.")
		}
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read prep-upload response: ", err)
	}
	log.Println(string(respBytes))
	var sessInfo data.SessionInfo
	err = json.Unmarshal(respBytes, &sessInfo)
	if err != nil {
		log.Println("Failed to unmarshal session info for prep-upload: ", err)
		return "", err
	}
	log.Printf("Received session: %v\n", sessInfo)
	sessID := cl.sessMan.RegisterSession(&sessInfo, idmap)

	return sessID, nil
}

func (cl *Uploader) singleUpload(peer *data.PeerInfo, sessID string, file *data.File) error {

	base := url.URL{}
	base.Scheme = "http"
	base.Host = fmt.Sprintf("%s:%d", peer.IP, peer.Port)
	base.Path = "/api/localsend/v2/upload"

	params := url.Values{}
	params.Add("sessionId", sessID)
	params.Add("fileId", file.ID)
	params.Add("token", file.Token)
	base.RawQuery = params.Encode()

	fh, err := os.Open(file.Destination)
	if err != nil {
		log.Fatal("Failed to open file for upload: ", err)
	}

	resp, err := cl.client.Post(base.String(), "Content-Type:application/octet-stream", fh)
	if err != nil {
		log.Printf("Failed to send the file to the server: %v\n", err)
		return err
	}

	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 400:
			return errors.New("Missing parameters")
		case 403:
			return errors.New("Invalid token or ip address")
		case 409:
			return errors.New("Blocked by another session")
		case 500:
			return errors.New("Server error")
		default:
			return errors.New("Something is not good. And it's really you.")
		}
	}
	return nil
}
