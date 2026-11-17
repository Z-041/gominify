// Package history 负责压缩历史的本地 JSON 持久化。
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxEntries 是保留的历史条数。
const maxEntries = 100

// Entry 是一条导出记录。只存统计信息，不存文件内容。
type Entry struct {
	Time          string `json:"time"`
	FileCount     int    `json:"fileCount"`
	Failed        int    `json:"failed"`
	OriginalTotal int    `json:"originalTotal"`
	MinifiedTotal int    `json:"minifiedTotal"`
	DestDir       string `json:"destDir"`
	JSMangle      bool   `json:"jsMangle"`
	KeepCopyright bool   `json:"keepCopyright"`
	CSSPrecision  int    `json:"cssPrecision"`
}

type fileData struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Store 是并发的历史记录存储，写入使用临时文件 + rename 原子替换。
type Store struct {
	mu      sync.Mutex
	path    string
	entries []Entry
}

// Open 打开用户配置目录下的历史文件。读取失败（损坏）会自动备份为 .bad 并重新开始。
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return openAt(filepath.Join(dir, "GoMinify"))
}

// openAt 便于测试指定目录。
func openAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "history.json")}

	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	var d fileData
	if err := json.Unmarshal(b, &d); err != nil {
		_ = os.Rename(s.path, s.path+".bad")
		return s, nil
	}
	s.entries = d.Entries
	return s, nil
}

// List 返回历史副本，最新在前。
func (s *Store) List() []Entry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// LastDir 返回最近一次导出目录，没有记录时返回空串。
func (s *Store) LastDir() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) == 0 {
		return ""
	}
	return s.entries[0].DestDir
}

// Add 插入一条记录并持久化。
func (s *Store) Add(e Entry) error {
	if s == nil {
		return nil
	}
	if e.Time == "" {
		e.Time = time.Now().Format(time.RFC3339)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append([]Entry{e}, s.entries...)
	if len(s.entries) > maxEntries {
		s.entries = s.entries[:maxEntries]
	}
	return s.persistLocked()
}

// Clear 清空全部历史。
func (s *Store) Clear() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	return s.persistLocked()
}

func (s *Store) persistLocked() error {
	b, err := json.MarshalIndent(fileData{Version: 1, Entries: s.entries}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.writeAtomic(b); err != nil {
		// Windows 下 rename 可能被短暂占用，重试一次。
		time.Sleep(50 * time.Millisecond)
		return s.writeAtomic(b)
	}
	return nil
}

func (s *Store) writeAtomic(b []byte) error {
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("写入历史失败: %w", err)
	}
	return nil
}
