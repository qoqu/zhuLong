// 备份与恢复系统
package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Snapshot 备份快照
type Snapshot struct {
	ID        string
	CreatedAt time.Time
	Size      int64
	Path      string
	Includes  []string
}

// Manager 备份管理器
type Manager struct {
	backupDir string
}

func NewManager(dir string) *Manager {
	os.MkdirAll(dir, 0755)
	return &Manager{backupDir: dir}
}

// Create 创建备份
func (m *Manager) Create(name string, paths []string) (*Snapshot, error) {
	id := fmt.Sprintf("backup-%s-%d", name, time.Now().Unix())
	zipPath := filepath.Join(m.backupDir, id+".zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return nil, err
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	for _, p := range paths {
		baseDir := filepath.Dir(p)
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(baseDir, path)
			f, err := zw.Create(rel)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = f.Write(data)
			return err
		})
		if err != nil {
			zw.Close()
			return nil, err
		}
	}
	zw.Close()

	info, _ := os.Stat(zipPath)
	return &Snapshot{
		ID: id, CreatedAt: time.Now(),
		Size: info.Size(), Path: zipPath, Includes: paths,
	}, nil
}

// Restore 恢复备份
func (m *Manager) Restore(snapshot *Snapshot, targetDir string) error {
	r, err := zip.OpenReader(snapshot.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, _ := io.ReadAll(rc)
		os.WriteFile(target, data, 0644)
		rc.Close()
	}
	return nil
}

// List 列出备份
func (m *Manager) List() ([]Snapshot, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, err
	}
	var snapshots []Snapshot
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			info, _ := e.Info()
			snapshots = append(snapshots, Snapshot{
				ID: e.Name()[:len(e.Name())-4],
				CreatedAt: info.ModTime(),
				Size: info.Size(),
				Path: filepath.Join(m.backupDir, e.Name()),
			})
		}
	}
	return snapshots, nil
}
