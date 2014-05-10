package TnT_v2_1

import (
	"fmt"
	"net"
	"net/rpc"
	"log"
	"os"
	"encoding/gob"
	"time"
	"path/filepath"
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
	tnt.UpdateTreeWrapper("./") //ToDo: We should be more specific?

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
		fmt.Println("GET VERSION:", reply, fsn)
	}
	return nil
}

func (tnt *TnTServer) PropagateUp(VersionVector map[int]int64, SyncVector map[int]int64, path string) {

	//path:path of the parent
	//Propagate the changes in a version vector upwards

	fst := tnt.Tree.MyTree // for ease of code
	fmt.Println("PROPAGATE UP: path ",path)
	tnt.ParseTree("./",0)

	setMaxVersionVect(fst[path].VerVect, VersionVector)
	setMinVersionVect(fst[path].SyncVect, SyncVector)

	if path != "./" {
		tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
	}
}

func (tnt *TnTServer) UpdateTreeWrapper(path string) {
	tnt.Tree.LogicalTime += 1
	tnt.UpdateTree(path)
}

func (tnt *TnTServer) DeleteTree(path string) {
	// Deletes entire sub-tree under 'path' from FStree

	fmt.Println("DELETE TREE:", path)
	fst := tnt.Tree.MyTree

	if _, present := fst[path]; present {
		// Delete all children; recursively delete if child is a directory
		delete(fst[fst[path].Parent].Children, path)
		for child, _ := range fst[path].Children {
			tnt.DeleteTree(child)
		}
		delete(fst, path)
	}
}

func (tnt *TnTServer) UpdateTree(path string) {

	/* Remark: 'path' can be either a file or directory.

	(1) explores the file system, and make appropriate changes in FStree
	(2) deletes nodes in FStree which are not in the file system

	Caution: Should ensure that the first time UpdateTree is called, then
	'path' should exist in 'fst', and it's 'Parent' should be set appropriately.
	Otherwise, UpdateTree will create a spurious sub-tree which will not be
	reachable from the root. It is fine if stuff under 'path' are not in FST already.
	*/

	fst := tnt.Tree.MyTree

	fi, err := os.Lstat(tnt.root + path)

	if err != nil {
		tnt.DeleteTree(path)
		return
	}

	if _, present := fst[path]; present == false {
		fst[path] = new(FSnode)
		fst[path].Name = fi.Name()
		fst[path].Size = fi.Size()
		fst[path].IsDir = fi.IsDir()
		fst[path].LastModTime = fi.ModTime()
		fst[path].Creator = tnt.me
		fst[path].CreationTime = tnt.Tree.LogicalTime

		if fi.IsDir() {
			fst[path].Children = make(map[string]bool)
		}

		fst[path].VerVect = make(map[int]int64)
		fst[path].SyncVect = make(map[int]int64)
		for i:=0; i<len(tnt.servers); i++ {
			fst[path].VerVect[i] = 0
			fst[path].SyncVect[i] = 0
		}

		fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
		// fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime // set outside - unconditionally
	}

	fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime

	if fi.IsDir() == false {
		if fst[path].LastModTime.Equal(fi.ModTime()) == false {
			fst[path].LastModTime = fi.ModTime()
			fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
		}
	} else {

		d, err := os.Open(tnt.root + path)
		defer d.Close()
		cfi, err := d.Readdir(-1)
		if err != nil {
			log.Println("LOL Error in UpdateTree:", err)
			os.Exit(1)
		}

		// Book-keeping : if fst.Tree[dir].Children[child] remains false in the end,
		// then it means child does not exist in file system
		for child, _ := range fst[path].Children {
			fst[path].Children[child] = false
		}

		for _, cfi := range cfi {
			var child string
			if cfi.IsDir() {
				child = path + cfi.Name() + string(filepath.Separator)
			} else {
				child = path + cfi.Name()
			}
			tnt.UpdateTree(child)
			fst[path].Children[child] = true
			fst[child].Parent = path

			// Update LastModTime, VerVect and SyncVect for 'dir' :

			// set my last mod time to be latest among all my children
			if fst[path].LastModTime.Before(fst[child].LastModTime) {
				fst[path].LastModTime = fst[child].LastModTime
			}

			setMaxVersionVect(fst[path].VerVect, fst[child].VerVect)  // VerVect is element-wise maximum of children's VerVect
			setMinVersionVect(fst[path].SyncVect, fst[child].SyncVect) // SyncVect is element-wise minimum of children's SyncVect
		}

		// Delete all my children, who were deleted since the last sync
		for child, exists := range fst[path].Children {
			if exists == false {
				tnt.DeleteTree(child)
				fst[path].VerVect[tnt.me]=tnt.Tree.LogicalTime
				//delete(fst[path].Children, child)
			}
		}
	}
}

