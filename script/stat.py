#!/usr/bin/env python
# coding=utf-8

import sys

precise = 1e-9

def getRespTime(f):
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
    tv = getRespTime(f)

    bt4s = 0
    bt1s = 0
    bt100ms = 0
    bt500ms = 0
    bt1ms = 0
    maxRes = 0.0
    total = len(tv)
    for v in tv:
        maxRes = float(v) if float(v) > maxRes else maxRes


    for t in tv:
        if float(t) > 4.0 or abs(float(t) - 4.0) < precise:
            bt4s += 1
        elif float(t) > 1.0 or abs(float(t) - 1.0) < precise:
            bt1s += 1
        elif float(t) > 0.05 or abs(float(t) - 0.05) <precise:
            bt500ms += 1
        elif float(t) > 0.01 or abs(float(t) - 0.01) < precise:
            bt100ms += 1
        elif float(t) > 0.001 or abs(float(t) - 0.001) < precise:
            bt1ms += 1


    print "Total: %d, max response time: %f s" % (len(tv), maxRes)
    print "Time >= 4s count: %d, %.9f%%" % (bt4s, 100*bt4s/float(total))
    print "Time >= 1s && < 4s count: %d, %.9f%%" % (bt1s, 100*bt1s/float(total))
    print "Time >= 500ms && < 1s count: %d, %.9f%%" % (bt500ms, 100*bt500ms/float(total))
    print "Time >= 100ms && < 500ms count: %d, %.9f%%" % (bt100ms, 100*bt100ms/float(total))
    print "Time >= 1ms && < 100ms count: %d, %.9f%%" % (bt1ms, 100*bt1ms/float(total))
