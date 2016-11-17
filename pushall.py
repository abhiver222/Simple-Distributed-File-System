import os
import sys
import socket

# make executable
try:
	os.system("go build sdfs.go helpers.go introducer_restart.go membership.go messages.go")
except Exception as e:
	print e
	exit()

# do add
os.system("git add -A")

comMsg = ",".join(sys.argv[1:]).replace(',', ' ')
print "git commit -m \"" + comMsg + "\""

# do commit
os.system("git commit -m \"" + comMsg + "\"")

#push
try:
	os.system("git push origin master")
except:
	print "commit failed"
	exit(0)

with open("sendToList", 'r') as f:
	names = f.readlines()
	curHost = socket.gethostname()
	for n in names:
		stmt1 = "scp ddle2@" + curHost + ":/home/ddle2/CS425-MP3/*.go  " + n
		stmt2 = "scp ddle2@" + curHost + ":/home/ddle2/CS425-MP3/sdfs  " + n
        try:
		    os.system(stmt1)
		    os.system(stmt2)
        except:
		    print "scp to n failed"

