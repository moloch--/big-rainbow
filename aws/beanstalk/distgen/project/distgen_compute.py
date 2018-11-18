#!/usr/bin/env python3
'''
Compute distributed keyspace generator
'''

import os
import sys
import time
import json
import logging
import argparse

from os import getcwd, _exit
from binascii import hexlify
from base64 import b64encode

import boto3

from generate_seeded_keyspace import KeyspaceGenerator
from algorithms import algorithms

ALL = 'all'
TRUNCATE = 6

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


def compute_keyspace(start, stop, hash_algorithms, charset, fout):
    seed = KeyspaceGenerator.to_base_n(start, charset)
    keyspace = KeyspaceGenerator(seed, stop)
    for word in keyspace:
        results = {"preimage": word}
        for name, algo in hash_algorithms.items():
            results[name] = b64encode(algo(word.encode()).digest()[:TRUNCATE]).decode()
        data = json.dumps(results)+"\n"
        fout.write(data)


def main(args):
    logging.info('Starting big rainbow dist-gen')
    charset = KeyspaceGenerator.DEFAULT_CHARSET if args.charset is None else args.charset
    hash_algorithms = get_hash_algorithms(args)
    
    sqs = boto3.resource('sqs')
    s3 = boto3.client('s3')

    queue = sqs.get_queue_by_name(QueueName=args.sqs_queue)
    for message in queue.receive_messages():
        logging.info("Processing msg: %s", message)
        msg = json.loads(message)
        fname = "generated_keyspace_{}_{}.json".format(msg['start'], msg['stop'])
        fpath = os.path.join(args.output, fname)
        with open(fpath, 'w') as fout:
            compute_keyspace(msg['start'], msg['stop'], hash_algorithms, charset, fout)
        message.delete()
        with open(fpath, 'r') as fp:
            logging.info("S3 Put '%s' -> %s:%s", fpath, args.s3_bucket, fname)
            s3.put_object(bucket=args.s3_bucket, key=fname, body=fp.read())
        logging.info("Unlink '%s'", fpath)
        os.unlink(fpath) 


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Create JSON keyspaces for hashes')
    
    parser.add_argument('-a',
        nargs='*',
        dest='algorithms',
        default='all',
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())))

    parser.add_argument('-o',
        dest='output',
        default=getcwd(),
        help='output file to write data to')

    parser.add_argument('-Q',
        dest='sqs_queue',
        default=os.environ.get('DISTGEN_SQS_QUEUE', 'big_rainbow_distgen'),
        help='sqs queue name (should be fifo queue)')

    parser.add_argument('-b',
        dest='s3_bucket',
        default=os.environ.get('DISTGEN_S3_BUCKET', 'big_rainbow_distgen'),
        help='s3 bucket name to store results')
    
    main(parser.parse_args())
