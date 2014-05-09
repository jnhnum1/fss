package main

import (
  	//"bufio"
  	"fmt"
  	//"log"
  	"os"
  	"strconv"
  	"TnT_v2"
  	"path/filepath"
)

const (
  	common_root = "../roots/nest"
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

func cleanup(tnts []*TnT_v2.TnTServer) {
  	for i:=0; i < len(tnts); i++ {
    	tnts[i].Kill()
  	}
}

/*
func readLines(path string) ([]string, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  var lines []string
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return lines, scanner.Err()
}
*/
func spaces(depth int) {
    for i:=0; i<depth; i++ {
        fmt.Printf("|")
    }
    fmt.Printf("|- ")
}


func DFT(dirname string, depth int) {
    d, err := os.Open(dirname)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer d.Close()
    fi, err := d.Readdir(-1)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    for _, fi := range fi {
        if fi.Mode().IsRegular() {
            spaces(depth)
            fmt.Println(fi.Name(), "size:", fi.Size(), "modified:", fi.ModTime())
        }
        if fi.IsDir() {
            spaces(depth)
            //fmt.Println(fi.Name(), ":")
            fmt.Println(dirname+fi.Name()+string(filepath.Separator), ":", fi.ModTime())
            DFT(dirname+fi.Name()+string(filepath.Separator), depth+1)
        }
    }
}
func printfiles(nservers int) {
  	for i:=0; i<nservers; i++ {
    	path := common_root + strconv.Itoa(i) + "/" 
    	DFT(path,0)
  	}
}

func setup(tag string, nservers int) ([]*TnT_v2.TnTServer, func()) {

  	var peers []string = make([]string, nservers)
  	var tnts []*TnT_v2.TnTServer = make([]*TnT_v2.TnTServer, nservers)

  	for i:=0; i<nservers; i++ {
    	peers[i] = port(tag, i)
  	}

  	for i:=0; i<nservers; i++ {
    //tnts[i] = TnT_single.StartServer(peers, i, common_root+strconv.Itoa(i)+"/", fname)
  	tnts[i]=TnT_v2.StartServer(peers,i, common_root+strconv.Itoa(i)+"/", "WatchLog"+strconv.Itoa(i))

  	fmt.Println("Initialize Watcher on ", strconv.Itoa(i))

  	go tnts[i].FST_watch_files(common_root+strconv.Itoa(i)+"/")

  	}

  	clean := func() { (cleanup(tnts)) }
  	return tnts, clean
}

func main() {

  	const nservers = 3


  	//printfiles(nservers)

  	tnts, clean := setup("sync", nservers)
  	defer clean()
  
  	fmt.Println("Test: Single File Syncing ...")

  	/*
  	fmt.Println("Enter -1 to quit the loop")
  	a := 100
  	b := 100
  	for a >= 0 && b >= 0 {

      	fmt.Printf("Sync? Enter (who) and (from): ")
      	n, err := fmt.Scanf("%d %d\n", &a, &b)
      	if err != nil {
          	fmt.Println("Scanf error:", n, err)
      	}

      	if 0 <= a && a < nservers && 0 <= b && b < nservers && a != b {
          	tnts[a].SyncWrapper(b,"./")
        	//printfiles(nservers)
      	}

    	fmt.Println("-----------------------------")
  	}
  	*/

  	fmt.Println("Test: Sync File ...")	

  	//Create file on nest0
  	os.Create(common_root+strconv.Itoa(0)+"/"+"1.txt")

  	//Sync with nest1
	tnts[1].SyncWrapper(0,"./")

  	//Check that file is in nest1
  	_,err := os.Open(common_root+strconv.Itoa(1)+"/"+"1.txt")
  	fmt.Println(err)
  	if err != nil {
  		fmt.Println("Transfer Failed")
  	}

}

