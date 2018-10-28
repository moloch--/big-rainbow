#!/usr/bin/env python3

'''

> Generates a keyspace given an optional seed value to skip/resume segments

-----------------------------------------------------------------------
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
-----------------------------------------------------------------------
'''

import sys

from string import printable


class KeyspaceGenerator(object):

    MAX_SEED_LENGTH = 8
    DEFAULT_CHARSET = printable[:-5]

    def __init__(self, seed=None, stop=None, charset=None):
        self.generated_count = 0
        self.charset = charset if charset is not None else self.DEFAULT_CHARSET
        self._charmax = len(self.charset)
        if stop is not None and not isinstance(stop, int):
            raise TypeError('Stop must be an integer')
        self.stop = self.to_base_n(stop, self.charset) if stop is not None else None
        self._value = seed if seed is not None else self.charset[0]
        if not isinstance(self._value, str):
            raise TypeError('Seed must be a string')
        if self.MAX_SEED_LENGTH < len(self._value):
            raise ValueError('Max seed length is %d' % self.MAX_SEED_LENGTH)

    @classmethod
    def keyspace_length(cls, seed, charset):
        return len(charset) ** len(seed)

    @classmethod
    def to_base_n(cls, x, charset):
        ''' Convert a number `x` to base `n` str where `n` is the len of the charset '''
        base = len(charset)
        if x < base:
            return charset[x]
        else:
            return cls.to_base_n(x // base, charset) + charset[x % base]

    def _char_indexes(self):
        ''' Get the indexes of each char in the current value '''
        return [self.charset.index(char) for char in self._value]
    
    def _char_indexes_to_str(self, char_indexes):
        ''' Converts char indexes back into a string '''
        return ''.join(self.charset[index] for index in char_indexes)

    def _next_char_indexes(self, char_indexes):
        ''' Given a list of char indexes, return the next list of char indexes '''
        next_char_indexes = [index for index in char_indexes]
        for index, char_index in enumerate(char_indexes[::-1]):
            rindex = (index + 1) * -1  # reverse index
            next_char_indexes[rindex] = (char_index + 1) % self._charmax
            if next_char_indexes[rindex] != 0:
                return next_char_indexes

    def _is_keyspace_limit(self):
        ''' Have we hit the limit of the keyspace? '''
        if all(char == (self._charmax - 1) for char in self._char_indexes()):
            return True
        if self.stop == self._value:
            return True
        return False

    def __len__(self):
        return self.keyspace_length(self._value, self.charset)

    def __getitem__(self, key):
        return self.to_base_n(key, self.charset)

    def __str__(self):
        return self._value

    def __iter__(self):
        ''' Iterate over the entire keyspace starting at `seed` (inclusive) '''
        if self.generated_count == 0:
            yield self._value
        while not self._is_keyspace_limit():
            char_indexes = self._char_indexes()
            next_char_indexes = self._next_char_indexes(char_indexes)
            next_value = self._char_indexes_to_str(next_char_indexes)
            self._value = next_value
            self.generated_count += 1
            yield next_value


if __name__ == '__main__':
    seed = sys.argv[1] if len(sys.argv) == 2 else '00'
    print('Generating from seed: %s' % seed)
    keygen = KeyspaceGenerator(seed)
    with open('generated_keyspace_%d.txt' % len(seed), 'w') as fp:
        for value in keygen:
            fp.write("%s\n" % value)
            print("\r%d" % (keygen.generated_count), end='')
    print('\rGenerated %d values' % (keygen.generated_count + 1))
