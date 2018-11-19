#!/usr/bin/env python3

'''
Creates and fills up an SQS queue for distgen to consume
'''

import os
import json
import argparse

from binascii import hexlify

import boto3

from generate_seeded_keyspace import KeyspaceGenerator

MAX_ENTIRES = 10


def gen_id():
    return hexlify(os.urandom(16)).decode()

def fill_queue(args, queue_url):
    ''' Fills up the SQS queue with blocks for the given key space '''
    charset = KeyspaceGenerator.DEFAULT_CHARSET if args.charset is None else args.charset
    z = charset[0]  # The "zero value" symbol in the charset

    end = KeyspaceGenerator.keyspace_length(z*args.chars_len, charset)
    if args.inclusive:
        start = 0
    else:
        start = KeyspaceGenerator.keyspace_length(z*(args.chars_len-1), charset) + 1
    print('Keyspace {} -> {}'.format(start, end))

    sqs = boto3.resource('sqs')
    queue = sqs.Queue(queue_url)

    entries = []
    block_count = 0
    send_count = 0
    for entry in range(start, end, args.block_size):
        stop = entry+args.block_size
        stop = stop if stop < end else end
        entries.append({
            'Id': gen_id(),
            'MessageDeduplicationId': gen_id(),
            'MessageGroupId': gen_id(),
            'MessageBody': json.dumps({
                'start': entry,
                'stop': stop,
            })
        })
        if len(entries) == MAX_ENTIRES:
            send_count += 1
            block_count += len(entries)
            queue.send_messages(Entries=entries)
            entries = []
    if len(entries):
        send_count += 1
        block_count += len(entries)
        queue.send_messages(Entries=entries)
    print('Sent {} block(s) in {} message(s)'.format(block_count, send_count))


def main(args):
    sqs = boto3.resource('sqs', region_name=os.environ.get('AWS_REGION', 'us-west-2'))
    print('Creating sqs queue: %s' % args.sqs_queue)
    sqs_queue = sqs.create_queue(
        QueueName=args.sqs_queue,
        Attributes={'FifoQueue': 'true'})
    print('Create queue response: %r' % sqs_queue)
    if sqs_queue:
        fill_queue(args, sqs_queue.url)
    else:
        print('Failed to create sqs queue: %r', sqs_queue)


if __name__ == '__main__':

    parser = argparse.ArgumentParser(
        description='generate and fill an sqs queue')

    parser.add_argument('-i',
        type=bool,
        dest='inclusive',
        help='generate entire keyspace inclusively (e.g. 1 char, 2 char ...)',
        default=False)

    parser.add_argument('-c',
        type=str,
        dest='charset',
        help='generate keyspace using a given charset',
        default=None)

    parser.add_argument('-k',
        type=int,
        dest='chars_len',
        help='generate keyspace for `n` chars',
        required=True)

    parser.add_argument('-Q',
        dest='sqs_queue',
        default=os.environ.get('DISTGEN_SQS_QUEUE', 'big_rainbow_distgen.fifo'),
        help='sqs queue name (should be fifo queue)')

    parser.add_argument('-B',
        type=int,
        dest='block_size',
        default=int(os.environ.get('DISTGEN_BLOCK_SIZE', 500000)),
        help='block size')

    main(parser.parse_args())
