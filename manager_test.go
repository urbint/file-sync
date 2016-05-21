package fsync

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/alexflint/go-cloudfile"
	. "github.com/smartystreets/goconvey/convey"

	"os"
)

var mockFileSource = &MockRemoteFileSource{}

func TestManager(t *testing.T) {
	bootstrap()

	Convey("Manager", t, func() {
		mockFileSource.Reset()
		manager := NewManager(&BasicPlan{
			Local:       "test/tmp/to_fetch.txt",
			Remote:      "mock://tmp-file.txt",
			UpdateEvery: 3 * time.Second,
		})

		Convey("NewManager", func() {
			Convey("Looks for existing files and sets lastFetchedAt", func() {
				manager := NewManager(&BasicPlan{
					Local: "test/tmp/already_exists.txt",
				})
				So(manager.lastFetchedAt, ShouldNotResemble, time.Time{})
			})
		})

		Convey("Start", func() {
			Convey("Fetches a file if needed", func() {
				manager.Start()
				time.Sleep(50 * time.Millisecond)
				So(mockFileSource.fetchCount, ShouldEqual, 1)
			})
		})

		Convey("Open", func() {
			Convey("Fetches a file as needed", func() {
				manager := NewManager(&BasicPlan{
					Local:       "test/tmp/only_fetch_once.txt",
					Remote:      "mock://tmp-file.txt",
					UpdateEvery: 3 * time.Second,
				})
				file, err := manager.Open()
				So(err, ShouldBeNil)
				defer file.Close()

				file, err = manager.Open()
				So(err, ShouldBeNil)
				defer file.Close()

				So(mockFileSource.fetchCount, ShouldEqual, 1)
			})

		})

		Convey("Fetch", func() {
			manager := NewManager(&BasicPlan{
				Local:  "test/tmp/same.txt",
				Remote: "test/tmp/already_exists.txt",
			})

			file, err := manager.Fetch()
			So(err, ShouldBeNil)
			So(file.Name(), ShouldEqual, "test/tmp/same.txt")

			Convey("Updates the lastFetchedAt time", func() {
				So(manager.lastFetchedAt, ShouldHappenAfter, time.Time{})
			})

			Convey("Coppies from RemotePath to LocalPath", func() {
				contents, err := ioutil.ReadAll(file)
				So(err, ShouldBeNil)
				So(string(contents), ShouldEqual, "Hello world")
			})

			Convey("Does nothing if the paths are the same", func() {
				manager := NewManager(&BasicPlan{
					Local:  "test/tmp/same.txt",
					Remote: "test/tmp/same.txt",
				})

				file, err := manager.Fetch()
				So(err, ShouldBeNil)
				So(file.Name(), ShouldEqual, "test/tmp/same.txt")
			})
		})
	})
}

func bootstrap() {
	os.RemoveAll("test/tmp")
	if err := os.MkdirAll("test/tmp", 0777); err != nil {
		panic(err.Error())
	}

	if err := ioutil.WriteFile("test/tmp/already_exists.txt", []byte("Hello world"), 0644); err != nil {
		panic(err.Error())
	}

	cloudfile.Drivers["mock:"] = mockFileSource
}

type MockRemoteFileSource struct {
	fetchCount int
}

func (m *MockRemoteFileSource) Open(url string) (io.ReadCloser, error) {
	m.fetchCount++
	return ioutil.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (m *MockRemoteFileSource) ReadFile(url string) ([]byte, error) {
	panic("not implemented")
}

func (m *MockRemoteFileSource) WriteFile(url string, buf []byte) error {
	panic("not implemented")
}

func (m *MockRemoteFileSource) Reset() {
	m.fetchCount = 0
}
