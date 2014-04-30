package TnT

import (
  "fmt"
  "io/ioutil"
  "log"
  "os"
  "strconv"
  "testing"
  "time"
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

func printfile(nservers int, fname string) {
  for i:=0; i<nservers; i++ {
    path := "../roots/root"+ strconv.Itoa(i) + "/" + fname
    data, err := ioutil.ReadFile(path)
    if err != nil {
        log.Println("Error opening file:", err)
    } else {
        fmt.Println(path, ":", string(data))
    }
  }
}

func setup(tag string, nservers int, fname string) ([]*TnTServer, func()) {

  var peers []string = make([]string, nservers)
  var tnts []*TnTServer = make([]*TnTServer, nservers)

  for i:=0; i<nservers; i++ {
    peers[i] = port(tag, i)
  }

  for i:=0; i<nservers; i++ {
    tnts[i] = StartServer(peers, i, "../roots/root"+strconv.Itoa(i)+"/", fname)
  }

  clean := func() { (cleanup(tnts)) }
  return tnts, clean
}

/*
func TestGetFile(t *testing.T) {
  tnts, clean := setup("getfile", 2)
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

func TestSync(t *testing.T) {

  const nservers = 3
  const fname = "foo.txt"
  tnts, clean := setup("sync", nservers, fname)
  defer clean()

  fmt.Println("Test: Single File Syncing ...")

  fmt.Println("Enter -1 to quit the loop")
  a := 100
  b := 100
  for a >= 0 && b >= 0 {

      fmt.Printf("Sync? Enter (who) and (from): ")
      n, err := fmt.Scanf("%d\n", &a)
      if err != nil {
          fmt.Println("Scanf error:", n, err)
      } else {
          fmt.Println("a, b:", a, b)
      }

      if 0 <= a && a < nservers && 0 <= b && b < nservers && a != b {
          tnts[a].SyncNow(b)
      }

      time.Sleep(time.Second)
  }
}
