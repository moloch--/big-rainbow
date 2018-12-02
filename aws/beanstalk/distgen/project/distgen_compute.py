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
import  multiprocessing as mp

from os import getcwd, _exit
from binascii import hexlify
from base64 import b64encode

import boto3

from generate_seeded_keyspace import KeyspaceGenerator
from algorithms import algorithms

ALL = 'all'
TRUNCATE = 6


def get_hash_algorithms(algorithm_names):
    ''' Gets the algorithm objects by name(s) '''
    if ALL in algorithm_names:
        return algorithms
    else:
        hash_algorithms = {}
        for name in algorithm_names:
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


def start_worker(worker_id, sqs_queue_name, s3_bucket, algorithm_names=None, charset=None):
    ''' Excuted as a worker process '''
    
    s3 = boto3.client('s3')
    sqs = boto3.resource('sqs', region_name=os.environ.get('AWS_REGION', 'us-west-2'))
    sqs_queue = sqs.get_queue_by_name(QueueName=sqs_queue_name, region_name=os.environ.get('AWS_REGION', 'us-west-2'))
    charset = KeyspaceGenerator.DEFAULT_CHARSET if charset is None else charset
    algorithm_names = ['all'] if algorithm_names is None else algorithm_names
    hash_algorithms = get_hash_algorithms(algorithm_names)

    while True:
        for message in sqs_queue.receive_messages():
            print('Recieved message(s): %r' % message)
            try:
                block = json.loads(message.body)
                fname = "generated_keyspace_{}_{}.json".format(block['start'], block['stop'])
                fpath = os.path.join(getcwd(), fname)
                with open(fpath, 'w') as fout:
                    compute_keyspace(block['start'], block['stop'], hash_algorithms, charset, fout)
                with open(fpath, 'r') as fout:
                    print("S3 Put '{}' -> {}://{}".format(fpath, s3_bucket, fname))
                    s3.put_object(Bucket=s3_bucket, Key=fname, Body=fout.read())
                os.unlink(fpath)
                message.delete()
            except:
                logging.exception('Error in worker process')

def main(args):
    ''' Starts worker processes '''
    print('Starting big rainbow dist-gen')
    workers = []
    print('Starting %d worker processes' % mp.cpu_count())
    for worker_id in range(mp.cpu_count()):
        worker = mp.Process(target=start_worker, 
                            args=(worker_id, args.sqs_queue, args.s3_bucket, args.algorithms))
        worker.start()
        workers.append(worker)
    [worker.join() for worker in workers]       


def get_default_algorithms():
    env_algos = os.environ.get('DISTGEN_ALGORITHMS', None)
    if env_algos is None or env_algos == '':
        return ['all']
    else:
        return env_algos.split(' ')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Create JSON keyspaces for hashes')
    
    parser.add_argument('-a',
        nargs='*',
        dest='algorithms',
        default=get_default_algorithms(),
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())))

    parser.add_argument('-Q',
        dest='sqs_queue',
        default=os.environ.get('DISTGEN_SQS_QUEUE', 'big_rainbow_distgen'),
        help='sqs queue name (should be fifo queue)')

    parser.add_argument('-b',
        dest='s3_bucket',
        default=os.environ.get('DISTGEN_S3_BUCKET', 'big-rainbow-distgen'),
        help='s3 bucket name to store results')
    
    main(parser.parse_args())
