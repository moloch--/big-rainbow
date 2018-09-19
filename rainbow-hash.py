#!/usr/bin/env python3


import sys
import time
import json
import struct
import hashlib
import argparse
import threading
import platform

from binascii import hexlify
from base64 import b64encode
from os import _exit, getcwd, path

try:
    from algorithms import algorithms
except ImportError:
    sys.stderr.write("Missing file Algorithms.py")
    _exit(2)

if platform.system().lower() in ['windows']:
    print("[!] It appears you're running a shitty operating system" + \
          " make sure to use a real terminal emulator (not cmd.exe)")

W = "\033[0m"  # default/white
R = "\033[31m"  # red
P = "\033[35m"  # purple
C = "\033[36m"  # cyan
O = "\033[33m"
bold = "\033[1m"
clear = chr(27) + '[2K\r'

INFO = bold + C + "[*] " + W
WARN = bold + R + "[!] " + W
MONEY = bold + O + "[$] " + W
PROMPT = bold + P + "[?] " + W


def create_index(fword, fout, hash_algorithms, flock):
    ''' Create an index and write to file '''
    line = fword.readline()
    while line:
        word = line.strip()
        try:
            results = {"preimage": word.decode()}
            for name, algo in hash_algorithms.items():
                results[name] = b64encode(algo(word).digest()).decode()
            data = json.dumps(results)+"\n"
            flock.acquire()
            fout.write(data.encode())
            flock.release()
        except UnicodeDecodeError:
            pass
        except KeyboardInterrupt:
            return
        finally:
            line = fword.readline()


def display_status(fword, fout, flock):
    ''' Display status / progress '''
    try:
        megabyte = (1024.0 ** 2.0)
        fpath = path.abspath(fword.name)
        size = path.getsize(fpath) / megabyte
        sys.stdout.write(INFO + 'Reading %s ...\n' % fpath)
        while not fword.closed and not fout.closed:
            flock.acquire()
            fword_pos = float(fword.tell() / megabyte)
            fout_pos = fout.tell()
            flock.release()
            sys.stdout.write(clear)
            sys.stdout.write(INFO + '%.2f Mb of %.2f Mb' % (fword_pos, size))
            sys.stdout.write(' (%3.2f%s) ->' % ((100.0 * (fword_pos / size)), '%',))
            sys.stdout.write(' "%s" (%.2f Mb)' % (fout.name, float(fout_pos / megabyte)))
            sys.stdout.flush()
            time.sleep(0.25)
    except Exception as error:
        raise error


def index_wordlist(fword, fout, hash_algorithms, flock):
    try:
        thread = threading.Thread(target=display_status, args=(fword, fout, flock))
        thread.start()
        create_index(fword, fout, hash_algorithms, flock)
    except KeyboardInterrupt:
        sys.stdout.write(clear + WARN + 'User requested stop ...\n')
        return
    finally:
        fout.close()
        fword.close()
        thread.join()

def get_hash_algorithms(args):
    if 'all' in args.algorithms:
        return algorithms
    else:
        hash_algorithms = {}
        for name in args.algorithms:
            if name in algorithms:
                hash_algorithms[name] = algorithms[name]
        return hash_algorithms

def main(args):
    flock = threading.Lock()
    hash_algorithms = get_hash_algorithms(args)
    fword = open(args.wordlist, 'rb')
    mode = 'wb'
    if path.exists(args.output) and path.isfile(args.output):
        prompt = input(PROMPT+'File already exists %s [w/a/skip]: ' % args.output)
        if prompt.lower() == 'a':
            mode = 'ab'
        elif prompt.lower() != 'w':
            mode = None
    if mode is not None:
        fout = open(args.output, mode)
        sys.stdout.write(clear + INFO + "Creating " + bold)
        sys.stdout.write(','.join([k for k in hash_algorithms]) + W + " index ...\n")
        sys.stdout.flush()
        index_wordlist(fword, fout, hash_algorithms, flock)
        sys.stdout.write(clear + INFO + "Completed index file %s\n" % args.output)
    sys.stdout.write(clear + MONEY + 'All Done.\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Create unsorted IDX files')
    parser.add_argument('-v', '--version',
        action='version',
        version='Create IDX 0.1.1')
    parser.add_argument('-w',
        dest='wordlist',
        help='index passwords from text file',
        required=True)
    parser.add_argument('-a',
        nargs='*',
        dest='algorithms',
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())),
        required=True)
    parser.add_argument('-o',
        dest='output',
        default=getcwd(),
        help='output directory to write data to')
    args = parser.parse_args()
    if path.exists(args.wordlist) and path.isfile(args.wordlist):
        main(args)
    else:
        sys.stderr.write('Wordlist does not exist, or is not file')
