FROM arla/base

# install
COPY . /tmp/build
RUN cd /tmp/build && make -f Makefile.mk clean install && rm -rf /tmp/build

EXPOSE 3000
ENTRYPOINT ["/var/lib/arla/bin/init"]
