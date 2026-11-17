package history

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAddListAndLastDir(t *testing.T) {
	dir := t.TempDir()
	s, err := openAt(dir)
	if err != nil {
		t.Fatalf("openAt: %v", err)
	}
	if got := s.LastDir(); got != "" {
		t.Fatalf("空历史 LastDir = %q", got)
	}

	entries := []Entry{
		{FileCount: 1, DestDir: "D:/out1"},
		{FileCount: 2, DestDir: "D:/out2"},
		{FileCount: 3, DestDir: "D:/out3"},
	}
	for _, e := range entries {
		if err := s.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	list := s.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d，期望 3", len(list))
	}
	if list[0].DestDir != "D:/out3" || list[2].DestDir != "D:/out1" {
		t.Fatalf("顺序错误: %+v", list)
	}
	if list[0].Time == "" {
		t.Fatal("Time 应自动填充")
	}
	if got := s.LastDir(); got != "D:/out3" {
		t.Fatalf("LastDir = %q，期望 D:/out3", got)
	}

	// 重新打开确认已落盘
	s2, err := openAt(dir)
	if err != nil {
		t.Fatalf("重新打开: %v", err)
	}
	if len(s2.List()) != 3 {
		t.Fatalf("落盘后条数 = %d，期望 3", len(s2.List()))
	}
}

func TestAddTruncatesToMaxEntries(t *testing.T) {
	s, err := openAt(t.TempDir())
	if err != nil {
		t.Fatalf("openAt: %v", err)
	}
	for i := 0; i < maxEntries+5; i++ {
		if err := s.Add(Entry{FileCount: i, DestDir: "D:/out"}); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if n := len(s.List()); n != maxEntries {
		t.Fatalf("条数 = %d，期望 %d", n, maxEntries)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	s, err := openAt(dir)
	if err != nil {
		t.Fatalf("openAt: %v", err)
	}
	_ = s.Add(Entry{FileCount: 1, DestDir: "D:/out"})
	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(s.List()) != 0 {
		t.Fatal("清空后不应有记录")
	}
	s2, _ := openAt(dir)
	if len(s2.List()) != 0 {
		t.Fatal("清空应持久化")
	}
}

func TestCorruptFileRecovers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "history.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openAt(dir)
	if err != nil {
		t.Fatalf("损坏文件不应导致打开失败: %v", err)
	}
	if len(s.List()) != 0 {
		t.Fatal("损坏文件应从空历史开始")
	}
	if _, err := os.Stat(filepath.Join(dir, "history.json.bad")); err != nil {
		t.Fatalf("应备份为 .bad: %v", err)
	}
	if err := s.Add(Entry{FileCount: 1, DestDir: "D:/out"}); err != nil {
		t.Fatalf("恢复后写入失败: %v", err)
	}
}

func TestConcurrentAdd(t *testing.T) {
	dir := t.TempDir()
	s, err := openAt(dir)
	if err != nil {
		t.Fatalf("openAt: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.Add(Entry{FileCount: i, DestDir: "D:/out"}); err != nil {
				t.Errorf("Add: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if n := len(s.List()); n != 20 {
		t.Fatalf("条数 = %d，期望 20", n)
	}
	tmps, _ := filepath.Glob(filepath.Join(dir, "history-*.tmp"))
	if len(tmps) != 0 {
		t.Fatalf("存在残留临时文件: %v", tmps)
	}
}

func TestNilStoreIsSafe(t *testing.T) {
	var s *Store
	if s.LastDir() != "" || len(s.List()) != 0 {
		t.Fatal("nil Store 应返回空值")
	}
	if err := s.Add(Entry{}); err != nil {
		t.Fatalf("nil Store Add: %v", err)
	}
	if err := s.Clear(); err != nil {
		t.Fatalf("nil Store Clear: %v", err)
	}
}
