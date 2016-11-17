package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

//IP address, as a string, for the introducer - the VM that outher VM's will ping to join the group
var INTRODUCER = "172.22.149.18/23"

//File path for membershipList. Only applies to INTRODUCER
const FILE_PATH = "MList.txt"

//Minimum number of VM's in the group before Syn/Ack-ing begins
const MIN_HOSTS = 5

//Maximum time a VM will wait for an ACK from a machine before marking it as failed
const MAX_TIME = time.Millisecond * 3500

//IP of the local machine as a string
var currHost string

//Flag to indicate if the machine is currently connected to the group
//1 = machine is connected, 0 = machine is not connected
var isConnected int

//Mutex used for membershipList and timers
var mutex = &sync.Mutex{}

//Timers for checking last ack - used in checkLastAck function
//When timer reaches 0, corresponding VM is marked as failed unless reset flags are 1
var timers [2]*time.Timer

//resetFlags indicate whether timer went off or were stopped to reset the timers
//1 = timers were forcefully stopped
var resetFlags [2]int

//Contains all members connected to the group
var membershipList = make([]member, 0)

//Used if introducer crashes and reboots using a locally stored membership list
var validFlags []int

//type and functions used to sort membershipLists
type memList []member

func (slice memList) Len() int           { return len(slice) }
func (slice memList) Less(i, j int) bool { return slice[i].Host < slice[j].Host }
func (slice memList) Swap(i, j int)      { slice[i], slice[j] = slice[j], slice[i] }

//For logging
var logfile *os.File
var errlog *log.Logger
var infolog *log.Logger
var joinlog *log.Logger
var leavelog *log.Logger
var faillog *log.Logger
var emptylog *log.Logger

//For simulating packet loss in percent
const PACKET_LOSS = 0

var packets_lost int

//struct for information sent from client to server
type message struct {
	Host string
	//port string
	Status    string
	TimeStamp string
	File_Info file_info
}

//Information kept for each VM in the group, stored in membershipList
type member struct {
	Host      string
	TimeStamp string
}

//var startup = flag.Int("s", 0, "Value to decide if startup node")

func main() {
	fmt.Println("Harambe")
	initializeVars()

	//start servers to receive connections for messages and membershipList
	//updates from the introducer when new VM's join
	go messageServer()
	go membershipServer()
	go metaDataServer()

	//Reader to take console input from the user
	reader := bufio.NewReader(os.Stdin)

	//Start functions sending syn's and checking for ack's in seperate threads
	go sendSyn()
	go checkLastAck(1)
	go checkLastAck(2)

	//Take user input
	for {
		fmt.Println("1 -> Print membership list")
		fmt.Println("2 -> Print self ID")
		fmt.Println("3 -> Join group")
		fmt.Println("4 -> Leave group")
		fmt.Println("5 -> put [localfilename] [sdfsfilename]")
		fmt.Println("6 -> get [sdfsfilename]")
		fmt.Println("7 -> delete [sdfsfilename]")
		fmt.Println("8 -> ls [sdfsfilename]")
		fmt.Println("9 -> store")
		input, _ := reader.ReadString('\n')
		switch input {
		case "1\n":
			for _, element := range membershipList {
				fmt.Println(element)
			}
		case "2\n":
			fmt.Println(currHost)
		case "3\n":
			if currHost != INTRODUCER && isConnected == 0 {
				fmt.Println("Joining group")
				connectToIntroducer()
				infoCheck(currHost + " is connecting to master")
				isConnected = 1
			} else {
				fmt.Println("I AM THE MASTER")
			}
		case "4\n":
			if isConnected == 1 {
				fmt.Println("Leaving group")
				leaveGroup()
				infoCheck(currHost + " left group")
				os.Exit(0)
			} else {
				fmt.Println("You are currently not connected to a group")
			}
		case "5\n":
			fmt.Println("Local path?")
			local_path, _ := reader.ReadString('\n')
			local_path = strings.TrimRight(local_path, "\n")
			fmt.Println("SDFS name?")
			sdfs_name, _ := reader.ReadString('\n')
			sdfs_name = strings.TrimRight(sdfs_name, "\n")
			store_file(local_path, sdfs_name)
		case "6\n":
			fmt.Println("SDFS name?")
			sdfs_name, _ := reader.ReadString('\n')
			sdfs_name = strings.TrimRight(sdfs_name, "\n")
			fetchFile(sdfs_name)
		case "7\n":
			fmt.Println("SDFS name?")
			sdfs_name, _ := reader.ReadString('\n')
			sdfs_name = strings.TrimRight(sdfs_name, "\n")
			delete_file(sdfs_name)
		case "8\n":
			fmt.Println("SDFS name?")
			sdfs_name, _ := reader.ReadString('\n')
			sdfs_name = strings.TrimRight(sdfs_name, "\n")
			getFileLocations(sdfs_name)
		case "9\n":
			for _, element := range local_files {
				fmt.Println(element)
			}
		case "10\n":
			fmt.Println(INTRODUCER)
		case "11\n":
			fmt.Println(file_list)
		default:
			fmt.Print("Invalid command")
		}
		fmt.Println("\n\n")
	}
}

