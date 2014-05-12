package TnT_final

import (
	"fmt"
	"net"
	"net/rpc"
	"log"
	"os"
	"encoding/gob"
	"time"
	//"path/filepath"
)

const (
	RPC_SLEEP_INTERVAL = 100*time.Millisecond

	DO_NOTHING = 0
	UPDATE = 1
	DELETE = 2
	SYNC_DOWN = 3

    END_OF_WORLD = 0x7fffffffffffffff
)

func (tnt *TnTServer) GetVersion(args *GetVersionArgs, reply *GetVersionReply) error {

	//fmt.Println("Syncing ", args, tnt)
	//tnt.UpdateTreeWrapper("./") //ToDo: We should be more specific?

	fst := tnt.Tree.MyTree
	fsn, present := fst[args.Path]

	if present == false {
		reply.Exists=false
		reply.SyncVect = fst[tnt.LiveAncestor(args.Path)].SyncVect
	} else {
		reply.IsDir=make(map[string]bool)
		for k,_ := range fsn.Children{
			reply.IsDir[k]=tnt.Tree.MyTree[k].IsDir
		}
		reply.Exists = true
		reply.VerVect, reply.SyncVect, reply.Children = fsn.VerVect, fsn.SyncVect, fsn.Children
		reply.Creator, reply.CreationTime = fsn.Creator, fsn.CreationTime
	}
	fmt.Println("GET VERSION:", args.Path, reply)
	return nil
}

func (tnt *TnTServer) Kill() {
	tnt.dead = true
	tnt.l.Close()
}

func StartServer(servers []string, me int, root string, dump string, tmp string, test bool) *TnTServer {
	gob.Register(GetFileArgs{})
	gob.Register(GetDirArgs{})
	tnt := new(TnTServer)
	tnt.me = me
	tnt.servers = servers
	tnt.root = root
	tnt.Test = test
	if _, err := os.Lstat(root); err != nil {
		os.Mkdir(root, 0777)
	}
	tnt.dump = dump //root+"FST_watch_new"
	tnt.tmp = tmp
	if _, err := os.Lstat(tmp); err != nil {
		os.Mkdir(tmp, 0777)
	}

	f, err := os.Open(tnt.dump)
	defer f.Close()
	if err != nil {
		fmt.Println(tnt.dump, "not found. Creating new tree...")
		fst := new(FStree)
		fst.MyTree = make(map[string]*FSnode)
		fst.MyTree["./"] = new(FSnode)
		fst.MyTree["./"].Name = "root"
		fst.MyTree["./"].IsDir=true
		fst.MyTree["./"].Children = make(map[string]bool)
		fst.MyTree["./"].LastModTime = time.Now()
		fst.MyTree["./"].Creator = 0
		fst.MyTree["./"].CreationTime = END_OF_WORLD // If a guy accidentally deletes root, it will be imported from the other!
		fst.MyTree["./"].VerVect = make(map[int]int64)
		fst.MyTree["./"].SyncVect = make(map[int]int64)
		fst.MyTree["./"].Parent = "./"

		// Initialize VecVect, SyncVect
		for i:=0; i<len(tnt.servers); i++ {
			fst.MyTree["./"].VerVect[i] = 0
			fst.MyTree["./"].SyncVect[i] = 0
		}

		fst.NewFiles = make([]NewData, 0, 64)
		fst.DelFiles = make([]DelData, 0, 64)

		tnt.Tree = fst
	} else {
		fmt.Println(tnt.dump, "found! Fetching tree...")
		var fst1 FStree
		decoder := gob.NewDecoder(f)
		decoder.Decode(&fst1)
		tnt.Tree = &fst1

		fmt.Println(fst1.DelFiles)
		fmt.Println(fst1.NewFiles)

		tnt.mu.Lock()
		tnt.Apply()
		tnt.LogToFile()
		tnt.mu.Unlock()
	}
	tnt.UpdateTreeWrapper("./")
	fmt.Println("in start server",tnt.Tree)

	go tnt.FST_watch_files(tnt.root)

	//RPC TCP
	rpc.Register(tnt)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1235")
	tnt.l=l
	go func(){
		if e != nil {
			log.Fatal("listen error:", e)
		} else {
			defer l.Close()
			fmt.Println("listening for requests...")
			http.Serve(l, nil)
		}
	}()

	return tnt
}
