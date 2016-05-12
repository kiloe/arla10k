FROM arla/base
EXPOSE 80
COPY src/arla/querystore/conf/postgresql.conf /var/lib/postgresql/postgresql.conf
COPY src/arla/querystore/conf/pg_hba.conf /var/lib/postgresql/pg_hba.conf
COPY bin/arla /bin/
COPY bin/test /bin/
RUN mkdir -p /var/state \
	&& mkdir -p /app \
	&& chown -R postgres:postgres /var/lib/postgresql \
	&& mkdir -p /var/run/postgresql/stats \
	&& chown -R postgres:postgres /var/run/postgresql
WORKDIR /app
ENTRYPOINT ["/bin/arla"]
