#!/usr/bin/env python3
'''

generate-keyspace.py

Generate non-inclusive length combintations of ascii printables
e.g. '3' will produce all possible ascii combinations of 3 chars
     but will not include any 2 length combinations


This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
'''

import sys

from itertools import product
from string import printable

keyspace = int(sys.argv[1]) if len(sys.argv) >= 2 else int(input('keyspace: '))
outfile = sys.argv[2] if len(sys.argv) >= 2 else input('outfile: ')

with open(outfile, 'a') as fp:
    sys.stdout.write('working ... ')
    for count, value in enumerate(product(printable[:-5], repeat=keyspace)):
        fp.write(''.join(value) + '\n')
print('done')
