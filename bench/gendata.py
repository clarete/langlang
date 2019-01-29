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
import json


def gencsvdata(file_name, lines, cols):
    "Generate CSV files with random integers"
    with io.open(file_name, 'w') as fp:
        for line in range(lines):
            for col in range(cols):
                data = random.randint(0, col+1 * 100)
                fp.write(str(data))
                fp.write(col < cols-1 and ',' or '')
            fp.write('\n')


def genjsondata(file_name, elements, depth):
    "Generate JSON data with some depth"
    sys.setrecursionlimit(2000)
    def create(sdepth):
        node = {}
        for el in range(elements):
            node["el%d" % el] = el
        if sdepth:
            node['depth%d' % sdepth] = create(sdepth-1)
        return node
    with io.open(file_name, 'w') as fp:
        json.dump(create(depth), fp)


if __name__ == '__main__':
    gencsvdata('./data/1.a.csv', 10, 10)
    gencsvdata('./data/1.b.csv', 100, 100)
    gencsvdata('./data/1.c.csv', 1000, 1000)

    gencsvdata('./data/2.a.csv', 1000, 500)
    gencsvdata('./data/2.b.csv', 500, 1000)
    gencsvdata('./data/2.c.csv', 1000, 1000)

    genjsondata('./data/1.a.json', 10, 0)
    genjsondata('./data/1.b.json', 100, 0)
    genjsondata('./data/1.c.json', 1000, 0)

    genjsondata('./data/2.a.json', 0, 10)
    genjsondata('./data/2.b.json', 0, 100)
    genjsondata('./data/2.c.json', 0, 1000)

    genjsondata('./data/3.a.json', 10, 10)
    genjsondata('./data/3.b.json', 100, 100)
    genjsondata('./data/3.c.json', 1000, 1000)

