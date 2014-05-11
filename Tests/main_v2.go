package main

import (
  	//"bufio"
  	"fmt"
  	//"log"
  	"os"
  	"strconv"
  	"TnT_v2"
  	"path/filepath"
    //"reflect"
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

        os.RemoveAll(common_root+strconv.Itoa(i)+"/")
        os.Remove("../Tests/WatchLog"+strconv.Itoa(i))
        os.Mkdir(common_root+strconv.Itoa(i)+"/", 0777)
  	    tnts[i]=TnT_v2.StartServer(peers,i, common_root+strconv.Itoa(i)+"/", "WatchLog"+strconv.Itoa(i))

  	    fmt.Println("Initialize Watcher on ", strconv.Itoa(i))

  	    go tnts[i].FST_watch_files(common_root+strconv.Itoa(i)+"/")

  	}

  	clean := func() { (cleanup(tnts)) }
  	return tnts, clean
}

func SyncAll(nservers int, tnts []*TnT_v2.TnTServer){

    for i := 0; i<nservers; i++ {
        for j := 0; i<nservers; i++ {
            tnts[i].SyncWrapper(j,"./")
        }
    }

}

func main() {

  	const nservers = 3


  	//printfiles(nservers)

  	tnts, clean := setup("sync", nservers)
  	fmt.Println(tnts)
  	defer clean()
  
  	// fmt.Println("Test: Single File Syncing ...")

  	
  	// fmt.Println("Enter -1 to quit the loop")
  	// a := 100
  	// b := 100
  	// for a >= 0 && b >= 0 {

   //    	fmt.Printf("Sync? Enter (who) and (from): ")
   //    	n, err := fmt.Scanf("%d %d\n", &a, &b)
   //    	if err != nil {
   //        	fmt.Println("Scanf error:", n, err)
   //    	}

   //    	if 0 <= a && a < nservers && 0 <= b && b < nservers && a != b {
   //        	tnts[a].SyncWrapper(b,"./")
   //      	//printfiles(nservers)
   //    	}

   //  	fmt.Println("-----------------------------")
  	// }
  	

  	
  	var test_count int = 0
  	fmt.Println("Test: Sync File ...")	

  	//Create file on nest0
  	file_name := common_root+strconv.Itoa(0)+"/"+strconv.Itoa(test_count)+".txt"
  	os.Create(file_name)

  	//Sync all servers
	SyncAll(nservers, tnts)

  	//Check that file is in nest1, Open throws error is file does not exist
  	_,err := os.Open(common_root+strconv.Itoa(1)+"/"+strconv.Itoa(test_count)+".txt")
  	if err != nil {
  		fmt.Println("File Transfer Failed")
  		os.Exit(1)
  	}


  	fmt.Println("Test: Sync Folder ...")
  	test_count++
  	//Create folder on nest0
  	folder_name := common_root+strconv.Itoa(0)+"/"+strconv.Itoa(test_count)+"/"
  	os.Mkdir(folder_name, 0777)

    SyncAll(nservers, tnts)

  	_,err = os.Open(common_root+strconv.Itoa(2)+"/"+strconv.Itoa(test_count)+"/")
  	if err != nil {
  		fmt.Println("Folder Transfer Failed")
  		os.Exit(1)
  	}

    fmt.Println("Test: Randomly Create Directories and Files ...")

    action_list := [] string {
        "Create_Dir",
        "Delete_Dir",
        "Create_File",
        "Delete_File",
        "Modify_File",
    }

    for i := 0; i<3 i++ {
        this_action := action_list[rand.Intn(len(action_list))]
        fmt.Println(this_action)
        if this_action == "Create_Dir" {
            fmt.Println("Creating Directory")


        } else if this_action == "Delete_Dir" {
            fmt.Println("Deleting Directory")

        } else if this_action == "Create_File" {
            fmt.Println("Creating File")

        } else if this_action == "Delete_File" {
            fmt.Println("Deleting File")

        } else if this_action == "Modify_File" {
            fmt.Println("Modifying File")
        }
    }

}

