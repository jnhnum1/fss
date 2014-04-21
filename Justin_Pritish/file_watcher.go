package main

import "code.google.com/p/go.exp/inotify"
import "fmt"
import "log"

func main() {
  
  watcher, err := inotify.NewWatcher()
  if err != nil {
    log.Fatal(err)
  }
  err = watcher.Watch("./watch_folder")
  if err != nil {
    log.Fatal(err)
  }
  err = watcher.Watch("./watch_folder/nest1")
  if err != nil {
    log.Fatal(err)
  }
  
  for {
    select {
    case ev := <-watcher.Event:
        log.Println("event:", ev)
    case err := <-watcher.Error:
        log.Println("error:", err)
    }
  }
  
  fmt.Println("Hello, playground")
}
