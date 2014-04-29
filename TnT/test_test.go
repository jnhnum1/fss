package TnT

import (
  "fmt"
  "log"
  "os"
  "strconv"
  "testing"
  "code.google.com/p/go.exp/inotify"
)

func port(tag string, host int) string {
  s := "/var/tmp/824-"
  s += strconv.Itoa(os.Getuid()) + "/"
  os.Mkdir(s, 0777)
  s += "tnt-"
  s += strconv.Itoa(os.Getpid()) + "-"
  s += tag + "-"
  s += strconv.Itoa(host)
  return s
}

func cleanup(tnts []*TnTServer) {
  for i:=0; i < len(tnts); i++ {
    tnts[i].kill()
  }
}

func SetupServers(tag string) ([]*TnTServer, func()) {
  fmt.Println("set up servers")
  const nservers = 1
  var peers []string = make([]string, nservers)
  var tnts []*TnTServer = make([]*TnTServer, nservers)

  for i:=0; i<nservers; i++ {
    peers[i] = port(tag, i)
  }

  //Initialize each of the test servers
  for i:=0; i<nservers; i++ {
    dirname := "/home/zek/fss/roots/root"+strconv.Itoa(i)+"/"
    tnts[i] = StartServer(peers, i, dirname)
    tnts[i].FST_create(dirname, 0)

    //This will automatically write the tree to disk
    WritetoDisk(dirname, tnts[i])

  }

  clean := func() { (cleanup(tnts)) }
  return tnts, clean
}



/*
func TestGetFile(t *testing.T) {
  tnts, clean := SetupServers("getfile")
  defer clean()

  fmt.Println("Test: GetFile ...")

  //path := "a_meeting_by_the_river.mp3"
  path := "hw.txt"

  err := tnts[1].CopyFileFromPeer(0, path, path)
  if err != nil {
      log.Println("CopyFile failed:", err)
      t.Fatal("")
  }
  
  fmt.Println("  ... Passed")
}
*/

func TestWatchFolder(t *testing.T) {
  tnts, clean := SetupServers("getfile")
  defer clean()

  root_folder := "/home/zek/fss/roots/root"
  
  watcher, err := inotify.NewWatcher()
  if err != nil {
      log.Fatal(err)
  }

  test_server := 0
  dirname := root_folder + strconv.Itoa(test_server)+"/"

  tnts[test_server].FST_set_watch(dirname, watcher)
  tnts[test_server].FST_watch_files(dirname, watcher)
  for {

  }
}






