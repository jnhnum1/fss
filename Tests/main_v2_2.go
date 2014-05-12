package main

import (
  //"bufio"
  "fmt"
  //"log"
  "os"
  "strconv"
  "TnT_v2_2"
  //"path/filepath"
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

func cleanup(tnts []*TnT_v2_2.TnTServer) {
  for i:=0; i < len(tnts); i++ {
    tnts[i].Kill()
  }
}

func SyncAll(nservers int, tnts []*TnT_v2_2.TnTServer){

    for i := 0; i<nservers; i++ {
        for j := 0; i<nservers; i++ {
            tnts[i].SyncWrapper(j,"./")
        }
    }

}

func setup(tag string, nservers int) ([]*TnT_v2_2.TnTServer, func()) {

	var peers []string = make([]string, nservers)
	var tnts []*TnT_v2_2.TnTServer = make([]*TnT_v2_2.TnTServer, nservers)

	peers[1]="12"
	peers[2]="12"

	for i:=0; i<nservers; i++ {
		tnts[i] = TnT_single.StartServer(peers, i, common_root+strconv.Itoa(i)+"/", fname)

        //os.RemoveAll(common_root+strconv.Itoa(i)+"/")
        //os.Remove("../TestsDeepak/WatchLog"+strconv.Itoa(i))
        //os.Mkdir(common_root+strconv.Itoa(i)+"/", 0777)

		tnts[i]=TnT_v2_2.StartServer(peers, i, common_root+"nest"+strconv.Itoa(i)+"/", "WatchLog"+strconv.Itoa(i), common_root+"tmp"+strconv.Itoa(i)+"/", false)
	}

	clean := func() { (cleanup(tnts)) }
	return tnts, clean
}

func main() {

	const nservers = 2
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
			//print_tree(nservers)
		}

		fmt.Println("-----------------------------")
	}
}

