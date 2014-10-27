#!/usr/bin/env python
import gevent
import logging
import os
import sys
from optparse import OptionParser

from elastic_polaris.index_creation import IndexCreator
from elastic_polaris.escommon import get_conn
from settings import ES_ARGS, ES_HOSTS, ES_MAX_ALIASES, ZK_HOSTS

Commands = ["create", "cleanup"]

USAGE = """
USAGE!
%prog <command> [options]

Commands:
""" + '\n'.join(["%10s: " % x for x in Commands])

HERE = os.path.abspath(os.path.dirname(__file__))

logging.basicConfig(
    filename=os.path.join(HERE, '%s.log' % __name__),
    format='[%(asctime)s][%(levelname)s][%(module)s:%(lineno)s][] %(message)s',
    level=logging.DEBUG
)


es_conn = get_conn(ES_HOSTS, **ES_ARGS)
index_creator = IndexCreator(es_conn, ZK_HOSTS, ES_MAX_ALIASES)
user_id = "es_alias_concurrency_test_user_id_%s"


class TestAlias:
    def __init__(self, num):
        self.num_clients = num
        self.num_concurrency = num

    def cleanup(self):
        for i in range(self.num_clients):
            alias = user_id % i
            try:
                indexes = es_conn.get_alias(alias)
                for index in indexes:
                    index_creator._remove_alias(index, alias)
            except Exception:
                continue

    def create(self):
        create_alias = lambda x: index_creator.create_alias(x)
        for i in range(self.num_clients):
            alias = user_id % i
            greenlets = [gevent.spawn(create_alias, alias) for i in range(self.num_concurrency)]
            gevent.joinall(greenlets)


def main():
    parser = OptionParser(USAGE)

    parser.add_option('-n', '--number', type="int", dest="number", default=50,
                      help="config the number of routine to execute")


    options, args = parser.parse_args()

    if len(args) != 1:
        parser.print_help()
        print "Error: config the command"
        return 1

    cmd = args[0]
    if cmd not in Commands:
        parser.print_help()
        print "Error: Unkown command: ", cmd
        return 1

    t = TestAlias(options.__dict__['number'])
    getattr(t, cmd)()



if __name__ == "__main__":
    main()
