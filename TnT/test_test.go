package TnT

import (
  "fmt"
  "log"
  "os"
  "strconv"
  "testing"
  "encoding/gob"
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

func setup(tag string) ([]*TnTServer, func()) {

  const nservers = 2
  var peers []string = make([]string, nservers)
  var tnts []*TnTServer = make([]*TnTServer, nservers)

  for i:=0; i<nservers; i++ {
    peers[i] = port(tag, i)
  }

  for i:=0; i<nservers; i++ {
    tnts[i] = StartServer(peers, i, "../roots/root"+strconv.Itoa(i)+"/")
  }

  clean := func() { (cleanup(tnts)) }
  return tnts, clean
}

func TestGetFile(t *testing.T) {
  tnts, clean := setup("getfile")
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

func TestWatchFolder(t *testing.T) {
  root_folder := "/home/zek/fss/roots/root0"
  //dirname := "." + string(filepath.Separator) + root_folder + string(filepath.Separator)
  dirname := root_folder

  //setup the watch
  fst := new(FStree)
  fst.Tree = make(map[string]*FSnode)
  fst.Tree[dirname] = new(FSnode)
  fst.Tree[dirname].Name = root_folder
  fst.Tree[dirname].Depth = 0
  fst.Tree[dirname].Children = make(map[string]bool)

  FST_create(dirname, 0, fst)
  f, err := os.OpenFile("FST_watch", os.O_WRONLY | os.O_CREATE, 0777)
  if err != nil {
    log.Println("Error opening file:", err)
  }

  encoder := gob.NewEncoder(f)
  encoder.Encode(fst)
  f.Close()
  fmt.Println("FST_watch dumped!")

  //Test the watch here
  fmt.Println(dirname)
  f, err = os.Open("FST_watch")
  defer f.Close()
  if err != nil {
      log.Println("Error opening file:", err)
  }
  var fst1 FStree
  decoder := gob.NewDecoder(f)
  decoder.Decode(&fst1)

  watcher, err := inotify.NewWatcher()
  if err != nil {
      log.Fatal(err)
  }

  FST_parse_watch(&fst1, dirname, watcher)

  FST_watch_files(&fst1, dirname, watcher)
  for {
    /*
      select {
      case ev := <-watcher.Event:
          log.Println("event:", ev)

      case err := <-watcher.Error:
          log.Println("error:", err)
      }
      */
  }
}






