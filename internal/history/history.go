package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type index struct {
	Sessions []SessionSummary `json:"sessions"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".oc", "sessions")
	if err := os.MkdirAll(d, 0755); err != nil {
		return "", err
	}
	return d, nil
}

func indexFile() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "index.json"), nil
}

func readIndex() (*index, error) {
	p, err := indexFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &index{}, nil
		}
		return nil, err
	}
	var idx index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func writeIndex(idx *index) error {
	p, err := indexFile()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func sessionFile(id string) (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fmt.Sprintf("sess_%s.json", id)), nil
}

func CreateSession(id, title string) error {
	title = truncate(title, 60)
	now := time.Now()
	s := Session{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		Messages:  nil,
	}

	p, err := sessionFile(id)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return err
	}

	idx, err := readIndex()
	if err != nil {
		return err
	}
	idx.Sessions = append(idx.Sessions, SessionSummary{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		Count:     0,
	})
	return writeIndex(idx)
}



func AppendMessage(sessionID, role, content string) error {
	p, err := sessionFile(sessionID)
	if err != nil {
		return err
	}

	var s Session
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			title := content
			if role != "user" {
				title = sessionID
			}
			s = Session{ID: sessionID, Title: truncate(title, 60), CreatedAt: time.Now()}
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	}

	s.Messages = append(s.Messages, Message{Role: role, Content: content})

	data, err = json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return err
	}

	idx, err := readIndex()
	if err != nil {
		return err
	}
	found := false
	for i := range idx.Sessions {
		if idx.Sessions[i].ID == sessionID {
			idx.Sessions[i].Count = len(s.Messages)
			if role == "user" && idx.Sessions[i].Title == sessionID {
				idx.Sessions[i].Title = truncate(content, 60)
			}
			found = true
			break
		}
	}
	if !found {
		title := content
		if role != "user" {
			title = sessionID
		}
		idx.Sessions = append(idx.Sessions, SessionSummary{
			ID:        sessionID,
			Title:     truncate(title, 60),
			CreatedAt: s.CreatedAt,
			Count:     len(s.Messages),
		})
	}
	return writeIndex(idx)
}

func ListSessions() ([]SessionSummary, error) {
	idx, err := readIndex()
	if err != nil {
		return nil, err
	}
	// reverse order (newest first)
	for i, j := 0, len(idx.Sessions)-1; i < j; i, j = i+1, j-1 {
		idx.Sessions[i], idx.Sessions[j] = idx.Sessions[j], idx.Sessions[i]
	}
	return idx.Sessions, nil
}

func LoadSession(id string) (*Session, error) {
	p, err := sessionFile(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
