domasimu
========

domasimu is a command line tool that enables reading and management of dnsimple domains from the command line.

    $ domasimu -l
    yourdomain.com
    yourotherdomain.net
    $ domasimu yourdomain.com
     A 1.2.3.4
    www A 1.2.3.4



Installation
------------

    $ go get github.com/jrwren/domasimu/...
    $ go install github.com/jrwren/domasimu/...
    
Configuration
-------------

    $ echo 'user = "youraccount@example.com' > ~/.domasimurc
    $ echo 'token = "YOURTOKENGOESHERE_YESQUOTED"' >> ~/.domasimurc

Alternate Configuration
-----------------------
domasimu will read config from a different file if DOMASIMU_CONF environment variable is set.

    $ DOMASIMU_CONF="alt-domasimurc"
    $ echo 'user = "yourotheraccount@example.com' > $DOMASIMU_CONF
    $ echo 'token = "YOURTOKENGOESHERE_YESQUOTED"' >> $DOMASIMU_CONF
    $ domasimu -l
    moardomains.org

