FROM arla/base

# install
COPY . /tmp/build
RUN cd /tmp/build && make -f Makefile.mk clean install && rm -rf /tmp/build && mkdir -p /var/state

EXPOSE 80
ENTRYPOINT ["/var/lib/arla/bin/init"]