func (tnt *TnTServer) SyncWrapper(srv int, path string) {
	//Update tree and then sync
	tnt.mu.Lock()
	tnt.UpdateTreeWrapper(path)
	tnt.mu.Unlock()
	tnt.SyncDir(srv, path)
	//tnt.ParseTree("./", 0)
	tnt.LogToFile()
}

func (tnt *TnTServer) OnlySync(path string) {

	fst := tnt.Tree.MyTree

	parent := fst[path].Parent
	setMaxVersionVect(fst[path].SyncVect, fst[parent].SyncVect)
	for k, _ := range fst[path].Children {
		tnt.OnlySync(k)
	}
}

func (tnt *TnTServer) SyncDir(srv int, path string) (bool, map[int]int64, map[int]int64) {

	fst := tnt.Tree.MyTree // for ease of code

	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for {
		ok:=call(tnt.servers[srv], "TnTServer.GetVersion", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	_, exists := fst[path]
	action := DO_NOTHING
	choice := -1
	if reply.Exists == false && exists == true {
		if reply.SyncVect[fst[path].Creator] < fst[path].CreationTime {
			action = SYNC_DOWN
		} else if mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect); mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = DELETE
		} else {
			// Delete-Update conflict
			fmt.Println("Delete-Update conflict on", path, ":", srv, "has deleted, but", tnt.me, "has updated")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = SYNC_DOWN
			} else {
				action = DELETE
			}
		}
	} else if reply.Exists == true && exists == false {
		live_ancestor := tnt.LiveAncestor(path)
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[live_ancestor].SyncVect)
		//fmt.Println("UPDATE-DELETE:", path, live_ancestor, fst[live_ancestor].SyncVect, reply.Creator, reply.CreationTime, reply.VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if fst[live_ancestor].SyncVect[reply.Creator] < reply.CreationTime {
			action = UPDATE
		} else {
			// Update-Delete conflict
			fmt.Println("Update-Delete conflict on", path, ":", srv, "has updated, but", tnt.me, "has deleted")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = UPDATE
			}
		}
	} else /* reply.Exists == true && exists == true */ {
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
		//mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = SYNC_DOWN
		} else {
			action = UPDATE
		}
	}

	var verVect map[int]int64
	var syncVect map[int]int64

	if action == DO_NOTHING {
		live_ancestor := tnt.LiveAncestor(path) // should be the parent actually
		syncVect = MaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
		exists = false
		//setMaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[live_ancestor].VerVect, fst[live_ancestor].SyncVect, fst[live_ancestor].Parent)
	} else if action == DELETE {
		fmt.Println("ACTION:", tnt.me, "is deleting file due to", srv)
		os.RemoveAll(tnt.root + path)
		syncVect = MaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
		tnt.DeleteTree(path)
		exists = false
	} else if action == UPDATE {
		if exists == false {
			tnt.CopyFileFromPeer(srv, path, path)
			// set tnt.LastModTime
			fi, err := os.Lstat(tnt.root + path)
			if err != nil {
				log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
			} else {

				// Create new FSnode
				fst[path] = new(FSnode)
				fst[path].Name = fi.Name()
				fst[path].Size = fi.Size()
				fst[path].IsDir = fi.IsDir()
				fst[path].LastModTime = fi.ModTime()
				fst[path].Children = make(map[string]bool)
				fst[path].VerVect = make(map[int]int64)
				fst[path].SyncVect = make(map[int]int64)
				for i:=0; i<len(tnt.servers); i++ {
					fst[path].VerVect[i] = 0
					fst[path].SyncVect[i] = 0
				}
				fst[path].Parent = parent(path)
				fst[parent(path)].Children[path] = true
			}
		}

		for k, _ := range fst[path].Children {
			var c_exists bool
			var c_verVect map[int]int64
			var c_syncVect map[int]int64
			if fst[k].IsDir == true {
				c_exists, c_verVect, c_syncVect = tnt.SyncDir(srv,k)
			} else {
				c_exists, c_verVect, c_syncVect = tnt.SyncFile(srv,k)
			}
			if c_exists == true {
				setMaxVersionVect(fst[path].VerVect, c_verVect)
				setMinVersionVect(fst[path].SyncVect, c_syncVect)
			} else {
				setMinVersionVect(fst[path].SyncVect, c_syncVect)
			}
		}
		for k, _ := range reply.Children {
			_, present := fst[path].Children[k]
			if present == false {
				var c_exists bool
				var c_verVect map[int]int64
				var c_syncVect map[int]int64
				if reply.IsDir[k] == true {
					c_exists, c_verVect, c_syncVect = tnt.SyncDir(srv,k)
				} else {
					c_exists, c_verVect, c_syncVect = tnt.SyncFile(srv,k)
				}
				if c_exists == true {
					setMaxVersionVect(fst[path].VerVect, c_verVect)
					setMinVersionVect(fst[path].SyncVect, c_syncVect)
				} else {
					setMinVersionVect(fst[path].SyncVect, c_syncVect)
				}
			}
		}
		fst[path].Creator, fst[path].CreationTime = reply.Creator, reply.CreationTime
		setVersionVect(fst[path].VerVect, reply.VerVect)
		setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
		exists = true
	} else /* action == SYNC_DOWN */ {
		setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		tnt.OnlySync(path)
		verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
		exists = true
	}
	return exists, verVect, syncVect
}

