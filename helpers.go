package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

//Called by introducer after receiving a join message. Compares timestamp of the message and.
//the membershiplist. If timestamp in memberhiplist is more recent than message time, return 1
// otherwise return 0
func checkTimeStamp(m member) int {
	for _, element := range membershipList {
		if m.Host == element.Host {
			t1, _ := time.Parse(time.RFC850, m.TimeStamp)
			t2, _ := time.Parse(time.RFC850, element.TimeStamp)
			if t2.After(t1) {
				element = m
				return 1
			} else {
				break
			}
		}
	}
	return 0
}

//Initialize membershipList with current time and local IP
func initializeML() {
	node := member{currHost, time.Now().Format(time.RFC850)}
	membershipList = append(membershipList, node)
}

// returns 0 if not update, 1 if update
func updateML(hostIndex int, msg message) int {
	printComment(msg.Status, msg.Host)

	localTime, _ := time.Parse(time.RFC850, membershipList[hostIndex].TimeStamp)
	givenTime, _ := time.Parse(time.RFC850, msg.TimeStamp)

	if givenTime.After(localTime) {
		initialSize := len(membershipList)
		membershipList = append(membershipList[:hostIndex], membershipList[hostIndex+1:]...)
		if initialSize >= MIN_HOSTS && len(membershipList) < MIN_HOSTS {
			printComment("Stopping Syn/Ack", "")
		}
		fmt.Println(len(membershipList))
		return 1
	} else {
		//CHECK THIS LATER
		return 0
	}
}

//Helper function to hard reset both timers (stop both and set resetFlags to 1)
func resetTimers() {
	resetFlags[0] = 1
	resetFlags[1] = 1
	timers[0].Reset(0)
	timers[1].Reset(0)
}

//get local IP address in the form of a string
func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		errorCheck(err)
	}
	return addrs[1].String()
}

//get index for local VM in membershipList
func getIndex(ip string) int {
	for i, element := range membershipList {
		if ip == element.Host {
			return i
		}
	}
	return -1

}

/*Takes the host for a member and checks its index with the index for the local VM. Returns 1 if
host is (localIndex + 1)%N or 2 if host is (localIndex + 2)%N, where N is size of membershipList*/
func getRelativeIndex(host string) int {
	localIndex := getIndex(currHost)
	if strings.Compare(membershipList[(localIndex+1)%len(membershipList)].Host, host) == 0 {
		return 1
	} else if strings.Compare(membershipList[(localIndex+2)%len(membershipList)].Host, host) == 0 {
		return 2
	}
	return -1
}

//Helper function to log errors
func errorCheck(err error) {
	if err != nil {
		errlog.Println(err)
	}
}

//Helper function to log general information
func infoCheck(info string) {
	infolog.Println(info)
}

//Helper function to log joining, failing, and leaving
func msgCheck(msg message) {
	switch msg.Status {
	case "Joining":
		joinlog.Println("IP: " + msg.Host)
	case "Failed":
		faillog.Println("IP: " + msg.Host)
	case "Adios":
		leavelog.Println("IP: " + msg.Host)
	default:
		infolog.Println("IP: " + msg.Host + " -> Status: " + msg.Status)
	}
}

//Sets currHost to local IP (as a string)
//Sets membershipList with currHost as its only member with current time
//Initializes timers with MAX_TIME and subsequently stops them. This is to prevent false firing of timers when Syn/Ack begins
func initializeVars() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nLeader/introducer IP? (e.g. 172.22.149.18)")
	INTRODUCER, _ = reader.ReadString('\n')
	INTRODUCER = strings.TrimRight(INTRODUCER, "\n")
	INTRODUCER = INTRODUCER + "/23"
	currHost = getIP()
	initializeML()
	timers[0] = time.NewTimer(MAX_TIME)
	timers[1] = time.NewTimer(MAX_TIME)
	timers[0].Stop()
	timers[1].Stop()

	rand.Seed(time.Now().UTC().UnixNano())

	logfile_exists := 1
	if _, err := os.Stat("logfile.log"); os.IsNotExist(err) {
		logfile_exists = 0
	}

	logfile, _ := os.OpenFile("logfile.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	errlog = log.New(logfile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	infolog = log.New(logfile, "INFO: ", log.Ldate|log.Ltime)
	joinlog = log.New(logfile, "JOINING: ", log.Ldate|log.Ltime)
	leavelog = log.New(logfile, "LEAVING: ", log.Ldate|log.Ltime)
	faillog = log.New(logfile, "FAILED: ", log.Ldate|log.Ltime)
	emptylog = log.New(logfile, "\n----------------------------------------------------------------------------------------\n", log.Ldate|log.Ltime)

	if logfile_exists == 1 {
		emptylog.Println("")
	}
}

/*Helper function to print comments*/
func printComment(src string, tgt string) {
	switch src {
	case "Starting Syn/Ack":
		fmt.Println("\nStarting Syn/Ack\n")
	case "Stopping Syn/Ack":
		fmt.Println("\nStopping Syn/Ack\n")
	case "Failed":
		fmt.Println("\nProcess " + tgt + " failed.\n")
	case "Adios":
		fmt.Println("\nProcess " + tgt + " left group.\n")
	case "Joining":
		fmt.Println("\nProcess " + tgt + " joined group.\n")
	default:
		fmt.Println("...")
	}
}
