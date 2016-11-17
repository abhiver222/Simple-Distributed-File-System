There are 4 files required to build MP3.
    1. membership.go
    2. messages.go
    3. helpers.go
    4. sdfs.go

There must also be a directory called 'files' that is in the same directory as the .go files.

To run the MP3 code, type the command:
    go run sdfs.go membership.go messages.go helpers.go

To compile an executable, type the command:
    go build sdfs.go membership.go messages.go helpers.go

At startup, the user is prompted for the INTRODUCER/MASTER. By convention, the machine with the lowest IP address is assigned to be the introducer. For the first machine to initiate the group, use the IP of that machine. Otherwise, the user can enter the command '10' in any running machine to obtain the ip of the INTRODUCER/MASTER. After inputing the ip address for the INTRODUCER/MASTER, 9 commands will be printed out for the user. There are 2 hidden ocommands, '10' and '11', for debugging purposes. '10' prints the current MASTER. '11' prints the sdfs file list that the machine has stored locally. As the program is running, a logfile named logfile.txt is created and/or appended to.

TMUX can be used to run the program in the background.


