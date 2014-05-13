package main

import (
  	"fmt"
  	"os"
  	"strconv"
  	"TnT_final"
  	"path/filepath"
    "math/rand"
    "time"
    "io/ioutil"
    "hash/fnv"
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

func cleanup(tnts []*TnT_final.TnTServer) {
  	for i:=0; i < len(tnts); i++ {
    	tnts[i].Kill()
  	}
}

func DFT(dirname string, depth int ,str string)string {
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
           str=str+fi.Name()
        }
        if fi.IsDir() {
  			str=str+fi.Name()
            str=DFT(dirname+fi.Name()+string(filepath.Separator), depth+1,str)
        }else{
		data,_:=ioutil.ReadFile(dirname+fi.Name())
        	str=str+string(data)
        }
    }
    return str
}

func hash(s string) uint32 {
  h := fnv.New32a()
  h.Write([]byte(s))
  return h.Sum32()
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

func setup(tag string, nservers int) ([]*TnT_final.TnTServer, func()) {

  	var peers []string = make([]string, nservers)
  	var tnts []*TnT_final.TnTServer = make([]*TnT_final.TnTServer, nservers)

  	for i:=0; i<nservers; i++ {
    	peers[i] = port(tag, i)
  	}

  	for i:=0; i<nservers; i++ {

        os.RemoveAll(common_root+strconv.Itoa(i)+"/")
        os.Remove("../Tests/WatchLog"+strconv.Itoa(i))
        os.Mkdir(common_root+strconv.Itoa(i)+"/", 0777)
  	    tnts[i]=TnT_final.StartServer(peers,i, common_root+strconv.Itoa(i)+"/", "WatchLog"+strconv.Itoa(i), common_root+"tmp"+strconv.Itoa(i)+"/", true)
  	}

  	clean := func() { (cleanup(tnts)) }
  	return tnts, clean
}

func SyncAll(nservers int, tnts []*TnT_final.TnTServer){

    for i := 0; i<nservers; i++ {
        if i != 0 {
          tnts[0].SyncWrapper(i,"./")
        }
    }

    for i := 0; i<nservers; i++ {
        if i != 0 {
          tnts[i].SyncWrapper(0,"./")
        }
    }

    for i := 0; i<nservers; i++ {
        if i != 0 {
            tnts[0].SyncWrapper(i,"./")
        }
    }

    // for i := 0; i<nservers; i++ {
    //     for j := 0; j<nservers; j++ {
    //         if j != i {
    //           tnts[i].SyncWrapper(j,"./")
    //         }
    //     }
    // }

}

func EditDirectory(num_actions int, nservers int, me int, root string, tnt *TnT_final.TnTServer, c chan int, stop_all chan int){
    fmt.Println("Edit Directory ...")

    rand.Seed( time.Now().UTC().UnixNano())

    action_list := [] string{
        "Create_Dir",
        "Create_File",
        "Delete_File",
        "Modify_File",
        "Move_Up",
        "Move_Down",
        "Move_Down",
    }

    cur_dir := root

    for i := 0; i<num_actions; i++ {
        my_num := rand.Intn(200)
        this_action := action_list[rand.Intn(len(action_list))]

        if this_action == "Create_Dir" {
            dir_name := cur_dir+strconv.Itoa(my_num)+"/"
            os.Mkdir(dir_name, 0777)
            fmt.Println("Creating Directory ", dir_name)

        } else if this_action == "Delete_Dir" {      

            d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(cur_dir + new_file_name )
                fmt.Println(file,cur_dir,cur_dir + new_file_name )
                if file.IsDir() {
                    os.RemoveAll(cur_dir + new_file_name + "/")
                    fmt.Println("Deleting Directory", cur_dir + new_file_name + "/")
                    break
                }
            }

        } else if this_action == "Create_File" {
            file_name := cur_dir+strconv.Itoa(my_num)+".txt"
            os.Create(file_name)
            fmt.Println("Creating File ", file_name)

        } else if this_action == "Delete_File" {

             d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(cur_dir + new_file_name)

                if !file.IsDir() {
                    os.RemoveAll(cur_dir + new_file_name)
                    fmt.Println("Deleting File ", cur_dir + new_file_name)
                    break
                }
            }

        } else if this_action == "Modify_File" {
            fmt.Println("Modifying File")
            d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(cur_dir + new_file_name)
                //fmt.Println(file,cur_dir,cur_dir + new_file_name)
                if !file.IsDir() {
                    //open_file,_ := os.OpenFile(cur_dir + new_file_name, syscall.O_APPEND,  0777)
                    wr_str := []byte(strconv.Itoa(my_num))
                    err := ioutil.WriteFile(cur_dir + new_file_name,wr_str,0777)
                    fmt.Println("Modifying File ", cur_dir + new_file_name, wr_str, err)
                    break
                }
            }

        } else if this_action == "Move_Up" {
            fmt.Println("Moving Up")
            if cur_dir != root {
                cur_dir = parent(cur_dir)
            }
            
        } else if this_action == "Move_Down" {

            d, _ := os.Open(cur_dir)
            defer d.Close()
            file_name, _ := d.Readdirnames(-1)
            for _,new_file_name := range file_name{
                file,_ := os.Lstat(cur_dir + new_file_name)
                fmt.Println(file,cur_dir,cur_dir + new_file_name)
                if file.IsDir() {
                    cur_dir = cur_dir + new_file_name + "/"
                    fmt.Println("Moving Down to ", cur_dir)
                    break
                }
            }
            
        }
        
        
        //time.Sleep(10 * time.Millisecond)
    }
    fmt.Println(me, " am done ...")
    c <- 1
}

func main() {

  	const nservers = 5


  	//printfiles(nservers)

  	tnts, clean := setup("sync", nservers)
  	fmt.Println(tnts)
  	defer clean()
  

    SyncAll(nservers, tnts)

    fmt.Println("Test: Randomly Create Directories and Files ...")
 
    var c [nservers]chan int
    for i:=0; i<nservers; i++ {
      fmt.Println("CAPITAL LADDERS ........", i)
      c[i] = make(chan int)
    }
    //c := make(chan int)
    stop_all := make(chan int)
    for j:=0;j<5;j++{
      for i:=0; i<nservers; i++ {

          go EditDirectory(50, nservers, i, common_root+strconv.Itoa(i)+"/", tnts[i],c[i], stop_all)
      }
      for i:=0;i<nservers;i++{
          <-c[i]
      }
      i:= rand.Intn(nservers)
      sync_with := rand.Intn(nservers)
      
      
      if sync_with != i {
        tnts[i].SyncWrapper(sync_with,"./")
      }
    }
    
    SyncAll(nservers, tnts)
    SyncAll(nservers, tnts)
    var h [nservers]uint32
    for i:=0;i<nservers;i++{
    	fmt.Println(hash(DFT(common_root+strconv.Itoa(i)+"/",0,"")))
      h[i]=hash(DFT(common_root+strconv.Itoa(i)+"/",0,""))
      if(h[i]!=h[0]){
        fmt.Println("Failed");
        return
      }
    }
    fmt.Println("Passed.")
}

