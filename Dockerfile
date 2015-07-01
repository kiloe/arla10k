FROM arla/base
EXPOSE 80
ENTRYPOINT ["/usr/bin/arla"]
RUN mkdir -p /var/state
COPY src/db/postgresql.conf /etc/postgresql/9.4/main/postgresql.conf
COPY src/db/pg_hba.conf /etc/postgresql/9.4/main/pg_hba.conf
COPY bin/arla /usr/bin/
