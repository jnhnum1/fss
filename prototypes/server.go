package main

//Run on Server 1, initiate sync with server 2. This function can be called
//on startup or frequently during scheduled updates.  Its arguments
//should be the server 2's address.
//This function carries out the entire sync process.  It starts by sending 
//its metadata to server 2, where server 2 will determine which of its files
//need to be updated through an RPC call MetaDataSync().  This
//request is allowed to time out as it is not guaranteed that we will reach
//server 2 every time.
//After the files have been transfered, server 2 will replace its files with
//those from the temp folder. This will occur is UpdateFiles().

func InitiateSync() {

    call(MetaDataSync)
    TransferFiles()
}


//This is the initial RPC call that will be received by server 2.
//It should respond with which files by server 1 need to be sent over.
//FilesToTransmit() will be run in here.  This includes conflict detection.

func MetaDataSync() {
    FilesToTransmit()
}


//This function is run in MetaDataSync on server 2 to determine the differences 
//in MetaData arrays.  Based on which files need to be updated, it returns a list 
//to server 1 which will transfer them.  In addition, it must determine if server 2
//has files that need to be sent to server 1 in the case that server 1 has been offline 
//and needs to get up to date.  If server 1 has out of date files without conflicts, maybe
//now is a time to transfer them over.
//***I'm not sure where server conflict shoud be decided.  On one hand, server 1
//initiates the sync so someone is on that machine to decide conflicts manually, this is not 
//guaranteed if the other machine is just sitting around.  On the
//other hand, server 2 should know which files are to be replaced so that it can lock them
//from edits that may occur in the time of transfer.
//Or if someone is working on server 2, his updates will be overwritten by the decision made
//on server 1 without his consent.

func FilesToTransmit() {
}

//This function is run on server 1, it takes care of transferring the files MetaDataSync()
//decided to transfer to server 2. Not sure if an RPC call is the most effective way to
//transfer large files (I doubt it).  It will transfer the necessary files to a temp folder
//on server 2.

func TransferFiles() {
}

// This function is run on server 2 after the file transfer is completed and the necessary 
// files are sitting in a tmp folder.  This function will make the necessary system calls 
// to move and replace the files.  It must detect if there are additional conflicts and alert
// the user as well as propagate new changes back to server 1.  It will also bring all metadata
// up to date such that the program in the background looking for changes knows these should not
// be recorded.

func UpdateFiles() {
}