//Creates a server to respond to messages
func messageServer() {
	ServerAddr, err := net.ResolveUDPAddr("udp", ":10000")
	errorCheck(err)

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	errorCheck(err)
	defer ServerConn.Close()

	buf := make([]byte, 1024)

	for {
		msg := message{}
		n, _, err := ServerConn.ReadFromUDP(buf)
		err = gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(&msg)
		errorCheck(err)
		switch msg.Status {
		/* 	if joining, create a member with the host and current time, add member to membershiplist,
		sort the membershiplist, and sent list to all members in membershipList (only the introducer will receive
		joing message.*/
		case "Joining":
			msgCheck(msg)
			node := member{msg.Host, time.Now().Format(time.RFC850)}
			if checkTimeStamp(node) == 0 {
				mutex.Lock()
				resetTimers()
				initialSize := len(membershipList)
				membershipList = append(membershipList, node)
				sort.Sort(memList(membershipList))
				finalSize := len(membershipList)
				mutex.Unlock()

				printComment(msg.Status, msg.Host)
				if initialSize < MIN_HOSTS && finalSize >= MIN_HOSTS {
					printComment("Starting Syn/Ack", "")
				}
			}
			//propagateMsg(msg)
			sendList()
			sendFileMD()
		/*	if syn, send an ACK back to to the ip that sent the syn*/
		case "SYN":
			/*fmt.Print("Syn received from: ")
			fmt.Println(msg.Host)*/
			sendAck(msg.Host)
		/*	if ack, check if ip that sent the message is either (currIndex + 1)%N or (currIndex + 2)%N
			and reset the corresponding timer to MAX_TIME*/
		case "ACK":
			if msg.Host == membershipList[(getIndex(currHost)+1)%len(membershipList)].Host {
				/*fmt.Print("ACK received from ")
				fmt.Println(msg.Host)*/
				timers[0].Reset(MAX_TIME)
			} else if msg.Host == membershipList[(getIndex(currHost)+2)%len(membershipList)].Host {
				/*fmt.Print("ACK received from ")
				fmt.Println(msg.Host)*/
				timers[1].Reset(MAX_TIME)
			}
		/*	if message status is failed, propagate the message (timers will be taken care of in checkLastAck*/
		case "Failed":
			if propagateMsg(msg) == 1 {
				if currHost == INTRODUCER {
					go replicate(msg.Host)
				}
			}
			INTRODUCER = membershipList[0].Host
			if currHost == INTRODUCER {
				go crossCheckMD()
			}
		/*	if a node leaves, propagate message and reset timers*/
		case "Adios":
			mutex.Lock()
			resetTimers()
			if propagateMsg(msg) == 1 {
				if currHost == INTRODUCER {
					go replicate(msg.Host)
				}
			}
			INTRODUCER = membershipList[0].Host
			if currHost == INTRODUCER {
				go crossCheckMD()
			}
			mutex.Unlock()
		/*	received only by introducer. sent when a process wants to add a file to the sdfs. introducer first checks
			if file exists. If it does, the introducer replies with a 'file exists' message as to not overwrite data.
			Else, introducer adds file to the sdfs, adds the file to file_ips, adds the vm that sent the 'addfile' message
			to file_ips, replicates the file in the subsequent 3 processes in the membership list, and adds those 3 processes
			to the file_ips list as well.*/
		case "AddFile":
			if _, exists := file_list[msg.File_Info.Name]; exists {
				go sendFileExists(msg)
			} else {
				ip_dest1 := membershipList[(getIndex(msg.Host)+1)%len(membershipList)].Host
				ip_dest2 := membershipList[(getIndex(msg.Host)+2)%len(membershipList)].Host
				ip_dest3 := membershipList[(getIndex(msg.Host)+3)%len(membershipList)].Host
				go sendFile(msg.Host, ip_dest1, msg.File_Info.Name)
				go sendFile(msg.Host, ip_dest2, msg.File_Info.Name)
				go sendFile(msg.Host, ip_dest3, msg.File_Info.Name)

				file_ips := make([]string, 0)
				file_ips = append(file_ips, msg.Host)
				file_ips = append(file_ips, ip_dest1)
				file_ips = append(file_ips, ip_dest2)
				file_ips = append(file_ips, ip_dest3)
				info := file_info{msg.File_Info.Name, file_ips, msg.File_Info.Size}
				file_list[msg.File_Info.Name] = info
				sendFileMD()

				message := message{currHost, "FileSent", time.Now().Format(time.RFC850), info}
				var targetHosts = make([]string, 3)
				targetHosts[0] = ip_dest1
				targetHosts[1] = ip_dest2
				targetHosts[2] = ip_dest3

				sendMsg(message, targetHosts)
			}
		/*	If processes that receives this message is introducer, check to see if file exists. If it does delete file
			from file_list and send a 'delete file' message to all machines that contain file. Else, reply with a
			'file does not exist' message. If process that receives message is not introducer, delete the file*/
		case "DeleteFile":
			if currHost == INTRODUCER {
				if tgtFileInfo, exists := file_list[msg.File_Info.Name]; exists {
					delete(file_list, msg.File_Info.Name)
					sendFileMD()
					sendDeleteFile(tgtFileInfo)
				} else {
					message := message{currHost, "FileDoesntExist", time.Now().Format(time.RFC850), file_info{msg.File_Info.Name, nil, 0}}
					var targetHosts = make([]string, 1)
					targetHosts[0] = msg.Host
					sendMsg(message, targetHosts)
				}
			} else {
				removeFile(msg.File_Info.Name)
			}
		case "FileExists":
			removeFile(msg.File_Info.Name)
		case "FileDoesntExist":
			fmt.Println("File does not exist")
			infoCheck("file " + msg.File_Info.Name + " doesn't exist")			
		/*	Received when a file was scp'ed and is the local file list needs to be updated. Checks first to see if file
			already exists in local file list. If it does, do nothing. Else, add it to the list*/
		case "FileSent":
			exists := false
			for _, file := range local_files {
				if file == msg.File_Info.Name {
					exists = true
					break
				}
			}
			if !exists {
				local_files = append(local_files, msg.File_Info.Name)
				infoCheck("file " + msg.File_Info.Name + " added to " + currHost)
			}
		/*	Received only by the introducer. Checks if file exists. If it does, replies with file information. Else,
			replies with 'file does not exist' message.*/
		case "getFileLocations":
			Message := message{"", "", "", file_info{"", nil, 0}}
			if tgtFileInfo, exists := file_list[msg.File_Info.Name]; exists {
				Message = message{currHost, "sentFileLocations", time.Now().Format(time.RFC850), tgtFileInfo}
			} else {
				Message = message{currHost, "FileDoesntExist", time.Now().Format(time.RFC850), file_info{msg.File_Info.Name, nil, 0}}
			}
			var targetHosts = make([]string, 1)
			targetHosts[0] = msg.Host
			sendMsg(Message, targetHosts)
		/*	Received by process that request file locations if the file exists*/
		case "sentFileLocations":
			for _, element := range msg.File_Info.IPs {
				fmt.Println(element)
			}
		/*	Received only by the introducer. Checks if file exists. If it does, scp's file from one of the
			machines that contains the file to the machine the requested the file. Else, replies with a file
			does not exist message.*/
		case "requestFile":
			fmt.Println("File requested")
			if tgtFileInfo, exists := file_list[msg.File_Info.Name]; exists {
				//Does not take into account if the first ip in tgtFileInfo's IP's list fails during transfer
				go sendFile(tgtFileInfo.IPs[0], msg.Host, msg.File_Info.Name)
				new_file_ips := tgtFileInfo.IPs
				new_file_ips = append(new_file_ips, msg.Host)
				info := file_info{msg.File_Info.Name, new_file_ips, tgtFileInfo.Size}
				file_list[msg.File_Info.Name] = info
				message := message{currHost, "FileSent", time.Now().Format(time.RFC850), tgtFileInfo}
				var targetHosts = make([]string, 1)
				targetHosts[0] = msg.Host

				sendMsg(message, targetHosts)
			} else {
				message := message{currHost, "FileDoesntExist", time.Now().Format(time.RFC850), file_info{msg.File_Info.Name, nil, 0}}
				var targetHosts = make([]string, 1)
				targetHosts[0] = msg.Host
				sendMsg(message, targetHosts)
			}
		}
	}
}

