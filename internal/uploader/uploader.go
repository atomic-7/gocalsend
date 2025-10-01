package uploader

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/sessions"
)

type Uploader struct {
	node      *data.PeerInfo
	client    *http.Client
	tlsclient *http.Client
	SessMan   *sessions.SessionManager
}

// node is the peerinfo of the local node
func CreateUploader(node *data.PeerInfo, sman *sessions.SessionManager) *Uploader {
	slog.Debug("Creating client")

	// TODO: Look into cloning the default transport
	// https://stackoverflow.com/questions/12122159/how-to-do-a-https-request-with-bad-certificate
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: time.Duration(60 * time.Second),
		},
		Timeout: time.Duration(120 * time.Second),
	}
	tlsclient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ResponseHeaderTimeout: time.Duration(60 * time.Second),
		},
		Timeout: time.Duration(120 * time.Second),
	}
	return &Uploader{
		node:      node,
		client:    client,
		tlsclient: tlsclient,
		SessMan:   sman,
	}
}

// peer is the peerinfo of the target remote, files is a list of filepaths
func (cl *Uploader) UploadFiles(peer *data.PeerInfo, files []string) error {

	sessionID, err := cl.prepareUpload(peer, files)
	if err != nil {
		// TODO: pass in a context to use?
		slog.Error("failed to prepare file upload", slog.Any("error", err))
		return err
	}

	sess := cl.SessMan.Uploads[sessionID]
	ctx := sess.GetCtx()
	for _, file := range sess.Files {
		select {
		case <-ctx.Done():
			// TODO: clean up downloaded files?
			// break is useless here but just putting something here for now
			break
		default:
			slog.Info("uploading file", slog.String("file", file.FileName))
			err = cl.singleUpload(ctx, peer, sess.SessionID, file)
			if err != nil {
				slog.Error("failed to upload", slog.String("file", file.FileName), slog.Any("error", err))
				if !errors.Is(err, context.Canceled) {
					slog.Debug("cancelling session because the error was not a context cancel")
					cl.SessMan.CancelSession(sessionID)
				}
				return err
			}
			cl.SessMan.FinishFile(sessionID, file.ID)

		}

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
			slog.Error("Failed to stat", slog.String("file", path), slog.Any("error", err))
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
	endpoint, err := url.Parse("/api/localsend/v2/prepare-upload")
	if err != nil {
		slog.Error("failed to parse endpoint string", slog.Any("error", err))
		os.Exit(1)
	}
	endpoint.Host = fmt.Sprintf("%s:%d", peer.IP, peer.Port)
	endpoint.Scheme = "http"
	if peer.Protocol == "https" {
		endpoint.Scheme = "https"
	}
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		slog.Error("Failed to marshal prep-upload data", slog.Any("error", err))
		os.Exit(1)
	}
	client := cl.client
	if peer.Protocol == "https" {
		client = cl.tlsclient
	}
	resp, err := client.Post(endpoint.String(), "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		slog.Error("error sending prepare-upload payload", slog.Any("error", err))
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
		slog.Error("failed to read prep-upload response", slog.Any("error", err))
		os.Exit(1)
	}
	var sessInfo data.SessionInfo
	err = json.Unmarshal(respBytes, &sessInfo)
	if err != nil {
		slog.Error("failed to unmarshal session info for prep-upload", slog.Any("error", err))
		return "", err
	}
	slog.Info("received session", slog.String("peer", peer.Alias), slog.Any("session", sessInfo))

	sessID := cl.SessMan.CreateUpload(peer, &sessInfo, idmap)

	return sessID, nil
}

func (cl *Uploader) singleUpload(ctx context.Context, peer *data.PeerInfo, sessID string, file *data.File) error {

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
		slog.Error("failed to open file for upload", slog.Any("error", err))
		os.Exit(1)
	}
	defer fh.Close()

	client := cl.client
	if peer.Protocol == "https" {
		base.Scheme = "https"
		client = cl.tlsclient
	}
	req, err := http.NewRequestWithContext(ctx, "POST", base.String(), fh)
	req.Header.Set("Content-Type", "application/octet-stream")
	if err != nil {
		slog.Error("failed to create request with context", slog.Any("error", err))
		return err
	}
	resp, err := client.Do(req)

	// resp, err := client.Post(base.String(), "Content-Type:application/octet-stream", fh)
	if err != nil {
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
