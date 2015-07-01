FROM arla/base
EXPOSE 80
ENTRYPOINT ["/usr/bin/arla"]
RUN mkdir -p /var/state
COPY src/arla/querystore/conf/postgresql.conf /etc/postgresql/9.4/main/postgresql.conf
COPY src/arla/querystore/conf/pg_hba.conf /etc/postgresql/9.4/main/pg_hba.conf
COPY bin/arla /usr/bin/
