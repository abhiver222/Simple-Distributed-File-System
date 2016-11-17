import os
import sys

for x in range(19, 28):
    os.system("scp -r test_files ddle2@172.22.149.{d}:/home/ddle2/CS425-MP3/".format(d=x))