//Server to receieve updated membershipList from introducer if a new member joins
func membershipServer() {
	ServerAddr, err := net.ResolveUDPAddr("udp", ":10001")
	errorCheck(err)

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	errorCheck(err)
	defer ServerConn.Close()

	buf := make([]byte, 1024)

	for {
		mL := make([]member, 0)
		n, _, err := ServerConn.ReadFromUDP(buf)
		err = gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(&mL)
		errorCheck(err)

		//restart timers if membershipList is updated
		mutex.Lock()
		resetTimers()
		membershipList = mL
		mutex.Unlock()

		var msg = "New VM joined the group: \n\t["
		var N = len(mL) - 1
		for i, host := range mL {
			msg += "(" + host.Host + " | " + host.TimeStamp + ")"
			if i != N {
				msg += ", \n\t"
			} else {
				msg += "]"
			}
		}
		infoCheck(msg)
	}
}

//VM's are marked as failed if they have not responded with an ACK within MAX_TIME
//2 checkLastAck calls persist at any given time, one to check the VM at (currIndex + 1)%N and one to
//check the VM (currIndex + 2)%N, where N is the size of the membershipList
//relativeIndex can be 1 or 2 and indicates what VM the function to watch
//A timer for each of the two VM counts down from MAX_TIME and is reset whenever an ACK is received (handled in
// messageServer function.
//Timers are reset whenever the membershipList is modified
//The timer will reach 0 if an ACK isn't received from the corresponding VM
// within MAX_TIME, or the timer is reset. If a timer was reset, the corresponding resetFlag will be 1
// and indicate that checkLastAck should be called again and that the failure detection should not be called
//If a timer reaches 0 because an ACK was not received in time, the VM is marked as failed and th message is
//propagated to the next 2 VM's in the membershipList. Both timers are then restarted.
func checkLastAck(relativeIndex int) {
	//Wait until number of members in group is at least MIN_HOSTS before checking for ACKs
	for len(membershipList) < MIN_HOSTS {
		time.Sleep(100 * time.Millisecond)
	}

	//Get host at (currIndex + relativeIndex)%N
	host := membershipList[(getIndex(currHost)+relativeIndex)%len(membershipList)].Host
	/*fmt.Print("Checking ")
	fmt.Print(relativeIndex)
	fmt.Print(": ")
	fmt.Println(host)*/

	//Create a new timer and hold until timer reaches 0 or is reset
	timers[relativeIndex-1] = time.NewTimer(MAX_TIME)
	<-timers[relativeIndex-1].C

	/*	3 conditions will prevent failure detection from going off
		1. Number of members is less than the MIN_HOSTS
		2. The target host's relative index is no longer the same as when the checkLastAck function was called. Meaning
		the membershipList has been updated and the checkLastAck should update it's host
		3. resetFlags for the corresponding timer is set to 1, again meaning that the membership list was updated and
		checkLastack needs to reset the VM it is monitoring.*/
	mutex.Lock()
	if len(membershipList) >= MIN_HOSTS && getRelativeIndex(host) == relativeIndex && resetFlags[relativeIndex-1] != 1 {
		msg := message{membershipList[(getIndex(currHost)+relativeIndex)%len(membershipList)].Host, "Failed", time.Now().Format(time.RFC850), file_info{"", nil, 0}}
		/*fmt.Print("Failure detected: ")
		fmt.Println(msg.Host)*/
		if propagateMsg(msg) == 1 && currHost == INTRODUCER {
			//fmt.Println("Replicating in checkLastAck")
			replicate(msg.Host)
		}
		INTRODUCER = membershipList[0].Host
		if currHost == INTRODUCER {
			crossCheckMD()
		}
	}
	//If a failure is detected for one timer, reset the other as well.
	if resetFlags[relativeIndex-1] == 0 {
		/*fmt.Print("Force stopping timer")
		fmt.Println(relativeIndex)*/
		resetFlags[relativeIndex%2] = 1
		timers[relativeIndex%2].Reset(0)
	} else {
		resetFlags[relativeIndex-1] = 0
	}

	mutex.Unlock()
	go checkLastAck(relativeIndex)

}
