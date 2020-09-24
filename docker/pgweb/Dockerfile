### build pgweb in two stages

FROM golang:1 as builder
LABEL maintainer="wvh"

ENV \
	GOPATH=/go \
	IMPORTPATH=github.com/sosedoff/pgweb

# cache
RUN set -eux; \
	mkdir -p $GOPATH/src/$IMPORTPATH; \
	echo git clone https://$IMPORTPATH $GOPATH/src/$IMPORTPATH; \
	git clone https://$IMPORTPATH $GOPATH/src/$IMPORTPATH

WORKDIR $GOPATH/src/$IMPORTPATH

# checkout latest tag if there is one
RUN \
	TAG=$(git describe --tags --always $(git rev-list --tags --max-count=1) 2>/dev/null) && \
	[ -n "$TAG" ] && \
	echo "lastest tag is $TAG" && \
	git checkout -q $TAG

RUN make setup
RUN make build && mkdir -p /go/bin && cp -a pgweb /go/bin

# can't copy and chown in one go; put this here so the chown layer doesn't end up in the final image
COPY wait-for-it.sh /go/bin
RUN chmod +x /go/bin/wait-for-it.sh


### make a more minimal runtime image from the builder image

FROM debian:stable-slim as pgweb
COPY --from=builder /go/bin/pgweb /go/bin/wait-for-it.sh /bin/

# set defaults so the command below actually makes sense if the variables are unset;
# some of these will be overridden on run
ENV \
	PORT=8432 \
	USER=unset \
	PASS=unset \
	DB=unset \
	PGAPPNAME=pgweb

EXPOSE $PORT
ENTRYPOINT /bin/wait-for-it.sh -h db -p 5432 -t 5 -- /bin/pgweb --bind 0.0.0.0 --listen $PORT --ssl disable --host db --user $PGUSER --pass $PGPASSWORD --db $PGDATABASE --skip-open
