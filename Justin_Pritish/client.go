//This runs in the background to determine when to call InitiateSync()
//in server.go.  It includes all of the work Pritish has done with DFT and watching files.
//Instead of importing all of those files, hopefully you can just make
//simple function calls here. This file will deal with the MetaData Array

package main

func MakeClient() {
}


//This function is called when a machine is started or reset or if the user wants to 
//manually sync his work.  It initiates the sync process as defined in server.go.

func ManualSync() {
}


//This function has a buffer of edits made to the different files since the last sync.
//It watches all files in the directory and pushes the syncs to other machines when its
//buffer of updates fills.  It will also call InitiateSync() in server.go

func PushSync() {
}

