FROM ubuntu:14.10

# Set lang (used when initializing postgres)
ENV LANGUAGE="en_GB.UTF-8"
ENV LANG="en_GB.UTF-8"
ENV LC_ALL="en_GB.UTF-8"

# install dev dependencies
RUN apt-get update && apt-get install -y golang make

# install postgres plus work around for AUFS bug
# as per https://github.com/docker/docker/issues/783#issuecomment-56013588
RUN apt-get install -y \
    postgresql-9.4 \
    postgresql-contrib-9.4 \
    postgresql-9.4-plv8 \
    postgresql-9.4-postgis-2.1 && \
    echo "working around AUFS bug...." && \
    mkdir /etc/ssl/private-copy; \
    mv /etc/ssl/private/* /etc/ssl/private-copy/; \
    rm -r /etc/ssl/private; mv /etc/ssl/private-copy /etc/ssl/private; \
    chmod -R 0700 /etc/ssl/private; \
    chown -R postgres /etc/ssl/private

# install nodejs and globally install some modules to speed up rebuilds
RUN apt-get install -y nodejs npm && \
    ln -s /usr/bin/nodejs /usr/bin/node && \
    npm install -g browserify babelify && \
    npm install -g mocha chai supertest tmp && \
    npm install -g pegjs

# install
COPY . /tmp/build
RUN cd /tmp/build && make -f Makefile.mk clean install && rm -rf /tmp/build

# tidy up


EXPOSE 3000
ENTRYPOINT ["/var/lib/arla/bin/init"]
