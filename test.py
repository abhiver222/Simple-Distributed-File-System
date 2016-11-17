import time
import os

start = time.time()
os.system("scp averma11@172.22.149.19:/home/averma11/CS425-MP3/files/30MBfile.gz averma11@172.22.149.22:/home/averma11/CS425-MP3/files/")
print ("---- %s seconds ----" % (time.time()-start))
