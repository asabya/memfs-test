package memfs

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/winfsp/cgofuse/fuse"
)

func TestMultiDirWithFiles(t *testing.T) {
	entries := []struct {
		path    string
		isDir   bool
		size    int64
		content []byte
	}{
		{
			path:  "dir1",
			isDir: true,
		},
		{
			path:  "dir2",
			isDir: true,
		},
		{
			path:  "dir3",
			isDir: true,
		},
		{
			path: "file1",
			size: 1024 * 1024,
		},
		{
			path: "dir1/file11",
			size: 1024 * 512,
		},
		{
			path: "dir1/file12",
			size: 1024 * 1024,
		},
		{
			path: "dir3/file31",
			size: 1024 * 1024,
		},
		{
			path: "dir3/file32",
			size: 1024 * 1024,
		},
		{
			path: "dir3/file33",
			size: 1024,
		},
		{
			path:  "dir2/dir4",
			isDir: true,
		},
		{
			path:  "dir2/dir4/dir5",
			isDir: true,
		},
		{
			path: "dir2/dir4/file241",
			size: 5 * 1024 * 1024,
		},
		{
			path: "dir2/dir4/dir5/file2451",
			size: 10 * 1024 * 1024,
		},
	}

	mntDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatal(err)
	}
	memfs := NewMemfs()
	host := fuse.NewFileSystemHost(memfs)
	host.SetCapReaddirPlus(true)
	defer host.Unmount()
	go func() {
		host.Mount(mntDir, []string{})
	}()
	<-time.After(time.Second * 5)

	t.Run("create structure", func(t *testing.T) {
		for idx, v := range entries {
			if v.isDir {
				err := os.Mkdir(filepath.Join(mntDir, v.path), 0755)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				f, err := os.Create(filepath.Join(mntDir, v.path))
				if err != nil {
					t.Fatal(err)
				}
				buf := make([]byte, 1024)
				var off int64 = 0
				for off < v.size {
					_, err = rand.Read(buf)
					if err != nil {
						t.Fatal(err)
					}
					n, err := f.Write(buf)
					if err != nil {
						t.Fatal(err)
					}
					if n != 1024 {
						t.Fatalf("wrote %d bytes exp %d", n, 1024)
					}
					entries[idx].content = append(entries[idx].content, buf...)
					off += int64(n)
				}
				err = f.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	})

	verify := func(t *testing.T, mnt string) {
		t.Helper()
		for _, v := range entries {
			st, err := os.Stat(filepath.Join(mnt, v.path))
			if err != nil {
				t.Fatal(err)
			}
			if st.Mode().IsDir() != v.isDir {
				t.Fatalf("isDir expected: %t found: %t", v.isDir, st.Mode().IsDir())
			}
			if !v.isDir {
				if st.Size() != v.size {
					t.Fatalf("expected size %d found %d", v.size, st.Size())
				}
				if got, err := ioutil.ReadFile(filepath.Join(mnt, v.path)); err != nil {
					t.Fatalf("ReadFile: %v", err)
				} else if !bytes.Equal(got, v.content) {
					t.Fatalf("ReadFile %s: got %q, want %q", filepath.Join(mnt, v.path), got[:30], v.content[:30])
				}
			}
		}
	}

	t.Run("verify structure", func(t *testing.T) {
		verify(t, mntDir)
	})
}
