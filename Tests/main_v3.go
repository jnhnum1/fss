package main

import (
  "fmt"
  "os"
  "strconv"
  "TnT_v3"
  "path/filepath"
)

const (
  common_root = "../roots/"
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

func cleanup(tnts []*TnT_v3.TnTServer) {
  for i:=0; i < len(tnts); i++ {
    tnts[i].Kill()
  }
}

func spaces(depth int) {
	for i:=0; i<depth; i++ {
		fmt.Printf(" |")
	}
	fmt.Printf(" |---- ")
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
            fmt.Println(fi.Name()+string(filepath.Separator), ":", fi.ModTime())
            DFT(dirname+fi.Name()+string(filepath.Separator), depth+1)
        }
    }
}

func print_tree(nservers int) {
	fmt.Println("main : printing tree")
    DFT("../roots/", 0)
	/*for i:=0; i<nservers; i++ {
		path := common_root + strconv.Itoa(i) + "/" 
		DFT(path,0)
	}
    */
}

func SyncAll(nservers int, tnts []*TnT_v3.TnTServer){

    for i := 0; i<nservers; i++ {
        for j := 0; i<nservers; i++ {
            tnts[i].SyncWrapper(j,"./")
        }
    }

}

func setup(tag string, nservers int) ([]*TnT_v3.TnTServer, func()) {

	var peers []string = make([]string, nservers)
	var tnts []*TnT_v3.TnTServer = make([]*TnT_v3.TnTServer, nservers)

	for i:=0; i<nservers; i++ {
		peers[i] = port(tag, i)
	}

	for i:=0; i<nservers; i++ {
		//tnts[i] = TnT_single.StartServer(peers, i, common_root+strconv.Itoa(i)+"/", fname)

        //os.RemoveAll(common_root+strconv.Itoa(i)+"/")
        //os.Remove("../TestsDeepak/WatchLog"+strconv.Itoa(i))
        //os.Mkdir(common_root+strconv.Itoa(i)+"/", 0777)

		tnts[i]=TnT_v3.StartServer(peers, i, common_root+"nest"+strconv.Itoa(i)+"/", "WatchLog"+strconv.Itoa(i), common_root+"tmp"+strconv.Itoa(i)+"/")
	}

	clean := func() { (cleanup(tnts)) }
	return tnts, clean
}

func main() {

	const nservers = 3
	//printfiles(nservers)

	tnts, clean := setup("sync", nservers)
	defer clean()

	fmt.Println("Test: File System Syncing ...")

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
			print_tree(nservers)
		}

		fmt.Println("-----------------------------")
	}
}

