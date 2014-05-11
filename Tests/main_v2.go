package main

import (
  	//"bufio"
  	"fmt"
  	//"log"
  	"os"
  	"strconv"
  	"TnT_v2"
  	"path/filepath"
    "math/rand"
    "time"
    "strings"
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


func parent(path string) string {
    /*
    Gives the path of the parent. For example,
    (1) "./root/nest/tra/foo" will gives "./root/nest/tra/"
    (2) "./root/nest/tra/foo/" also gives "./root/nest/tra/"
    (3) If input does not contain a "/", then it will return ""
    */
    if len(path) == 0 {
        return path
    }
    end := len(path) - 1
    if path[end] == filepath.Separator {
        end--
    }
    for ; end >= 0; end-- {
        if path[end] == filepath.Separator {
            break
        }
    }
    return path[0:end+1]
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

func EditDirectory(num_actions int, tnt *TnT_v2.TnTServer, root string){
    fmt.Println("Edit Directory ...")

    //rand.Seed(42)
    action_list := [] string{
        "Create_Dir",
        "Delete_Dir",
        "Create_File",
        "Delete_File",
        "Modify_File",
        "Move_Up",
        "Move_Down",
    }
    cur_dir := root

    for i := 0; i<num_actions; i++ {
        this_action := action_list[rand.Intn(len(action_list))]
        // this_action := "Move_Down"
        
        key_path := "./"+strings.TrimPrefix(cur_dir,root)
        fmt.Println(cur_dir, key_path)

        if this_action == "Create_Dir" {
            fmt.Println("Creating Directory")
            os.Mkdir(cur_dir+"/"+strconv.Itoa(i)+"/", 0777)

        } else if this_action == "Delete_Dir" {      

            d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(root + new_file_name)

                if file.IsDir() {
                    os.RemoveAll(root + new_file_name)
                    fmt.Println("Deleting Directory", root + new_file_name)
                    break
                }
            }

        } else if this_action == "Create_File" {
            fmt.Println("Creating File")
            os.Create(cur_dir+strconv.Itoa(i)+".txt")

        } else if this_action == "Delete_File" {

             d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(root + new_file_name)

                if !file.IsDir() {
                    os.RemoveAll(root + new_file_name)
                    fmt.Println("Deleting File ", root + new_file_name)
                    break
                }
            }

        } else if this_action == "Modify_File" {
            fmt.Println("Modifying File")

        } else if this_action == "Move_Up" {
            fmt.Println("Moving Up")
            if cur_dir != root {
                cur_dir = parent(cur_dir) + "/"
            }
            
        } else if this_action == "Move_Down" {

            d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(cur_dir + new_file_name + "/")
                fmt.Println(file,cur_dir,cur_dir + new_file_name + "/" )
                if file.IsDir() {
                    cur_dir = root + new_file_name
                    fmt.Println("Moving Down to ", cur_dir)
                    break
                }
            }
            
        }
        time.Sleep(100 * time.Millisecond)
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

    EditDirectory(30,tnts[0], common_root+strconv.Itoa(0)+"/")

    // for {

    // }
    // for i:=0;i<1;i++{
    //     go EditDirectory(5,tnts[i])
    // }

    

}

