# -*- coding: utf-8; -*-
#
# peg.py - Parsing Expression Grammar implementation
#
# Copyright (C) 2019  Lincoln Clarete
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

import io
import random
import sys


def gencsvdata(file_name, lines, cols):
    "Generate CSV files with random integers"
    with io.open(file_name, 'w') as fp:
        for line in range(lines):
            for col in range(cols):
                data = random.randint(0, col+1 * 100)
                fp.write(str(data))
                fp.write(col < cols-1 and ',' or '')
            fp.write('\n')


if __name__ == '__main__':
    gencsvdata('./data/1.a.csv', 1000, 500)
    gencsvdata('./data/1.b.csv', 500, 1000)
    gencsvdata('./data/1.c.csv', 1000, 1000)
