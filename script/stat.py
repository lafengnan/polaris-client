#!/usr/bin/env python
# coding=utf-8

import sys

precise = 1e-6

def getTime(f):
    tv = []
    try:
        with open(f, 'r') as fh:
            l = fh.readlines()
            for each_l in l:
                tv.append(each_l.split(' ')[-1])
            return tv
    except Exception as e:
        print e



if __name__ == '__main__':


    f = sys.argv[1]
    tv = getTime(f)

    bt1s = 0
    bt100ms = 0
    bt1ms = 0
    total = len(tv)


    for t in tv:
        if float(t) > 1.0:
            bt1s += 1
        elif float(t) > 0.01 and float(t) < 1.0:
            bt100ms += 1
        elif float(t) > 0.001 and float(t) < 0.01:
            bt1ms += 1


    print "Total: %d" % len(tv)
    print "Time > 1s count: %d, %d%%" % (bt1s, 100*bt1s/total)
    print "Time > 100ms count: %d, %d%%" % (bt100ms, 100*bt100ms/total)
    print "Time > 1ms count: %d, %d%%" % (bt1ms, 100*bt1ms/total)
