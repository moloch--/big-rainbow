#!/usr/bin/env python3
'''
Multiprocess keyspace generator
'''

import os
import sys
import time
import json
import argparse
import threading
import  multiprocessing as mp

from os import getcwd, _exit
from binascii import hexlify
from base64 import b64encode

from generate_seeded_keyspace import KeyspaceGenerator
from algorithms import algorithms

ALL = 'all'
TRUNCATE = 6
CLEAR  = "\r\x1b[2K"
MAX_SIZE = 32000


def get_hash_algorithms(args):
    ''' Gets the algorithm objects by name(s) '''
    if ALL in args.algorithms:
        return algorithms
    else:
        hash_algorithms = {}
        for name in args.algorithms:
            if name in algorithms:
                hash_algorithms[name] = algorithms[name]
        return hash_algorithms


def compute_keyspace(start, stop, hash_algorithms, fout):
    seed = KeyspaceGenerator.to_base_n(start, KeyspaceGenerator.DEFAULT_CHARSET)
    keyspace = KeyspaceGenerator(seed, stop)
    for word in keyspace:
        results = {"preimage": word}
        for name, algo in hash_algorithms.items():
            results[name] = b64encode(algo(word.encode()).digest()[:TRUNCATE]).decode()
        data = json.dumps(results)+"\n"
        fout.write(data)


def start_worker(worker_id, queue, chars_len, hash_algorithms, output):
    fname = "generated_keyspace_%s_%s.json" % (chars_len, worker_id)
    fout = open(os.path.join(output, fname), 'w')
    while not queue.empty():
        start, stop = queue.get()
        compute_keyspace(start, stop, hash_algorithms, fout)
    fout.close()


def main(args):
    charset = KeyspaceGenerator.DEFAULT_CHARSET if args.charset is None else args.charset
    hash_algorithms = get_hash_algorithms(args)
    queue = mp.Queue()
    keyspace_len = KeyspaceGenerator.keyspace_length('0'*args.chars_len, charset)
    
    block_size = (keyspace_len // MAX_SIZE) + 1
    if block_size < 1:
        block_size = 1

    print('Keyspace is %d' % keyspace_len)
    print('Block size is %d' % block_size)

    for index, block_start in enumerate(range(0, keyspace_len, block_size)):
        queue.put((block_start, block_start+block_size))
        sys.stdout.write(CLEAR)
        sys.stdout.write('Generated {} block ({} -> {}) ...'.format(
            index+1, block_start, block_start+block_size))
        sys.stdout.flush()
    print(CLEAR+'Block generation completed')

    print('Starting %d workers ...' % mp.cpu_count())
    workers = []
    for worker_id in range(mp.cpu_count()):
        worker = mp.Process(target=start_worker, 
                            args=(worker_id, queue, args.chars_len, hash_algorithms, args.output))
        worker.start()
        workers.append(worker)
    [worker.join() for worker in workers]
    print(CLEAR+"Done.")


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Create JSON keyspaces for hashes')
    parser.add_argument('-a',
        nargs='*',
        dest='algorithms',
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())),
        required=True)
    parser.add_argument('-o',
        dest='output',
        default=getcwd(),
        help='output directory to write data to')
    parser.add_argument('-k',
        type=int,
        dest='chars_len',
        help='generate keyspace for `n` chars',
        required=True)
    parser.add_argument('-c',
        type=str,
        dest='charset',
        help='generate keyspace using a given charset',
        default=None)
    main(parser.parse_args())
