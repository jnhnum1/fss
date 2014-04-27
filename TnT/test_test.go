package TnT

import (
  "fmt"
  "log"
  "os"
  "strconv"
  "testing"
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
