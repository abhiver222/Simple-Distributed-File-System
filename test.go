package main

import (
	"fmt"
	"os/exec"
)

type file_info struct {
	name string
	IP   []string
	size int
}

var MASTER = "172.22.149.18"

var file_list = make(map[string]file_info)

var local_files = make([]string, 0)

func main() {
	//_, err := exec.Command("scp", "averma11@172.22.149.18:/home/averma11/server.c", "/home/ddle2/CS425-MP3/files").Output()
	_, err := exec.Command("cp", "/home/ddle2/CS425-MP3/files/te", "/home/ddle2/CS425-MP3/files/temp1"+"cp").Output()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("done")
}
