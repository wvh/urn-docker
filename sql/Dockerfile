# Dockerfile for the official postgres image
#
# NOTE: This Dockerfile needs access to the SQL directory so it can add the SQL files to the build context;
#       paths are relative to this directory, so execute docker commands from there.

FROM postgres:12

RUN \
	localedef -i fi_FI -c -f UTF-8 -A /usr/share/locale/locale.alias fi_FI.UTF-8 && \
	localedef -i en_GB -c -f UTF-8 -A /usr/share/locale/locale.alias en_GB.UTF-8

COPY createdb.sh /docker-entrypoint-initdb.d/
COPY *.sql /docker-entrypoint-initdb.d/schemas/

#ENV LANG en_GB.utf8
ENV \
	LC_TIME=en_GB.utf8 \
	LC_CTYPE=fi_FI.utf8 \
	LC_COLLATE=fi_FI.utf8 \
	LC_PAPER=en_GB.utf8 \
	LC_MEASUREMENT=en_GB.utf8