func (tnt *TnTServer) SyncFile(srv int, path string) (bool, map[int]int64, map[int]int64) {
	/*
	(1) Check for updates on local version of file: update VerVect, SyncVect if required
	(2) Get VerVect and SyncVect from 'srv'
	(3) Decide between:
		(a) do nothing
		(b) fetch the file
		(c) conflict
	(4) If there is conflict, 
		(a) Check if it's a delete-delete conflict. If yes, then update syncHist and ignore
		(b) If it is some other conflict, then ask the user for action.
		(c) Do appropriate action as specified by user.
		(d) In any case set syncHist appropriately
	*/

	fst := tnt.Tree.MyTree

	//Sync Recursively
	args:=&GetVersionArgs{Path:path}
	var reply GetVersionReply
	for {
		ok:=call(tnt.servers[srv], "TnTServer.GetVersion", args, &reply)
		if ok {
			break
		}
		time.Sleep(RPC_SLEEP_INTERVAL)
	}

	// A : srv ; B : tnt.me
	action := DO_NOTHING
	choice := -1
	_, exists := fst[path]
	if reply.Exists == false && exists == false {
		action = DO_NOTHING // will never happen actually! :)
	} else if reply.Exists == false && exists == true {
		if reply.SyncVect[fst[path].Creator] < fst[path].CreationTime {
			action = DO_NOTHING
		} else if mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect); mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = DELETE
		} else {
			// Delete-Update conflict
			fmt.Println("Delete-Update conflict on", path, ":", srv, "has deleted, but", tnt.me, "has updated")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = DELETE
			}
		}
	} else if reply.Exists == true && exists == false {
		live_ancestor := tnt.LiveAncestor(path)
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[live_ancestor].SyncVect)
		fmt.Println("UPDATE-DELETE:", path, live_ancestor, fst[live_ancestor].SyncVect, reply.Creator, reply.CreationTime, reply.VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if fst[live_ancestor].SyncVect[reply.Creator] < reply.CreationTime {
			action = UPDATE
		} else {
			// Update-Delete conflict
			fmt.Println("Update-Delete conflict on", path, ":", srv, "has updated, but", tnt.me, "has deleted")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = UPDATE
			}
		}
	} else /* reply.Exists == true && exists == true */ {
		mA_vs_sB := compareVersionVects(reply.VerVect, fst[path].SyncVect)
		mB_vs_sA := compareVersionVects(fst[path].VerVect, reply.SyncVect)
		if mA_vs_sB == LESSER || mA_vs_sB == EQUAL {
			action = DO_NOTHING
		} else if  mB_vs_sA == LESSER || mB_vs_sA == EQUAL {
			action = UPDATE
		} else {
			// Update-Update conflict
			fmt.Println("Update-Update conflict on", path, ":", srv, "and", tnt.me, "have independently updated")
			for choice != tnt.me && choice != srv {
				fmt.Printf("Which version do you want (%d or %d)? ", tnt.me, srv)
				fmt.Scanf("%d", &choice)
			}
			if choice == tnt.me {
				action = DO_NOTHING
			} else {
				action = UPDATE
			}
		}
	}

	var verVect map[int]int64
	var syncVect map[int]int64

	if action == DO_NOTHING {
		fmt.Println("ACTION:", tnt.me, "has nothing to do for", path)
		if exists == true {
			setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)
			verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
			//tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
		} else {
			live_ancestor := tnt.LiveAncestor(path) // should be the parent actually
			syncVect = MaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
			//setMaxVersionVect(fst[live_ancestor].SyncVect, reply.SyncVect)
			//tnt.PropagateUp(fst[live_ancestor].VerVect, fst[live_ancestor].SyncVect, fst[live_ancestor].Parent)
		}
	} else if action == DELETE {
		fmt.Println("ACTION:", tnt.me, "is deleting file due to", srv)
		os.Remove(tnt.root + path) // os.RemoveAll(tnt.root + path)
		syncVect = MaxVersionVect(fst[path].SyncVect, reply.SyncVect)
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
		tnt.DeleteTree(path)
		exists = false
	} else if action == UPDATE {
		fmt.Println("ACTION:", tnt.me, "is getting file from", srv)
		// get file
		tnt.CopyDirFromPeer(srv, path, path)
		// set tnt.LastModTime
		fi, err := os.Lstat(tnt.root + path)
		if err != nil {
			log.Println(tnt.me, ": File does not exist:", err, ": LOL - had copied just now!")
		} else {
			// set Exists, VerVect, SyncVect
			if exists == false {
				// Create new FSnode
				fst[path] = new(FSnode)
				fst[path].Name = fi.Name()
				fst[path].Size = fi.Size()
				fst[path].IsDir = fi.IsDir()
				fst[path].VerVect = make(map[int]int64)
				fst[path].SyncVect = make(map[int]int64)
				for i:=0; i<len(tnt.servers); i++ {
					fst[path].VerVect[i] = 0
					fst[path].SyncVect[i] = 0
				}
				fst[path].Parent = parent(path)
				fst[parent(path)].Children[path] = true
			}
			fst[path].LastModTime = fi.ModTime()
			fst[path].Creator, fst[path].CreationTime = reply.Creator, reply.CreationTime
			setVersionVect(fst[path].VerVect, reply.VerVect)
			setMaxVersionVect(fst[path].SyncVect, reply.SyncVect)

			verVect, syncVect = fst[path].VerVect, fst[path].SyncVect
			exists = true
		}
		//tnt.PropagateUp(fst[path].VerVect,fst[path].SyncVect,fst[path].Parent)
	}

	return exists, verVect, syncVect
}

