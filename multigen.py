#!/usr/bin/env python3
'''
Multiprocess keyspace generator
'''

import os
import sys
import time
import json
import argparse
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


def compute_single_entry(word, hash_algorithms):
    results = {"preimage": word}
    for name, algo in hash_algorithms.items():
        results[name] = b64encode(algo(word.encode()).digest()[:TRUNCATE]).decode()
    return json.dumps(results)+"\n"


def start_worker(worker_id, queue, chars_len, hash_algorithms, output):
    fname = "generated_keyspace_%s_%s.json" % (chars_len, worker_id)
    fout = open(os.path.join(output, fname), 'w')
    while not queue.empty():
        start, stop = queue.get()
        compute_keyspace(start, stop, hash_algorithms, fout)
    fout.close()


def sizeof_fmt(num, suffix='B'):
    ''' https://stackoverflow.com/questions/1094841/reusable-library-to-get-human-readable-version-of-file-size '''
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if abs(num) < 1024.0:
            return "%3.1f%s%s" % (num, unit, suffix)
        num /= 1024.0
    return "%.1f%s%s" % (num, 'Yi', suffix)


def main(args):
    charset = KeyspaceGenerator.DEFAULT_CHARSET if args.charset is None else args.charset
    hash_algorithms = get_hash_algorithms(args)
    queue = mp.Queue()
    
    z = charset[0]  # zero symbol
    end = KeyspaceGenerator.keyspace_length(z*args.chars_len, charset)
    if args.inclusive:
        start = 0
    else:
        start = KeyspaceGenerator.keyspace_length(z*(args.chars_len-1), charset) + 1

    if args.skip is not None and start + abs(args.skip) < end:
        start += abs(args.skip)
    if args.limit is not None and start + abs(args.limit) <= end:
        end = start + abs(args.limit)

    block_size = (end // MAX_SIZE) + 1

    print('Keyspace is %d -> %d (%s entries)' % (start, end, end-start))

    # For inclusive spaces we'll end up over estimating a little but whatever
    file_size = (end-start) * len(compute_single_entry(z*args.chars_len, hash_algorithms))
    print('Estimated output is %d bytes (%s)' % (file_size, sizeof_fmt(file_size)))

    print('Block size is %d' % block_size)
    if args.keyspace_only:
        sys.exit()

    for index, block_start in enumerate(range(start, end, block_size)):
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
    parser.add_argument('-a', '--algorithms',
        nargs='*',
        dest='algorithms',
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())),
        required=True)
    parser.add_argument('-o', '--output',
        dest='output',
        default=getcwd(),
        help='output directory to write data to')
    
    parser.add_argument('-k', '--keyspace',
        type=int,
        dest='chars_len',
        help='generate keyspace for `n` chars',
        required=True)
    parser.add_argument('-s', '--skip',
        type=int,
        dest='skip',
        default=None,
        help='skip first `n` entries of a given keyspace')
    parser.add_argument('-l', '--limit',
        type=int,
        dest='limit',
        default=None,
        help='limit work to `n` entires from start')

    parser.add_argument('-c', '--charset',
        type=str,
        dest='charset',
        help='generate keyspace using a given charset',
        default=None)
    parser.add_argument('-i', '--inclusive',
        type=bool,
        dest='inclusive',
        help='generate entire keyspace inclusively (e.g. 1 char, 2 char ...)',
        default=False)
    parser.add_argument('-K', '--keyspace-only',
        action='store_true',
        dest='keyspace_only',
        help='calculate the keyspace and exit',
        default=False)
    main(parser.parse_args())
