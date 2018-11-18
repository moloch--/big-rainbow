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


def start_worker(worker_id, sqs_queue_name, s3_bucket, algorithms='all', charset=None):
    ''' Excuted as a worker process '''
    
    s3 = boto3.client('s3')
    sqs = boto3.resource('sqs')
    sqs_queue = sqs.get_queue_by_name(QueueName=sqs_queue_name)
    charset = KeyspaceGenerator.DEFAULT_CHARSET if charset is None else charset
    hash_algorithms = get_hash_algorithms(algorithms)

    for messages in sqs_queue.receive_messages(MaxNumberOfMessages=1):
        logging.info('Recieved message(s): %r', messages)
        message = messages[0]  # We should only get one at a time
        try:
            block = json.loads(message.Body)
            fname = "generated_keyspace_{}_{}.json".format(block['start'], block['stop'])
            fpath = os.path.join(getcwd(), fname)
            with open(fpath, 'w') as fout:
                compute_keyspace(block['start'], block['stop'], hash_algorithms, charset, fout)
            with open(fpath, 'r') as fout:
                logging.info("S3 Put '%s' -> %s:%s", fpath, s3_bucket, fname)
                s3.put_object(bucket=s3_bucket, key=fname, body=fout.read())
            os.unlink(fpath)
            sqs_queue.delete_messages(Entries=[{
                'Id': message['Id'],
                'ReceiptHandle': message['ReceiptHandle'],
            }])
        except:
            logging.exception('Error in worker process')

def main(args):
    ''' Starts worker processes '''
    logging.info('Starting big rainbow dist-gen')
    workers = []
    logging.info('Starting %d worker processes', mp.cpu_count())
    for worker_id in range(mp.cpu_count()):
        worker = mp.Process(target=start_worker, 
                            args=(worker_id, args.sqs_queue, args.s3_bucket))
        worker.start()
        workers.append(worker)
    [worker.join() for worker in workers]       


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Create JSON keyspaces for hashes')
    
    parser.add_argument('-a',
        nargs='*',
        dest='algorithms',
        default='all',
        help='hashing algorithm to use: %s' % (['all']+ sorted(algorithms.keys())))

    parser.add_argument('-Q',
        dest='sqs_queue',
        default=os.environ.get('DISTGEN_SQS_QUEUE', 'big_rainbow_distgen'),
        help='sqs queue name (should be fifo queue)')

    parser.add_argument('-b',
        dest='s3_bucket',
        default=os.environ.get('DISTGEN_S3_BUCKET', 'big_rainbow_distgen'),
        help='s3 bucket name to store results')
    
    main(parser.parse_args())