func (tnt *TnTServer) Kill() {
	tnt.dead = true
	tnt.l.Close()
}

func StartServer(servers []string, me int, root string, fstpath string) *TnTServer {
	gob.Register(GetFileArgs{})
	gob.Register(GetDirArgs{})
	tnt := new(TnTServer)
	tnt.me = me
	tnt.servers = servers
	tnt.root = root
	tnt.dump = fstpath //root+"FST_watch_new"
  
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

		tnt.Tree = fst
	} else {
		fmt.Println(tnt.dump, "found! Fetching tree...")
		var fst1 FStree
		decoder := gob.NewDecoder(f)
		decoder.Decode(&fst1)
		tnt.Tree = &fst1
	}
	tnt.UpdateTreeWrapper("./")
	fmt.Println("in start server",tnt.Tree)

	// RPC set-up borrowed from Lab
	rpcs := rpc.NewServer()
	rpcs.Register(tnt)
	os.Remove(servers[me])
	l, e := net.Listen("unix", servers[me]);
	if e != nil {
		log.Fatal("listen error: ", e);
	}
	tnt.l = l

	go func() {
		for tnt.dead == false {
			conn, err := tnt.l.Accept()
			if err == nil && tnt.dead == false {
				go rpcs.ServeConn(conn)
			} else if err == nil {
				conn.Close()
			}
			if err != nil && tnt.dead == false {
				fmt.Printf("TnTServer(%v) accept: %v\n", me, err.Error())
				tnt.Kill()
			}
		}
	}()

	return tnt
}
